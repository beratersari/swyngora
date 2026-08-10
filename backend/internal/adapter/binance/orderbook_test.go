package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetOrderBook_OK_AndCache(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path != "/api/v3/depth" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("symbol") != "BTCUSDT" {
			t.Fatalf("query %v", r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lastUpdateId": 1,
			"bids":         [][]string{{"100.00", "1.5"}, {"99.90", "2"}},
			"asks":         [][]string{{"100.10", "0.8"}, {"100.20", "3"}},
		})
	}))
	defer srv.Close()
	books := cache.New[*domain.RawOrderBook](time.Minute)
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client(), OrderBookCache: books})
	a, err := c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "btcusdt", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.GetOrderBook(context.Background(), domain.OrderBookQuery{Symbol: "BTCUSDT", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("cache hits=%d", hits)
	}
	if len(a.Bids) != 2 || a.Bids[0].Price != 100 || b.Asks[0].Quantity != 0.8 {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
}
