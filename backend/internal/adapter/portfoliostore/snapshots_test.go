package portfoliostore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_EquitySnapshots(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "snap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	if _, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "s1", Currency: "USDT", StartingBalance: 1000, CashBalance: 1000,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	ids, err := s.ListPortfolioClientIDs(ctx)
	if err != nil || len(ids) != 1 || ids[0] != "s1" {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	b1 := domain.SnapshotBucket(now, domain.DefaultSnapshotInterval)
	b2 := b1.Add(domain.DefaultSnapshotInterval)
	if err := s.UpsertEquitySnapshot(ctx, domain.EquitySnapshot{
		ClientID: "s1", BucketAt: b1, TakenAt: b1, Equity: 1000, CashBalance: 1000,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertEquitySnapshot(ctx, domain.EquitySnapshot{
		ClientID: "s1", BucketAt: b2, TakenAt: b2, Equity: 1100, CashBalance: 1100,
	}); err != nil {
		t.Fatal(err)
	}
	// Same bucket overwrite
	if err := s.UpsertEquitySnapshot(ctx, domain.EquitySnapshot{
		ClientID: "s1", BucketAt: b2, TakenAt: b2.Add(time.Minute), Equity: 1110, CashBalance: 1110,
	}); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListEquitySnapshots(ctx, "s1", b1, b2)
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	if list[1].Equity != 1110 {
		t.Fatalf("upsert overwrite=%v", list[1].Equity)
	}
	carry, err := s.LatestEquitySnapshotBefore(ctx, "s1", b2)
	if err != nil || carry == nil || carry.Equity != 1000 {
		t.Fatalf("carry=%+v err=%v", carry, err)
	}
	none, err := s.LatestEquitySnapshotBefore(ctx, "s1", b1)
	if err != nil || none != nil {
		t.Fatalf("none=%+v err=%v", none, err)
	}
	n, err := s.DeleteEquitySnapshotsBefore(ctx, b2)
	if err != nil || n != 1 {
		t.Fatalf("deleted=%d err=%v", n, err)
	}
}
