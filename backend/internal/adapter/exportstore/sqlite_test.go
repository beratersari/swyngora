package exportstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_ExportLifecycle(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "export.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	now := time.Now().UTC()
	job, err := s.Create(ctx, domain.ExportJob{
		ID: "e1", ClientID: "c1", Format: domain.ExportFormatJSON,
		Sections: []domain.ExportSection{domain.ExportSectionWatchlist},
		Status: domain.ExportPending, CreatedAt: now,
	})
	if err != nil || job.ID != "e1" {
		t.Fatalf("%+v %v", job, err)
	}
	active, err := s.FindActive(ctx, "c1")
	if err != nil || active.ID != "e1" {
		t.Fatalf("%+v %v", active, err)
	}
	ok, err := s.Claim(ctx, "e1", now)
	if err != nil || !ok {
		t.Fatalf("claim %v %v", ok, err)
	}
	_ = s.UpdateProgress(ctx, "e1", 50, "watchlist")
	exp := now.Add(time.Hour)
	if err := s.Finish(ctx, "e1", domain.ExportCompleted, "f.json", "/tmp/f.json", 10, &exp, "", now); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "c1", "e1")
	if err != nil || got.Status != domain.ExportCompleted || got.ProgressPct != 100 {
		t.Fatalf("%+v %v", got, err)
	}
	// Expired cleanup list
	past := now.Add(-time.Minute)
	_ = s.Finish(ctx, "e1", domain.ExportCompleted, "f.json", "/tmp/f.json", 10, &past, "", now)
	list, err := s.ListExpiredCompleted(ctx, now, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
}
