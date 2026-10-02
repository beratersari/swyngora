package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

func TestGetDerivativesBoard_NamesMissingVenues(t *testing.T) {
	h := NewMarketHandler(market.New(nil, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/derivatives-board?limit=5", nil)
	rr := httptest.NewRecorder()
	h.GetDerivativesBoard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body derivativesBoardResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Universe != 5 || body.Definite || len(body.Coins) != 5 || body.Coins[0].OpenInterest != "" {
		t.Fatalf("%+v", body)
	}
	if len(body.Coverage.Missing) == 0 {
		t.Fatalf("coverage %+v", body.Coverage)
	}
}

func TestGetDerivativesBoard_BadLimit(t *testing.T) {
	h := NewMarketHandler(market.New(nil, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/derivatives-board?limit=nope", nil)
	rr := httptest.NewRecorder()
	h.GetDerivativesBoard(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rr.Code)
	}
}
