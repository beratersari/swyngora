package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetPrintSize_BuyAvgJumped(t *testing.T) {
	now := time.Now().UTC()
	var prints []domain.TakerPrint
	for i := 0; i < 20; i++ {
		prints = append(prints, domain.TakerPrint{
			Side: domain.TakerSideBuy, Notional: 20_000,
			Time: now.Add(-20*time.Minute + time.Duration(i)*time.Second),
		})
	}
	for i := 0; i < 4; i++ {
		prints = append(prints, domain.TakerPrint{
			Side: domain.TakerSideBuy, Notional: 180_000,
			Time: now.Add(-2*time.Minute + time.Duration(i)*time.Second),
		})
	}
	fm := &fakeMarket{prints: map[string][]domain.TakerPrint{"BTCUSDT": prints}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, &fakeSupply{})
	got, err := svc.GetPrintSize(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Combined != nil || len(got.Venues) != 1 {
		t.Fatalf("%+v", got)
	}
	var w domain.PrintSizeWindow
	for _, x := range got.Venues[0].Windows {
		if x.Window == domain.PrintSizeWindow15m {
			w = x
		}
	}
	if w.Buy.Kind != domain.PrintSizeKindJumped {
		t.Fatalf("15m %+v summary=%s", w.Buy, got.Venues[0].Summary)
	}
}

func TestGetPrintSize_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetPrintSize(context.Background(), "all", ""); err == nil {
		t.Fatal("expected error")
	}
}
