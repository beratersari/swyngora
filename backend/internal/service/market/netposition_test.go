package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type npOI struct {
	series *domain.OpenInterestSeries
}

func (f npOI) GetOpenInterestSeries(context.Context, string) (*domain.OpenInterestSeries, error) {
	return f.series, nil
}

type npTaker struct {
	buckets []domain.TakerBucket
}

func (f npTaker) GetTakerFlow(context.Context, string) (*domain.TakerVenueFlow, error) {
	return &domain.TakerVenueFlow{}, nil
}

func (f npTaker) GetTakerBuckets(context.Context, string) ([]domain.TakerBucket, error) {
	return f.buckets, nil
}

func TestGetNetPosition_UsesOIAndTaker(t *testing.T) {
	now := time.Now().UTC()
	svc := New(nil, nil).WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
		domain.ExchangeBinance: npOI{series: &domain.OpenInterestSeries{
			Current: domain.OpenInterestPoint{Time: now, Value: 130},
			History: []domain.OpenInterestPoint{{Time: now.Add(-15 * time.Minute), Value: 100}},
		}},
	}).WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: npTaker{buckets: []domain.TakerBucket{
			{Start: now.Add(-10 * time.Minute), BuyNotional: 80, SellNotional: 20},
		}},
	})
	got, err := svc.GetNetPosition(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || len(got.Windows) != 3 {
		t.Fatalf("%+v", got)
	}
	var m15 domain.NetPositionWindow
	for _, w := range got.Windows {
		if w.Window == "15m" {
			m15 = w
		}
	}
	if !m15.Complete || m15.NetLong <= 0 {
		t.Fatalf("15m %+v", m15)
	}
}

func TestGetNetPosition_RejectsOKX(t *testing.T) {
	svc := New(nil, nil)
	_, err := svc.GetNetPosition(context.Background(), "okx", "BTCUSDT")
	if err == nil {
		t.Fatal("expected error")
	}
}
