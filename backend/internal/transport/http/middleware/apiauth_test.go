package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
