package domain

import (
	"context"
	"time"
)

// MaxPriceAlertsPerClient is the hard cap on alerts per client id.
const MaxPriceAlertsPerClient = 50

// AlertCondition is the price comparison direction.
type AlertCondition string

const (
	// AlertAbove fires when last price is greater than or equal to the target.
	AlertAbove AlertCondition = "above"
	// AlertBelow fires when last price is less than or equal to the target.
	AlertBelow AlertCondition = "below"
)

// AlertStatus is the lifecycle state of a price alert.
type AlertStatus string

const (
	AlertStatusActive    AlertStatus = "active"
	AlertStatusTriggered AlertStatus = "triggered"
)

// PriceAlert is a one-shot price threshold for a symbol on an exchange.
// Triggered alerts stay stored; they are never re-fired automatically.
type PriceAlert struct {
	ID             string
	ClientID       string
	Exchange       Exchange
	Symbol         string
	Condition      AlertCondition
	TargetPrice    float64
	Status         AlertStatus
	CreatedAt      time.Time
	TriggeredAt    *time.Time
	TriggeredPrice float64 // last price when triggered; 0 if still active
}

// PriceAlertPort persists price alerts. Implementations must be concurrent-safe.
// MarkTriggered must succeed at most once per alert (active → triggered).
type PriceAlertPort interface {
	Create(ctx context.Context, alert PriceAlert) (*PriceAlert, error)
	Get(ctx context.Context, clientID, id string) (*PriceAlert, error)
	ListByClient(ctx context.Context, clientID string) ([]PriceAlert, error)
	// ListActive returns all alerts with status active (background checker).
	ListActive(ctx context.Context) ([]PriceAlert, error)
	// MarkTriggered sets status to triggered only if still active.
	// Returns (nil, ErrNotFound) if missing or already triggered.
	MarkTriggered(ctx context.Context, id string, price float64, at time.Time) (*PriceAlert, error)
	Delete(ctx context.Context, clientID, id string) error
	// CountByClient returns how many alerts a client currently has (any status).
	CountByClient(ctx context.Context, clientID string) (int, error)
}

// AlertConditionMet reports whether lastPrice satisfies the alert threshold.
func AlertConditionMet(cond AlertCondition, lastPrice, target float64) bool {
	switch cond {
	case AlertAbove:
		return lastPrice >= target
	case AlertBelow:
		return lastPrice <= target
	default:
		return false
	}
}

// IsValidAlertCondition reports whether s is a known condition.
func IsValidAlertCondition(s string) bool {
	switch AlertCondition(s) {
	case AlertAbove, AlertBelow:
		return true
	default:
		return false
	}
}