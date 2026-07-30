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
