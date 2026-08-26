package market

import (
	"context"
	"strings"
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
	ann := time.Date(2026, 8, 10, 6, 0, 0, 0, time.UTC)
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "HFTUSDT", DelistTime: when, AnnouncedAt: ann},
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
	if items[0].DelistAnnouncedAt == nil || !items[0].DelistAnnouncedAt.Equal(ann) {
		t.Fatalf("HFT announced=%v", items[0].DelistAnnouncedAt)
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

func TestHydrateDelistQuotesFillsEmptyStub(t *testing.T) {
	store := deliststore.NewMemory()
	halt := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	store.ReplaceAll(domain.ExchangeBybit, []domain.SpotDelistEntry{
		{Symbol: "VANRYUSDT", DelistTime: halt, AnnouncedAt: halt.Add(-6 * 24 * time.Hour)},
	})
	fm := &fakeMarket{
		tickerErr: domain.ErrNotFound,
		candles: []domain.Candle{{
			OpenTime:    halt.Add(-24 * time.Hour),
			Open:        "0.0010",
			High:        "0.0012",
			Low:         "0.0008",
			Close:       "0.0009",
			Volume:      "1000",
			QuoteVolume: "0.95",
			CloseTime:   halt.Add(-time.Millisecond),
		}},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBybit: fm,
	}, nil).WithDelistStore(store)
	items := svc.injectUpcomingDelists(domain.ExchangeBybit, nil)
	if len(items) != 1 || items[0].LastPrice != "" {
		t.Fatalf("pre-hydrate %+v", items)
	}
	svc.hydrateDelistQuotes(context.Background(), domain.ExchangeBybit, items)
	if items[0].LastPrice != "0.0009" || items[0].QuoteVolume != "0.95" || items[0].HighPrice != "0.0012" {
		t.Fatalf("hydrated %+v", items[0])
	}
	if items[0].PriceChangePercent == "" {
		t.Fatal("expected change pct from last candle")
	}
}

type endAwareMarket struct {
	fakeMarket
}

func (m *endAwareMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	m.lastQ = q
	if m.err != nil {
		return nil, m.err
	}
	if q.EndTime.IsZero() {
		return nil, nil
	}
	var out []domain.Candle
	for _, c := range m.candles {
		if !c.OpenTime.IsZero() && c.OpenTime.After(q.EndTime) {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func TestGetCandles_DelistKeepsLastSessionAfterMidnightSchedule(t *testing.T) {
	store := deliststore.NewMemory()
	halt := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "VICUSDT", DelistTime: halt},
	})
	last := domain.Candle{
		OpenTime: halt.Add(2 * time.Hour),
		Open:     "0.008", High: "0.009", Low: "0.006", Close: "0.0077",
		Volume: "1", CloseTime: halt.Add(3*time.Hour - time.Millisecond),
	}
	fm := &endAwareMarket{fakeMarket: fakeMarket{
		candles: []domain.Candle{
			{OpenTime: halt.Add(-time.Hour), Open: "0.01", High: "0.01", Low: "0.01", Close: "0.01"},
			last,
		},
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
	}, nil).WithDelistStore(store)
	bars, err := svc.GetCandles(context.Background(), "binance", "VICUSDT", "1h", 10, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 2 {
		t.Fatalf("want last session after official halt, got %d end=%v", len(bars), fm.lastQ.EndTime)
	}
	if !bars[1].OpenTime.Equal(last.OpenTime) || bars[1].Close != last.Close {
		t.Fatalf("last bar %+v", bars[1])
	}
}

func TestGetTicker24hFallsBackToHaltCandle(t *testing.T) {
	store := deliststore.NewMemory()
	halt := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	store.ReplaceAll(domain.ExchangeBybit, []domain.SpotDelistEntry{
		{Symbol: "VANRYUSDT", DelistTime: halt},
	})
	fm := &fakeMarket{
		tickerErr: domain.ErrNotFound,
		candles: []domain.Candle{{
			Open: "1", Close: "2", High: "3", Low: "0.5", Volume: "10", QuoteVolume: "20",
		}},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBybit: fm,
	}, nil).WithDelistStore(store)
	tkr, err := svc.GetTicker24h(context.Background(), "bybit", "VANRYUSDT")
	if err != nil || tkr == nil || tkr.LastPrice != "2" {
		t.Fatalf("ticker %+v err=%v", tkr, err)
	}
}

func TestGetTicker24hBlanksFrozenChangeWhenHalted(t *testing.T) {
	store := deliststore.NewMemory()
	halt := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "PYRUSDT", DelistTime: halt},
	})
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{
			ticker: &domain.Ticker24h{
				LastPrice: "0.021", PriceChangePercent: "-57.143", PriceChange: "-0.028",
			},
		},
	}, nil).WithDelistStore(store)
	tkr, err := svc.GetTicker24h(context.Background(), "binance", "PYRUSDT")
	if err != nil || tkr == nil {
		t.Fatalf("err=%v tkr=%+v", err, tkr)
	}
	if !tkr.Halted || tkr.LastPrice != "0.021" {
		t.Fatalf("%+v", tkr)
	}
	if tkr.PriceChangePercent != "" || tkr.PriceChange != "" {
		t.Fatalf("frozen 24h still live: %+v", tkr)
	}
}

func TestGetTicker24hUsesCoinGeckoChangeWhenHalted(t *testing.T) {
	store := deliststore.NewMemory()
	halt := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "PYRUSDT", DelistTime: halt},
	})
	pct := -12.5
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  &fakeMarket{ticker: &domain.Ticker24h{LastPrice: "0.021", PriceChangePercent: "-57.143"}},
		domain.ExchangeBybit:    &fakeMarket{tickerErr: domain.ErrNotFound},
		domain.ExchangeCoinbase: &fakeMarket{tickerErr: domain.ErrNotFound},
	}, nil).WithDelistStore(store).WithOffVenuePrice(&fakeOffVenue{
		quote: &domain.OffVenueQuote{LastUSD: 0.03, ChangePct: &pct},
	})
	tkr, err := svc.GetTicker24h(context.Background(), "binance", "PYRUSDT")
	if err != nil || tkr == nil {
		t.Fatalf("err=%v tkr=%+v", err, tkr)
	}
	if !tkr.Halted || tkr.LastPrice != "0.021" {
		t.Fatalf("%+v", tkr)
	}
	if tkr.PriceChangePercent != "-12.500" {
		t.Fatalf("want CoinGecko 24h, got %+v", tkr)
	}
	if tkr.PriceChange == "" {
		t.Fatalf("want 24h delta, got %+v", tkr)
	}
}

type fakeSymbolSupply struct {
	by map[string]*domain.AssetSupply
}

func (f fakeSymbolSupply) SupplyBySymbols(_ context.Context, symbols []string) (map[string]*domain.AssetSupply, error) {
	out := map[string]*domain.AssetSupply{}
	for _, s := range symbols {
		if v, ok := f.by[strings.ToUpper(s)]; ok {
			out[strings.ToUpper(s)] = v
		}
	}
	return out, nil
}

func TestEnrichDelistMcapUsesFallback(t *testing.T) {
	store := deliststore.NewMemory()
	halt := time.Now().UTC()
	store.ReplaceAll(domain.ExchangeBybit, []domain.SpotDelistEntry{
		{Symbol: "HFTUSDT", DelistTime: halt},
	})
	circ := 9e8
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBybit: &fakeMarket{},
	}, nil).WithDelistStore(store).WithDelistSupplyFallback(fakeSymbolSupply{
		by: map[string]*domain.AssetSupply{
			"HFT": {Asset: "HFT", CirculatingSupply: &circ, Source: "coingecko"},
		},
	})
	items := []domain.SpotMarket{{
		Symbol: "HFTUSDT", BaseAsset: "HFT", QuoteAsset: "USDT",
		LastPrice: "0.01", DelistTime: &halt,
	}}
	svc.enrichDelistMcap(context.Background(), domain.ExchangeBybit, items)
	if items[0].MarketCapCirculating == nil || *items[0].MarketCapCirculating != 9e6 {
		t.Fatalf("mcap=%v", items[0].MarketCapCirculating)
	}
}

func TestDropUnquotedDelistStubs(t *testing.T) {
	halt := time.Now().UTC()
	got := dropUnquotedDelistStubs([]domain.SpotMarket{
		{Symbol: "BTCUSDT", LastPrice: "100"},
		{Symbol: "GONEUSDT", LastPrice: "", DelistTime: &halt},
		{Symbol: "KEEPUSDT", LastPrice: "1", DelistTime: &halt},
	})
	if len(got) != 2 || got[0].Symbol != "BTCUSDT" || got[1].Symbol != "KEEPUSDT" {
		t.Fatalf("%+v", got)
	}
}

func TestHydrateDelistQuotesLeavesLivePrice(t *testing.T) {
	store := deliststore.NewMemory()
	halt := time.Now().UTC().Add(5 * 24 * time.Hour)
	store.ReplaceAll(domain.ExchangeBybit, []domain.SpotDelistEntry{
		{Symbol: "TAIUSDT", DelistTime: halt},
	})
	fm := &fakeMarket{candles: []domain.Candle{{Close: "9"}}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBybit: fm,
	}, nil).WithDelistStore(store)
	items := []domain.SpotMarket{{
		Symbol: "TAIUSDT", LastPrice: "0.0065", DelistTime: &halt,
	}}
	svc.hydrateDelistQuotes(context.Background(), domain.ExchangeBybit, items)
	if items[0].LastPrice != "0.0065" {
		t.Fatalf("live price overwritten: %+v", items[0])
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
