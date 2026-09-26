package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const printSizeDisclaimer = "Average aggressive futures print size (USDT notional per taker fill). Compare the last 5m / 15m to the same length just before: a jump from 20K to 180K is a few large market orders, not many small ones. Binance pages aggTrades; Bybit uses the newest ~1000 public trades (minutes of tape when busy). Informational only — not financial advice."

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

const printSizeOverviewDisclaimer = "Whole-market print size sums aggressive futures notional and fill counts across the most liquid USDT pairs, then divides. A busier coin weighs more in the average. Each pair also votes whether its own average got larger, smaller, a few large orders, or many small ones. Windows are 5m and 15m versus the same length just before. Informational only — not financial advice."

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
	symbols, err := s.orderCountUniverse(ctx, ex, quote, limit)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	from := now.Add(-domain.PrintSizeLookback())
	coins := make([]domain.PrintSizeVenue, len(symbols))
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
			coins[i] = s.printSizeSymbol(ctx, ex, symbol, from, now)
		}(i, symbol)
	}
	wg.Wait()
	out := domain.BuildPrintSizeOverview(ex, quote, coins, now)
	out.Note = printSizeOverviewDisclaimer
	return &out, nil
}

func (s *Service) printSizeSymbol(ctx context.Context, ex, symbol string, from, to time.Time) domain.PrintSizeVenue {
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
	if ex == "all" {
		if combined := domain.CombinePrintSizeVenues(symbol, venues); combined != nil {
			return *combined
		}
	}
	for _, v := range venues {
		if v.Error == "" {
			return v
		}
	}
	if len(venues) > 0 {
		return venues[0]
	}
	return domain.PrintSizeVenue{Exchange: domain.Exchange(ex), Symbol: symbol, Error: "print size not configured", Summary: "print size not configured"}
}

func (s *Service) printSizeOne(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) domain.PrintSizeVenue {
	out := domain.PrintSizeVenue{Exchange: ex, Symbol: symbol, Windows: []domain.PrintSizeWindow{}}
	prints := s.fetchPrintRange(ctx, ex, symbol, from, to)
	if len(prints) == 0 {
		p := s.printPort(ex)
		if p == nil {
			out.Error = "print size not configured"
			out.Summary = out.Error
			return out
		}
	}
	got := domain.BuildPrintSizeVenue(ex, symbol, prints, to)
	if len(prints) == 0 && got.Summary == "No aggressive print-size tape yet." {
		if s.printPort(ex) == nil {
			got.Error = "print size not configured"
			got.Summary = got.Error
		}
	}
	return got
}

func (s *Service) fetchPrintRange(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) []domain.TakerPrint {
	p := s.printPort(ex)
	if p == nil {
		return nil
	}
	if rp, ok := p.(domain.PrintRangePort); ok {
		got, err := rp.ListPrints(ctx, symbol, from, to)
		if err == nil {
			return got
		}
	}
	got, err := p.GetRecentPrints(ctx, symbol)
	if err != nil {
		return nil
	}
	return domain.FilterPrintsInRange(got, from, to)
}
