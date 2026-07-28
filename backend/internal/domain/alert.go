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

// MaxWebhookURLLen bounds stored webhook URLs.
const MaxWebhookURLLen = 2048

// DefaultWebhookMaxAttempts is the default permanent-failure threshold for deliveries.
const DefaultWebhookMaxAttempts = 8

// NotificationStatus is the delivery lifecycle of a webhook notification.
type NotificationStatus string

const (
	NotificationPending   NotificationStatus = "pending"
	NotificationDelivered NotificationStatus = "delivered"
	NotificationFailed    NotificationStatus = "failed" // exhausted retries
)

// ClientWebhook is a client's registered webhook destination for price alerts.
type ClientWebhook struct {
	ClientID  string
	URL       string
	UpdatedAt time.Time
}

// AlertNotification is a durable outbox row for one alert trigger delivery.
// At most one row exists per AlertID (enforced by store unique constraint).
type AlertNotification struct {
	ID            string
	AlertID       string
	ClientID      string
	WebhookURL    string
	PayloadJSON   string
	Status        NotificationStatus
	Attempts      int
	NextAttemptAt time.Time
	LastError     string
	CreatedAt     time.Time
	DeliveredAt   *time.Time
}

// PriceAlertPort persists price alerts, webhooks, and the notification outbox.
// Implementations must be concurrent-safe.
// MarkTriggered must succeed at most once per alert (active → triggered).
// EnqueueNotification must insert at most one row per AlertID.
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

	// Webhook settings (one URL per client).
	GetWebhook(ctx context.Context, clientID string) (*ClientWebhook, error)
	SetWebhook(ctx context.Context, clientID, url string) (*ClientWebhook, error)
	DeleteWebhook(ctx context.Context, clientID string) error

	// Notification outbox (durable queue).
	// EnqueueNotification inserts if alert_id is new; if a row already exists, returns it without error.
	EnqueueNotification(ctx context.Context, n AlertNotification) (*AlertNotification, error)
	// ListDueNotifications returns pending rows with next_attempt_at <= now.
	ListDueNotifications(ctx context.Context, now time.Time, limit int) ([]AlertNotification, error)
	MarkNotificationDelivered(ctx context.Context, id string, at time.Time) error
	// ScheduleNotificationRetry keeps status pending and bumps attempts / next_attempt_at.
	ScheduleNotificationRetry(ctx context.Context, id string, attempts int, nextAt time.Time, lastErr string) error
	// FailNotification marks permanent failure after max attempts.
	FailNotification(ctx context.Context, id string, lastErr string) error
	// GetNotificationByAlertID returns the outbox row for an alert, or ErrNotFound.
	GetNotificationByAlertID(ctx context.Context, alertID string) (*AlertNotification, error)
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