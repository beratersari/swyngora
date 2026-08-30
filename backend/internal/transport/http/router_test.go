package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type routerMarket struct{}

func (routerMarket) GetCandles(_ context.Context, _ domain.CandleQuery) ([]domain.Candle, error) {
	return []domain.Candle{{
		OpenTime: time.Unix(0, 0).UTC(), Open: "1", High: "1", Low: "1", Close: "1",
		Volume: "1", CloseTime: time.Unix(1, 0).UTC(), QuoteVolume: "1", TradeCount: 1,
	}}, nil
}

func (routerMarket) ListSpotMarkets(_ context.Context) ([]domain.SpotMarket, error) {
	return []domain.SpotMarket{{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1", Tags: []string{"Payments"}}}, nil
}

func (routerMarket) TagsByBase(_ context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func (routerMarket) ListProductTags(_ context.Context) ([]string, error) {
	return []string{"Meme", "Payments"}, nil
}

func (routerMarket) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{
		Symbol: symbol, LastPrice: "1", Volume: "2", QuoteVolume: "3",
		OpenTime: time.Unix(0, 0).UTC(), CloseTime: time.Unix(1, 0).UTC(),
	}, nil
}

func (routerMarket) GetOrderBook(_ context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{
		Symbol: q.Symbol,
		Bids:   []domain.PriceLevel{{Price: 1, Quantity: 1}},
		Asks:   []domain.PriceLevel{{Price: 1.1, Quantity: 1}},
	}, nil
}

type routerSupply struct{}

func (routerSupply) Refresh(context.Context) (int, error) { return 0, nil }

func (routerSupply) GetSupply(_ context.Context, asset string) (*domain.AssetSupply, error) {
	return &domain.AssetSupply{Asset: asset, Source: "test", AsOf: time.Unix(0, 0).UTC()}, nil
}

type routerHolders struct{}

func (routerHolders) GetHolders(_ context.Context, asset string) (*domain.AssetHolders, error) {
	return &domain.AssetHolders{Asset: asset, HolderCount: 1, Source: "test", AsOf: time.Unix(0, 0).UTC()}, nil
}

type rsiRouterMarket struct{ routerMarket }

func (rsiRouterMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	n := q.Limit
	if n < 30 {
		n = 30
	}
	out := make([]domain.Candle, n)
	for i := 0; i < n; i++ {
		out[i] = domain.Candle{
			OpenTime: time.Unix(int64(i)*3600, 0).UTC(),
			Close:    "100",
		}
		if i > 0 {
			out[i].Close = "101"
		}
	}
	return out, nil
}

func (rsiRouterMarket) ListSpotMarkets(_ context.Context) ([]domain.SpotMarket, error) {
	return []domain.SpotMarket{
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "9", LastPrice: "100"},
		{Symbol: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "8", LastPrice: "50"},
	}, nil
}

func TestRSIHeatmap_HTTPIntegration(t *testing.T) {
	svc := market.New(rsiRouterMarket{}, routerSupply{})
	h := NewRouterWithOptions(svc, nil, RouterOptions{RateLimitRPS: 0, RateLimitBurst: 0})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/rsi-heatmap?interval=1h&limit=2&sort=quoteVolume", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	items, _ := body["items"].([]any)
	if body["exchange"] != "binance" || len(items) != 2 {
		t.Fatalf("%s", rr.Body.String())
	}
	first, _ := items[0].(map[string]any)
	if first["symbol"] == nil || first["rsi"] == nil {
		t.Fatalf("row=%v", first)
	}
}

func TestNewRouter_RoutesAndCORS(t *testing.T) {
	svc := market.New(routerMarket{}, routerSupply{}).WithHolders(routerHolders{})
	h := NewRouterWithOptions(svc, nil, RouterOptions{RateLimitRPS: 0, RateLimitBurst: 0})

	paths := []struct {
		path  string
		want  int
		check func(t *testing.T, body []byte)
	}{
		{"/health", http.StatusOK, func(t *testing.T, body []byte) {
			var m map[string]any
			if err := json.Unmarshal(body, &m); err != nil || m["status"] != "ok" {
				t.Fatalf("health body=%s", body)
			}
		}},
		{"/api/v1/market/fx", http.StatusOK, func(t *testing.T, body []byte) {
			var m map[string]any
			if err := json.Unmarshal(body, &m); err != nil {
				t.Fatal(err)
			}
			if m["base"] != "USD" {
				t.Fatalf("fx=%v", m)
			}
			vq, _ := m["venueQuotes"].(map[string]any)
			if vq["bist"] != "TRY" || vq["binance"] != "USDT" {
				t.Fatalf("venueQuotes=%v", vq)
			}
		}},
		{"/api/v1/market/intervals", http.StatusOK, func(t *testing.T, body []byte) {
			var m map[string]any
			if err := json.Unmarshal(body, &m); err != nil {
				t.Fatal(err)
			}
			iv, ok := m["intervals"].([]any)
			if !ok || len(iv) == 0 {
				t.Fatalf("intervals=%v", m["intervals"])
			}
		}},
		{"/api/v1/market/candles?symbol=BTCUSDT&interval=1h&limit=1", http.StatusOK, nil},
		{"/api/v1/market/ticker/24h?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/orderbook?symbol=BTCUSDT&group=0.1", http.StatusOK, nil},
		{"/api/v1/market/orderbook/icebergs?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/orderbook/history?symbol=BTCUSDT", http.StatusBadGateway, nil},
		{"/api/v1/market/orderbook/combined?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/orderbook/impact?symbol=BTCUSDT&quantity=0.1", http.StatusOK, nil},
		{"/api/v1/market/orderbook/liquidity?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/orderbook/heatmap?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/liquidations?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/liquidations/overview", http.StatusOK, nil},
		{"/api/v1/market/open-interest?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/funding-rate?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/long-short-ratio?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/liquidation-hunt?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/liquidation-levels?symbol=all", http.StatusOK, nil},
		{"/api/v1/market/liquidation-hunt/heatmap?symbol=BTCUSDT&range=24h", http.StatusOK, nil},
		{"/api/v1/market/squeeze-risk?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/positioning?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/venue-divergence?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/taker-flow?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/cvd?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/volume-profile?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/vwap?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/around?symbol=BTCUSDT&at=2026-08-23T12:00:00Z", http.StatusOK, nil},
		{"/api/v1/market/around/compare?symbol=BTCUSDT&from=2026-08-22T12:00:00Z&to=2026-08-23T12:00:00Z", http.StatusOK, nil},
		{"/api/v1/market/around/moves?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/around/precursors?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/around/similar?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/absorption?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/liquidity-sweeps?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/basis?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/correlation?symbol=SOLUSDT", http.StatusOK, nil},
		{"/api/v1/market/breadth", http.StatusOK, nil},
		{"/api/v1/market/volatility?symbol=SOLUSDT", http.StatusOK, nil},
		{"/api/v1/market/snapshot?symbol=SOLUSDT", http.StatusOK, nil},
		{"/api/v1/market/levels?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/whales?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/supply?asset=BTC", http.StatusOK, nil},
		{"/api/v1/market/holders?asset=BTC", http.StatusOK, nil},
		{"/api/v1/market/tags", http.StatusOK, func(t *testing.T, body []byte) {
			var m map[string]any
			if err := json.Unmarshal(body, &m); err != nil {
				t.Fatal(err)
			}
			tags, ok := m["tags"].([]any)
			if !ok || len(tags) == 0 {
				t.Fatalf("tags=%v", m["tags"])
			}
		}},
		{"/api/v1/market/spot?limit=5&tag=Payments", http.StatusOK, nil},
		{"/api/v1/market/indicators?symbol=BTCUSDT&interval=1h&limit=30", http.StatusOK, nil},
		{"/api/v1/market/rsi-heatmap?intervals=1h&limit=3", http.StatusOK, nil},
		{"/api/v1/market/volume-surge?symbol=BTCUSDT", http.StatusOK, nil},
		{"/api/v1/market/volume-surge/scan", http.StatusOK, nil},
		{"/api/v1/realtime", http.StatusNotFound, nil}, // hub not wired in this test
		{"/nope", http.StatusNotFound, nil},
	}

	for _, tc := range paths {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Fatal("expected CORS header on all responses")
			}
			if tc.check != nil {
				tc.check(t, rr.Body.Bytes())
			}
		})
	}

	// Preflight through full stack
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/market/candles", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status=%d", rr.Code)
	}
}
