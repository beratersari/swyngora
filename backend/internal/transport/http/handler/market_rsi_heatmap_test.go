package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRSIHeatmap_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/rsi-heatmap?interval=1h&limit=5", nil)
	rr := httptest.NewRecorder()
	h.GetRSIHeatmap(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body rsiHeatmapResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Exchange != "binance" || body.Period != 14 {
		t.Fatalf("%+v", body)
	}
	if body.Interval != "1h" {
		t.Fatalf("interval=%q", body.Interval)
	}
	if body.Note == "" {
		t.Fatal("expected disclaimer")
	}
}

func TestGetRSIHeatmap_BadLimit(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/rsi-heatmap?limit=nope", nil)
	rr := httptest.NewRecorder()
	h.GetRSIHeatmap(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
