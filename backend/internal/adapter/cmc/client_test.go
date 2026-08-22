package cmc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type stubCatalog struct {
	entry *domain.AssetCatalogEntry
	err   error
}

func (s stubCatalog) LookupAsset(context.Context, string) (*domain.AssetCatalogEntry, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.entry, nil
}

func btcEntry() *domain.AssetCatalogEntry {
	return &domain.AssetCatalogEntry{Asset: "BTC", Name: "Bitcoin", CMCID: 1, Slug: "bitcoin"}
}

func fixtureDetail() []byte {
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"id": 1, "name": "Bitcoin", "symbol": "BTC",
			"holders": map[string]any{
				"holderCount":           50_708_169,
				"dailyActive":           963_625,
				"topTenHolderRatio":     5.4,
				"topTwentyHolderRatio":  7.48,
				"topFiftyHolderRatio":   10.86,
				"topHundredHolderRatio": 13.72,
				"holderList": []map[string]any{
					{"address": "34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo", "balance": 248597.39, "share": 1.18},
					{"address": "bc1qgdjqv0av3q56jvd82tkdjpy7gdp9ut8tlqmgrpmv24sq90ecnvqqjwvw97", "balance": 190010.08, "share": 0.9},
				},
			},
		},
	})
	return body
}

func newClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Catalog:    stubCatalog{entry: btcEntry()},
		Cache:      cache.New[*domain.AssetHolders](time.Hour),
	})
}

func TestGetHolders_ParsesSnapshot(t *testing.T) {
	var hits int
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if !strings.Contains(r.URL.Path, "/data-api/v3/cryptocurrency/detail") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("id") != "1" {
			t.Fatalf("id=%s", r.URL.Query().Get("id"))
		}
		_, _ = w.Write(fixtureDetail())
	}))
	got, err := c.GetHolders(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Asset != "BTC" || got.HolderCount != 50_708_169 || got.Source != "coinmarketcap" {
		t.Fatalf("%+v", got)
	}
	if got.DailyActive == nil || *got.DailyActive != 963_625 {
		t.Fatalf("daily=%v", got.DailyActive)
	}
	if got.TopTenSharePct == nil || *got.TopTenSharePct != 5.4 {
		t.Fatalf("top10=%v", got.TopTenSharePct)
	}
	if len(got.TopHolders) != 2 || got.TopHolders[0].SharePct != 1.18 {
		t.Fatalf("list=%+v", got.TopHolders)
	}
	// Second call is cache-only.
	if _, err := c.GetHolders(context.Background(), "BTC"); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("hits=%d want 1", hits)
	}
}

func TestGetHolders_NotPublished(t *testing.T) {
	hits := 0
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 1, "name": "Bitcoin", "holders": nil},
		})
	}))
	_, err := c.GetHolders(context.Background(), "BTC")
	if !errors.Is(err, domain.ErrHoldersUnpublished) {
		t.Fatalf("err=%v", err)
	}
	_, err = c.GetHolders(context.Background(), "BTC")
	if !errors.Is(err, domain.ErrHoldersUnpublished) {
		t.Fatalf("neg err=%v", err)
	}
	if hits != 1 {
		t.Fatalf("unpublished should be negative-cached, hits=%d", hits)
	}
}

func TestGetAssetProfile_LogoFromCatalog(t *testing.T) {
	c := New(Options{
		Catalog: stubCatalog{entry: btcEntry()},
		Cache:   cache.New[*domain.AssetHolders](time.Hour),
	})
	got, err := c.GetAssetProfile(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.LogoURL != domain.CMCLogoURL(1) || got.Asset != "BTC" {
		t.Fatalf("%+v", got)
	}
}

func TestGetHolders_CatalogMiss(t *testing.T) {
	c := New(Options{
		Catalog: stubCatalog{err: fmt.Errorf("%w: AAPL", domain.ErrCatalogUnmapped)},
		Cache:   cache.New[*domain.AssetHolders](time.Hour),
	})
	_, err := c.GetHolders(context.Background(), "AAPL")
	if !errors.Is(err, domain.ErrCatalogUnmapped) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetHolders_RateLimitServesStale(t *testing.T) {
	var n int
	c := newClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		if n == 1 {
			_, _ = w.Write(fixtureDetail())
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	if _, err := c.GetHolders(context.Background(), "BTC"); err != nil {
		t.Fatal(err)
	}
	c.holders.SetWithTTL("BTC", &domain.AssetHolders{Asset: "BTC", HolderCount: 1, Source: "stale"}, time.Nanosecond)
	time.Sleep(2 * time.Millisecond)
	got, err := c.GetHolders(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if got.HolderCount != 1 || !got.Stale {
		t.Fatalf("want stale last-good, got %+v", got)
	}
}

func TestParseDetail_CapsList(t *testing.T) {
	rows := make([]map[string]any, 30)
	for i := range rows {
		rows[i] = map[string]any{"address": "addr" + strconv.Itoa(i), "balance": i, "share": 0.1}
	}
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"name": "Pepe",
			"holders": map[string]any{
				"holderCount": 100,
				"holderList":  rows,
			},
		},
	})
	got, err := parseDetail(body, &domain.AssetCatalogEntry{Asset: "PEPE", CMCID: 24478})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.TopHolders) != domain.MaxHolderList {
		t.Fatalf("len=%d", len(got.TopHolders))
	}
}
