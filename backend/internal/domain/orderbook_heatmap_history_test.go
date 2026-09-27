package domain

import (
	"strconv"
	"testing"
	"time"
)

func TestBuildHeatmapFromBookHistory_UsesLadderMovesAndFlashes(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 10, 0, 0, time.UTC)
	minute := now.Add(-2 * time.Minute)
	snaps := []BookHistorySnapshot{{
		SampledAt: minute,
		Mid:       100,
		GroupSize: 1,
		Bids:      []BookHistoryLevel{{Price: 99, Notional: 1000}},
		Asks:      []BookHistoryLevel{{Price: 101, Notional: 500}},
	}}
	flashes := []BookHistoryFlash{{
		Side: "bid", Price: 98, Notional: 8000, SeenFrom: minute.Add(10 * time.Second),
	}}
	moves := []BookHistoryMove{{
		Side: "ask", Price: 101, PeakNotional: 4000, SeenFrom: minute.Add(20 * time.Second),
	}}
	got := BuildHeatmapFromBookHistory(ExchangeBinance, "BTCUSDT", 4*time.Hour, now, snaps, flashes, moves)
	if got.Source != "history" || got.WindowSeconds != 4*3600 || len(got.Columns) != 1 {
		t.Fatalf("%+v cols=%d", got.Source, len(got.Columns))
	}
	col := got.Columns[0]
	if col.Mid != "100" {
		t.Fatalf("mid %s", col.Mid)
	}
	var bidWall, ask float64
	var wall bool
	for _, lv := range col.Bids {
		if lv.Price == "98" {
			bidWall = parseHeat(t, lv.Notional)
			wall = lv.IsWall
		}
		if lv.Price == "99" && parseHeat(t, lv.Notional) != 1000 {
			t.Fatalf("ladder bid %+v", lv)
		}
	}
	for _, lv := range col.Asks {
		if lv.Price == "101" {
			ask = parseHeat(t, lv.Notional)
		}
	}
	if bidWall != 8000 || !wall {
		t.Fatalf("flash bid %+v wall=%v", col.Bids, wall)
	}
	if ask != 4000 {
		t.Fatalf("move ask %+v", col.Asks)
	}
}

func TestBuildHeatmapFromBookHistory_BucketsLongWindows(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	snaps := make([]BookHistorySnapshot, 0, 400)
	for i := 0; i < 400; i++ {
		snaps = append(snaps, BookHistorySnapshot{
			SampledAt: now.Add(-time.Duration(i) * time.Minute),
			Mid:       100,
			Bids:      []BookHistoryLevel{{Price: 99, Notional: 10}},
		})
	}
	got := BuildHeatmapFromBookHistory(ExchangeBinance, "BTCUSDT", 7*24*time.Hour, now, snaps, nil, nil)
	if len(got.Columns) == 0 || len(got.Columns) > HeatmapHistoryMaxColumns {
		t.Fatalf("cols %d", len(got.Columns))
	}
	if got.SampleEveryMs < int(time.Minute/time.Millisecond) {
		t.Fatalf("bucket %d", got.SampleEveryMs)
	}
}

func parseHeat(t *testing.T, raw string) float64 {
	t.Helper()
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
