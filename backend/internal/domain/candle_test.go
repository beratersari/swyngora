package domain

import "testing"

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
