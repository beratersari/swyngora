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

type compareBooks struct{}

func (compareBooks) GetRawOrderBook(_ context.Context, exchange, _ string) (*domain.RawOrderBook, error) {
	switch exchange {
	case "binance":
		return &domain.RawOrderBook{
			Asks: []domain.PriceLevel{{Price: 100, Quantity: 1}, {Price: 110, Quantity: 5}},
			Bids: []domain.PriceLevel{{Price: 100, Quantity: 5}},
		}, nil
	case "bybit":
		return &domain.RawOrderBook{
			Asks: []domain.PriceLevel{{Price: 102, Quantity: 5}},
			Bids: []domain.PriceLevel{{Price: 99, Quantity: 5}},
		}, nil
	default:
		return &domain.RawOrderBook{
			Asks: []domain.PriceLevel{{Price: 100, Quantity: 5}},
			Bids: []domain.PriceLevel{{Price: 99, Quantity: 5}},
		}, nil
	}
}

func TestPortfolioHTTP_ExecutionCompareDoesNotFill(t *testing.T) {
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := portfolio.New(st, pfPx{}).WithPaperCosts(domain.TradingCostFor)
	svc.SetBookSource(compareBooks{})
	h := NewPortfolioHandler(svc)

	body, _ := json.Marshal(map[string]any{"clientId": "http-exec", "startingBalance": 10000})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	body, _ = json.Marshal(map[string]any{
		"clientId": "http-exec", "exchange": "binance", "symbol": "BTCUSDT",
		"side": "buy", "quantity": 2, "spreadMinutes": 10, "sliceCount": 2,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders/execution-compare", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.CompareExecution(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("compare %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	rows, _ := got["rows"].([]any)
	if len(rows) != 3 {
		t.Fatalf("rows %+v", got)
	}
	spread, _ := rows[2].(map[string]any)
	if spread["status"] != "filled" || spread["assumption"] != "refill" {
		t.Fatalf("spread %+v", spread)
	}
	saved, _ := spread["savedVsMarket"].(float64)
	if saved <= 0 {
		t.Fatalf("saved %v", saved)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio?clientId=http-exec", nil)
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d %s", rr.Code, rr.Body.String())
	}
	var view map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &view)
	if view["cashBalance"].(float64) != 10000 {
		t.Fatalf("cash changed: %+v", view["cashBalance"])
	}
}
