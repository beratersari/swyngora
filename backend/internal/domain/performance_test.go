package domain

import (
	"math"
	"testing"
	"time"
)

func TestParsePerformancePeriod(t *testing.T) {
	p, d, err := ParsePerformancePeriod("1d")
	if err != nil || p != PerformancePeriod1D || d != 24*time.Hour {
		t.Fatalf("1d: %s %v %v", p, d, err)
	}
	p, d, err = ParsePerformancePeriod("")
	if err != nil || p != PerformancePeriod1W || d != 7*24*time.Hour {
		t.Fatalf("default: %s %v %v", p, d, err)
	}
	p, d, err = ParsePerformancePeriod("3M")
	if err != nil || p != PerformancePeriod3M || d != 90*24*time.Hour {
		t.Fatalf("3M: %s %v %v", p, d, err)
	}
	if _, _, err := ParsePerformancePeriod("year"); err == nil {
		t.Fatal("want invalid period")
	}
}

func TestSnapshotBucket(t *testing.T) {
	ts := time.Date(2026, 8, 7, 12, 17, 44, 0, time.UTC)
	got := SnapshotBucket(ts, 15*time.Minute)
	want := time.Date(2026, 8, 7, 12, 15, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestPerformanceChangePct(t *testing.T) {
	pct := PerformanceChangePct(10000, 11000)
	if pct == nil || math.Abs(*pct-10) > 1e-9 {
		t.Fatalf("pct=%v", pct)
	}
	if PerformanceChangePct(0, 100) != nil {
		t.Fatal("zero start should be nil pct")
	}
}

func TestAssemblePerformance_CarryForwardAndLive(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	carry := &EquitySnapshot{Equity: 10000, CashBalance: 4000, PositionsValue: 6000, TakenAt: start.Add(-time.Hour)}
	mid := EquitySnapshot{
		BucketAt: start.Add(24 * time.Hour), TakenAt: start.Add(24 * time.Hour),
		Equity: 10500, CashBalance: 4000, PositionsValue: 6500,
	}
	live := EquityPoint{Time: end, Equity: 11200, CashBalance: 4200, PositionsValue: 7000}
	got := AssemblePerformance(
		PerformancePeriod1W, start, end, start.Add(-30*24*time.Hour),
		10000, "USDT", "c1", carry, []EquitySnapshot{mid}, live, "note",
	)
	if got.StartEquity != 10000 || got.EndEquity != 11200 {
		t.Fatalf("start/end %v %v", got.StartEquity, got.EndEquity)
	}
	if math.Abs(got.ChangeAmount-1200) > 1e-9 {
		t.Fatalf("change=%v", got.ChangeAmount)
	}
	if got.ChangePct == nil || math.Abs(*got.ChangePct-12) > 1e-9 {
		t.Fatalf("pct=%v", got.ChangePct)
	}
	if got.Partial {
		t.Fatal("not partial")
	}
	if got.PointCount < 3 {
		t.Fatalf("points=%d %+v", got.PointCount, got.Points)
	}
	if got.Points[0].Equity != 10000 || !got.Points[0].Time.Equal(start) {
		t.Fatalf("synthetic start %+v", got.Points[0])
	}
	last := got.Points[len(got.Points)-1]
	if last.Equity != 11200 {
		t.Fatalf("live last %+v", last)
	}
}

func TestAssemblePerformance_YoungPortfolio(t *testing.T) {
	created := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
	requested := end.Add(-7 * 24 * time.Hour)
	live := EquityPoint{Time: end, Equity: 10100, CashBalance: 10100}
	got := AssemblePerformance(
		PerformancePeriod1W, requested, end, created,
		10000, "USDT", "c2", nil, nil, live, "",
	)
	if !got.Partial {
		t.Fatal("want partial")
	}
	if got.StartEquity != 10000 || got.EndEquity != 10100 {
		t.Fatalf("%+v", got)
	}
	// Synthetic open + live so charts always get a segment.
	if got.PointCount < 2 {
		t.Fatalf("want ≥2 points for young book, got %d %+v", got.PointCount, got.Points)
	}
}

func TestAssemblePerformance_SnapBeforeWindowStillSyntheticsStart(t *testing.T) {
	created := time.Date(2026, 8, 12, 17, 46, 47, 0, time.UTC)
	end := time.Date(2026, 8, 12, 17, 50, 0, 0, time.UTC)
	requested := end.Add(-7 * 24 * time.Hour)
	// Snapshot bucket slightly before create time (15m grid) must not wipe the open point.
	pre := EquitySnapshot{
		BucketAt: created.Add(-2 * time.Minute), TakenAt: created.Add(-2 * time.Minute),
		Equity: 9990, CashBalance: 3000, PositionsValue: 6990,
	}
	live := EquityPoint{Time: end, Equity: 9992, CashBalance: 3019, PositionsValue: 6973}
	got := AssemblePerformance(
		PerformancePeriod1W, requested, end, created,
		10000, "USDT", "c3", nil, []EquitySnapshot{pre}, live, "",
	)
	if got.PointCount < 2 {
		t.Fatalf("points=%d %+v", got.PointCount, got.Points)
	}
	if got.Points[0].Equity != 10000 {
		t.Fatalf("synthetic start equity=%v want 10000", got.Points[0].Equity)
	}
	if got.Points[len(got.Points)-1].Equity != 9992 {
		t.Fatalf("live last=%v", got.Points[len(got.Points)-1].Equity)
	}
}
