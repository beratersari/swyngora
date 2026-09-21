package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListFuturesVolumeBars_USDMargin(t *testing.T) {
	open := time.Date(2026, 4, 2, 11, 55, 0, 0, time.UTC).UnixMilli()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/klines" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" || r.URL.Query().Get("interval") != "5m" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([][]any{{
			open, "1", "1", "1", "1", "10", open + 299999, "1234567.8", 1,
		}})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListFuturesVolumeBars(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].QuoteVolume != 1234567.8 {
		t.Fatalf("%+v", got)
	}
	if !got[0].Time.Equal(time.UnixMilli(open).UTC()) {
		t.Fatalf("time %v", got[0].Time)
	}
}
