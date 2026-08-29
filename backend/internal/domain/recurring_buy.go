package domain

import (
	"fmt"
	"math"
	"strings"
	"time"

	_ "time/tzdata"
)

// Recurring buy plan limits (paper trading only).
const (
	MaxRecurringBuyPlansPerClient = 20
	MinRecurringBuyAmount         = 1.0
	MaxRecurringBuyAmount         = 1_000_000.0
	MaxRecurringBuyNameLen        = 80
	MinRecurringIntervalHours     = 1
	MaxRecurringIntervalHours     = 168 // 7 days
	MaxRecurringBuyMaxPrice       = 1e12
	// RecurringHourUnset means keep the clock from startAt / previous nextRunAt.
	RecurringHourUnset = -1
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
	Name          string  // user label e.g. "Salary Day Buy"
	Amount        float64 // cash notional per run
	Frequency     RecurringBuyFrequency
	Weekday       string  // monday..sunday; weekly only (optional)
	DayOfMonth    int     // 1-31; monthly only (optional; 0 = anniversary of start)
	IntervalHours int     // 1-168; interval frequency only
	TimeZone      string  // IANA e.g. Europe/Istanbul; empty = UTC
	HasLocalTime  bool    // when true, Hour:Minute is the local clock
	Hour          int     // 0-23 local when HasLocalTime
	Minute        int     // 0-59 local
	MaxPrice      float64 // 0 = no cap; last, slipped fill, and fee-inclusive unit must all stay <= this
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

// NormalizeRecurringTimeZone accepts empty (UTC) or an IANA name.
func NormalizeRecurringTimeZone(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "utc") || s == "Z" {
		return "", nil
	}
	if _, err := time.LoadLocation(s); err != nil {
		return "", fmt.Errorf("%w: timeZone must be an IANA name e.g. Europe/Istanbul", ErrInvalidArgument)
	}
	return s, nil
}

// RecurringLocation is UTC when TimeZone is empty.
func RecurringLocation(tz string) *time.Location {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

// NormalizeRecurringClock validates hour/minute. hour RecurringHourUnset means inherit.
func NormalizeRecurringClock(hour, minute int) (int, int, error) {
	if hour == RecurringHourUnset {
		if minute < 0 || minute > 59 {
			return RecurringHourUnset, 0, fmt.Errorf("%w: minute must be 0-59", ErrInvalidArgument)
		}
		return RecurringHourUnset, 0, nil
	}
	if hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("%w: hour must be 0-23", ErrInvalidArgument)
	}
	if minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("%w: minute must be 0-59", ErrInvalidArgument)
	}
	return hour, minute, nil
}

// ResolveRecurringMaxPrice accepts 0 (no cap) or a positive finite price.
func ResolveRecurringMaxPrice(v float64) (float64, error) {
	if v == 0 {
		return 0, nil
	}
	if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) || v > MaxRecurringBuyMaxPrice {
		return 0, fmt.Errorf("%w: maxPrice must be between 0 and %g", ErrInvalidArgument, MaxRecurringBuyMaxPrice)
	}
	return v, nil
}

// RecurringEffectivePrice is slipped fill including the taker fee (quote per base).
func RecurringEffectivePrice(last, slipRate, feeRate float64) (fill, unit float64) {
	fill = ApplySlippage(last, TradeSideBuy, slipRate)
	unit = BuyUnitCost(fill, feeRate)
	return fill, unit
}

// RecurringMaxPriceBlocks reports why a buy should be skipped. Empty means ok.
func RecurringMaxPriceBlocks(last, slipRate, feeRate, maxPrice float64) string {
	if maxPrice <= 0 || last <= 0 {
		return ""
	}
	const eps = 1e-9
	if last > maxPrice+eps {
		return "last price above maxPrice"
	}
	fill, unit := RecurringEffectivePrice(last, slipRate, feeRate)
	if fill > maxPrice+eps || unit > maxPrice+eps {
		return "fill would exceed maxPrice after fee and slippage"
	}
	return ""
}

// usesWallClock is true when the plan pins a local time and/or timezone.
func (p RecurringBuyPlan) usesWallClock() bool {
	return p.HasLocalTime || strings.TrimSpace(p.TimeZone) != ""
}

func (p RecurringBuyPlan) location() *time.Location {
	return RecurringLocation(p.TimeZone)
}

func applyRecurringClock(t time.Time, p RecurringBuyPlan) time.Time {
	loc := p.location()
	lt := t.In(loc)
	h, m, sec, nsec := lt.Hour(), lt.Minute(), lt.Second(), lt.Nanosecond()
	if p.HasLocalTime || strings.TrimSpace(p.TimeZone) != "" {
		h, m, sec, nsec = p.Hour, p.Minute, 0, 0
	}
	return time.Date(lt.Year(), lt.Month(), lt.Day(), h, m, sec, nsec, loc)
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
	at = at.In(p.location())
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
		return fmt.Sprintf("i%d-%d", h, at.UTC().Unix())
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
	if p.Frequency == RecurringInterval || !p.usesWallClock() {
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
	loc := p.location()
	lt := from.In(loc)
	var next time.Time
	switch p.Frequency {
	case RecurringWeekly:
		next = lt.AddDate(0, 0, 7)
	case RecurringMonthly:
		next = addMonthsClampedIn(lt, p.DayOfMonth, loc)
	default:
		next = lt.AddDate(0, 0, 1)
	}
	return applyRecurringClock(next, p).UTC()
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
	return addMonthsClampedIn(from.UTC(), dayOfMonth, time.UTC)
}

func addMonthsClampedIn(from time.Time, dayOfMonth int, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	from = from.In(loc)
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
	return time.Date(y, m, day, from.Hour(), from.Minute(), from.Second(), from.Nanosecond(), loc)
}

// AlignRecurringStart moves `from` to the next valid slot on or after from (weekday / salary day).
func AlignRecurringStart(from time.Time, p RecurringBuyPlan) time.Time {
	if p.Frequency == RecurringInterval {
		return from.UTC()
	}
	loc := p.location()
	from = from.In(loc)
	base := applyRecurringClock(from, p)
	switch p.Frequency {
	case RecurringWeekly:
		if strings.TrimSpace(p.Weekday) == "" {
			if p.usesWallClock() {
				if base.Before(from) {
					return applyRecurringClock(from.AddDate(0, 0, 1), p).UTC()
				}
				return base.UTC()
			}
			return from.UTC()
		}
		wd, err := ParseRecurringWeekday(p.Weekday)
		if err != nil {
			return from.UTC()
		}
		delta := int(wd - base.Weekday())
		if delta < 0 {
			delta += 7
		}
		cand := applyRecurringClock(base.AddDate(0, 0, delta), p)
		if cand.Before(from) {
			cand = applyRecurringClock(cand.AddDate(0, 0, 7), p)
		}
		return cand.UTC()
	case RecurringMonthly:
		if p.DayOfMonth <= 0 && !p.usesWallClock() {
			return from.UTC()
		}
		day := p.DayOfMonth
		if day <= 0 {
			day = from.Day()
		}
		day = clampMonthDay(from.Year(), from.Month(), day)
		cand := applyRecurringClock(time.Date(from.Year(), from.Month(), day, 0, 0, 0, 0, loc), p)
		if cand.Before(from) {
			return applyRecurringClock(addMonthsClampedIn(from, p.DayOfMonth, loc), p).UTC()
		}
		return cand.UTC()
	default: // daily
		if !p.usesWallClock() {
			return from.UTC()
		}
		if base.Before(from) {
			return applyRecurringClock(from.AddDate(0, 0, 1), p).UTC()
		}
		return base.UTC()
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
