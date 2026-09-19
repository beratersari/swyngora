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
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
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
