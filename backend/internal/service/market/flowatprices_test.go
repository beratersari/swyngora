package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetFlowAtPrices_LevelAtPrice(t *testing.T) {
	now := time.Now().UTC()
	prints := []domain.TakerPrint{
		{Side: domain.TakerSideBuy, Price: 65520, Notional: 8_000_000, Time: now.Add(-4 * time.Minute)},
		{Side: domain.TakerSideSell, Price: 65520, Notional: 2_000_000, Time: now.Add(-3 * time.Minute)},
		{Side: domain.TakerSideSell, Price: 65400, Notional: 500_000, Time: now.Add(-2 * time.Minute)},
	}
	fm := &fakeMarket{
		prints: map[string][]domain.TakerPrint{"BTCUSDT": prints},
		ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "65500"},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, &fakeSupply{})
	got, err := svc.GetFlowAtPrices(context.Background(), "binance", "BTCUSDT", "15m", nil, nil, 10, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Combined != nil || len(got.Venues) != 1 {
		t.Fatalf("%+v", got)
	}
	v := got.Venues[0]
	if v.POC.Price != 65520 {
		t.Fatalf("poc %+v", v.POC)
	}
	var row domain.FlowAtPricesLevel
	for _, lv := range v.Levels {
		if lv.Price == 65520 {
			row = lv
		}
	}
	if row.BuyVolume != 8_000_000 || row.SellVolume != 2_000_000 || row.Delta != 6_000_000 || row.Trades != 2 {
		t.Fatalf("65520 %+v", row)
	}
}

func TestGetFlowAtPrices_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetFlowAtPrices(context.Background(), "all", "", "15m", nil, nil, 0, 0, 0); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetFlowAtPrices_BadWindow(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetFlowAtPrices(context.Background(), "binance", "BTCUSDT", "24h", nil, nil, 0, 0, 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetFlowAtPrices_BadPriceBand(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetFlowAtPrices(context.Background(), "binance", "BTCUSDT", "15m", nil, nil, 0, 65600, 65400)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
