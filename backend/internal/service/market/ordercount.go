package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const orderCountDisclaimer = "Market order count ratio is (buy prints − sell prints) / (buy prints + sell prints), from −1 to +1. It counts aggressive futures fills, not their USDT size. Windows are 1m, 5m, and 15m. Binance pages aggTrades; Bybit and OKX use the newest public trades. Informational only — not financial advice."

// GetOrderCount returns how many aggressive buys versus sells printed.
func (s *Service) GetOrderCount(ctx context.Context, exchange, symbol string) (*domain.OrderCountReport, error) {
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
	from := now.Add(-domain.OrderCountLookback())
	out := &domain.OrderCountReport{
		Symbol: symbol, Exchange: ex, AsOf: now,
		Venues: make([]domain.OrderCountVenue, 0, len(want)),
		Note:   orderCountDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.orderCountOne(ctx, v, symbol, from, now)
			mu.Lock()
			out.Venues = append(out.Venues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" {
		out.Combined = domain.CombineOrderCount(symbol, out.Venues)
	}
	out.Summary = domain.ExplainOrderCountReport(*out)
	return out, nil
}

func (s *Service) orderCountOne(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) domain.OrderCountVenue {
	if s.printPort(ex) == nil {
		return domain.OrderCountVenue{Exchange: ex, Symbol: symbol, Error: "order count not configured", Summary: "order count not configured"}
	}
	prints := s.fetchPrintRange(ctx, ex, symbol, from, to)
	return domain.BuildOrderCount(ex, symbol, prints, to)
}
