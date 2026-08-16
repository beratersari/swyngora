package domain

import (
	"testing"
	"time"
)

func testHeatBook(ex Exchange, symbol, mid string, bids, asks []OrderBookLevel) OrderBook {
	return OrderBook{
		Exchange:  ex,
		Symbol:    symbol,
		LastPrice: mid,
		GroupSize: "1",
		Bids:      bids,
		Asks:      asks,
	}
}

func TestHeatmapTape_RecordsAndWindows(t *testing.T) {
	tape := NewHeatmapTape()
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	bid := []OrderBookLevel{{Price: "99", Notional: "1000", IsWall: true}}
	ask := []OrderBookLevel{{Price: "101", Notional: "800"}}

	tape.Record(now, testHeatBook(ExchangeBinance, "btcusdt", "100", bid, ask))
	tape.Record(now.Add(3*time.Second), testHeatBook(ExchangeBinance, "BTCUSDT", "100.2", bid, ask))
	tape.Record(now.Add(6*time.Second), testHeatBook(ExchangeBinance, "BTCUSDT", "100.4", bid, ask))

	got := tape.View("binance", "BTCUSDT", 10*time.Minute)
	if got.Symbol != "BTCUSDT" || got.Exchange != ExchangeBinance {
		t.Fatalf("meta %+v", got)
	}
	if got.GroupSize != "1" || got.WindowSeconds != 600 {
		t.Fatalf("window/group %+v", got)
	}
	if len(got.Columns) != 3 {
		t.Fatalf("cols=%d", len(got.Columns))
	}
	if got.Columns[0].Mid != "100" || got.Columns[2].Mid != "100.4" {
		t.Fatalf("mids %+v", got.Columns)
	}
	if !got.Columns[0].Bids[0].IsWall || got.Columns[0].Asks[0].Notional != "800" {
		t.Fatalf("levels %+v", got.Columns[0])
	}
	if got.From != now || got.To != now.Add(6*time.Second) {
		t.Fatalf("from/to %s %s", got.From, got.To)
	}

	short := tape.View("binance", "BTCUSDT", 4*time.Second)
	if len(short.Columns) != 2 {
		t.Fatalf("short window cols=%d", len(short.Columns))
	}
}

func TestHeatmapTape_ReplacesFastPoll(t *testing.T) {
	tape := NewHeatmapTape()
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	bid := []OrderBookLevel{{Price: "99", Notional: "1"}}
	tape.Record(now, testHeatBook(ExchangeBinance, "ETHUSDT", "100", bid, nil))
	tape.Record(now.Add(400*time.Millisecond), testHeatBook(ExchangeBinance, "ETHUSDT", "101", bid, nil))
	got := tape.View("binance", "ETHUSDT", time.Minute)
	if len(got.Columns) != 1 {
		t.Fatalf("cols=%d", len(got.Columns))
	}
	if got.Columns[0].Mid != "101" {
		t.Fatalf("expected replace, got %+v", got.Columns[0])
	}
}

func TestHeatmapTape_SkipsEmptyNotionalAndUnknownPair(t *testing.T) {
	tape := NewHeatmapTape()
	now := time.Now().UTC()
	tape.Record(now, testHeatBook(ExchangeBinance, "SOLUSDT", "20",
		[]OrderBookLevel{{Price: "19", Notional: "0"}, {Price: "18", Notional: "nope"}},
		nil,
	))
	got := tape.View("binance", "SOLUSDT", time.Minute)
	if len(got.Columns) != 1 || len(got.Columns[0].Bids) != 0 {
		t.Fatalf("zero notionals should drop: %+v", got.Columns)
	}
	empty := tape.View("binance", "MISSING", time.Minute)
	if len(empty.Columns) != 0 || empty.Symbol != "MISSING" {
		t.Fatalf("%+v", empty)
	}
}

func TestHeatmapTape_KeepsFifteenMinutes(t *testing.T) {
	if HeatmapMaxColumns < 15*60 {
		t.Fatalf("max columns %d cannot hold a 15m 1s tape", HeatmapMaxColumns)
	}
	tape := NewHeatmapTape()
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	bid := []OrderBookLevel{{Price: "99", Notional: "1"}}
	// 16 minutes of 1s samples — older than 15m must drop from the view, not the tape cap.
	for i := 0; i <= 16*60; i++ {
		tape.Record(now.Add(time.Duration(i)*time.Second), testHeatBook(ExchangeBinance, "BTCUSDT", "100", bid, nil))
	}
	got := tape.View("binance", "BTCUSDT", 15*time.Minute)
	if got.WindowSeconds != 900 {
		t.Fatalf("window %d", got.WindowSeconds)
	}
	span := got.To.Sub(got.From)
	if span < 14*time.Minute+50*time.Second || span > 15*time.Minute {
		t.Fatalf("15m view span %s from=%s to=%s cols=%d", span, got.From, got.To, len(got.Columns))
	}
}

func TestClampHeatmapWindowSeconds(t *testing.T) {
	if got := ClampHeatmapWindowSeconds(0); got != 600 {
		t.Fatalf("default %d", got)
	}
	if got := ClampHeatmapWindowSeconds(10); got != 60 {
		t.Fatalf("min %d", got)
	}
	if got := ClampHeatmapWindowSeconds(99999); got != 1800 {
		t.Fatalf("max %d", got)
	}
	if got := ClampHeatmapWindowSeconds(300); got != 300 {
		t.Fatalf("passthrough %d", got)
	}
}
