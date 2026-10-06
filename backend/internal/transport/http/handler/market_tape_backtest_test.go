package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTapeBacktest_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/tape-backtest?symbol=BTCUSDT&condition=cvd&exchange=binance", nil)
	rr := httptest.NewRecorder()
	h.GetTapeBacktest(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Symbol    string   `json:"symbol"`
		Condition string   `json:"condition"`
		Definite  bool     `json:"definite"`
		Missing   []string `json:"missing"`
		Note      string   `json:"note"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Condition != "cvd" || body.Definite {
		t.Fatalf("%+v", body)
	}
	if len(body.Missing) != 1 || body.Missing[0] != "binance" || body.Note == "" {
		t.Fatalf("%+v", body)
	}
}

func TestGetTapeBacktest_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/tape-backtest?condition=book&exchange=okx", nil)
	rr := httptest.NewRecorder()
	h.GetTapeBacktest(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
