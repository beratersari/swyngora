package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/fundingarbstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/fundingarb"
)

func newFundingArbWatchHandler(t *testing.T) *FundingArbWatchHandler {
	t.Helper()
	store, err := fundingarbstore.Open(filepath.Join(t.TempDir(), "fa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return NewFundingArbWatchHandler(fundingarb.New(store, nil))
}

func TestFundingArbWatchHTTP_CreateListGetDelete(t *testing.T) {
	h := newFundingArbWatchHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "fa-client", "symbol": "BTCUSDT", "notional": 10000,
		"holdHours": 24, "minProfit": 10,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/funding-arb/watches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "fa-client")
	rr := httptest.NewRecorder()
	h.CreateWatch(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created fundingArbWatchDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil || created.ID == "" || created.MinProfit != 10 {
		t.Fatalf("%s %v", rr.Body.String(), err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/funding-arb/watches?clientId=fa-client", nil)
	req.Header.Set("X-Client-Id", "fa-client")
	rr = httptest.NewRecorder()
	h.ListWatches(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rr.Code, rr.Body.String())
	}
	var listed struct {
		Watches []fundingArbWatchDTO `json:"watches"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &listed); err != nil || len(listed.Watches) != 1 {
		t.Fatalf("%s %v", rr.Body.String(), err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/funding-arb/watches/"+created.ID+"?clientId=fa-client", nil)
	req.Header.Set("X-Client-Id", "fa-client")
	req.SetPathValue("id", created.ID)
	rr = httptest.NewRecorder()
	h.GetWatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/funding-arb/signals?clientId=fa-client", nil)
	req.Header.Set("X-Client-Id", "fa-client")
	rr = httptest.NewRecorder()
	h.ListSignals(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("signals status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/funding-arb/watches/"+created.ID+"?clientId=fa-client", nil)
	req.Header.Set("X-Client-Id", "fa-client")
	req.SetPathValue("id", created.ID)
	rr = httptest.NewRecorder()
	h.DeleteWatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestFundingArbWatchHTTP_CreateScanFollow(t *testing.T) {
	h := newFundingArbWatchHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "fa-client", "minProfit": 10, "notional": 10000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/funding-arb/watches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "fa-client")
	rr := httptest.NewRecorder()
	h.CreateWatch(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created fundingArbWatchDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil || created.Scope != "scan" || created.Symbol != "*" {
		t.Fatalf("%s %v", rr.Body.String(), err)
	}
}

func TestFundingArbWatchHTTP_BadMinProfit(t *testing.T) {
	h := newFundingArbWatchHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "fa-client", "symbol": "BTCUSDT", "minProfit": 0,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/funding-arb/watches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "fa-client")
	rr := httptest.NewRecorder()
	h.CreateWatch(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
