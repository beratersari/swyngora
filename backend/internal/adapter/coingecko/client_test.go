package coingecko

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetSupply_WellKnown(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		circ := 19_800_000.0
		total := 19_800_000.0
		max := 21_000_000.0
		price := 64000.0
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "bitcoin",
			"symbol": "btc",
			"name":   "Bitcoin",
			"market_data": map[string]any{
				"circulating_supply": circ,
				"total_supply":       total,
				"max_supply":         max,
				"current_price":      map[string]any{"usd": price},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	sup, err := c.GetSupply(context.Background(), "btc")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/v3/coins/bitcoin" {
		t.Fatalf("path=%s", path)
	}
	if sup.Asset != "BTC" || sup.MaxSupply == nil || *sup.MaxSupply != 21_000_000 {
		t.Fatalf("supply=%+v", sup)
	}
	if sup.CirculatingSupply == nil || *sup.CirculatingSupply != 19_800_000 {
		t.Fatalf("circulating=%v", sup.CirculatingSupply)
	}
	if sup.Source != "coingecko" {
		t.Fatalf("source=%s", sup.Source)
	}
}

func TestGetSupply_SearchFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v3/search":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"coins": []map[string]any{
					{"id": "foo-coin", "symbol": "FOO", "name": "Foo"},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/api/v3/coins/"):
			circ := 1_000.0
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "foo-coin",
				"symbol": "foo",
				"name":   "Foo",
				"market_data": map[string]any{
					"circulating_supply": circ,
					"total_supply":       circ,
					"max_supply":         nil,
					"current_price":      map[string]any{"usd": 1.5},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Options{
		BaseURL:     srv.URL,
		HTTPClient:  srv.Client(),
		SymbolCache: cache.New[string](time.Hour),
		SupplyCache: cache.New[*domain.AssetSupply](time.Hour),
	})
	sup, err := c.GetSupply(context.Background(), "FOO")
	if err != nil {
		t.Fatal(err)
	}
	if sup.ProviderID != "foo-coin" || sup.MaxSupply != nil {
		t.Fatalf("supply=%+v", sup)
	}

	// Second call should be fully cached (no panic if server dies).
	srv.Close()
	sup2, err := c.GetSupply(context.Background(), "FOO")
	if err != nil {
		t.Fatal(err)
	}
	if sup2.Name != "Foo" {
		t.Fatalf("cached=%+v", sup2)
	}
}

func TestGetSupply_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"coins": []any{}})
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetSupply(context.Background(), "ZZZNOPE")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeAsset_PairSuffix(t *testing.T) {
	if got := normalizeAsset("btcusdt"); got != "BTC" {
		t.Fatalf("got %s", got)
	}
	if got := normalizeAsset("ETH"); got != "ETH" {
		t.Fatalf("got %s", got)
	}
}

func TestGetSupply_EmptyAsset(t *testing.T) {
	c := NewClient(Options{BaseURL: "http://example.invalid"})
	_, err := c.GetSupply(context.Background(), "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetSupply_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := c.GetSupply(context.Background(), "BTC")
	if !errors.Is(err, domain.ErrNotFound) && !errors.Is(err, domain.ErrRateLimited) {
		// well-known BTC hits coins endpoint directly
	}
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("err=%v want rate limited", err)
	}
}
