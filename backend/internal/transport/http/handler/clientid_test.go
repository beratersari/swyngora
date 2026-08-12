package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

func TestResolveClientID_UserKeyForcesBinding(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", nil)
	req = req.WithContext(middleware.WithIdentity(req.Context(), &middleware.AuthIdentity{
		ClientID: "alice", UserKey: true, CanTrade: true, KeyID: "k1",
	}))
	got, err := resolveClientID(req, "bob")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want forbidden, got %q err=%v", got, err)
	}
	got, err = resolveClientID(req, "")
	if err != nil || got != "alice" {
		t.Fatalf("got %q err=%v", got, err)
	}
	got, err = resolveClientID(req, "alice")
	if err != nil || got != "alice" {
		t.Fatalf("same id %q err=%v", got, err)
	}
}

func TestResolveClientID_MasterUsesBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Client-Id", "header-client")
	got, err := resolveClientID(req, "body-client")
	if err != nil || got != "body-client" {
		t.Fatalf("got %q err=%v", got, err)
	}
	got, err = resolveClientID(req, "")
	if err != nil || got != "header-client" {
		t.Fatalf("header fallback %q err=%v", got, err)
	}
}
