package market

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationPrintHeatmap is stored force-order notional by price and time.
// exchange=all returns Binance, Bybit, OKX, and a combined grid.
// A combined column is filled only when every venue recorded that column.
func (s *Service) GetLiquidationPrintHeatmap(ctx context.Context, exchange, symbol, rawRange string) (*domain.HuntHeatmapReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseHuntHeatmapRange(rawRange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	if s.liqWatch != nil {
		s.liqWatch.Watch(symbol)
	}
	now := time.Now().UTC()
	times, origin, until := domain.MeasuredHeatmapWindow(spec, now)
	venues := domain.FuturesVenues()
	if ex != "" && ex != "all" {
		venues = []domain.Exchange{domain.Exchange(ex)}
	}
	pMin, pMax := s.measuredPrintExtent(ctx, venues, symbol, spec, now, times, origin, until)
	in := domain.MeasuredHeatmapInput{
		Symbol: symbol, Spec: spec, To: now, PriceMin: pMin, PriceMax: pMax, Venues: venues,
	}
	if s.liq != nil {
		in.Coverage = s.liq.CoverageSnapshot(now)
	}
	if pMax <= pMin || len(times) == 0 {
		got := domain.BuildMeasuredLiquidationHeatmap(in)
		domain.AttachPrintAnchors(&got, symbol, now, in.Coverage, s.measuredPrintAnchorWindows(ctx, symbol, now), venues)
		return &got, nil
	}
	q := domain.MeasuredPrintQuery{
		Symbol: symbol, Origin: origin, Until: until, Step: spec.Step, Columns: len(times),
		PriceMin: pMin, PriceMax: pMax, PriceBins: domain.HuntHeatmapPriceBins, Exchanges: venues,
	}
	scan, err := s.measuredPrintScan(ctx, q, venues)
	if err != nil {
		return nil, err
	}
	scan = mergeBookPrints(scan, q, s.liq)
	in.Bins = scan.Bins
	got := domain.BuildMeasuredLiquidationHeatmap(in)
	domain.AttachPrintAnchors(&got, symbol, now, in.Coverage, s.measuredPrintAnchorWindows(ctx, symbol, now), venues)
	return &got, nil
}

// measuredPrintAnchorWindows loads the UTC day and the current 4-hour print sums.
// A failed window stays incomplete. It does not fail the heatmap.
func (s *Service) measuredPrintAnchorWindows(ctx context.Context, symbol string, now time.Time) []domain.PrintAnchorWindow {
	day, block := domain.PrintAnchorStarts(now)
	specs := []struct {
		id   string
		from time.Time
	}{
		{domain.PrintAnchorDay, day},
		{domain.PrintAnchor4h, block},
	}
	out := make([]domain.PrintAnchorWindow, len(specs))
	var wg sync.WaitGroup
	for i, spec := range specs {
		wg.Add(1)
		go func(i int, id string, from time.Time) {
			defer wg.Done()
			sides, ok := s.summedPrintSides(ctx, symbol, from, now)
			out[i] = domain.PrintAnchorWindow{Anchor: id, From: from, ArchiveOK: ok, Sides: sides}
		}(i, spec.id, spec.from)
	}
	wg.Wait()
	return out
}

func (s *Service) measuredPrintExtent(ctx context.Context, venues []domain.Exchange, symbol string, spec domain.HuntHeatmapSpec, now time.Time, times []time.Time, origin, until time.Time) (float64, float64) {
	from := now.Add(-spec.Window)
	var mu sync.Mutex
	var wg sync.WaitGroup
	points := make([]domain.HuntHeatmapPricePoint, 0, 64)
	for _, ex := range venues {
		wg.Add(1)
		go func(ex domain.Exchange) {
			defer wg.Done()
			pts := s.huntHeatmapPrices(ctx, ex, symbol, spec, from, now)
			if len(pts) == 0 {
				return
			}
			mu.Lock()
			points = append(points, pts...)
			mu.Unlock()
		}(ex)
	}
	wg.Wait()
	if len(points) > 0 {
		return domain.MeasuredPriceExtent(points, times)
	}
	lo, hi, ok := bookPrintPrices(s.liq, venues, symbol, origin, until)
	if s.futHist != nil {
		if span, okSpan := s.futHist.(domain.LiquidationPrintAggregator); okSpan {
			slo, shi, err := span.LiquidationPrintPriceSpan(ctx, symbol, origin, until, venues)
			if err != nil {
				slog.Warn("print heatmap price span", "symbol", symbol, "err", err)
			} else if shi > 0 {
				if !ok {
					lo, hi = slo, shi
				} else {
					if slo > 0 && slo < lo {
						lo = slo
					}
					if shi > hi {
						hi = shi
					}
				}
				ok = true
			}
		}
	}
	if !ok {
		return 0, 0
	}
	return domain.MeasuredPrintPriceExtent(lo, hi)
}

func (s *Service) measuredPrintScan(ctx context.Context, q domain.MeasuredPrintQuery, venues []domain.Exchange) (domain.MeasuredPrintScan, error) {
	if agg, ok := s.futHist.(domain.LiquidationPrintAggregator); ok {
		scan, err := agg.AggregateLiquidationPrints(ctx, q)
		if err == nil {
			return scan, nil
		}
		if !bookHasWindowPrints(s.liq, venues, q.Symbol, q.Origin, q.Until) {
			return domain.MeasuredPrintScan{}, err
		}
		slog.Warn("print heatmap aggregate", "symbol", q.Symbol, "err", err)
		return domain.MeasuredPrintScan{}, nil
	}
	if s.futHist == nil {
		return domain.MeasuredPrintScan{}, nil
	}
	events := make([]domain.LiquidationEvent, 0, 64)
	for _, ex := range venues {
		rows := s.huntHistLiquidations(ctx, string(ex), q.Symbol, q.Origin, q.Until)
		for _, e := range rows {
			if e.Exchange == ex {
				events = append(events, e)
			}
		}
	}
	return domain.BinMeasuredPrintEvents(q, events), nil
}

func bookPrintPrices(book *domain.LiquidationBook, venues []domain.Exchange, symbol string, origin, until time.Time) (float64, float64, bool) {
	if book == nil {
		return 0, 0, false
	}
	lo, hi := 0.0, 0.0
	ok := false
	for _, ex := range venues {
		for _, e := range book.Events(string(ex), symbol, origin) {
			if e.Price <= 0 || e.Time.IsZero() {
				continue
			}
			if !origin.IsZero() && e.Time.Before(origin) {
				continue
			}
			if !until.IsZero() && !e.Time.Before(until) {
				continue
			}
			if !ok || e.Price < lo {
				lo = e.Price
			}
			if e.Price > hi {
				hi = e.Price
			}
			ok = true
		}
	}
	return lo, hi, ok
}

func bookHasWindowPrints(book *domain.LiquidationBook, venues []domain.Exchange, symbol string, origin, until time.Time) bool {
	lo, hi, ok := bookPrintPrices(book, venues, symbol, origin, until)
	return ok && hi >= lo && hi > 0
}

// mergeBookPrints adds in-memory prints strictly newer than the newest stored
// print for that venue. A zero stored time keeps every in-window book print.
func mergeBookPrints(scan domain.MeasuredPrintScan, q domain.MeasuredPrintQuery, book *domain.LiquidationBook) domain.MeasuredPrintScan {
	if book == nil {
		return scan
	}
	if scan.Latest == nil {
		scan.Latest = map[domain.Exchange]time.Time{}
	}
	venues := q.Exchanges
	if len(venues) == 0 {
		venues = domain.FuturesVenues()
	}
	extra := make([]domain.LiquidationEvent, 0, 16)
	for _, ex := range venues {
		latest := scan.Latest[ex]
		for _, e := range book.Events(string(ex), q.Symbol, q.Origin) {
			if !latest.IsZero() && !e.Time.After(latest) {
				continue
			}
			extra = append(extra, e)
		}
	}
	if len(extra) == 0 {
		return scan
	}
	add := domain.BinMeasuredPrintEvents(q, extra)
	scan.Bins = append(scan.Bins, add.Bins...)
	for ex, ts := range add.Latest {
		if prev, ok := scan.Latest[ex]; !ok || ts.After(prev) {
			scan.Latest[ex] = ts
		}
	}
	return scan
}
