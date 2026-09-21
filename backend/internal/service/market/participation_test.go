package market

import (
	"context"
	"math"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetParticipation_CombinedRatio(t *testing.T) {
	now := time.Now().UTC()
	oi := &fakeOI{ser: &domain.OpenInterestSeries{
		Exchange: domain.ExchangeBinance,
		Current:  domain.OpenInterestPoint{Time: now, Value: 10_300_000_000},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-15 * time.Minute), Value: 10_000_000_000},
			{Time: now.Add(-time.Hour), Value: 10_000_000_000},
			{Time: now.Add(-4 * time.Hour), Value: 10_000_000_000},
		},
	}}
	var buckets []domain.TakerBucket
	for i := 0; i < 3; i++ {
		buckets = append(buckets, domain.TakerBucket{
			Start:        now.Add(-time.Duration(i+1) * 5 * time.Minute),
			BuyNotional:  400_000_000,
			SellNotional: 266_666_666.6666667,
		})
	}
	tk := &fakeTaker{flow: &domain.TakerVenueFlow{Exchange: domain.ExchangeBinance}, buckets: buckets}
	svc := New(&fakeMarket{}, &fakeSupply{}).
		WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{domain.ExchangeBinance: oi}).
		WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{domain.ExchangeBinance: tk})
	got, err := svc.GetParticipation(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || len(got.Venues) != 1 {
		t.Fatalf("%+v", got)
	}
	var w domain.ParticipationWindow
	for _, x := range got.Venues[0].Windows {
		if x.Window == domain.ParticipationWindow15m {
			w = x
		}
	}
	if !w.Complete || math.Abs(w.Ratio-0.15) > 0.001 {
		t.Fatalf("15m %+v", w)
	}
	if w.Lean != domain.ParticipationEntering {
		t.Fatalf("lean=%s", w.Lean)
	}
}

func TestGetParticipation_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetParticipation(context.Background(), "all", ""); err == nil {
		t.Fatal("expected error")
	}
}
