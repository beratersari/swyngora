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

func TestLiquidationBook_CoverageIsPerCoinAndVenue(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return now }
	// Process / Binance stream has been up more than 24h.
	b.SetLive(ExchangeBinance, true)
	b.venueSince[ExchangeBinance] = now.Add(-25 * time.Hour)

	// New Bybit coin just subscribed.
	b.MarkWatch(ExchangeBybit, "DOGEUSDT")

	byb := b.Snapshot("bybit", "DOGEUSDT")
	var w24, w5 LiquidationWindowTotals
	for _, w := range byb.Windows {
		switch w.Window {
		case LiquidationWindow24h:
			w24 = w
		case LiquidationWindow5m:
			w5 = w
		}
	}
	if w24.Complete || w5.Complete {
		t.Fatalf("new bybit coin must not inherit server uptime: 24h=%+v 5m=%+v since=%v", w24, w5, byb.CollectingSince)
	}
	if byb.CollectingSince.Before(now.Add(-time.Minute)) {
		t.Fatalf("collectingSince should be first watch, got %v", byb.CollectingSince)
	}

	// Combined uses the later start (Bybit), so 24h is still incomplete.
	all := b.Snapshot("all", "DOGEUSDT")
	for _, w := range all.Windows {
		if w.Window == LiquidationWindow24h && w.Complete {
			t.Fatalf("all/24h complete while bybit just started: %+v", w)
		}
	}

	// Binance-only for that coin uses venue stream start → 24h complete.
	bn := b.Snapshot("binance", "DOGEUSDT")
	for _, w := range bn.Windows {
		if w.Window == LiquidationWindow24h && !w.Complete {
			t.Fatalf("binance all-market stream should complete 24h: %+v", w)
		}
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
