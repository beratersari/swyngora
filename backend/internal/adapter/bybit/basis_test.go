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

func TestGetBasisQuote_OK(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/market/tickers":
			if r.URL.Query().Get("category") == "spot" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"retCode": 0, "time": now.UnixMilli(),
					"result": map[string]any{"list": []map[string]any{{"lastPrice": "64005"}}},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "time": now.UnixMilli(),
				"result": map[string]any{"list": []map[string]any{{
					"lastPrice": "64100", "markPrice": "64080", "indexPrice": "64000",
				}}},
			})
		case "/v5/market/mark-price-kline", "/v5/market/index-price-kline":
			px := "64080"
			if r.URL.Path == "/v5/market/index-price-kline" {
				px = "64000"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0,
				"result": map[string]any{"list": [][]string{{
					strconv.FormatInt(now.Add(-time.Hour).UnixMilli(), 10), "0", "0", "0", px,
				}}},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetBasisQuote(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.FuturesLast != 64100 || got.SpotIndex != 64000 {
		t.Fatalf("%+v", got)
	}
	if len(got.History) != 1 {
		t.Fatalf("hist %+v", got.History)
	}
}

func TestGetBasisQuote_BadSymbol(t *testing.T) {
	c := NewClient(Options{})
	_, err := c.GetBasisQuote(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
