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

func TestGetPrintSizeOverview_SumsLiquidPairs(t *testing.T) {
	now := time.Now().UTC()
	var btc, eth []domain.TakerPrint
	for i := 0; i < 4; i++ {
		btc = append(btc, domain.TakerPrint{Side: domain.TakerSideBuy, Notional: 20_000, Time: now.Add(-20 * time.Minute)})
		btc = append(btc, domain.TakerPrint{Side: domain.TakerSideBuy, Notional: 100_000, Time: now.Add(-2 * time.Minute)})
		eth = append(eth, domain.TakerPrint{Side: domain.TakerSideBuy, Notional: 40_000, Time: now.Add(-20 * time.Minute)})
		eth = append(eth, domain.TakerPrint{Side: domain.TakerSideBuy, Notional: 10_000, Time: now.Add(-2 * time.Minute)})
	}
	fm := &fakeMarket{
		spot: []domain.SpotMarket{
			{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000"},
			{Symbol: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "500"},
		},
		prints: map[string][]domain.TakerPrint{"BTCUSDT": btc, "ETHUSDT": eth},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, &fakeSupply{})
	got, err := svc.GetPrintSizeOverview(context.Background(), "binance", "USDT", 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Universe != len(domain.DefaultFuturesHistorySymbols) {
		t.Fatalf("universe %d, want the recorder seed list", got.Universe)
	}
	var w domain.PrintSizeMarketWindow
	for _, x := range got.Windows {
		if x.Window == domain.PrintSizeWindow15m {
			w = x
		}
	}
	// Current 4*100k + 4*10k = 440k over 8 fills. Prior 4*20k + 4*40k = 240k over 8.
	if w.Buy.Avg < 50_000 || w.Buy.Kind == "" || w.LargerCoins+w.SmallerCoins == 0 {
		t.Fatalf("%+v", w)
	}
}

func TestGetPrintSize_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetPrintSize(context.Background(), "all", ""); err == nil {
		t.Fatal("expected error")
	}
}
