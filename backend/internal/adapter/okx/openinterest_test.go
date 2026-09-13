package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetOpenInterestSeries_OK(t *testing.T) {
	now := time.Date(2026, 9, 13, 16, 0, 0, 0, time.UTC)
	t5 := now.Add(-5 * time.Minute).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v5/public/open-interest":
			if r.URL.Query().Get("instId") != "BTC-USDT-SWAP" {
				t.Fatalf("query %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "0",
				"data": []map[string]any{{
					"oi": "1000", "oiCcy": "10", "oiUsd": "800000",
					"ts": "1690000000000",
				}},
			})
		case "/api/v5/rubik/stat/contracts/open-interest-history":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "0",
				"data": [][]string{{
					strconv.FormatInt(t5, 10), "900", "9", "720000",
				}},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client(), OpenInterestCache: cache.New[*domain.OpenInterestSeries](time.Minute)})
	got, err := c.GetOpenInterestSeries(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Exchange != domain.ExchangeOKX || got.Current.Contracts != 10 || got.Current.Value != 800000 {
		t.Fatalf("current %+v", got.Current)
	}
	if len(got.History) != 1 || got.History[0].Contracts != 9 || got.History[0].Value != 720000 {
		t.Fatalf("hist %+v", got.History)
	}
}
