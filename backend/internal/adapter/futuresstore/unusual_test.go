package futuresstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestUnusualAggregatesIgnoreRowCaps(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	old := now.Add(-40 * 24 * time.Hour)
	if _, err := s.InsertSnapshot(ctx, domain.FuturesSnapshot{
		Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		SampledAt: old, Value: 10,
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2501; i++ {
		if _, err := s.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			SampledAt: now.Add(-time.Duration(i) * 5 * time.Minute), Value: 20,
		}); err != nil {
			t.Fatal(err)
		}
	}
	oi, err := s.HourlyOpenInterest(ctx, "binance", "BTCUSDT", now.Add(-90*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(oi) < 2 {
		t.Fatalf("hourly buckets = %d", len(oi))
	}
	if !oi[0].At.Equal(old.Truncate(time.Hour)) || oi[0].Value != 10 {
		t.Fatalf("oldest hourly point dropped: %+v", oi[0])
	}
	if len(oi) >= 2500 {
		t.Fatalf("aggregate should collapse 5-minute rows, got %d buckets", len(oi))
	}

	if _, err := s.InsertSnapshot(ctx, domain.FuturesSnapshot{
		Metric: domain.FuturesMetricFunding, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		SampledAt: now.Add(-8 * time.Hour), Predicted: false, FundingRate: 0.0002,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InsertSnapshot(ctx, domain.FuturesSnapshot{
		Metric: domain.FuturesMetricFunding, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		SampledAt: now.Add(-5 * time.Minute), Predicted: true, FundingRate: 0.01,
	}); err != nil {
		t.Fatal(err)
	}
	settled, err := s.SettledFunding(ctx, "binance", "BTCUSDT", now.Add(-90*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(settled) != 1 || settled[0].Value != 0.0002 {
		t.Fatalf("predicted row leaked into settlements: %+v", settled)
	}
	pred, ok, err := s.LatestPredictedFunding(ctx, "binance", "BTCUSDT")
	if err != nil || !ok || pred.Value != 0.01 {
		t.Fatalf("predicted %+v ok %v err %v", pred, ok, err)
	}

	for _, n := range []float64{10, 15} {
		if _, err := s.InsertLiquidation(ctx, domain.LiquidationEvent{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: "long",
			Price: 100, Quantity: n, Notional: n, Time: now.Add(-2 * time.Hour).Add(time.Duration(n) * time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.InsertLiquidation(ctx, domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: "short",
		Price: 100, Quantity: 7, Notional: 7, Time: now.Add(-3 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	liq, err := s.HourlyLiquidationNotional(ctx, "binance", "BTCUSDT", now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(liq) != 2 || liq[0].Value != 7 || liq[1].Value != 25 {
		t.Fatalf("%+v", liq)
	}

	day := now.Truncate(24 * time.Hour)
	if _, err := s.UpsertTakerBuckets(ctx, []domain.TakerBucket{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: day.Add(time.Hour), BuyNotional: 10, SellNotional: 4},
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: day.Add(2 * time.Hour), BuyNotional: 3, SellNotional: 1},
	}); err != nil {
		t.Fatal(err)
	}
	cvd, err := s.DailyCVD(ctx, "binance", "BTCUSDT", day)
	if err != nil {
		t.Fatal(err)
	}
	if len(cvd) != 1 || cvd[0].Value != 8 || !cvd[0].At.Equal(day) {
		t.Fatalf("%+v", cvd)
	}
}
