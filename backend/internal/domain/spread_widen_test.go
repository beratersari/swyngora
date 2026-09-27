package domain

import (
	"testing"
	"time"
)

func topOfBook(bid, ask float64) RawOrderBook {
	return RawOrderBook{
		Bids: []PriceLevel{{Price: bid, Quantity: 1}},
		Asks: []PriceLevel{{Price: ask, Quantity: 1}},
	}
}

func TestBuildSpreadScan_RanksOpenedBooks(t *testing.T) {
	mem := NewSpreadMemory()
	now := time.Unix(1_700_000_000, 0).UTC()
	for i := 0; i < 4; i++ {
		mem.Observe(now.Add(time.Duration(i)*time.Second), ExchangeBinance, "BTCUSDT", topOfBook(100, 100.01))
		mem.Observe(now.Add(time.Duration(i)*time.Second), ExchangeBybit, "ETHUSDT", topOfBook(100, 100.02))
	}
	mem.Observe(now.Add(10*time.Second), ExchangeBinance, "BTCUSDT", topOfBook(100, 100.08))
	mem.Observe(now.Add(10*time.Second), ExchangeBybit, "ETHUSDT", topOfBook(100, 100.021))

	got := BuildSpreadScan("all", "USDT", mem.Tracks(), 10)
	if len(got.Hits) != 1 || got.Hits[0].Symbol != "BTCUSDT" || got.Hits[0].Ratio < 1.5 {
		t.Fatalf("%+v summary %s", got.Hits, got.Summary)
	}
	one := SpreadReportFor("ETHUSDT", "bybit", mem.Tracks())
	if one.Opened {
		t.Fatalf("eth should stay normal: %+v", one)
	}
	btc := SpreadReportFor("BTCUSDT", "all", mem.Tracks())
	if !btc.Opened || btc.Exchange != "binance" {
		t.Fatalf("%+v", btc)
	}
}

func TestTopSpreadPct(t *testing.T) {
	pct, ok := TopSpreadPct(topOfBook(100, 100.1))
	if !ok || pct < 0.09 || pct > 0.11 {
		t.Fatalf("%v %v", pct, ok)
	}
}
