package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubKeyCount struct {
	n   int
	err error
}

func (s stubKeyCount) CountActive(_ context.Context, _ string) (int, error) {
	return s.n, s.err
}

func TestMasterTenant_RemoteMasterBlocked(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := MasterTenant(true, stubKeyCount{n: 1})(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolio", nil)
	req.RemoteAddr = "203.0.113.9:443"
	req = req.WithContext(WithIdentity(req.Context(), &AuthIdentity{
		Master: true, DenyImpersonate: true, Loopback: false,
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("remote master GET want 403 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestMasterTenant_LoopbackAndUserKeyAllowed(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := MasterTenant(true, stubKeyCount{n: 3})(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolio", nil)
	req = req.WithContext(WithIdentity(req.Context(), &AuthIdentity{
		Master: true, DenyImpersonate: true, Loopback: true,
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("loopback master: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio", nil)
	req = req.WithContext(WithIdentity(req.Context(), &AuthIdentity{
		UserKey: true, ClientID: "alice", CanTrade: true,
	}))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("user key: %d", rec.Code)
	}
}

func TestMasterTenant_BootstrapCreateKeyWhenNoneExist(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	h := MasterTenant(true, stubKeyCount{n: 0})(inner)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/account/api-keys", strings.NewReader(`{"clientId":"new-user","name":"desk"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "new-user")
	req.RemoteAddr = "203.0.113.9:443"
	req = req.WithContext(WithIdentity(req.Context(), &AuthIdentity{
		Master: true, DenyImpersonate: true, Loopback: false,
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("bootstrap create: %d %s", rec.Code, rec.Body.String())
	}

	h = MasterTenant(true, stubKeyCount{n: 2})(inner)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/account/api-keys", strings.NewReader(`{"clientId":"old-user","name":"steal"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "old-user")
	req.RemoteAddr = "203.0.113.9:443"
	req = req.WithContext(WithIdentity(req.Context(), &AuthIdentity{
		Master: true, DenyImpersonate: true, Loopback: false,
	}))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("existing tenant mint: %d %s", rec.Code, rec.Body.String())
	}
}

func TestMasterTenant_DisabledPassthrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := MasterTenant(false, nil)(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolio", nil)
	req = req.WithContext(WithIdentity(req.Context(), &AuthIdentity{
		Master: true, DenyImpersonate: true, Loopback: false,
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("strict=false: %d", rec.Code)
	}
}
