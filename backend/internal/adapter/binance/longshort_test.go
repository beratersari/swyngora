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

func TestGetLongShortSeries_OK(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/futures/data/globalLongShortAccountRatio" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("period") != "5m" || r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"longAccount": "0.60", "shortAccount": "0.40", "longShortRatio": "1.5", "timestamp": now.Add(-5 * time.Minute).UnixMilli()},
			{"longAccount": "0.63", "shortAccount": "0.37", "longShortRatio": "1.7027", "timestamp": now.UnixMilli()},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetLongShortSeries(context.Background(), "btc-usd", 24)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.LongShare != 0.63 || got.Current.Ratio < 1.70 {
		t.Fatalf("current %+v", got.Current)
	}
	if len(got.History) != 1 || got.History[0].LongShare != 0.60 {
		t.Fatalf("hist %+v", got.History)
	}
}

func TestGetLongShortSeries_InvalidSymbol(t *testing.T) {
	c := NewClient(Options{FuturesBaseURL: "http://example.invalid"})
	_, err := c.GetLongShortSeries(context.Background(), "  ", 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
