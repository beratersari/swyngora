package domain

import "testing"
import "time"

func TestBookMoveWatch_KeepsSizeThatLeavesBeforeSnapshot(t *testing.T) {
	w := &BookMoveWatch{}
	t0 := time.Date(2026, 9, 22, 12, 0, 5, 0, time.UTC)
	level := BookMoveSample{Side: "bid", Price: 100, Notional: 500}
	if got := w.Observe(ExchangeBinance, "BTCUSDT", t0, []BookMoveSample{level, {Side: "bid", Price: 99, Notional: 9500}}, false); len(got) != 0 {
		t.Fatalf("still resting: %+v", got)
	}
	gone := w.Observe(ExchangeBinance, "BTCUSDT", t0.Add(10*time.Second), []BookMoveSample{{Side: "bid", Price: 99, Notional: 9500}}, false)
	if len(gone) != 1 || gone[0].Change != BookHistoryChangeRemoved || gone[0].PeakNotional != 500 || gone[0].ToNotional != 0 {
		t.Fatalf("move %+v", gone)
	}
}

func TestBookMoveWatch_KeepsSwingThatLandedSmaller(t *testing.T) {
	w := &BookMoveWatch{}
	t0 := time.Date(2026, 9, 22, 12, 0, 5, 0, time.UTC)
	rest := BookMoveSample{Side: "ask", Price: 101, Notional: 200}
	other := BookMoveSample{Side: "ask", Price: 102, Notional: 800}
	_ = w.Observe(ExchangeBinance, "BTCUSDT", t0, []BookMoveSample{rest, other}, false)
	big := rest
	big.Notional = 900
	_ = w.Observe(ExchangeBinance, "BTCUSDT", t0.Add(5*time.Second), []BookMoveSample{big, other}, false)
	got := w.Observe(ExchangeBinance, "BTCUSDT", t0.Add(20*time.Second), []BookMoveSample{rest, other}, true)
	if len(got) != 1 || got[0].Change != BookHistoryChangeGained || got[0].PeakNotional != 900 || got[0].ToNotional != 200 {
		t.Fatalf("swing %+v", got)
	}
}

func TestBookMoveWatch_IgnoresWallsAndDust(t *testing.T) {
	w := &BookMoveWatch{}
	t0 := time.Date(2026, 9, 22, 12, 0, 5, 0, time.UTC)
	levels := []BookMoveSample{
		{Side: "bid", Price: 100, Notional: 8000, Wall: true},
		{Side: "bid", Price: 99, Notional: 50},
		{Side: "bid", Price: 98, Notional: 1950},
	}
	_ = w.Observe(ExchangeBinance, "BTCUSDT", t0, levels, false)
	got := w.Observe(ExchangeBinance, "BTCUSDT", t0.Add(10*time.Second), []BookMoveSample{{Side: "bid", Price: 98, Notional: 1950}}, false)
	if len(got) != 0 {
		t.Fatalf("expected no move, got %+v", got)
	}
}

func TestAttachBookMoves_AddsMoveOnThatMinute(t *testing.T) {
	at := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	snap := BookHistorySnapshot{SampledAt: at}
	AttachBookMoves(&snap, []BookHistoryMove{{
		Side: "bid", Price: 100, Change: BookHistoryChangeRemoved, PeakNotional: 500,
		SeenFrom: at.Add(5 * time.Second), SeenUntil: at.Add(15 * time.Second),
	}})
	if len(snap.Moves) != 1 {
		t.Fatalf("moves %+v", snap.Moves)
	}
}
