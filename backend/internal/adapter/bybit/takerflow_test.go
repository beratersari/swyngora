package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetTakerFlow_SeedsRecentTrades(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/market/recent-trade" {
			t.Fatalf("path %s", r.URL.Path)
		}
		now := time.Now().UTC().UnixMilli()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0,
			"result": map[string]any{
				"list": []map[string]any{
					{"side": "Buy", "price": "100", "size": "2", "time": strconv.FormatInt(now, 10)},
					{"side": "Sell", "price": "100", "size": "1", "time": strconv.FormatInt(now-1000, 10)},
				},
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetTakerFlow(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Windows[0].BuyNotional != 200 || got.Windows[0].SellNotional != 100 {
		t.Fatalf("%+v", got.Windows[0])
	}
}

func TestGetSpotTakerBuckets_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/market/recent-trade" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("category") != "spot" {
			t.Fatalf("category %s", r.URL.Query().Get("category"))
		}
		now := time.Now().UTC().UnixMilli()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0,
			"result": map[string]any{
				"list": []map[string]any{
					{"side": "Buy", "price": "100", "size": "3", "time": strconv.FormatInt(now, 10)},
					{"side": "Sell", "price": "100", "size": "1", "time": strconv.FormatInt(now-1000, 10)},
				},
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetSpotTakerBuckets(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("empty")
	}
	var buy, sell float64
	for _, b := range got {
		buy += b.BuyNotional
		sell += b.SellNotional
	}
	if buy != 300 || sell != 100 {
		t.Fatalf("buy=%v sell=%v %+v", buy, sell, got)
	}
}

func TestGetTakerFlow_BadSymbol(t *testing.T) {
	c := NewClient(Options{})
	_, err := c.GetTakerFlow(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
