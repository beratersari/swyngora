package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestListFuturesVolumeBars_Swap(t *testing.T) {
	open := time.Date(2026, 4, 2, 11, 55, 0, 0, time.UTC).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/market/candles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("instId") != "BTC-USDT-SWAP" || r.URL.Query().Get("bar") != "5m" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": [][]string{{
				strconv.FormatInt(open, 10), "1", "1", "1", "1", "10", "10", "555000.5", "1",
			}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListFuturesVolumeBars(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].QuoteVolume != 555000.5 {
		t.Fatalf("%+v", got)
	}
}
