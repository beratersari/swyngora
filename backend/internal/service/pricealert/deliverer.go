package pricealert

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Deliverer drains the durable notification outbox and POSTs webhooks.
// Pending rows survive restarts; each alert_id is delivered at most once (status delivered).
type Deliverer struct {
	Alerts      *Service
	HTTP        *http.Client
	Interval    time.Duration
	MaxAttempts int
	BatchSize   int
	Logger      *slog.Logger
	now         func() time.Time
	sleep       func(context.Context, time.Duration) error
}

// Start runs the delivery loop until ctx is cancelled. Blocking — call in a goroutine.
func (d *Deliverer) Start(ctx context.Context) {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.now == nil {
		d.now = time.Now
	}
	if d.sleep == nil {
		d.sleep = sleepCtx
	}
	if d.Interval <= 0 {
		d.Interval = 5 * time.Second
	}
	if d.MaxAttempts <= 0 {
		d.MaxAttempts = domain.DefaultWebhookMaxAttempts
	}
	if d.BatchSize <= 0 {
		d.BatchSize = 50
	}
	if d.HTTP == nil {
		d.HTTP = &http.Client{Timeout: 10 * time.Second}
	}
	d.RunOnce(ctx)
	for {
		if err := d.sleep(ctx, d.Interval); err != nil {
			d.Logger.Info("webhook deliverer stopped", "err", err)
			return
		}
		d.RunOnce(ctx)
	}
}

// RunOnce seals completed digest windows, then delivers due digests and immediate notifications.
func (d *Deliverer) RunOnce(ctx context.Context) {
	if d.Alerts == nil {
		return
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.now == nil {
		d.now = time.Now
	}
	if d.MaxAttempts <= 0 {
		d.MaxAttempts = domain.DefaultWebhookMaxAttempts
	}
	if d.BatchSize <= 0 {
		d.BatchSize = 50
	}
	if d.HTTP == nil {
		d.HTTP = &http.Client{Timeout: 10 * time.Second}
	}

	now := d.now().UTC()
	if n, err := d.Alerts.SealOpenDigests(ctx, now); err != nil {
		d.Logger.Error("seal open digests", "err", err)
	} else if n > 0 {
		d.Logger.Info("sealed digests", "count", n)
	}

	dueDigests, err := d.Alerts.ListDueDigests(ctx, now, d.BatchSize)
	if err != nil {
		d.Logger.Error("list due digests", "err", err)
	} else {
		for i := range dueDigests {
			d.deliverDigest(ctx, &dueDigests[i])
		}
	}

	due, err := d.Alerts.ListDueNotifications(ctx, now, d.BatchSize)
	if err != nil {
		d.Logger.Error("list due webhook notifications", "err", err)
		return
	}
	for i := range due {
		d.deliverOne(ctx, &due[i])
	}
}

func (d *Deliverer) deliverDigest(ctx context.Context, dig *domain.AlertDigest) {
	if dig == nil || dig.Status != domain.DigestPending {
		return
	}
	code, bodySnippet, err := d.postWebhook(ctx, dig.WebhookURL, dig.PayloadJSON)
	attempts := dig.Attempts + 1
	now := d.now().UTC()

	if err == nil && code >= 200 && code < 300 {
		if err := d.Alerts.MarkDigestDelivered(ctx, dig.ID, now); err != nil {
			d.Logger.Error("mark digest delivered", "id", dig.ID, "err", err)
			return
		}
		d.Logger.Info("digest webhook delivered",
			"id", dig.ID,
			"clientId", dig.ClientID,
			"status", code,
			"attempts", attempts,
		)
		return
	}

	errMsg := fmt.Sprintf("status=%d", code)
	if err != nil {
		errMsg = err.Error()
	} else if bodySnippet != "" {
		errMsg = fmt.Sprintf("status=%d body=%s", code, bodySnippet)
	}

	if attempts >= d.MaxAttempts {
		if err := d.Alerts.FailDigest(ctx, dig.ID, errMsg); err != nil {
			d.Logger.Error("fail digest", "id", dig.ID, "err", err)
		}
		d.Logger.Warn("digest permanently failed",
			"id", dig.ID, "clientId", dig.ClientID, "attempts", attempts, "err", errMsg)
		return
	}

	next := now.Add(webhookBackoff(attempts))
	if wh, werr := d.Alerts.GetWebhook(ctx, dig.ClientID); werr == nil {
		next = domain.NextAllowedDeliveryTime(next, wh)
	}
	if err := d.Alerts.ScheduleDigestRetry(ctx, dig.ID, attempts, next, errMsg); err != nil {
		d.Logger.Error("schedule digest retry", "id", dig.ID, "err", err)
		return
	}
	d.Logger.Info("digest delivery deferred",
		"id", dig.ID, "clientId", dig.ClientID, "attempts", attempts, "next", next.Format(time.RFC3339), "err", errMsg)
}

func (d *Deliverer) deliverOne(ctx context.Context, n *domain.AlertNotification) {
	if n == nil || n.Status != domain.NotificationPending {
		return
	}
	code, bodySnippet, err := d.postWebhook(ctx, n.WebhookURL, n.PayloadJSON)
	attempts := n.Attempts + 1
	now := d.now().UTC()

	if err == nil && code >= 200 && code < 300 {
		if err := d.Alerts.MarkNotificationDelivered(ctx, n.ID, now); err != nil {
			d.Logger.Error("mark webhook delivered", "id", n.ID, "alertId", n.AlertID, "err", err)
			return
		}
		d.Logger.Info("webhook delivered",
			"id", n.ID,
			"alertId", n.AlertID,
			"clientId", n.ClientID,
			"status", code,
			"attempts", attempts,
		)
		return
	}

	errMsg := fmt.Sprintf("status=%d", code)
	if err != nil {
		errMsg = err.Error()
	} else if bodySnippet != "" {
		errMsg = fmt.Sprintf("status=%d body=%s", code, bodySnippet)
	}

	if attempts >= d.MaxAttempts {
		if err := d.Alerts.FailNotification(ctx, n.ID, errMsg); err != nil {
			d.Logger.Error("fail webhook notification", "id", n.ID, "err", err)
		}
		d.Logger.Warn("webhook permanently failed",
			"id", n.ID, "alertId", n.AlertID, "attempts", attempts, "err", errMsg)
		return
	}

	next := now.Add(webhookBackoff(attempts))
	if wh, werr := d.Alerts.GetWebhook(ctx, n.ClientID); werr == nil {
		next = domain.NextAllowedDeliveryTime(next, wh)
	}
	if err := d.Alerts.ScheduleNotificationRetry(ctx, n.ID, attempts, next, errMsg); err != nil {
		d.Logger.Error("schedule webhook retry", "id", n.ID, "err", err)
		return
	}
	d.Logger.Info("webhook delivery deferred",
		"id", n.ID, "alertId", n.AlertID, "attempts", attempts, "next", next.Format(time.RFC3339), "err", errMsg)
}

func (d *Deliverer) postWebhook(ctx context.Context, webhookURL, payload string) (status int, bodySnippet string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBufferString(payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "swyngora-alerts/1.0")
	resp, err := d.HTTP.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return resp.StatusCode, string(b), nil
}

// webhookBackoff returns exponential backoff capped at 1 hour.
func webhookBackoff(attempts int) time.Duration {
	// attempts is 1-based after a failed try.
	if attempts < 1 {
		attempts = 1
	}
	// 30s * 2^(attempts-1): 30s, 60s, 2m, 4m, 8m, 16m, 32m, 64m...
	d := 30 * time.Second
	for i := 1; i < attempts; i++ {
		d *= 2
		if d >= time.Hour {
			return time.Hour
		}
	}
	return d
}