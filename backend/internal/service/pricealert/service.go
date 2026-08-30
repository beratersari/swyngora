package pricealert

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// CreateInput is the application input for creating a price or order-book alert.
type CreateInput struct {
	ClientID    string
	Exchange    string
	Symbol      string
	Condition   string // price/imbalance: above|below; wall: bid|ask|any
	TargetPrice float64
	// Mode is one_time (default for price) or repeating (default for book).
	Mode string
	// Kind is price (default), imbalance, wall, liquidation_feed, or liquidation_cascade.
	Kind string
	// RangePct is the live-book analysis band (book kinds) or window minutes
	// for liquidation_notional (1, 5, 15, 60).
	RangePct float64
	// Window is 1m|5m|15m|1h for liquidation_notional (overrides RangePct).
	Window string
}

// Service orchestrates price-alert use cases.
type Service struct {
	store domain.PriceAlertPort
	// AllowPrivateWebhooks permits loopback/RFC1918 webhook targets (local tests only).
	// Production must leave this false to prevent SSRF.
	AllowPrivateWebhooks bool
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
	kind, ok := domain.NormalizeAlertKind(in.Kind)
	if !ok {
		return nil, fmt.Errorf("%w: kind must be price, imbalance, wall, liquidation_feed, liquidation_cascade, or liquidation_notional", domain.ErrInvalidArgument)
	}
	ex, sym, err := normalizeExchangeSymbolForKind(kind, in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	cond := strings.ToLower(strings.TrimSpace(in.Condition))
	if kind == domain.AlertKindLiqNotional {
		d, _, err := domain.ParseLiqNotionalWindow(in.RangePct, in.Window)
		if err != nil {
			return nil, err
		}
		in.RangePct = d.Minutes()
	}
	if err := domain.ValidateAlertSpec(kind, cond, in.TargetPrice, in.RangePct); err != nil {
		return nil, err
	}
	mode, ok := domain.NormalizeAlertMode(in.Mode)
	if !ok {
		return nil, fmt.Errorf("%w: mode must be one_time or repeating", domain.ErrInvalidArgument)
	}
	if in.Mode == "" && (domain.IsBookAlert(kind) || domain.IsLiquidationAlert(kind)) {
		mode = domain.AlertModeRepeating
	}
	rangePct := 0.0
	if domain.IsBookAlert(kind) {
		rangePct = domain.ClampRangePct(in.RangePct)
	}
	if kind == domain.AlertKindLiqNotional {
		d, _, err := domain.ParseLiqNotionalWindow(in.RangePct, in.Window)
		if err != nil {
			return nil, err
		}
		rangePct = d.Minutes()
	}
	n, err := s.store.CountByClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxPriceAlertsPerClient {
		return nil, fmt.Errorf("%w: max %d alerts per client", domain.ErrInvalidArgument, domain.MaxPriceAlertsPerClient)
	}
	// Price repeating starts disarmed (avoid firing if already on the target side).
	// Book and liquidation repeating start armed so the first appearance can fire.
	armed := false
	if mode == domain.AlertModeRepeating && (domain.IsBookAlert(kind) || domain.IsLiquidationAlert(kind)) {
		armed = true
	}
	alert := domain.PriceAlert{
		ID:          uuid.NewString(),
		ClientID:    clientID,
		Exchange:    ex,
		Symbol:      sym,
		Kind:        kind,
		Condition:   domain.AlertCondition(cond),
		TargetPrice: in.TargetPrice,
		RangePct:    rangePct,
		Mode:        mode,
		Armed:       armed,
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
	return s.ProcessObservation(ctx, a, domain.EvaluateAlert(a, price), price, at)
}

// ProcessBook evaluates a live order-book snapshot for an imbalance/wall alert.
func (s *Service) ProcessBook(ctx context.Context, a domain.PriceAlert, an domain.OrderBookAnalysis, at time.Time) (*domain.PriceAlert, bool, error) {
	ev, metric := domain.EvaluateBookAlert(a, an)
	return s.ProcessObservation(ctx, a, ev, metric, at)
}

// ProcessObservation persists fire / re-arm from a pure eval result.
func (s *Service) ProcessObservation(ctx context.Context, a domain.PriceAlert, ev domain.AlertEvalResult, metric float64, at time.Time) (*domain.PriceAlert, bool, error) {
	return s.processObservation(ctx, a, ev, metric, at, nil)
}

// ProcessObservationExtra is ProcessObservation plus webhook payload extras
// (which exchange / coin fired a liquidation alert).
func (s *Service) ProcessObservationExtra(ctx context.Context, a domain.PriceAlert, ev domain.AlertEvalResult, metric float64, at time.Time, extra map[string]any) (*domain.PriceAlert, bool, error) {
	return s.processObservation(ctx, a, ev, metric, at, extra)
}

func (s *Service) processObservation(ctx context.Context, a domain.PriceAlert, ev domain.AlertEvalResult, metric float64, at time.Time, extra map[string]any) (*domain.PriceAlert, bool, error) {
	if s.store == nil {
		return nil, false, fmt.Errorf("%w: alert store not configured", domain.ErrUpstream)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if !ev.Fire && !ev.UpdateArmed && !ev.OneTimeDone {
		return &a, false, nil
	}

	var (
		out *domain.PriceAlert
		err error
	)
	switch {
	case ev.OneTimeDone && ev.Fire:
		out, err = s.store.MarkTriggered(ctx, a.ID, metric, at)
		if err != nil {
			return nil, false, err
		}
		if err := s.enqueueWebhookNotificationExtra(ctx, out, extra); err != nil {
			slog.Error("enqueue webhook after alert trigger", "alertId", out.ID, "clientId", out.ClientID, "err", err)
		}
		return out, true, nil
	case ev.Fire && a.Mode == domain.AlertModeRepeating:
		out, err = s.store.RecordRepeatingTrigger(ctx, a.ID, metric, at)
		if err != nil {
			return nil, false, err
		}
		if err := s.enqueueWebhookNotificationExtra(ctx, out, extra); err != nil {
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
	u, err := validateWebhookURL(in.URL, s.AllowPrivateWebhooks)
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

// NotifyClient enqueues a webhook payload on the client's registered alert
// webhook (immediate or digest). No-op when no URL is set.
func (s *Service) NotifyClient(ctx context.Context, clientID, sourceID, payloadJSON string) error {
	if s == nil || s.store == nil {
		return nil
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return err
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		sourceID = uuid.NewString()
	}
	wh, err := s.store.GetWebhook(ctx, clientID)
	if err != nil {
		return err
	}
	if wh == nil || strings.TrimSpace(wh.URL) == "" {
		return nil
	}
	now := time.Now().UTC()
	mode := wh.DeliveryMode
	if mode == "" {
		mode = domain.DeliveryImmediate
	}
	if mode == domain.DeliveryHourlyDigest {
		_, err = s.store.AddDigestItem(ctx, clientID, wh.URL, sourceID, payloadJSON, now)
		return err
	}
	nextAt := domain.NextAllowedDeliveryTime(now, wh)
	_, err = s.store.EnqueueNotification(ctx, domain.AlertNotification{
		ID: uuid.NewString(), AlertID: sourceID, ClientID: clientID,
		WebhookURL: wh.URL, PayloadJSON: payloadJSON,
		Status: domain.NotificationPending, Attempts: 0, NextAttemptAt: nextAt, CreatedAt: now,
	})
	return err
}

func (s *Service) enqueueWebhookNotification(ctx context.Context, a *domain.PriceAlert) error {
	return s.enqueueWebhookNotificationExtra(ctx, a, nil)
}

func (s *Service) enqueueWebhookNotificationExtra(ctx context.Context, a *domain.PriceAlert, extra map[string]any) error {
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
	payload, err := buildAlertWebhookPayload(a, extra)
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

func buildAlertWebhookPayload(a *domain.PriceAlert, extra map[string]any) (string, error) {
	trigAt := ""
	if a.TriggeredAt != nil {
		trigAt = a.TriggeredAt.UTC().Format(time.RFC3339Nano)
	}
	mode := string(a.Mode)
	if mode == "" {
		mode = string(domain.AlertModeOneTime)
	}
	kind := string(domain.EffectiveAlertKind(*a))
	typ := "price_alert.triggered"
	switch {
	case domain.IsBookAlert(a.Kind):
		typ = "orderbook_alert.triggered"
	case domain.IsLiqFeedAlert(a.Kind):
		typ = "liquidation_feed.triggered"
	case domain.IsLiqCascadeAlert(a.Kind):
		typ = "liquidation_cascade.triggered"
	case domain.IsLiqNotionalAlert(a.Kind):
		typ = "liquidation_notional.triggered"
	}
	body := map[string]any{
		"type":        typ,
		"alertId":     a.ID,
		"clientId":    a.ClientID,
		"exchange":    string(a.Exchange),
		"symbol":      a.Symbol,
		"kind":        kind,
		"condition":   string(a.Condition),
		"mode":        mode,
		"targetPrice": a.TargetPrice,
		"lastPrice":   a.TriggeredPrice,
		"triggeredAt": trigAt,
		"status":      string(a.Status),
		"note":        "Informational only — not financial advice.",
	}
	if domain.IsBookAlert(a.Kind) {
		body["rangePct"] = a.RangePct
		body["metric"] = a.TriggeredPrice
	}
	if domain.IsLiquidationAlert(a.Kind) {
		body["metric"] = a.TriggeredPrice
	}
	for k, v := range extra {
		if v == nil {
			continue
		}
		body[k] = v
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func normalizeClientID(id string) (string, error) {
	return domain.NormalizeClientID(id)
}

func normalizeExchangeSymbolForKind(kind domain.AlertKind, exchange, symbol string) (domain.Exchange, string, error) {
	if !domain.IsLiquidationAlert(kind) {
		return normalizeExchangeSymbol(exchange, symbol)
	}
	ex, err := domain.ParseLiquidationExchange(exchange)
	if err != nil {
		return "", "", err
	}
	if kind == domain.AlertKindLiqFeed {
		return domain.Exchange(ex), domain.LiqAlertSymbolAll, nil
	}
	if kind == domain.AlertKindLiqNotional {
		sym := domain.NormalizeLiquidationSymbol(symbol)
		if sym == "" || strings.EqualFold(sym, "all") {
			return "", "", fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
		}
		return domain.Exchange(ex), sym, nil
	}
	sym, err := domain.ParseLiquidationLevelsSymbol(symbol)
	if err != nil {
		return "", "", err
	}
	if strings.EqualFold(sym, "all") {
		return domain.Exchange(ex), domain.LiqAlertSymbolAll, nil
	}
	return domain.Exchange(ex), domain.NormalizeLiquidationSymbol(sym), nil
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
