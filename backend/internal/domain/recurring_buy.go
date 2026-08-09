package domain

import (
	"fmt"
	"strings"
	"time"
)

// Recurring buy plan limits (paper trading only).
const (
	MaxRecurringBuyPlansPerClient = 20
	MinRecurringBuyAmount         = 1.0
	MaxRecurringBuyAmount         = 1_000_000.0
	MaxRecurringBuyNameLen        = 80
	MinRecurringIntervalHours     = 1
	MaxRecurringIntervalHours     = 168 // 7 days
)

// RecurringBuyFrequency is how often a plan executes.
type RecurringBuyFrequency string

const (
	RecurringDaily    RecurringBuyFrequency = "daily"
	RecurringWeekly   RecurringBuyFrequency = "weekly"
	RecurringMonthly  RecurringBuyFrequency = "monthly"
	RecurringInterval RecurringBuyFrequency = "interval" // every IntervalHours hours
)

// RecurringBuyPlanStatus is active (runs) or paused (held).
type RecurringBuyPlanStatus string

const (
	RecurringBuyActive RecurringBuyPlanStatus = "active"
	RecurringBuyPaused RecurringBuyPlanStatus = "paused"
)

// RecurringBuyRunStatus is the outcome of one scheduled execution attempt.
type RecurringBuyRunStatus string

const (
	RecurringBuyRunSucceeded RecurringBuyRunStatus = "succeeded"
	RecurringBuyRunFailed    RecurringBuyRunStatus = "failed"
)

// RecurringBuyPlan is a paper DCA-style buy schedule for a client.
// Amount is cash (portfolio currency) spent each run at slipped last price including the taker fee.
type RecurringBuyPlan struct {
	ID            string
	ClientID      string
	Exchange      Exchange
	Symbol        string
	Name          string // user label e.g. "Salary Day Buy"
	Amount        float64 // cash notional per run
	Frequency     RecurringBuyFrequency
	Weekday       string // monday..sunday; weekly only (optional)
	DayOfMonth    int    // 1-31; monthly only (optional; 0 = anniversary of start)
	IntervalHours int    // 1-168; interval frequency only
	Status        RecurringBuyPlanStatus
	NextRunAt     time.Time
	LastRunAt     *time.Time
	LastPeriodKey string // last claimed period key (idempotency helper)
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RecurringBuyRun records one attempt for a plan period (unique plan_id + period_key).
type RecurringBuyRun struct {
	ID           string
	PlanID       string
	ClientID     string
	PeriodKey    string // e.g. 2024-06-15 | 2024-W24 | 2024-06
	Status       RecurringBuyRunStatus
	Amount       float64
	Quantity     float64
	Price        float64
	TradeID      string
	FailReason   string
	ScheduledFor time.Time // the schedule instant this run represents
	ExecutedAt   time.Time
}

// IsValidRecurringBuyFrequency reports daily|weekly|monthly|interval.
func IsValidRecurringBuyFrequency(s string) bool {
	switch RecurringBuyFrequency(strings.ToLower(strings.TrimSpace(s))) {
	case RecurringDaily, RecurringWeekly, RecurringMonthly, RecurringInterval:
		return true
	default:
		return false
	}
}

// NormalizeRecurringBuyFrequency parses frequency.
func NormalizeRecurringBuyFrequency(s string) (RecurringBuyFrequency, error) {
	f := RecurringBuyFrequency(strings.ToLower(strings.TrimSpace(s)))
	if !IsValidRecurringBuyFrequency(string(f)) {
		return "", fmt.Errorf("%w: frequency must be daily, weekly, monthly, or interval", ErrInvalidArgument)
	}
	return f, nil
}

// ParseRecurringWeekday returns time.Weekday for monday..sunday.
func ParseRecurringWeekday(s string) (time.Weekday, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sunday":
		return time.Sunday, nil
	case "monday":
		return time.Monday, nil
	case "tuesday":
		return time.Tuesday, nil
	case "wednesday":
		return time.Wednesday, nil
	case "thursday":
		return time.Thursday, nil
	case "friday":
		return time.Friday, nil
	case "saturday":
		return time.Saturday, nil
	default:
		return 0, fmt.Errorf("%w: weekday must be monday..sunday", ErrInvalidArgument)
	}
}

// NormalizeRecurringBuyName trims name; empty becomes a default from symbol + frequency.
func NormalizeRecurringBuyName(name, symbol string, freq RecurringBuyFrequency) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		if symbol == "" {
			symbol = "coin"
		}
		f := string(freq)
		if f == "" {
			f = string(RecurringDaily)
		}
		return symbol + " " + f, nil
	}
	if strings.ContainsAny(name, "\x00\r\n") {
		return "", fmt.Errorf("%w: name must be a single line", ErrInvalidArgument)
	}
	if len([]rune(name)) > MaxRecurringBuyNameLen {
		return "", fmt.Errorf("%w: name must be at most %d characters", ErrInvalidArgument, MaxRecurringBuyNameLen)
	}
	return name, nil
}

// ValidateRecurringSchedule checks weekday / dayOfMonth / intervalHours vs frequency.
func ValidateRecurringSchedule(freq RecurringBuyFrequency, weekday string, dayOfMonth, intervalHours int) error {
	weekday = strings.TrimSpace(weekday)
	switch freq {
	case RecurringInterval:
		if intervalHours < MinRecurringIntervalHours || intervalHours > MaxRecurringIntervalHours {
			return fmt.Errorf("%w: intervalHours must be between %d and %d", ErrInvalidArgument, MinRecurringIntervalHours, MaxRecurringIntervalHours)
		}
		if weekday != "" || dayOfMonth != 0 {
			return fmt.Errorf("%w: weekday and dayOfMonth are not valid for interval frequency", ErrInvalidArgument)
		}
	case RecurringWeekly:
		if intervalHours != 0 {
			return fmt.Errorf("%w: intervalHours is only valid for interval frequency", ErrInvalidArgument)
		}
		if weekday != "" {
			if _, err := ParseRecurringWeekday(weekday); err != nil {
				return err
			}
		}
		if dayOfMonth != 0 {
			return fmt.Errorf("%w: dayOfMonth is only valid for monthly frequency", ErrInvalidArgument)
		}
	case RecurringMonthly:
		if intervalHours != 0 {
			return fmt.Errorf("%w: intervalHours is only valid for interval frequency", ErrInvalidArgument)
		}
		if weekday != "" {
			return fmt.Errorf("%w: weekday is only valid for weekly frequency", ErrInvalidArgument)
		}
		if dayOfMonth != 0 && (dayOfMonth < 1 || dayOfMonth > 31) {
			return fmt.Errorf("%w: dayOfMonth must be 1-31", ErrInvalidArgument)
		}
	default: // daily
		if intervalHours != 0 || weekday != "" || dayOfMonth != 0 {
			return fmt.Errorf("%w: weekday, dayOfMonth, and intervalHours are not valid for %s frequency", ErrInvalidArgument, freq)
		}
	}
	return nil
}

// RecurringPeriodKey identifies a calendar period for idempotent execution.
func RecurringPeriodKey(at time.Time, freq RecurringBuyFrequency) string {
	return RecurringPeriodKeyPlan(at, RecurringBuyPlan{Frequency: freq})
}

// RecurringPeriodKeyPlan uses interval-aware keys when frequency is interval.
func RecurringPeriodKeyPlan(at time.Time, p RecurringBuyPlan) string {
	at = at.UTC()
	switch p.Frequency {
	case RecurringWeekly:
		y, w := at.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", y, w)
	case RecurringMonthly:
		return at.Format("2006-01")
	case RecurringInterval:
		h := p.IntervalHours
		if h < 1 {
			h = 1
		}
		return fmt.Sprintf("i%d-%d", h, at.Unix())
	default: // daily
		return at.Format("2006-01-02")
	}
}

// AdvanceRecurringRunAt returns the next run time after `from` for the frequency.
func AdvanceRecurringRunAt(from time.Time, freq RecurringBuyFrequency) time.Time {
	return AdvanceRecurringSchedule(from, RecurringBuyPlan{Frequency: freq})
}

// AdvanceRecurringSchedule returns the next run after `from` using weekday / month day / interval.
func AdvanceRecurringSchedule(from time.Time, p RecurringBuyPlan) time.Time {
	from = from.UTC()
	switch p.Frequency {
	case RecurringWeekly:
		return from.AddDate(0, 0, 7)
	case RecurringMonthly:
		return addMonthsClamped(from, p.DayOfMonth)
	case RecurringInterval:
		h := p.IntervalHours
		if h < 1 {
			h = 1
		}
		return from.Add(time.Duration(h) * time.Hour)
	default:
		return from.AddDate(0, 0, 1)
	}
}

func daysInMonth(year int, m time.Month) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func clampMonthDay(year int, m time.Month, day int) int {
	last := daysInMonth(year, m)
	if day < 1 {
		return last
	}
	if day > last {
		return last
	}
	return day
}

func addMonthsClamped(from time.Time, dayOfMonth int) time.Time {
	from = from.UTC()
	y, m := from.Year(), from.Month()+1
	if m > 12 {
		y++
		m = 1
	}
	day := dayOfMonth
	if day <= 0 {
		day = from.Day()
	}
	day = clampMonthDay(y, m, day)
	return time.Date(y, m, day, from.Hour(), from.Minute(), from.Second(), from.Nanosecond(), time.UTC)
}

// AlignRecurringStart moves `from` to the next valid slot on or after from (weekday / salary day).
func AlignRecurringStart(from time.Time, p RecurringBuyPlan) time.Time {
	from = from.UTC()
	switch p.Frequency {
	case RecurringWeekly:
		if strings.TrimSpace(p.Weekday) == "" {
			return from
		}
		wd, err := ParseRecurringWeekday(p.Weekday)
		if err != nil {
			return from
		}
		delta := int(wd - from.Weekday())
		if delta < 0 {
			delta += 7
		}
		return from.AddDate(0, 0, delta)
	case RecurringMonthly:
		if p.DayOfMonth <= 0 {
			return from
		}
		day := clampMonthDay(from.Year(), from.Month(), p.DayOfMonth)
		cand := time.Date(from.Year(), from.Month(), day, from.Hour(), from.Minute(), from.Second(), from.Nanosecond(), time.UTC)
		if cand.Before(from) {
			return addMonthsClamped(from, p.DayOfMonth)
		}
		return cand
	default:
		return from
	}
}

// FirstRecurringRunAt is the first nextRunAt for a new or updated plan.
// Explicit startAt (including past) is aligned but not forced into the future, so tests and
// catch-up still work. When startAt is omitted, alignment snaps to the next upcoming slot.
func FirstRecurringRunAt(now time.Time, startAt *time.Time, p RecurringBuyPlan) time.Time {
	now = now.UTC()
	explicit := startAt != nil && !startAt.IsZero()
	anchor := now
	if explicit {
		anchor = startAt.UTC()
	}
	aligned := AlignRecurringStart(anchor, p)
	if !explicit && aligned.Before(now) {
		return AdvanceRecurringSchedule(aligned, p)
	}
	return aligned
}

// LatestDueRunAt returns the latest scheduled time <= now starting from nextRunAt,
// skipping intermediate missed slots without executing them.
// If nextRunAt is in the future, returns nextRunAt unchanged (not due).
func LatestDueRunAt(nextRunAt, now time.Time, freq RecurringBuyFrequency) time.Time {
	return LatestDueRunAtPlan(nextRunAt, now, RecurringBuyPlan{Frequency: freq})
}

// LatestDueRunAtPlan is LatestDueRunAt with interval / month-day aware advances.
func LatestDueRunAtPlan(nextRunAt, now time.Time, p RecurringBuyPlan) time.Time {
	nextRunAt = nextRunAt.UTC()
	now = now.UTC()
	if nextRunAt.After(now) {
		return nextRunAt
	}
	for i := 0; i < 100_000; i++ {
		n := AdvanceRecurringSchedule(nextRunAt, p)
		if !n.After(nextRunAt) {
			return nextRunAt
		}
		if n.After(now) {
			return nextRunAt
		}
		nextRunAt = n
	}
	return nextRunAt
}
