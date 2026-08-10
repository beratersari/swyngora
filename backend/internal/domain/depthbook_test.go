package domain

import (
	"errors"
	"testing"
	"time"
)

func TestLocalDepthBook_SnapshotThenDiff(t *testing.T) {
	b := NewLocalDepthBook("BTCUSDT")
	b.LoadSnapshot(10, []DepthLevel{{Price: "100.0", Quantity: 2}, {Price: "99.0", Quantity: 1}},
		[]DepthLevel{{Price: "101.0", Quantity: 3}})
	if b.Synced() {
		t.Fatal("snapshot alone is not synced")
	}
	if _, err := b.Snapshot(20); !errors.Is(err, ErrConflict) {
		t.Fatalf("unsynced snapshot: %v", err)
	}
	// First event attaches: U=9 u=12 covers lastID+1=11
	if err := b.ApplyDiff(DepthDiff{FirstID: 9, FinalID: 12,
		Bids: []DepthLevel{{Price: "100.0", Quantity: 5}, {Price: "99.0", Quantity: 0}},
		Asks: []DepthLevel{{Price: "101.0", Quantity: 1}, {Price: "102.0", Quantity: 4}},
	}); err != nil {
		t.Fatal(err)
	}
	if !b.Synced() || b.LastUpdateID() != 12 {
		t.Fatalf("synced=%v last=%d", b.Synced(), b.LastUpdateID())
	}
	snap, err := b.Snapshot(20)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Live || snap.Source != OrderBookSourceWebSocket {
		t.Fatalf("source %+v", snap)
	}
	if len(snap.Bids) != 1 || snap.Bids[0].Price != 100 || snap.Bids[0].Quantity != 5 {
		t.Fatalf("bids %+v", snap.Bids)
	}
	if len(snap.Asks) != 2 || snap.Asks[0].Price != 101 {
		t.Fatalf("asks %+v", snap.Asks)
	}
}

func TestLocalDepthBook_StaleEventIgnored(t *testing.T) {
	b := NewLocalDepthBook("ETHUSDT")
	b.LoadSnapshot(50, []DepthLevel{{Price: "1", Quantity: 1}}, nil)
	if err := b.ApplyDiff(DepthDiff{FirstID: 40, FinalID: 50}); err != nil {
		t.Fatal(err)
	}
	if b.Synced() {
		t.Fatal("stale first event must not sync")
	}
	if err := b.ApplyDiff(DepthDiff{FirstID: 51, FinalID: 51, Bids: []DepthLevel{{Price: "1", Quantity: 2}}}); err != nil {
		t.Fatal(err)
	}
	if err := b.ApplyDiff(DepthDiff{FirstID: 40, FinalID: 50}); err != nil {
		t.Fatal(err)
	}
	if b.LastUpdateID() != 51 {
		t.Fatalf("last=%d", b.LastUpdateID())
	}
}

func TestLocalDepthBook_GapInvalidates(t *testing.T) {
	b := NewLocalDepthBook("SOLUSDT")
	b.LoadSnapshot(1, []DepthLevel{{Price: "10", Quantity: 1}}, nil)
	if err := b.ApplyDiff(DepthDiff{FirstID: 2, FinalID: 2}); err != nil {
		t.Fatal(err)
	}
	err := b.ApplyDiff(DepthDiff{FirstID: 4, FinalID: 5}) // skipped 3
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	if b.Synced() {
		t.Fatal("gap must unsync")
	}
	if _, err := b.Snapshot(10); !errors.Is(err, ErrConflict) {
		t.Fatalf("must not serve after gap: %v", err)
	}
}

func TestLocalDepthBook_ReplaceAndSequential(t *testing.T) {
	b := NewLocalDepthBook("BTCUSDT")
	b.ReplaceSnapshot(10, []DepthLevel{{Price: "100", Quantity: 1}}, []DepthLevel{{Price: "101", Quantity: 1}})
	if !b.Synced() {
		t.Fatal("replace snapshot should sync")
	}
	if err := b.ApplySequential(11, []DepthLevel{{Price: "100", Quantity: 4}}, nil, time.Time{}); err != nil {
		t.Fatal(err)
	}
	snap, err := b.Snapshot(10)
	if err != nil || snap.Bids[0].Quantity != 4 {
		t.Fatalf("%+v %v", snap, err)
	}
	if err := b.ApplySequential(13, nil, nil, time.Time{}); !errors.Is(err, ErrConflict) {
		t.Fatalf("gap: %v", err)
	}
	if b.Synced() {
		t.Fatal("gap must unsync")
	}
}

func TestLocalDepthBook_SequentialRestart(t *testing.T) {
	b := NewLocalDepthBook("ETHUSDT")
	b.ReplaceSnapshot(10, []DepthLevel{{Price: "1", Quantity: 1}}, []DepthLevel{{Price: "2", Quantity: 1}})
	if err := b.ApplySequential(1, nil, nil, time.Time{}); !errors.Is(err, ErrConflict) {
		t.Fatalf("u=1: %v", err)
	}
	if b.Synced() {
		t.Fatal("u=1 must unsync so callers resync")
	}
}

func TestLocalDepthBook_Unsequenced(t *testing.T) {
	b := NewLocalDepthBook("BTC-USD")
	if err := b.ApplyUnsequenced([]DepthLevel{{Price: "1", Quantity: 1}}, nil, time.Time{}); !errors.Is(err, ErrConflict) {
		t.Fatalf("want unsynced, got %v", err)
	}
	b.ReplaceSnapshot(0, []DepthLevel{{Price: "1", Quantity: 1}}, []DepthLevel{{Price: "2", Quantity: 1}})
	if err := b.ApplyUnsequenced([]DepthLevel{{Price: "1", Quantity: 0}, {Price: "0.9", Quantity: 3}}, nil, time.Time{}); err != nil {
		t.Fatal(err)
	}
	snap, err := b.Snapshot(10)
	if err != nil || len(snap.Bids) != 1 || snap.Bids[0].Price != 0.9 {
		t.Fatalf("%+v %v", snap, err)
	}
}

func TestLocalDepthBook_FirstEventMustAttach(t *testing.T) {
	b := NewLocalDepthBook("X")
	b.LoadSnapshot(100, []DepthLevel{{Price: "1", Quantity: 1}}, nil)
	err := b.ApplyDiff(DepthDiff{FirstID: 110, FinalID: 112})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want attach fail, got %v", err)
	}
	if b.Synced() || b.LastUpdateID() != 0 {
		t.Fatalf("cleared book last=%d synced=%v", b.LastUpdateID(), b.Synced())
	}
}
