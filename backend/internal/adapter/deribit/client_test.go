package deribit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetOptionQuotes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != bookPath || r.URL.Query().Get("currency") != "BTC" || r.URL.Query().Get("kind") != "option" {
			t.Fatalf("path=%s q=%s", r.URL.Path, r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"result": []map[string]any{
				{
					"instrument_name":  "BTC-27MAR26-100000-C",
					"open_interest":    12.5,
					"volume":           1.2,
					"volume_usd":       120000,
					"mark_iv":          48.2,
					"underlying_price": 100000,
				},
				{
					"instrument_name":  "BTC-PERPETUAL",
					"open_interest":    1,
					"volume":           1,
					"underlying_price": 100000,
				},
			},
		})
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client(), CacheTTL: time.Minute})
	got, err := c.GetOptionQuotes(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Side != domain.OptionSideCall || got[0].MarkIV != 48.2 {
		t.Fatalf("%+v", got)
	}
	// Cache hit — server would fail a second book call with a different path.
	got2, err := c.GetOptionQuotes(context.Background(), "btcusdt")
	if err != nil || len(got2) != 1 {
		t.Fatalf("cache %v %v", got2, err)
	}
}

func TestGetOptionsDVOLAndIndex(t *testing.T) {
	t.Parallel()
	last := 51.0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case tickerPath:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					"last_price": last, "mark_price": 50.5, "index_price": 50.4,
					"stats": map[string]any{"price_change": -2.0},
				},
			})
		case dvolHistPath:
			now := time.Now().UTC().UnixMilli()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					"data": [][]float64{
						{float64(now - 25*3600*1000), 52, 53, 51, 52.5},
						{float64(now), 51, 51, 50, 51},
					},
				},
			})
		case indexPath:
			if r.URL.Query().Get("index_name") != "btc_usd" {
				t.Fatalf("index=%s", r.URL.Query().Get("index_name"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"index_price": 98765.4},
			})
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client(), CacheTTL: time.Minute})
	dvol, err := c.GetOptionsDVOL(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if dvol.Value != 51 || !dvol.HasChange {
		t.Fatalf("%+v", dvol)
	}
	if dvol.Change24h > -1.4 || dvol.Change24h < -1.6 {
		t.Fatalf("change=%v", dvol.Change24h)
	}
	px, err := c.GetOptionsIndex(context.Background(), "BTC")
	if err != nil || px != 98765.4 {
		t.Fatalf("index %v %v", px, err)
	}
}

func TestGetOptionQuotes_UpstreamError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	c := New(Options{BaseURL: srv.URL, HTTPClient: srv.Client(), CacheTTL: time.Minute})
	if _, err := c.GetOptionQuotes(context.Background(), "ETH"); err == nil {
		t.Fatal("expected error")
	}
}
