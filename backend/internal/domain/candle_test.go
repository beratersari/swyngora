package domain

import (
	"testing"
	"time"
)

func TestIsValidInterval(t *testing.T) {
	for _, iv := range SupportedIntervals {
		if !IsValidInterval(string(iv)) {
			t.Fatalf("expected valid: %s", iv)
		}
	}
	for _, bad := range []string{"", "2y", "1H", "60m", "daily"} {
		if IsValidInterval(bad) {
			t.Fatalf("expected invalid: %q", bad)
		}
	}
}

func TestSupportedIntervals_UniqueAndNonEmpty(t *testing.T) {
	if len(SupportedIntervals) == 0 {
		t.Fatal("empty SupportedIntervals")
	}
	seen := map[CandleInterval]bool{}
	for _, iv := range SupportedIntervals {
		if iv == "" {
			t.Fatal("empty interval in list")
		}
		if seen[iv] {
			t.Fatalf("duplicate interval %s", iv)
		}
		seen[iv] = true
	}
	// Sanity: core intervals present
	for _, want := range []CandleInterval{Interval1m, Interval1h, Interval1d, Interval1M} {
		if !seen[want] {
			t.Fatalf("missing %s", want)
		}
	}
}

func TestClosedCandlesDropsForming(t *testing.T) {
	now := time.Date(2026, 8, 28, 10, 30, 0, 0, time.UTC)
	closed := Candle{OpenTime: now.Add(-time.Hour), CloseTime: now.Add(-time.Millisecond), Close: "100"}
	forming := Candle{OpenTime: now.Truncate(time.Hour), CloseTime: now.Add(30 * time.Minute), Close: "1"}
	got := ClosedCandles([]Candle{closed, forming}, now)
	if len(got) != 1 || got[0].Close != "100" {
		t.Fatalf("got %+v", got)
	}
	if keep := ClosedCandles([]Candle{closed}, now); len(keep) != 1 {
		t.Fatalf("already-closed dropped: %+v", keep)
	}
	if zero := ClosedCandles([]Candle{{Close: "50"}}, now); len(zero) != 1 {
		t.Fatal("zero CloseTime must be kept")
	}
	if empty := ClosedCandles(nil, now); empty != nil {
		t.Fatalf("empty=%v", empty)
	}
	future := Candle{CloseTime: time.Now().UTC().Add(time.Hour), Close: "9"}
	if dropped := ClosedCandles([]Candle{closed, future}, time.Time{}); len(dropped) != 1 {
		t.Fatalf("zero now must still drop a future close: %+v", dropped)
	}
}
