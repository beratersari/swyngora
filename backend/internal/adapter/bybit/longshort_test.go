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

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetLongShortSeries_OK(t *testing.T) {
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/market/account-ratio" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("period") != "5min" || r.URL.Query().Get("category") != "linear" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0, "retMsg": "OK",
			"result": map[string]any{"list": []map[string]any{
				{"buyRatio": "0.5836", "sellRatio": "0.4164", "timestamp": strconv.FormatInt(now.UnixMilli(), 10)},
				{"buyRatio": "0.55", "sellRatio": "0.45", "timestamp": strconv.FormatInt(now.Add(-5*time.Minute).UnixMilli(), 10)},
			}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetLongShortSeries(context.Background(), "BTCUSDT", 24)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.LongShare != 0.5836 || got.Current.Ratio < 1.40 {
		t.Fatalf("current %+v", got.Current)
	}
	if len(got.History) != 1 || got.History[0].LongShare != 0.55 {
		t.Fatalf("hist %+v", got.History)
	}
}

func TestGetLongShortSeries_InvalidSymbol(t *testing.T) {
	c := NewClient(Options{BaseURL: "http://example.invalid"})
	_, err := c.GetLongShortSeries(context.Background(), "", 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
