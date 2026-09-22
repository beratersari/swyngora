package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetNetPosition_MissingSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/net-position", nil)
	rr := httptest.NewRecorder()
	h.GetNetPosition(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
