package market

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const perpDepthDisclaimer = "Resting USD-M / linear / USDT-swap perpetual depth: bid notional within N% below mid vs ask notional within N% above. Binance USD-M (1000 levels), Bybit linear (500 levels), and OKX USDT-swap (400 levels, contracts converted to base) separately plus combined. Combined adds each venue's own ±% band (not a shared price grid). Snapshot may not fully cover 2%. Informational only — not financial advice."

// GetPerpDepth returns bid-below vs ask-above liquidity on BTC (and other) perps.
func (s *Service) GetPerpDepth(ctx context.Context, exchange, symbol string, rangePct float64) (*domain.PerpDepthReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParsePerpDepthExchange(exchange)
	if err != nil {
		return nil, err
	}
	rangePct, err = domain.ParsePerpDepthRangePct(rangePct)
	if err != nil {
		return nil, err
	}
	want := domain.FuturesVenues()
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	s.noteFutures(symbol)

	type fetched struct {
		ex   domain.Exchange
		book *domain.RawOrderBook
		err  error
	}
	got := make([]fetched, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			p := s.futuresDepthPort(v)
			if p == nil {
				got[i] = fetched{ex: v, err: fmt.Errorf("perpetual depth not configured")}
				return
			}
			book, ferr := p.GetFuturesDepth(ctx, symbol)
			got[i] = fetched{ex: v, book: book, err: ferr}
		}(i, v)
	}
	wg.Wait()

	out := &domain.PerpDepthReport{
		Symbol:   symbol,
		Exchange: ex,
		RangePct: rangePct,
		AsOf:     time.Now().UTC(),
		Venues:   make([]domain.PerpDepthVenue, 0, len(want)),
		Note:     perpDepthDisclaimer,
	}
	for _, f := range got {
		if f.err != nil && (f.book == nil || (len(f.book.Bids) == 0 && len(f.book.Asks) == 0)) {
			row := domain.PerpDepthVenue{
				Exchange: f.ex, Symbol: symbol, Error: f.err.Error(), Summary: f.err.Error(),
			}
			if f.book != nil {
				row.AsOf = f.book.FetchedAt
			}
			out.Venues = append(out.Venues, row)
			continue
		}
		row := domain.BuildPerpDepthVenue(f.ex, symbol, f.book, rangePct)
		if f.book != nil {
			row.AsOf = f.book.FetchedAt
		}
		out.Venues = append(out.Venues, row)
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" && len(out.Venues) > 0 {
		out.Combined = domain.CombinePerpDepthVenues(symbol, out.Venues, rangePct)
	}
	out.Summary = domain.ExplainPerpDepthReport(*out)
	return out, nil
}

func perpCoverageSamples(exchange string, rep *domain.PerpDepthReport, err error) []domain.VenueSample {
	want := domain.ExpectedFuturesVenues(exchange)
	if err != nil || rep == nil {
		out := make([]domain.VenueSample, len(want))
		for i, ex := range want {
			out[i] = domain.VenueSample{Exchange: ex}
		}
		return out
	}
	out := make([]domain.VenueSample, 0, len(rep.Venues))
	seen := map[domain.Exchange]bool{}
	for _, v := range rep.Venues {
		if v.Exchange == "" || v.Exchange == "all" || seen[v.Exchange] {
			continue
		}
		seen[v.Exchange] = true
		out = append(out, domain.VenueSample{Exchange: v.Exchange, OK: v.Error == "", At: v.AsOf})
	}
	for _, ex := range want {
		if !seen[ex] {
			out = append(out, domain.VenueSample{Exchange: ex})
		}
	}
	return out
}

func (s *Service) futuresDepthPort(ex domain.Exchange) domain.FuturesDepthPort {
	if s != nil && s.futDepth != nil {
		if p := s.futDepth[ex]; p != nil {
			return p
		}
	}
	p, err := s.port(ex)
	if err != nil || p == nil {
		return nil
	}
	fp, ok := p.(domain.FuturesDepthPort)
	if !ok {
		return nil
	}
	return fp
}

const (
	perpOverviewCacheTTL = 8 * time.Second
	perpOverviewWorkers  = 4
)

// GetPerpDepthOverview sums bid/ask within rangePct across the liquid perps we follow.
func (s *Service) GetPerpDepthOverview(ctx context.Context, exchange string, rangePct float64, limit int) (*domain.PerpDepthOverview, error) {
	ex, err := domain.ParsePerpDepthExchange(exchange)
	if err != nil {
		return nil, err
	}
	rangePct, err = domain.ParsePerpDepthRangePct(rangePct)
	if err != nil {
		return nil, err
	}
	limit = domain.ParsePerpDepthOverviewLimit(limit)
	key := ex + "|" + strconv.FormatFloat(rangePct, 'f', -1, 64) + "|" + strconv.Itoa(limit)
	if c := s.perpOverviewCache(); c != nil {
		if hit, ok := c.Get(key); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}

	symbols := s.perpOverviewUniverse(ctx, limit)
	type fetched struct {
		in      domain.PerpDepthCoinInput
		samples []domain.VenueSample
	}
	got := make([]fetched, len(symbols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, perpOverviewWorkers)
	for i, sym := range symbols {
		wg.Add(1)
		go func(i int, sym string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				got[i] = fetched{in: domain.PerpDepthCoinInput{Symbol: sym, Error: ctx.Err().Error()}}
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			rep, ferr := s.GetPerpDepth(ctx, ex, sym, rangePct)
			base, _ := domain.SplitBaseQuote(domain.ExchangeBinance, sym)
			in := domain.PerpDepthCoinInput{Symbol: sym, Base: base}
			samples := perpCoverageSamples(ex, rep, ferr)
			if ferr != nil {
				in.Error = ferr.Error()
				got[i] = fetched{in: in, samples: samples}
				return
			}
			in.Bid, in.Ask, in.Lean, in.Complete, in.Error = perpDepthHeadline(rep)
			in.Bands = perpDepthCoinBands(rep)
			in.Venues = perpDepthVenueBooks(rep)
			got[i] = fetched{in: in, samples: samples}
		}(i, sym)
	}
	wg.Wait()
	coins := make([]domain.PerpDepthCoinInput, 0, len(got))
	for _, f := range got {
		if f.in.Symbol == "" {
			continue
		}
		coins = append(coins, f.in)
	}
	now := time.Now().UTC()
	out := domain.BuildPerpDepthOverview(ex, rangePct, coins, now)
	samples := make([]domain.VenueSample, 0, len(got)*3)
	for _, f := range got {
		samples = append(samples, f.samples...)
	}
	s.persistPerpDepthOverview(ctx, out, coins)
	s.attachPerpDepthWindows(ctx, out)
	domain.ApplyPerpDepthCoverage(out, samples, now)
	if c := s.perpOverviewCache(); c != nil && out != nil {
		cp := *out
		c.Set(key, &cp)
	}
	return out, nil
}

func (s *Service) persistPerpDepthOverview(ctx context.Context, out *domain.PerpDepthOverview, coins []domain.PerpDepthCoinInput) {
	if s == nil || s.perpHist == nil || out == nil {
		return
	}
	rows := domain.PerpDepthSamplesFromOverview(out)
	if out.Exchange == "all" {
		rows = append(rows, domain.PerpDepthVenueSamples(out.AsOf, out.RangePct, coins)...)
	}
	if len(rows) == 0 {
		return
	}
	_, _ = s.perpHist.InsertSamples(ctx, rows)
}

func (s *Service) attachPerpDepthWindows(ctx context.Context, out *domain.PerpDepthOverview) {
	if s == nil || s.perpHist == nil || out == nil {
		return
	}
	from := out.AsOf.Add(-4*time.Hour - 20*time.Minute)
	rows, err := s.perpHist.ListSamples(ctx, out.Exchange, out.RangePct, from, out.AsOf.Add(time.Minute))
	if err != nil {
		return
	}
	domain.AttachPerpDepthWindows(out, rows)
	fromTotals := out.AsOf.Add(-domain.PerpDepthTotalChartSpan)
	totals, err := s.perpHist.ListMinuteTotals(ctx, out.Exchange, out.RangePct, fromTotals, out.AsOf.Add(time.Minute))
	if err != nil {
		return
	}
	domain.AttachPerpDepthHistory(out, totals)
	s.attachPerpDepthBTC(ctx, out)
	coinRows, err := s.perpHist.ListSamples(ctx, out.Exchange, out.RangePct, fromTotals, out.AsOf.Add(time.Minute))
	if err != nil {
		return
	}
	domain.AttachPerpDepthMoveCoins(out, coinRows)
	s.attachPerpDepthMoveVenues(ctx, out)
}

func (s *Service) attachPerpDepthMoveVenues(ctx context.Context, out *domain.PerpDepthOverview) {
	if s == nil || s.perpHist == nil || out == nil || len(out.Moves) == 0 {
		return
	}
	from := out.AsOf.Add(-domain.PerpDepthTotalChartSpan)
	to := out.AsOf.Add(time.Minute)
	by := map[string][]domain.PerpDepthTotalPoint{}
	for _, ex := range domain.FuturesVenues() {
		pts, err := s.perpHist.ListMinuteTotals(ctx, string(ex), out.RangePct, from, to)
		if err != nil || len(pts) == 0 {
			continue
		}
		by[string(ex)] = pts
	}
	domain.AttachPerpDepthMoveVenues(out, by)
}

func (s *Service) attachPerpDepthBTC(ctx context.Context, out *domain.PerpDepthOverview) {
	if s == nil || out == nil || len(out.History) < 2 {
		return
	}
	start := out.History[0].Time
	end := out.AsOf.Add(time.Minute)
	ex := "binance"
	if out.Exchange == string(domain.ExchangeBybit) {
		ex = string(domain.ExchangeBybit)
	}
	candles, err := s.GetCandles(ctx, ex, "BTCUSDT", string(domain.Interval1m), 400, &start, &end)
	if (err != nil || len(candles) == 0) && ex != string(domain.ExchangeBinance) {
		candles, err = s.GetCandles(ctx, string(domain.ExchangeBinance), "BTCUSDT", string(domain.Interval1m), 400, &start, &end)
	}
	if err != nil || len(candles) == 0 {
		return
	}
	domain.AttachPerpDepthPrices(out, candles)
}

// PurgePerpDepthHistory drops samples older than cutoff.
func (s *Service) PurgePerpDepthHistory(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.perpHist == nil {
		return 0, nil
	}
	return s.perpHist.PurgeOlderThan(ctx, cutoff)
}

func (s *Service) perpOverviewCache() *cache.TTL[*domain.PerpDepthOverview] {
	if s == nil {
		return nil
	}
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if s.perpOverview == nil {
		s.perpOverview = cache.New[*domain.PerpDepthOverview](perpOverviewCacheTTL)
	}
	return s.perpOverview
}

func (s *Service) perpOverviewUniverse(ctx context.Context, limit int) []string {
	listEx := domain.ExchangeBinance
	if _, err := s.port(listEx); err != nil {
		for _, ex := range []domain.Exchange{domain.ExchangeBybit, domain.ExchangeCoinbase} {
			if _, perr := s.port(ex); perr == nil {
				listEx = ex
				break
			}
		}
	}
	spot, err := s.ListSpotMarkets(ctx, string(listEx), domain.SpotListQuery{
		QuoteAsset: "USDT",
		Status:     "TRADING",
		SortBy:     domain.SpotSortQuoteVolume,
		Order:      domain.SortDesc,
		Limit:      limit + 20,
	})
	seen := map[string]struct{}{}
	out := make([]string, 0, limit)
	add := func(symbol string) {
		symbol = domain.NormalizeLiquidationSymbol(symbol)
		if symbol == "" {
			return
		}
		base, _ := domain.SplitBaseQuote(listEx, symbol)
		if !domain.IsPerpDepthEligible(base) {
			return
		}
		if _, ok := seen[base]; ok {
			return
		}
		if len(out) >= limit && base != "BTC" && base != "ETH" {
			return
		}
		seen[base] = struct{}{}
		out = append(out, symbol)
	}
	if err == nil && spot != nil {
		for _, m := range spot.Items {
			add(m.Symbol)
			if len(out) >= limit {
				break
			}
		}
	}
	for _, seed := range domain.DefaultFuturesHistorySymbols {
		add(seed)
		if len(out) >= limit {
			break
		}
	}
	if _, ok := seen["BTC"]; !ok {
		add("BTCUSDT")
	}
	if _, ok := seen["ETH"]; !ok {
		add("ETHUSDT")
	}
	if len(out) > limit {
		// Keep BTC/ETH even if we overflowed after filling from volume.
		kept := out[:limit]
		haveBTC, haveETH := false, false
		for _, sym := range kept {
			b, _ := domain.SplitBaseQuote(domain.ExchangeBinance, sym)
			if b == "BTC" {
				haveBTC = true
			}
			if b == "ETH" {
				haveETH = true
			}
		}
		if !haveBTC {
			kept[len(kept)-1] = "BTCUSDT"
		}
		if !haveETH && limit >= 2 {
			kept[len(kept)-2] = "ETHUSDT"
		}
		out = kept
	}
	return out
}

func perpDepthCoinBands(rep *domain.PerpDepthReport) []domain.PerpDepthCoinBandInput {
	v := perpDepthBestVenue(rep)
	if v == nil {
		return nil
	}
	out := make([]domain.PerpDepthCoinBandInput, 0, len(v.Bands))
	for _, b := range v.Bands {
		out = append(out, domain.PerpDepthCoinBandInput{
			RangePct: b.RangePct,
			Bid:      b.Below.Notional,
			Ask:      b.Above.Notional,
			Complete: b.Complete,
		})
	}
	return out
}

func perpDepthBestVenue(rep *domain.PerpDepthReport) *domain.PerpDepthVenue {
	if rep == nil {
		return nil
	}
	if rep.Combined != nil && rep.Combined.Error == "" && (rep.Combined.Headline.Below.Notional+rep.Combined.Headline.Above.Notional > 0 || len(rep.Combined.Bands) > 0) {
		return rep.Combined
	}
	for i := range rep.Venues {
		v := &rep.Venues[i]
		if v.Error == "" {
			return v
		}
	}
	return nil
}

func perpDepthVenueBooks(rep *domain.PerpDepthReport) []domain.PerpDepthVenueBook {
	if rep == nil || rep.Exchange != "all" {
		return nil
	}
	out := make([]domain.PerpDepthVenueBook, 0, len(rep.Venues))
	for _, v := range rep.Venues {
		if v.Error != "" {
			continue
		}
		bid, ask := v.Headline.Below.Notional, v.Headline.Above.Notional
		if bid+ask <= 0 {
			continue
		}
		out = append(out, domain.PerpDepthVenueBook{Exchange: string(v.Exchange), Bid: bid, Ask: ask})
	}
	return out
}

func perpDepthHeadline(rep *domain.PerpDepthReport) (bid, ask float64, lean string, complete bool, errMsg string) {
	if rep == nil {
		return 0, 0, "", false, "no perpetual depth"
	}
	if rep.Combined != nil && rep.Combined.Error == "" && (rep.Combined.Headline.Below.Notional+rep.Combined.Headline.Above.Notional > 0 || rep.Combined.Mid > 0) {
		h := rep.Combined.Headline
		return h.Below.Notional, h.Above.Notional, h.Lean, h.Complete, ""
	}
	for _, v := range rep.Venues {
		if v.Error == "" {
			h := v.Headline
			return h.Below.Notional, h.Above.Notional, h.Lean, h.Complete, ""
		}
	}
	for _, v := range rep.Venues {
		if v.Error != "" {
			return 0, 0, "", false, v.Error
		}
	}
	if rep.Summary != "" {
		return 0, 0, "", false, rep.Summary
	}
	return 0, 0, "", false, "no perpetual depth"
}
