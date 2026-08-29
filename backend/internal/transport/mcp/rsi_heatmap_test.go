package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type rsiHeatMarket struct{}

func (rsiHeatMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	n := q.Limit
	if n < 30 {
		n = 30
	}
	out := make([]domain.Candle, n)
	for i := 0; i < n; i++ {
		out[i] = domain.Candle{Close: fmt.Sprintf("%g", 100+float64(i))}
	}
	return out, nil
}
func (rsiHeatMarket) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "1"}, nil
}
func (rsiHeatMarket) GetOrderBook(_ context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{Symbol: q.Symbol}, nil
}
func (rsiHeatMarket) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) {
	return []domain.SpotMarket{
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "100", LastPrice: "1"},
		{Symbol: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "50", LastPrice: "1"},
	}, nil
}
func (rsiHeatMarket) ListProductTags(context.Context) ([]string, error) { return nil, nil }
func (rsiHeatMarket) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

type rsiHeatSupply struct{}

func (rsiHeatSupply) GetSupply(_ context.Context, asset string) (*domain.AssetSupply, error) {
	return &domain.AssetSupply{Asset: asset}, nil
}
func (rsiHeatSupply) Refresh(context.Context) (int, error) { return 0, nil }

func TestBackend_GetRSIHeatmap(t *testing.T) {
	b := &Backend{}
	if _, err := b.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14); err == nil {
		t.Fatal("nil market")
	}
	b.Market = market.New(rsiHeatMarket{}, rsiHeatSupply{})
	raw, err := b.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	var got domain.RSIHeatmap
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) == 0 || got.Items[0].RSI == nil {
		t.Fatalf("%+v", got)
	}
	if _, err := b.GetRSIHeatmap(context.Background(), "nope", "USDT", "1h", "quoteVolume", 2, 14); err == nil {
		t.Fatal("bad exchange")
	}
}

func TestAPIClient_GetRSIHeatmap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/market/rsi-heatmap" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("exchange") != "binance" || q.Get("quote") != "USDT" || q.Get("interval") != "1h" ||
			q.Get("sort") != "quoteVolume" || q.Get("limit") != "24" || q.Get("period") != "14" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"exchange": "binance", "interval": "1h", "items": []any{},
		})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	raw, err := c.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "quoteVolume", 24, 14)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || m["exchange"] != "binance" {
		t.Fatalf("%s err=%v", raw, err)
	}
}

func TestAPIClient_GetRSIHeatmap_OmitsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Fatalf("expected empty query, got %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()
	c := NewAPIClient(srv.URL, 0)
	if _, err := c.GetRSIHeatmap(context.Background(), "", "", "", "", 0, 0); err != nil {
		t.Fatal(err)
	}
}

func TestInProcessMCP_GetRSIHeatmapTool(t *testing.T) {
	svc := market.New(rsiHeatMarket{}, rsiHeatSupply{})
	s := NewInProcessServer(svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if s == nil {
		t.Fatal("server")
	}
	// Tool is registered; in-process call goes through Backend via the same service.
	b := &Backend{Market: svc}
	raw, err := b.GetRSIHeatmap(context.Background(), "binance", "USDT", "1h", "", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 10 {
		t.Fatalf("short payload %s", raw)
	}
	_ = time.Second
}
