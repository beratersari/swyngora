package market

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeMarket struct {
	candles      []domain.Candle
	ticker       *domain.Ticker24h
	tickers      map[string]*domain.Ticker24h
	spot         []domain.SpotMarket
	prints       map[string][]domain.TakerPrint
	printSyms    []string
	err          error
	tickerErr    error
	lastQ        domain.CandleQuery
	lastSym      string
	lastBookQ    domain.OrderBookQuery
	snapshotN    int
	failSnapshot bool
}

func sampleRawBook(symbol string) *domain.RawOrderBook {
	return &domain.RawOrderBook{
		Symbol: symbol,
		Bids: []domain.PriceLevel{
			{Price: 100.04, Quantity: 1},
			{Price: 100.02, Quantity: 1},
			{Price: 99.50, Quantity: 40},
			{Price: 99.10, Quantity: 2},
			{Price: 80, Quantity: 9_000}, // far — analysis at 2% must ignore
		},
		Asks: []domain.PriceLevel{
			{Price: 100.06, Quantity: 1},
			{Price: 100.20, Quantity: 1},
			{Price: 100.80, Quantity: 50},
			{Price: 101.00, Quantity: 2},
			{Price: 120, Quantity: 9_000},
		},
	}
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
	if f.tickerErr != nil {
		return nil, f.tickerErr
	}
	if f.err != nil {
		return nil, f.err
	}
	if f.tickers != nil {
		if t, ok := f.tickers[strings.ToUpper(symbol)]; ok {
			return t, nil
		}
	}
	return f.ticker, nil
}

func (f *fakeMarket) GetRecentPrints(_ context.Context, symbol string) ([]domain.TakerPrint, error) {
	f.printSyms = append(f.printSyms, symbol)
	if f.err != nil {
		return nil, f.err
	}
	if f.prints != nil {
		return append([]domain.TakerPrint(nil), f.prints[strings.ToUpper(symbol)]...), nil
	}
	return nil, nil
}

func (f *fakeMarket) GetOrderBook(_ context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	f.lastBookQ = q
	if q.SnapshotOnly {
		f.snapshotN++
		if f.failSnapshot {
			return nil, fmt.Errorf("%w: snapshot down", domain.ErrUpstream)
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return sampleRawBook(q.Symbol), nil
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

func (f *fakeMarket) TagsByBase(_ context.Context) (map[string][]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return map[string][]string{
		"BTC":  {"Payments", "Layer1_Layer2"},
		"ETH":  {"Layer1_Layer2", "pos"},
		"DOGE": {"Meme"},
		"XRP":  {"Payments"},
	}, nil
}

type fakeSupply struct {
	sup     *domain.AssetSupply
	byAsset map[string]*domain.AssetSupply
	err     error
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

type fakeHolders struct {
	snap *domain.AssetHolders
	err  error
}

func (f *fakeHolders) GetHolders(context.Context, string) (*domain.AssetHolders, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.snap, nil
}

func TestGetHolders(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetHolders(context.Background(), "BTC")
	if !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("nil port: %v", err)
	}
	svc = svc.WithHolders(&fakeHolders{snap: &domain.AssetHolders{
		Asset:       "BTC",
		HolderCount: 9,
		TopHolders:  []domain.AssetHolder{{Address: "34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo"}},
	}})
	_, err = svc.GetHolders(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty: %v", err)
	}
	got, err := svc.GetHolders(context.Background(), "BTC")
	if err != nil || got.HolderCount != 9 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if len(got.TopHolders) != 1 || got.TopHolders[0].Label != "Binance" {
		t.Fatalf("label=%+v", got.TopHolders)
	}
}

func TestGetHolders_FallbackWhenCMCUnpublished(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{}).
		WithHolders(&fakeHolders{err: fmt.Errorf("%w: ACE", domain.ErrHoldersUnpublished)}).
		WithHoldersFallback(&fakeHolders{snap: &domain.AssetHolders{Asset: "ACE", HolderCount: 120_664, Source: "cryptoid"}})
	got, err := svc.GetHolders(context.Background(), "ACE")
	if err != nil || got.HolderCount != 120_664 || got.Source != "cryptoid" {
		t.Fatalf("got=%+v err=%v", got, err)
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

func TestListSpotMarkets_EnrichesTagsFromBinance(t *testing.T) {
	// Coinbase-like rows with no tags; Binance catalog provides tags by base.
	coinbase := &fakeMarket{
		spot: []domain.SpotMarket{
			{Symbol: "BTC-USD", BaseAsset: "BTC", QuoteAsset: "USD", Status: "TRADING", QuoteVolume: "100", LastPrice: "100"},
			{Symbol: "DOGE-USD", BaseAsset: "DOGE", QuoteAsset: "USD", Status: "TRADING", QuoteVolume: "10", LastPrice: "0.1"},
		},
	}
	// Override ListProductTags empty for coinbase-only path; TagsByBase used from binance port.
	binance := &fakeMarket{}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  binance,
		domain.ExchangeCoinbase: coinbase,
	}, &fakeSupply{})
	res, err := svc.ListSpotMarkets(context.Background(), "coinbase", domain.SpotListQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]domain.SpotMarket{}
	for _, m := range res.Items {
		by[m.Symbol] = m
	}
	if len(by["BTC-USD"].Tags) == 0 || by["BTC-USD"].Tags[0] != "Payments" {
		t.Fatalf("btc tags=%v", by["BTC-USD"].Tags)
	}
	if len(by["DOGE-USD"].Tags) == 0 || by["DOGE-USD"].Tags[0] != "Meme" {
		t.Fatalf("doge tags=%v", by["DOGE-USD"].Tags)
	}
	// Tag filter works after enrichment.
	res, err = svc.ListSpotMarkets(context.Background(), "coinbase", domain.SpotListQuery{Tags: []string{"Meme"}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].Symbol != "DOGE-USD" {
		t.Fatalf("meme filter: %+v", res)
	}
}

func TestListSpotMarkets_EquitySkipsBinanceCryptoEnrichment(t *testing.T) {
	// BIST LINK/QUICK are Turkish companies. Binance must not paint crypto
	// tags or circulating-supply mcap onto them.
	bist := &fakeMarket{
		spot: []domain.SpotMarket{
			{Symbol: "LINK", BaseAsset: "LINK", QuoteAsset: "TRY", Status: "TRADING", QuoteVolume: "278e6", LastPrice: "5.94"},
			{Symbol: "QUICK", BaseAsset: "QUICK", QuoteAsset: "TRY", Status: "TRADING", QuoteVolume: "578e6", LastPrice: "73.35"},
			{Symbol: "A1CAP", BaseAsset: "A1CAP", QuoteAsset: "TRY", Status: "TRADING", QuoteVolume: "48e6", LastPrice: "8.02"},
		},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBist:    bist,
	}, &fakeSupply{byAsset: map[string]*domain.AssetSupply{
		"LINK":  {CirculatingSupply: ptr(1.13e9)},
		"QUICK": {CirculatingSupply: ptr(9.6e4)},
	}})
	res, err := svc.ListSpotMarkets(context.Background(), "bist", domain.SpotListQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]domain.SpotMarket{}
	for _, m := range res.Items {
		by[m.Symbol] = m
	}
	if len(by["LINK"].Tags) != 0 {
		t.Fatalf("LINK must not inherit crypto tags: %v", by["LINK"].Tags)
	}
	if len(by["QUICK"].Tags) != 0 {
		t.Fatalf("QUICK must not inherit crypto tags: %v", by["QUICK"].Tags)
	}
	if by["LINK"].MarketCapCirculating != nil {
		t.Fatalf("LINK must not get crypto mcap: %v", *by["LINK"].MarketCapCirculating)
	}
	if by["QUICK"].MarketCapCirculating != nil {
		t.Fatalf("QUICK must not get crypto mcap: %v", *by["QUICK"].MarketCapCirculating)
	}
}

func TestListProductTags_EquityDoesNotFallBackToBinance(t *testing.T) {
	empty := &tagsEmptyMarket{fakeMarket: &fakeMarket{spot: []domain.SpotMarket{}}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBist:    empty,
	}, &fakeSupply{})
	tags, err := svc.ListProductTags(context.Background(), "bist")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 0 {
		t.Fatalf("bist must not reuse Binance crypto tags: %v", tags)
	}
}

func TestListProductTags_FallsBackToBinance(t *testing.T) {
	emptyTags := &fakeMarket{spot: []domain.SpotMarket{}}
	// coinbase-style empty tags list
	emptyTagsTags := &tagsEmptyMarket{fakeMarket: emptyTags}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  &fakeMarket{},
		domain.ExchangeCoinbase: emptyTagsTags,
	}, &fakeSupply{})
	tags, err := svc.ListProductTags(context.Background(), "coinbase")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) == 0 {
		t.Fatal("expected Binance tag fallback")
	}
}

type tagsEmptyMarket struct {
	*fakeMarket
}

func (t *tagsEmptyMarket) ListProductTags(context.Context) ([]string, error) {
	return []string{}, nil
}

func (t *tagsEmptyMarket) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func TestGetSpotOrderBook_Groups(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	book, err := svc.GetSpotOrderBook(context.Background(), "binance", "btcusdt", "0.1", 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if book.Exchange != domain.ExchangeBinance || book.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", book)
	}
	if book.GroupSize != "0.1" || len(book.Bids) == 0 || len(book.Asks) == 0 {
		t.Fatalf("%+v", book)
	}
	if book.Analysis.RangePct != 2 || book.Analysis.Pressure == "" {
		t.Fatalf("analysis %+v", book.Analysis)
	}
	if book.Analysis.BidLevels != 4 || book.Analysis.AskLevels != 4 {
		t.Fatalf("far depth must be excluded: %+v", book.Analysis)
	}
	var sawLife bool
	for _, w := range book.Analysis.Walls {
		if w.Behavior == domain.WallBehaviorShort && w.AppearCount == 1 {
			sawLife = true
		}
	}
	if len(book.Analysis.Walls) > 0 && !sawLife {
		t.Fatalf("new walls should be short: %+v", book.Analysis.Walls)
	}
	if _, err := svc.GetSpotOrderBook(context.Background(), "binance", "BTCUSDT", "nope", 10, 2); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("bad group: %v", err)
	}
}

func TestGetOrderBookHeatmap_SeedsTape(t *testing.T) {
	fx := &fakeMarket{}
	svc := New(fx, &fakeSupply{})
	got, err := svc.GetOrderBookHeatmap(context.Background(), "binance", "btcusdt", "0.1", 600)
	if err != nil {
		t.Fatal(err)
	}
	if got.Exchange != domain.ExchangeBinance || got.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", got)
	}
	if got.WindowSeconds != 600 || got.GroupSize != "0.1" {
		t.Fatalf("meta %+v", got)
	}
	if len(got.Columns) != 1 {
		t.Fatalf("expected seeded column, got %d", len(got.Columns))
	}
	if got.Columns[0].Mid == "" || (len(got.Columns[0].Bids) == 0 && len(got.Columns[0].Asks) == 0) {
		t.Fatalf("empty column %+v", got.Columns[0])
	}
	if _, err := svc.GetOrderBookHeatmap(context.Background(), "binance", "", "", 0); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("missing symbol: %v", err)
	}
	if !fx.lastBookQ.SnapshotOnly {
		t.Fatal("heatmap GET must try REST snapshot first")
	}
}

func TestGetOrderBookHeatmap_FallsBackToLiveBook(t *testing.T) {
	fx := &fakeMarket{failSnapshot: true}
	svc := New(fx, &fakeSupply{})
	got, err := svc.GetOrderBookHeatmap(context.Background(), "binance", "BTCUSDT", "", 600)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Columns) != 1 {
		t.Fatalf("expected live fallback column, got %d", len(got.Columns))
	}
	if fx.lastBookQ.SnapshotOnly {
		t.Fatal("after REST failure, last fetch should be the live book")
	}
	if fx.snapshotN != 1 {
		t.Fatalf("should attempt snapshot once, got %d", fx.snapshotN)
	}
}

func TestWallWatchIdleOutlastsPersistentMin(t *testing.T) {
	idle := wallWatchIdle()
	need := domain.WallPersistentMin + wallSampleEvery
	if idle <= need {
		t.Fatalf("idle %s must be > persistent min + sample %s so the last tick can pass 2m", idle, need)
	}
}

func TestGetCombinedOrderBookAnalysis(t *testing.T) {
	bin := &fakeMarket{}
	cb := &fakeMarket{}
	bb := &fakeMarket{}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  bin,
		domain.ExchangeCoinbase: cb,
		domain.ExchangeBybit:    bb,
	}, &fakeSupply{})
	got, err := svc.GetCombinedOrderBookAnalysis(context.Background(), "btcusdt", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.VenueCount != 3 || got.Pressure == "" || got.Symbol != "BTCUSDT" {
		t.Fatalf("%+v", got)
	}
	if len(got.Venues) != 3 {
		t.Fatalf("venues %+v", got.Venues)
	}
}

func TestGetMarketLiquidity_AllVenues(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  &fakeMarket{},
		domain.ExchangeCoinbase: &fakeMarket{},
		domain.ExchangeBybit:    &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.GetMarketLiquidity(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.VenueCount != 3 || got.Market.Score <= 0 || len(got.Market.Bands) != 3 {
		t.Fatalf("%+v", got)
	}
	one, err := svc.GetMarketLiquidity(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if one.VenueCount != 1 || string(one.Venues[0].Exchange) != "binance" {
		t.Fatalf("one venue %+v", one)
	}
}

func TestGetLiquidations_EmptyFeed(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	got, err := svc.GetLiquidations(context.Background(), "all", "BTCUSDT")
	if err != nil || got.Symbol != "BTCUSDT" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestGetLiquidations_FromBook(t *testing.T) {
	book := domain.NewLiquidationBook()
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 100, Quantity: 2, Notional: 200, Time: time.Now().UTC(),
	})
	svc := New(&fakeMarket{}, &fakeSupply{}).WithLiquidations(book, nil)
	got, err := svc.GetLiquidations(context.Background(), "binance", "btc-usd")
	if err != nil {
		t.Fatal(err)
	}
	if got.Windows[0].Count != 1 {
		t.Fatalf("%+v", got)
	}
}

type fakeOI struct {
	ser *domain.OpenInterestSeries
	err error
}

func (f *fakeOI) GetOpenInterestSeries(_ context.Context, symbol string) (*domain.OpenInterestSeries, error) {
	if f.err != nil {
		return nil, f.err
	}
	cp := *f.ser
	cp.Symbol = symbol
	return &cp, nil
}

func TestGetOpenInterest_Combined(t *testing.T) {
	now := time.Now().UTC()
	bin := &fakeOI{ser: &domain.OpenInterestSeries{
		Exchange: domain.ExchangeBinance,
		Current:  domain.OpenInterestPoint{Time: now, Contracts: 100, Value: 10000},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-5 * time.Minute), Contracts: 90, Value: 9000},
			{Time: now.Add(-time.Hour), Contracts: 80, Value: 8000},
			{Time: now.Add(-4 * time.Hour), Contracts: 70, Value: 7000},
			{Time: now.Add(-24 * time.Hour), Contracts: 50, Value: 5000},
		},
	}}
	byb := &fakeOI{ser: &domain.OpenInterestSeries{
		Exchange: domain.ExchangeBybit,
		Current:  domain.OpenInterestPoint{Time: now, Contracts: 50, Value: 5000},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-5 * time.Minute), Contracts: 40, Value: 4000},
			{Time: now.Add(-time.Hour), Contracts: 40, Value: 4000},
			{Time: now.Add(-4 * time.Hour), Contracts: 30, Value: 3000},
			{Time: now.Add(-24 * time.Hour), Contracts: 20, Value: 2000},
		},
	}}
	svc := New(&fakeMarket{}, &fakeSupply{}).WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
		domain.ExchangeBinance: bin,
		domain.ExchangeBybit:   byb,
	})
	got, err := svc.GetOpenInterest(context.Background(), "all", "btc-usd")
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || got.Unit != "BTC" || got.VenueCount != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Current.Contracts != "150" {
		t.Fatalf("current %s", got.Current.Contracts)
	}
	one, err := svc.GetOpenInterest(context.Background(), "binance", "BTCUSDT")
	if err != nil || one.VenueCount != 1 || one.Current.Contracts != "100" {
		t.Fatalf("%+v %v", one, err)
	}
}

func TestGetOpenInterest_BadExchange(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetOpenInterest(context.Background(), "coinbase", "BTCUSDT")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetOpenInterest_EmptyPorts(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	got, err := svc.GetOpenInterest(context.Background(), "all", "BTCUSDT")
	if err != nil || got.VenueCount != 0 || got.Symbol != "BTCUSDT" {
		t.Fatalf("%+v %v", got, err)
	}
}

type fakeFunding struct {
	ser *domain.FundingSeries
	err error
}

func (f *fakeFunding) GetFundingSeries(_ context.Context, symbol string, _ int) (*domain.FundingSeries, error) {
	if f.err != nil {
		return nil, f.err
	}
	cp := *f.ser
	cp.Symbol = symbol
	return &cp, nil
}

func TestGetFundingRate_Combined(t *testing.T) {
	now := time.Now().UTC()
	bin := &fakeFunding{ser: &domain.FundingSeries{
		Exchange:        domain.ExchangeBinance,
		Current:         domain.FundingPoint{Time: now, Rate: 0.0001, Predicted: true},
		NextFundingTime: now.Add(time.Hour),
		IntervalHours:   8,
		History:         []domain.FundingPoint{{Time: now.Add(-8 * time.Hour), Rate: 0.00008}},
	}}
	byb := &fakeFunding{ser: &domain.FundingSeries{
		Exchange:        domain.ExchangeBybit,
		Current:         domain.FundingPoint{Time: now, Rate: 0.00005, Predicted: true},
		NextFundingTime: now.Add(time.Hour),
		IntervalHours:   8,
		History:         []domain.FundingPoint{{Time: now.Add(-8 * time.Hour), Rate: 0.00003}},
	}}
	svc := New(&fakeMarket{}, &fakeSupply{}).WithFundingRate(map[domain.Exchange]domain.FundingRatePort{
		domain.ExchangeBinance: bin,
		domain.ExchangeBybit:   byb,
	})
	got, err := svc.GetFundingRate(context.Background(), "all", "btc-usd", 12)
	if err != nil || got.VenueCount != 2 || got.Current != nil {
		t.Fatalf("%+v %v", got, err)
	}
	one, err := svc.GetFundingRate(context.Background(), "binance", "BTCUSDT", 0)
	if err != nil || one.Current == nil || one.Current.Payer != "long" {
		t.Fatalf("%+v %v", one, err)
	}
}

type fakeLS struct {
	ser *domain.LongShortSeries
	err error
}

func (f *fakeLS) GetLongShortSeries(_ context.Context, symbol string, _ int) (*domain.LongShortSeries, error) {
	if f.err != nil {
		return nil, f.err
	}
	cp := *f.ser
	cp.Symbol = symbol
	return &cp, nil
}

func TestGetLongShortRatio_Combined(t *testing.T) {
	now := time.Now().UTC()
	bin := &fakeLS{ser: &domain.LongShortSeries{
		Exchange: domain.ExchangeBinance,
		Current:  domain.LongShortPoint{Time: now, LongShare: 0.63, ShortShare: 0.37, Ratio: 1.70},
	}}
	byb := &fakeLS{ser: &domain.LongShortSeries{
		Exchange: domain.ExchangeBybit,
		Current:  domain.LongShortPoint{Time: now, LongShare: 0.58, ShortShare: 0.42, Ratio: 1.38},
	}}
	svc := New(&fakeMarket{}, &fakeSupply{}).WithLongShortRatio(map[domain.Exchange]domain.LongShortRatioPort{
		domain.ExchangeBinance: bin,
		domain.ExchangeBybit:   byb,
	})
	got, err := svc.GetLongShortRatio(context.Background(), "all", "btc-usd", 24)
	if err != nil || got.VenueCount != 2 || got.Current != nil {
		t.Fatalf("%+v %v", got, err)
	}
	one, err := svc.GetLongShortRatio(context.Background(), "binance", "BTCUSDT", 0)
	if err != nil || one.Current == nil || one.Current.Bias != "long" {
		t.Fatalf("%+v %v", one, err)
	}
}

func TestGetOpenInterest_AttachesFunding(t *testing.T) {
	now := time.Now().UTC()
	oi := &fakeOI{ser: &domain.OpenInterestSeries{
		Exchange: domain.ExchangeBinance,
		Current:  domain.OpenInterestPoint{Time: now, Contracts: 100, Value: 10000},
		History:  []domain.OpenInterestPoint{{Time: now.Add(-5 * time.Minute), Contracts: 90, Value: 9000}},
	}}
	fund := &fakeFunding{ser: &domain.FundingSeries{
		Exchange:      domain.ExchangeBinance,
		Current:       domain.FundingPoint{Time: now, Rate: 0.0001, Predicted: true},
		IntervalHours: 8,
	}}
	svc := New(&fakeMarket{}, &fakeSupply{}).
		WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{domain.ExchangeBinance: oi}).
		WithFundingRate(map[domain.Exchange]domain.FundingRatePort{domain.ExchangeBinance: fund})
	got, err := svc.GetOpenInterest(context.Background(), "binance", "BTCUSDT")
	if err != nil || got.Funding == nil || got.Funding.Current == nil {
		t.Fatalf("funding missing %+v %v", got, err)
	}
}

type fakeBasis struct {
	q   *domain.BasisQuote
	err error
}

func (f *fakeBasis) GetBasisQuote(_ context.Context, symbol string) (*domain.BasisQuote, error) {
	if f.err != nil {
		return nil, f.err
	}
	cp := *f.q
	cp.Symbol = symbol
	return &cp, nil
}

func TestGetBasis_PerVenueAndAgreement(t *testing.T) {
	bin := &fakeBasis{q: &domain.BasisQuote{
		Exchange: domain.ExchangeBinance, FuturesLast: 100.15, FuturesMark: 100.12, SpotIndex: 100,
	}}
	byb := &fakeBasis{q: &domain.BasisQuote{
		Exchange: domain.ExchangeBybit, FuturesLast: 100.12, FuturesMark: 100.10, SpotIndex: 100,
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{}).WithBasis(map[domain.Exchange]domain.BasisPort{
		domain.ExchangeBinance: bin, domain.ExchangeBybit: byb,
	})
	got, err := svc.GetBasis(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 2 || got.Agreement == nil || got.Agreement.Alignment != domain.AlignSame {
		t.Fatalf("%+v", got)
	}
	if got.Venues[0].Summary == "" {
		t.Fatal("missing summary")
	}
}

type intervalSeriesMarket struct {
	fakeMarket
	by map[string]map[domain.CandleInterval][]domain.Candle
}

func (m *intervalSeriesMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	m.lastQ = q
	if rows, ok := m.by[q.Symbol]; ok {
		if c, ok := rows[q.Interval]; ok {
			return c, nil
		}
	}
	return nil, domain.ErrNotFound
}

func synthCloses(start time.Time, n int, step time.Duration, px func(i int) float64) []domain.Candle {
	out := make([]domain.Candle, n)
	for i := 0; i < n; i++ {
		p := px(i)
		s := strconv.FormatFloat(p, 'f', 6, 64)
		t0 := start.Add(time.Duration(i) * step)
		out[i] = domain.Candle{
			OpenTime: t0, CloseTime: t0.Add(step),
			Open: s, High: s, Low: s, Close: s, Volume: "1", QuoteVolume: "1",
		}
	}
	return out
}

func TestGetCorrelation_FollowsBTCAndETH(t *testing.T) {
	start := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	n1m, n5m := 80, 300
	m := &intervalSeriesMarket{by: map[string]map[domain.CandleInterval][]domain.Candle{
		"SOLUSDT": {
			"1m": synthCloses(start, n1m, time.Minute, func(i int) float64 { return 20 + float64(i)*0.05 }),
			"5m": synthCloses(start, n5m, 5*time.Minute, func(i int) float64 { return 20 + float64(i)*0.06 }),
		},
		"BTCUSDT": {
			"1m": synthCloses(start, n1m, time.Minute, func(i int) float64 { return 100 + float64(i)*0.2 }),
			"5m": synthCloses(start, n5m, 5*time.Minute, func(i int) float64 { return 100 + float64(i)*0.3 }),
		},
		"ETHUSDT": {
			"1m": synthCloses(start, n1m, time.Minute, func(i int) float64 { return 50 + float64(i)*0.08 }),
			"5m": synthCloses(start, n5m, 5*time.Minute, func(i int) float64 { return 50 + float64(i)*0.1 }),
		},
	}}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetCorrelation(context.Background(), "binance", "SOLUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary == "" || len(got.Windows) != 3 {
		t.Fatalf("%+v", got)
	}
	for _, w := range got.Windows {
		if !w.BTC.Complete || w.BTC.Relation != domain.CorrRelationFollows {
			t.Fatalf("window %s btc %+v", w.Window, w.BTC)
		}
		if !w.ETH.Complete || w.ETH.Relation != domain.CorrRelationFollows {
			t.Fatalf("window %s eth %+v", w.Window, w.ETH)
		}
	}
}

func TestGetCorrelation_BTCIsSelf(t *testing.T) {
	start := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	m := &intervalSeriesMarket{by: map[string]map[domain.CandleInterval][]domain.Candle{
		"BTCUSDT": {
			"1m": synthCloses(start, 80, time.Minute, func(i int) float64 { return 100 + float64(i) }),
			"5m": synthCloses(start, 300, 5*time.Minute, func(i int) float64 { return 100 + float64(i) }),
		},
		"ETHUSDT": {
			"1m": synthCloses(start, 80, time.Minute, func(i int) float64 { return 50 + float64(i)*0.4 }),
			"5m": synthCloses(start, 300, 5*time.Minute, func(i int) float64 { return 50 + float64(i)*0.4 }),
		},
	}}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetCorrelation(context.Background(), "", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Windows[0].BTC.Self {
		t.Fatalf("%+v", got.Windows[0].BTC)
	}
}

func TestGetCorrelation_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetCorrelation(context.Background(), "binance", "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetVolatility_MoreJumpyThanBTC(t *testing.T) {
	start := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	n1m, n5m := 400, 400
	m := &intervalSeriesMarket{by: map[string]map[domain.CandleInterval][]domain.Candle{
		"SOLUSDT": {
			"1m": synthCloses(start, n1m, time.Minute, func(i int) float64 { return 20 + float64(i%5)*0.8 }),
			"5m": synthCloses(start, n5m, 5*time.Minute, func(i int) float64 { return 20 + float64(i%5)*1.2 }),
		},
		"BTCUSDT": {
			"1m": synthCloses(start, n1m, time.Minute, func(i int) float64 { return 100 + float64(i)*0.001 }),
			"5m": synthCloses(start, n5m, 5*time.Minute, func(i int) float64 { return 100 + float64(i)*0.002 }),
		},
		"ETHUSDT": {
			"1m": synthCloses(start, n1m, time.Minute, func(i int) float64 { return 50 + float64(i)*0.0005 }),
			"5m": synthCloses(start, n5m, 5*time.Minute, func(i int) float64 { return 50 + float64(i)*0.001 }),
		},
	}}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetVolatility(context.Background(), "binance", "SOLUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary == "" || len(got.Windows) != 3 {
		t.Fatalf("%+v", got)
	}
	if !got.Windows[0].Coin.Complete || got.Windows[0].VsMarket != domain.VolVsMore {
		t.Fatalf("1h %+v", got.Windows[0])
	}
}

func TestGetVolatility_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetVolatility(context.Background(), "binance", "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetSnapshot_PriceAndOITogether(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	start := now.Add(-48 * time.Hour)
	m := &intervalSeriesMarket{
		fakeMarket: fakeMarket{ticker: &domain.Ticker24h{Symbol: "SOLUSDT", LastPrice: "100", QuoteVolume: "5000"}},
		by: map[string]map[domain.CandleInterval][]domain.Candle{
			"SOLUSDT": {"1h": synthCloses(start, 50, time.Hour, func(i int) float64 { return 100 + float64(i%3)*0.1 })},
		},
	}
	oi := &fakeOI{ser: &domain.OpenInterestSeries{
		Exchange: domain.ExchangeBinance,
		Current:  domain.OpenInterestPoint{Time: now, Contracts: 120, Value: 12000},
		History:  []domain.OpenInterestPoint{{Time: now.Add(-time.Hour), Contracts: 100, Value: 10000}},
	}}
	tk := &fakeTaker{flow: &domain.TakerVenueFlow{
		Exchange: domain.ExchangeBinance,
		Windows:  []domain.TakerWindowFlow{domain.SummarizeTakerWindow(80, 20, domain.TakerWindow1h, true)},
	}}
	svc := New(m, &fakeSupply{}).
		WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{domain.ExchangeBinance: oi}).
		WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{domain.ExchangeBinance: tk})
	got, err := svc.GetSnapshot(context.Background(), "binance", "SOLUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Spot.Price != 100 || len(got.Venues) != 1 || got.Summary == "" {
		t.Fatalf("%+v", got)
	}
	if got.Venues[0].OIValue != 12000 {
		t.Fatalf("oi %+v", got.Venues[0])
	}
}

func TestGetSnapshot_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetSnapshot(context.Background(), "all", "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetLevels_ReturnsReport(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	// Bounce off ~100 several times, finish higher.
	px := func(i int) float64 {
		cycle := i % 7
		if cycle == 3 || cycle == 4 {
			return 100
		}
		return 100 + float64(cycle)*1.5
	}
	m := &intervalSeriesMarket{
		fakeMarket: fakeMarket{ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "110"}},
		by: map[string]map[domain.CandleInterval][]domain.Candle{
			"BTCUSDT": {"1h": synthCloses(start, 80, time.Hour, px)},
		},
	}
	svc := New(m, &fakeSupply{})
	got, err := svc.GetLevels(context.Background(), "binance", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Price != 110 || got.Summary == "" {
		t.Fatalf("%+v", got)
	}
}

func TestGetLevels_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetLevels(context.Background(), "binance", "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetWhales_SortsBiggestAndFlagsSmallCap(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	fm := &fakeMarket{
		ticker: &domain.Ticker24h{LastPrice: "100"},
		tickers: map[string]*domain.Ticker24h{
			"PEPEUSDT": {Symbol: "PEPEUSDT", LastPrice: "0.000001"},
			"BTCUSDT":  {Symbol: "BTCUSDT", LastPrice: "60000"},
		},
		spot: []domain.SpotMarket{
			{Symbol: "PEPEUSDT", BaseAsset: "PEPE", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "9000", LastPrice: "0.000001"},
			{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "8000", LastPrice: "60000"},
		},
		prints: map[string][]domain.TakerPrint{
			"PEPEUSDT": {{
				Exchange: domain.ExchangeBinance, Symbol: "PEPEUSDT", Side: domain.TakerSideBuy,
				Price: 0.000001, Quantity: 5e11, Notional: 500_000, Time: t0,
			}},
			"BTCUSDT": {{
				Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.TakerSideSell,
				Price: 60000, Quantity: 2, Notional: 120_000, Time: t0,
			}},
		},
	}
	pepeCirc := 10_000_000_000.0
	btcCirc := 19_000_000.0
	svc := New(fm, &fakeSupply{byAsset: map[string]*domain.AssetSupply{
		"PEPE": {Asset: "PEPE", CirculatingSupply: &pepeCirc},
		"BTC":  {Asset: "BTC", CirculatingSupply: &btcCirc},
	}})
	got, err := svc.GetWhales(context.Background(), "binance", "PEPEUSDT", 50_000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Events) != 1 || !got.Events[0].Unusual || got.Events[0].Notional != 500_000 {
		t.Fatalf("%+v", got)
	}
	if got.Events[0].AvgPrice <= 0 || got.Events[0].FirstTime.IsZero() {
		t.Fatalf("avg/time %+v", got.Events[0])
	}

	scan, err := svc.GetWhales(context.Background(), "binance", "", 50_000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Events) < 2 || scan.Events[0].Symbol != "PEPEUSDT" || scan.Events[1].Symbol != "BTCUSDT" {
		t.Fatalf("scan %+v", scan.Events)
	}
	if !scan.Events[0].Unusual {
		t.Fatalf("pepe should be unusual vs tiny mcap %+v", scan.Events[0])
	}
}

func TestGetWhales_IncludesLiquidation(t *testing.T) {
	now := time.Now().UTC()
	book := domain.NewLiquidationBook()
	book.Record(domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 63000, Quantity: 4, Notional: 252_000, Time: now.Add(-time.Minute),
	})
	svc := New(&fakeMarket{}, &fakeSupply{}).WithLiquidations(book, nil)
	got, err := svc.GetWhales(context.Background(), "binance", "BTCUSDT", 100_000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Events) != 1 || got.Events[0].Kind != domain.WhaleKindLiquidation || got.Events[0].Side != domain.WhaleSideLong {
		t.Fatalf("%+v", got)
	}
}

type seqBookMarket struct {
	fakeMarket
	books []*domain.RawOrderBook
	i     int
}

func (s *seqBookMarket) GetOrderBook(_ context.Context, _ domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	if len(s.books) == 0 {
		return sampleRawBook("BTCUSDT"), nil
	}
	b := s.books[s.i]
	if s.i < len(s.books)-1 {
		s.i++
	}
	return b, nil
}

func TestGetIcebergs_DetectsAskRefill(t *testing.T) {
	ask := func(qty float64) *domain.RawOrderBook {
		return &domain.RawOrderBook{
			Symbol: "BTCUSDT",
			Bids:   []domain.PriceLevel{{Price: 63900, Quantity: 1}},
			Asks:   []domain.PriceLevel{{Price: 64000, Quantity: qty}, {Price: 64100, Quantity: 1}},
			Live:   true,
		}
	}
	m := &seqBookMarket{books: []*domain.RawOrderBook{
		ask(10), ask(1), ask(10), ask(0.8), ask(10),
	}}
	svc := New(m, &fakeSupply{})
	for i := 0; i < 5; i++ {
		if _, err := svc.GetSpotOrderBook(context.Background(), "binance", "BTCUSDT", "", 20, 2); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.GetIcebergs(context.Background(), "binance", "BTCUSDT", 25_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Asks) == 0 || got.Asks[0].Refills < 2 {
		t.Fatalf("%+v", got)
	}
}

func TestGetIcebergs_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetIcebergs(context.Background(), "binance", "  ", 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetBookHistory_NotConfigured(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetBookHistory(context.Background(), "binance", "BTCUSDT", nil, nil, nil, 0)
	if !errors.Is(err, domain.ErrUpstream) {
		t.Fatalf("%v", err)
	}
}

type stubBookHist struct {
	snap domain.BookHistorySnapshot
}

func (s stubBookHist) SnapshotAt(_ context.Context, exchange, symbol string, at time.Time) (*domain.BookHistorySnapshot, error) {
	out := s.snap
	out.Exchange = domain.Exchange(exchange)
	out.Symbol = symbol
	if out.SampledAt.IsZero() {
		out.SampledAt = at
	}
	return &out, nil
}
func (s stubBookHist) List(_ context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	out := s.snap
	out.Symbol = q.Symbol
	return []domain.BookHistorySnapshot{out}, nil
}
func (s stubBookHist) Compare(_ context.Context, exchange, symbol string, from, to time.Time) (*domain.BookHistoryDiff, error) {
	a := s.snap
	a.SampledAt, a.Symbol, a.Exchange = from, symbol, domain.Exchange(exchange)
	b := s.snap
	b.SampledAt, b.Symbol, b.BidNotional = to, symbol, s.snap.BidNotional+1000
	b.Bids = []domain.BookHistoryLevel{{Price: 100, Quantity: 2, Notional: 200}}
	diff := domain.CompareBookHistory(a, b)
	return &diff, nil
}
func (stubBookHist) Note(string, string) {}

func TestGetBookHistory_AtAndList(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	svc := New(&fakeMarket{}, &fakeSupply{}).WithBookHistory(stubBookHist{snap: domain.BookHistorySnapshot{
		Mid: 100, Spread: 0.2, BidNotional: 500, AskNotional: 400, SampledAt: t0,
	}})
	at := t0
	got, err := svc.GetBookHistory(context.Background(), "binance", "BTCUSDT", &at, nil, nil, 0)
	if err != nil || got.Snapshot == nil || got.Snapshot.Mid != 100 || got.Summary == "" {
		t.Fatalf("%+v %v", got, err)
	}
	list, err := svc.GetBookHistory(context.Background(), "binance", "BTCUSDT", nil, nil, nil, 10)
	if err != nil || len(list.Snapshots) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
}

func TestCompareBookHistory_RequiresTimes(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{}).WithBookHistory(stubBookHist{})
	_, err := svc.CompareBookHistory(context.Background(), "binance", "BTCUSDT", time.Time{}, time.Now())
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	got, err := svc.CompareBookHistory(context.Background(), "binance", "btcusdt", t0, t0.Add(5*time.Minute))
	if err != nil || got.Summary == "" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestGetWhales_BadExchange(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetWhales(context.Background(), "coinbase", "BTCUSDT", 0, 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

type fakeWindows struct {
	byWindow map[string][]domain.WindowChange
}

func (f *fakeWindows) GetWindowChanges(_ context.Context, window string, _ []string) ([]domain.WindowChange, error) {
	return f.byWindow[window], nil
}

func TestGetBreadth_CountsAndCarrying(t *testing.T) {
	names := []string{"BTC", "ETH", "SOL", "XRP", "DOGE", "ADA", "AVAX", "LINK", "DOT", "UNI", "ATOM", "NEAR"}
	spot := make([]domain.SpotMarket, 0, len(names))
	chg1h := make([]domain.WindowChange, 0, len(names))
	for i, b := range names {
		pct24 := "-1"
		if b == "BTC" || b == "ETH" {
			pct24 = "2"
		}
		spot = append(spot, domain.SpotMarket{
			Symbol: b + "USDT", BaseAsset: b, QuoteAsset: "USDT", Status: "TRADING",
			PriceChangePercent: pct24, QuoteVolume: strconv.Itoa(100 - i),
		})
		ch := -0.8
		if b == "BTC" || b == "ETH" {
			ch = 1.5
		}
		chg1h = append(chg1h, domain.WindowChange{Symbol: b + "USDT", ChangePct: ch})
	}
	svc := New(&fakeMarket{spot: spot}, &fakeSupply{}).WithWindowChanges(map[domain.Exchange]domain.WindowChangePort{
		domain.ExchangeBinance: &fakeWindows{byWindow: map[string][]domain.WindowChange{
			domain.BreadthWindow1h: chg1h,
			domain.BreadthWindow4h: chg1h,
		}},
	})
	got, err := svc.GetBreadth(context.Background(), "binance", 20)
	if err != nil {
		t.Fatal(err)
	}
	if got.Universe < 10 || len(got.Windows) != 3 || got.Summary == "" {
		t.Fatalf("%+v", got)
	}
	var h1 domain.BreadthWindow
	for _, w := range got.Windows {
		if w.Window == domain.BreadthWindow1h {
			h1 = w
		}
	}
	if h1.Alignment != domain.BreadthAlignCarrying {
		t.Fatalf("1h %+v", h1)
	}
}

type fakeTaker struct {
	flow    *domain.TakerVenueFlow
	buckets []domain.TakerBucket
	err     error
}

func (f *fakeTaker) GetTakerFlow(_ context.Context, symbol string) (*domain.TakerVenueFlow, error) {
	if f.err != nil {
		return nil, f.err
	}
	cp := *f.flow
	cp.Symbol = symbol
	return &cp, nil
}

func (f *fakeTaker) GetTakerBuckets(_ context.Context, symbol string) ([]domain.TakerBucket, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]domain.TakerBucket(nil), f.buckets...), nil
}

func TestGetCVD_PerVenueAndCombined(t *testing.T) {
	t0 := time.Now().UTC().Add(-55 * time.Minute).Truncate(5 * time.Minute)
	var bn, by []domain.TakerBucket
	var candles []domain.Candle
	for i := 0; i < 12; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		bn = append(bn, domain.TakerBucket{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: at, BuyNotional: 100, SellNotional: 20})
		by = append(by, domain.TakerBucket{Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Start: at, BuyNotional: 10, SellNotional: 40})
		candles = append(candles, domain.Candle{OpenTime: at, Close: "64000", QuoteVolume: "200", TakerBuyQuote: "150"})
	}
	m := &intervalSeriesMarket{
		fakeMarket: fakeMarket{},
		by:         map[string]map[domain.CandleInterval][]domain.Candle{"BTCUSDT": {"5m": candles}},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: m, domain.ExchangeBybit: m,
	}, &fakeSupply{}).WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: &fakeTaker{buckets: bn, flow: &domain.TakerVenueFlow{}},
		domain.ExchangeBybit:   &fakeTaker{buckets: by, flow: &domain.TakerVenueFlow{}},
	})
	got, err := svc.GetCVD(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 2 || got.Combined == nil || len(got.Combined.Points) == 0 {
		t.Fatalf("%+v", got)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
	if got.Combined.Complete {
		t.Fatal("1h overlap must not mark combined complete")
	}
	if len(got.Combined.Contributions) != 2 {
		t.Fatalf("contributions %+v", got.Combined.Contributions)
	}
	if len(got.SpotVenues) == 0 {
		t.Fatal("spot venues")
	}
	if got.SpotFutures == nil {
		t.Fatal("spot vs futures")
	}
}

func TestGetCVD_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetCVD(context.Background(), "all", "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetTakerFlow_PerVenueAndCombined(t *testing.T) {
	bin := &fakeTaker{flow: &domain.TakerVenueFlow{
		Exchange: domain.ExchangeBinance,
		Windows: []domain.TakerWindowFlow{
			domain.SummarizeTakerWindow(300, 100, domain.TakerWindow5m, true),
			domain.SummarizeTakerWindow(800, 400, domain.TakerWindow1h, true),
			domain.SummarizeTakerWindow(2000, 1500, domain.TakerWindow4h, true),
		},
		Dominant: domain.TakerSideBuy,
	}}
	byb := &fakeTaker{flow: &domain.TakerVenueFlow{
		Exchange: domain.ExchangeBybit,
		Windows: []domain.TakerWindowFlow{
			domain.SummarizeTakerWindow(50, 150, domain.TakerWindow5m, true),
			domain.SummarizeTakerWindow(200, 200, domain.TakerWindow1h, true),
			domain.SummarizeTakerWindow(400, 500, domain.TakerWindow4h, true),
		},
		Dominant: domain.TakerSideSell,
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{}).WithTakerFlow(map[domain.Exchange]domain.TakerFlowPort{
		domain.ExchangeBinance: bin, domain.ExchangeBybit: byb,
	})
	got, err := svc.GetTakerFlow(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 2 || got.Combined == nil {
		t.Fatalf("%+v", got)
	}
	if got.Combined.Windows[0].BuyNotional != 350 || got.Combined.Windows[0].SellNotional != 250 {
		t.Fatalf("combined 5m %+v", got.Combined.Windows[0])
	}
	if got.Venues[0].Summary == "" {
		t.Fatal("missing summary")
	}
}

func TestGetVenueDivergence_Opposite(t *testing.T) {
	now := time.Now().UTC()
	binCandles := make([]domain.Candle, 0, 25)
	bybCandles := make([]domain.Candle, 0, 25)
	for i := 0; i < 25; i++ {
		t0 := now.Add(-time.Duration(24-i) * time.Hour)
		binCandles = append(binCandles, domain.Candle{OpenTime: t0, Close: strconv.FormatFloat(100+float64(i)*0.3, 'f', 2, 64)})
		bybCandles = append(bybCandles, domain.Candle{OpenTime: t0, Close: strconv.FormatFloat(107-float64(i)*0.2, 'f', 2, 64)})
	}
	binM := &fakeMarket{candles: binCandles, ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "107", PriceChangePercent: "4"}}
	bybM := &fakeMarket{candles: bybCandles, ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "102", PriceChangePercent: "-4"}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: binM, domain.ExchangeBybit: bybM,
	}, &fakeSupply{}).
		WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
			domain.ExchangeBinance: &fakeOI{ser: &domain.OpenInterestSeries{
				Current: domain.OpenInterestPoint{Time: now, Contracts: 120, Value: 12e6},
				History: []domain.OpenInterestPoint{{Time: now.Add(-4 * time.Hour), Contracts: 100, Value: 10e6}},
			}},
			domain.ExchangeBybit: &fakeOI{ser: &domain.OpenInterestSeries{
				Current: domain.OpenInterestPoint{Time: now, Contracts: 80, Value: 8e6},
				History: []domain.OpenInterestPoint{{Time: now.Add(-4 * time.Hour), Contracts: 70, Value: 7e6}},
			}},
		}).
		WithLongShortRatio(map[domain.Exchange]domain.LongShortRatioPort{
			domain.ExchangeBinance: &fakeLS{ser: &domain.LongShortSeries{
				Current: domain.LongShortPoint{Time: now, LongShare: 0.66, ShortShare: 0.34, Ratio: 1.9},
			}},
			domain.ExchangeBybit: &fakeLS{ser: &domain.LongShortSeries{
				Current: domain.LongShortPoint{Time: now, LongShare: 0.34, ShortShare: 0.66, Ratio: 0.5},
			}},
		}).
		WithFundingRate(map[domain.Exchange]domain.FundingRatePort{
			domain.ExchangeBinance: &fakeFunding{ser: &domain.FundingSeries{
				Current: domain.FundingPoint{Time: now, Rate: 0.0002, Predicted: true},
			}},
			domain.ExchangeBybit: &fakeFunding{ser: &domain.FundingSeries{
				Current: domain.FundingPoint{Time: now, Rate: -0.0002, Predicted: true},
			}},
		})
	got, err := svc.GetVenueDivergence(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Alignment == domain.AlignUnknown {
		t.Fatalf("%+v", got)
	}
	if got.Summary == "" || len(got.Diffs) < 3 {
		t.Fatalf("%+v", got)
	}
}

func TestGetPositioning_PerVenueAndCombined(t *testing.T) {
	now := time.Now().UTC()
	// Build candles via fakeMarket defaults if any; set ticker for price.
	binOI := &fakeOI{ser: &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Contracts: 110, Value: 11_000_000},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-time.Hour), Contracts: 100, Value: 10_000_000},
			{Time: now.Add(-4 * time.Hour), Contracts: 90, Value: 9_000_000},
			{Time: now.Add(-24 * time.Hour), Contracts: 80, Value: 8_000_000},
		},
	}}
	bybOI := &fakeOI{ser: &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Contracts: 55, Value: 5_500_000},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-4 * time.Hour), Contracts: 50, Value: 5_000_000},
		},
	}}
	// Rising prices over 24h candles so price↑ OI↑ → long buildup when candles exist.
	candles := make([]domain.Candle, 0, 25)
	for i := 0; i < 25; i++ {
		candles = append(candles, domain.Candle{
			OpenTime: now.Add(-time.Duration(24-i) * time.Hour),
			Close:    strconv.FormatFloat(100+float64(i)*0.2, 'f', 2, 64),
		})
	}
	fm := &fakeMarket{
		candles: candles,
		ticker:  &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "105", PriceChangePercent: "4.5"},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fm,
		domain.ExchangeBybit:   fm,
	}, &fakeSupply{}).
		WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
			domain.ExchangeBinance: binOI, domain.ExchangeBybit: bybOI,
		}).
		WithLongShortRatio(map[domain.Exchange]domain.LongShortRatioPort{
			domain.ExchangeBinance: &fakeLS{ser: &domain.LongShortSeries{
				Current: domain.LongShortPoint{Time: now, LongShare: 0.6, ShortShare: 0.4, Ratio: 1.5},
			}},
		}).
		WithFundingRate(map[domain.Exchange]domain.FundingRatePort{
			domain.ExchangeBinance: &fakeFunding{ser: &domain.FundingSeries{
				Current: domain.FundingPoint{Time: now, Rate: 0.0001, Predicted: true},
			}},
		})
	got, err := svc.GetPositioning(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 2 || got.Combined == nil {
		t.Fatalf("%+v", got)
	}
	// OI is rising; price candles rise → expect long buildup on at least one venue or combined.
	if got.Combined.Regime == "" {
		t.Fatalf("empty combined %+v", got.Combined)
	}
}

func TestGetPositioning_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetPositioning(context.Background(), "all", " ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetSqueezeRisk_PerVenueAndCombined(t *testing.T) {
	now := time.Now().UTC()
	binOI := &fakeOI{ser: &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Contracts: 100, Value: 9_000_000},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-time.Hour), Contracts: 90, Value: 8_100_000},
			{Time: now.Add(-4 * time.Hour), Contracts: 80, Value: 7_200_000},
		},
	}}
	bybOI := &fakeOI{ser: &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Contracts: 20, Value: 1_000_000},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-time.Hour), Contracts: 18, Value: 900_000},
		},
	}}
	binLS := &fakeLS{ser: &domain.LongShortSeries{
		Current: domain.LongShortPoint{Time: now, LongShare: 0.68, ShortShare: 0.32, Ratio: 2.125},
	}}
	bybLS := &fakeLS{ser: &domain.LongShortSeries{
		Current: domain.LongShortPoint{Time: now, LongShare: 0.55, ShortShare: 0.45, Ratio: 1.22},
	}}
	binFund := &fakeFunding{ser: &domain.FundingSeries{
		Current: domain.FundingPoint{Time: now, Rate: 0.00025, Predicted: true},
		History: []domain.FundingPoint{{Time: now.Add(-8 * time.Hour), Rate: 0.0002}},
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "100", PriceChangePercent: "1.5"}},
		domain.ExchangeBybit:   &fakeMarket{ticker: &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "100", PriceChangePercent: "1.2"}},
	}, &fakeSupply{}).
		WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
			domain.ExchangeBinance: binOI, domain.ExchangeBybit: bybOI,
		}).
		WithLongShortRatio(map[domain.Exchange]domain.LongShortRatioPort{
			domain.ExchangeBinance: binLS, domain.ExchangeBybit: bybLS,
		}).
		WithFundingRate(map[domain.Exchange]domain.FundingRatePort{
			domain.ExchangeBinance: binFund,
		})
	got, err := svc.GetSqueezeRisk(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 2 || got.Combined == nil {
		t.Fatalf("%+v", got)
	}
	if got.Combined.DominantVenue != "binance" {
		t.Fatalf("dom %s", got.Combined.DominantVenue)
	}
	if got.Combined.LongSqueeze.Score <= 0 {
		t.Fatalf("combined long score %+v", got.Combined)
	}
}

func TestGetSqueezeRisk_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetSqueezeRisk(context.Background(), "all", "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetLiquidationHunt_PerVenueFailSoft(t *testing.T) {
	now := time.Now().UTC()
	binOI := &fakeOI{ser: &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Contracts: 10, Value: 1_000_000},
	}}
	bybOI := &fakeOI{err: errors.New("bybit down")}
	binLS := &fakeLS{ser: &domain.LongShortSeries{
		Current: domain.LongShortPoint{Time: now, LongShare: 0.4, ShortShare: 0.6, Ratio: 0.67},
	}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{}).
		WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
			domain.ExchangeBinance: binOI,
			domain.ExchangeBybit:   bybOI,
		}).
		WithLongShortRatio(map[domain.Exchange]domain.LongShortRatioPort{
			domain.ExchangeBinance: binLS,
		})
	got, err := svc.GetLiquidationHunt(context.Background(), "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Venues) != 2 {
		t.Fatalf("venues %+v", got.Venues)
	}
	var bin *domain.HuntVenueReport
	for i := range got.Venues {
		if got.Venues[i].Exchange == domain.ExchangeBinance {
			bin = &got.Venues[i]
		}
	}
	if bin == nil || bin.OpenInterestValue != 1_000_000 || bin.Price <= 0 {
		t.Fatalf("binance %+v", bin)
	}
	if bin.UpHunt.Thesis == "" || bin.DownHunt.Thesis == "" {
		t.Fatalf("missing scenarios %+v", bin)
	}
}

func TestGetLiquidationHunt_BadSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.GetLiquidationHunt(context.Background(), "all", "  ")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetRawOrderBook(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  &fakeMarket{},
		domain.ExchangeCoinbase: &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.GetRawOrderBook(context.Background(), "binance", "btcusdt")
	if err != nil || got == nil || len(got.Asks) == 0 {
		t.Fatalf("%+v %v", got, err)
	}
	cb, err := svc.GetRawOrderBook(context.Background(), "coinbase", "BTCUSDT")
	if err != nil || cb == nil {
		t.Fatalf("%v", err)
	}
	_, err = svc.GetRawOrderBook(context.Background(), "binance", "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestEstimateOrderBookImpact_Buy(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  &fakeMarket{},
		domain.ExchangeCoinbase: &fakeMarket{},
		domain.ExchangeBybit:    &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.EstimateOrderBookImpact(context.Background(), "binance", "BTCUSDT", "buy", 1.5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Exhausted || got.Scope != "binance" || got.FilledQuantity == "" {
		t.Fatalf("%+v", got)
	}
	avg, _ := strconv.ParseFloat(got.AveragePrice, 64)
	// sample asks: 1@100.06 + 0.5@100.20 — first ask gone, new best 100.20
	if avg < 100.1 || avg > 100.15 {
		t.Fatalf("avg %s", got.AveragePrice)
	}
	if !got.ImpactAvailable || got.NewBestPrice != "100.2" || got.ImpactPct <= 0 {
		t.Fatalf("touch move %+v", got)
	}
	tiny, err := svc.EstimateOrderBookImpact(context.Background(), "binance", "BTCUSDT", "buy", 0.25, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !tiny.ImpactAvailable || tiny.ImpactPct != 0 {
		t.Fatalf("partial best must be 0 impact, got %+v", tiny)
	}
	big, err := svc.EstimateOrderBookImpact(context.Background(), "all", "BTCUSDT", "buy", 0, 1e9)
	if err != nil {
		t.Fatal(err)
	}
	if big.Scope != domain.ImpactScopeCombined || big.VenueCount != 3 {
		t.Fatalf("combined %+v", big)
	}
	if big.ImpactAvailable || big.ImpactNote == "" {
		t.Fatalf("oversize must not invent impact: %+v", big)
	}
}

func TestEstimateOrderBookImpact_RejectsBothSizes(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	_, err := svc.EstimateOrderBookImpact(context.Background(), "binance", "BTCUSDT", "buy", 1, 100)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
	_, err = svc.EstimateOrderBookImpact(context.Background(), "binance", "BTCUSDT", "buy", 0, 0)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
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

func TestCmpFloatStringNullsLast_EmptyNotZero(t *testing.T) {
	// Empty / bad strings must sort last under both orders, never as zero.
	items := []domain.SpotMarket{
		{Symbol: "A", QuoteVolume: ""},
		{Symbol: "B", QuoteVolume: "100"},
		{Symbol: "C", QuoteVolume: "0"},
		{Symbol: "D", QuoteVolume: "bad"},
	}
	sortSpotMarkets(items, domain.SpotSortQuoteVolume, domain.SortDesc)
	if items[0].Symbol != "B" {
		t.Fatalf("want B first, order=%v", symbolsOf(items))
	}
	// Defined zeros before missing: C then A,D (ties by symbol among missing)
	if items[1].Symbol != "C" {
		t.Fatalf("zero before missing, got %v", symbolsOf(items))
	}
	if items[2].Symbol != "A" || items[3].Symbol != "D" {
		t.Fatalf("missing last by symbol: %v", symbolsOf(items))
	}

	// Asc: defined first, missing last
	items = []domain.SpotMarket{
		{Symbol: "A", QuoteVolume: ""},
		{Symbol: "B", QuoteVolume: "100"},
		{Symbol: "C", QuoteVolume: "0"},
	}
	sortSpotMarkets(items, domain.SpotSortQuoteVolume, domain.SortAsc)
	if items[0].Symbol != "C" || items[1].Symbol != "B" || items[2].Symbol != "A" {
		t.Fatalf("asc nulls last: %v", symbolsOf(items))
	}
}

func TestCmpTags_OrderIndependent(t *testing.T) {
	// Same tag set in different order must compare equal (sorted join).
	a := []string{"Meme", "defi"}
	b := []string{"defi", "Meme"}
	if cmpTags(a, b, false) != 0 {
		t.Fatal("tag order must not affect sort key")
	}
}

func TestNormalizeSymbolForExchange_Coinbase(t *testing.T) {
	if got := normalizeSymbolForExchange(domain.ExchangeCoinbase, "btcusd"); got != "BTC-USD" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeSymbolForExchange(domain.ExchangeBinance, "btc-usdt"); got != "BTCUSDT" {
		t.Fatalf("got %q", got)
	}
}

func symbolsOf(items []domain.SpotMarket) []string {
	out := make([]string, len(items))
	for i, m := range items {
		out[i] = m.Symbol
	}
	return out
}
