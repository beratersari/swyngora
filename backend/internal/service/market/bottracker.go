package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const botTrackerDisclaimer = "Repeated same-size aggressive futures prints. A size is flagged when it hits at least 4 times inside any 60 seconds during the last 5 minutes, buy and sell kept apart. Sizes within about 0.2% count as the same clip. Prints under 5,000 USDT are ignored. Binance pages aggTrades; Bybit and OKX use the newest public trades. This is a repeated-size pattern, not proof of a specific bot. Informational only — not financial advice."

// GetBotTracker returns same-size market orders that repeat in a short burst.
func (s *Service) GetBotTracker(ctx context.Context, exchange, symbol string) (*domain.BotTrackerReport, error) {
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
	from := now.Add(-domain.BotTrackerWindow)
	out := &domain.BotTrackerReport{
		Symbol: symbol, Exchange: ex, AsOf: now,
		Window: "5m", Burst: "60s", MinCount: domain.BotTrackerMinCount,
		Venues: make([]domain.BotTrackerVenue, 0, len(want)),
		Note:   botTrackerDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.botTrackerOne(ctx, v, symbol, from, now)
			mu.Lock()
			out.Venues = append(out.Venues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	out.Clusters = domain.MergeBotClusters(out.Venues)
	for _, c := range out.Clusters {
		if c.Side == domain.TakerSideBuy {
			out.BuyCount += c.Count
		} else {
			out.SellCount += c.Count
		}
	}
	out.Delta = out.BuyCount - out.SellCount
	out.Summary = domain.ExplainBotReport(out.Clusters, out.Delta)
	return out, nil
}

func (s *Service) botTrackerOne(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) domain.BotTrackerVenue {
	out := domain.BotTrackerVenue{Exchange: ex, Symbol: symbol, Clusters: []domain.BotCluster{}}
	if s.printPort(ex) == nil {
		out.Error = "bot tracker not configured"
		out.Summary = out.Error
		return out
	}
	prints := s.fetchPrintRange(ctx, ex, symbol, from, to)
	return domain.BuildBotTracker(ex, symbol, prints, to)
}
