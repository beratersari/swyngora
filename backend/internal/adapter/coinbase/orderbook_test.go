package coinbase

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
		if r.URL.Path != "/products/BTC-USD/book" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("level") != "2" {
			t.Fatalf("query %v", r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sequence": 1,
			"bids":     [][]string{{"65000", "0.4", "1"}},
			"asks":     [][]string{{"65010", "0.2", "2"}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{ExchangeURL: srv.URL, HTTPClient: srv.Client()})
	book, err := c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}
	if len(book.Bids) != 1 || book.Bids[0].Quantity != 0.4 {
		t.Fatalf("%+v", book)
	}
}
