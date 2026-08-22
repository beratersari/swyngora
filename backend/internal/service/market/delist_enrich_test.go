package market

import (
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/deliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestEnsureTagPrependsOnce(t *testing.T) {
	got := ensureTag([]string{"defi", "Meme"}, domain.TagDelist)
	if len(got) != 3 || got[0] != domain.TagDelist {
		t.Fatalf("got=%v", got)
	}
	// case-insensitive de-dupe
	got2 := ensureTag(got, "delist")
	if len(got2) != 3 {
		t.Fatalf("dedupe failed: %v", got2)
	}
}

func TestEnrichDelistTimesAddsTagAndTime(t *testing.T) {
	store := deliststore.NewMemory()
	when := time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "HFTUSDT", DelistTime: when},
	})
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{}, nil).
		WithDelistStore(store).
		WithDelistEnabled(true)

	items := []domain.SpotMarket{
		{Symbol: "HFTUSDT", Tags: []string{"defi"}},
		{Symbol: "BTCUSDT", Tags: []string{"Layer1_Layer2"}},
	}
	svc.enrichDelistTimes(domain.ExchangeBinance, items)

	if items[0].DelistTime == nil || !items[0].DelistTime.Equal(when) {
		t.Fatalf("HFT delist time=%v", items[0].DelistTime)
	}
	if items[0].Tags[0] != domain.TagDelist {
		t.Fatalf("HFT tags=%v", items[0].Tags)
	}
	if items[1].DelistTime != nil {
		t.Fatal("BTC should not be delisted")
	}
	if len(items[1].Tags) != 1 || items[1].Tags[0] != "Layer1_Layer2" {
		t.Fatalf("BTC tags mutated: %v", items[1].Tags)
	}
}

func TestInjectUpcomingDelistsAddsMissingSoonPair(t *testing.T) {
	store := deliststore.NewMemory()
	when := time.Now().UTC().Add(10 * 24 * time.Hour)
	store.ReplaceAll(domain.ExchangeBybit, []domain.SpotDelistEntry{
		{Symbol: "SOONUSDT", DelistTime: when},
		{Symbol: "LATERUSDT", DelistTime: time.Now().UTC().Add(60 * 24 * time.Hour)},
	})
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{}, nil).WithDelistStore(store)
	items := []domain.SpotMarket{{Symbol: "BTCUSDT", Status: "TRADING"}}
	got := svc.injectUpcomingDelists(domain.ExchangeBybit, items)
	if len(got) != 2 {
		t.Fatalf("len=%d %+v", len(got), got)
	}
	if got[1].Symbol != "SOONUSDT" || got[1].DelistTime == nil || got[1].Tags[0] != domain.TagDelist {
		t.Fatalf("injected=%+v", got[1])
	}
}

func TestInjectUpcomingDelistsAddsRecentPastPair(t *testing.T) {
	store := deliststore.NewMemory()
	when := time.Now().UTC().Add(-10 * 24 * time.Hour)
	store.ReplaceAll(domain.ExchangeBybit, []domain.SpotDelistEntry{
		{Symbol: "GONEUSDT", DelistTime: when},
		{Symbol: "ANCIENTUSDT", DelistTime: time.Now().UTC().Add(-40 * 24 * time.Hour)},
	})
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{}, nil).WithDelistStore(store)
	items := []domain.SpotMarket{{Symbol: "BTCUSDT", Status: "TRADING"}}
	got := svc.injectUpcomingDelists(domain.ExchangeBybit, items)
	if len(got) != 2 {
		t.Fatalf("len=%d %+v", len(got), got)
	}
	if got[1].Symbol != "GONEUSDT" || got[1].DelistTime == nil || got[1].Tags[0] != domain.TagDelist {
		t.Fatalf("injected=%+v", got[1])
	}
}

func TestFilterSpotKeepsUpcomingDelistOnTradingQuery(t *testing.T) {
	when := time.Now().UTC().Add(5 * 24 * time.Hour)
	past := time.Now().UTC().Add(-10 * 24 * time.Hour)
	all := []domain.SpotMarket{
		{Symbol: "BTCUSDT", Status: "TRADING", QuoteAsset: "USDT"},
		{Symbol: "DEADUSDT", Status: "BREAK", QuoteAsset: "USDT", DelistTime: &when, Tags: []string{domain.TagDelist}},
		{Symbol: "GONEUSDT", Status: "BREAK", QuoteAsset: "USDT", DelistTime: &past, Tags: []string{domain.TagDelist}},
	}
	got := filterSpotMarkets(all, domain.SpotListQuery{Status: "TRADING", QuoteAsset: "USDT"})
	if len(got) != 3 {
		t.Fatalf("want BTC + upcoming + last-30d delist, got %+v", got)
	}
}

func TestWithDelistFilterTag(t *testing.T) {
	store := deliststore.NewMemory()
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "ACXUSDT", DelistTime: time.Now().UTC()},
	})
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{}, nil).WithDelistStore(store)
	tags := svc.withDelistFilterTag(domain.ExchangeBinance, []string{"defi", "Meme"})
	if tags[0] != domain.TagDelist {
		t.Fatalf("expected Delist first, got %v", tags)
	}
}
