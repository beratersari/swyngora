package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCandleFootprint_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/flow-at-prices/candles?symbol=BTCUSDT&interval=1h", nil)
	rr := httptest.NewRecorder()
	h.GetCandleFootprint(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body candleFootprintResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Interval != "1h" || body.StepSec != 3600 {
		t.Fatalf("%+v", body)
	}
}

func TestGetCandleFootprint_BadInterval(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/flow-at-prices/candles?symbol=BTCUSDT&interval=nope", nil)
	rr := httptest.NewRecorder()
	h.GetCandleFootprint(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
