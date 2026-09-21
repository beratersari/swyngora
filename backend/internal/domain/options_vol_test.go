package domain

import (
	"math"
	"testing"
	"time"
)

func TestAnnualizedCloseVol_KnownSeries(t *testing.T) {
	t.Parallel()
	// 21 alternating ±1% simple moves → 20 log returns, enough for the floor.
	closes := make([]float64, 21)
	closes[0] = 100
	for i := 1; i < len(closes); i++ {
		if i%2 == 1 {
			closes[i] = closes[i-1] * 1.01
		} else {
			closes[i] = closes[i-1] / 1.01
		}
	}
	got, ok := AnnualizedCloseVol(closes, OptionsVolYearDays)
	if !ok || got <= 0 {
		t.Fatalf("got=%v ok=%v", got, ok)
	}
	rets := LogReturns(closes)
	if len(rets) != 20 {
		t.Fatalf("returns=%d", len(rets))
	}
	want := stdev(rets) * math.Sqrt(OptionsVolYearDays) * 100
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestAnnualizedCloseVol_TooFew(t *testing.T) {
	t.Parallel()
	if _, ok := AnnualizedCloseVol([]float64{100, 101, 102}, OptionsVolYearDays); ok {
		t.Fatal("expected false")
	}
}

func TestRealizedVolAt_Window(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	bars := make([]OHLCBar, 0, 40)
	px := 100.0
	for i := 0; i < 40; i++ {
		day := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		if i%2 == 1 {
			px *= 1.012
		} else {
			px *= 0.991
		}
		bars = append(bars, OHLCBar{Time: day, Close: px})
	}
	got, n, ok := RealizedVolAt(bars, at, OptionsRealizedDays)
	if !ok || n < optionsRealizedMinReturns+1 || got <= 0 {
		t.Fatalf("got=%v n=%d ok=%v", got, n, ok)
	}
	// Bars after `at` must not count.
	future := append([]OHLCBar{}, bars...)
	future = append(future, OHLCBar{Time: at.Add(24 * time.Hour), Close: px * 2})
	got2, n2, ok2 := RealizedVolAt(future, at, OptionsRealizedDays)
	if !ok2 || n2 != n || math.Abs(got2-got) > 1e-9 {
		t.Fatalf("future leaked n=%d/%d got=%v/%v", n, n2, got, got2)
	}
}

func TestAttachOptionsVolCompare_Gap(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 3, 31, 15, 0, 0, 0, time.UTC)
	bars := volTestBars(at, 35)
	rv, _, ok := RealizedVolAt(bars, at, OptionsRealizedDays)
	if !ok {
		t.Fatal("rv")
	}
	snap := &OptionsSnapshot{Asset: "BTC", AsOf: at, DVOL: 70, HasDVOL: true}
	AttachOptionsVolCompare(snap, bars)
	if snap.VolCompare == nil || !snap.VolCompare.HasGap {
		t.Fatalf("%+v", snap.VolCompare)
	}
	if math.Abs(snap.VolCompare.Realized-rv) > 1e-9 {
		t.Fatalf("realized=%v want=%v", snap.VolCompare.Realized, rv)
	}
	wantGap := 70 - rv
	if math.Abs(snap.VolCompare.Gap-wantGap) > 1e-9 {
		t.Fatalf("gap=%v want=%v", snap.VolCompare.Gap, wantGap)
	}
	if snap.VolCompare.Days != 30 {
		t.Fatalf("days=%d", snap.VolCompare.Days)
	}
	if wantGap >= optionsVolGapLeanPts && snap.VolCompare.Lean != OptionsVolLeanImplied {
		t.Fatalf("lean=%s gap=%v", snap.VolCompare.Lean, wantGap)
	}
	if snap.VolCompare.Summary == "" {
		t.Fatal("empty summary")
	}
}

func TestClassifyOptionsVolGap(t *testing.T) {
	t.Parallel()
	if ClassifyOptionsVolGap(25) != OptionsVolLeanImplied {
		t.Fatal("implied")
	}
	if ClassifyOptionsVolGap(-12) != OptionsVolLeanRealized {
		t.Fatal("realized")
	}
	if ClassifyOptionsVolGap(1.2) != OptionsVolLeanBalanced {
		t.Fatal("balanced")
	}
}

func TestExplainOptionsVolCompare_Example(t *testing.T) {
	t.Parallel()
	s := ExplainOptionsVolCompare("BTC", OptionsVolCompare{
		DVOL: 70, HasDVOL: true, Realized: 45, HasRealized: true,
		Gap: 25, HasGap: true, Lean: OptionsVolLeanImplied,
	})
	if s != "BTC DVOL 70 vs 30-day realized 45. Implied is 25 vol points richer." {
		t.Fatalf("%q", s)
	}
}

func volTestBars(at time.Time, n int) []OHLCBar {
	start := at.UTC().Truncate(24*time.Hour).AddDate(0, 0, -n)
	out := make([]OHLCBar, 0, n)
	px := 80000.0
	for i := 0; i < n; i++ {
		if i%3 == 0 {
			px *= 1.02
		} else if i%3 == 1 {
			px *= 0.985
		} else {
			px *= 1.004
		}
		out = append(out, OHLCBar{Time: start.AddDate(0, 0, i), Close: px})
	}
	return out
}
