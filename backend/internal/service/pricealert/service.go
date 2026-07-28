package pricealert

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const maxClientIDLen = 128

// CreateInput is the application input for creating a price alert.
type CreateInput struct {
	ClientID    string
	Exchange    string
	Symbol      string
	Condition   string // above | below
	TargetPrice float64
	// Mode is one_time (default) or repeating.
	Mode string
}

// Service orchestrates price-alert use cases.
type Service struct {
	store domain.PriceAlertPort
}

// New constructs a price-alert service.
func New(store domain.PriceAlertPort) *Service {
	return &Service{store: store}
}

// Create validates input and persists a new active alert.
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	cond := domain.AlertCondition(strings.ToLower(strings.TrimSpace(in.Condition)))
	if !domain.IsValidAlertCondition(string(cond)) {
		return nil, fmt.Errorf("%w: condition must be above or below", domain.ErrInvalidArgument)
	}
	if in.TargetPrice <= 0 || math.IsNaN(in.TargetPrice) || math.IsInf(in.TargetPrice, 0) {
		return nil, fmt.Errorf("%w: targetPrice must be a positive number", domain.ErrInvalidArgument)
	}
	mode, ok := domain.NormalizeAlertMode(in.Mode)
	if !ok {
		return nil, fmt.Errorf("%w: mode must be one_time or repeating", domain.ErrInvalidArgument)
	}
	n, err := s.store.CountByClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxPriceAlertsPerClient {
		return nil, fmt.Errorf("%w: max %d alerts per client", domain.ErrInvalidArgument, domain.MaxPriceAlertsPerClient)
	}
	// Repeating starts disarmed so we only fire on a true cross (not while already on the trigger side).
	alert := domain.PriceAlert{
		ID:          uuid.NewString(),
		ClientID:    clientID,
		Exchange:    ex,
		Symbol:      sym,
		Condition:   cond,
		TargetPrice: in.TargetPrice,
		Mode:        mode,
		Armed:       false,
		Status:      domain.AlertStatusActive,
		CreatedAt:   time.Now().UTC(),
	}
	return s.store.Create(ctx, alert)
}

// Get returns one alert for the client.
func (s *Service) Get(ctx context.Context, clientID, id string) (*domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: alert id is required", domain.ErrInvalidArgument)
	}
	return s.store.Get(ctx, clientID, id)
}

// List returns all alerts for a client.
func (s *Service) List(ctx context.Context, clientID string) ([]domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.ListByClient(ctx, clientID)
}

// Delete removes an alert owned by the client.
func (s *Service) Delete(ctx context.Context, clientID, id string) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: alert id is required", domain.ErrInvalidArgument)
	}
	return s.store.Delete(ctx, clientID, id)
}

// ListActive returns active alerts for the background checker.
func (s *Service) ListActive(ctx context.Context) ([]domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.ListActive(ctx)
}

// MarkTriggered records a one-shot trigger and enqueues a webhook notification if configured.
func (s *Service) MarkTriggered(ctx context.Context, id string, price float64, at time.Time) (*domain.PriceAlert, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: alert id is required", domain.ErrInvalidArgument)
	}
	a, err := s.store.MarkTriggered(ctx, id, price, at)
	if err != nil {
		return nil, err
	}
	if err := s.enqueueWebhookNotification(ctx, a); err != nil {
		slog.Error("enqueue webhook after alert trigger", "alertId", a.ID, "clientId", a.ClientID, "err", err)
	}
	return a, nil
}

// ProcessPrice evaluates a price tick for one alert (one_time or repeating).
// Returns the updated alert, whether a fire event occurred, and any error.
func (s *Service) ProcessPrice(ctx context.Context, a domain.PriceAlert, price float64, at time.Time) (*domain.PriceAlert, bool, error) {
	if s.store == nil {
		return nil, false, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	ev := domain.EvaluateAlert(a, price)
	if !ev.Fire && !ev.UpdateArmed && !ev.OneTimeDone {
		return &a, false, nil
	}

	var (
		out *domain.PriceAlert
		err error
	)
	switch {
	case ev.OneTimeDone && ev.Fire:
		out, err = s.store.MarkTriggered(ctx, a.ID, price, at)
		if err != nil {
			return nil, false, err
		}
		if err := s.enqueueWebhookNotification(ctx, out); err != nil {
			slog.Error("enqueue webhook after alert trigger", "alertId", out.ID, "clientId", out.ClientID, "err", err)
		}
		return out, true, nil
	case ev.Fire && a.Mode == domain.AlertModeRepeating:
		out, err = s.store.RecordRepeatingTrigger(ctx, a.ID, price, at)
		if err != nil {
			return nil, false, err
		}
		if err := s.enqueueWebhookNotification(ctx, out); err != nil {
			slog.Error("enqueue webhook after repeating trigger", "alertId", out.ID, "clientId", out.ClientID, "err", err)
		}
		return out, true, nil
	case ev.UpdateArmed:
		out, err = s.store.SetArmed(ctx, a.ID, ev.NewArmed)
		if err != nil {
			return nil, false, err
		}
		return out, false, nil
	default:
		return &a, false, nil
	}
}

// GetWebhook returns the client's webhook settings (empty URL if unset).
func (s *Service) GetWebhook(ctx context.Context, clientID string) (*domain.ClientWebhook, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.GetWebhook(ctx, clientID)
}

// SetWebhook validates and stores the client's webhook URL and notification preferences.
func (s *Service) SetWebhook(ctx context.Context, clientID string, in domain.WebhookSettings) (*domain.ClientWebhook, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	u, err := normalizeWebhookURL(in.URL)
	if err != nil {
		return nil, err
	}
	dm, ok := domain.NormalizeDeliveryMode(in.DeliveryMode)
	if !ok {
		return nil, fmt.Errorf("%w: deliveryMode must be immediate or hourly_digest", domain.ErrInvalidArgument)
	}
	tz := strings.TrimSpace(in.TimeZone)
	if tz == "" {
		tz = "UTC"
	} else if _, err := time.LoadLocation(tz); err != nil {
		return nil, fmt.Errorf("%w: timeZone must be a valid IANA name", domain.ErrInvalidArgument)
	}
	if err := domain.ValidateQuietHoursClock(in.QuietHoursEnabled, in.QuietStart, in.QuietEnd); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidArgument, err)
	}
	in.URL = u
	in.DeliveryMode = string(dm)
	in.TimeZone = tz
	return s.store.SetWebhook(ctx, clientID, in)
}

// DeleteWebhook clears the client's webhook URL.
func (s *Service) DeleteWebhook(ctx context.Context, clientID string) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return err
	}
	return s.store.DeleteWebhook(ctx, clientID)
}

// ListDueNotifications is used by the background deliverer.
func (s *Service) ListDueNotifications(ctx context.Context, now time.Time, limit int) ([]domain.AlertNotification, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.ListDueNotifications(ctx, now, limit)
}

// MarkNotificationDelivered records a successful webhook POST.
func (s *Service) MarkNotificationDelivered(ctx context.Context, id string, at time.Time) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.MarkNotificationDelivered(ctx, id, at)
}

// ScheduleNotificationRetry schedules another attempt.
func (s *Service) ScheduleNotificationRetry(ctx context.Context, id string, attempts int, nextAt time.Time, lastErr string) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.ScheduleNotificationRetry(ctx, id, attempts, nextAt, lastErr)
}

// FailNotification permanently fails a notification.
func (s *Service) FailNotification(ctx context.Context, id string, lastErr string) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.FailNotification(ctx, id, lastErr)
}

// GetNotificationByAlertID returns the outbox row for tests/ops.
func (s *Service) GetNotificationByAlertID(ctx context.Context, alertID string) (*domain.AlertNotification, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.GetNotificationByAlertID(ctx, alertID)
}

// SealOpenDigests seals completed hour windows (used by deliverer / tests).
func (s *Service) SealOpenDigests(ctx context.Context, now time.Time) (int, error) {
	if s.store == nil {
		return 0, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.SealOpenDigests(ctx, now)
}

// ListDueDigests returns sealed digests ready to send.
func (s *Service) ListDueDigests(ctx context.Context, now time.Time, limit int) ([]domain.AlertDigest, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.ListDueDigests(ctx, now, limit)
}

// GetDigest returns one digest by id.
func (s *Service) GetDigest(ctx context.Context, id string) (*domain.AlertDigest, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.GetDigest(ctx, id)
}

// ListDigestItems returns items in a digest.
func (s *Service) ListDigestItems(ctx context.Context, digestID string) ([]domain.AlertDigestItem, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.ListDigestItems(ctx, digestID)
}

// MarkDigestDelivered marks a digest delivered.
func (s *Service) MarkDigestDelivered(ctx context.Context, id string, at time.Time) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.MarkDigestDelivered(ctx, id, at)
}

// ScheduleDigestRetry schedules another digest delivery attempt.
func (s *Service) ScheduleDigestRetry(ctx context.Context, id string, attempts int, nextAt time.Time, lastErr string) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.ScheduleDigestRetry(ctx, id, attempts, nextAt, lastErr)
}

// FailDigest permanently fails a digest.
func (s *Service) FailDigest(ctx context.Context, id string, lastErr string) error {
	if s.store == nil {
		return fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	return s.store.FailDigest(ctx, id, lastErr)
}

func (s *Service) enqueueWebhookNotification(ctx context.Context, a *domain.PriceAlert) error {
	if a == nil {
		return nil
	}
	wh, err := s.store.GetWebhook(ctx, a.ClientID)
	if err != nil {
		return err
	}
	if wh == nil || strings.TrimSpace(wh.URL) == "" {
		return nil
	}
	payload, err := buildAlertWebhookPayload(a)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	mode := wh.DeliveryMode
	if mode == "" {
		mode = domain.DeliveryImmediate
	}
	if mode == domain.DeliveryHourlyDigest {
		// Items still collect during quiet hours; delivery waits until seal + quiet end.
		_, err = s.store.AddDigestItem(ctx, a.ClientID, wh.URL, a.ID, payload, now)
		return err
	}
	// Immediate: schedule first attempt after quiet hours if needed.
	nextAt := domain.NextAllowedDeliveryTime(now, wh)
	_, err = s.store.EnqueueNotification(ctx, domain.AlertNotification{
		ID:            uuid.NewString(),
		AlertID:       a.ID,
		ClientID:      a.ClientID,
		WebhookURL:    wh.URL,
		PayloadJSON:   payload,
		Status:        domain.NotificationPending,
		Attempts:      0,
		NextAttemptAt: nextAt,
		CreatedAt:     now,
	})
	return err
}

func buildAlertWebhookPayload(a *domain.PriceAlert) (string, error) {
	trigAt := ""
	if a.TriggeredAt != nil {
		trigAt = a.TriggeredAt.UTC().Format(time.RFC3339Nano)
	}
	mode := string(a.Mode)
	if mode == "" {
		mode = string(domain.AlertModeOneTime)
	}
	body := map[string]any{
		"type":        "price_alert.triggered",
		"alertId":     a.ID,
		"clientId":    a.ClientID,
		"exchange":    string(a.Exchange),
		"symbol":      a.Symbol,
		"condition":   string(a.Condition),
		"mode":        mode,
		"targetPrice": a.TargetPrice,
		"lastPrice":   a.TriggeredPrice,
		"triggeredAt": trigAt,
		"status":      string(a.Status),
		"note":        "Informational only — not financial advice.",
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func normalizeWebhookURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: webhook URL is required", domain.ErrInvalidArgument)
	}
	if len(raw) > domain.MaxWebhookURLLen {
		return "", fmt.Errorf("%w: webhook URL too long", domain.ErrInvalidArgument)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%w: webhook URL must be an absolute http(s) URL", domain.ErrInvalidArgument)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("%w: webhook URL scheme must be http or https", domain.ErrInvalidArgument)
	}
	// Reconstruct without fragment.
	u.Fragment = ""
	return u.String(), nil
}

func normalizeClientID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument)
	}
	if len(id) > maxClientIDLen {
		return "", fmt.Errorf("%w: clientId too long", domain.ErrInvalidArgument)
	}
	if strings.EqualFold(id, "default") {
		return "", fmt.Errorf("%w: clientId must not be the shared name \"default\"", domain.ErrInvalidArgument)
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("%w: clientId has invalid characters", domain.ErrInvalidArgument)
	}
	return id, nil
}

func normalizeExchangeSymbol(exchange, symbol string) (domain.Exchange, string, error) {
	rawEx := strings.TrimSpace(exchange)
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return "", "", fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
		ex = domain.ParseExchange(rawEx)
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return "", "", fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return ex, sym, nil
}