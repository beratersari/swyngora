package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestAPIAuth_NoopWhenEmpty(t *testing.T) {
	h := APIAuth("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/watchlist", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestAPIAuth_ProtectsTenantAndMCP(t *testing.T) {
	h := APIAuth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Protected without token
	for _, path := range []string{"/api/v1/watchlist", "/api/v1/portfolio", "/mcp", "/api/v1/ai/chat"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: want 401 got %d", path, rec.Code)
		}
		ct := rec.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Fatalf("%s: content-type=%q want application/json", path, ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"code":"unauthorized"`) || !strings.Contains(body, `"message"`) {
			t.Fatalf("%s: body not nested Error envelope: %s", path, body)
		}
	}

	// Public market + health
	for _, path := range []string{"/health", "/api/v1/market/ticker/24h"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: want 200 got %d", path, rec.Code)
		}
	}

	// Bearer ok
	req := httptest.NewRequest(http.MethodGet, "/api/v1/watchlist", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bearer: %d", rec.Code)
	}

	// X-API-Key ok
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("X-API-Key", "secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("api-key: %d", rec.Code)
	}

	// Wrong token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/watchlist", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong: %d", rec.Code)
	}
}

type stubKeys struct {
	keys map[string]*domain.APIKey
}

func (s stubKeys) Authenticate(_ context.Context, secret string) (*domain.APIKey, error) {
	k, ok := s.keys[secret]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return k, nil
}

func TestAPIAuth_UserKeyBindsClientAndScope(t *testing.T) {
	lookup := stubKeys{keys: map[string]*domain.APIKey{
		"swy_readsecret00000000000000000000000000000000": {
			ID: "k1", ClientID: "user-a", Permission: domain.APIKeyPermissionRead,
		},
		"swy_tradesecret0000000000000000000000000000000": {
			ID: "k2", ClientID: "user-a", Permission: domain.APIKeyPermissionTrade,
		},
	}}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := IdentityFrom(r.Context())
		if id != nil {
			w.Header().Set("X-Got-Client", id.ClientID)
			if id.CanTrade {
				w.Header().Set("X-Can-Trade", "1")
			}
		}
		w.WriteHeader(http.StatusOK)
	})
	h := APIAuthWith("master-token", lookup)(APIKeyScope(inner))

	// read key GET ok
	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolio", nil)
	req.Header.Set("X-API-Key", "swy_readsecret00000000000000000000000000000000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("X-Got-Client") != "user-a" {
		t.Fatalf("read get %d client=%s", rec.Code, rec.Header().Get("X-Got-Client"))
	}

	// read key POST order forbidden
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", nil)
	req.Header.Set("X-API-Key", "swy_readsecret00000000000000000000000000000000")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("read post want 403 got %d", rec.Code)
	}

	// trade key POST ok
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", nil)
	req.Header.Set("X-API-Key", "swy_tradesecret0000000000000000000000000000000")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("X-Can-Trade") != "1" {
		t.Fatalf("trade post %d", rec.Code)
	}

	// user key cannot manage keys
	req = httptest.NewRequest(http.MethodGet, "/api/v1/account/api-keys", nil)
	req.Header.Set("X-API-Key", "swy_tradesecret0000000000000000000000000000000")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("manage keys %d", rec.Code)
	}

	// master still works
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", nil)
	req.Header.Set("Authorization", "Bearer master-token")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("master %d", rec.Code)
	}
}

func TestAPIAuth_OPTIONSBypass(t *testing.T) {
	h := APIAuth("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/api/v1/watchlist", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code=%d", rec.Code)
	}
}
