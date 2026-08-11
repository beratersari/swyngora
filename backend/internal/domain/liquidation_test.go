package domain

import (
	"testing"
	"time"
)

func TestNormalizeLiquidationSymbol(t *testing.T) {
	if got := NormalizeLiquidationSymbol("btc-usd"); got != "BTCUSDT" {
		t.Fatalf("got %s", got)
	}
	if got := NormalizeLiquidationSymbol("ETHUSDT"); got != "ETHUSDT" {
		t.Fatalf("got %s", got)
	}
}

func TestLiquidationSideMapping(t *testing.T) {
	s, err := LiquidationSideFromBinanceOrder("SELL")
	if err != nil || s != LiquidationSideLong {
		t.Fatalf("binance sell → long, got %s %v", s, err)
	}
	s, err = LiquidationSideFromBinanceOrder("BUY")
	if err != nil || s != LiquidationSideShort {
		t.Fatalf("binance buy → short, got %s %v", s, err)
	}
	s, err = LiquidationSideFromBybit("Buy")
	if err != nil || s != LiquidationSideLong {
		t.Fatalf("bybit buy → long, got %s %v", s, err)
	}
	s, err = LiquidationSideFromBybit("Sell")
	if err != nil || s != LiquidationSideShort {
		t.Fatalf("bybit sell → short, got %s %v", s, err)
	}
}

func TestLiquidationBook_WindowsAndBiggest(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return now }
	b.started = now.Add(-2 * time.Hour)
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 64000, Quantity: 2, Notional: 128000, Time: now.Add(-2 * time.Minute),
	})
	b.Record(LiquidationEvent{
		Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: LiquidationSideShort,
		Price: 64100, Quantity: 1, Notional: 64100, Time: now.Add(-30 * time.Minute),
	})
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: LiquidationSideLong,
		Price: 3000, Quantity: 10, Notional: 30000, Time: now.Add(-time.Minute),
	})
	snap := b.Snapshot("all", "btcusdt")
	if snap.Symbol != "BTCUSDT" || len(snap.Windows) != 4 {
		t.Fatalf("%+v", snap)
	}
	var w5, w1h LiquidationWindowTotals
	for _, w := range snap.Windows {
		switch w.Window {
		case LiquidationWindow5m:
			w5 = w
		case LiquidationWindow1h:
			w1h = w
		}
	}
	if w5.Count != 1 || w5.Biggest == nil || w5.Biggest.Side != LiquidationSideLong {
		t.Fatalf("5m %+v", w5)
	}
	if w1h.Count != 2 || w1h.Biggest == nil || w1h.Biggest.Notional != "128000" {
		t.Fatalf("1h %+v", w1h)
	}
	onlyBn := b.Snapshot("binance", "BTCUSDT")
	if onlyBn.Windows[0].Count != 1 {
		t.Fatalf("binance-only %+v", onlyBn.Windows[0])
	}
}

func TestLiquidationBook_PrunesOld(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return now }
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 1, Quantity: 1, Notional: 1, Time: now.Add(-25 * time.Hour),
	})
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 1, Quantity: 1, Notional: 2, Time: now.Add(-time.Minute),
	})
	snap := b.Snapshot("all", "BTCUSDT")
	var w24 LiquidationWindowTotals
	for _, w := range snap.Windows {
		if w.Window == LiquidationWindow24h {
			w24 = w
		}
	}
	if w24.Count != 1 {
		t.Fatalf("old event leaked: %+v", w24)
	}
}
