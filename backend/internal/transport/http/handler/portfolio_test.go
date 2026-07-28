package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type pfPx struct{}

func (pfPx) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "100"}, nil
}

func newPortfolioHandler(t *testing.T) *PortfolioHandler {
	t.Helper()
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return NewPortfolioHandler(portfolio.New(st, pfPx{}))
}

func TestPortfolioHTTP_CreateOrderTrades(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "http-pf", "startingBalance": 10000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	body, _ = json.Marshal(map[string]any{
		"clientId": "http-pf", "symbol": "BTCUSDT", "side": "buy", "quantity": 1,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.PlaceOrder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("order %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio?clientId=http-pf", nil)
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d", rr.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["cashBalance"].(float64) != 9900 {
		t.Fatalf("%v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/trades?clientId=http-pf", nil)
	rr = httptest.NewRecorder()
	h.ListTrades(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("trades %d", rr.Code)
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if int(got["total"].(float64)) != 1 {
		t.Fatalf("%v", got)
	}
}