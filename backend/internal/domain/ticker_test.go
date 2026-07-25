package domain

import (
	"testing"
	"time"
)

// Smoke tests for Ticker24h field layout (types-only domain model).

func TestTicker24h_Fields(t *testing.T) {
	open := time.Unix(1_700_000_000, 0).UTC()
	closeT := open.Add(24 * time.Hour)
	tk := Ticker24h{
		Symbol:             "BTCUSDT",
		PriceChange:        "-100",
		PriceChangePercent: "-1.5",
		LastPrice:          "64000",
		OpenPrice:          "65000",
		HighPrice:          "66000",
		LowPrice:           "63000",
		Volume:             "1000.5",
		QuoteVolume:        "64000000",
		OpenTime:           open,
		CloseTime:          closeT,
		TradeCount:         42,
	}
	if tk.Symbol != "BTCUSDT" {
		t.Fatalf("symbol=%s", tk.Symbol)
	}
	if tk.Volume != "1000.5" || tk.QuoteVolume != "64000000" {
		t.Fatalf("volume fields: base=%s quote=%s", tk.Volume, tk.QuoteVolume)
	}
	if tk.TradeCount != 42 {
		t.Fatalf("trades=%d", tk.TradeCount)
	}
	if !tk.CloseTime.After(tk.OpenTime) {
		t.Fatal("close should be after open")
	}
	// Zero value is valid empty snapshot
	var zero Ticker24h
	if zero.Symbol != "" || zero.TradeCount != 0 {
		t.Fatalf("zero=%+v", zero)
	}
}
