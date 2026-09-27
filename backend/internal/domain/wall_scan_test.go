package domain

import (
	"testing"
	"time"
)

func TestWallMemory_ExportFlickerAfterShortPull(t *testing.T) {
	m := NewWallMemory()
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	walls := []OrderBookWall{testBidWall("100")}
	m.Observe(now, "binance", "SOLUSDT", walls)
	gone := now.Add(5 * time.Second)
	m.Observe(gone, "binance", "SOLUSDT", nil)
	hits := m.Export(gone.Add(wallAbsenceGrace))
	if len(hits) != 1 || hits[0].Kind != WallScanKindFlicker {
		t.Fatalf("want one flicker, got %+v", hits)
	}
	if hits[0].Symbol != "SOLUSDT" || hits[0].Present {
		t.Fatalf("hit %+v", hits[0])
	}
}

func TestWallMemory_ExportSkipsGraceGap(t *testing.T) {
	m := NewWallMemory()
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	m.Observe(now, "binance", "BTCUSDT", []OrderBookWall{testBidWall("100")})
	at := now.Add(2 * time.Second)
	m.Observe(at, "binance", "BTCUSDT", nil)
	if hits := m.Export(at); len(hits) != 0 {
		t.Fatalf("gap inside grace is not a flicker: %+v", hits)
	}
}

func TestBuildWallScan_SuspiciousBeforeFlicker(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	rows := []WallScanHit{
		{Exchange: "bybit", Symbol: "ETHUSDT", Kind: WallScanKindFlicker, Notional: "900000", Side: "ask"},
		{Exchange: "binance", Symbol: "SOLUSDT", Kind: WallScanKindSuspicious, AppearCount: 4, Notional: "1000", Side: "bid"},
		{Exchange: "coinbase", Symbol: "BTC-USD", Kind: WallScanKindSuspicious, AppearCount: 6, Notional: "500", Side: "ask"},
	}
	got := BuildWallScan("all", "USDT", 20, rows, now, 10)
	if got.Suspicious != 2 || got.Flickers != 1 {
		t.Fatalf("counts %+v", got)
	}
	if got.Hits[0].Symbol != "BTC-USD" || got.Hits[1].Symbol != "SOLUSDT" || got.Hits[2].Symbol != "ETHUSDT" {
		t.Fatalf("order %+v", got.Hits)
	}
	only := BuildWallScan("binance", "usdt", 20, rows, now, 10)
	if len(only.Hits) != 1 || only.Hits[0].Symbol != "SOLUSDT" || only.Quote != "USDT" {
		t.Fatalf("filter %+v", only)
	}
}

func TestParseWallScanExchange(t *testing.T) {
	if _, err := ParseWallScanExchange("okx"); err == nil {
		t.Fatal("okx is not a spot book")
	}
	got, err := ParseWallScanExchange("")
	if err != nil || got != "all" {
		t.Fatalf("%s %v", got, err)
	}
}
