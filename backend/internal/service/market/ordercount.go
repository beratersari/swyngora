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

const orderCountOverviewDisclaimer = "Whole-market order count sums aggressive futures print counts across the most liquid USDT pairs, then divides. A busier coin counts for more of the ratio. The coin counts are one vote each: how many pairs lean buy, sell, or balanced. Quiet pairs had no prints in that window. Windows are 1m, 5m, and 15m. Informational only — not financial advice."

// GetOrderCountOverview reports the print-count ratio across liquid pairs.
func (s *Service) GetOrderCountOverview(ctx context.Context, exchange, quote string, limit int) (*domain.OrderCountOverview, error) {
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	if quote == "" {
		quote = "USDT"
	}
	limit = domain.ParseOrderCountOverviewLimit(limit)
	symbols, err := s.orderCountUniverse(ctx, ex, quote, limit)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	from := now.Add(-domain.OrderCountLookback())
	coins := make([]domain.OrderCountVenue, len(symbols))
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
			coins[i] = s.orderCountSymbol(ctx, ex, symbol, from, now)
		}(i, symbol)
	}
	wg.Wait()
	out := domain.BuildOrderCountOverview(ex, quote, coins, now)
	out.Note = orderCountOverviewDisclaimer
	return &out, nil
}

func (s *Service) orderCountUniverse(ctx context.Context, ex, quote string, limit int) ([]string, error) {
	listEx := domain.ExchangeBinance
	if ex == string(domain.ExchangeBybit) {
		listEx = domain.ExchangeBybit
	}
	spot, err := s.ListSpotMarkets(ctx, string(listEx), domain.SpotListQuery{
		QuoteAsset: quote,
		SortBy:     domain.SpotSortQuoteVolume,
		Order:      domain.SortDesc,
		Limit:      limit * 3,
	})
	if err != nil && listEx == domain.ExchangeBinance && ex != string(domain.ExchangeBybit) {
		spot, err = s.ListSpotMarkets(ctx, string(domain.ExchangeBybit), domain.SpotListQuery{
			QuoteAsset: quote,
			SortBy:     domain.SpotSortQuoteVolume,
			Order:      domain.SortDesc,
			Limit:      limit * 3,
		})
	}
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, limit)
	seen := map[string]bool{}
	for _, m := range spot.Items {
		if m.Symbol == "" || !domain.IsBreadthEligible(m.BaseAsset) || seen[m.Symbol] {
			continue
		}
		seen[m.Symbol] = true
		out = append(out, m.Symbol)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Service) orderCountSymbol(ctx context.Context, ex, symbol string, from, to time.Time) domain.OrderCountVenue {
	want := configuredFutures(ex, func(v domain.Exchange) bool { return s.printPort(v) != nil })
	venues := make([]domain.OrderCountVenue, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			venues[i] = s.orderCountOne(ctx, v, symbol, from, to)
		}(i, v)
	}
	wg.Wait()
	if ex == "all" {
		if combined := domain.CombineOrderCount(symbol, venues); combined != nil {
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
	return domain.OrderCountVenue{Exchange: domain.Exchange(ex), Symbol: symbol, Error: "order count not configured", Summary: "order count not configured"}
}

func (s *Service) orderCountOne(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) domain.OrderCountVenue {
	if s.printPort(ex) == nil {
		return domain.OrderCountVenue{Exchange: ex, Symbol: symbol, Error: "order count not configured", Summary: "order count not configured"}
	}
	prints := s.fetchPrintRange(ctx, ex, symbol, from, to)
	return domain.BuildOrderCount(ex, symbol, prints, to)
}
