package okx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetFuturesDepth_ConvertsContracts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/market/books" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("instId") != "BTC-USDT-SWAP" {
			t.Fatalf("inst %s", r.URL.Query().Get("instId"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": []map[string]any{{
				"ts":   "1710000000000",
				"bids": [][]string{{"100", "10", "0", "1"}},
				"asks": [][]string{{"101", "5", "0", "1"}},
			}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	c.ctVal["BTC-USDT-SWAP"] = 0.01
	got, err := c.GetFuturesDepth(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || len(got.Bids) != 1 || len(got.Asks) != 1 {
		t.Fatalf("%+v", got)
	}
	if got.Bids[0].Price != 100 || got.Bids[0].Quantity != 0.1 {
		t.Fatalf("bid %+v", got.Bids[0])
	}
	if got.Asks[0].Quantity != 0.05 {
		t.Fatalf("ask %+v", got.Asks[0])
	}
	if got.Bids[0].Quantity*got.Bids[0].Price != 10 {
		t.Fatalf("notional %v", got.Bids[0])
	}
}

func TestGetFuturesDepth_MissingContractSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": "0", "data": []any{}})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetFuturesDepth(context.Background(), "BTCUSDT")
	if !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("%v", err)
	}
}
