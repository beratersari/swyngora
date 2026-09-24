package domain

import (
	"testing"
	"time"
)

func TestBookFlashWatch_KeepsWallThatVanishesBeforeSnapshot(t *testing.T) {
	t0 := time.Date(2026, 9, 22, 12, 0, 10, 0, time.UTC)
	w := &BookFlashWatch{}
	wall := BookHistoryWall{Side: "bid", Price: 100, Quantity: 80, Notional: 8000}
	if got := w.Observe(ExchangeBinance, "BTCUSDT", t0, []BookHistoryWall{wall}, false); len(got) != 0 {
		t.Fatalf("still resting %+v", got)
	}
	gone := w.Observe(ExchangeBinance, "BTCUSDT", t0.Add(10*time.Second), nil, false)
	if len(gone) != 1 {
		t.Fatalf("expected one flash, got %+v", gone)
	}
	if gone[0].Notional != 8000 || gone[0].Duration != 10*time.Second || !gone[0].Brief {
		t.Fatalf("flash %+v", gone[0])
	}
	if again := w.Observe(ExchangeBinance, "BTCUSDT", t0.Add(15*time.Second), nil, false); len(again) != 0 {
		t.Fatalf("emitted twice %+v", again)
	}
}

func TestBookFlashWatch_SnapshotWallIsNotAMiss(t *testing.T) {
	t0 := time.Date(2026, 9, 22, 12, 1, 0, 0, time.UTC)
	w := &BookFlashWatch{}
	wall := BookHistoryWall{Side: "ask", Price: 101, Notional: 9000}
	_ = w.Observe(ExchangeBinance, "BTCUSDT", t0, []BookHistoryWall{wall}, true)
	gone := w.Observe(ExchangeBinance, "BTCUSDT", t0.Add(20*time.Second), nil, false)
	if len(gone) != 0 {
		t.Fatalf("minute snapshot already kept this wall %+v", gone)
	}
}

func TestAttachBookFlashes_AddsBriefWall(t *testing.T) {
	at := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	snap := BookHistorySnapshot{SampledAt: at, Walls: []BookHistoryWall{{Side: "bid", Price: 90, Notional: 1}}}
	AttachBookFlashes(&snap, []BookHistoryFlash{{
		Side: "bid", Price: 100, Notional: 8000,
		SeenFrom: at.Add(5 * time.Second), SeenUntil: at.Add(15 * time.Second),
		Duration: 10 * time.Second, Brief: true,
	}})
	if len(snap.Flashes) != 1 || len(snap.Walls) != 2 || !snap.Walls[1].Brief {
		t.Fatalf("snap flashes=%d walls=%+v", len(snap.Flashes), snap.Walls)
	}
}
