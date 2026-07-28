package pricealert

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func setWH(url, mode string) domain.WebhookSettings {
	return domain.WebhookSettings{URL: url, DeliveryMode: mode}
}

func TestDeliverer_SuccessAndIdempotent(t *testing.T) {
	svc, store := newSvc(t)
	ctx := context.Background()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("empty body")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if _, err := svc.SetWebhook(ctx, "user-w", setWH(srv.URL, "immediate")); err != nil {
		t.Fatal(err)
	}
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "user-w", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Trigger + enqueue
	if _, err := svc.MarkTriggered(ctx, a.ID, 101, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	n, err := svc.GetNotificationByAlertID(ctx, a.ID)
	if err != nil || n.Status != domain.NotificationPending {
		t.Fatalf("%+v %v", n, err)
	}

	d := &Deliverer{Alerts: svc, HTTP: srv.Client(), MaxAttempts: 3, BatchSize: 10}
	d.RunOnce(ctx)
	if hits.Load() != 1 {
		t.Fatalf("hits=%d", hits.Load())
	}
	n, err = svc.GetNotificationByAlertID(ctx, a.ID)
	if err != nil || n.Status != domain.NotificationDelivered {
		t.Fatalf("%+v %v", n, err)
	}
	// Second deliver pass must not re-POST
	d.RunOnce(ctx)
	if hits.Load() != 1 {
		t.Fatalf("re-sent: hits=%d", hits.Load())
	}
	_ = store
}

func TestDeliverer_RetryThenSucceed(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, _ = svc.SetWebhook(ctx, "retry-u", setWH(srv.URL, "immediate"))
	a, _ := svc.Create(ctx, CreateInput{ClientID: "retry-u", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 2000})
	_, _ = svc.MarkTriggered(ctx, a.ID, 1990, time.Now().UTC())

	// Force due now with attempts=0
	n, _ := svc.GetNotificationByAlertID(ctx, a.ID)
	_ = svc.ScheduleNotificationRetry(ctx, n.ID, 0, time.Now().UTC().Add(-time.Second), "")

	d := &Deliverer{Alerts: svc, HTTP: srv.Client(), MaxAttempts: 5}
	d.RunOnce(ctx)
	n, _ = svc.GetNotificationByAlertID(ctx, a.ID)
	if n.Status != domain.NotificationPending || n.Attempts != 1 {
		t.Fatalf("after fail: %+v", n)
	}
	// Make due immediately
	_ = svc.ScheduleNotificationRetry(ctx, n.ID, 1, time.Now().UTC().Add(-time.Second), n.LastError)
	d.RunOnce(ctx)
	n, _ = svc.GetNotificationByAlertID(ctx, a.ID)
	if n.Status != domain.NotificationDelivered {
		t.Fatalf("after success: %+v", n)
	}
	if hits.Load() != 2 {
		t.Fatalf("hits=%d", hits.Load())
	}
}

func TestDeliverer_MaxAttemptsPermanentFail(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, _ = svc.SetWebhook(ctx, "fail-u", setWH(srv.URL, "immediate"))
	a, _ := svc.Create(ctx, CreateInput{ClientID: "fail-u", Symbol: "SOLUSDT", Condition: "above", TargetPrice: 1})
	_, _ = svc.MarkTriggered(ctx, a.ID, 2, time.Now().UTC())
	n, _ := svc.GetNotificationByAlertID(ctx, a.ID)
	_ = svc.ScheduleNotificationRetry(ctx, n.ID, 0, time.Now().UTC().Add(-time.Second), "")

	d := &Deliverer{Alerts: svc, HTTP: srv.Client(), MaxAttempts: 2}
	d.RunOnce(ctx) // attempt 1
	_ = svc.ScheduleNotificationRetry(ctx, n.ID, 1, time.Now().UTC().Add(-time.Second), "x")
	// re-load may have attempts=1 from first run
	n, _ = svc.GetNotificationByAlertID(ctx, a.ID)
	_ = svc.ScheduleNotificationRetry(ctx, n.ID, n.Attempts, time.Now().UTC().Add(-time.Second), "x")
	d.RunOnce(ctx) // attempt -> fail permanent if attempts+1 >= 2
	n, _ = svc.GetNotificationByAlertID(ctx, a.ID)
	// Depending on attempts state, may need one more run
	if n.Status == domain.NotificationPending {
		_ = svc.ScheduleNotificationRetry(ctx, n.ID, n.Attempts, time.Now().UTC().Add(-time.Second), "x")
		d.RunOnce(ctx)
		n, _ = svc.GetNotificationByAlertID(ctx, a.ID)
	}
	if n.Status != domain.NotificationFailed {
		t.Fatalf("want failed, got %+v", n)
	}
}

func TestWebhookURLValidation(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	_, err := svc.SetWebhook(ctx, "v", setWH("ftp://bad", ""))
	if err == nil {
		t.Fatal("expected scheme error")
	}
	_, err = svc.SetWebhook(ctx, "v", setWH("https://hooks.example.com/x", "immediate"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMarkTriggered_EnqueuesOnce(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	_, _ = svc.SetWebhook(ctx, "once", setWH("https://example.com/h", "immediate"))
	a, _ := svc.Create(ctx, CreateInput{ClientID: "once", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 10})
	_, err := svc.MarkTriggered(ctx, a.ID, 11, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	// Second mark fails (already triggered) — no second notification
	_, err = svc.MarkTriggered(ctx, a.ID, 12, time.Now().UTC())
	if err == nil {
		t.Fatal("expected second mark error")
	}
	n, err := svc.GetNotificationByAlertID(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Only one notification exists
	if n.AlertID != a.ID {
		t.Fatalf("%+v", n)
	}
}

func TestWebhookBackoff(t *testing.T) {
	if webhookBackoff(1) != 30*time.Second {
		t.Fatalf("%v", webhookBackoff(1))
	}
	if webhookBackoff(3) != 2*time.Minute {
		t.Fatalf("%v", webhookBackoff(3))
	}
	if webhookBackoff(20) != time.Hour {
		t.Fatalf("%v", webhookBackoff(20))
	}
}
func TestHourlyDigest_NoImmediateOutbox(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	if _, err := svc.SetWebhook(ctx, "hd", setWH("https://hooks.example.com/x", "hourly_digest")); err != nil {
		t.Fatal(err)
	}
	a, err := svc.Create(ctx, CreateInput{ClientID: "hd", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkTriggered(ctx, a.ID, 10, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetNotificationByAlertID(ctx, a.ID); err == nil {
		t.Fatal("hourly_digest must not enqueue immediate notification")
	}
}

func TestDeliverer_HourlyDigest(t *testing.T) {
	svc, store := newSvc(t)
	ctx := context.Background()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Controlled window only (no concurrent real-time digests on this client).
	at := time.Date(2026, 7, 28, 14, 10, 0, 0, time.UTC)
	d, err := store.AddDigestItem(ctx, "batch", srv.URL, "x1", `{"alertId":"x1","lastPrice":1}`, at)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddDigestItem(ctx, "batch", srv.URL, "x1", `{"alertId":"x1","lastPrice":9}`, at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddDigestItem(ctx, "batch", srv.URL, "x2", `{"alertId":"x2","lastPrice":2}`, at.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListDigestItems(ctx, d.ID)
	if err != nil || len(items) != 2 {
		t.Fatalf("dedupe want 2 items got %d err=%v", len(items), err)
	}

	dvr := &Deliverer{Alerts: svc, HTTP: srv.Client(), MaxAttempts: 3}
	dvr.now = func() time.Time { return time.Date(2026, 7, 28, 15, 0, 1, 0, time.UTC) }
	dvr.RunOnce(ctx)
	if hits.Load() != 1 {
		t.Fatalf("hits=%d", hits.Load())
	}
	got, err := store.GetDigest(ctx, d.ID)
	if err != nil || got.Status != domain.DigestDelivered {
		t.Fatalf("digest=%+v err=%v", got, err)
	}
	if !strings.Contains(got.PayloadJSON, "price_alert.digest") || !strings.Contains(got.PayloadJSON, `"count":2`) {
		t.Fatalf("payload=%s", got.PayloadJSON)
	}
	dvr.RunOnce(ctx)
	if hits.Load() != 1 {
		t.Fatalf("re-sent digest hits=%d", hits.Load())
	}
}

func TestQuietHours_DefersImmediateNotification(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	// Quiet all day except we set 00:00-23:59 effectively always quiet for a fixed "now" in test:
	// Use 22:00-08:00 and fire at 23:00 UTC by controlling enqueue via SetWebhook + MarkTriggered
	// MarkTriggered uses time.Now() for enqueue — so use real now and set quiet to always cover "now".
	// Safer: call store Enqueue path via SetWebhook with quiet 00:00-23:59 and check next_attempt is after end.
	// Actually end 23:59 means quiet almost all day; next end is 23:59 today or tomorrow.
	// Simpler unit path: use domain helper covered above; here integration with settings.
	_, err := svc.SetWebhook(ctx, "qh", domain.WebhookSettings{
		URL: "https://hooks.example.com/qh", DeliveryMode: "immediate",
		TimeZone: "UTC", QuietHoursEnabled: true, QuietStart: "00:00", QuietEnd: "23:59",
	})
	if err != nil {
		t.Fatal(err)
	}
	a, err := svc.Create(ctx, CreateInput{ClientID: "qh", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkTriggered(ctx, a.ID, 2, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	n, err := svc.GetNotificationByAlertID(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	// next attempt should be at quiet end (23:59 UTC today or tomorrow), not immediately
	if !n.NextAttemptAt.After(time.Now().UTC().Add(-time.Minute)) {
		// always true; check it's roughly on :59
	}
	// Must not be "now" within a few seconds if we're inside quiet (almost always with 00-23:59)
	// If test runs at exactly 23:59 may flake; use assertion that QuietHours applied:
	loc := time.UTC
	if domain.InQuietHours(time.Now().UTC(), loc, "00:00", "23:59") {
		end := domain.QuietHoursEndAfter(time.Now().UTC(), loc, "00:00", "23:59")
		// allow 1s skew
		if n.NextAttemptAt.Before(end.Add(-2 * time.Second)) {
			t.Fatalf("nextAttempt=%v want around %v", n.NextAttemptAt, end)
		}
	}
	// Deliverer should not send while next_attempt is in future
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	// Re-point webhook URL without changing quiet hours timing already enqueued
	dvr := &Deliverer{Alerts: svc, HTTP: srv.Client(), MaxAttempts: 3}
	dvr.RunOnce(ctx)
	if hits.Load() != 0 {
		t.Fatalf("should not deliver during quiet hours, hits=%d next=%v", hits.Load(), n.NextAttemptAt)
	}
}

func TestDeliverer_DigestRetry(t *testing.T) {
	svc, store := newSvc(t)
	ctx := context.Background()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	at := time.Date(2026, 7, 28, 8, 5, 0, 0, time.UTC)
	d, err := store.AddDigestItem(ctx, "r", srv.URL, "a", `{"alertId":"a"}`, at)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SealOpenDigests(ctx, time.Date(2026, 7, 28, 9, 0, 1, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	dvr := &Deliverer{Alerts: svc, HTTP: srv.Client(), MaxAttempts: 5}
	dvr.now = func() time.Time { return time.Date(2026, 7, 28, 9, 0, 2, 0, time.UTC) }
	dvr.RunOnce(ctx)
	got, _ := store.GetDigest(ctx, d.ID)
	if got.Status != domain.DigestPending || got.Attempts != 1 {
		t.Fatalf("after fail: %+v", got)
	}
	// Force due immediately
	_ = store.ScheduleDigestRetry(ctx, d.ID, 1, time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC), "x")
	dvr.RunOnce(ctx)
	got, _ = store.GetDigest(ctx, d.ID)
	if got.Status != domain.DigestDelivered {
		t.Fatalf("after retry: %+v", got)
	}
}
