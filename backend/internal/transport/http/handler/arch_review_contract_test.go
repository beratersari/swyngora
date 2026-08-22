package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArchReview_HTTPOrderRequiresPortfolioIDWithTwoBooks(t *testing.T) {
	h := newPortfolioHandler(t)
	create := func(name string) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{
			"clientId": "arch-http", "startingBalance": 10000, "name": name,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Client-Id", "arch-http")
		rr := httptest.NewRecorder()
		h.Create(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create %s: %d %s", name, rr.Code, rr.Body.String())
		}
	}
	create("Main")
	create("Risky")

	body, _ := json.Marshal(map[string]any{
		"clientId": "arch-http", "symbol": "BTCUSDT", "side": "buy", "quantity": 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "arch-http")
	rr := httptest.NewRecorder()
	h.PlaceOrder(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("order without portfolioId succeeded with two books: %s", rr.Body.String())
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "portfolioId") {
		t.Fatalf("error should mention portfolioId: %s", rr.Body.String())
	}
}
