package market

import (
	"context"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestScanOrderBookImpact_RanksWiderBookFirst(t *testing.T) {
	fm := &fakeMarket{
		spot: []domain.SpotMarket{
			{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000"},
			{Symbol: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "500"},
		},
		books: map[string]*domain.RawOrderBook{
			"BTCUSDT": {
				Symbol: "BTCUSDT",
				Bids:   []domain.PriceLevel{{Price: 99.9, Quantity: 100}},
				Asks:   []domain.PriceLevel{{Price: 100.1, Quantity: 100}},
			},
			"ETHUSDT": {
				Symbol: "ETHUSDT",
				Bids:   []domain.PriceLevel{{Price: 99, Quantity: 100}},
				Asks:   []domain.PriceLevel{{Price: 110, Quantity: 100}},
			},
		},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, &fakeSupply{})
	got, err := svc.ScanOrderBookImpact(context.Background(), "binance", "USDT", "buy", 1000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Universe != 2 || len(got.Hits) < 2 || got.Hits[0].Symbol != "ETHUSDT" {
		t.Fatalf("%+v", got.Hits)
	}
	if got.Hits[0].SlippagePct <= got.Hits[1].SlippagePct {
		t.Fatalf("eth %v btc %v", got.Hits[0].SlippagePct, got.Hits[1].SlippagePct)
	}
}
