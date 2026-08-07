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

func TestPortfolioHTTP_CashMovements(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "http-cash", "startingBalance": 10000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	body, _ = json.Marshal(map[string]any{"clientId": "http-cash", "amount": 1500, "note": "bonus"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/deposits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.Deposit(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("deposit %d %s", rr.Code, rr.Body.String())
	}
	var dep map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &dep)
	pf := dep["portfolio"].(map[string]any)
	if pf["cashBalance"].(float64) != 11500 || pf["totalPnL"].(float64) != 0 {
		t.Fatalf("after deposit %+v", pf)
	}

	body, _ = json.Marshal(map[string]any{"clientId": "http-cash", "amount": 500})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/withdrawals", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.Withdraw(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("withdraw %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/cash-movements?clientId=http-cash", nil)
	rr = httptest.NewRecorder()
	h.ListCashMovements(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list %d %s", rr.Code, rr.Body.String())
	}
	var hist map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &hist)
	if int(hist["total"].(float64)) != 3 {
		t.Fatalf("hist %+v", hist)
	}
}

func TestPortfolioHTTP_Performance(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{
		"clientId": "http-perf", "startingBalance": 10000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/performance?clientId=http-perf&period=1d", nil)
	rr = httptest.NewRecorder()
	h.GetPerformance(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("perf %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["period"] != "1d" {
		t.Fatalf("%v", got)
	}
	if got["startEquity"].(float64) != 10000 {
		t.Fatalf("start=%v", got["startEquity"])
	}
	if got["endEquity"].(float64) != 10000 {
		t.Fatalf("end=%v", got["endEquity"])
	}
	if got["changeAmount"].(float64) != 0 {
		t.Fatalf("chg=%v", got["changeAmount"])
	}
	pts, ok := got["points"].([]any)
	if !ok || len(pts) < 1 {
		t.Fatalf("points=%v", got["points"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/performance?clientId=http-perf&period=nope", nil)
	rr = httptest.NewRecorder()
	h.GetPerformance(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad period status=%d", rr.Code)
	}
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

func TestPortfolioHTTP_AllocationBasketRebalance(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{"clientId": "http-bsk", "startingBalance": 10000})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("pf %d %s", rr.Code, rr.Body.String())
	}
	body, _ = json.Marshal(map[string]any{
		"clientId": "http-bsk", "symbol": "BTCUSDT", "side": "buy", "quantity": 80,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.PlaceOrder(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("buy %d %s", rr.Code, rr.Body.String())
	}
	body, _ = json.Marshal(map[string]any{
		"clientId": "http-bsk", "name": "Core 50/30/20",
		"targets": []map[string]any{
			{"asset": "BTC", "weightPct": 50},
			{"asset": "ETH", "weightPct": 30},
			{"asset": "USDT", "weightPct": 20},
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/baskets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.CreateBasket(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("basket %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	id := created["id"].(string)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/baskets/"+id+"?clientId=http-bsk", nil)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.GetBasket(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d %s", rr.Code, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/baskets/"+id+"/rebalance?clientId=http-bsk", nil)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	h.RebalanceBasket(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("rebalance %d %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if int(out["tradeCount"].(float64)) == 0 {
		t.Fatalf("expected trades %v", out)
	}
}

func TestPortfolioHTTP_RiskLimits(t *testing.T) {
	h := newPortfolioHandler(t)
	body, _ := json.Marshal(map[string]any{"clientId": "http-risk", "startingBalance": 10000})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("pf %d", rr.Code)
	}
	w := 30.0
	body, _ = json.Marshal(map[string]any{"maxAssetWeightPct": w})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/portfolio/risk-limits?clientId=http-risk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.PutRiskLimits(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("put %d %s", rr.Code, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/portfolio/risk-limits?clientId=http-risk", nil)
	rr = httptest.NewRecorder()
	h.GetRiskLimits(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	lim := got["limits"].(map[string]any)
	if lim["maxAssetWeightPct"].(float64) != 30 {
		t.Fatalf("%v", lim)
	}
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/portfolio/risk-limits?clientId=http-risk", nil)
	rr = httptest.NewRecorder()
	h.DeleteRiskLimits(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("del %d", rr.Code)
	}
}