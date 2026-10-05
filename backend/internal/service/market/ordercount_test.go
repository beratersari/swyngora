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

func TestGetOrderCountOverview_SumsLiquidPairs(t *testing.T) {
	now := time.Now().UTC()
	fm := &fakeMarket{
		spot: []domain.SpotMarket{
			{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000"},
			{Symbol: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "500"},
			{Symbol: "BTCUPUSDT", BaseAsset: "BTCUP", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "900"},
		},
		prints: map[string][]domain.TakerPrint{
			"BTCUSDT": {
				{Side: domain.TakerSideBuy, Time: now.Add(-10 * time.Second)},
				{Side: domain.TakerSideBuy, Time: now.Add(-10 * time.Second)},
				{Side: domain.TakerSideBuy, Time: now.Add(-10 * time.Second)},
				{Side: domain.TakerSideSell, Time: now.Add(-10 * time.Second)},
			},
			"ETHUSDT": {
				{Side: domain.TakerSideSell, Time: now.Add(-10 * time.Second)},
				{Side: domain.TakerSideSell, Time: now.Add(-10 * time.Second)},
				{Side: domain.TakerSideSell, Time: now.Add(-10 * time.Second)},
				{Side: domain.TakerSideBuy, Time: now.Add(-10 * time.Second)},
			},
		},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, &fakeSupply{})
	got, err := svc.GetOrderCountOverview(context.Background(), "binance", "USDT", 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Universe != len(domain.DefaultFuturesHistorySymbols) {
		t.Fatalf("universe %d, want the recorder seed list", got.Universe)
	}
	var m1 domain.OrderCountMarketWindow
	for _, w := range got.Windows {
		if w.Window == domain.OrderCount1m {
			m1 = w
		}
	}
	if m1.BuyCount != 4 || m1.SellCount != 4 || m1.Ratio != 0 || m1.Lean != domain.OrderCountLeanBalanced {
		t.Fatalf("%+v", m1)
	}
	if m1.BuyCoins != 1 || m1.SellCoins != 1 || m1.QuietCoins != len(domain.DefaultFuturesHistorySymbols)-2 {
		t.Fatalf("leans %+v", m1)
	}
}
