package bybit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestListLiquidationHistory_PerSymbol(t *testing.T) {
	from := time.Date(2026, 8, 30, 17, 0, 0, 0, time.UTC)
	to := from.Add(15 * time.Minute)
	seen := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/market/recent-liquidation" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("category") != "linear" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		sym := r.URL.Query().Get("symbol")
		seen[sym]++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0,
			"result": map[string]any{
				"list": []map[string]any{
					{"T": from.Add(time.Minute).UnixMilli(), "s": sym, "S": "Buy", "v": "0.5", "p": "64000"},
				},
			},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeBybit, Symbols: []string{"BTCUSDT", "ETHUSDT"}, From: from, To: to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasCoveredRange() || len(got.Events) != 2 {
		t.Fatalf("%+v", got)
	}
	if seen["BTCUSDT"] != 1 || seen["ETHUSDT"] != 1 {
		t.Fatalf("seen %+v", seen)
	}
	if got.Events[0].Exchange != domain.ExchangeBybit || got.Events[0].Side != domain.LiquidationSideLong {
		t.Fatalf("event %+v", got.Events[0])
	}
}

func TestListLiquidationHistory_404Uncovered(t *testing.T) {
	from := time.Date(2026, 8, 30, 17, 0, 0, 0, time.UTC)
	to := from.Add(5 * time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeBybit, Symbols: []string{"BTCUSDT"}, From: from, To: to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.HasCoveredRange() || len(got.Events) != 0 {
		t.Fatalf("404 must leave the gap %+v", got)
	}
}

func TestListLiquidationHistory_NoSymbolsUncovered(t *testing.T) {
	c := NewClient(Options{BaseURL: "http://127.0.0.1:1", HTTPClient: http.DefaultClient})
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeBybit,
		From:     time.Unix(1, 0),
		To:       time.Unix(1, 0).Add(time.Minute),
	})
	if err != nil || got.HasCoveredRange() {
		t.Fatalf("%+v %v", got, err)
	}
}
