package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestTypicalPrice(t *testing.T) {
	if got := TypicalPrice(120, 100, 110); got != 110 {
		t.Fatalf("%v", got)
	}
	if got := TypicalPrice(0, 0, 50); got != 50 {
		t.Fatalf("close-only %v", got)
	}
}

func TestComputeVWAP_WeightsByVolume(t *testing.T) {
	from := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	bars := []VolumeProfileBar{
		{Time: from, High: 100, Low: 100, Close: 100, Volume: 1_000_000},
		{Time: from.Add(time.Hour), High: 110, Low: 110, Close: 110, Volume: 9_000_000},
	}
	got := ComputeVWAP(ExchangeBinance, "BTCUSDT", bars, 112, from, from.Add(2*time.Hour), Interval1h)
	// (100*1 + 110*9) / 10 = 109
	if math.Abs(got.VWAP-109) > 1e-9 {
		t.Fatalf("vwap %v", got.VWAP)
	}
	if got.Volume != 10_000_000 || got.BarCount != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Side != VolumeProfileVsAbove || got.DistancePct < 2.7 || got.DistancePct > 2.8 {
		t.Fatalf("vs last %+v", got)
	}
	if !strings.Contains(got.Summary, "above") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestComputeVWAP_Empty(t *testing.T) {
	got := ComputeVWAP(ExchangeBinance, "BTCUSDT", nil, 0, time.Time{}, time.Time{}, Interval1h)
	if got.Error == "" {
		t.Fatal("expected error")
	}
}

func TestCombineVWAP_WeightsVenues(t *testing.T) {
	from := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	a := ComputeVWAP(ExchangeBinance, "ETHUSDT", []VolumeProfileBar{
		{High: 100, Low: 100, Close: 100, Volume: 2},
	}, 100, from, to, Interval5m)
	b := ComputeVWAP(ExchangeBybit, "ETHUSDT", []VolumeProfileBar{
		{High: 110, Low: 110, Close: 110, Volume: 8},
	}, 110, from, to, Interval5m)
	got := CombineVWAP("ETHUSDT", []VWAPVenue{a, b}, from, to, Interval5m)
	if got == nil || math.Abs(got.VWAP-108) > 1e-9 {
		t.Fatalf("%+v", got)
	}
	if got.Volume != 10 || len(got.Shares) != 2 {
		t.Fatalf("shares %+v", got)
	}
	// last = (100*2 + 110*8)/10 = 108 → at VWAP
	if got.LastPrice != 108 {
		t.Fatalf("last %v", got.LastPrice)
	}
}

func TestExplainVWAPReport(t *testing.T) {
	rep := VWAPReport{Combined: &VWAPVenue{Summary: "BTC VWAP 65000", Error: ""}}
	if ExplainVWAPReport(rep) != "Combined: BTC VWAP 65000" {
		t.Fatalf("%s", ExplainVWAPReport(rep))
	}
}
