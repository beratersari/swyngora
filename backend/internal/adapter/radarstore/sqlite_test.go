package radarstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestInsertIfQuietAndFill(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "radar.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	hit := domain.ForensicsScanHit{Symbol: "BTCUSDT", Exchange: "binance", Lean: domain.CVDDirUp, Similarity: 80}
	c := domain.NewRadarCase(hit, at, 100, "3d", "1h")
	c.ID = "c1"
	got, inserted, err := s.InsertIfQuiet(ctx, c, domain.DefaultRadarCooldown)
	if err != nil || !inserted || got.Symbol != "BTCUSDT" {
		t.Fatalf("%+v %v %v", got, inserted, err)
	}
	again := c
	again.ID = "c2"
	again.FoundAt = at.Add(time.Hour)
	_, inserted, err = s.InsertIfQuiet(ctx, again, domain.DefaultRadarCooldown)
	if err != nil || inserted {
		t.Fatalf("cooldown %v %v", inserted, err)
	}
	later := c
	later.ID = "c3"
	later.FoundAt = at.Add(5 * time.Hour)
	_, inserted, err = s.InsertIfQuiet(ctx, later, domain.DefaultRadarCooldown)
	if err != nil || !inserted {
		t.Fatalf("after cooldown %v %v", inserted, err)
	}
	list, err := s.List(ctx, domain.RadarListQuery{Since: at.Add(-time.Hour), Limit: 10})
	if err != nil || len(list) != 2 {
		t.Fatalf("list %+v %v", list, err)
	}
	due, err := s.ListDueAfters(ctx, at.Add(20*time.Minute), 10)
	if err != nil || len(due) < 1 {
		t.Fatalf("due %+v %v", due, err)
	}
	fill := domain.FillRadarAfter(100, 101.5, domain.CVDDirUp, at.Add(20*time.Minute))
	fill.Horizon = "15m"
	if err := s.FillAfter(ctx, "c1", "15m", fill); err != nil {
		t.Fatal(err)
	}
	if err := s.FillAfter(ctx, "c1", "15m", fill); err == nil {
		t.Fatal("second fill")
	}
	listed, err := s.List(ctx, domain.RadarListQuery{Symbol: "BTCUSDT", Limit: 5})
	if err != nil || len(listed) == 0 || listed[len(listed)-1].After[0].FilledAt == nil {
		t.Fatalf("filled %+v %v", listed, err)
	}
}
