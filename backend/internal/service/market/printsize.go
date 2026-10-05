package market

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const printSizeDisclaimer = "Average aggressive futures print size (USDT notional per taker fill). Compare the last 5m / 15m to the same length just before: a jump from 20K to 180K is a few large market orders, not many small ones. Binance, Bybit, and OKX use the live trade stream when it covers the window. When that tape is empty, stored flow-at-prices minutes supply the notional and the fill count. Otherwise the newest public trades are used. Informational only — not financial advice."

// GetPrintSize returns average buy vs sell trade size and how it just changed.
func (s *Service) GetPrintSize(ctx context.Context, exchange, symbol string) (*domain.PrintSizeReport, error) {
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
	want := configuredFutures(ex, func(v domain.Exchange) bool { return s.printPort(v) != nil })
	now := time.Now().UTC()
	from := now.Add(-domain.PrintSizeLookback())
	out := &domain.PrintSizeReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.PrintSizeVenue, 0, len(want)),
		Note:     printSizeDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.printSizeOne(ctx, v, symbol, from, now)
			mu.Lock()
			out.Venues = append(out.Venues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" && len(out.Venues) > 0 {
		out.Combined = domain.CombinePrintSizeVenues(symbol, out.Venues)
	}
	return out, nil
}

const printSizeOverviewDisclaimer = "Whole-market print size sums aggressive futures notional and fill counts across the liquid USDT perpetuals the recorder follows (up to 100, seed pairs first), then divides. A busier coin weighs more in the average. Each pair also votes whether its own average got larger, smaller, a few large orders, or many small ones. Windows are 5m and 15m versus the same length just before. Informational only — not financial advice."

// GetPrintSizeOverview reports whether average print size grew or shrank across liquid pairs.
func (s *Service) GetPrintSizeOverview(ctx context.Context, exchange, quote string, limit int) (*domain.PrintSizeOverview, error) {
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	if quote == "" {
		quote = "USDT"
	}
	limit = domain.ParsePrintSizeOverviewLimit(limit)
	symbols, err := s.liquidOverviewSymbols(ctx, ex, quote, limit)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	from := now.Add(-domain.PrintSizeLookback())
	coins := make([]domain.PrintSizeVenue, len(symbols))
	parts := make([][]domain.VenueSample, len(symbols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, symbol := range symbols {
		wg.Add(1)
		go func(i int, symbol string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			coins[i], parts[i] = s.printSizeSymbol(ctx, ex, symbol, from, now)
		}(i, symbol)
	}
	wg.Wait()
	samples := make([]domain.VenueSample, 0, len(symbols)*3)
	for _, part := range parts {
		samples = append(samples, part...)
	}
	out := domain.BuildPrintSizeOverview(ex, quote, coins, now)
	out.Note = printSizeOverviewDisclaimer
	domain.ApplyPrintSizeCoverage(&out, samples, now)
	return &out, nil
}

func (s *Service) printSizeSymbol(ctx context.Context, ex, symbol string, from, to time.Time) (domain.PrintSizeVenue, []domain.VenueSample) {
	want := configuredFutures(ex, func(v domain.Exchange) bool { return s.printPort(v) != nil })
	venues := make([]domain.PrintSizeVenue, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			venues[i] = s.printSizeOne(ctx, v, symbol, from, to)
		}(i, v)
	}
	wg.Wait()
	samples := venueSamplesFromPrintSize(venues)
	if ex == "all" {
		if combined := domain.CombinePrintSizeVenues(symbol, venues); combined != nil {
			return *combined, samples
		}
	}
	for _, v := range venues {
		if v.Error == "" {
			return v, samples
		}
	}
	if len(venues) > 0 {
		return venues[0], samples
	}
	return domain.PrintSizeVenue{Exchange: domain.Exchange(ex), Symbol: symbol, Error: "print size not configured", Summary: "print size not configured"}, samples
}

func venueSamplesFromPrintSize(venues []domain.PrintSizeVenue) []domain.VenueSample {
	out := make([]domain.VenueSample, 0, len(venues))
	for _, v := range venues {
		if v.Exchange == "" || v.Exchange == "all" {
			continue
		}
		out = append(out, domain.VenueSample{Exchange: v.Exchange, OK: v.Error == "", At: v.Newest})
	}
	return out
}

func (s *Service) printSizeOne(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) domain.PrintSizeVenue {
	out := domain.PrintSizeVenue{Exchange: ex, Symbol: symbol, Windows: []domain.PrintSizeWindow{}}
	if s.printPort(ex) == nil {
		out.Error = "print size not configured"
		out.Summary = out.Error
		return out
	}
	prints, err := s.tapePrints(ctx, ex, symbol, from, to)
	if err != nil {
		out.Error = err.Error()
		out.Summary = out.Error
		return out
	}
	return domain.BuildPrintSizeVenue(ex, symbol, prints, to)
}

// countedPrints is the live tape when it covers [from, to], otherwise stored
// flow minutes. ok is false when neither source has fills, so the caller can
// leave the read unknown instead of walking public trade history.
func (s *Service) countedPrints(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) ([]domain.TakerPrint, bool) {
	p := s.printPort(ex)
	if p == nil {
		return nil, false
	}
	if mem, ok := p.(domain.MemoryPrintPort); ok {
		if prints, covered := mem.MemoryPrints(symbol, from, to); covered {
			return prints, true
		}
	}
	if s == nil || s.flowHist == nil {
		return nil, false
	}
	cells, err := s.flowHist.ListFlowCells(ctx, ex, symbol, from, to)
	if err != nil || len(cells) == 0 {
		return nil, false
	}
	prints := domain.PrintsFromFlowCells(cells)
	if len(prints) == 0 {
		return nil, false
	}
	return prints, true
}

// tapePrints prefers the live tape, then stored flow minutes, then the
// venue's public trade history.
func (s *Service) tapePrints(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) ([]domain.TakerPrint, error) {
	if prints, ok := s.countedPrints(ctx, ex, symbol, from, to); ok {
		return prints, nil
	}
	return s.fetchPrintRangeResult(ctx, ex, symbol, from, to)
}

func (s *Service) fetchPrintRange(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) []domain.TakerPrint {
	prints, _ := s.fetchPrintRangeResult(ctx, ex, symbol, from, to)
	return prints
}

func (s *Service) fetchPrintRangeResult(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) ([]domain.TakerPrint, error) {
	p := s.printPort(ex)
	if p == nil {
		return nil, fmt.Errorf("prints not configured")
	}
	if rp, ok := p.(domain.PrintRangePort); ok {
		got, err := rp.ListPrints(ctx, symbol, from, to)
		if err == nil {
			return got, nil
		}
	}
	got, err := p.GetRecentPrints(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return domain.FilterPrintsInRange(got, from, to), nil
}
