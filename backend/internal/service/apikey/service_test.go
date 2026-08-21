package apikey

import (
	"context"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestAPIKey_CreateListRevokeAuth(t *testing.T) {
	svc := New(accountstore.NewMemory())
	ctx := context.Background()
	created, err := svc.Create(ctx, CreateInput{ClientID: "u1", Name: "Bot", Permission: "trade"})
	if err != nil || created.Secret == "" || created.Key.Prefix == "" {
		t.Fatalf("%+v %v", created, err)
	}
	if created.Key.Permission != domain.APIKeyPermissionTrade {
		t.Fatalf("%s", created.Key.Permission)
	}
	got, err := svc.Authenticate(ctx, created.Secret)
	if err != nil || got.ID != created.Key.ID || !got.CanTrade() {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := svc.Authenticate(ctx, "swy_deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"); err != domain.ErrNotFound {
		t.Fatalf("bad secret: %v", err)
	}

	list, err := svc.List(ctx, "u1")
	if err != nil || len(list) != 1 || list[0].Hash == "" {
		t.Fatalf("%+v %v", list, err)
	}

	rev, err := svc.Revoke(ctx, "u1", created.Key.ID)
	if err != nil || !rev.IsRevoked() {
		t.Fatalf("%+v %v", rev, err)
	}
	if _, err := svc.Authenticate(ctx, created.Secret); err != domain.ErrNotFound {
		t.Fatalf("revoked still works: %v", err)
	}
}

func TestAPIKey_CreateRejectsIDsDomainWouldReject(t *testing.T) {
	svc := New(accountstore.NewMemory())
	ctx := context.Background()
	for _, id := range []string{"anonymous", "ai-assistant", "http-default", "tg-12345"} {
		if _, err := domain.NormalizeClientID(id); err == nil {
			t.Fatalf("precondition: domain must reject %q", id)
		}
		if _, err := svc.Create(ctx, CreateInput{ClientID: id, Name: "k", Permission: "read"}); err == nil {
			t.Fatalf("%s: Create must use domain.NormalizeClientID", id)
		}
	}
}

func TestAPIKey_ReadCannotTrade(t *testing.T) {
	svc := New(accountstore.NewMemory())
	created, err := svc.Create(context.Background(), CreateInput{ClientID: "u2", Name: "Reader", Permission: "read"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Authenticate(context.Background(), created.Secret)
	if err != nil || got.CanTrade() {
		t.Fatalf("read key must not trade: %+v %v", got, err)
	}
}

func TestAPIKey_MaxActive(t *testing.T) {
	svc := New(accountstore.NewMemory())
	ctx := context.Background()
	for i := 0; i < domain.MaxAPIKeysPerClient; i++ {
		if _, err := svc.Create(ctx, CreateInput{ClientID: "u3", Name: "k", Permission: "read"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "u3", Name: "one-more", Permission: "read"}); err == nil {
		t.Fatal("want max keys error")
	}
}
