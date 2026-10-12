package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

func TestGetCoinSituation_FutureAtIsRejected(t *testing.T) {
	h := NewMarketHandler(market.New(nil, nil))
	at := url.QueryEscape(time.Now().UTC().Add(2 * time.Minute).Format(time.RFC3339Nano))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/situation?symbol=BTCUSDT&at="+at, nil)
	rr := httptest.NewRecorder()
	h.GetCoinSituation(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetCoinSituation_BadAtIsRejected(t *testing.T) {
	h := NewMarketHandler(market.New(nil, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/situation?symbol=BTCUSDT&at=not-a-time", nil)
	rr := httptest.NewRecorder()
	h.GetCoinSituation(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetCoinSituation_PastAtLeavesThePocketOff(t *testing.T) {
	h := NewMarketHandler(market.New(nil, nil))
	at := url.QueryEscape(time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC).Format(time.RFC3339))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/situation?symbol=BTCUSDT&at="+at, nil)
	rr := httptest.NewRecorder()
	h.GetCoinSituation(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["liquidation"]; ok {
		t.Fatalf("pocket leaked %s", rr.Body.String())
	}
	if _, ok := body["liquidationMissing"]; ok {
		t.Fatalf("pocket missing leaked %s", rr.Body.String())
	}
	if !strings.Contains(string(body["note"]), "not part of this read") {
		t.Fatalf("note %s", body["note"])
	}
}
