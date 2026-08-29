package domain

import (
	"testing"
	"time"
)

func TestGradeFromScore(t *testing.T) {
	t.Parallel()
	if GradeFromScore(3) != ScannerSetupGradeA || GradeFromScore(2) != ScannerSetupGradeB || GradeFromScore(1) != ScannerSetupGradeC {
		t.Fatalf("grades A/B/C")
	}
}

func scannerHit(rule ScannerRuleType, matched, bar string) ScannerResult {
	mt, _ := time.Parse(time.RFC3339, matched)
	return ScannerResult{
		ID:            string(rule) + "-" + bar,
		RuleID:        "r1",
		Exchange:      ExchangeBinance,
		Symbol:        "BTCUSDT",
		RuleType:      rule,
		Interval:      "4h",
		MarketDataKey: bar,
		MatchedAt:     mt,
		Summary:       string(rule) + " hit",
	}
}

func TestBuildScannerSetups_RequiresTwoFactors(t *testing.T) {
	t.Parallel()
	now, _ := time.Parse(time.RFC3339, "2026-08-01T12:00:00Z")
	out := BuildScannerSetups([]ScannerResult{
		scannerHit(ScannerRuleRSI, "2026-08-01T10:00:00Z", "b1"),
		scannerHit(ScannerRuleRSI, "2026-08-01T11:00:00Z", "b2"),
	}, now, 0)
	if len(out) != 0 {
		t.Fatalf("got %+v", out)
	}
}

func TestBuildScannerSetups_GradesSameBar(t *testing.T) {
	t.Parallel()
	now, _ := time.Parse(time.RFC3339, "2026-08-01T12:00:00Z")
	bar := "2026-08-01T08:00:00Z"
	out := BuildScannerSetups([]ScannerResult{
		scannerHit(ScannerRuleRSI, "2026-08-01T08:01:00Z", bar),
		scannerHit(ScannerRuleMACrossover, "2026-08-01T08:01:00Z", bar),
		scannerHit(ScannerRuleVolumeIncrease, "2026-08-01T04:01:00Z", "2026-08-01T04:00:00Z"),
	}, now, 0)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	got := out[0]
	if got.Score != 3 || got.Grade != ScannerSetupGradeA || !got.SameBar {
		t.Fatalf("%+v", got)
	}
	if len(got.Factors) != 3 || got.Factors[0] != ScannerRuleMACrossover {
		t.Fatalf("factors=%v", got.Factors)
	}
}

func TestBuildScannerSetups_DropsOldHits(t *testing.T) {
	t.Parallel()
	now, _ := time.Parse(time.RFC3339, "2026-08-01T12:00:00Z")
	out := BuildScannerSetups([]ScannerResult{
		scannerHit(ScannerRuleRSI, "2026-07-01T00:00:00Z", "old"),
		scannerHit(ScannerRuleMACrossover, "2026-08-01T10:00:00Z", "new"),
	}, now, 0)
	if len(out) != 0 {
		t.Fatalf("got %+v", out)
	}
}

func TestCountHitsSince(t *testing.T) {
	t.Parallel()
	since, _ := time.Parse(time.RFC3339, "2026-08-01T00:00:00Z")
	n := CountHitsSince([]ScannerResult{
		scannerHit(ScannerRuleRSI, "2026-08-01T01:00:00Z", "a"),
		scannerHit(ScannerRuleRSI, "2026-07-01T01:00:00Z", "b"),
	}, since)
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
}
