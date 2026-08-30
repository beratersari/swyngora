package domain

import (
	"testing"
	"time"
)

func TestLiquidationBook_CombinedDoesNotBorrowOtherVenue(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return now }
	b.SetLive(ExchangeBinance, true)
	b.venueSince[ExchangeBinance] = now.Add(-25 * time.Hour)
	b.venueClock[ExchangeBinance].sessionStart = now.Add(-25 * time.Hour)
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 1, Quantity: 1, Notional: 100, Time: now.Add(-time.Minute),
	})

	all := b.Snapshot("all", "BTCUSDT")
	if all.Live {
		t.Fatal("combined must not be live when Bybit never started")
	}
	if len(all.Feed.Missing) == 0 {
		t.Fatal("expected bybit in missing")
	}
	var w24 LiquidationWindowTotals
	for _, w := range all.Windows {
		if w.Window == LiquidationWindow24h {
			w24 = w
		}
	}
	if w24.Complete {
		t.Fatalf("combined 24h must not be complete on Binance-only coverage: %+v", w24)
	}
	if w24.Count != 1 {
		t.Fatalf("still show the prints we have: %+v", w24)
	}

	ov := b.Overview("all", LiquidationWindow24h, 10)
	if ov.Live || ov.CollectingSince.IsZero() == false && ov.Feed.Missing == nil {
		// collectingSince must be zero when Bybit never started
	}
	if !ov.CollectingSince.IsZero() {
		t.Fatalf("overview collectingSince borrowed Binance start: %v", ov.CollectingSince)
	}
	if ov.Live {
		t.Fatal("overview live borrowed Binance")
	}
	found := false
	for _, m := range ov.Feed.Missing {
		if m == "bybit" {
			found = true
		}
	}
	if !found {
		t.Fatalf("overview missing %+v", ov.Feed.Missing)
	}
}

func TestLiquidationBook_GapsAndLastEvent(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	cur := now
	b.now = func() time.Time { return cur }

	b.SetLive(ExchangeBybit, true)
	b.NoteSeen(ExchangeBybit)
	b.Record(LiquidationEvent{
		Exchange: ExchangeBybit, Symbol: "ETHUSDT", Side: LiquidationSideShort,
		Price: 1, Quantity: 1, Notional: 10, Time: now,
	})
	cur = now.Add(10 * time.Minute)
	b.SetLive(ExchangeBybit, false)
	cur = now.Add(40 * time.Minute)
	b.SetLive(ExchangeBybit, true)
	b.NoteSeen(ExchangeBybit)

	feed := b.Feed("bybit")
	if len(feed.Venues) != 1 {
		t.Fatalf("%+v", feed)
	}
	v := feed.Venues[0]
	if v.LastEventAt.IsZero() || !v.LastEventAt.Equal(now) {
		t.Fatalf("last event %+v", v.LastEventAt)
	}
	if !v.Live {
		t.Fatal("expected live after reconnect")
	}
	if len(v.Gaps) != 1 {
		t.Fatalf("gaps %+v", v.Gaps)
	}
	if v.Gaps[0].Seconds < 29*60 || v.Gaps[0].Seconds > 31*60 {
		t.Fatalf("gap seconds %d", v.Gaps[0].Seconds)
	}

	// Restore after a 3h downtime must record a gap, not pretend the book was live.
	fresh := NewLiquidationBook()
	fresh.now = func() time.Time { return now.Add(3 * time.Hour) }
	cov := b.CoverageSnapshot(now.Add(40 * time.Minute))
	for _, c := range cov {
		if c.Exchange == ExchangeBybit && c.Symbol == "" {
			c.LastSaved = now.Add(40 * time.Minute)
			fresh.RestoreTracking(c.Exchange, c.Symbol, c.FirstWatch, c.Live)
			fresh.RestoreFeed(c, now.Add(3*time.Hour))
		}
	}
	restored := fresh.Feed("bybit")
	if len(restored.Venues) == 0 || len(restored.Venues[0].Gaps) == 0 {
		t.Fatalf("expected downtime gap %+v", restored)
	}
}

func TestLastHuntPrice_NoCrossVenue(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	venues := []HuntHeatmapVenueSeries{
		{Exchange: ExchangeBinance, Prices: []HuntHeatmapPricePoint{{Time: now, Price: 100}}},
		{Exchange: ExchangeBybit, Prices: []HuntHeatmapPricePoint{{Time: now.Add(time.Minute), Price: 50}}},
	}
	if LastHuntPrice(venues, "all") != 0 {
		t.Fatal("combined must not pick a venue last")
	}
	if LastHuntPrice(venues, "binance") != 100 {
		t.Fatal("binance last")
	}
	if LastHuntPrice(venues, "bybit") != 50 {
		t.Fatal("bybit last")
	}
}
