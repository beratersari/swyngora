package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Finding 1 over HTTP: POST /transfers must not 1:1-credit a different book currency.
func TestVerifyHTTP_TransferRejectsDifferentCurrency(t *testing.T) {
	h := newPortfolioHandler(t)

	create := func(name, currency string, start float64) string {
		t.Helper()
		body, _ := json.Marshal(map[string]any{
			"clientId": "http-fx", "startingBalance": start, "name": name, "currency": currency,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.Create(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create %s %d %s", name, rr.Code, rr.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &out)
		id, _ := out["id"].(string)
		if id == "" {
			t.Fatalf("no id in %s", rr.Body.String())
		}
		return id
	}

	usdtID := create("USDT desk", "USDT", 10000)
	tryID := create("TRY desk", "TRY", 1000)

	body, _ := json.Marshal(map[string]any{
		"clientId": "http-fx", "fromPortfolioId": usdtID, "toPortfolioId": tryID, "amount": 2000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Transfer(rr, req)
	if rr.Code == http.StatusOK {
		t.Errorf("CONFIRMED F1 HTTP: transfer 2000 USDT→TRY returned 200 %s", rr.Body.String())
		return
	}
	if rr.Code != http.StatusBadRequest {
		t.Logf("rejected with status %d (still closed): %s", rr.Code, rr.Body.String())
	}
	t.Logf("FALSE POSITIVE / NOT REPRODUCED F1 HTTP: status=%d body=%s", rr.Code, rr.Body.String())
}
