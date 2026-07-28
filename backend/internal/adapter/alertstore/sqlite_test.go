package alertstore

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func openTemp(t *testing.T) *SQLite {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "alerts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func sample(id, client string) domain.PriceAlert {
	return domain.PriceAlert{
		ID:          id,
		ClientID:    client,
		Exchange:    domain.ExchangeBinance,
		Symbol:      "BTCUSDT",
		Condition:   domain.AlertAbove,
		TargetPrice: 100_000,
		Status:      domain.AlertStatusActive,
		CreatedAt:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestSQLite_CRUD(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	a, err := s.Create(ctx, sample("a1", "c1"))
	if err != nil || a.ID != "a1" {
		t.Fatalf("%+v %v", a, err)
	}
	got, err := s.Get(ctx, "c1", "a1")
	if err != nil || got.TargetPrice != 100_000 {
		t.Fatalf("%+v %v", got, err)
	}
	list, err := s.ListByClient(ctx, "c1")
	if err != nil || len(list) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	if err := s.Delete(ctx, "c1", "a1"); err != nil {
		t.Fatal(err)
	}
	_, err = s.Get(ctx, "c1", "a1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestSQLite_PersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alerts.db")
	ctx := context.Background()

	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Create(ctx, sample("persist-1", "web-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Create(ctx, domain.PriceAlert{
		ID: "persist-2", ClientID: "web-1", Exchange: domain.ExchangeBybit,
		Symbol: "ETHUSDT", Condition: domain.AlertBelow, TargetPrice: 2000,
		Status: domain.AlertStatusActive, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	list, err := s2.ListByClient(ctx, "web-1")
	if err != nil || len(list) != 2 {
		t.Fatalf("after reopen len=%d err=%v", len(list), err)
	}
	active, err := s2.ListActive(ctx)
	if err != nil || len(active) != 2 {
		t.Fatalf("active after reopen len=%d err=%v", len(active), err)
	}
}

func TestSQLite_MarkTriggeredOnce(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	if _, err := s.Create(ctx, sample("t1", "c")); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	got, err := s.MarkTriggered(ctx, "t1", 101_000, at)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.AlertStatusTriggered || got.TriggeredPrice != 101_000 {
		t.Fatalf("%+v", got)
	}
	if got.TriggeredAt == nil || !got.TriggeredAt.Equal(at) {
		t.Fatalf("triggeredAt=%v", got.TriggeredAt)
	}
	// Second mark must fail (already triggered).
	_, err = s.MarkTriggered(ctx, "t1", 102_000, time.Now().UTC())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second mark: %v", err)
	}
	// Still only one row, still triggered once.
	list, _ := s.ListByClient(ctx, "c")
	if len(list) != 1 || list[0].Status != domain.AlertStatusTriggered || list[0].TriggeredPrice != 101_000 {
		t.Fatalf("%+v", list)
	}
	active, _ := s.ListActive(ctx)
	if len(active) != 0 {
		t.Fatalf("active should be empty: %+v", active)
	}
}

func TestSQLite_CountAndIsolation(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	_, _ = s.Create(ctx, sample("x1", "a"))
	_, _ = s.Create(ctx, sample("x2", "b"))
	n, err := s.CountByClient(ctx, "a")
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if err := s.Delete(ctx, "b", "x1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-client delete: %v", err)
	}
}
func TestSQLite_WebhookAndNotificationOutbox(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	wh, err := s.SetWebhook(ctx, "c1", "https://hooks.example.com/swyngora", "immediate")
	if err != nil || wh.URL == "" {
		t.Fatalf("%+v %v", wh, err)
	}
	got, err := s.GetWebhook(ctx, "c1")
	if err != nil || got.URL != "https://hooks.example.com/swyngora" {
		t.Fatalf("%+v %v", got, err)
	}
	// Enqueue once per alert
	n1, err := s.EnqueueNotification(ctx, domain.AlertNotification{
		ID: "n1", AlertID: "alert-1", ClientID: "c1",
		WebhookURL: got.URL, PayloadJSON: `{"ok":true}`,
		Status: domain.NotificationPending, NextAttemptAt: time.Now().UTC().Add(-time.Second),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Second fire for same alert is a new outbox row (repeating support).
	n2, err := s.EnqueueNotification(ctx, domain.AlertNotification{
		ID: "n2", AlertID: "alert-1", ClientID: "c1",
		WebhookURL: got.URL, PayloadJSON: `{"again":true}`,
		Status: domain.NotificationPending, NextAttemptAt: time.Now().UTC().Add(-time.Second),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if n2.ID == n1.ID {
		t.Fatalf("expected distinct notification rows per fire")
	}
	due, err := s.ListDueNotifications(ctx, time.Now().UTC(), 10)
	if err != nil || len(due) != 2 {
		t.Fatalf("due=%+v err=%v", due, err)
	}
	if err := s.MarkNotificationDelivered(ctx, n1.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	due, err = s.ListDueNotifications(ctx, time.Now().UTC(), 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("after one deliver due=%+v", due)
	}
}

func TestSQLite_NotificationRetryAndPersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alerts.db")
	ctx := context.Background()
	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s1.SetWebhook(ctx, "u", "https://example.com/hook", "immediate")
	past := time.Now().UTC().Add(-time.Minute)
	_, err = s1.EnqueueNotification(ctx, domain.AlertNotification{
		ID: "pend-1", AlertID: "a-1", ClientID: "u",
		WebhookURL: "https://example.com/hook", PayloadJSON: `{}`,
		Status: domain.NotificationPending, NextAttemptAt: past, CreatedAt: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s1.ScheduleNotificationRetry(ctx, "pend-1", 1, past, "timeout"); err != nil {
		t.Fatal(err)
	}
	_ = s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	due, err := s2.ListDueNotifications(ctx, time.Now().UTC(), 10)
	if err != nil || len(due) != 1 || due[0].Attempts != 1 || due[0].LastError != "timeout" {
		t.Fatalf("after reopen: %+v err=%v", due, err)
	}
	wh, _ := s2.GetWebhook(ctx, "u")
	if wh.URL != "https://example.com/hook" {
		t.Fatalf("webhook lost: %+v", wh)
	}
}

func TestSQLite_DigestDedupeAndSeal(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 28, 10, 15, 0, 0, time.UTC)
	d1, err := s.AddDigestItem(ctx, "c", "https://hooks.example.com/d", "alert-a", `{"alertId":"alert-a","lastPrice":1}`, at)
	if err != nil {
		t.Fatal(err)
	}
	// Same alert again in same hour — still one item, payload updated
	d2, err := s.AddDigestItem(ctx, "c", "https://hooks.example.com/d", "alert-a", `{"alertId":"alert-a","lastPrice":2}`, at.Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if d1.ID != d2.ID {
		t.Fatalf("same hour digest ids differ: %s vs %s", d1.ID, d2.ID)
	}
	_, err = s.AddDigestItem(ctx, "c", "https://hooks.example.com/d", "alert-b", `{"alertId":"alert-b","lastPrice":3}`, at.Add(20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.ListDigestItems(ctx, d1.ID)
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	// Seal after window ends
	n, err := s.SealOpenDigests(ctx, time.Date(2026, 7, 28, 11, 0, 1, 0, time.UTC))
	if err != nil || n != 1 {
		t.Fatalf("sealed=%d err=%v", n, err)
	}
	got, err := s.GetDigest(ctx, d1.ID)
	if err != nil || got.Status != domain.DigestPending || got.PayloadJSON == "" {
		t.Fatalf("%+v %v", got, err)
	}
	if !strings.Contains(got.PayloadJSON, `"count":2`) && !strings.Contains(got.PayloadJSON, `"count": 2`) {
		// json.Marshal uses no space after colon
		if !strings.Contains(got.PayloadJSON, `"count":2`) {
			t.Fatalf("payload=%s", got.PayloadJSON)
		}
	}
	due, err := s.ListDueDigests(ctx, time.Date(2026, 7, 28, 11, 0, 1, 0, time.UTC), 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("due=%+v err=%v", due, err)
	}
}

func TestSQLite_DigestPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alerts.db")
	ctx := context.Background()
	at := time.Date(2026, 7, 28, 12, 5, 0, 0, time.UTC)
	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := s1.AddDigestItem(ctx, "u", "https://example.com/h", "a1", `{"alertId":"a1"}`, at)
	if err != nil {
		t.Fatal(err)
	}
	_ = s1.Close()
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	items, err := s2.ListDigestItems(ctx, d.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("%+v %v", items, err)
	}
	n, err := s2.SealOpenDigests(ctx, time.Date(2026, 7, 28, 13, 0, 0, 0, time.UTC))
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
