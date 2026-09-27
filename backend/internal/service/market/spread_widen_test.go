package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestScanWideSpreads_OnlyOpenedBooks(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	now := time.Unix(1_700_000_000, 0).UTC()
	mem := svc.spreadMemory()
	for i := 0; i < 4; i++ {
		mem.Observe(now.Add(time.Duration(i)*time.Second), domain.ExchangeBinance, "BTCUSDT", domain.RawOrderBook{
			Bids: []domain.PriceLevel{{Price: 100, Quantity: 1}},
			Asks: []domain.PriceLevel{{Price: 100.01, Quantity: 1}},
		})
	}
	mem.Observe(now.Add(10*time.Second), domain.ExchangeBinance, "BTCUSDT", domain.RawOrderBook{
		Bids: []domain.PriceLevel{{Price: 100, Quantity: 1}},
		Asks: []domain.PriceLevel{{Price: 100.08, Quantity: 1}},
	})
	got, err := svc.ScanWideSpreads(context.Background(), "all", "USDT", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Hits) != 1 || got.Hits[0].Symbol != "BTCUSDT" {
		t.Fatalf("%+v", got)
	}
	one, err := svc.GetWideSpread(context.Background(), "binance", "BTCUSDT")
	if err != nil || !one.Opened {
		t.Fatalf("%v %+v", err, one)
	}
}
