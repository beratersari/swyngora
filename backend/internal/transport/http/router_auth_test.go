package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/realtime"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

func TestRouter_QuerySecretRejectedAndTicketWorks(t *testing.T) {
	hub := realtime.NewHub(realtime.Options{Interval: time.Hour})
	h := NewRouterWithOptions(evidenceMarket(), nil, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster, Realtime: hub,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/watchlist?clientId=a&token="+evidenceMaster, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("query token: %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/realtime/ticket", nil)
	req.Header.Set("Authorization", "Bearer "+evidenceMaster)
	req.Header.Set("X-Client-Id", "alice")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("mint ticket: %d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	ticket, _ := body["ticket"].(string)
	if ticket == "" || body["clientId"] != "alice" {
		t.Fatalf("ticket body=%v", body)
	}

	// Ticket is not a REST credential.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/watchlist?clientId=alice&ticket="+ticket, nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("ticket on REST: %d %s", rr.Code, rr.Body.String())
	}
}

func TestRouter_StrictMasterRemoteForbidden(t *testing.T) {
	watch := watchlist.New(watchliststore.NewMemory())
	h := NewRouterWithOptions(evidenceMarket(), watch, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster, StrictMasterTenant: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/watchlist?clientId=victim", nil)
	req.Header.Set("Authorization", "Bearer "+evidenceMaster)
	req.Header.Set("X-Client-Id", "victim")
	req.RemoteAddr = "203.0.113.10:443"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("remote master want 403 got %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/watchlist?clientId=victim", nil)
	req.Header.Set("Authorization", "Bearer "+evidenceMaster)
	req.Header.Set("X-Client-Id", "victim")
	req.RemoteAddr = "127.0.0.1:9999"
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("loopback master want 200 got %d %s", rr.Code, rr.Body.String())
	}
}
