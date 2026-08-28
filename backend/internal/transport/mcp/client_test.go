package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIClient_GetTicker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/ticker/24h" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("symbol=%s", r.URL.Query().Get("symbol"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "lastPrice": "100", "exchange": "binance",
		})
	}))
	defer srv.Close()

	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetTicker(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["symbol"] != "BTCUSDT" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetLongShortRatio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/long-short-ratio" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"symbol": "BTCUSDT", "venues": []any{}})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetLongShortRatio(context.Background(), "all", "BTCUSDT", 24)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["symbol"] != "BTCUSDT" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetFundingArb(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/funding-arb" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" || r.URL.Query().Get("notional") != "10000" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"symbol": "BTCUSDT", "trade": map[string]any{"longExchange": "binance"}})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetFundingArb(context.Background(), "BTCUSDT", 10000, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["symbol"] != "BTCUSDT" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_ScanFundingArb(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/funding-arb/scan" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"hits": []any{}})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.ScanFundingArb(context.Background(), "USDT", 10000, 24, nil, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["hits"]; !ok {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetFundingArbHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/funding-arb/history" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" || r.URL.Query().Get("from") == "" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"symbol": "BTCUSDT", "runs": []any{}})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetFundingArbHistory(context.Background(), "BTCUSDT", "2026-08-01", "2026-08-08", 10000, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["symbol"] != "BTCUSDT" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_FundingArbWatches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/funding-arb/watches":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "w1", "symbol": "BTCUSDT", "minProfit": 10})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/funding-arb/watches":
			if r.URL.Query().Get("clientId") != "c1" {
				t.Fatalf("query=%s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"watches": []any{}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/funding-arb/watches/w1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "w1"})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/funding-arb/watches/w1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "w1", "minProfit": 12})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pause"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "w1", "status": "paused"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/resume"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "w1", "status": "active"})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/funding-arb/watches/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/funding-arb/signals":
			_ = json.NewEncoder(w).Encode(map[string]any{"signals": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	ctx := context.Background()
	raw, err := c.CreateFundingArbWatch(ctx, "c1", "BTCUSDT", 10000, 24, 10, "", 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var created map[string]any
	if err := json.Unmarshal(raw, &created); err != nil || created["id"] != "w1" {
		t.Fatalf("%s %v", raw, err)
	}
	if _, err := c.ListFundingArbWatches(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetFundingArbWatch(ctx, "c1", "w1"); err != nil {
		t.Fatal(err)
	}
	minP := 12.0
	if _, err := c.UpdateFundingArbWatch(ctx, "c1", "w1", nil, nil, &minP, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.PauseFundingArbWatch(ctx, "c1", "w1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ResumeFundingArbWatch(ctx, "c1", "w1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DeleteFundingArbWatch(ctx, "c1", "w1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListFundingArbSignals(ctx, "c1", "open", 10); err != nil {
		t.Fatal(err)
	}
}

func TestAPIClient_GetFundingRate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/funding-rate" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "venues": []any{map[string]any{"exchange": "binance", "current": map[string]any{"rate": "0.0001"}}},
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetFundingRate(context.Background(), "all", "BTCUSDT", 12)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["symbol"] != "BTCUSDT" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetOpenInterest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/open-interest" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "unit": "BTC", "current": map[string]any{"contracts": "100", "value": "10000"},
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetOpenInterest(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["symbol"] != "BTCUSDT" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetLiquidations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/liquidations" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "windows": []any{map[string]any{"window": "24h", "count": 2}},
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetLiquidations(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["symbol"] != "BTCUSDT" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetOrderBookHeatmap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/orderbook/heatmap" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" || r.URL.Query().Get("window") != "300" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "windowSeconds": 300, "columns": []any{map[string]any{"t": "2026-08-16T12:00:00Z", "mid": "100"}},
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetOrderBookHeatmap(context.Background(), "binance", "BTCUSDT", "", 300)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["windowSeconds"] != float64(300) {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetMarketLiquidity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/orderbook/liquidity" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "venueCount": 3, "market": map[string]any{"score": 70, "grade": "high"},
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetMarketLiquidity(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["venueCount"] != float64(3) {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_ScanPriceDiffQuotes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/price-diff/quote/scan" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("notional") != "10000" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"symbol": "BTCUSDT", "profitableCount": 1})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.ScanPriceDiffQuotes(context.Background(), "BTCUSDT", 10000, 0, 0.1, 0.6, 0.1, 0, 0.5, 10)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["profitableCount"] != float64(1) {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_QuotePriceDiff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/price-diff/quote" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("symbol") != "BTCUSDT" || q.Get("buyExchange") != "binance" || q.Get("notional") != "10000" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "averageBuyPrice": "100", "averageSellPrice": "101", "profitAfterFees": "8",
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.QuotePriceDiff(context.Background(), "BTCUSDT", "binance", "bybit", 10000, 0, 0.1, 0.1, 0)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["profitAfterFees"] != "8" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_EstimateOrderBookImpact(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/orderbook/impact" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("symbol") != "BTCUSDT" || q.Get("quantity") != "5" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "side": "buy", "averagePrice": "100.5", "exhausted": false,
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.EstimateOrderBookImpact(context.Background(), "all", "BTCUSDT", "buy", 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["averagePrice"] != "100.5" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetPortfolioPerformance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/portfolio/performance" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("clientId") != "c1" || r.URL.Query().Get("period") != "1m" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"period": "1m", "startEquity": 10000, "endEquity": 11000, "changeAmount": 1000, "changePct": 10,
			"points": []any{},
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetPortfolioPerformance(context.Background(), "c1", "1m")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["period"] != "1m" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_GetHolders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/holders" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("asset") != "BTC" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"asset": "BTC", "holderCount": 9})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetHolders(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["asset"] != "BTC" {
		t.Fatalf("%v", m)
	}
}

func TestAPIClient_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_argument","message":"bad"}}`))
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	_, err := c.GetSupply(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPrettyJSON(t *testing.T) {
	raw := json.RawMessage(`{"a":1}`)
	out := PrettyJSON(raw)
	if out == "" || out[0] != '{' {
		t.Fatalf("%q", out)
	}
}

func TestNewServer_RegistersTools(t *testing.T) {
	s := NewServer(ServerOptions{APIBaseURL: "http://127.0.0.1:9"})
	if s == nil {
		t.Fatal("nil server")
	}
}

func TestInProcessServerAndHTTPHandler(t *testing.T) {
	// Smoke: construct in-process server without market services using HTTP fallback is already tested.
	// NewHTTPHandler must be a valid http.Handler.
	s := NewServer(ServerOptions{APIBaseURL: "http://127.0.0.1:9"})
	h := NewHTTPHandler(s)
	if h == nil {
		t.Fatal("nil handler")
	}
}
