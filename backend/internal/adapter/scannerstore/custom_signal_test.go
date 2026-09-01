package scannerstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_CustomSignalProfiles(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "scan.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	now := time.Now().UTC()
	w, err := domain.ParseCustomSignalWeightMap(map[string]float64{
		domain.CustomFactorRSIRecovery: 40,
		domain.CustomFactorHuntUp:      60,
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.CreateProfile(ctx, domain.CustomSignalProfile{
		ID: "p1", ClientID: "c1", Name: "RSI + hunt", Weights: w, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || p.Name != "RSI + hunt" {
		t.Fatalf("%+v %v", p, err)
	}
	got, err := s.GetProfile(ctx, "c1", "p1")
	if err != nil || got.Weights.Pct[domain.CustomFactorHuntUp] != 60 {
		t.Fatalf("%+v %v", got, err)
	}
	n, err := s.CountProfiles(ctx, "c1")
	if err != nil || n != 1 {
		t.Fatalf("count=%d %v", n, err)
	}
	got.Name = "Renamed"
	got.UpdatedAt = now.Add(time.Minute)
	if _, err := s.UpdateProfile(ctx, *got); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListProfiles(ctx, "c1")
	if err != nil || len(list) != 1 || list[0].Name != "Renamed" {
		t.Fatalf("%+v %v", list, err)
	}
	if err := s.SetDefaultProfile(ctx, "c1", "p1"); err != nil {
		t.Fatal(err)
	}
	def, err := s.GetProfile(ctx, "c1", "p1")
	if err != nil || !def.IsDefault {
		t.Fatalf("default %+v %v", def, err)
	}
	w2, err := domain.ParseCustomSignalWeightMap(map[string]float64{domain.CustomFactorHuntUp: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateProfile(ctx, domain.CustomSignalProfile{
		ID: "p2", ClientID: "c1", Name: "Other", Weights: w2, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDefaultProfile(ctx, "c1", "missing"); err != domain.ErrNotFound {
		t.Fatalf("missing default err=%v", err)
	}
	still, err := s.GetProfile(ctx, "c1", "p1")
	if err != nil || !still.IsDefault {
		t.Fatalf("failed set-default must not clear existing %+v %v", still, err)
	}
	if err := s.SetDefaultProfile(ctx, "c1", "p2"); err != nil {
		t.Fatal(err)
	}
	a, _ := s.GetProfile(ctx, "c1", "p1")
	b, _ := s.GetProfile(ctx, "c1", "p2")
	if a.IsDefault || !b.IsDefault {
		t.Fatalf("want only p2 default a=%v b=%v", a.IsDefault, b.IsDefault)
	}
	w3, err := domain.ParseCustomSignalWeightMap(map[string]float64{domain.CustomFactorRSIRecovery: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateProfile(ctx, domain.CustomSignalProfile{
		ID: "p3", ClientID: "c1", Name: "Created default", Weights: w3, IsDefault: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	b, _ = s.GetProfile(ctx, "c1", "p2")
	c, _ := s.GetProfile(ctx, "c1", "p3")
	if b.IsDefault || !c.IsDefault {
		t.Fatalf("create-as-default must clear others b=%v c=%v", b.IsDefault, c.IsDefault)
	}
	if err := s.DeleteProfile(ctx, "c1", "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetProfile(ctx, "c1", "p1"); err != domain.ErrNotFound {
		t.Fatalf("want not found, %v", err)
	}
}

func TestSQLite_PurgeClientRemovesCustomProfiles(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "scan.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	w := domain.DefaultCustomSignalWeights()
	if _, err := s.CreateProfile(ctx, domain.CustomSignalProfile{
		ID: "p2", ClientID: "gone", Name: "tmp", Weights: w, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.PurgeClient(ctx, "gone"); err != nil {
		t.Fatal(err)
	}
	n, err := s.CountProfiles(ctx, "gone")
	if err != nil || n != 0 {
		t.Fatalf("count=%d %v", n, err)
	}
}
