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

func TestPortfolioHTTP_PendingOrders(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "http-pend", "startingBalance": 10000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	body, _ = json.Marshal(map[string]any{
		"clientId": "http-pend", "symbol": "BTCUSDT", "type": "limit_buy",
		"quantity": 1, "triggerPrice": 90,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.PlaceOrder(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("pending %d %s", rr.Code, rr.Body.String())
	}
	var place map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &place)
	order := place["order"].(map[string]any)
	id := order["id"].(string)
	if order["status"] != "open" {
		t.Fatalf("%v", order)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/orders?clientId=http-pend", nil)
	rr = httptest.NewRecorder()
	h.ListOrders(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list %d %s", rr.Code, rr.Body.String())
	}
	var list map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	if int(list["count"].(float64)) != 1 {
		t.Fatalf("%v", list)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/portfolio/orders/"+id+"?clientId=http-pend", nil)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.CancelOrder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("cancel %d %s", rr.Code, rr.Body.String())
	}

	// Cancel again → 404
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/portfolio/orders/"+id+"?clientId=http-pend", nil)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.CancelOrder(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("cancel2 %d %s", rr.Code, rr.Body.String())
	}
}

func TestPortfolioHTTP_GetAndAmendOrder(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "http-amd", "startingBalance": 10000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	body, _ = json.Marshal(map[string]any{
		"clientId": "http-amd", "symbol": "BTCUSDT", "type": "limit_buy",
		"quantity": 2, "triggerPrice": 90,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.PlaceOrder(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("pending %d %s", rr.Code, rr.Body.String())
	}
	var place map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &place)
	id := place["order"].(map[string]any)["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/orders/"+id+"?clientId=http-amd", nil)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.GetOrder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d %s", rr.Code, rr.Body.String())
	}
	var detail map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &detail)
	if detail["editable"] != true {
		t.Fatalf("editable %v", detail)
	}
	if detail["lastPrice"].(float64) != 100 {
		t.Fatalf("lastPrice %v", detail["lastPrice"])
	}
	amd := detail["amend"].(map[string]any)
	if amd["availableCashForOrder"].(float64) != 10000 {
		t.Fatalf("amend hints %v", amd)
	}

	body, _ = json.Marshal(map[string]any{"triggerPrice": 80.0, "remainingQuantity": 1.0})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/portfolio/orders/"+id+"?clientId=http-amd", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.AmendOrder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("amend %d %s", rr.Code, rr.Body.String())
	}
	var patched map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &patched)
	order := patched["order"].(map[string]any)
	if order["triggerPrice"].(float64) != 80 || order["remainingQuantity"].(float64) != 1 {
		t.Fatalf("patched order %v", order)
	}
	if order["id"] != id {
		t.Fatalf("id changed %v", order["id"])
	}
	pf := patched["portfolio"].(map[string]any)
	if pf["reservedCash"].(float64) != 80 {
		t.Fatalf("reserved %v", pf["reservedCash"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/orders/missing?clientId=http-amd", nil)
	req.SetPathValue("id", "missing")
	rr = httptest.NewRecorder()
	h.GetOrder(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing get %d", rr.Code)
	}
}

func TestPortfolioHTTP_CancelAllOrders(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{"clientId": "http-cxl", "startingBalance": 10000})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	for _, trig := range []float64{90, 80} {
		body, _ = json.Marshal(map[string]any{
			"clientId": "http-cxl", "symbol": "BTCUSDT", "type": "limit_buy",
			"quantity": 1, "triggerPrice": trig,
		})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		h.PlaceOrder(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("place %d %s", rr.Code, rr.Body.String())
		}
	}
	body, _ = json.Marshal(map[string]any{"clientId": "http-cxl", "symbol": "BTCUSDT"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders/cancel-all", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.CancelAllOrders(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("cancel-all %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if int(got["canceled"].(float64)) != 2 || got["scope"] != "market" {
		t.Fatalf("%v", got)
	}
	body, _ = json.Marshal(map[string]any{"clientId": "http-cxl"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders/cancel-all", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.CancelAllOrders(rr, req)
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if int(got["canceled"].(float64)) != 0 {
		t.Fatalf("second cancel-all %v", got)
	}
}

func TestPortfolioHTTP_RecurringBuyNamedInterval(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{"clientId": "http-rb", "startingBalance": 10000})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	body, _ = json.Marshal(map[string]any{
		"clientId": "http-rb", "symbol": "ETHUSDT", "name": "Every 12h ETH",
		"amount": 1500, "frequency": "interval", "intervalHours": 12,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/recurring-buys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.CreateRecurringBuy(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("plan %d %s", rr.Code, rr.Body.String())
	}
	var plan map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &plan)
	if plan["name"] != "Every 12h ETH" || plan["intervalHours"].(float64) != 12 {
		t.Fatalf("%v", plan)
	}
	id := plan["id"].(string)
	body, _ = json.Marshal(map[string]any{"name": "Renamed 12h"})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/portfolio/recurring-buys/"+id+"?clientId=http-rb", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.UpdateRecurringBuy(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch %d %s", rr.Code, rr.Body.String())
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &plan)
	if plan["name"] != "Renamed 12h" {
		t.Fatalf("%v", plan)
	}
}