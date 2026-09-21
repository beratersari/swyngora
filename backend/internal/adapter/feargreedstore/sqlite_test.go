package feargreedstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLiteFearGreedRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "fear.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	bucket := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	first := domain.FearGreedSample{Time: bucket, Score: 42, Level: domain.FearGreedFear, TopID: domain.FearGreedFactorVolatility, TopPull: -6.2}
	if err := s.SaveFearGreed(ctx, first); err != nil {
		t.Fatal(err)
	}
	updated := first
	updated.Score = 61
	updated.Level = domain.FearGreedGreed
	updated.TopID = domain.FearGreedFactorFunding
	updated.TopPull = 4.5
	if err := s.SaveFearGreed(ctx, updated); err != nil {
		t.Fatal(err)
	}
	older := domain.FearGreedSample{Time: bucket.Add(-48 * time.Hour), Score: 20, Level: domain.FearGreedExtremeFear}
	if err := s.SaveFearGreed(ctx, older); err != nil {
		t.Fatal(err)
	}
	rows, err := s.ListFearGreed(ctx, bucket.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Score != 61 || rows[0].TopID != domain.FearGreedFactorFunding {
		t.Fatalf("%+v", rows)
	}
	n, err := s.PurgeFearGreedBefore(ctx, bucket.Add(-24*time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("purged %d err %v", n, err)
	}
}

func TestMemoryFearGreedReplacesBucket(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	at := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	_ = m.SaveFearGreed(ctx, domain.FearGreedSample{Time: at, Score: 10, Level: domain.FearGreedExtremeFear})
	_ = m.SaveFearGreed(ctx, domain.FearGreedSample{Time: at, Score: 80, Level: domain.FearGreedExtremeGreed})
	rows, err := m.ListFearGreed(ctx, at.Add(-time.Minute))
	if err != nil || len(rows) != 1 || rows[0].Score != 80 {
		t.Fatalf("%+v %v", rows, err)
	}
}
