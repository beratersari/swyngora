package binance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetTakerFlow_FromAggTrades(t *testing.T) {
	now := time.Now().UTC()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/aggTrades" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"p": "100", "q": "2", "T": now.Add(-2 * time.Minute).UnixMilli(), "m": false},
			{"p": "100", "q": "8", "T": now.Add(-20 * time.Minute).UnixMilli(), "m": true},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.GetTakerFlow(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Windows[0].BuyNotional != 200 { // 2 * 100, measured
		t.Fatalf("5m buy %+v", got.Windows[0])
	}
	if got.Windows[0].Dominant != domain.TakerSideBuy {
		t.Fatalf("dom %s", got.Windows[0].Dominant)
	}
	again, err := c.GetTakerFlow(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if again.Windows[0].BuyNotional != 200 {
		t.Fatalf("second read doubled: %+v", again.Windows[0])
	}
}

func TestPullAggTrades_ContinuesFromLastID(t *testing.T) {
	now := time.Now().UTC()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Query().Get("fromId"))
		if r.URL.Query().Get("fromId") == "" {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"a": 10, "p": "100", "q": "1", "T": now.Add(-20 * time.Second).UnixMilli(), "m": false},
			})
			return
		}
		if r.URL.Query().Get("fromId") != "11" {
			t.Fatalf("fromId %s", r.URL.Query().Get("fromId"))
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"a": 11, "p": "100", "q": "3", "T": now.Add(-10 * time.Second).UnixMilli(), "m": true},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	if err := c.PullAggTrades(context.Background(), "BTCUSDT", 1); err != nil {
		t.Fatal(err)
	}
	if err := c.PullAggTrades(context.Background(), "BTCUSDT", 1); err != nil {
		t.Fatal(err)
	}
	var buy, sell float64
	for _, b := range c.TakerBook().Buckets(domain.ExchangeBinance, "BTCUSDT") {
		buy += b.BuyNotional
		sell += b.SellNotional
	}
	if buy != 100 || sell != 300 || len(calls) != 2 || calls[1] != "11" {
		t.Fatalf("calls=%v buy=%v sell=%v", calls, buy, sell)
	}
}

func TestGetTakerFlow_BadSymbol(t *testing.T) {
	c := NewClient(Options{FuturesBaseURL: "http://example.invalid"})
	_, err := c.GetTakerFlow(context.Background(), " ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}
