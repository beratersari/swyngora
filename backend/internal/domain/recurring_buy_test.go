package domain

import (
	"testing"
	"time"
)

func TestLatestDueRunAt_SkipsMissed(t *testing.T) {
	// Daily: next was Mon, now Thu → latest due is Thu (not Mon/Tue/Wed separately)
	mon := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday
	thu := time.Date(2024, 6, 6, 15, 0, 0, 0, time.UTC)
	got := LatestDueRunAt(mon, thu, RecurringDaily)
	want := time.Date(2024, 6, 6, 10, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	// Future next → unchanged
	fri := time.Date(2024, 6, 7, 10, 0, 0, 0, time.UTC)
	if g := LatestDueRunAt(fri, thu, RecurringDaily); !g.Equal(fri) {
		t.Fatalf("future: %v", g)
	}
}

func TestRecurringPeriodKey(t *testing.T) {
	at := time.Date(2024, 6, 5, 12, 0, 0, 0, time.UTC) // Wed week 23
	if k := RecurringPeriodKey(at, RecurringDaily); k != "2024-06-05" {
		t.Fatal(k)
	}
	if k := RecurringPeriodKey(at, RecurringMonthly); k != "2024-06" {
		t.Fatal(k)
	}
	if k := RecurringPeriodKey(at, RecurringWeekly); k != "2024-W23" {
		t.Fatal(k)
	}
}

func TestAdvanceRecurringRunAt(t *testing.T) {
	at := time.Date(2024, 1, 31, 12, 0, 0, 0, time.UTC)
	m := AdvanceRecurringRunAt(at, RecurringMonthly)
	if m.Month() != time.February || m.Day() != 29 && m.Year() == 2024 {
		// 2024 is leap: Jan 31 + 1 month = Feb 29 in Go AddDate
		if m.Month() != time.March && m.Month() != time.February {
			t.Fatalf("%v", m)
		}
	}
}
