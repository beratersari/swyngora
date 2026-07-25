package coingecko

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetSupply_CacheOnly(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "should not call", 500)
	}))
	defer srv.Close()

	supCache := cache.New[*domain.AssetSupply](time.Hour)
	circ := 21_000_000.0
	supCache.Set("BTC", &domain.AssetSupply{Asset: "BTC", CirculatingSupply: &circ, Source: "coingecko"})

	c := NewClient(Options{BaseURL: srv.URL, HTTPClient: srv.Client(), SupplyCache: supCache})
	got, err := c.GetSupply(context.Background(), "btc")
	if err != nil {
		t.Fatal(err)
	}
	if got.CirculatingSupply == nil || *got.CirculatingSupply != circ {
		t.Fatalf("got=%+v", got)
	}
	if hits != 0 {
		t.Fatalf("GetSupply must not hit network, hits=%d", hits)
	}

	_, err = c.GetSupply(context.Background(), "ETH")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("miss should be not found, err=%v", err)
	}
}

func TestRefresh_PopulatesCache(t *testing.T) {
	pages := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/coins/markets" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		pages++
		page := r.URL.Query().Get("page")
		circ := 10.0
		max := 21.0
		price := 100.0
		if page == "1" {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "bitcoin", "symbol": "btc", "name": "Bitcoin",
					"current_price": price, "circulating_supply": circ, "total_supply": circ, "max_supply": max},
				{"id": "ethereum", "symbol": "eth", "name": "Ethereum",
					"current_price": 50.0, "circulating_supply": circ, "total_supply": circ, "max_supply": nil},
			})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	supCache := cache.New[*domain.AssetSupply](time.Hour)
	c := NewClient(Options{
		BaseURL:          srv.URL,
		HTTPClient:       srv.Client(),
		SupplyCache:      supCache,
		RefreshPages:     2,
		RefreshPageDelay: time.Millisecond,
	})
	n, err := c.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("stored=%d", n)
	}
	// markets pages + well-known ids batch
	if pages < 2 {
		t.Fatalf("pages=%d", pages)
	}

	btc, err := c.GetSupply(context.Background(), "BTC")
	if err != nil || btc.MaxSupply == nil {
		t.Fatalf("btc=%+v err=%v", btc, err)
	}
	eth, err := c.GetSupply(context.Background(), "ETH")
	if err != nil || eth.MaxSupply != nil {
		t.Fatalf("eth=%+v err=%v", eth, err)
	}
}

func TestGetSupply_EmptyAsset(t *testing.T) {
	c := NewClient(Options{SupplyCache: cache.New[*domain.AssetSupply](time.Hour)})
	_, err := c.GetSupply(context.Background(), "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

// Test that GetSupply returns a safe copy (cloned *float64 fields). Mutation of
// the returned value must not affect the cache or future callers.
func TestGetSupply_ReturnsIndependentFloats(t *testing.T) {
	supCache := cache.New[*domain.AssetSupply](time.Hour)
	orig := 21000000.0
	supCache.Set("BTC", &domain.AssetSupply{
		Asset:             "BTC",
		CirculatingSupply: &orig,
		CurrentPriceUSD:   domain.CloneFloatPtr(ptrForTest(100.0)),
	})

	c := NewClient(Options{SupplyCache: supCache})

	s1, err := c.GetSupply(context.Background(), "BTC")
	if err != nil || s1.CirculatingSupply == nil {
		t.Fatalf("s1 err=%v", err)
	}

	// Mutate the copy we received
	*s1.CirculatingSupply = 999999999.0
	if s1.CurrentPriceUSD != nil {
		*s1.CurrentPriceUSD = 999.0
	}

	// Cache and subsequent Get must be unaffected
	s2, err := c.GetSupply(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if s2.CirculatingSupply == nil || *s2.CirculatingSupply != orig {
		t.Fatalf("cache was mutated, got %v want %v", s2.CirculatingSupply, orig)
	}
	if s2.CurrentPriceUSD == nil || *s2.CurrentPriceUSD != 100.0 {
		t.Fatalf("price mutated in cache")
	}
}

func ptrForTest(f float64) *float64 { return &f }

func TestNormalizeAsset_PairSuffix(t *testing.T) {
	cases := map[string]string{
		"btcusdt":  "BTC",
		"BTCUSDT":  "BTC",
		"ethusdc":  "ETH",
		"BTC":      "BTC",
		"WBTC":     "WBTC", // must NOT become "W"
		"wbtcusdt": "WBTC",
		"RENBTC":   "RENBTC",
		"WBETH":    "WBETH",
		"BETH":     "BETH",
		"ETHBTC":   "ETHBTC", // ambiguous pair — do not strip crypto quotes
	}
	for in, want := range cases {
		if got := normalizeAsset(in); got != want {
			t.Fatalf("normalizeAsset(%q)=%q want %q", in, got, want)
		}
	}
}

func TestRefresh_WellKnownIncludesWBTC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids := r.URL.Query().Get("ids")
		if ids != "" {
			// well-known batch
			circ := 150_000.0
			price := 64000.0
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "wrapped-bitcoin", "symbol": "wbtc", "name": "Wrapped Bitcoin",
					"current_price": price, "circulating_supply": circ, "total_supply": circ, "max_supply": circ},
			})
			return
		}
		// empty top pages
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	supCache := cache.New[*domain.AssetSupply](time.Hour)
	c := NewClient(Options{
		BaseURL: srv.URL, HTTPClient: srv.Client(), SupplyCache: supCache,
		RefreshPages: 1, RefreshPageDelay: time.Millisecond,
	})
	if _, err := c.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	wbtc, err := c.GetSupply(context.Background(), "WBTC")
	if err != nil {
		t.Fatal(err)
	}
	if wbtc.CirculatingSupply == nil || *wbtc.CirculatingSupply != 150_000 {
		t.Fatalf("wbtc supply=%v", wbtc.CirculatingSupply)
	}
	// Sanity: mcap ~ 150k * 64k = 9.6B, not hundreds of trillions
	mcap := *wbtc.CirculatingSupply * *wbtc.CurrentPriceUSD
	if mcap > 1e12 {
		t.Fatalf("absurd mcap %v", mcap)
	}
}
