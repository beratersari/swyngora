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

func TestGetOpenInterestSeries_OK_AndCache(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/fapi/v1/openInterest":
			if r.URL.Query().Get("symbol") != "BTCUSDT" {
				t.Fatalf("symbol %s", r.URL.Query().Get("symbol"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"openInterest": "110.5", "symbol": "BTCUSDT", "time": now.UnixMilli(),
			})
		case "/fapi/v1/premiumIndex":
			_ = json.NewEncoder(w).Encode(map[string]any{"markPrice": "100", "symbol": "BTCUSDT"})
		case "/futures/data/openInterestHist":
			if r.URL.Query().Get("period") != "5m" {
				t.Fatalf("period %s", r.URL.Query().Get("period"))
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"sumOpenInterest": "100", "sumOpenInterestValue": "10000", "timestamp": now.Add(-5 * time.Minute).UnixMilli()},
				{"sumOpenInterest": "105", "sumOpenInterestValue": "10500", "timestamp": now.Add(-time.Minute).UnixMilli()},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Options{
		FuturesBaseURL:    srv.URL,
		HTTPClient:        srv.Client(),
		OpenInterestCache: cache.New[*domain.OpenInterestSeries](time.Minute),
	})
	got, err := c.GetOpenInterestSeries(context.Background(), "btc-usd")
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || got.Exchange != domain.ExchangeBinance {
		t.Fatalf("%+v", got)
	}
	if got.Current.Contracts != 110.5 || got.Current.Value != 11050 {
		t.Fatalf("current %+v", got.Current)
	}
	if len(got.History) != 2 || got.History[0].Contracts != 100 {
		t.Fatalf("hist %+v", got.History)
	}
	firstHits := hits
	again, err := c.GetOpenInterestSeries(context.Background(), "BTCUSDT")
	if err != nil || again.Current.Contracts != 110.5 {
		t.Fatalf("cache %v %+v", err, again)
	}
	if hits != firstHits {
		t.Fatalf("cache should skip upstream, hits %d → %d", firstHits, hits)
	}
}

func TestGetOpenInterestSeries_InvalidSymbol(t *testing.T) {
	c := NewClient(Options{FuturesBaseURL: "http://example.invalid"})
	_, err := c.GetOpenInterestSeries(context.Background(), "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetOpenInterestSeries_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":-1121,"msg":"Invalid symbol."}`, http.StatusBadRequest)
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetOpenInterestSeries(context.Background(), "NOPEUSDT")
	if err == nil {
		t.Fatal("expected error")
	}
}
