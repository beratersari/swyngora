package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

func TestPortfolioHTTP_UserKeyCannotImpersonateClientId(t *testing.T) {
	h := newPortfolioHandler(t)
	// Seed alice portfolio via open (no identity) create.
	body, _ := json.Marshal(map[string]any{"clientId": "alice", "startingBalance": 10000})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create alice %d %s", rr.Code, rr.Body.String())
	}

	// Trade key for alice tries to deposit as bob → forbidden.
	body, _ = json.Marshal(map[string]any{"clientId": "bob", "amount": 100})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/deposits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithIdentity(req.Context(), &middleware.AuthIdentity{
		ClientID: "alice", UserKey: true, CanTrade: true, KeyID: "k1",
	}))
	rr = httptest.NewRecorder()
	h.Deposit(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d %s", rr.Code, rr.Body.String())
	}

	// Same key depositing without body clientId acts as alice.
	body, _ = json.Marshal(map[string]any{"amount": 100})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/deposits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithIdentity(req.Context(), &middleware.AuthIdentity{
		ClientID: "alice", UserKey: true, CanTrade: true, KeyID: "k1",
	}))
	rr = httptest.NewRecorder()
	h.Deposit(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("alice deposit %d %s", rr.Code, rr.Body.String())
	}
}
