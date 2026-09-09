package domain

import (
	"math"
	"testing"
	"time"
)

func TestForwardReturnPct(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{OpenTime: t0, Close: "100"},
		{OpenTime: t0.Add(24 * time.Hour), Close: "110"},
		{OpenTime: t0.Add(5 * 24 * time.Hour), Close: "120"},
		{OpenTime: t0.Add(20 * 24 * time.Hour), Close: "130"},
	}
	r1 := ForwardReturnPct(candles, 0, 1)
	if r1 == nil || math.Abs(*r1-10) > 1e-9 {
		t.Fatalf("1d=%v", r1)
	}
	r5 := ForwardReturnPct(candles, 0, 5)
	if r5 == nil || math.Abs(*r5-20) > 1e-9 {
		t.Fatalf("5d=%v", r5)
	}
	r20 := ForwardReturnPct(candles, 0, 20)
	if r20 == nil || math.Abs(*r20-30) > 1e-9 {
		t.Fatalf("20d=%v", r20)
	}
	if ForwardReturnPct(candles, 0, 30) != nil {
		t.Fatal("want nil for missing future")
	}
}

func TestSummarizeBacktestReturns(t *testing.T) {
	t.Parallel()
	ptr := func(v float64) *float64 { return &v }
	got := SummarizeBacktestReturns([]ScannerBacktestSignal{
		{Return1d: ptr(10), Return5d: ptr(-2), Return20d: ptr(8)},
		{Return1d: ptr(-4), Return5d: ptr(6)},
		{Return1d: ptr(4), Return5d: ptr(4), Return20d: ptr(-2)},
	})
	if got.SignalCount != 3 {
		t.Fatalf("n=%d", got.SignalCount)
	}
	if got.Return1d.SampleCount != 3 || got.Return1d.Average == nil || *got.Return1d.Average != 3.3 {
		t.Fatalf("1d %+v", got.Return1d)
	}
	if got.Return1d.Median == nil || *got.Return1d.Median != 4 {
		t.Fatalf("1d median %+v", got.Return1d)
	}
	if got.Return1d.WinRate == nil || *got.Return1d.WinRate != 66.7 {
		t.Fatalf("1d win %+v", got.Return1d)
	}
	if got.Return5d.SampleCount != 3 || got.Return5d.Average == nil || *got.Return5d.Average != 2.7 {
		t.Fatalf("5d %+v", got.Return5d)
	}
	if got.Return20d.SampleCount != 2 || got.Return20d.WinRate == nil || *got.Return20d.WinRate != 50 {
		t.Fatalf("20d %+v", got.Return20d)
	}
	empty := SummarizeBacktestReturns(nil)
	if empty.Return1d.SampleCount != 0 || empty.Return1d.Average != nil {
		t.Fatalf("empty %+v", empty)
	}
}
