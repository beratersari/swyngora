package domain

import (
	"testing"
	"time"
)

func TestBuildOptionsStrikeOI_GainsLossesAndCloses(t *testing.T) {
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	snap := &OptionsSnapshot{
		Asset: "BTC",
		AsOf:  now,
		Expiries: []OptionsExpiryBucket{{
			Expiry: exp,
			Label:  "27 Mar 2026",
			Strikes: []OptionsExpiryStrike{
				{Strike: 90000, CallOI: 5, PutOI: 12},
				{Strike: 100000, CallOI: 30, PutOI: 8},
			},
		}},
	}
	pastAt := now.Add(-4 * time.Hour)
	books := map[string]OptionsStrikeBook{
		OptionsHistoryWindow4h: {
			SampledAt: pastAt,
			Points: []OptionsStrikeOIPoint{
				{Expiry: exp, Strike: 90000, CallOI: 2, PutOI: 20},
				{Expiry: exp, Strike: 100000, CallOI: 10, PutOI: 8},
				{Expiry: exp, Strike: 110000, CallOI: 4, PutOI: 1},
			},
		},
	}
	got := BuildOptionsStrikeOI(snap, books, now)
	if got == nil || len(got.Windows) != len(OptionsHistoryWindows) {
		t.Fatalf("windows %+v", got)
	}
	var win *OptionsStrikeOIWindow
	for i := range got.Windows {
		if got.Windows[i].Window == OptionsHistoryWindow4h {
			win = &got.Windows[i]
		}
		if got.Windows[i].Window == OptionsHistoryWindow1h && got.Windows[i].Complete {
			t.Fatal("1h should stay incomplete without a book")
		}
	}
	if win == nil || !win.Complete || len(win.Strikes) != 3 {
		t.Fatalf("4h %+v", win)
	}
	byStrike := map[float64]OptionsStrikeOIRow{}
	for _, row := range win.Strikes {
		byStrike[row.Strike] = row
	}
	if byStrike[100000].CallChange != 20 || byStrike[100000].PutChange != 0 || byStrike[100000].NetChange != 20 {
		t.Fatalf("100k %+v", byStrike[100000])
	}
	if byStrike[90000].CallChange != 3 || byStrike[90000].PutChange != -8 {
		t.Fatalf("90k %+v", byStrike[90000])
	}
	if !byStrike[110000].Closed || byStrike[110000].NetChange != -5 || byStrike[110000].CallOI != 0 {
		t.Fatalf("closed %+v", byStrike[110000])
	}
	if byStrike[100000].Label != "27 Mar 2026" {
		t.Fatalf("label %q", byStrike[100000].Label)
	}
}

func TestBuildOptionsStrikeOI_RejectsStaleBook(t *testing.T) {
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	exp := now.Add(48 * time.Hour)
	snap := &OptionsSnapshot{
		Expiries: []OptionsExpiryBucket{{
			Expiry:  exp,
			Strikes: []OptionsExpiryStrike{{Strike: 100, CallOI: 1, PutOI: 1}},
		}},
	}
	books := map[string]OptionsStrikeBook{
		OptionsHistoryWindow1h: {
			SampledAt: now.Add(-3 * time.Hour),
			Points:    []OptionsStrikeOIPoint{{Expiry: exp, Strike: 100, CallOI: 1, PutOI: 1}},
		},
	}
	got := BuildOptionsStrikeOI(snap, books, now)
	for _, w := range got.Windows {
		if w.Window == OptionsHistoryWindow1h && w.Complete {
			t.Fatal("stale 1h book should not count")
		}
	}
}

func TestOptionsStrikePointsFromSnapshotSkipsEmpty(t *testing.T) {
	exp := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	pts := OptionsStrikePointsFromSnapshot(&OptionsSnapshot{
		Expiries: []OptionsExpiryBucket{{
			Expiry: exp,
			Strikes: []OptionsExpiryStrike{
				{Strike: 1, CallOI: 0, PutOI: 0},
				{Strike: 2, CallOI: 1, PutOI: 0},
			},
		}},
	})
	if len(pts) != 1 || pts[0].Strike != 2 || pts[0].CallOI != 1 {
		t.Fatalf("%+v", pts)
	}
}
