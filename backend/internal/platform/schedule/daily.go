package schedule

import "time"

// NextDailyAt returns the next local time at hour:minute in loc after (or equal to) now.
// If that time today has already passed, it returns tomorrow's occurrence.
func NextDailyAt(now time.Time, hour, minute int, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	if hour < 0 {
		hour = 0
	}
	if hour > 23 {
		hour = 23
	}
	if minute < 0 {
		minute = 0
	}
	if minute > 59 {
		minute = 59
	}
	now = now.In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// DurationUntilDaily is nextDaily - now (always > 0 when next is strictly after now).
func DurationUntilDaily(now time.Time, hour, minute int, loc *time.Location) time.Duration {
	next := NextDailyAt(now, hour, minute, loc)
	d := next.Sub(now.In(loc))
	if d < 0 {
		return 0
	}
	return d
}
