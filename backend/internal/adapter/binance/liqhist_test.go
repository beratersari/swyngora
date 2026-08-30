package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestListLiquidationHistory_VenueWide(t *testing.T) {
	from := time.Date(2026, 8, 30, 17, 0, 0, 0, time.UTC)
	to := from.Add(15 * time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/allForceOrders" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("symbol") != "" {
			t.Fatal("venue-wide must omit symbol")
		}
		if r.URL.Query().Get("startTime") == "" || r.URL.Query().Get("endTime") == "" {
			t.Fatalf("query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"symbol": "BTCUSDT", "side": "SELL", "price": "64000", "averagePrice": "63990", "origQty": "2", "executedQty": "2", "time": from.Add(time.Minute).UnixMilli()},
			{"symbol": "ETHUSDT", "side": "BUY", "price": "3000", "averagePrice": "3000", "origQty": "1", "executedQty": "1", "time": from.Add(2 * time.Minute).UnixMilli()},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeBinance, From: from, To: to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasCoveredRange() || len(got.Events) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Events[0].Side != domain.LiquidationSideLong || got.Events[1].Side != domain.LiquidationSideShort {
		t.Fatalf("sides %+v", got.Events)
	}
}

func TestListLiquidationHistory_PerSymbolFallback(t *testing.T) {
	from := time.Date(2026, 8, 30, 17, 0, 0, 0, time.UTC)
	to := from.Add(10 * time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("symbol") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": -1102, "msg": "Mandatory parameter 'symbol' was not sent."})
			return
		}
		sym := r.URL.Query().Get("symbol")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"symbol": sym, "side": "SELL", "price": "1", "averagePrice": "1", "origQty": "1", "executedQty": "1", "time": from.Add(time.Minute).UnixMilli()},
		})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeBinance, Symbols: []string{"BTCUSDT", "ETHUSDT"}, From: from, To: to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasCoveredRange() || len(got.Events) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestListLiquidationHistory_EmptyWindowCovered(t *testing.T) {
	from := time.Date(2026, 8, 30, 17, 0, 0, 0, time.UTC)
	to := from.Add(5 * time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeBinance, From: from, To: to,
	})
	if err != nil || len(got.Events) != 0 || !got.HasCoveredRange() {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestListLiquidationHistory_404LeavesUncovered(t *testing.T) {
	from := time.Date(2026, 8, 30, 17, 0, 0, 0, time.UTC)
	to := from.Add(5 * time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := NewClient(Options{FuturesBaseURL: srv.URL, HTTPClient: srv.Client()})
	got, err := c.ListLiquidationHistory(context.Background(), domain.LiquidationHistoryQuery{
		Exchange: domain.ExchangeBinance, Symbols: []string{"BTCUSDT"}, From: from, To: to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.HasCoveredRange() {
		t.Fatalf("404 must not cover %+v", got)
	}
}

func TestParseForceOrderHistory_IgnoresJunk(t *testing.T) {
	raw := []byte(`[{"symbol":"BTCUSDT","side":"SELL","price":"1","averagePrice":"1","origQty":"1","executedQty":"1","time":1},{"symbol":"X","side":"NOPE"}]`)
	got, err := parseForceOrderHistory(raw)
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
}
