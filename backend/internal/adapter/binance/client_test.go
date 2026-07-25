package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetCandles_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/klines" {
			t.Fatalf("path %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("symbol") != "BTCUSDT" || q.Get("interval") != "1h" {
			t.Fatalf("query %v", q)
		}
		_ = json.NewEncoder(w).Encode([][]any{
			{1_700_000_000_000, "100", "110", "90", "105", "12.5", 1_700_003_600_000, "1300", 42, "0", "0", "0"},
		})
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	candles, err := c.GetCandles(context.Background(), domain.CandleQuery{
		Symbol:   "btcusdt",
		Interval: domain.Interval1h,
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 {
		t.Fatalf("len=%d", len(candles))
	}
	if candles[0].Open != "100" || candles[0].Close != "105" || candles[0].TradeCount != 42 {
		t.Fatalf("candle=%+v", candles[0])
	}
}

func TestGetCandles_InvalidInterval(t *testing.T) {
	c := NewClient(Options{BaseURL: "http://example.invalid"})
	_, err := c.GetCandles(context.Background(), domain.CandleQuery{
		Symbol:   "BTCUSDT",
		Interval: "2y",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetTicker24h_OK_AndCache(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol":             "ETHUSDT",
			"priceChange":        "1",
			"priceChangePercent": "2.5",
			"lastPrice":          "3000",
			"openPrice":          "2900",
			"highPrice":          "3100",
			"lowPrice":           "2800",
			"volume":             "1000",
			"quoteVolume":        "3000000",
			"openTime":           1_700_000_000_000,
			"closeTime":          1_700_086_400_000,
			"count":              99,
		})
	}))
	defer srv.Close()

	tickers := cache.New[*domain.Ticker24h](time.Minute)
	c := NewClient(Options{
		BaseURL:     srv.URL,
		HTTPClient:  srv.Client(),
		TickerCache: tickers,
	})
	t1, err := c.GetTicker24h(context.Background(), "ethusdt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GetTicker24h(context.Background(), "ETHUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("expected cache hit, upstream calls=%d", hits)
	}
	if t1.LastPrice != "3000" || t1.QuoteVolume != "3000000" {
		t.Fatalf("ticker=%+v", t1)
	}
}

// Test that GetTicker24h returns a copy. Mutation must not pollute the cache.
func TestGetTicker24h_ReturnsCopy_NotSharedPointer(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"symbol": "BTCUSDT", "lastPrice": "50000", "volume": "100", "quoteVolume": "5000000",
			"priceChange": "0", "priceChangePercent": "0", "openPrice": "49000",
			"highPrice": "51000", "lowPrice": "48000", "openTime": 1_700_000_000_000,
			"closeTime": 1_700_086_400_000, "count": 10,
		})
	}))
	defer srv.Close()

	tickers := cache.New[*domain.Ticker24h](time.Minute)
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client(), TickerCache: tickers})

	t1, err := c.GetTicker24h(context.Background(), "btcusdt")
	if err != nil {
		t.Fatal(err)
	}

	t1.LastPrice = "HACKED"

	t2, err := c.GetTicker24h(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if t2.LastPrice == "HACKED" {
		t.Fatal("ticker cache was polluted by mutation of returned value")
	}
	if hits != 1 {
		t.Fatalf("unexpected hits: %d", hits)
	}
}

func TestGetTicker24h_InvalidSymbol(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": -1121, "msg": "Invalid symbol."})
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetTicker24h(context.Background(), "NOPEUSDT")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetCandles_MissingSymbol(t *testing.T) {
	c := NewClient(Options{BaseURL: "http://example.invalid"})
	_, err := c.GetCandles(context.Background(), domain.CandleQuery{Interval: domain.Interval1h})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGet_RateLimited_429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetTicker24h(context.Background(), "BTCUSDT")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("err=%v", err)
	}
}

func TestGet_RateLimited_418(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetTicker24h(context.Background(), "BTCUSDT")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetCandles_CacheHit(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode([][]any{
			{1_700_000_000_000, "1", "2", "0.5", "1.5", "10", 1_700_003_600_000, "20", 5, "0", "0", "0"},
		})
	}))
	defer srv.Close()

	candles := cache.New[[]domain.Candle](time.Minute)
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client(), CandleCache: candles})
	q := domain.CandleQuery{Symbol: "BTCUSDT", Interval: domain.Interval1h, Limit: 5}
	a, err := c.GetCandles(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.GetCandles(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("upstream hits=%d", hits)
	}
	// Returned slice must be a copy (mutating caller result must not poison cache).
	a[0].Open = "MUTATED"
	if b[0].Open == "MUTATED" {
		t.Fatal("cache returned shared slice")
	}
}

func TestListSpotMarkets_JoinFilterCache(t *testing.T) {
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		switch {
		case r.URL.Path == "/api/v3/exchangeInfo":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"symbols": []map[string]any{
					{
						"symbol": "BTCUSDT", "status": "TRADING", "baseAsset": "BTC", "quoteAsset": "USDT",
						"isSpotTradingAllowed": true, "permissions": []string{"SPOT"},
					},
					{
						"symbol": "ETHUSDT", "status": "TRADING", "baseAsset": "ETH", "quoteAsset": "USDT",
						"isSpotTradingAllowed": true, "permissions": []string{"SPOT"},
					},
					{
						"symbol": "NVDABUSDT", "status": "TRADING", "baseAsset": "NVDAB", "quoteAsset": "USDT",
						"isSpotTradingAllowed": true, "permissions": []string{"SPOT"},
					},
					{
						"symbol": "TSLABUSDT", "status": "TRADING", "baseAsset": "TSLAB", "quoteAsset": "USDT",
						"isSpotTradingAllowed": true, "permissions": []string{"SPOT"},
					},
					{
						"symbol": "PAXGUSDT", "status": "TRADING", "baseAsset": "PAXG", "quoteAsset": "USDT",
						"isSpotTradingAllowed": true, "permissions": []string{"SPOT"},
					},
					{
						"symbol": "BTCUSDC", "status": "BREAK", "baseAsset": "BTC", "quoteAsset": "USDC",
						"isSpotTradingAllowed": false, "permissions": []string{"SPOT"},
					},
					{
						"symbol": "XYZUSDT", "status": "TRADING", "baseAsset": "XYZ", "quoteAsset": "USDT",
						"isSpotTradingAllowed": true, "permissions": []string{"MARGIN"},
					},
				},
			})
		case r.URL.Path == "/api/v3/ticker/24hr":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"symbol": "BTCUSDT", "lastPrice": "100", "volume": "10", "quoteVolume": "1000", "priceChangePercent": "1.5", "count": 9},
				{"symbol": "ETHUSDT", "lastPrice": "50", "volume": "20", "quoteVolume": "500", "priceChangePercent": "-2", "count": 3},
				{"symbol": "NVDABUSDT", "lastPrice": "200", "volume": "1", "quoteVolume": "200", "count": 1},
				{"symbol": "TSLABUSDT", "lastPrice": "300", "volume": "1", "quoteVolume": "300", "count": 1},
				{"symbol": "PAXGUSDT", "lastPrice": "2000", "volume": "1", "quoteVolume": "2000", "count": 1},
			})
		case strings.Contains(r.URL.Path, "get-products"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "000000", "success": true,
				"data": []map[string]any{
					{"s": "BTCUSDT", "b": "BTC", "tags": []string{"Payments"}},
					{"s": "ETHUSDT", "b": "ETH", "tags": []string{"Layer1_Layer2"}},
					{"s": "NVDABUSDT", "b": "NVDAB", "tags": []string{"bStocks"}},
					{"s": "TSLABUSDT", "b": "TSLAB", "tags": []string{"bStocks"}},
					{"s": "PAXGUSDT", "b": "PAXG", "tags": []string{"tCommodities"}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	spotCache := cache.New[[]domain.SpotMarket](time.Minute)
	c := NewClient(Options{
		BaseURL:         srv.URL,
		ProductBaseURL:  srv.URL,
		HTTPClient:      srv.Client(),
		SpotMarketCache: spotCache,
	})
	list, err := c.ListSpotMarkets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d want 2 crypto spot only: %+v", len(list), list)
	}
	bySym := map[string]domain.SpotMarket{}
	for _, m := range list {
		bySym[m.Symbol] = m
	}
	if _, ok := bySym["NVDABUSDT"]; ok {
		t.Fatal("expected bStocks NVDABUSDT excluded")
	}
	if _, ok := bySym["TSLABUSDT"]; ok {
		t.Fatal("expected bStocks TSLABUSDT excluded")
	}
	if _, ok := bySym["PAXGUSDT"]; ok {
		t.Fatal("expected tCommodities PAXGUSDT excluded")
	}
	if bySym["BTCUSDT"].QuoteVolume != "1000" || bySym["ETHUSDT"].LastPrice != "50" {
		t.Fatalf("metrics=%+v", bySym)
	}

	// Cache: second call does not hit upstream again (joined list still warm).
	list2, err := c.ListSpotMarkets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits["/api/v3/exchangeInfo"] != 1 || hits["/api/v3/ticker/24hr"] != 1 {
		t.Fatalf("hits=%v", hits)
	}
	list[0].Symbol = "MUTATED"
	if list2[0].Symbol == "MUTATED" {
		t.Fatal("shared slice from cache")
	}
}

// After the short price cache expires, only ticker/24hr is re-fetched — not exchangeInfo or product catalog.
func TestListSpotMarkets_PriceRefreshDoesNotReloadMeta(t *testing.T) {
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		switch {
		case r.URL.Path == "/api/v3/exchangeInfo":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"symbols": []map[string]any{
					{
						"symbol": "BTCUSDT", "status": "TRADING", "baseAsset": "BTC", "quoteAsset": "USDT",
						"isSpotTradingAllowed": true, "permissions": []string{"SPOT"},
					},
				},
			})
		case r.URL.Path == "/api/v3/ticker/24hr":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"symbol": "BTCUSDT", "lastPrice": fmt.Sprintf("%d", hits["/api/v3/ticker/24hr"]), "volume": "1", "quoteVolume": "1", "count": 1},
			})
		case strings.Contains(r.URL.Path, "get-products"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "000000", "success": true,
				"data": []map[string]any{
					{"s": "BTCUSDT", "b": "BTC", "tags": []string{"Payments"}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Very short joined-list TTL so the second call rebuilds prices.
	spotCache := cache.New[[]domain.SpotMarket](30 * time.Millisecond)
	c := NewClient(Options{
		BaseURL:         srv.URL,
		ProductBaseURL:  srv.URL,
		HTTPClient:      srv.Client(),
		SpotMarketCache: spotCache,
	})

	if _, err := c.ListSpotMarkets(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	list, err := c.ListSpotMarkets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits["/api/v3/exchangeInfo"] != 1 {
		t.Fatalf("exchangeInfo should stay cached, hits=%d", hits["/api/v3/exchangeInfo"])
	}
	if hits["/api/v3/ticker/24hr"] != 2 {
		t.Fatalf("ticker should refresh, hits=%d", hits["/api/v3/ticker/24hr"])
	}
	// product catalog path may be full path under test server
	prodHits := 0
	for p, n := range hits {
		if strings.Contains(p, "get-products") {
			prodHits += n
		}
	}
	if prodHits != 1 {
		t.Fatalf("product catalog should stay cached, hits=%d map=%v", prodHits, hits)
	}
	if len(list) != 1 || list[0].LastPrice != "2" {
		t.Fatalf("want refreshed price 2, got %+v", list)
	}
}
