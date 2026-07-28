package domain

import (
	"context"
	"strings"
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
	AlertStatusTriggered AlertStatus = "triggered" // terminal for one_time only
)

// AlertMode selects one-shot vs repeating edge-cross behavior.
type AlertMode string

const (
	// AlertModeOneTime fires once then stays triggered (default).
	AlertModeOneTime AlertMode = "one_time"
	// AlertModeRepeating fires on each cross into the condition zone; stays active.
	// While price remains on the trigger side it does not re-fire; re-arms on the safe side.
	AlertModeRepeating AlertMode = "repeating"
)

// PriceAlert is a price threshold for a symbol on an exchange.
// One-time alerts become status=triggered after the first fire.
// Repeating alerts stay status=active and use Armed for edge detection.
type PriceAlert struct {
	ID             string
	ClientID       string
	Exchange       Exchange
	Symbol         string
	Condition      AlertCondition
	TargetPrice    float64
	Mode           AlertMode
	// Armed is used for repeating alerts: true means ready to fire on the next
	// transition into the condition-met zone. Ignored for one_time.
	Armed          bool
	Status         AlertStatus
	CreatedAt      time.Time
	TriggeredAt    *time.Time // last fire time (one_time or repeating)
	TriggeredPrice float64    // last price at most recent fire; 0 if never fired
}

// AlertEvalResult is the outcome of evaluating a price tick against an alert.
type AlertEvalResult struct {
	// Fire is true when a notification / trigger event should be emitted.
	Fire bool
	// OneTimeDone is true when a one_time alert should become status=triggered.
	OneTimeDone bool
	// NewArmed is the armed flag to persist for repeating alerts (and one_time unused).
	NewArmed bool
	// UpdateArmed is true when NewArmed should be written (repeating only).
	UpdateArmed bool
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

// NotificationDeliveryMode controls when webhook notifications are sent.
type NotificationDeliveryMode string

const (
	// DeliveryImmediate POSTs each fire as soon as it is enqueued (default).
	DeliveryImmediate NotificationDeliveryMode = "immediate"
	// DeliveryHourlyDigest batches fires into one webhook POST per UTC hour.
	DeliveryHourlyDigest NotificationDeliveryMode = "hourly_digest"
)

// ClientWebhook is a client's registered webhook destination for price alerts.
type ClientWebhook struct {
	ClientID     string
	URL          string
	DeliveryMode NotificationDeliveryMode
	UpdatedAt    time.Time
}

// AlertNotification is a durable outbox row for one immediate alert fire delivery.
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

// DigestStatus is the lifecycle of an hourly digest batch.
type DigestStatus string

const (
	// DigestOpen collects items until the hour window ends.
	DigestOpen DigestStatus = "open"
	// DigestPending is sealed and waiting for (re)delivery.
	DigestPending DigestStatus = "pending"
	DigestDelivered DigestStatus = "delivered"
	DigestFailed    DigestStatus = "failed"
)

// AlertDigest is a durable hourly batch of alert fires for one client.
type AlertDigest struct {
	ID            string
	ClientID      string
	WebhookURL    string
	WindowStart   time.Time
	WindowEnd     time.Time
	Status        DigestStatus
	Attempts      int
	NextAttemptAt time.Time
	LastError     string
	PayloadJSON   string // set when sealed
	CreatedAt     time.Time
	SealedAt      *time.Time
	DeliveredAt   *time.Time
	Items         []AlertDigestItem // optional, loaded when needed
}

// AlertDigestItem is one alert's contribution to a digest (unique per digest).
type AlertDigestItem struct {
	DigestID    string
	AlertID     string
	PayloadJSON string
	CreatedAt   time.Time
}

// PriceAlertPort persists price alerts, webhooks, and the notification outbox.
// Implementations must be concurrent-safe.
// MarkTriggered must succeed at most once per one_time alert (active → triggered).
// RecordRepeatingTrigger keeps status=active and disarms until price returns to the safe side.
type PriceAlertPort interface {
	Create(ctx context.Context, alert PriceAlert) (*PriceAlert, error)
	Get(ctx context.Context, clientID, id string) (*PriceAlert, error)
	ListByClient(ctx context.Context, clientID string) ([]PriceAlert, error)
	// ListActive returns all alerts with status active (background checker).
	ListActive(ctx context.Context) ([]PriceAlert, error)
	// MarkTriggered sets status to triggered only if still active (one_time).
	// Returns (nil, ErrNotFound) if missing or already triggered.
	MarkTriggered(ctx context.Context, id string, price float64, at time.Time) (*PriceAlert, error)
	// RecordRepeatingTrigger records a fire for a repeating alert (status stays active, armed=false).
	RecordRepeatingTrigger(ctx context.Context, id string, price float64, at time.Time) (*PriceAlert, error)
	// SetArmed updates the armed flag for a repeating (or any) active alert.
	SetArmed(ctx context.Context, id string, armed bool) (*PriceAlert, error)
	Delete(ctx context.Context, clientID, id string) error
	// CountByClient returns how many alerts a client currently has (any status).
	CountByClient(ctx context.Context, clientID string) (int, error)

	// Webhook settings (one URL + delivery mode per client).
	GetWebhook(ctx context.Context, clientID string) (*ClientWebhook, error)
	SetWebhook(ctx context.Context, clientID, url, deliveryMode string) (*ClientWebhook, error)
	DeleteWebhook(ctx context.Context, clientID string) error

	// Immediate notification outbox (durable queue). Each fire enqueues a new row.
	EnqueueNotification(ctx context.Context, n AlertNotification) (*AlertNotification, error)
	// ListDueNotifications returns pending rows with next_attempt_at <= now.
	ListDueNotifications(ctx context.Context, now time.Time, limit int) ([]AlertNotification, error)
	MarkNotificationDelivered(ctx context.Context, id string, at time.Time) error
	// ScheduleNotificationRetry keeps status pending and bumps attempts / next_attempt_at.
	ScheduleNotificationRetry(ctx context.Context, id string, attempts int, nextAt time.Time, lastErr string) error
	// FailNotification marks permanent failure after max attempts.
	FailNotification(ctx context.Context, id string, lastErr string) error
	// GetNotificationByAlertID returns the latest outbox row for an alert, or ErrNotFound.
	GetNotificationByAlertID(ctx context.Context, alertID string) (*AlertNotification, error)

	// Hourly digests: AddDigestItem upserts by (digest, alert_id) so an alert appears once.
	AddDigestItem(ctx context.Context, clientID, webhookURL, alertID, itemPayloadJSON string, at time.Time) (*AlertDigest, error)
	// SealOpenDigests seals open digests whose window_end <= now into pending with payload.
	SealOpenDigests(ctx context.Context, now time.Time) (int, error)
	ListDueDigests(ctx context.Context, now time.Time, limit int) ([]AlertDigest, error)
	GetDigest(ctx context.Context, id string) (*AlertDigest, error)
	ListDigestItems(ctx context.Context, digestID string) ([]AlertDigestItem, error)
	MarkDigestDelivered(ctx context.Context, id string, at time.Time) error
	ScheduleDigestRetry(ctx context.Context, id string, attempts int, nextAt time.Time, lastErr string) error
	FailDigest(ctx context.Context, id string, lastErr string) error
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

// IsValidAlertMode reports whether s is a known mode (empty is not valid here;
// callers may default empty to one_time before checking).
func IsValidAlertMode(s string) bool {
	switch AlertMode(s) {
	case AlertModeOneTime, AlertModeRepeating:
		return true
	default:
		return false
	}
}

// NormalizeAlertMode returns one_time for empty input; validates otherwise.
func NormalizeAlertMode(s string) (AlertMode, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return AlertModeOneTime, true
	}
	if !IsValidAlertMode(s) {
		return "", false
	}
	return AlertMode(s), true
}

// IsValidDeliveryMode reports whether s is immediate or hourly_digest.
func IsValidDeliveryMode(s string) bool {
	switch NotificationDeliveryMode(s) {
	case DeliveryImmediate, DeliveryHourlyDigest:
		return true
	default:
		return false
	}
}

// NormalizeDeliveryMode returns immediate for empty input; validates otherwise.
func NormalizeDeliveryMode(s string) (NotificationDeliveryMode, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return DeliveryImmediate, true
	}
	if !IsValidDeliveryMode(s) {
		return "", false
	}
	return NotificationDeliveryMode(s), true
}

// DigestHourWindow returns the UTC hour bucket [start, end) containing t.
func DigestHourWindow(t time.Time) (start, end time.Time) {
	t = t.UTC()
	start = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
	end = start.Add(time.Hour)
	return start, end
}

// EvaluateAlert applies one price observation to an alert (pure; no I/O).
//
// One-time: fires when condition is met while still active.
// Repeating: fires only on the edge into the condition zone while Armed;
// re-arms when price is on the safe side; does not fire while staying on the trigger side.
func EvaluateAlert(a PriceAlert, lastPrice float64) AlertEvalResult {
	if a.Status != AlertStatusActive {
		return AlertEvalResult{}
	}
	mode := a.Mode
	if mode == "" {
		mode = AlertModeOneTime
	}
	met := AlertConditionMet(a.Condition, lastPrice, a.TargetPrice)

	if mode == AlertModeOneTime {
		if met {
			return AlertEvalResult{Fire: true, OneTimeDone: true}
		}
		return AlertEvalResult{}
	}

	// Repeating edge detection.
	if met {
		if a.Armed {
			return AlertEvalResult{Fire: true, NewArmed: false, UpdateArmed: true}
		}
		// Already on trigger side and disarmed — stay quiet.
		return AlertEvalResult{NewArmed: false, UpdateArmed: true}
	}
	// Safe side — re-arm for the next cross.
	return AlertEvalResult{NewArmed: true, UpdateArmed: true}
}