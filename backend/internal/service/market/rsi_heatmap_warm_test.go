package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestStartRSIHeatmapWarmer_NilService(t *testing.T) {
	var s *Service
	s.StartRSIHeatmapWarmer(context.Background())
}

func TestStartRSIHeatmapWarmer_DefaultEveryWhenUnset(t *testing.T) {
	prev := rsiHeatWarmEvery
	rsiHeatWarmEvery = 0
	t.Cleanup(func() { rsiHeatWarmEvery = prev })
	svc := New(&indicatorMarket{}, &fakeSupply{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		svc.StartRSIHeatmapWarmer(ctx)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("warmer did not stop")
	}
}

func TestStartRSIHeatmapWarmer_FillsDefaultCacheThenStops(t *testing.T) {
	prev := rsiHeatWarmEvery
	rsiHeatWarmEvery = 15 * time.Millisecond
	t.Cleanup(func() { rsiHeatWarmEvery = prev })
	svc := New(&indicatorMarket{}, &fakeSupply{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		svc.StartRSIHeatmapWarmer(ctx)
		close(done)
	}()
	deadline := time.Now().Add(2 * time.Second)
	key := domain.RSIHeatmapCacheKey(domain.ExchangeBinance, "USDT", string(domain.SpotSortMarketCapCirculating), domain.RSIHeatmapDefaultInterval, domain.DefaultRSIPeriod)
	filled := false
	for time.Now().Before(deadline) {
		if hit, ok := svc.rsiHeat.Get(key); ok && hit != nil && len(hit.Items) > 0 {
			filled = true
			break
		}
		// market-cap sort may fall back to quoteVolume
		alt := domain.RSIHeatmapCacheKey(domain.ExchangeBinance, "USDT", string(domain.SpotSortQuoteVolume), domain.RSIHeatmapDefaultInterval, domain.DefaultRSIPeriod)
		if hit, ok := svc.rsiHeat.Get(alt); ok && hit != nil && len(hit.Items) > 0 {
			filled = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("warmer did not stop")
	}
	if !filled {
		t.Fatal("default map was not warmed")
	}
}

func TestWarmDefaultRSIHeatmap_LogsMissingVenue(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeCoinbase: &indicatorMarket{},
	}, &fakeSupply{})
	svc.warmDefaultRSIHeatmap(context.Background())
}
