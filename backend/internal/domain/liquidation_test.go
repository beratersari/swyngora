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

func TestLiquidationBaseAsset(t *testing.T) {
	if got := LiquidationBaseAsset("BTCUSDT"); got != "BTC" {
		t.Fatalf("got %s", got)
	}
	if got := LiquidationBaseAsset("eth-usd"); got != "ETH" {
		t.Fatalf("got %s", got)
	}
}

func TestParseLiquidationOverviewWindow(t *testing.T) {
	got, err := ParseLiquidationOverviewWindow("")
	if err != nil || got != LiquidationWindow24h {
		t.Fatalf("empty → 24h, got %s %v", got, err)
	}
	got, err = ParseLiquidationOverviewWindow("12h")
	if err != nil || got != LiquidationWindow12h {
		t.Fatalf("12h, got %s %v", got, err)
	}
	if _, err := ParseLiquidationOverviewWindow("5m"); err == nil {
		t.Fatal("5m should be rejected")
	}
}

func TestLiquidationBook_Overview(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 15, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return now }
	b.SetLive(ExchangeBinance, true)
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 64000, Quantity: 2, Notional: 128000, Time: now.Add(-30 * time.Minute),
	})
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: LiquidationSideShort,
		Price: 3000, Quantity: 10, Notional: 30000, Time: now.Add(-2 * time.Hour),
	})
	b.Record(LiquidationEvent{
		Exchange: ExchangeBybit, Symbol: "SOLUSDT", Side: LiquidationSideLong,
		Price: 150, Quantity: 100, Notional: 15000, Time: now.Add(-10 * time.Minute),
	})

	ov := b.Overview("all", LiquidationWindow1h, 10)
	if ov.CoinWindow != LiquidationWindow1h || len(ov.Windows) != 4 {
		t.Fatalf("%+v", ov)
	}
	var w1h, w12h LiquidationWindowTotals
	for _, w := range ov.Windows {
		switch w.Window {
		case LiquidationWindow1h:
			w1h = w
		case LiquidationWindow12h:
			w12h = w
		}
	}
	if w1h.Count != 2 || w1h.TotalNotional != "143000" {
		t.Fatalf("1h %+v", w1h)
	}
	if w12h.Count != 3 {
		t.Fatalf("12h %+v", w12h)
	}
	if len(ov.Coins) != 2 || ov.Coins[0].Symbol != "BTCUSDT" || ov.Coins[0].Base != "BTC" {
		t.Fatalf("coins %+v", ov.Coins)
	}

	onlyBn := b.Overview("binance", LiquidationWindow24h, 10)
	if len(onlyBn.Coins) != 2 {
		t.Fatalf("binance coins %+v", onlyBn.Coins)
	}
	limited := b.Overview("all", LiquidationWindow24h, 1)
	if len(limited.Coins) != 1 || limited.Coins[0].Symbol != "BTCUSDT" {
		t.Fatalf("limit %+v", limited.Coins)
	}
}

func TestLiquidationBook_RecentLarge(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 16, 15, 0, 0, 0, time.UTC)
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 64000, Quantity: 2, Notional: 128000, Time: now.Add(-2 * time.Minute),
	})
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: LiquidationSideShort,
		Price: 3000, Quantity: 1, Notional: 3000, Time: now.Add(-time.Minute),
	})
	b.Record(LiquidationEvent{
		Exchange: ExchangeBybit, Symbol: "SOLUSDT", Side: LiquidationSideLong,
		Price: 150, Quantity: 1000, Notional: 150000, Time: now.Add(-2 * time.Hour),
	})
	got := b.RecentLarge(now.Add(-10*time.Minute), 100_000)
	if len(got) != 1 || got[0].Symbol != "BTCUSDT" {
		t.Fatalf("%+v", got)
	}
}

func TestLiquidationBook_CoverageIsPerCoinAndVenue(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return now }
	// Process / Binance stream has been live more than 24h.
	b.SetLive(ExchangeBinance, true)
	b.venueSince[ExchangeBinance] = now.Add(-25 * time.Hour)
	b.venueClock[ExchangeBinance].sessionStart = now.Add(-25 * time.Hour)

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

func TestLiquidationBook_CoverageFreezesWhenSocketDown(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	cur := now
	b.now = func() time.Time { return cur }

	b.MarkWatch(ExchangeBybit, "BTCUSDT")
	// Never connected: time passing does not grow coverage.
	cur = now.Add(2 * time.Hour)
	down := b.Snapshot("bybit", "BTCUSDT")
	for _, w := range down.Windows {
		if w.CoverageSeconds != 0 || w.Complete {
			t.Fatalf("offline watch must stay at 0 coverage: %+v", w)
		}
	}

	cur = now
	b.SetLive(ExchangeBybit, true)
	cur = now.Add(6 * time.Minute)
	live := b.Snapshot("bybit", "BTCUSDT")
	var w5, w1h LiquidationWindowTotals
	for _, w := range live.Windows {
		switch w.Window {
		case LiquidationWindow5m:
			w5 = w
		case LiquidationWindow1h:
			w1h = w
		}
	}
	if !w5.Complete || w1h.Complete || w1h.CoverageSeconds < 350 || w1h.CoverageSeconds > 370 {
		t.Fatalf("after 6m live: 5m=%+v 1h=%+v", w5, w1h)
	}

	b.SetLive(ExchangeBybit, false)
	frozen := w1h.CoverageSeconds
	cur = now.Add(3 * time.Hour)
	after := b.Snapshot("bybit", "BTCUSDT")
	for _, w := range after.Windows {
		if w.Window == LiquidationWindow1h {
			if w.Complete || w.CoverageSeconds != frozen {
				t.Fatalf("coverage grew while socket down: before=%d after=%+v", frozen, w)
			}
		}
		if w.Window == LiquidationWindow4h && w.Complete {
			t.Fatalf("4h must not complete while disconnected: %+v", w)
		}
	}

	// Reconnect resumes from the frozen total; it does not count the gap.
	b.SetLive(ExchangeBybit, true)
	cur = now.Add(3*time.Hour + 4*time.Minute)
	resumed := b.Snapshot("bybit", "BTCUSDT")
	for _, w := range resumed.Windows {
		if w.Window == LiquidationWindow1h {
			if w.Complete || w.CoverageSeconds < 580 || w.CoverageSeconds > 620 {
				t.Fatalf("reconnect should add ~4m to ~6m live: %+v", w)
			}
		}
	}
}

func TestLiquidationBook_BinanceCoverageIsVenueLiveNotFirstEvent(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	cur := now
	b.now = func() time.Time { return cur }

	cur = now.Add(2 * time.Hour)
	offline := b.Snapshot("binance", "BTCUSDT")
	for _, w := range offline.Windows {
		if w.CoverageSeconds != 0 || w.Complete {
			t.Fatalf("binance never connected must stay at 0: %+v", w)
		}
	}

	cur = now
	b.SetLive(ExchangeBinance, true)
	cur = now.Add(2 * time.Hour)
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 1, Quantity: 1, Notional: 1, Time: cur,
	})
	live := b.Snapshot("binance", "BTCUSDT")
	var w1h, w4h LiquidationWindowTotals
	for _, w := range live.Windows {
		switch w.Window {
		case LiquidationWindow1h:
			w1h = w
		case LiquidationWindow4h:
			w4h = w
		}
	}
	if !w1h.Complete || w4h.Complete || w4h.CoverageSeconds < 7100 || w4h.CoverageSeconds > 7300 {
		t.Fatalf("first print must not reset 2h venue live: 1h=%+v 4h=%+v", w1h, w4h)
	}

	b.SetLive(ExchangeBinance, false)
	frozen := w4h.CoverageSeconds
	cur = now.Add(10 * time.Hour)
	dropped := b.Snapshot("binance", "BTCUSDT")
	for _, w := range dropped.Windows {
		if w.Window == LiquidationWindow4h && (w.Complete || w.CoverageSeconds != frozen) {
			t.Fatalf("binance coverage grew while socket down: before=%d after=%+v", frozen, w)
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
