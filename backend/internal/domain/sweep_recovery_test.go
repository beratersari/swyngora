package domain

import (
	"testing"
	"time"
)

func TestMeasureSweepRecovery_RefillsInFiveMinutes(t *testing.T) {
	base := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	sweep := LiquiditySweep{
		Side: LiquiditySweepSideHigh, Level: 100, Status: LiquiditySweepSwept,
		ExcursionPct: 0.4, PiercedAt: base, ReclaimedAt: base.Add(10 * time.Minute),
	}
	snaps := []BookHistorySnapshot{
		{SampledAt: base.Add(-2 * time.Minute), Mid: 100, Asks: []BookHistoryLevel{{Price: 100.2, Notional: 10000}}},
		{SampledAt: base.Add(4 * time.Minute), Mid: 100.3, Asks: []BookHistoryLevel{{Price: 100.2, Notional: 1000}}},
		{SampledAt: base.Add(15 * time.Minute), Mid: 100, Asks: []BookHistoryLevel{{Price: 100.2, Notional: 8500}}},
	}
	got := MeasureSweepRecovery("binance", "BTCUSDT", sweep, snaps)
	if !got.Recovered || got.RecoverySeconds != 300 {
		t.Fatalf("%+v", got)
	}
	if got.TroughNotional != "1000" || got.BeforeNotional != "10000" {
		t.Fatalf("sizes %+v", got)
	}
}

func TestMeasureSweepRecovery_StillThin(t *testing.T) {
	base := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	sweep := LiquiditySweep{
		Side: LiquiditySweepSideLow, Level: 100, Status: LiquiditySweepSwept,
		ExcursionPct: 0.4, PiercedAt: base, ReclaimedAt: base.Add(5 * time.Minute),
	}
	snaps := []BookHistorySnapshot{
		{SampledAt: base.Add(-1 * time.Minute), Mid: 100, Bids: []BookHistoryLevel{{Price: 99.8, Notional: 8000}}},
		{SampledAt: base.Add(10 * time.Minute), Mid: 100, Bids: []BookHistoryLevel{{Price: 99.8, Notional: 2000}}},
	}
	got := MeasureSweepRecovery("bybit", "ETHUSDT", sweep, snaps)
	if got.Recovered || got.BeforeNotional == "" {
		t.Fatalf("%+v", got)
	}
}

func TestBuildSweepRecoveryReport_Median(t *testing.T) {
	now := time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)
	rows := []SweepRecovery{
		{Recovered: true, RecoverySeconds: 120, ReclaimedAt: now.Add(-time.Hour), BeforeNotional: "1"},
		{Recovered: true, RecoverySeconds: 600, ReclaimedAt: now, BeforeNotional: "1"},
		{Recovered: false, BeforeNotional: "1", ReclaimedAt: now.Add(-30 * time.Minute)},
	}
	got := BuildSweepRecoveryReport("BTCUSDT", "all", rows, now)
	if got.MedianRecoverySeconds != 360 || got.Recovered != 2 || got.Pending != 1 {
		t.Fatalf("%+v", got)
	}
}
