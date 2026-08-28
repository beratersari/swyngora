package market

import (
	"context"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetRSIHeatmap_FillsDots(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 5, 14)
	if err != nil {
		t.Fatal(err)
	}
	if got.Period != 14 || got.Exchange != domain.ExchangeBinance || got.Interval != "1h" {
		t.Fatalf("%+v", got)
	}
	if len(got.Items) == 0 {
		t.Fatal("expected rows")
	}
	if got.AverageRSI == nil {
		t.Fatal("expected average RSI")
	}
	for _, row := range got.Items {
		if row.Error != "" {
			t.Fatalf("%s: %s", row.Symbol, row.Error)
		}
		if row.RSI == nil {
			t.Fatalf("missing RSI for %s", row.Symbol)
		}
		if row.Zone == domain.RSIZoneUnknown {
			t.Fatalf("zone empty for %s", row.Symbol)
		}
		if row.Rank < 1 {
			t.Fatalf("rank=%d", row.Rank)
		}
	}
}

func TestGetRSIHeatmap_DropsStables(t *testing.T) {
	m := &indicatorMarket{}
	m.spot = []domain.SpotMarket{
		{Symbol: "USDCUSDT", BaseAsset: "USDC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "99999", LastPrice: "1"},
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000", LastPrice: "100"},
	}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 5, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Symbol != "BTCUSDT" {
		t.Fatalf("items=%+v", got.Items)
	}
}

func TestGetRSIHeatmap_RejectsBadInterval(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	_, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "3y", "quoteVolume", 5, 14)
	if err == nil {
		t.Fatal("expected invalid interval")
	}
}

func TestGetRSIHeatmap_CacheHit(t *testing.T) {
	svc := New(&indicatorMarket{}, &fakeSupply{})
	first, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 3, 14)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 3, 14)
	if err != nil {
		t.Fatal(err)
	}
	if first.AsOf != second.AsOf {
		t.Fatalf("cache miss: %v vs %v", first.AsOf, second.AsOf)
	}
}
