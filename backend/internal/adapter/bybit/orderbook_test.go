package bybit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetOrderBook_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/market/orderbook" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0,
			"retMsg":  "OK",
			"result": map[string]any{
				"s":  "ETHUSDT",
				"b":  [][]string{{"3000", "2"}},
				"a":  [][]string{{"3001", "1"}},
				"ts": 1_700_000_000_000,
				"u":  9,
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	book, err := c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "ETHUSDT", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if book.Symbol != "ETHUSDT" || len(book.Bids) != 1 || book.Bids[0].Price != 3000 {
		t.Fatalf("%+v", book)
	}
}
