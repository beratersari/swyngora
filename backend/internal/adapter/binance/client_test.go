package binance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
