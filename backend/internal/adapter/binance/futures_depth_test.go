package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetFuturesDepth_USDMargin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/depth" || r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("%s %s", r.URL.Path, r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lastUpdateId": 1,
			"bids":         [][]string{{"65500", "2"}, {"65400", "10"}},
			"asks":         [][]string{{"65510", "1"}, {"65600", "4"}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetFuturesDepth(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Bids) != 2 || got.Bids[0].Price != 65500 || len(got.Asks) != 2 {
		t.Fatalf("%+v", got)
	}
}
