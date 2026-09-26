package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetOrderCount_BuyHeavyOneMinute(t *testing.T) {
	now := time.Now().UTC()
	var prints []domain.TakerPrint
	for i := 0; i < 9; i++ {
		prints = append(prints, domain.TakerPrint{Side: domain.TakerSideBuy, Time: now.Add(-15 * time.Second)})
	}
	prints = append(prints, domain.TakerPrint{Side: domain.TakerSideSell, Time: now.Add(-15 * time.Second)})
	fm := &fakeMarket{prints: map[string][]domain.TakerPrint{"BTCUSDT": prints}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, &fakeSupply{})
	got, err := svc.GetOrderCount(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Combined != nil || len(got.Venues) != 1 {
		t.Fatalf("venues %d", len(got.Venues))
	}
	var w domain.OrderCountWindow
	for _, x := range got.Venues[0].Windows {
		if x.Window == domain.OrderCount1m {
			w = x
		}
	}
	if w.BuyCount != 9 || w.SellCount != 1 || w.Lean != domain.OrderCountLeanBuy {
		t.Fatalf("%+v", w)
	}
}
