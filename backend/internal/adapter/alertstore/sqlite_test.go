package alertstore

import (
	"context"
	"errors"
	"path/filepath"
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