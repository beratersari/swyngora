package holders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeCMC struct {
	snap *domain.AssetHolders
	err  error
}

func (f fakeCMC) GetHolders(context.Context, string) (*domain.AssetHolders, error) {
	return f.snap, f.err
}

type fakeContracts struct {
	list []domain.AssetContract
	err  error
}

func (f fakeContracts) LookupContracts(context.Context, string) ([]domain.AssetContract, error) {
	return f.list, f.err
}

type fakeChain struct {
	snap *domain.AssetHolders
	err  error
	hits int
}

func (f *fakeChain) FromContracts(context.Context, string, []domain.AssetContract) (*domain.AssetHolders, error) {
	f.hits++
	return f.snap, f.err
}

func TestCascade_UsesCMCFirst(t *testing.T) {
	gt := &fakeChain{snap: &domain.AssetHolders{Asset: "UNI", HolderCount: 9, Source: "geckoterminal"}}
	c := New(Options{
		CMC:       fakeCMC{snap: &domain.AssetHolders{Asset: "UNI", HolderCount: 100, Source: "coinmarketcap"}},
		GeckoTerm: gt,
	})
	got, err := c.GetHolders(context.Background(), "UNI")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "coinmarketcap" || got.HolderCount != 100 {
		t.Fatalf("%+v", got)
	}
	if gt.hits != 0 {
		t.Fatalf("should not call fallbacks when CMC works, hits=%d", gt.hits)
	}
}

func TestCascade_FallsBackToCoinMetrics(t *testing.T) {
	gt := &fakeChain{snap: &domain.AssetHolders{Asset: "ETH", HolderCount: 1, Source: "geckoterminal"}}
	c := New(Options{
		CMC:       fakeCMC{err: domain.ErrHoldersUnpublished},
		Metrics:   fakeCMC{snap: &domain.AssetHolders{Asset: "ETH", HolderCount: 204_136_337, Source: "coinmetrics"}},
		GeckoTerm: gt,
	})
	got, err := c.GetHolders(context.Background(), "ETH")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "coinmetrics" || got.HolderCount != 204_136_337 {
		t.Fatalf("%+v", got)
	}
	if gt.hits != 0 {
		t.Fatal("should not need contract fallbacks when CoinMetrics works")
	}
}

type fakeProfile struct {
	contracts []domain.AssetContract
}

func (f fakeProfile) GetAssetProfile(context.Context, string) (*domain.AssetProfile, error) {
	return &domain.AssetProfile{Contracts: f.contracts}, nil
}

func TestCascade_UsesProfileContractsWhenGeckoSearchMissing(t *testing.T) {
	gt := &fakeChain{snap: &domain.AssetHolders{Asset: "LINK", HolderCount: 700_000, Source: "geckoterminal"}}
	c := New(Options{
		CMC:       fakeCMC{err: domain.ErrHoldersUnpublished},
		Metrics:   fakeCMC{err: domain.ErrHoldersUnpublished},
		Profile:   fakeProfile{contracts: []domain.AssetContract{{Chain: "ethereum", Address: "0x5149"}}},
		GeckoTerm: gt,
	})
	got, err := c.GetHolders(context.Background(), "LINK")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "geckoterminal" || gt.hits != 1 {
		t.Fatalf("%+v hits=%d", got, gt.hits)
	}
}

func TestCascade_FallsBackToGeckoTerminal(t *testing.T) {
	gt := &fakeChain{snap: &domain.AssetHolders{Asset: "PEPE", HolderCount: 50, Source: "geckoterminal"}}
	eth := &fakeChain{snap: &domain.AssetHolders{Asset: "PEPE", HolderCount: 1, Source: "ethplorer"}}
	c := New(Options{
		CMC:       fakeCMC{err: domain.ErrHoldersUnpublished},
		Contracts: fakeContracts{list: []domain.AssetContract{{Chain: "ethereum", Address: "0xabc"}}},
		GeckoTerm: gt,
		Ethplorer: eth,
		Cache:     cache.New[*domain.AssetHolders](time.Hour),
	})
	got, err := c.GetHolders(context.Background(), "PEPEUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "geckoterminal" || got.HolderCount != 50 {
		t.Fatalf("%+v", got)
	}
	if eth.hits != 0 {
		t.Fatal("ethplorer should not run after GT succeeds")
	}
	again, err := c.GetHolders(context.Background(), "PEPE")
	if err != nil || again.HolderCount != 50 {
		t.Fatalf("cache %+v err=%v", again, err)
	}
	if gt.hits != 1 {
		t.Fatalf("second call should be cached, hits=%d", gt.hits)
	}
}

func TestCascade_FallsBackToEthplorer(t *testing.T) {
	gt := &fakeChain{err: domain.ErrHoldersUnpublished}
	eth := &fakeChain{snap: &domain.AssetHolders{
		Asset: "UNI", HolderCount: 12, Source: "ethplorer",
		TopHolders: []domain.AssetHolder{{Address: "0x1", SharePct: 10}},
	}}
	c := New(Options{
		CMC:       fakeCMC{err: domain.ErrCatalogUnmapped},
		Contracts: fakeContracts{list: []domain.AssetContract{{Chain: "ethereum", Address: "0xuni"}}},
		GeckoTerm: gt,
		Ethplorer: eth,
	})
	got, err := c.GetHolders(context.Background(), "UNI")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "ethplorer" || got.HolderCount != 12 {
		t.Fatalf("%+v", got)
	}
}

func TestCascade_FallsBackToRouteScan(t *testing.T) {
	scan := &fakeChain{snap: &domain.AssetHolders{Asset: "CITY", HolderCount: 1775, Source: "routescan"}}
	c := New(Options{
		CMC:       fakeCMC{err: domain.ErrHoldersUnpublished},
		Metrics:   fakeCMC{err: domain.ErrHoldersUnpublished},
		Profile:   fakeProfile{contracts: []domain.AssetContract{{Chain: "Chiliz", Address: "0x7bd6"}}},
		GeckoTerm: &fakeChain{err: domain.ErrHoldersUnpublished},
		Ethplorer: &fakeChain{err: domain.ErrHoldersUnpublished},
		RouteScan: scan,
	})
	got, err := c.GetHolders(context.Background(), "CITY")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "routescan" || got.HolderCount != 1775 {
		t.Fatalf("%+v", got)
	}
}

func TestCascade_FallsBackToTronScan(t *testing.T) {
	tron := &fakeChain{snap: &domain.AssetHolders{Asset: "JST", HolderCount: 441846, Source: "tronscan"}}
	c := New(Options{
		CMC:       fakeCMC{err: domain.ErrHoldersUnpublished},
		Metrics:   fakeCMC{err: domain.ErrHoldersUnpublished},
		Profile:   fakeProfile{contracts: []domain.AssetContract{{Chain: "tron20", Address: "TCFL"}}},
		GeckoTerm: &fakeChain{err: domain.ErrHoldersUnpublished},
		Ethplorer: &fakeChain{err: domain.ErrHoldersUnpublished},
		RouteScan: &fakeChain{err: domain.ErrHoldersUnpublished},
		TronScan:  tron,
	})
	got, err := c.GetHolders(context.Background(), "JST")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "tronscan" || got.HolderCount != 441846 {
		t.Fatalf("%+v", got)
	}
}

func TestCascade_AllMiss(t *testing.T) {
	c := New(Options{
		CMC:       fakeCMC{err: domain.ErrCatalogUnmapped},
		Contracts: fakeContracts{err: domain.ErrNotFound},
	})
	_, err := c.GetHolders(context.Background(), "ZZZ")
	if !errors.Is(err, domain.ErrCatalogUnmapped) {
		t.Fatalf("err=%v", err)
	}
}

func TestCascade_MetricsUpstreamDoesNotMaskUnpublished(t *testing.T) {
	c := New(Options{
		CMC:     fakeCMC{err: domain.ErrHoldersUnpublished},
		Metrics: fakeCMC{err: fmt.Errorf("%w: coinmetrics status 400", domain.ErrUpstream)},
	})
	_, err := c.GetHolders(context.Background(), "CITY")
	if !errors.Is(err, domain.ErrHoldersUnpublished) {
		t.Fatalf("err=%v", err)
	}
}

// keyedCMC returns a distinct snapshot per requested asset so cache-key
// collisions (WBTC vs W) are visible.
type keyedCMC struct {
	byAsset map[string]*domain.AssetHolders
	hits    []string
}

func (f *keyedCMC) GetHolders(_ context.Context, asset string) (*domain.AssetHolders, error) {
	key := domain.NormalizeAssetKey(asset)
	if key == "" {
		key = strings.ToUpper(strings.TrimSpace(asset))
	}
	f.hits = append(f.hits, key)
	if snap, ok := f.byAsset[key]; ok {
		return domain.CloneHolders(snap), nil
	}
	return nil, domain.ErrHoldersUnpublished
}

func TestCascade_WBTCDoesNotReuseWormholeWCache(t *testing.T) {
	src := &keyedCMC{byAsset: map[string]*domain.AssetHolders{
		"W":    {Asset: "W", HolderCount: 11_000, Source: "cmc-w"},
		"WBTC": {Asset: "WBTC", HolderCount: 250_000, Source: "cmc-wbtc"},
	}}
	c := New(Options{CMC: src, Cache: cache.New[*domain.AssetHolders](time.Hour)})

	w, err := c.GetHolders(context.Background(), "W")
	if err != nil || w.HolderCount != 11_000 || w.Asset != "W" {
		t.Fatalf("W: %+v err=%v", w, err)
	}
	wbtc, err := c.GetHolders(context.Background(), "WBTC")
	if err != nil {
		t.Fatal(err)
	}
	if wbtc.Asset != "WBTC" || wbtc.HolderCount != 250_000 || wbtc.Source != "cmc-wbtc" {
		t.Fatalf("WBTC reused W cache: asset=%s count=%d source=%s", wbtc.Asset, wbtc.HolderCount, wbtc.Source)
	}
	if len(src.hits) != 2 || src.hits[0] != "W" || src.hits[1] != "WBTC" {
		t.Fatalf("hits=%v", src.hits)
	}
}

func TestCascade_WDoesNotReuseWBTCCache(t *testing.T) {
	src := &keyedCMC{byAsset: map[string]*domain.AssetHolders{
		"W":    {Asset: "W", HolderCount: 11_000, Source: "cmc-w"},
		"WBTC": {Asset: "WBTC", HolderCount: 250_000, Source: "cmc-wbtc"},
	}}
	c := New(Options{CMC: src, Cache: cache.New[*domain.AssetHolders](time.Hour)})

	wbtc, err := c.GetHolders(context.Background(), "WBTCUSDT")
	if err != nil || wbtc.HolderCount != 250_000 {
		t.Fatalf("WBTCUSDT: %+v err=%v", wbtc, err)
	}
	again, err := c.GetHolders(context.Background(), "WBTC")
	if err != nil || again.HolderCount != 250_000 {
		t.Fatalf("WBTC pair/base should share cache: %+v err=%v", again, err)
	}
	if len(src.hits) != 1 {
		t.Fatalf("WBTCUSDT then WBTC should be one fetch, hits=%v", src.hits)
	}
	w, err := c.GetHolders(context.Background(), "W")
	if err != nil {
		t.Fatal(err)
	}
	if w.Asset != "W" || w.HolderCount != 11_000 || w.Source != "cmc-w" {
		t.Fatalf("W reused WBTC cache: asset=%s count=%d source=%s", w.Asset, w.HolderCount, w.Source)
	}
}

func TestCascade_STETHDoesNotReuseSTCache(t *testing.T) {
	src := &keyedCMC{byAsset: map[string]*domain.AssetHolders{
		"ST":    {Asset: "ST", HolderCount: 1, Source: "cmc-st"},
		"STETH": {Asset: "STETH", HolderCount: 400_000, Source: "cmc-steth"},
	}}
	c := New(Options{CMC: src, Cache: cache.New[*domain.AssetHolders](time.Hour)})
	if _, err := c.GetHolders(context.Background(), "ST"); err != nil {
		t.Fatal(err)
	}
	got, err := c.GetHolders(context.Background(), "STETH")
	if err != nil {
		t.Fatal(err)
	}
	if got.Asset != "STETH" || got.HolderCount != 400_000 {
		t.Fatalf("STETH reused ST cache: %+v", got)
	}
}
