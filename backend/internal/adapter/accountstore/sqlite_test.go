package accountstore

import (
	"context"
	"path/filepath"
	"testing"
)

func TestTelegramIdentityPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "accounts.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	id1, err := s.ClientIDForTelegramUser(ctx, 42)
	if err != nil || id1 == "" {
		t.Fatalf("%q %v", id1, err)
	}
	_ = s.CloseDB()
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s2.CloseDB() })
	id2, err := s2.ClientIDForTelegramUser(ctx, 42)
	if err != nil || id2 != id1 {
		t.Fatalf("want %q got %q err=%v", id1, id2, err)
	}
	other, err := s2.ClientIDForTelegramUser(ctx, 7)
	if err != nil || other == id1 {
		t.Fatalf("expected distinct id for other user: %q", other)
	}
}
