package domain

import (
	"strings"
	"testing"
	"time"
)

func icebergAskBook(askPx, askQty float64) RawOrderBook {
	return RawOrderBook{
		Symbol: "BTCUSDT",
		Bids:   []PriceLevel{{Price: askPx - 1, Quantity: 2}},
		Asks:   []PriceLevel{{Price: askPx, Quantity: askQty}, {Price: askPx + 10, Quantity: 1}},
		Live:   true,
	}
}

func TestIcebergMemory_AskRefillsAfterEat(t *testing.T) {
	m := NewIcebergMemory()
	now := time.Date(2026, 8, 16, 14, 0, 0, 0, time.UTC)
	px := 64000.0
	clip := 10.0 // 10 BTC * 64k = 640k notional
	m.ObserveBook(now, "binance", "BTCUSDT", icebergAskBook(px, clip), nil, 25_000)
	// Buyers eat it at the touch.
	m.ObserveBook(now.Add(5*time.Second), "binance", "BTCUSDT", icebergAskBook(px, 0.5), nil, 25_000)
	// Same clip comes back.
	m.ObserveBook(now.Add(15*time.Second), "binance", "BTCUSDT", icebergAskBook(px, clip), nil, 25_000)
	// Eat and refill again.
	m.ObserveBook(now.Add(25*time.Second), "binance", "BTCUSDT", icebergAskBook(px, 0.4), nil, 25_000)
	m.ObserveBook(now.Add(35*time.Second), "binance", "BTCUSDT", icebergAskBook(px, clip), nil, 25_000)

	got := m.Active("binance", "BTCUSDT", 0)
	if len(got) != 1 || got[0].Side != "ask" || got[0].Refills < 2 {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got[0].Summary, "selling") {
		t.Fatalf("summary %s", got[0].Summary)
	}
	walls := []OrderBookWall{{Side: "ask", Price: "64000", Quantity: "10", Notional: "640000"}}
	m.AnnotateWalls("binance", "BTCUSDT", walls)
	if !walls[0].Iceberg || walls[0].Behavior != WallBehaviorIceberg || walls[0].IcebergRefills < 2 {
		t.Fatalf("wall %+v", walls[0])
	}
}

func TestIcebergMemory_BidSide(t *testing.T) {
	m := NewIcebergMemory()
	now := time.Date(2026, 8, 16, 14, 0, 0, 0, time.UTC)
	book := func(qty float64) RawOrderBook {
		return RawOrderBook{
			Bids: []PriceLevel{{Price: 63900, Quantity: qty}},
			Asks: []PriceLevel{{Price: 64010, Quantity: 1}},
		}
	}
	m.ObserveBook(now, "bybit", "BTCUSDT", book(8), nil, 10_000)
	m.ObserveBook(now.Add(4*time.Second), "bybit", "BTCUSDT", book(1), nil, 10_000)
	m.ObserveBook(now.Add(10*time.Second), "bybit", "BTCUSDT", book(8), nil, 10_000)
	m.ObserveBook(now.Add(14*time.Second), "bybit", "BTCUSDT", book(0.5), nil, 10_000)
	m.ObserveBook(now.Add(20*time.Second), "bybit", "BTCUSDT", book(8), nil, 10_000)
	got := m.Active("bybit", "BTCUSDT", 0)
	if len(got) != 1 || got[0].Side != "bid" || got[0].Refills < 2 {
		t.Fatalf("%+v", got)
	}
}

func TestIcebergMemory_FarPullIsNotIceberg(t *testing.T) {
	m := NewIcebergMemory()
	now := time.Date(2026, 8, 16, 14, 0, 0, 0, time.UTC)
	// Wall sits well above the touch, then vanishes (pulled), then returns.
	far := func(qty float64) RawOrderBook {
		return RawOrderBook{
			Bids: []PriceLevel{{Price: 63900, Quantity: 1}},
			Asks: []PriceLevel{{Price: 64010, Quantity: 1}, {Price: 65000, Quantity: qty}},
		}
	}
	m.ObserveBook(now, "binance", "BTCUSDT", far(10), nil, 25_000)
	m.ObserveBook(now.Add(5*time.Second), "binance", "BTCUSDT", far(0), nil, 25_000)
	m.ObserveBook(now.Add(15*time.Second), "binance", "BTCUSDT", far(10), nil, 25_000)
	m.ObserveBook(now.Add(25*time.Second), "binance", "BTCUSDT", far(0), nil, 25_000)
	m.ObserveBook(now.Add(35*time.Second), "binance", "BTCUSDT", far(10), nil, 25_000)
	if got := m.Active("binance", "BTCUSDT", 0); len(got) != 0 {
		t.Fatalf("spoof far from touch should not be iceberg: %+v", got)
	}
}

func TestIcebergMemory_PrintConfirmsLikely(t *testing.T) {
	m := NewIcebergMemory()
	now := time.Date(2026, 8, 16, 14, 0, 0, 0, time.UTC)
	px := 64000.0
	prints := []TakerPrint{{Side: TakerSideBuy, Price: px, Quantity: 8, Notional: px * 8, Time: now}}
	m.ObserveBook(now, "binance", "BTCUSDT", icebergAskBook(px, 10), nil, 25_000)
	m.ObserveBook(now.Add(3*time.Second), "binance", "BTCUSDT", icebergAskBook(px, 1), prints, 25_000)
	m.ObserveBook(now.Add(8*time.Second), "binance", "BTCUSDT", icebergAskBook(px, 10), nil, 25_000)
	m.ObserveBook(now.Add(12*time.Second), "binance", "BTCUSDT", icebergAskBook(px, 1), prints, 25_000)
	m.ObserveBook(now.Add(16*time.Second), "binance", "BTCUSDT", icebergAskBook(px, 10), nil, 25_000)
	got := m.Active("binance", "BTCUSDT", 0)
	if len(got) != 1 || got[0].Confidence != "likely" || got[0].PrintHits < 1 {
		t.Fatalf("%+v", got)
	}
}
