package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
)

func TestAccountGate_BodyClientIDClosed(t *testing.T) {
	svc := account.New(accountstore.NewMemory(), account.DataPurgeDeps{})
	if _, err := svc.Close(context.Background(), "user1"); err != nil {
		t.Fatal(err)
	}
	called := false
	h := AccountGate(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", strings.NewReader(`{"clientId":"user1","side":"buy"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("handler should not run for closed account")
	}
	if !strings.Contains(rec.Body.String(), "account_closed") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestAccountGate_MissingClientIDFailClosed(t *testing.T) {
	svc := account.New(accountstore.NewMemory(), account.DataPurgeDeps{})
	h := AccountGate(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/watchlist", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "clientId is required") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestAccountGate_BodyClientIDRestoredAndActive(t *testing.T) {
	svc := account.New(accountstore.NewMemory(), account.DataPurgeDeps{})
	var gotBody string
	h := AccountGate(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	payload := `{"clientId":"user2","side":"buy"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if gotBody != payload {
		t.Fatalf("body not restored: %q", gotBody)
	}
}

func TestAccountGate_HeaderBodyMismatchRejected(t *testing.T) {
	svc := account.New(accountstore.NewMemory(), account.DataPurgeDeps{})
	if _, err := svc.Close(context.Background(), "closed-user"); err != nil {
		t.Fatal(err)
	}
	called := false
	h := AccountGate(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", strings.NewReader(`{"clientId":"closed-user","side":"buy"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "active-decoy")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("handler must not run when header and body clientId disagree")
	}
}

func TestAccountGate_MarketUnaffected(t *testing.T) {
	svc := account.New(accountstore.NewMemory(), account.DataPurgeDeps{})
	h := AccountGate(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/market/ticker/24h?symbol=BTCUSDT", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d", rec.Code)
	}
}
