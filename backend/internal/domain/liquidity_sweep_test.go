package domain

import (
	"strings"
	"testing"
	"time"
)

func sweepAt(t0 time.Time, i int, o, h, l, c, vol float64) SweepBar {
	return SweepBar{
		Time: t0.Add(time.Duration(i) * 15 * time.Minute),
		Open: o, High: h, Low: l, Close: c, Volume: vol,
		BuyVolume: vol * 0.4, SellVolume: vol * 0.6, BuySellKnown: true,
	}
}

func TestDetectLiquiditySweeps_HighWickComesBack(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	// Two swing highs near 100.5, then a poke to 100.8 that closes back under.
	seq := [][5]float64{
		{96, 96.2, 95.8, 96, 10},
		{96, 97, 95.9, 97, 10},
		{97, 98, 96.8, 98, 10},
		{98, 99, 97.8, 99, 10},
		{99, 100.5, 98.8, 99.2, 20}, // swing high
		{99.2, 99.3, 98.5, 98.8, 10},
		{98.8, 98.9, 97.4, 97.8, 10},
		{97.8, 98.6, 97.5, 98.4, 10},
		{98.4, 100.4, 98.2, 99.1, 20}, // second swing high
		{99.1, 99.2, 98.4, 98.7, 10},
		{98.7, 98.8, 97.9, 98.2, 10},
		{98.2, 99.1, 98.0, 98.9, 10},
		{98.9, 100.8, 98.7, 99.3, 50}, // sweep
	}
	bars := make([]SweepBar, len(seq))
	for i, r := range seq {
		bars[i] = sweepAt(t0, i, r[0], r[1], r[2], r[3], r[4])
	}
	got := DetectLiquiditySweeps(bars, 15*time.Minute)
	if len(got) != 1 {
		t.Fatalf("sweeps=%d %+v", len(got), got)
	}
	s := got[0]
	if s.Side != LiquiditySweepSideHigh || s.Status != LiquiditySweepSwept {
		t.Fatalf("%+v", s)
	}
	if s.Tests < 2 {
		t.Fatalf("tests %d", s.Tests)
	}
	if s.Level < 100.4 || s.Level > 100.6 {
		t.Fatalf("level %v", s.Level)
	}
	if s.Extreme < 100.79 || s.ExcursionPct < 0.2 {
		t.Fatalf("excursion %+v", s)
	}
	if s.Volume != 50 || !s.BuySellKnown {
		t.Fatalf("vol %+v", s)
	}
	if s.Bars != 1 || s.DurationSeconds != 15*60 {
		t.Fatalf("dur %+v", s)
	}
	if !strings.Contains(s.Summary, "swept") {
		t.Fatalf("summary %s", s.Summary)
	}
}

func TestDetectLiquiditySweeps_LowWickComesBack(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seq := [][5]float64{
		{104, 104.2, 103.8, 104, 10},
		{104, 104.1, 103, 103.2, 10},
		{103.2, 103.4, 102.2, 102.5, 10},
		{102.5, 102.8, 101.2, 101.5, 10},
		{101.5, 101.8, 99.5, 100.2, 20}, // swing low 99.5
		{100.2, 101.2, 100.1, 100.8, 10},
		{100.8, 102.4, 100.6, 102, 10},
		{102, 102.2, 101.4, 101.6, 10},
		{101.6, 101.8, 99.6, 100.4, 20}, // second swing low 99.6
		{100.4, 101.1, 100.2, 100.6, 10},
		{100.6, 101.4, 100.4, 101.1, 10},
		{101.1, 101.3, 100.5, 100.8, 10},
		{100.8, 101.0, 99.2, 100.5, 40}, // sweep low 99.2, close back over
	}
	bars := make([]SweepBar, len(seq))
	for i, r := range seq {
		bars[i] = sweepAt(t0, i, r[0], r[1], r[2], r[3], r[4])
	}
	got := DetectLiquiditySweeps(bars, 15*time.Minute)
	if len(got) != 1 || got[0].Side != LiquiditySweepSideLow {
		t.Fatalf("%+v", got)
	}
	if got[0].Extreme > 99.25 || got[0].Level < 99.45 {
		t.Fatalf("%+v", got[0])
	}
}

func TestDetectLiquiditySweeps_BreakoutIsNotSweep(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seq := [][5]float64{
		{96, 96.2, 95.8, 96, 10},
		{96, 97, 95.9, 97, 10},
		{97, 98, 96.8, 98, 10},
		{98, 99, 97.8, 99, 10},
		{99, 100.5, 98.8, 99.2, 20},
		{99.2, 99.3, 98.5, 98.8, 10},
		{98.8, 98.9, 97.4, 97.8, 10},
		{97.8, 98.6, 97.5, 98.4, 10},
		{98.4, 100.4, 98.2, 99.1, 20},
		{99.1, 99.2, 98.4, 98.7, 10},
		{98.7, 98.8, 97.9, 98.2, 10},
	}
	// Stay through the high for longer than sweepMaxBars.
	for i := 0; i < sweepMaxBars+2; i++ {
		c := 101.0 + float64(i)*0.1
		seq = append(seq, [5]float64{c, c + 0.4, c - 0.1, c + 0.2, 10})
	}
	bars := make([]SweepBar, len(seq))
	for i, r := range seq {
		bars[i] = sweepAt(t0, i, r[0], r[1], r[2], r[3], r[4])
	}
	got := DetectLiquiditySweeps(bars, 15*time.Minute)
	if len(got) != 0 {
		t.Fatalf("breakout counted as sweep %+v", got)
	}
}

func TestDetectLiquiditySweeps_NeedsTwoTests(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seq := [][5]float64{
		{96, 96.2, 95.8, 96, 10},
		{96, 97, 95.9, 97, 10},
		{97, 98, 96.8, 98, 10},
		{98, 99, 97.8, 99, 10},
		{99, 100.5, 98.8, 99.2, 20},
		{99.2, 99.3, 98.5, 98.8, 10},
		{98.8, 98.9, 97.4, 97.8, 10},
		{97.8, 98.2, 97.5, 97.9, 10},
		{97.9, 100.8, 97.6, 98.1, 40},
	}
	bars := make([]SweepBar, len(seq))
	for i, r := range seq {
		bars[i] = sweepAt(t0, i, r[0], r[1], r[2], r[3], r[4])
	}
	got := DetectLiquiditySweeps(bars, 15*time.Minute)
	if len(got) != 0 {
		t.Fatalf("single test should not sweep %+v", got)
	}
}

func TestBuildLiquiditySweepVenue_Summary(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	seq := [][5]float64{
		{96, 96.2, 95.8, 96, 10}, {96, 97, 95.9, 97, 10}, {97, 98, 96.8, 98, 10},
		{98, 99, 97.8, 99, 10}, {99, 100.5, 98.8, 99.2, 20}, {99.2, 99.3, 98.5, 98.8, 10},
		{98.8, 98.9, 97.4, 97.8, 10}, {97.8, 98.6, 97.5, 98.4, 10}, {98.4, 100.4, 98.2, 99.1, 20},
		{99.1, 99.2, 98.4, 98.7, 10}, {98.7, 98.8, 97.9, 98.2, 10}, {98.2, 99.1, 98.0, 98.9, 10},
		{98.9, 100.8, 98.7, 99.3, 50},
	}
	bars := make([]SweepBar, len(seq))
	for i, r := range seq {
		bars[i] = sweepAt(t0, i, r[0], r[1], r[2], r[3], r[4])
	}
	got := BuildLiquiditySweepVenue(ExchangeBinance, "BTCUSDT", bars, 99.3, 15*time.Minute)
	if got.Summary == "" || got.Current == nil || len(got.Sweeps) != 1 {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "BTC") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestDetectLiquiditySweeps_LaterSwingDoesNotRewritePastSweep(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	// Shallow poke through 100.5 (100.55). A later swing at 100.64 is still
	// inside the 0.15% cluster — it must not raise the old shelf and erase the sweep.
	seq := [][5]float64{
		{96, 96.2, 95.8, 96, 10},
		{96, 97, 95.9, 97, 10},
		{97, 98, 96.8, 98, 10},
		{98, 99, 97.8, 99, 10},
		{99, 100.5, 98.8, 99.2, 20},
		{99.2, 99.3, 98.5, 98.8, 10},
		{98.8, 98.9, 97.4, 97.8, 10},
		{97.8, 98.6, 97.5, 98.4, 10},
		{98.4, 100.4, 98.2, 99.1, 20},
		{99.1, 99.2, 98.4, 98.7, 10},
		{98.7, 98.8, 97.9, 98.2, 10},
		{98.2, 99.1, 98.0, 98.9, 10},
		{98.9, 100.55, 98.7, 99.3, 50}, // sweep — only just through 100.5
		{99.3, 99.4, 98.8, 99.0, 10},
		{99.0, 99.1, 98.4, 98.6, 10},
		{98.6, 99.2, 98.5, 99.0, 10},
		{99.0, 100.64, 98.8, 99.2, 20}, // later swing, same cluster
		{99.2, 99.3, 98.7, 98.9, 10},
		{98.9, 99.0, 98.5, 98.7, 10},
	}
	short := make([]SweepBar, 13)
	full := make([]SweepBar, len(seq))
	for i, r := range seq {
		b := sweepAt(t0, i, r[0], r[1], r[2], r[3], r[4])
		full[i] = b
		if i < 13 {
			short[i] = b
		}
	}
	before := DetectLiquiditySweeps(short, 15*time.Minute)
	if len(before) != 1 || before[0].Status != LiquiditySweepSwept || before[0].Level > 100.51 {
		t.Fatalf("before %+v", before)
	}
	after := DetectLiquiditySweeps(full, 15*time.Minute)
	var kept *LiquiditySweep
	for i := range after {
		if after[i].PiercedAt.Equal(before[0].PiercedAt) {
			kept = &after[i]
			break
		}
	}
	if kept == nil {
		t.Fatalf("past sweep disappeared after later swing joined the cluster: %+v", after)
	}
	if kept.Level != before[0].Level {
		t.Fatalf("level rewritten %v -> %v", before[0].Level, kept.Level)
	}
	if kept.Status != LiquiditySweepSwept {
		t.Fatalf("status %+v", kept)
	}
}

func TestSweepBarsFromCandles_BuySell(t *testing.T) {
	bars := SweepBarsFromCandles([]Candle{
		{OpenTime: time.Unix(1, 0).UTC(), Open: "10", High: "11", Low: "9", Close: "10.5", QuoteVolume: "100", TakerBuyQuote: "70"},
		{OpenTime: time.Unix(2, 0).UTC(), Open: "10", High: "11", Low: "9", Close: "10.5", QuoteVolume: "50"},
	})
	if len(bars) != 2 || !bars[0].BuySellKnown || bars[0].BuyVolume != 70 || bars[0].SellVolume != 30 {
		t.Fatalf("%+v", bars)
	}
	if bars[1].BuySellKnown {
		t.Fatalf("bybit-like %+v", bars[1])
	}
}
