package domain

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestMidrankPercentileBands(t *testing.T) {
	past := make([]float64, 100)
	for i := range past {
		past[i] = float64(i) // 0..99
	}
	cases := []struct {
		current float64
		wantPct float64
		want    string
	}{
		{current: 100, wantPct: 100, want: UnusualBandExtremeHigh},
		{current: 99, wantPct: 100, want: UnusualBandExtremeHigh}, // 99 below, 1 equal → 99.5 → 100
		{current: 98, wantPct: 99, want: UnusualBandExtremeHigh},  // 98 below, 1 equal → 98.5 → 99? wait 98 equals past[98]
		{current: 97, wantPct: 98, want: UnusualBandExtremeHigh},
		{current: 90, wantPct: 91, want: UnusualBandHigh},
		{current: 89, wantPct: 90, want: UnusualBandHigh},
		{current: 50, wantPct: 51, want: UnusualBandTypical},
		{current: 11, wantPct: 12, want: UnusualBandTypical},
		{current: 10, wantPct: 11, want: UnusualBandTypical},
		{current: 9, wantPct: 10, want: UnusualBandLow},
		{current: 3, wantPct: 4, want: UnusualBandLow},
		{current: 2, wantPct: 3, want: UnusualBandLow},
		{current: 1, wantPct: 2, want: UnusualBandExtremeLow},
		{current: 0, wantPct: 1, want: UnusualBandExtremeLow}, // 0 equal, 0 below → 0.5 → 1? 0.5*1/100*100 = 0.5 → 1
		{current: -1, wantPct: 0, want: UnusualBandExtremeLow},
	}
	// Recompute expected from the function so the table documents the bands,
	// and assert the band thresholds directly below.
	for _, tc := range cases {
		got := midrankPercentile(tc.current, past)
		band := bandForPercentile(got)
		if band != tc.want {
			t.Fatalf("current %v pct %v band %s, want %s", tc.current, got, band, tc.want)
		}
		if got != tc.wantPct {
			t.Fatalf("current %v pct %v, want %v", tc.current, got, tc.wantPct)
		}
	}
	if bandForPercentile(98) != UnusualBandExtremeHigh || bandForPercentile(97) != UnusualBandHigh {
		t.Fatal("98/97 boundary")
	}
	if bandForPercentile(90) != UnusualBandHigh || bandForPercentile(89) != UnusualBandTypical {
		t.Fatal("90/89 boundary")
	}
	if bandForPercentile(10) != UnusualBandLow || bandForPercentile(11) != UnusualBandTypical {
		t.Fatal("10/11 boundary")
	}
	if bandForPercentile(2) != UnusualBandExtremeLow || bandForPercentile(3) != UnusualBandLow {
		t.Fatal("2/3 boundary")
	}
}

func TestUnusualFundingExcludesCurrentSettlement(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	last := now.Add(-time.Hour)
	settled := make([]UnusualSample, 21)
	for i := range settled {
		settled[i] = UnusualSample{
			At:    last.Add(-time.Duration(20-i) * 8 * time.Hour),
			Value: float64(i + 1),
		}
	}
	got := UnusualFunding(now, settled, nil)
	if got.Band != UnusualBandExtremeHigh || got.Samples != 20 || got.Percentile != 100 {
		t.Fatalf("settled current must stay out of the baseline: %+v", got)
	}
	if got.Label != UnusualLabelSettled {
		t.Fatalf("label %s", got.Label)
	}
	if strings.Contains(got.Summary, "30 day") {
		t.Fatalf("summary promised a window the samples do not cover: %s", got.Summary)
	}
	if !strings.Contains(got.Summary, "extremely high") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestUnusualFundingUsesFreshPredictedOnly(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	settled := make([]UnusualSample, 25)
	for i := range settled {
		settled[i] = UnusualSample{At: now.Add(-time.Duration(25-i) * 8 * time.Hour), Value: 0.0001}
	}
	fresh := &UnusualSample{At: now.Add(-10 * time.Minute), Value: 0.01}
	got := UnusualFunding(now, settled, fresh)
	if got.Label != UnusualLabelPredicted || got.Samples != 25 || got.Band != UnusualBandExtremeHigh {
		t.Fatalf("fresh predicted: %+v", got)
	}
	stale := &UnusualSample{At: now.Add(-2 * time.Hour), Value: 0.01}
	got = UnusualFunding(now, settled, stale)
	if got.Label != UnusualLabelSettled || got.Samples != 24 || got.Current != 0.0001 {
		t.Fatalf("stale predicted falls back to settled: %+v", got)
	}
	if got.Band != UnusualBandTypical {
		t.Fatalf("flat settlements should be typical, got %s", got.Band)
	}
}

func TestUnusualFundingTooFewSamples(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	settled := []UnusualSample{
		{At: now.Add(-16 * time.Hour), Value: 0.0001},
		{At: now.Add(-8 * time.Hour), Value: 0.0002},
	}
	got := UnusualFunding(now, settled, nil)
	if got.Band != UnusualBandUnknown || got.Reason != UnusualReasonFewSamples || got.Samples != 1 {
		t.Fatalf("%+v", got)
	}
	empty := UnusualFunding(now, nil, nil)
	if empty.Reason != UnusualReasonNoSample {
		t.Fatalf("%+v", empty)
	}
}

func TestUnusualOpenInterestNonOverlapping(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	var pts []UnusualSample
	for h := 0; h <= 21*24; h++ {
		pts = append(pts, UnusualSample{At: start.Add(time.Duration(h) * time.Hour), Value: 100 + float64(h)})
	}
	got := UnusualOpenInterest(pts)
	if got.Band == UnusualBandUnknown || got.Samples != 20 {
		t.Fatalf("21 days of hourly OI should clear 20 past changes: %+v", got)
	}
	if got.Label != UnusualLabel24hChange || got.Unit != UnusualUnitPctChange {
		t.Fatalf("%+v", got)
	}
	short := pts[:10*24+1]
	few := UnusualOpenInterest(short)
	if few.Band != UnusualBandUnknown || few.Reason != UnusualReasonFewSamples || few.Samples != 9 {
		t.Fatalf("10 days is not enough: %+v", few)
	}
}

func TestUnusualOpenInterestSkipsGapAndZeroStart(t *testing.T) {
	end := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	gap := UnusualOpenInterest([]UnusualSample{
		{At: end.Add(-30 * time.Hour), Value: 100},
		{At: end, Value: 110},
	})
	if gap.Reason != UnusualReasonNoSample {
		t.Fatalf("6h gap at the 24h boundary must not invent a change: %+v", gap)
	}
	skipped := UnusualOpenInterest([]UnusualSample{
		{At: end.Add(-72 * time.Hour), Value: 90},
		{At: end.Add(-48 * time.Hour), Value: 100},
		{At: end.Add(-24 * time.Hour), Value: 0},
		{At: end, Value: 110},
	})
	if skipped.Reason != UnusualReasonFewSamples || skipped.Samples != 0 {
		t.Fatalf("zero start is skipped: %+v", skipped)
	}
	if skipped.Current == 0 || math.Abs(skipped.Current-((100.0-90)/90*100)) > 1e-9 {
		t.Fatalf("current should be the earlier valid window, got %v", skipped.Current)
	}
}

func TestUnusualLiquidationsUsesCompletedHourAndZeros(t *testing.T) {
	now := time.Date(2026, 9, 29, 15, 40, 0, 0, time.UTC)
	hourNow := now.Truncate(time.Hour)
	first := hourNow.Add(-25 * time.Hour) // 24 completed hours before the last completed hour, plus that hour
	var pts []UnusualSample
	for t := first; t.Before(hourNow); t = t.Add(time.Hour) {
		pts = append(pts, UnusualSample{At: t.Add(10 * time.Minute), Value: 1})
	}
	pts = append(pts, UnusualSample{At: hourNow.Add(20 * time.Minute), Value: 1_000_000})
	got := UnusualLiquidations(now, pts)
	if got.Current != 1 || got.Band != UnusualBandTypical || got.Samples != 24 {
		t.Fatalf("in-progress hour must not be scored: %+v", got)
	}
	if got.Label != UnusualLabelCompletedHour {
		t.Fatal(got.Label)
	}
	onlyNow := UnusualLiquidations(now, []UnusualSample{{At: now, Value: 50}})
	if onlyNow.Reason != UnusualReasonNoSample {
		t.Fatalf("%+v", onlyNow)
	}
}

func TestUnusualLiquidationQuietHoursCount(t *testing.T) {
	now := time.Date(2026, 9, 29, 15, 40, 0, 0, time.UTC)
	lastDone := now.Truncate(time.Hour).Add(-time.Hour)
	first := lastDone.Add(-24 * time.Hour)
	got := UnusualLiquidations(now, []UnusualSample{
		{At: first.Add(time.Minute), Value: 5},
		{At: lastDone.Add(time.Minute), Value: 500},
	})
	if got.Samples != 24 || got.Current != 500 || got.Band != UnusualBandExtremeHigh {
		t.Fatalf("zeros inside the covered span should make a burst extreme: %+v", got)
	}
}

func TestUnusualCVDSkipsToday(t *testing.T) {
	now := time.Date(2026, 9, 29, 18, 0, 0, 0, time.UTC)
	today := now.Truncate(24 * time.Hour)
	var days []UnusualSample
	for i := 21; i >= 1; i-- {
		days = append(days, UnusualSample{At: today.Add(-time.Duration(i) * 24 * time.Hour), Value: 10})
	}
	days = append(days, UnusualSample{At: today.Add(time.Hour), Value: 1_000_000_000})
	got := UnusualCVD(now, days)
	if got.Current != 10 || got.Samples != 20 || got.Band != UnusualBandTypical {
		t.Fatalf("today must stay out: %+v", got)
	}
	if got.Label != UnusualLabelCompletedDay {
		t.Fatal(got.Label)
	}
	onlyToday := UnusualCVD(now, []UnusualSample{{At: today, Value: 10}})
	if onlyToday.Reason != UnusualReasonNoSample {
		t.Fatalf("%+v", onlyToday)
	}
}

func TestUnusualSpreadLatestHour(t *testing.T) {
	end := time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC)
	var pts []UnusualSample
	for i := 24; i >= 1; i-- {
		pts = append(pts, UnusualSample{At: end.Add(-time.Duration(i) * time.Hour), Value: 0.01})
	}
	pts = append(pts, UnusualSample{At: end, Value: 1})
	got := UnusualSpread(pts)
	if got.Band != UnusualBandExtremeHigh || got.Samples != 24 || got.Current != 1 {
		t.Fatalf("%+v", got)
	}
	if got.Label != UnusualLabelLatestHour {
		t.Fatal(got.Label)
	}
}

func TestUnusualPrintSizeAndNoBook(t *testing.T) {
	printSize := UnusualPrintSize()
	if printSize.Band != UnusualBandUnknown || printSize.Reason != UnusualReasonNoStoredHistory {
		t.Fatalf("%+v", printSize)
	}
	if !strings.Contains(printSize.Summary, "15 minutes") || strings.Contains(printSize.Summary, "30 day") {
		t.Fatalf("%s", printSize.Summary)
	}
	book := UnusualNoBook()
	if book.Metric != UnusualMetricSpread || book.Reason != UnusualReasonNoBook {
		t.Fatalf("%+v", book)
	}
	raw, err := json.Marshal(printSize)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "0001-01-01") {
		t.Fatalf("zero time leaked: %s", raw)
	}
}

func TestMedianEvenCount(t *testing.T) {
	if got := medianFloat([]float64{1, 2, 3, 4}); got != 2.5 {
		t.Fatal(got)
	}
}
