package coinbase

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestListSpotMarkets_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "products") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]any{
				{
					"product_id": "BTC-USD", "price": "100", "price_percentage_change_24h": "1.5",
					"volume_24h": "10", "approximate_quote_24h_volume": "1000",
					"base_currency_id": "BTC", "quote_currency_id": "USD",
					"status": "online", "trading_disabled": false, "is_disabled": false, "product_type": "SPOT",
				},
				{
					"product_id": "ETH-USD", "price": "50", "status": "delisted", "trading_disabled": true,
					"base_currency_id": "ETH", "quote_currency_id": "USD", "product_type": "SPOT",
				},
			},
			"pagination": map[string]any{"has_next": false, "next_cursor": ""},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{
		BaseURL: srv.URL, ExchangeURL: srv.URL, HTTPClient: srv.Client(),
		SpotMarketCache: cache.New[[]domain.SpotMarket](time.Minute),
	})
	list, err := c.ListSpotMarkets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Symbol != "BTC-USD" {
		t.Fatalf("%+v", list)
	}
}

func TestGetCandles_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "candles") {
			http.NotFound(w, r)
			return
		}
		// newest first: [time, low, high, open, close, volume]
		_ = json.NewEncoder(w).Encode([][]any{
			{1700003600, 1.0, 3.0, 2.0, 2.5, 10.0},
			{1700000000, 0.5, 2.0, 1.0, 1.5, 5.0},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{
		BaseURL: srv.URL, ExchangeURL: srv.URL, HTTPClient: srv.Client(),
		CandleCache: cache.New[[]domain.Candle](time.Minute),
	})
	out, err := c.GetCandles(context.Background(), domain.CandleQuery{
		Symbol: "BTC-USD", Interval: domain.Interval1h, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Open != "1" {
		t.Fatalf("chronological: %+v", out)
	}
}

func TestListProductTags_Empty(t *testing.T) {
	c := NewClient(Options{})
	tags, err := c.ListProductTags(context.Background())
	if err != nil || len(tags) != 0 {
		t.Fatalf("%v %v", tags, err)
	}
}
