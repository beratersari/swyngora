package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/pricediffstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
)

type handlerBooks struct {
	data map[string]*domain.RawOrderBook
}

func (f *handlerBooks) GetRawOrderBook(_ context.Context, exchange, symbol string) (*domain.RawOrderBook, error) {
	if f.data == nil {
		return nil, domain.ErrNotFound
	}
	b, ok := f.data[exchange+"|"+symbol]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return b, nil
}

func newPriceDiffHandler(t *testing.T) *PriceDiffHandler {
	t.Helper()
	store, err := pricediffstore.Open(filepath.Join(t.TempDir(), "pd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := pricediff.New(store, nil).WithBooks(&handlerBooks{data: map[string]*domain.RawOrderBook{
		"binance|BTCUSDT": {
			Symbol: "BTCUSDT", Live: true,
			Asks: []domain.PriceLevel{{Price: 100, Quantity: 2}},
			Bids: []domain.PriceLevel{{Price: 99, Quantity: 2}},
		},
		"bybit|BTCUSDT": {
			Symbol: "BTCUSDT", Live: true,
			Asks: []domain.PriceLevel{{Price: 104, Quantity: 2}},
			Bids: []domain.PriceLevel{{Price: 103, Quantity: 2}},
		},
	}})
	return NewPriceDiffHandler(svc)
}

func TestQuoteRoute_OK(t *testing.T) {
	h := newPriceDiffHandler(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/price-diff/quote?symbol=BTCUSDT&buyExchange=binance&sellExchange=bybit&notional=100&feeBuyPct=0.1&feeSellPct=0.1",
		nil)
	rr := httptest.NewRecorder()
	h.QuoteRoute(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body priceDiffQuoteDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AverageBuyPrice != "100" || body.AverageSellPrice != "103" || !body.Profitable {
		t.Fatalf("%+v", body)
	}
}

func TestQuoteRoute_BadSize(t *testing.T) {
	h := newPriceDiffHandler(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/price-diff/quote?symbol=BTCUSDT&buyExchange=binance&sellExchange=bybit",
		nil)
	rr := httptest.NewRecorder()
	h.QuoteRoute(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestQuoteScan_OK(t *testing.T) {
	h := newPriceDiffHandler(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/price-diff/quote/scan?symbol=BTCUSDT&notional=100&feeBinancePct=0.1&feeBybitPct=0.1",
		nil)
	rr := httptest.NewRecorder()
	h.QuoteScan(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body priceDiffQuoteScanDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.BestRoute == nil || len(body.Routes) < 2 {
		t.Fatalf("%+v", body)
	}
	for _, u := range body.Unavailable {
		if u.Exchange == "coinbase" && u.Message == "" {
			t.Fatalf("missing-book message empty: %+v", u)
		}
	}
}

func TestQuoteOpportunity_NotFound(t *testing.T) {
	h := newPriceDiffHandler(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/price-diff/opportunities/missing/quote?notional=100&clientId=u1",
		nil)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	h.QuoteOpportunity(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPriceDiffWatchHTTP_PauseResumeAndPatch(t *testing.T) {
	h := newPriceDiffHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "pd-client", "symbol": "BTCUSDT", "notional": 10000, "minProfit": 5,
		"minDurationSec": 30, "feeBinancePct": 0.1, "feeBybitPct": 0.1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/price-diff/watches", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "pd-client")
	rr := httptest.NewRecorder()
	h.CreateWatch(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	var created priceDiffWatchDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/price-diff/watches/"+created.ID+"/pause", nil)
	req.Header.Set("X-Client-Id", "pd-client")
	req.SetPathValue("id", created.ID)
	rr = httptest.NewRecorder()
	h.PauseWatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("pause %d %s", rr.Code, rr.Body.String())
	}
	var paused priceDiffWatchDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &paused); err != nil || paused.Status != "paused" {
		t.Fatalf("%s %v", rr.Body.String(), err)
	}

	patch, _ := json.Marshal(map[string]any{
		"notional": 15000, "minProfit": 8, "minDurationSec": 45, "feeBinancePct": 0.2,
	})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/price-diff/watches/"+created.ID, bytes.NewReader(patch))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "pd-client")
	req.SetPathValue("id", created.ID)
	rr = httptest.NewRecorder()
	h.UpdateWatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch %d %s", rr.Code, rr.Body.String())
	}
	var updated priceDiffWatchDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil ||
		updated.Notional != 15000 || updated.MinProfit != 8 || updated.MinDurationSec != 45 ||
		updated.FeeBinancePct != 0.2 || updated.Status != "paused" {
		t.Fatalf("%s %v", rr.Body.String(), err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/price-diff/watches/"+created.ID+"/resume", nil)
	req.Header.Set("X-Client-Id", "pd-client")
	req.SetPathValue("id", created.ID)
	rr = httptest.NewRecorder()
	h.ResumeWatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("resume %d %s", rr.Code, rr.Body.String())
	}
	var resumed priceDiffWatchDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &resumed); err != nil || resumed.Status != "active" || resumed.MinProfit != 8 {
		t.Fatalf("%s %v", rr.Body.String(), err)
	}
}
