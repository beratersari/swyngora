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
		10000, "USDT", "c1", carry, []EquitySnapshot{mid}, live, nil, 0, "note",
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

func TestMaxDrawdown_PeakToTroughTimes(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	pts := []EquityPoint{
		{Time: t0, Equity: 10000},
		{Time: t0.Add(time.Hour), Equity: 12000},
		{Time: t0.Add(2 * time.Hour), Equity: 9000},
		{Time: t0.Add(3 * time.Hour), Equity: 11000},
		{Time: t0.Add(4 * time.Hour), Equity: 8000},
	}
	amt, pct, from, to := MaxDrawdown(pts)
	if math.Abs(amt-4000) > 1e-9 {
		t.Fatalf("amount=%v", amt)
	}
	if pct == nil || math.Abs(*pct-4000.0/12000*100) > 1e-9 {
		t.Fatalf("pct=%v", pct)
	}
	if !from.Equal(t0.Add(time.Hour)) || !to.Equal(t0.Add(4*time.Hour)) {
		t.Fatalf("window %s → %s", from, to)
	}

	flat, flatPct, flatFrom, _ := MaxDrawdown([]EquityPoint{
		{Time: t0, Equity: 100},
		{Time: t0.Add(time.Hour), Equity: 110},
	})
	if flat != 0 || flatPct != nil || !flatFrom.IsZero() {
		t.Fatalf("rising book should have no drawdown: %v %v %s", flat, flatPct, flatFrom)
	}
}

func TestAssemblePerformance_YoungPortfolio(t *testing.T) {
	created := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
	requested := end.Add(-7 * 24 * time.Hour)
	live := EquityPoint{Time: end, Equity: 10100, CashBalance: 10100}
	got := AssemblePerformance(
		PerformancePeriod1W, requested, end, created,
		10000, "USDT", "c2", nil, nil, live, nil, 0, "",
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
		10000, "USDT", "c3", nil, []EquitySnapshot{pre}, live, nil, 0, "",
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

func TestAssemblePerformance_DepositAndWithdrawalAreNotPnL(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	created := start.Add(-30 * 24 * time.Hour)
	depositAt := start.Add(24 * time.Hour)
	withdrawAt := start.Add(48 * time.Hour)
	flows := []ExternalFlow{
		{At: depositAt, NetAfter: 5000},
		{At: withdrawAt, NetAfter: 3000},
	}
	snaps := []EquitySnapshot{
		{BucketAt: start, TakenAt: start, Equity: 10000, CashBalance: 10000},
		{BucketAt: depositAt, TakenAt: depositAt, Equity: 15000, CashBalance: 15000},
		{BucketAt: withdrawAt, TakenAt: withdrawAt, Equity: 13000, CashBalance: 13000},
	}
	live := EquityPoint{Time: end, Equity: 13100, CashBalance: 13100}
	got := AssemblePerformance(
		PerformancePeriod1W, start, end, created,
		10000, "USDT", "c4", nil, snaps, live, flows, 3000, "",
	)
	// +100 of trading after the cash moves. 5000 in and 2000 out are not P&L.
	if math.Abs(got.ChangeAmount-100) > 1e-9 {
		t.Fatalf("change=%v", got.ChangeAmount)
	}
	if math.Abs(got.NetFlow-3000) > 1e-9 {
		t.Fatalf("netFlow=%v", got.NetFlow)
	}
	if got.ChangePct == nil || *got.ChangePct <= 0 {
		t.Fatalf("pct=%v", got.ChangePct)
	}
	last := got.Points[len(got.Points)-1]
	if math.Abs((last.Equity-last.ExternalFlow)-10100) > 1e-6 {
		t.Fatalf("performance end=%v flow=%v", last.Equity, last.ExternalFlow)
	}
	// Raw equity falls 2000 on the withdrawal. The performance line does not.
	if got.MaxDrawdown > 1e-6 {
		t.Fatalf("drawdown=%v from external cash", got.MaxDrawdown)
	}
}

func TestAssemblePerformance_LossAfterDepositIsDrawdown(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)
	depositAt := start.Add(time.Hour)
	lossAt := start.Add(24 * time.Hour)
	flows := []ExternalFlow{{At: depositAt, NetAfter: 5000}}
	snaps := []EquitySnapshot{
		{BucketAt: start, TakenAt: start, Equity: 10000},
		{BucketAt: depositAt, TakenAt: depositAt, Equity: 15000},
		{BucketAt: lossAt, TakenAt: lossAt, Equity: 14000},
	}
	live := EquityPoint{Time: end, Equity: 14000}
	got := AssemblePerformance(
		PerformancePeriod1W, start, end, start.Add(-24*time.Hour),
		10000, "USDT", "c5", nil, snaps, live, flows, 5000, "",
	)
	if math.Abs(got.ChangeAmount-(-1000)) > 1e-9 {
		t.Fatalf("change=%v", got.ChangeAmount)
	}
	if math.Abs(got.MaxDrawdown-1000) > 1e-6 {
		t.Fatalf("drawdown=%v", got.MaxDrawdown)
	}
	// Deposit at 1h of a 48h window keeps most of its weight.
	// Base ≈ 10000 + 5000*(47/48). Profit -1000 → about -6.8%, not -10% and not +40%.
	if got.ChangePct == nil || *got.ChangePct > -6 || *got.ChangePct < -8 {
		t.Fatalf("pct=%v", got.ChangePct)
	}
}

func TestAssemblePerformance_StaleCarryDoesNotCountLaterDeposit(t *testing.T) {
	start := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	carryAt := start.Add(-2 * time.Hour)
	depositAt := start.Add(-time.Hour)
	carry := &EquitySnapshot{Equity: 10000, CashBalance: 10000, TakenAt: carryAt, BucketAt: carryAt}
	flows := []ExternalFlow{{At: depositAt, NetAfter: 5000}}
	live := EquityPoint{Time: end, Equity: 15000, CashBalance: 15000}
	got := AssemblePerformance(
		PerformancePeriod1D, start, end, start.Add(-30*24*time.Hour),
		10000, "USDT", "c6", carry, nil, live, flows, 5000, "",
	)
	if math.Abs(got.ChangeAmount) > 1e-9 {
		t.Fatalf("stale carry counted deposit as profit: %v", got.ChangeAmount)
	}
	if math.Abs(got.NetFlow-5000) > 1e-9 {
		t.Fatalf("netFlow=%v", got.NetFlow)
	}
	if got.MaxDrawdown > 1e-6 {
		t.Fatalf("drawdown=%v", got.MaxDrawdown)
	}
}

func TestModifiedDietz_WithdrawalKeepsZeroReturn(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)
	mid := start.Add(24 * time.Hour)
	pct := ModifiedDietzPct(10000, 0, start, end, []FlowDelta{{At: mid, Amount: -5000}})
	if pct == nil || math.Abs(*pct) > 1e-9 {
		t.Fatalf("pct=%v", pct)
	}
	if ModifiedDietzPct(0, 100, start, end, nil) != nil {
		t.Fatal("zero base should be nil")
	}
}
