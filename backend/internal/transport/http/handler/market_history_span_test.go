package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetHistorySpan_Empty(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/history-span?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetHistorySpan(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body domain.HistorySpan
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || len(body.Series) != 0 || body.Note == "" {
		t.Fatalf("%+v", body)
	}
}

func TestGetHistorySpan_RequiresSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/history-span", nil)
	rr := httptest.NewRecorder()
	h.GetHistorySpan(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
