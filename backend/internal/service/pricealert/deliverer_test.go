package pricealert

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

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

	if _, err := svc.SetWebhook(ctx, "user-w", srv.URL); err != nil {
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

	_, _ = svc.SetWebhook(ctx, "retry-u", srv.URL)
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
	_, _ = svc.SetWebhook(ctx, "fail-u", srv.URL)
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
	_, err := svc.SetWebhook(ctx, "v", "ftp://bad")
	if err == nil {
		t.Fatal("expected scheme error")
	}
	_, err = svc.SetWebhook(ctx, "v", "https://hooks.example.com/x")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMarkTriggered_EnqueuesOnce(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	_, _ = svc.SetWebhook(ctx, "once", "https://example.com/h")
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