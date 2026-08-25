package handler

import (
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
