package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type httpCorrCandles struct {
	bars map[string][]domain.Candle
}

func (c httpCorrCandles) GetCandles(_ context.Context, exchange, symbol, _ string, _ int, _, _ *time.Time) ([]domain.Candle, error) {
	rows, ok := c.bars[exchange+"|"+symbol]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return rows, nil
}

func httpCorrPath(start float64, n int) []domain.Candle {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	px := start
	out := make([]domain.Candle, 0, n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			px *= 1.02
		} else {
			px *= 0.98
		}
		out = append(out, domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * 5 * time.Minute),
			Close:    strconv.FormatFloat(px, 'f', 8, 64),
		})
	}
	return out
}

func TestPortfolioHTTP_Correlation(t *testing.T) {
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf-corr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := portfolio.New(st, pfPx{}).WithPaperCosts(domain.ZeroTradingCosts)
	svc.SetCandleSource(httpCorrCandles{bars: map[string][]domain.Candle{
		"binance|BTCUSDT": httpCorrPath(100, 40),
		"binance|ETHUSDT": httpCorrPath(50, 40),
	}})
	h := NewPortfolioHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/correlation?clientId=http-corr&addSymbol=SOLUSDT&addNotional=nope", nil)
	rr := httptest.NewRecorder()
	h.GetPortfolioCorrelation(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad notional %d %s", rr.Code, rr.Body.String())
	}

	body, _ := json.Marshal(map[string]any{"clientId": "http-corr", "startingBalance": 10000})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	for _, symbol := range []string{"BTCUSDT", "ETHUSDT"} {
		body, _ = json.Marshal(map[string]any{
			"clientId": "http-corr", "exchange": "binance", "symbol": symbol, "side": "buy", "quantity": 1,
		})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		h.PlaceOrder(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("buy %s %d %s", symbol, rr.Code, rr.Body.String())
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/correlation?clientId=http-corr", nil)
	rr = httptest.NewRecorder()
	h.GetPortfolioCorrelation(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	together, _ := got["together"].(float64)
	if together < 0.9 {
		t.Fatalf("together %v body %s", got["together"], rr.Body.String())
	}
	if _, ok := got["missing"].([]any); !ok {
		t.Fatalf("missing should be a list: %s", rr.Body.String())
	}
	if _, ok := got["add"]; ok {
		t.Fatalf("an unread add was attached: %s", rr.Body.String())
	}
}
