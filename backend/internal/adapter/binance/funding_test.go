package binance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetFundingSeries_OK(t *testing.T) {
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	next := now.Add(time.Hour).UnixMilli()
	lastSettled := now.Add(-7 * time.Hour)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fapi/v1/premiumIndex":
			if r.URL.Query().Get("symbol") != "BTCUSDT" {
				t.Fatalf("symbol %s", r.URL.Query().Get("symbol"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lastFundingRate": "0.00009996", "markPrice": "63500",
				"nextFundingTime": next, "time": now.UnixMilli(),
			})
		case "/fapi/v1/fundingRate":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"fundingRate": "0.00008", "fundingTime": lastSettled.UnixMilli(), "markPrice": "64000"},
				{"fundingRate": "0.00002", "fundingTime": lastSettled.Add(-8 * time.Hour).UnixMilli(), "markPrice": "64100"},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetFundingSeries(context.Background(), "btc-usd", 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.Rate != 0.00009996 || !got.Current.Predicted {
		t.Fatalf("current %+v", got.Current)
	}
	if got.IntervalHours != 8 || len(got.History) != 2 || got.History[0].Rate != 0.00008 {
		t.Fatalf("%+v", got)
	}
	again, err := c.GetFundingSeries(context.Background(), "BTCUSDT", 12)
	if err != nil || again.Current.Rate != 0.00009996 {
		t.Fatalf("cache %v %+v", err, again)
	}
}

func TestGetFundingSeries_InvalidSymbol(t *testing.T) {
	c := NewClient(Options{FuturesBaseURL: "http://example.invalid"})
	_, err := c.GetFundingSeries(context.Background(), "  ", 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
