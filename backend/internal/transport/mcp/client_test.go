package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
