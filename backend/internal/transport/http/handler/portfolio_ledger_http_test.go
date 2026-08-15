package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type mutPx struct {
	mu     sync.Mutex
	prices map[string]string
}

func (p *mutPx) set(symbol, price string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.prices == nil {
		p.prices = map[string]string{}
	}
	p.prices[symbol] = price
}

func (p *mutPx) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	price := "100"
	if p.prices != nil {
		if v := p.prices[symbol]; v != "" {
			price = v
		}
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: price}, nil
}

func TestPortfolioHTTP_IsolatedCloseDoesNotGoNegative(t *testing.T) {
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	px := &mutPx{prices: map[string]string{"BTCUSDT": "100"}}
	h := NewPortfolioHandler(portfolio.New(st, px).WithPaperCosts(domain.ZeroTradingCosts))

	post := func(path string, body map[string]any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Client-Id", "http-iso")
		rr := httptest.NewRecorder()
		switch path {
		case "/api/v1/portfolio":
			h.Create(rr, req)
		case "/api/v1/portfolio/margin/orders":
			h.PlaceMarginOrder(rr, req)
		default:
			h.CloseMarginPosition(rr, req)
		}
		return rr
	}

	rr := post("/api/v1/portfolio", map[string]any{"clientId": "http-iso", "startingBalance": 1000})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	rr = post("/api/v1/portfolio/margin/orders", map[string]any{
		"clientId": "http-iso", "symbol": "BTCUSDT", "side": "long", "type": "market",
		"quantity": 40, "leverage": 5,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("open %d %s", rr.Code, rr.Body.String())
	}
	var opened map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &opened)
	posID := opened["position"].(map[string]any)["id"].(string)

	px.set("BTCUSDT", "10")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/margin/positions/"+posID+"/close", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Id", "http-iso")
	req.SetPathValue("id", posID)
	rr = httptest.NewRecorder()
	h.CloseMarginPosition(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("close %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio", nil)
	req.Header.Set("X-Client-Id", "http-iso")
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d %s", rr.Code, rr.Body.String())
	}
	var view map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &view)
	cash := view["cashBalance"].(float64)
	if cash < 0 {
		t.Fatalf("HTTP isolated close left negative cash %v", cash)
	}
	if cash < 199 || cash > 201 {
		t.Fatalf("cash=%v want ~200 unassigned", cash)
	}
}
