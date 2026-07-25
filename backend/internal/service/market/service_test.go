package market

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeMarket struct {
	candles []domain.Candle
	ticker  *domain.Ticker24h
	spot    []domain.SpotMarket
	err     error
	lastQ   domain.CandleQuery
	lastSym string
}

func (f *fakeMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	f.lastQ = q
	if f.err != nil {
		return nil, f.err
	}
	return f.candles, nil
}

func (f *fakeMarket) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	f.lastSym = symbol
	if f.err != nil {
		return nil, f.err
	}
	return f.ticker, nil
}

func (f *fakeMarket) ListSpotMarkets(_ context.Context) ([]domain.SpotMarket, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.spot != nil {
		return append([]domain.SpotMarket(nil), f.spot...), nil
	}
	return []domain.SpotMarket{
		{Symbol: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "500", Volume: "20", LastPrice: "50", PriceChangePercent: "-2", TradeCount: 3, Tags: []string{"Layer1_Layer2", "pos"}},
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000", Volume: "10", LastPrice: "100", PriceChangePercent: "1.5", TradeCount: 9, Tags: []string{"Payments", "Layer1_Layer2"}},
		{Symbol: "BTCUSDC", BaseAsset: "BTC", QuoteAsset: "USDC", Status: "TRADING", QuoteVolume: "200", Volume: "2", LastPrice: "99", PriceChangePercent: "0.1", TradeCount: 1, Tags: []string{"Payments", "Layer1_Layer2"}},
		{Symbol: "XRPUSDT", BaseAsset: "XRP", QuoteAsset: "USDT", Status: "BREAK", QuoteVolume: "50", Volume: "100", LastPrice: "1", PriceChangePercent: "5", TradeCount: 100, Tags: []string{"Payments"}},
		{Symbol: "DOGEUSDT", BaseAsset: "DOGE", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "80", Volume: "1000", LastPrice: "0.1", PriceChangePercent: "3", TradeCount: 50, Tags: []string{"Meme"}},
	}, nil
}

func (f *fakeMarket) ListProductTags(_ context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []string{"Layer1_Layer2", "Meme", "Payments", "pos"}, nil
}

type fakeSupply struct {
	sup    *domain.AssetSupply
	byAsset map[string]*domain.AssetSupply
	err    error
}

func (f *fakeSupply) GetSupply(_ context.Context, asset string) (*domain.AssetSupply, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.byAsset != nil {
		if s, ok := f.byAsset[strings.ToUpper(asset)]; ok {
			return s, nil
		}
		return nil, domain.ErrNotFound
	}
	return f.sup, nil
}

func (f *fakeSupply) Refresh(context.Context) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	if f.byAsset != nil {
		return len(f.byAsset), nil
	}
	if f.sup != nil {
		return 1, nil
	}
	return 0, nil
}

func TestGetCandles_DefaultsAndValidation(t *testing.T) {
	fm := &fakeMarket{candles: []domain.Candle{{Open: "1"}}}
	svc := New(fm, &fakeSupply{})

	_, err := svc.GetCandles(context.Background(), "binance", "", "1h", 10, nil, nil)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty symbol: %v", err)
	}
	_, err = svc.GetCandles(context.Background(), "binance", "BTCUSDT", "9y", 10, nil, nil)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("bad interval: %v", err)
	}
	_, err = svc.GetCandles(context.Background(), "binance", "BTCUSDT", "1h", 2000, nil, nil)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("limit: %v", err)
	}

	out, err := svc.GetCandles(context.Background(), "binance", "btcusdt", "1h", 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || fm.lastQ.Limit != 100 || fm.lastQ.Symbol != "BTCUSDT" {
		t.Fatalf("q=%+v out=%+v", fm.lastQ, out)
	}
}

func TestGetCandles_TimeOrder(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	start := time.Now()
	end := start.Add(-time.Hour)
	_, err := svc.GetCandles(context.Background(), "binance", "BTCUSDT", "1h", 10, &start, &end)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetTicker24h(t *testing.T) {
	fm := &fakeMarket{ticker: &domain.Ticker24h{Symbol: "BTCUSDT", Volume: "1"}}
	svc := New(fm, &fakeSupply{})
	tkr, err := svc.GetTicker24h(context.Background(), "binance", " btcusdt ")
	if err != nil {
		t.Fatal(err)
	}
	if tkr.Volume != "1" || fm.lastSym != "BTCUSDT" {
		t.Fatalf("tkr=%+v last=%s", tkr, fm.lastSym)
	}
}

func TestGetSupply(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{sup: &domain.AssetSupply{Asset: "BTC"}})
	_, err := svc.GetSupply(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
	sup, err := svc.GetSupply(context.Background(), "BTC")
	if err != nil || sup.Asset != "BTC" {
		t.Fatalf("sup=%+v err=%v", sup, err)
	}
}

func TestListIntervals(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	iv, err := svc.ListIntervals("binance")
	if err != nil {
		t.Fatal(err)
	}
	if len(iv) != len(domain.SupportedIntervals) {
		t.Fatalf("len=%d", len(iv))
	}
	if _, err := svc.ListIntervals("coinbase"); err == nil {
		t.Fatal("expected error for unconfigured exchange")
	}
}

func TestMultiExchange_ListExchanges(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  &fakeMarket{},
		domain.ExchangeCoinbase: &fakeMarket{},
		domain.ExchangeBybit:    &fakeMarket{},
	}, &fakeSupply{})
	if len(svc.ListExchanges()) != 3 {
		t.Fatalf("ex=%v", svc.ListExchanges())
	}
	iv, err := svc.ListIntervals("coinbase")
	if err != nil || len(iv) == 0 {
		t.Fatalf("coinbase intervals %v %v", iv, err)
	}
}

func TestListSpotMarkets_DefaultSortAndSearch(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	res, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 5 || res.Limit != 50 || res.SortBy != domain.SpotSortQuoteVolume || res.Order != domain.SortDesc {
		t.Fatalf("meta=%+v", res)
	}
	if res.Items[0].Symbol != "BTCUSDT" || res.Items[1].Symbol != "ETHUSDT" {
		t.Fatalf("order=%v %v", res.Items[0].Symbol, res.Items[1].Symbol)
	}

	// Search BTC
	res, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{Query: "btc"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("search total=%d", res.Total)
	}

	// Quote filter USDT + TRADING
	res, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		QuoteAsset: "usdt",
		Status:     "TRADING",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 3 { // BTC, ETH, DOGE
		t.Fatalf("quote+status total=%d", res.Total)
	}

	// Sort by priceChangePercent desc → XRP first
	res, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		SortBy: domain.SpotSortPriceChangePercent,
		Order:  domain.SortDesc,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items[0].Symbol != "XRPUSDT" {
		t.Fatalf("want XRP first, got %s", res.Items[0].Symbol)
	}

	// Pagination
	res, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		SortBy: domain.SpotSortSymbol,
		Order:  domain.SortAsc,
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 5 || len(res.Items) != 1 || res.Items[0].Symbol != "BTCUSDT" {
		// symbols asc: BTCUSDC, BTCUSDT, DOGEUSDT, ETHUSDT, XRPUSDT
		t.Fatalf("page=%+v total=%d", res.Items, res.Total)
	}
}

func TestListSpotMarkets_FilterByTag(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	res, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		Tags:   []string{"meme"},
		Status: "TRADING",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].Symbol != "DOGEUSDT" {
		t.Fatalf("meme filter: total=%d items=%v", res.Total, symbolsOf(res.Items))
	}
	// OR match
	res, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		Tags: []string{"Meme", "Payments"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total < 3 {
		t.Fatalf("OR tags total=%d want >=3", res.Total)
	}
	// Search also matches tag names
	res, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{Query: "meme"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].Symbol != "DOGEUSDT" {
		t.Fatalf("search meme: %+v", res.Items)
	}
}

func TestListSpotMarkets_SortByTags(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	res, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		SortBy: domain.SpotSortTags,
		Order:  domain.SortAsc,
		Limit:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Order != domain.SortAsc {
		t.Fatalf("order=%s", res.Order)
	}
	// First non-empty tags should sort lexicographically by joined tags
	if len(res.Items) == 0 {
		t.Fatal("empty")
	}
}

func TestListProductTags(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	tags, err := svc.ListProductTags(context.Background(), "binance")
	if err != nil || len(tags) == 0 {
		t.Fatalf("tags=%v err=%v", tags, err)
	}
}

func TestListSpotMarkets_Validation(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{SortBy: "nope"})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("sort err=%v", err)
	}
	_, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{Order: "sideways"})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("order err=%v", err)
	}
	_, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{Limit: 9999})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("limit err=%v", err)
	}
	_, err = svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{Offset: -1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("offset err=%v", err)
	}
}

func TestListSpotMarkets_McapEnrichment(t *testing.T) {
	circ := 10.0
	total := 12.0
	// ETH has no max supply
	btcMax := 21.0
	sup := &fakeSupply{byAsset: map[string]*domain.AssetSupply{
		"BTC": {
			Asset: "BTC", CirculatingSupply: &circ, TotalSupply: &total, MaxSupply: &btcMax,
			CurrentPriceUSD: ptr(100.0),
		},
		"ETH": {
			Asset: "ETH", CirculatingSupply: &circ, TotalSupply: &total, MaxSupply: nil,
			CurrentPriceUSD: ptr(50.0),
		},
	}}
	svc := New(&fakeMarket{}, sup)
	res, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		QuoteAsset: "USDT",
		Status:     "TRADING",
		SortBy:     domain.SpotSortQuoteVolume,
		Limit:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]domain.SpotMarket{}
	for _, m := range res.Items {
		by[m.Symbol] = m
	}
	btc := by["BTCUSDT"]
	// lastPrice 100 * circ 10 = 1000
	if btc.MarketCapCirculating == nil || *btc.MarketCapCirculating != 1000 {
		t.Fatalf("btc circ mcap=%v", btc.MarketCapCirculating)
	}
	if btc.MarketCapMax == nil || *btc.MarketCapMax != 2100 {
		t.Fatalf("btc max mcap=%v", btc.MarketCapMax)
	}
	if btc.MarketCapMaxInfinite {
		t.Fatal("btc should not be infinite max")
	}
	eth := by["ETHUSDT"]
	// lastPrice 50 * circ 10 = 500
	if eth.MarketCapCirculating == nil || *eth.MarketCapCirculating != 500 {
		t.Fatalf("eth circ mcap=%v", eth.MarketCapCirculating)
	}
	if !eth.MarketCapMaxInfinite || eth.MarketCapMax != nil {
		t.Fatalf("eth max should be infinite, got max=%v inf=%v", eth.MarketCapMax, eth.MarketCapMaxInfinite)
	}
}

func TestListSpotMarkets_SortByMcap(t *testing.T) {
	circ := 10.0
	sup := &fakeSupply{byAsset: map[string]*domain.AssetSupply{
		"BTC": {Asset: "BTC", CirculatingSupply: &circ, CurrentPriceUSD: ptr(100.0)},
		"ETH": {Asset: "ETH", CirculatingSupply: &circ, CurrentPriceUSD: ptr(50.0)},
		"XRP": {Asset: "XRP", CirculatingSupply: &circ, CurrentPriceUSD: ptr(1.0)},
	}}
	svc := New(&fakeMarket{}, sup)
	res, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		SortBy: domain.SpotSortMarketCapCirculating,
		Order:  domain.SortDesc,
		Limit:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items[0].Symbol != "BTCUSDT" {
		t.Fatalf("want BTC first by mcap, got %s", res.Items[0].Symbol)
	}
}

func ptr(f float64) *float64 { return &f }

// Test that a nil SupplyPort produces a clean error (not a panic) from GetSupply.
func TestGetSupply_NilSupplyPortReturnsError(t *testing.T) {
	svc := New(&fakeMarket{}, nil)
	_, err := svc.GetSupply(context.Background(), "BTC")
	if err == nil || !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("expected upstream error, got %v", err)
	}
}

func TestListSpotMarkets_McapSortNullsLast(t *testing.T) {
	circ := 10.0
	sup := &fakeSupply{byAsset: map[string]*domain.AssetSupply{
		"BTC": {Asset: "BTC", CirculatingSupply: &circ, CurrentPriceUSD: ptr(100.0)},
		// ETH has supply but XRP has no row → nil mcap
		"ETH": {Asset: "ETH", CirculatingSupply: &circ, CurrentPriceUSD: ptr(50.0)},
	}}
	svc := New(&fakeMarket{}, sup)
	res, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		SortBy: domain.SpotSortMarketCapCirculating,
		Order:  domain.SortDesc,
		Limit:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	// XRP (no supply) and BREAK status XRP still in list unless filtered — last among mcaps
	last := res.Items[len(res.Items)-1]
	if last.MarketCapCirculating != nil {
		t.Fatalf("expected null mcap last, got %s mcap=%v", last.Symbol, last.MarketCapCirculating)
	}
}

func TestListSpotMarkets_McapSortEmptySupplyErrors(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{err: domain.ErrNotFound})
	_, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		SortBy: domain.SpotSortMarketCapCirculating,
	})
	if !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("want upstream when no supply hits, got %v", err)
	}
}

func TestListSpotMarkets_McapSortCollapsesMultiQuote(t *testing.T) {
	circ := 10.0
	sup := &fakeSupply{byAsset: map[string]*domain.AssetSupply{
		"BTC": {Asset: "BTC", CirculatingSupply: &circ, CurrentPriceUSD: ptr(100.0)},
		"ETH": {Asset: "ETH", CirculatingSupply: &circ, CurrentPriceUSD: ptr(50.0)},
		"XRP": {Asset: "XRP", CirculatingSupply: &circ, CurrentPriceUSD: ptr(1.0)},
	}}
	svc := New(&fakeMarket{}, sup)
	res, err := svc.ListSpotMarkets(context.Background(), "binance", domain.SpotListQuery{
		SortBy: domain.SpotSortMarketCapCirculating,
		Order:  domain.SortDesc,
		Limit:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Only one BTC row (preferred USDT over USDC)
	btcCount := 0
	for _, m := range res.Items {
		if m.BaseAsset == "BTC" || strings.HasPrefix(m.Symbol, "BTC") {
			btcCount++
		}
	}
	if btcCount != 1 {
		t.Fatalf("want 1 BTC row after collapse, got %d in %+v", btcCount, symbolsOf(res.Items))
	}
	if res.Items[0].Symbol != "BTCUSDT" {
		t.Fatalf("want BTCUSDT primary, got %s", res.Items[0].Symbol)
	}
}

func TestApplySupplyAndMcap_NoPriceNotInfinite(t *testing.T) {
	circ := 10.0
	m := domain.SpotMarket{Symbol: "FOOBTC", BaseAsset: "FOO", QuoteAsset: "BTC", LastPrice: "1"}
	sup := &domain.AssetSupply{Asset: "FOO", CirculatingSupply: &circ, MaxSupply: nil}
	applySupplyAndMcap(&m, sup)
	if m.MarketCapMaxInfinite {
		t.Fatal("must not set infinite max mcap without a USD price")
	}
	if m.MarketCapCirculating != nil {
		t.Fatal("no mcap without price")
	}
}

func symbolsOf(items []domain.SpotMarket) []string {
	out := make([]string, len(items))
	for i, m := range items {
		out[i] = m.Symbol
	}
	return out
}
