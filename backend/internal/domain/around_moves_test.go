package domain

import (
	"testing"
	"time"
)

func TestFindImportantMoves_RanksUpAndDownLegs(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	bars := []AroundBar{
		{Time: t0, Open: 100, High: 100.2, Low: 99.9, Close: 100.1, Volume: 1000},
		{Time: t0.Add(15 * time.Minute), Open: 100.1, High: 104, Low: 100, Close: 103.5, Volume: 4000},
		{Time: t0.Add(30 * time.Minute), Open: 103.5, High: 105, Low: 103, Close: 104.8, Volume: 3000},
		{Time: t0.Add(45 * time.Minute), Open: 104.8, High: 105, Low: 104.5, Close: 104.7, Volume: 800},
		{Time: t0.Add(60 * time.Minute), Open: 104.7, High: 104.8, Low: 101, Close: 101.2, Volume: 3500},
		{Time: t0.Add(75 * time.Minute), Open: 101.2, High: 101.4, Low: 100.8, Close: 101.0, Volume: 900},
	}
	got := FindImportantMoves(bars, 1.5, "both", 8)
	if len(got) < 2 {
		t.Fatalf("want 2 legs, got %+v", got)
	}
	if got[0].Direction != CVDDirUp || got[0].ReturnPct < 4 {
		t.Fatalf("largest should be the rise %+v", got[0])
	}
	if got[1].Direction != CVDDirDown || got[1].ReturnPct > -2 {
		t.Fatalf("second should be the drop %+v", got[1])
	}
	if got[0].Grade == "" || got[0].During == "" {
		t.Fatalf("annotate %+v", got[0])
	}
}

func TestFindImportantMoves_DirectionFilter(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	bars := []AroundBar{
		{Time: t0, Open: 100, High: 106, Low: 100, Close: 105.5, Volume: 2000},
		{Time: t0.Add(15 * time.Minute), Open: 105.5, High: 105.6, Low: 100, Close: 100.5, Volume: 2000},
	}
	up := FindImportantMoves(bars, 1.5, "up", 8)
	if len(up) != 1 || up[0].Direction != CVDDirUp {
		t.Fatalf("up %+v", up)
	}
	down := FindImportantMoves(bars, 1.5, "down", 8)
	if len(down) != 1 || down[0].Direction != CVDDirDown {
		t.Fatalf("down %+v", down)
	}
}

func TestFindImportantMoves_MinPctDropsSmallLegs(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	bars := []AroundBar{
		{Time: t0, Open: 100, High: 100.8, Low: 100, Close: 100.6, Volume: 500},
		{Time: t0.Add(15 * time.Minute), Open: 100.6, High: 108, Low: 100.5, Close: 107.5, Volume: 4000},
	}
	got := FindImportantMoves(bars, 3, "both", 8)
	if len(got) != 1 || got[0].ReturnPct < 6 {
		t.Fatalf("%+v", got)
	}
}

func TestAroundDuringFor(t *testing.T) {
	if AroundDuringFor(15*time.Minute) != AroundDuring15m {
		t.Fatal("15m")
	}
	if AroundDuringFor(time.Hour) != AroundDuring1h {
		t.Fatal("1h")
	}
}

func TestParseAroundLookback(t *testing.T) {
	id, d, err := ParseAroundLookback("")
	if err != nil || id != "24h" || d != 24*time.Hour {
		t.Fatalf("%s %s %v", id, d, err)
	}
	if _, _, err := ParseAroundLookback("2d"); err == nil {
		t.Fatal("expected error")
	}
}
