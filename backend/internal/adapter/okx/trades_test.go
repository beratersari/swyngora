package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestRecentPrints_NotionalUsesContractSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v5/market/trades" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "0",
			"data": []map[string]string{{
				"side": "buy", "px": "100", "sz": "10", "ts": "1710000000000",
			}},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	c.ctVal["BTC-USDT-SWAP"] = 0.01
	got, err := c.GetRecentPrints(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Side != domain.TakerSideBuy || got[0].Quantity != 0.1 || got[0].Notional != 10 {
		t.Fatalf("%+v", got)
	}
	flow, err := c.GetTakerFlow(context.Background(), "BTCUSDT")
	if err != nil || flow == nil || flow.Exchange != domain.ExchangeOKX {
		t.Fatalf("%v %+v", err, flow)
	}
}

func TestGetBasisQuote_MarkVersusIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v5/market/ticker":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "0", "data": []map[string]string{{"last": "101", "ts": "1710000000000"}},
			})
		case "/api/v5/public/mark-price":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "0", "data": []map[string]string{{"markPx": "100.5", "ts": "1710000000000"}},
			})
		case "/api/v5/market/index-tickers":
			if r.URL.Query().Get("instId") != "BTC-USDT" {
				t.Fatalf("index %s", r.URL.Query().Get("instId"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "0", "data": []map[string]string{{"idxPx": "100"}},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetBasisQuote(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Exchange != domain.ExchangeOKX || got.FuturesLast != 101 || got.FuturesMark != 100.5 || got.SpotIndex != 100 {
		t.Fatalf("%+v", got)
	}
}
