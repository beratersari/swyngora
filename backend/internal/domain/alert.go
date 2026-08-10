package domain

import (
	"context"
	"fmt"
	"strconv"
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

// AlertKind selects what an alert watches.
type AlertKind string

const (
	// AlertKindPrice is last-price above/below (default).
	AlertKindPrice AlertKind = "price"
	// AlertKindImbalance fires when live book imbalance reaches a threshold.
	AlertKindImbalance AlertKind = "imbalance"
	// AlertKindWall fires when a large bid/ask wall appears in the live book.
	AlertKindWall AlertKind = "wall"
)

const (
	// AlertWallBid / AlertWallAsk / AlertWallAny are conditions for kind=wall.
	AlertWallBid AlertCondition = "bid"
	AlertWallAsk AlertCondition = "ask"
	AlertWallAny AlertCondition = "any"
)

const (
	MinImbalanceAlertThreshold = 0.05
	MaxImbalanceAlertThreshold = 0.95
)

// PriceAlert is a threshold for a symbol on an exchange.
// Kind=price uses last price; kind=imbalance/wall uses live order-book analysis.
// One-time alerts become status=triggered after the first fire.
// Repeating alerts stay status=active and use Armed for edge detection.
type PriceAlert struct {
	ID          string
	ClientID    string
	Exchange    Exchange
	Symbol      string
	Kind        AlertKind // price (default) | imbalance | wall
	Condition   AlertCondition
	TargetPrice float64 // price target, |imbalance| threshold, or wall min share
	RangePct    float64 // analysis band for book kinds; 0 = default 2%
	Mode        AlertMode
	// Armed is used for repeating alerts: true means ready to fire on the next
	// transition into the condition-met zone. Ignored for one_time.
	Armed          bool
	Status         AlertStatus
	CreatedAt      time.Time
	TriggeredAt    *time.Time // last fire time (one_time or repeating)
	TriggeredPrice float64    // last metric at most recent fire; 0 if never fired
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
	// TimeZone is an IANA name (e.g. Europe/Istanbul). Empty means UTC.
	TimeZone string
	// QuietHoursEnabled when true delays delivery while local time is in [QuietStart, QuietEnd).
	QuietHoursEnabled bool
	// QuietStart / QuietEnd are local "HH:MM" (24h). End may be earlier than start (crosses midnight).
	QuietStart string
	QuietEnd   string
	UpdatedAt  time.Time
}

// WebhookSettings is the write model for SetWebhook.
type WebhookSettings struct {
	URL               string
	DeliveryMode      string
	TimeZone          string
	QuietHoursEnabled bool
	QuietStart        string // "HH:MM"
	QuietEnd          string // "HH:MM"
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
	DigestPending   DigestStatus = "pending"
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

	// Webhook settings (one URL + delivery prefs per client).
	GetWebhook(ctx context.Context, clientID string) (*ClientWebhook, error)
	SetWebhook(ctx context.Context, clientID string, settings WebhookSettings) (*ClientWebhook, error)
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

	// PurgeClient deletes all alerts, webhooks, and related outbox rows for clientID.
	PurgeClient(ctx context.Context, clientID string) error
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

// ParseClockHHMM parses "HH:MM" (24h) into hour and minute.
func ParseClockHHMM(s string) (hour, min int, err error) {
	s = strings.TrimSpace(s)
	if len(s) != 5 || s[2] != ':' {
		return 0, 0, fmt.Errorf("clock must be HH:MM")
	}
	h, err1 := strconv.Atoi(s[0:2])
	m, err2 := strconv.Atoi(s[3:5])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid clock %q", s)
	}
	return h, m, nil
}

// LoadWebhookLocation loads IANA timezone; empty or invalid → UTC.
func LoadWebhookLocation(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// minutesOfDay returns minutes since local midnight for t in loc.
func minutesOfDay(t time.Time, loc *time.Location) int {
	lt := t.In(loc)
	return lt.Hour()*60 + lt.Minute()
}

// InQuietHours reports whether local time of now is inside the quiet range.
// If start == end, quiet hours are treated as disabled (never quiet).
// If start < end: quiet is [start, end) same day.
// If start > end: quiet crosses midnight — [start, 24:00) U [00:00, end).
func InQuietHours(now time.Time, loc *time.Location, startHHMM, endHHMM string) bool {
	if loc == nil {
		loc = time.UTC
	}
	sh, sm, err1 := ParseClockHHMM(startHHMM)
	eh, em, err2 := ParseClockHHMM(endHHMM)
	if err1 != nil || err2 != nil {
		return false
	}
	startM := sh*60 + sm
	endM := eh*60 + em
	if startM == endM {
		return false
	}
	cur := minutesOfDay(now, loc)
	if startM < endM {
		return cur >= startM && cur < endM
	}
	// Crosses midnight.
	return cur >= startM || cur < endM
}

// QuietHoursEndAfter returns the next UTC instant when quiet hours end after now.
// If now is not in quiet hours, returns now.UTC().
func QuietHoursEndAfter(now time.Time, loc *time.Location, startHHMM, endHHMM string) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	if !InQuietHours(now, loc, startHHMM, endHHMM) {
		return now.UTC()
	}
	eh, em, err := ParseClockHHMM(endHHMM)
	if err != nil {
		return now.UTC()
	}
	lt := now.In(loc)
	// Candidate: today at quiet end.
	endToday := time.Date(lt.Year(), lt.Month(), lt.Day(), eh, em, 0, 0, loc)
	if !endToday.After(lt) {
		// Already past today's end clock while still in quiet → end is tomorrow
		// (only possible for cross-midnight ranges when now is after midnight end?
		// Actually for 22-08: at 23:00 endToday is 08:00 today which is Before lt → add day.
		// For 22-08 at 02:00: endToday is 08:00 today which is After lt → use endToday.
		endToday = endToday.Add(24 * time.Hour)
	}
	// For cross-midnight at 23:00: endToday was 08:00 today, not after, so +1 day → 08:00 tomorrow. Correct.
	// For same-day 13-17 at 14:00: endToday 17:00 is after → correct.
	return endToday.UTC()
}

// NextAllowedDeliveryTime returns now if outside quiet hours, otherwise quiet-hours end (UTC).
func NextAllowedDeliveryTime(now time.Time, wh *ClientWebhook) time.Time {
	if wh == nil || !wh.QuietHoursEnabled {
		return now.UTC()
	}
	if strings.TrimSpace(wh.QuietStart) == "" || strings.TrimSpace(wh.QuietEnd) == "" {
		return now.UTC()
	}
	loc := LoadWebhookLocation(wh.TimeZone)
	return QuietHoursEndAfter(now, loc, wh.QuietStart, wh.QuietEnd)
}

// ValidateQuietHoursClock validates optional quiet-hours clocks when enabled.
func ValidateQuietHoursClock(enabled bool, start, end string) error {
	if !enabled {
		return nil
	}
	if _, _, err := ParseClockHHMM(start); err != nil {
		return fmt.Errorf("quietStart: %w", err)
	}
	if _, _, err := ParseClockHHMM(end); err != nil {
		return fmt.Errorf("quietEnd: %w", err)
	}
	return nil
}

// EvaluateAlert applies one price observation to an alert (pure; no I/O).
//
// One-time: fires when condition is met while still active.
// Repeating: fires only on the edge into the condition zone while Armed;
// re-arms when price is on the safe side; does not fire while staying on the trigger side.
func EvaluateAlert(a PriceAlert, lastPrice float64) AlertEvalResult {
	return EvaluateAlertState(a, AlertConditionMet(a.Condition, lastPrice, a.TargetPrice))
}

// EvaluateAlertState is the shared one_time / repeating edge machine.
// met is whether the watched condition is currently true (price, imbalance, or wall).
func EvaluateAlertState(a PriceAlert, met bool) AlertEvalResult {
	if a.Status != AlertStatusActive {
		return AlertEvalResult{}
	}
	mode := a.Mode
	if mode == "" {
		mode = AlertModeOneTime
	}
	if mode == AlertModeOneTime {
		if met {
			return AlertEvalResult{Fire: true, OneTimeDone: true}
		}
		return AlertEvalResult{}
	}
	if met {
		if a.Armed {
			return AlertEvalResult{Fire: true, NewArmed: false, UpdateArmed: true}
		}
		return AlertEvalResult{NewArmed: false, UpdateArmed: true}
	}
	return AlertEvalResult{NewArmed: true, UpdateArmed: true}
}
