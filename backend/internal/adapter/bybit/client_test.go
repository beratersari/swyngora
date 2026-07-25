package bybit

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

func TestListSpotMarkets_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v5/market/instruments-info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0, "retMsg": "OK",
				"result": map[string]any{
					"list": []map[string]any{
						{"symbol": "BTCUSDT", "baseCoin": "BTC", "quoteCoin": "USDT", "status": "Trading"},
						{"symbol": "ETHUSDT", "baseCoin": "ETH", "quoteCoin": "USDT", "status": "Trading"},
					},
					"nextPageCursor": "",
				},
			})
		case r.URL.Path == "/v5/market/tickers":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retCode": 0,
				"result": map[string]any{
					"list": []map[string]any{
						{"symbol": "BTCUSDT", "lastPrice": "100", "volume24h": "10", "turnover24h": "1000", "price24hPcnt": "0.015", "highPrice24h": "110", "lowPrice24h": "90"},
						{"symbol": "ETHUSDT", "lastPrice": "50", "volume24h": "20", "turnover24h": "500", "price24hPcnt": "-0.02"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{
		BaseURL: srv.URL, HTTPClient: srv.Client(),
		SpotMarketCache: cache.New[[]domain.SpotMarket](time.Minute),
	})
	list, err := c.ListSpotMarkets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("%+v", list)
	}
	// 0.015 fraction → 1.5 percent
	if list[0].Symbol == "BTCUSDT" && list[0].PriceChangePercent != "1.5" {
		// order not guaranteed
	}
	by := map[string]domain.SpotMarket{}
	for _, m := range list {
		by[m.Symbol] = m
	}
	if by["BTCUSDT"].PriceChangePercent != "1.5" {
		t.Fatalf("pct=%s", by["BTCUSDT"].PriceChangePercent)
	}
}

func TestGetCandles_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0,
			"result": map[string]any{
				"list": [][]string{
					{"1700003600000", "2", "3", "1", "2.5", "10", "100"},
					{"1700000000000", "1", "2", "0.5", "1.5", "5", "50"},
				},
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{
		BaseURL: srv.URL, HTTPClient: srv.Client(),
		CandleCache: cache.New[[]domain.Candle](time.Minute),
	})
	out, err := c.GetCandles(context.Background(), domain.CandleQuery{
		Symbol: "BTCUSDT", Interval: domain.Interval1h, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Open != "1" {
		t.Fatalf("%+v", out)
	}
}
