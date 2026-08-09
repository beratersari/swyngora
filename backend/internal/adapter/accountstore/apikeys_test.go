package accountstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_APIKeys(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "acc.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.CloseDB()
	ctx := context.Background()
	now := time.Now().UTC()
	k, err := s.CreateAPIKey(ctx, domain.APIKey{
		ID: "id1", ClientID: "c", Name: "bot", Prefix: "swy_abcd", Hash: "hh",
		Permission: domain.APIKeyPermissionRead, CreatedAt: now,
	})
	if err != nil || k.Name != "bot" {
		t.Fatalf("%+v %v", k, err)
	}
	got, err := s.GetAPIKeyByHash(ctx, "hh")
	if err != nil || got.ID != "id1" {
		t.Fatalf("%+v %v", got, err)
	}
	n, err := s.CountActiveAPIKeys(ctx, "c")
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	rev, err := s.RevokeAPIKey(ctx, "c", "id1", now)
	if err != nil || !rev.IsRevoked() {
		t.Fatalf("%+v %v", rev, err)
	}
	n, _ = s.CountActiveAPIKeys(ctx, "c")
	if n != 0 {
		t.Fatalf("active after revoke=%d", n)
	}
	if err := s.DeleteAPIKeysByClient(ctx, "c"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAPIKey(ctx, "c", "id1"); err != domain.ErrNotFound {
		t.Fatalf("want gone: %v", err)
	}
}
