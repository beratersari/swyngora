package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetBotTracker_ReturnsRepeatedBuy(t *testing.T) {
	now := time.Now().UTC()
	var prints []domain.TakerPrint
	for i := 0; i < 5; i++ {
		prints = append(prints, domain.TakerPrint{
			Side: domain.TakerSideBuy, Notional: 50_000,
			Time: now.Add(-20*time.Second + time.Duration(i)*3*time.Second),
		})
	}
	fm := &fakeMarket{prints: map[string][]domain.TakerPrint{"BTCUSDT": prints}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, &fakeSupply{})
	got, err := svc.GetBotTracker(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Clusters) != 1 || got.Clusters[0].Count != 5 || got.Delta != 5 {
		t.Fatalf("%+v summary=%s", got.Clusters, got.Summary)
	}
}
