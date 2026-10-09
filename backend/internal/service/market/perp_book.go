package market

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const perpBookSampleWorkers = 4

// GetPerpBookHistory returns stored perp bid/ask bands for one coin.
// windowSec, when positive, is the lookback ending now. from and to override it.
func (s *Service) GetPerpBookHistory(ctx context.Context, exchange, symbol string, windowSec int, from, to *time.Time) (*domain.PerpBookHistory, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParsePerpDepthExchange(exchange)
	if err != nil {
		return nil, err
	}
	end := time.Now().UTC()
	if to != nil && !to.IsZero() {
		end = to.UTC()
	}
	start := end.Add(-24 * time.Hour)
	if windowSec > 0 {
		if windowSec < 60 {
			windowSec = 60
		}
		max := int(domain.PerpBookHistoryMaxSpan / time.Second)
		if windowSec > max {
			windowSec = max
		}
		start = end.Add(-time.Duration(windowSec) * time.Second)
	}
	if from != nil && !from.IsZero() {
		start = from.UTC()
	}
	if end.Sub(start) > domain.PerpBookHistoryMaxSpan {
		start = end.Add(-domain.PerpBookHistoryMaxSpan)
	}
	step := domain.PerpBookHistoryStep(end.Sub(start))
	if s == nil || s.perpHist == nil {
		return domain.BuildPerpBookHistory(symbol, ex, start, end, nil, nil), nil
	}
	samples, err := s.perpHist.ListBookBandBuckets(ctx, ex, symbol, start, end.Add(time.Minute), step)
	if err != nil {
		return nil, err
	}
	var gaps []domain.PerpBookGap
	if ex == "all" {
		gaps, err = s.perpHist.ListBookGaps(ctx, symbol, start, end.Add(time.Minute))
		if err != nil {
			return nil, err
		}
	}
	return domain.BuildPerpBookHistory(symbol, ex, start, end, samples, gaps), nil
}

// SamplePerpBookBands stores 0.1% / 0.5% / 1% perp books for the liquid list.
// A coin is written only when Binance, Bybit, and OKX are all fresh.
func (s *Service) SamplePerpBookBands(ctx context.Context) (stored, gaps int, err error) {
	if s == nil || s.perpHist == nil {
		return 0, 0, nil
	}
	symbols, err := s.LiquidPerpSymbols(ctx, domain.LiquidRecorderLimit)
	if len(symbols) == 0 {
		return 0, 0, err
	}
	now := time.Now().UTC()
	type result struct {
		rows []domain.PerpBookBandSample
		gap  *domain.PerpBookGap
	}
	got := make([]result, len(symbols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, perpBookSampleWorkers)
	for i, sym := range symbols {
		wg.Add(1)
		go func(i int, sym string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			rep, ferr := s.GetPerpDepth(ctx, "all", sym, 1)
			got[i] = perpBookSample(sym, rep, ferr, now)
		}(i, sym)
	}
	wg.Wait()
	rows := make([]domain.PerpBookBandSample, 0, len(symbols)*3)
	for _, item := range got {
		if item.gap != nil {
			if gerr := s.perpHist.InsertBookGap(ctx, *item.gap); gerr != nil && err == nil {
				err = gerr
			}
			gaps++
			continue
		}
		if len(item.rows) == 0 {
			continue
		}
		rows = append(rows, item.rows...)
		_ = s.perpHist.ClearBookGap(ctx, item.rows[0].Symbol, item.rows[0].Time)
		stored++
	}
	if len(rows) > 0 {
		if _, ierr := s.perpHist.InsertBookBands(ctx, rows); ierr != nil && err == nil {
			err = ierr
		}
	}
	return stored, gaps, err
}

func perpBookSample(symbol string, rep *domain.PerpDepthReport, ferr error, now time.Time) struct {
	rows []domain.PerpBookBandSample
	gap  *domain.PerpBookGap
} {
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	out := struct {
		rows []domain.PerpBookBandSample
		gap  *domain.PerpBookGap
	}{}
	if symbol == "" {
		return out
	}
	if ferr != nil || rep == nil {
		out.gap = &domain.PerpBookGap{Time: now, Symbol: symbol, Missing: []string{"binance", "bybit", "okx"}}
		return out
	}
	by := map[domain.Exchange]domain.PerpDepthVenue{}
	for _, v := range rep.Venues {
		by[v.Exchange] = v
	}
	var missing, stale []string
	fresh := make([]domain.PerpDepthVenue, 0, 3)
	for _, ex := range domain.FuturesVenues() {
		v, ok := by[ex]
		if !ok || v.Error != "" || v.Mid <= 0 {
			missing = append(missing, string(ex))
			continue
		}
		at := v.AsOf
		if at.IsZero() {
			at = now
		}
		if now.Sub(at) > domain.VenueBookMaxAge {
			stale = append(stale, string(ex))
			continue
		}
		fresh = append(fresh, v)
	}
	if len(missing)+len(stale) > 0 || rep.Combined == nil || rep.Combined.Mid <= 0 {
		out.gap = &domain.PerpBookGap{Time: now, Symbol: symbol, Missing: missing, Stale: stale}
		return out
	}
	at := domain.BucketPerpDepthSampleTime(now)
	coverBid, coverAsk := minPerpCover(fresh)
	out.rows = append(out.rows, perpBookBandRows(at, "all", symbol, rep.Combined.Mid, coverBid, coverAsk, rep.Combined.Bands)...)
	for _, v := range fresh {
		out.rows = append(out.rows, perpBookBandRows(at, string(v.Exchange), symbol, v.Mid, v.CoverBidPct, v.CoverAskPct, v.Bands)...)
	}
	if len(out.rows) == 0 {
		out.gap = &domain.PerpBookGap{Time: now, Symbol: symbol, Missing: missing, Stale: stale}
	}
	return out
}

func perpBookBandRows(at time.Time, exchange, symbol string, mid, coverBid, coverAsk float64, bands []domain.PerpDepthBand) []domain.PerpBookBandSample {
	out := make([]domain.PerpBookBandSample, 0, len(domain.PerpBookStoreBands))
	for _, want := range domain.PerpBookStoreBands {
		for _, b := range bands {
			if !bandPctEqual(b.RangePct, want) {
				continue
			}
			if b.Below.Notional+b.Above.Notional <= 0 {
				continue
			}
			out = append(out, domain.PerpBookBandSample{
				Time: at, Exchange: exchange, Symbol: symbol, RangePct: want,
				Bid: b.Below.Notional, Ask: b.Above.Notional, Mid: mid,
				CoverBid: coverBid, CoverAsk: coverAsk,
			})
			break
		}
	}
	return out
}

func minPerpCover(venues []domain.PerpDepthVenue) (bid, ask float64) {
	for _, v := range venues {
		if v.CoverBidPct > 0 && (bid == 0 || v.CoverBidPct < bid) {
			bid = v.CoverBidPct
		}
		if v.CoverAskPct > 0 && (ask == 0 || v.CoverAskPct < ask) {
			ask = v.CoverAskPct
		}
	}
	return bid, ask
}

func bandPctEqual(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-6
}
