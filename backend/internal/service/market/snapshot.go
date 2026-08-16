package market

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const snapshotDisclaimer = "This is price, volume, market cap, open interest, funding, account long/short, and taker buy/sell on one tape. 1h/4h/24h changes use each metric's own history — a missing window is incomplete, not zero. Volume change is this window versus the previous window of the same length. Market-cap change follows price when circulating supply is unchanged. Informational only — not financial advice."

// GetSnapshot returns current tape metrics and 1h / 4h / 24h changes for one coin.
func (s *Service) GetSnapshot(ctx context.Context, exchange, symbol string) (*domain.SnapshotReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	if s.liqWatch != nil {
		s.liqWatch.Watch(symbol)
	}
	now := time.Now().UTC()
	spotEx := domain.DefaultExchange
	if ex != "all" {
		if _, e := s.ResolveExchange(ex); e == nil {
			spotEx = domain.Exchange(ex)
		}
	}

	var (
		bars   []domain.OHLCBar
		tkr    *domain.Ticker24h
		supply *domain.AssetSupply
		wg     sync.WaitGroup
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		// 1h candles cover 24h+ prior 24h for volume comparison.
		if c, err := s.GetCandles(ctx, string(spotEx), symbol, "1h", 50, nil, nil); err == nil {
			bars = domain.BarsFromCandles(c)
		}
	}()
	go func() {
		defer wg.Done()
		tkr, _ = s.GetTicker24h(ctx, string(spotEx), symbol)
	}()
	go func() {
		defer wg.Done()
		base, _ := domain.SplitBaseQuote(spotEx, symbol)
		if base == "" {
			base = symbol
		}
		supply, _ = s.GetSupply(ctx, base)
	}()
	wg.Wait()

	spotWins := domain.PriceVolumeWindows(bars, now)
	lastPx := 0.0
	if tkr != nil {
		if v, err := strconv.ParseFloat(tkr.LastPrice, 64); err == nil {
			lastPx = v
		}
	}
	if lastPx <= 0 {
		if lb := lastSnapshotClose(bars); lb > 0 {
			lastPx = lb
		}
	}
	vol24 := 0.0
	if tkr != nil {
		if v, err := strconv.ParseFloat(tkr.QuoteVolume, 64); err == nil {
			vol24 = v
		}
	}
	circ := 0.0
	if supply != nil && supply.CirculatingSupply != nil {
		circ = *supply.CirculatingSupply
	}
	domain.ApplyMarketCap(spotWins, circ, lastPx)
	spot := domain.SnapshotSpot{
		Price: lastPx, Volume24h: vol24, MarketCap: circ * lastPx, Circulating: circ,
		Windows: spotWins,
	}

	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	venues := make([]domain.SnapshotVenue, 0, len(want))
	var mu sync.Mutex
	var vwg sync.WaitGroup
	for _, v := range want {
		vwg.Add(1)
		go func(v domain.Exchange) {
			defer vwg.Done()
			ven := s.snapshotVenue(ctx, v, symbol, spotWins, now)
			mu.Lock()
			venues = append(venues, ven)
			mu.Unlock()
		}(v)
	}
	vwg.Wait()
	sort.Slice(venues, func(i, j int) bool {
		return string(venues[i].Exchange) < string(venues[j].Exchange)
	})

	out := &domain.SnapshotReport{
		Symbol: symbol, Exchange: ex, AsOf: now, Spot: spot, Venues: venues, Note: snapshotDisclaimer,
	}
	if ex == "all" && len(venues) > 0 {
		out.Combined = combineSnapshotVenues(symbol, venues)
	}
	out.Summary = domain.ExplainSnapshotReport(symbol, spot, venues)
	return out, nil
}

func lastSnapshotClose(bars []domain.OHLCBar) float64 {
	var t time.Time
	var px float64
	for _, b := range bars {
		if b.Close > 0 && (t.IsZero() || b.Time.After(t)) {
			t, px = b.Time, b.Close
		}
	}
	return px
}

func (s *Service) snapshotVenue(ctx context.Context, ex domain.Exchange, symbol string, spot []domain.SnapshotWindow, now time.Time) domain.SnapshotVenue {
	var (
		oi    *domain.OpenInterestSeries
		fund  *domain.FundingSeries
		ls    *domain.LongShortSeries
		taker *domain.TakerVenueFlow
		wg    sync.WaitGroup
	)
	wg.Add(4)
	go func() {
		defer wg.Done()
		if p := s.oiPort(ex); p != nil {
			oi, _ = p.GetOpenInterestSeries(ctx, symbol)
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.fundingPort(ex); p != nil {
			fund, _ = p.GetFundingSeries(ctx, symbol, domain.DefaultFundingHistoryLimit)
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.longShortPort(ex); p != nil {
			ls, _ = p.GetLongShortSeries(ctx, symbol, domain.MaxLongShortHistoryLimit)
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.takerPort(ex); p != nil {
			taker, _ = p.GetTakerFlow(ctx, symbol)
		}
	}()
	wg.Wait()
	ven := domain.BuildSnapshotVenue(ex, spot, oi, fund, ls, taker, now)
	if oi == nil && fund == nil && ls == nil && taker == nil {
		ven.Error = "futures tape not configured"
		ven.Summary = ven.Error
	}
	return ven
}

func combineSnapshotVenues(symbol string, venues []domain.SnapshotVenue) *domain.SnapshotVenue {
	if len(venues) == 0 {
		return nil
	}
	out := domain.SnapshotVenue{Exchange: "combined", Windows: make([]domain.SnapshotWindow, 0, 3)}
	// Prefer the first complete 1h meaning; copy spot fields from first venue.
	for _, id := range []string{domain.SnapshotWindow1h, domain.SnapshotWindow4h, domain.SnapshotWindow24h} {
		row := domain.SnapshotWindow{Window: id}
		var oiCur, oiPast, oiN float64
		var fundW, fundN, lsW, lsN float64
		var buy, sell float64
		takerOK := false
		for _, v := range venues {
			for _, w := range v.Windows {
				if w.Window != id {
					continue
				}
				if row.Price.Window == "" {
					row.Price, row.Volume, row.MarketCap = w.Price, w.Volume, w.MarketCap
				}
				if w.OI.Complete {
					oiCur += w.OI.Current
					oiPast += w.OI.Past
					oiN++
				}
				if w.Funding.Complete {
					fundW += w.Funding.Current
					fundN++
				}
				if w.LongPct.Complete {
					lsW += w.LongPct.Current
					lsN++
				}
				if w.Taker.Complete {
					buy += w.Taker.Buy
					sell += w.Taker.Sell
					takerOK = true
				}
			}
		}
		if oiN > 0 {
			row.OI = domain.ChangeFromValues(id, oiCur, oiPast, true)
		}
		if fundN > 0 {
			row.Funding = domain.SnapshotChange{Window: id, Current: fundW / fundN, Complete: true, Direction: "flat"}
		}
		if lsN > 0 {
			row.LongPct = domain.SnapshotChange{Window: id, Current: lsW / lsN, Complete: true, Direction: "flat"}
		}
		if takerOK {
			row.Taker = domain.SnapshotTaker{
				Window: id, Buy: buy, Sell: sell, Delta: buy - sell,
				Dominant: domain.TakerDominant(buy, sell), Complete: true,
			}
			if buy+sell > 0 {
				row.Taker.BuyShare = buy / (buy + sell)
			}
		}
		out.Windows = append(out.Windows, row)
	}
	out.Summary = domain.ExplainSnapshotVenue(out)
	_ = symbol
	return &out
}
