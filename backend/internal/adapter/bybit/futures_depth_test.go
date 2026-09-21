package bybit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetFuturesDepth_Linear(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/market/orderbook" || r.URL.Query().Get("category") != "linear" {
			t.Fatalf("%s %s", r.URL.Path, r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0,
			"result": map[string]any{
				"u": 3,
				"b": [][]string{{"65500", "2"}},
				"a": [][]string{{"65510", "1"}},
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetFuturesDepth(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Bids) != 1 || got.Bids[0].Price != 65500 || len(got.Asks) != 1 {
		t.Fatalf("%+v", got)
	}
}
