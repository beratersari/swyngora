package market

import (
	"context"
	"errors"
	"strconv"
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
	if f.err != nil {
		return nil, f.err
	}
	return f.ticker, nil
}

func (f *fakeMarket) GetOrderBook(_ context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
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
