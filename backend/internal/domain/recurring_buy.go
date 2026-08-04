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
)

// RecurringBuyFrequency is how often a plan executes.
type RecurringBuyFrequency string

const (
	RecurringDaily   RecurringBuyFrequency = "daily"
	RecurringWeekly  RecurringBuyFrequency = "weekly"
	RecurringMonthly RecurringBuyFrequency = "monthly"
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
// Amount is cash (portfolio currency) spent each run at last market price.
type RecurringBuyPlan struct {
	ID            string
	ClientID      string
	Exchange      Exchange
	Symbol        string
	Amount        float64 // cash notional per run
	Frequency     RecurringBuyFrequency
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

// IsValidRecurringBuyFrequency reports daily|weekly|monthly.
func IsValidRecurringBuyFrequency(s string) bool {
	switch RecurringBuyFrequency(strings.ToLower(strings.TrimSpace(s))) {
	case RecurringDaily, RecurringWeekly, RecurringMonthly:
		return true
	default:
		return false
	}
}

// NormalizeRecurringBuyFrequency parses frequency.
func NormalizeRecurringBuyFrequency(s string) (RecurringBuyFrequency, error) {
	f := RecurringBuyFrequency(strings.ToLower(strings.TrimSpace(s)))
	if !IsValidRecurringBuyFrequency(string(f)) {
		return "", fmt.Errorf("%w: frequency must be daily, weekly, or monthly", ErrInvalidArgument)
	}
	return f, nil
}

// RecurringPeriodKey identifies a calendar period for idempotent execution.
func RecurringPeriodKey(at time.Time, freq RecurringBuyFrequency) string {
	at = at.UTC()
	switch freq {
	case RecurringWeekly:
		y, w := at.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", y, w)
	case RecurringMonthly:
		return at.Format("2006-01")
	default: // daily
		return at.Format("2006-01-02")
	}
}

// AdvanceRecurringRunAt returns the next run time after `from` for the frequency.
func AdvanceRecurringRunAt(from time.Time, freq RecurringBuyFrequency) time.Time {
	from = from.UTC()
	switch freq {
	case RecurringWeekly:
		return from.AddDate(0, 0, 7)
	case RecurringMonthly:
		return from.AddDate(0, 1, 0)
	default:
		return from.AddDate(0, 0, 1)
	}
}

// LatestDueRunAt returns the latest scheduled time <= now starting from nextRunAt,
// skipping intermediate missed slots without executing them.
// If nextRunAt is in the future, returns nextRunAt unchanged (not due).
func LatestDueRunAt(nextRunAt, now time.Time, freq RecurringBuyFrequency) time.Time {
	nextRunAt = nextRunAt.UTC()
	now = now.UTC()
	if nextRunAt.After(now) {
		return nextRunAt
	}
	// Advance while the following slot is still <= now.
	for {
		n := AdvanceRecurringRunAt(nextRunAt, freq)
		if n.After(now) {
			return nextRunAt
		}
		nextRunAt = n
	}
}
