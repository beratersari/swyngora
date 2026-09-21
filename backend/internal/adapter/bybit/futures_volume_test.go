package bybit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestListFuturesVolumeBars_Linear(t *testing.T) {
	open := time.Date(2026, 4, 2, 11, 55, 0, 0, time.UTC).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/market/kline" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("category") != "linear" || r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0,
			"result": map[string]any{
				"list": [][]string{{
					strconv.FormatInt(open, 10), "1", "1", "1", "1", "10", "987654.3",
				}},
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListFuturesVolumeBars(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].QuoteVolume != 987654.3 {
		t.Fatalf("%+v", got)
	}
}
