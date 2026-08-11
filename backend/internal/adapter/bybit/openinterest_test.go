package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetOpenInterestSeries_OK(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	t5 := now.Add(-5 * time.Minute).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/market/tickers":
			if r.URL.Query().Get("category") != "linear" || r.URL.Query().Get("symbol") != "BTCUSDT" {
				t.Fatalf("query %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK", "time": now.UnixMilli(),
				"result": map[string]any{
					"list": []map[string]any{{
						"symbol":       "BTCUSDT",
						"openInterest": "80", "openInterestValue": "8000",
						"singleOpenInterest": "40", "singleOpenInterestValue": "4000",
						"lastPrice": "100",
					}},
				},
			})
		case "/v5/market/open-interest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK",
				"result": map[string]any{
					"list": []map[string]any{{
						"openInterest": "70", "singleOpenInterest": "35",
						"timestamp": strconv.FormatInt(t5, 10),
					}},
				},
			})
		case "/v5/market/kline":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK",
				"result": map[string]any{
					"list": [][]string{{strconv.FormatInt(t5, 10), "99", "101", "98", "100", "1", "100"}},
				},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Options{
		BaseURL:           srv.URL,
		HTTPClient:        srv.Client(),
		OpenInterestCache: cache.New[*domain.OpenInterestSeries](time.Minute),
	})
	got, err := c.GetOpenInterestSeries(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.Contracts != 40 || got.Current.Value != 4000 {
		t.Fatalf("current %+v (want single-sided 40 / 4000)", got.Current)
	}
	if len(got.History) != 1 || got.History[0].Contracts != 35 || got.History[0].Value != 3500 {
		t.Fatalf("hist %+v (want single-sided 35)", got.History)
	}
}

func TestGetOpenInterestSeries_HalvesBilateralWhenSingleMissing(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	t5 := now.Add(-5 * time.Minute).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/market/tickers":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK", "time": now.UnixMilli(),
				"result": map[string]any{
					"list": []map[string]any{{
						"symbol": "BTCUSDT", "openInterest": "80", "openInterestValue": "8000", "lastPrice": "100",
					}},
				},
			})
		case "/v5/market/open-interest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK",
				"result": map[string]any{
					"list": []map[string]any{{
						"openInterest": "70", "timestamp": strconv.FormatInt(t5, 10),
					}},
				},
			})
		case "/v5/market/kline":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK",
				"result": map[string]any{
					"list": [][]string{{strconv.FormatInt(t5, 10), "99", "101", "98", "100", "1", "100"}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Options{
		BaseURL:           srv.URL,
		HTTPClient:        srv.Client(),
		OpenInterestCache: cache.New[*domain.OpenInterestSeries](time.Minute),
	})
	got, err := c.GetOpenInterestSeries(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.Contracts != 40 || got.Current.Value != 4000 {
		t.Fatalf("fallback current %+v", got.Current)
	}
	if len(got.History) != 1 || got.History[0].Contracts != 35 || got.History[0].Value != 3500 {
		t.Fatalf("fallback hist %+v", got.History)
	}
}

func TestBybitSingleSideOI(t *testing.T) {
	got, ok := bybitSingleSideOI("28888.268", "57776.536")
	if !ok || got != 28888.268 {
		t.Fatalf("prefer single got %v ok=%v", got, ok)
	}
	got, ok = bybitSingleSideOI("", "80")
	if !ok || got != 40 {
		t.Fatalf("halve both-sides got %v ok=%v", got, ok)
	}
	if _, ok := bybitSingleSideOI("", ""); ok {
		t.Fatal("empty")
	}
}

func TestGetOpenInterestSeries_InvalidSymbol(t *testing.T) {
	c := NewClient(Options{BaseURL: "http://example.invalid"})
	_, err := c.GetOpenInterestSeries(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestCloseAt_Bucket(t *testing.T) {
	at := time.Date(2026, 8, 11, 16, 3, 0, 0, time.UTC)
	bucket := at.Truncate(5 * time.Minute).UnixMilli()
	px := map[int64]float64{bucket: 42}
	if got := closeAt(px, at); got != 42 {
		t.Fatalf("got %v", got)
	}
}
