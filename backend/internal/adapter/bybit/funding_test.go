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

func TestGetFundingSeries_OK(t *testing.T) {
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	next := now.Add(time.Hour).UnixMilli()
	t8 := now.Add(-8 * time.Hour).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/market/tickers":
			if r.URL.Query().Get("category") != "linear" {
				t.Fatalf("query %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK", "time": now.UnixMilli(),
				"result": map[string]any{"list": []map[string]any{{
					"fundingRate":         "0.00008804",
					"nextFundingTime":     strconv.FormatInt(next, 10),
					"fundingIntervalHour": "8",
				}}},
			})
		case "/v5/market/funding/history":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK",
				"result": map[string]any{"list": []map[string]any{
					{"fundingRate": "0.00003597", "fundingRateTimestamp": strconv.FormatInt(t8, 10)},
				}},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetFundingSeries(context.Background(), "BTCUSDT", 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current.Rate != 0.00008804 || got.IntervalHours != 8 {
		t.Fatalf("%+v", got)
	}
	if len(got.History) != 1 || got.History[0].Rate != 0.00003597 {
		t.Fatalf("hist %+v", got.History)
	}
}

func TestGetFundingSeries_InvalidSymbol(t *testing.T) {
	c := NewClient(Options{BaseURL: "http://example.invalid"})
	_, err := c.GetFundingSeries(context.Background(), "", 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
