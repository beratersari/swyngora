package binance

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

func TestStripStableQuoteSuffix(t *testing.T) {
	cases := map[string]string{
		"BTCUSDT":   "BTC",
		"ETHUSDC":   "ETH",
		"TUSDUSDT":  "TUSD",
		"FDUSDUSDT": "FDUSD",
		"BFUSDUSDT": "BFUSD",
		"RLUSDUSDT": "RLUSD",
		"BTC":       "BTC",
		"TUSD":      "TUSD",
		"RLUSD":     "RLUSD",
		"BFUSD":     "BFUSD",
		"WBTC":      "WBTC",
	}
	for in, want := range cases {
		if got := stripStableQuoteSuffix(in); got != want {
			t.Fatalf("stripStableQuoteSuffix(%q)=%q want %q", in, got, want)
		}
	}
}

func TestRefresh_PopulatesAllSupplyFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "marketing/symbol/list") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "000000",
			"success": true,
			"data": []map[string]any{
				{
					"name": "BTC", "fullName": "Bitcoin", "symbol": "BTCUSDT",
					"tags": []string{"Payments"},
					"circulatingSupply": 19_800_000, "totalSupply": 19_800_000, "maxSupply": 21_000_000,
					"price": 64000.5, "infiniteSupply": false,
				},
				{
					"name": "ETH", "fullName": "Ethereum", "symbol": "ETHUSDT",
					"tags": []string{"Layer1_Layer2"},
					"circulatingSupply": 120_000_000, "totalSupply": 120_000_000, "maxSupply": nil,
					"price": 3000.0, "infiniteSupply": false,
				},
				{
					"name": "ACE", "fullName": "Fusionist", "symbol": "ACEUSDT",
					"tags": []string{"Gaming"},
					"circulatingSupply": 42_000_000, "totalSupply": 100_000_000, "maxSupply": 150_000_000,
					"price": 0.09,
				},
				// Non-crypto: must not enter supply cache.
				{
					"name": "NVDAB", "fullName": "NVIDIA", "symbol": "NVDABUSDT",
					"tags": []string{"bStocks"},
					"circulatingSupply": 32797, "totalSupply": 32797, "maxSupply": nil,
					"price": 200.0,
				},
				{
					"name": "TSLAB", "fullName": "Tesla", "symbol": "TSLABUSDT",
					"tags": []string{"bStocks"},
					"circulatingSupply": 1, "totalSupply": 1,
					"price": 300.0,
				},
				{
					"name": "PAXG", "fullName": "PAX Gold", "symbol": "PAXGUSDT",
					"tags": []string{"tCommodities"},
					"circulatingSupply": 100, "totalSupply": 100,
					"price": 2000.0,
				},
				{
					"name": "EMPTY", "fullName": "Empty", "symbol": "EMPTYUSDT",
					"circulatingSupply": 0, "totalSupply": nil, "maxSupply": nil,
				},
			},
		})
	}))
	defer srv.Close()

	supCache := cache.New[*domain.AssetSupply](time.Hour)
	c := NewClient(Options{
		ProductBaseURL: srv.URL,
		HTTPClient:     srv.Client(),
		SupplyCache:    supCache,
	})

	n, err := c.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("stored=%d want 3", n)
	}

	btc, err := c.GetSupply(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if btc.Source != "binance" || btc.Name != "Bitcoin" {
		t.Fatalf("btc=%+v", btc)
	}
	if btc.CirculatingSupply == nil || *btc.CirculatingSupply != 19_800_000 {
		t.Fatalf("circ=%v", btc.CirculatingSupply)
	}
	if btc.TotalSupply == nil || *btc.TotalSupply != 19_800_000 {
		t.Fatalf("total=%v", btc.TotalSupply)
	}
	if btc.MaxSupply == nil || *btc.MaxSupply != 21_000_000 {
		t.Fatalf("max=%v", btc.MaxSupply)
	}
	if btc.CurrentPriceUSD == nil || *btc.CurrentPriceUSD != 64000.5 {
		t.Fatalf("usd=%v", btc.CurrentPriceUSD)
	}

	eth, err := c.GetSupply(context.Background(), "ETH")
	if err != nil {
		t.Fatal(err)
	}
	if eth.TotalSupply == nil || *eth.TotalSupply != 120_000_000 {
		t.Fatalf("eth total=%v", eth.TotalSupply)
	}
	if eth.MaxSupply != nil {
		t.Fatalf("eth max should be nil, got %v", *eth.MaxSupply)
	}

	// Pair form
	tusdPair, err := c.GetSupply(context.Background(), "BTCUSDT")
	if err != nil || tusdPair.Asset != "BTC" {
		t.Fatalf("pair err=%v asset=%v", err, tusdPair)
	}

	ace, err := c.GetSupply(context.Background(), "ace")
	if err != nil || ace.MaxSupply == nil || *ace.MaxSupply != 150_000_000 {
		t.Fatalf("ace=%v err=%v", ace, err)
	}

	for _, asset := range []string{"NVDAB", "TSLAB", "PAXG"} {
		if _, err := c.GetSupply(context.Background(), asset); err == nil || !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected %s excluded from supply cache, err=%v", asset, err)
		}
	}
}

func TestGetSupply_CacheOnlyNoNetwork(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "should not call", 500)
	}))
	defer srv.Close()

	supCache := cache.New[*domain.AssetSupply](time.Hour)
	circ, total, max := 21_000_000.0, 21_000_000.0, 21_000_000.0
	supCache.Set("BTC", &domain.AssetSupply{
		Asset: "BTC", CirculatingSupply: &circ, TotalSupply: &total, MaxSupply: &max, Source: "binance",
	})
	c := NewClient(Options{
		ProductBaseURL: srv.URL,
		HTTPClient:     srv.Client(),
		SupplyCache:    supCache,
	})

	got, err := c.GetSupply(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalSupply == nil || *got.TotalSupply != total || got.MaxSupply == nil || *got.MaxSupply != max {
		t.Fatalf("got=%+v", got)
	}
	if hits != 0 {
		t.Fatalf("GetSupply must not call network, hits=%d", hits)
	}
}

func TestGetSupply_ReturnsIndependentFloats(t *testing.T) {
	supCache := cache.New[*domain.AssetSupply](time.Hour)
	orig := 21_000_000.0
	supCache.Set("BTC", &domain.AssetSupply{
		Asset:             "BTC",
		CirculatingSupply: &orig,
		TotalSupply:       domain.CloneFloatPtr(ptrF(orig)),
		MaxSupply:         domain.CloneFloatPtr(ptrF(21_000_000)),
		CurrentPriceUSD:   domain.CloneFloatPtr(ptrF(100)),
		Source:            "binance",
	})
	c := NewClient(Options{SupplyCache: supCache})

	s1, err := c.GetSupply(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	*s1.CirculatingSupply = 1
	*s1.TotalSupply = 1
	*s1.MaxSupply = 1
	s2, err := c.GetSupply(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if *s2.CirculatingSupply != orig || *s2.TotalSupply != orig || *s2.MaxSupply != 21_000_000 {
		t.Fatalf("cache mutated: %+v", s2)
	}
}

func TestGetSupply_BareUSDNamedBasesNotMangled(t *testing.T) {
	supCache := cache.New[*domain.AssetSupply](time.Hour)
	for asset, circ := range map[string]float64{"RLUSD": 1e10, "BFUSD": 2e9, "TUSD": 5e8} {
		c := circ
		supCache.Set(asset, &domain.AssetSupply{Asset: asset, CirculatingSupply: &c, Source: "binance"})
	}
	poison := 1.0
	supCache.Set("RL", &domain.AssetSupply{Asset: "RL", CirculatingSupply: &poison, Source: "binance"})
	supCache.Set("BF", &domain.AssetSupply{Asset: "BF", CirculatingSupply: &poison, Source: "binance"})

	c := NewClient(Options{SupplyCache: supCache})
	for asset, want := range map[string]float64{"RLUSD": 1e10, "rlusd": 1e10, "BFUSD": 2e9, "TUSDUSDT": 5e8} {
		got, err := c.GetSupply(context.Background(), asset)
		if err != nil {
			t.Fatalf("%s: %v", asset, err)
		}
		if got.CirculatingSupply == nil || *got.CirculatingSupply != want {
			t.Fatalf("%s supply=%v want %v (asset=%s)", asset, got.CirculatingSupply, want, got.Asset)
		}
	}
}

func TestGetSupply_NotFound(t *testing.T) {
	c := NewClient(Options{SupplyCache: cache.New[*domain.AssetSupply](time.Hour)})
	_, err := c.GetSupply(context.Background(), "NOPE")
	if err == nil || !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetSupply_EmptyAsset(t *testing.T) {
	c := NewClient(Options{SupplyCache: cache.New[*domain.AssetSupply](time.Hour)})
	_, err := c.GetSupply(context.Background(), "  ")
	if err == nil || !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestRefresh_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", 502)
	}))
	defer srv.Close()
	c := NewClient(Options{
		ProductBaseURL: srv.URL,
		HTTPClient:     srv.Client(),
		SupplyCache:    cache.New[*domain.AssetSupply](time.Hour),
	})
	_, err := c.Refresh(context.Background())
	if err == nil || !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("err=%v", err)
	}
}

func ptrF(f float64) *float64 { return &f }
