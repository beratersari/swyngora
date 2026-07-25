package schedule

import (
	"testing"
	"time"
)

func TestNextDailyAt_LaterToday(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 7, 25, 1, 0, 0, 0, loc)
	next := NextDailyAt(now, 3, 0, loc)
	want := time.Date(2026, 7, 25, 3, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %v want %v", next, want)
	}
}

func TestNextDailyAt_AlreadyPassed_Tomorrow(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 7, 25, 4, 0, 0, 0, loc)
	next := NextDailyAt(now, 3, 0, loc)
	want := time.Date(2026, 7, 26, 3, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %v want %v", next, want)
	}
}

func TestNextDailyAt_ExactNow_GoesTomorrow(t *testing.T) {
	// "After" is strict — at exactly 03:00, schedule next day to avoid double-fire loops.
	loc := time.UTC
	now := time.Date(2026, 7, 25, 3, 0, 0, 0, loc)
	next := NextDailyAt(now, 3, 0, loc)
	want := time.Date(2026, 7, 26, 3, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %v want %v", next, want)
	}
}

func TestDurationUntilDaily(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 7, 25, 2, 0, 0, 0, loc)
	d := DurationUntilDaily(now, 3, 0, loc)
	if d != time.Hour {
		t.Fatalf("d=%v", d)
	}
}
