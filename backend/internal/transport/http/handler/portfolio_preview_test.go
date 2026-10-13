package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPortfolioHTTP_OrderPreviewDoesNotFill(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{"clientId": "http-prev", "startingBalance": 10000})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	body, _ = json.Marshal(map[string]any{
		"clientId": "http-prev", "symbol": "BTCUSDT", "side": "buy", "quantity": 1,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.PreviewOrder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("preview %d %s", rr.Code, rr.Body.String())
	}
	var prev map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &prev); err != nil {
		t.Fatal(err)
	}
	if prev["fillsNow"] != true || prev["cashDebit"].(float64) != 100 {
		t.Fatalf("body %+v", prev)
	}
	risk, _ := prev["risk"].(map[string]any)
	if risk["blocked"] == true {
		t.Fatalf("risk %+v", risk)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio?clientId=http-prev", nil)
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d %s", rr.Code, rr.Body.String())
	}
	var view map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &view)
	if view["cashBalance"].(float64) != 10000 {
		t.Fatalf("cash changed: %+v", view["cashBalance"])
	}
}
