package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Long-lived meta caches: exchangeInfo and product tags change rarely.
// Price/volume freshness is controlled by SpotMarketCache TTL (short).
const (
	exchangeInfoCacheTTL = 10 * time.Minute
	nonCryptoBasesTTL    = 1 * time.Hour
	// multiCallTimeout bounds detached singleflight work (spot list / supply refresh).
	multiCallTimeout = 45 * time.Second
)

// Client implements domain.MarketDataPort and domain.SupplyPort against Binance.
// No API key is required for the market-data and product-catalog endpoints used here.
type Client struct {
	baseURL        string // official Spot REST (api.binance.com)
	productBaseURL string // product catalog host (www.binance.com bapi)
	apiKey         string // optional; required for delist schedule
	httpClient     *http.Client
	candles        *cache.TTL[[]domain.Candle]
	tickers        *cache.TTL[*domain.Ticker24h]
	orderBooks     *cache.TTL[*domain.RawOrderBook]
	spotMarkets    *cache.TTL[[]domain.SpotMarket] // joined list (short TTL → live prices)
	// Layered meta so a short price refresh does NOT re-download exchangeInfo / product catalog
	// (those were the main reason the UI saw ~30–40s between real updates).
	exchangeSpot *cache.TTL[[]spotSymbolMeta]
	// productMeta holds non-crypto exclusion bases + tags-by-base from one catalog fetch.
	productMeta *cache.TTL[*productMetaSnapshot]
	supply      *cache.TTL[*domain.AssetSupply]
	spotSF      singleflight.Group
	supplySF    singleflight.Group
	metaSF      singleflight.Group
	candleSF    singleflight.Group
	tickerSF    singleflight.Group
	orderBookSF singleflight.Group
	depth       *DepthHub
	depthOnce   sync.Once
	depthWait   time.Duration
	wsURL       string
	wsDial      wsDialer
	depthIdle   time.Duration
	futuresBase string // USD-M REST (fapi.binance.com)
	oiCache     *cache.TTL[*domain.OpenInterestSeries]
	oiSF        singleflight.Group
}

// Options configures the Binance client.
type Options struct {
	BaseURL         string
	ProductBaseURL  string // default https://www.binance.com — product catalog with circulating supply
	APIKey          string // optional BINANCE_API_KEY for delist schedule
	HTTPClient      *http.Client
	CandleCache     *cache.TTL[[]domain.Candle]
	TickerCache     *cache.TTL[*domain.Ticker24h]
	OrderBookCache  *cache.TTL[*domain.RawOrderBook]
	SpotMarketCache *cache.TTL[[]domain.SpotMarket]
	SupplyCache     *cache.TTL[*domain.AssetSupply]
	WSURL               string
	WSDial              wsDialer
	DepthIdle           time.Duration
	DepthWait           time.Duration
	FuturesBaseURL      string // default https://fapi.binance.com
	OpenInterestCache   *cache.TTL[*domain.OpenInterestSeries]
}

// NewClient constructs a Binance market-data + supply client.
func NewClient(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = "https://api.binance.com"
	}
	productBase := strings.TrimRight(opts.ProductBaseURL, "/")
	if productBase == "" {
		productBase = "https://www.binance.com"
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	wait := opts.DepthWait
	if wait <= 0 {
		wait = 8 * time.Second
	}
	c := &Client{
		baseURL:        base,
		productBaseURL: productBase,
		apiKey:         strings.TrimSpace(opts.APIKey),
		httpClient:     hc,
		candles:        opts.CandleCache,
		tickers:        opts.TickerCache,
		orderBooks:     opts.OrderBookCache,
		spotMarkets:    opts.SpotMarketCache,
		exchangeSpot:   cache.New[[]spotSymbolMeta](exchangeInfoCacheTTL),
		productMeta:    cache.New[*productMetaSnapshot](nonCryptoBasesTTL),
		supply:         opts.SupplyCache,
		depthWait:      wait,
		wsURL:          opts.WSURL,
		wsDial:         opts.WSDial,
		depthIdle:      opts.DepthIdle,
		futuresBase:    strings.TrimRight(opts.FuturesBaseURL, "/"),
		oiCache:        opts.OpenInterestCache,
	}
	if c.futuresBase == "" {
		c.futuresBase = "https://fapi.binance.com"
	}
	if c.oiCache == nil {
		c.oiCache = cache.New[*domain.OpenInterestSeries](30 * time.Second)
	}
	return c
}

func (c *Client) ensureDepth() {
	c.depthOnce.Do(func() {
		c.depth = newDepthHub(c.fetchDepthSnapshot, c.wsURL, c.wsDial, c.depthIdle)
	})
}

// Close stops live order-book streams.
func (c *Client) Close() {
	if c == nil || c.depth == nil {
		return
	}
	c.depth.Close()
}

// productMetaSnapshot is derived from the Binance product catalog (long-lived).
type productMetaSnapshot struct {
	NonCryptoBases []string
	// TagsByBase maps uppercased base asset → sorted unique product tags.
	TagsByBase map[string][]string
	// AllTags is the sorted unique set of crypto product tags (for filter UI).
	AllTags []string
}

// spotSymbolMeta is the static half of a spot row (from exchangeInfo).
type spotSymbolMeta struct {
	Symbol     string
	BaseAsset  string
	QuoteAsset string
	Status     string
}

// GetCandles fetches OHLCV klines for the given query.
// Concurrent cold misses are coalesced via singleflight.
// Historical range queries (start/end set) are not cached to avoid unbounded keys.
func (c *Client) GetCandles(ctx context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	symbol := normalizeSymbol(q.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if !domain.IsValidInterval(string(q.Interval)) {
		return nil, fmt.Errorf("%w: unsupported interval %q", domain.ErrInvalidArgument, q.Interval)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	// Only cache "latest N candles" queries — arbitrary start/end would unbounded-key the cache.
	cacheable := q.StartTime.IsZero() && q.EndTime.IsZero()
	cacheKey := fmt.Sprintf("%s|%s|%d", symbol, q.Interval, limit)
	if cacheable && c.candles != nil {
		if hit, ok := c.candles.Get(cacheKey); ok {
			return append([]domain.Candle(nil), hit...), nil
		}
	}

	type result struct {
		candles []domain.Candle
	}

	doFetch := func() ([]domain.Candle, error) {
		params := url.Values{}
		params.Set("symbol", symbol)
		params.Set("interval", string(q.Interval))
		params.Set("limit", strconv.Itoa(limit))
		if !q.StartTime.IsZero() {
			params.Set("startTime", strconv.FormatInt(q.StartTime.UnixMilli(), 10))
		}
		if !q.EndTime.IsZero() {
			params.Set("endTime", strconv.FormatInt(q.EndTime.UnixMilli(), 10))
		}

		fetchCtx := ctx
		if cacheable {
			// Shared flight must not die with one caller's cancel.
			var cancel context.CancelFunc
			fetchCtx, cancel = context.WithTimeout(context.Background(), multiCallTimeout)
			defer cancel()
		}

		body, err := c.get(fetchCtx, "/api/v3/klines", params)
		if err != nil {
			return nil, err
		}

		var raw [][]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, fmt.Errorf("%w: decode klines: %v", domain.ErrUpstream, err)
		}

		out := make([]domain.Candle, 0, len(raw))
		for _, row := range raw {
			candle, err := parseKline(row)
			if err != nil {
				return nil, fmt.Errorf("%w: parse kline: %v", domain.ErrUpstream, err)
			}
			out = append(out, candle)
		}

		if cacheable && c.candles != nil {
			c.candles.Set(cacheKey, append([]domain.Candle(nil), out...))
		}
		return out, nil
	}

	if !cacheable {
		out, err := doFetch()
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	v, err, _ := c.candleSF.Do(cacheKey, func() (any, error) {
		if c.candles != nil {
			if hit, ok := c.candles.Get(cacheKey); ok {
				return result{candles: hit}, nil
			}
		}
		out, err := doFetch()
		if err != nil {
			return nil, err
		}
		return result{candles: out}, nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	out := v.(result).candles
	return append([]domain.Candle(nil), out...), nil
}

// GetTicker24h fetches 24-hour rolling statistics for a symbol.
// Concurrent cold misses are coalesced via singleflight.
func (c *Client) GetTicker24h(ctx context.Context, symbol string) (*domain.Ticker24h, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}

	if c.tickers != nil {
		if hit, ok := c.tickers.Get(symbol); ok {
			cp := *hit
			return &cp, nil
		}
	}

	v, err, _ := c.tickerSF.Do(symbol, func() (any, error) {
		if c.tickers != nil {
			if hit, ok := c.tickers.Get(symbol); ok {
				return hit, nil
			}
		}
		fetchCtx, cancel := context.WithTimeout(context.Background(), multiCallTimeout)
		defer cancel()

		params := url.Values{}
		params.Set("symbol", symbol)
		body, err := c.get(fetchCtx, "/api/v3/ticker/24hr", params)
		if err != nil {
			return nil, err
		}

		var raw ticker24hResponse
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, fmt.Errorf("%w: decode ticker: %v", domain.ErrUpstream, err)
		}
		if raw.Code != 0 && raw.Msg != "" {
			return nil, mapBinanceError(raw.Code, raw.Msg)
		}

		t := &domain.Ticker24h{
			Symbol:             raw.Symbol,
			PriceChange:        raw.PriceChange,
			PriceChangePercent: raw.PriceChangePercent,
			LastPrice:          raw.LastPrice,
			OpenPrice:          raw.OpenPrice,
			HighPrice:          raw.HighPrice,
			LowPrice:           raw.LowPrice,
			Volume:             raw.Volume,
			QuoteVolume:        raw.QuoteVolume,
			OpenTime:           time.UnixMilli(raw.OpenTime),
			CloseTime:          time.UnixMilli(raw.CloseTime),
			TradeCount:         raw.Count,
		}

		if c.tickers != nil {
			c.tickers.Set(symbol, t)
		}
		return t, nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	hit := v.(*domain.Ticker24h)
	cp := *hit
	return &cp, nil
}

// ListSpotMarkets returns spot-tradable symbols joined with 24h ticker metrics.
// Sources: GET /api/v3/exchangeInfo + GET /api/v3/ticker/24hr (all symbols).
// Concurrent cold misses are coalesced via singleflight.
func (c *Client) ListSpotMarkets(ctx context.Context) ([]domain.SpotMarket, error) {
	const cacheKey = "all"
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if c.spotMarkets != nil {
		if hit, ok := c.spotMarkets.Get(cacheKey); ok {
			return append([]domain.SpotMarket(nil), hit...), nil
		}
	}

	v, err, _ := c.spotSF.Do(cacheKey, func() (any, error) {
		// Re-check cache inside flight (another caller may have filled it).
		if c.spotMarkets != nil {
			if hit, ok := c.spotMarkets.Get(cacheKey); ok {
				return hit, nil
			}
		}
		fetchCtx, cancel := context.WithTimeout(context.Background(), multiCallTimeout)
		defer cancel()
		return c.fetchSpotMarkets(fetchCtx)
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	list := v.([]domain.SpotMarket)
	return append([]domain.SpotMarket(nil), list...), nil
}

func (c *Client) fetchSpotMarkets(ctx context.Context) ([]domain.SpotMarket, error) {
	const cacheKey = "all"

	// Meta is long-lived; only the all-ticker call is on the hot path for live prices.
	symbols, err := c.getExchangeSpotSymbols(ctx)
	if err != nil {
		return nil, err
	}
	// Product catalog: non-crypto filter + tags. Fail closed without a warm/stale snapshot
	// so bStocks / commodities never appear as crypto when the catalog is unavailable.
	meta, err := c.getProductMeta(ctx)
	nonCrypto := map[string]struct{}{}
	tagsByBase := map[string][]string{}
	if err != nil {
		if stale, ok := c.productMeta.GetStale("all"); ok && stale != nil {
			for _, b := range stale.NonCryptoBases {
				nonCrypto[b] = struct{}{}
			}
			tagsByBase = stale.TagsByBase
		} else {
			return nil, fmt.Errorf("%w: product catalog unavailable (non-crypto filter required)", domain.ErrUpstream)
		}
	} else if meta != nil {
		for _, b := range meta.NonCryptoBases {
			nonCrypto[b] = struct{}{}
		}
		tagsByBase = meta.TagsByBase
	}

	tickerBody, err := c.get(ctx, "/api/v3/ticker/24hr", nil)
	if err != nil {
		return nil, err
	}
	var rawTickers []ticker24hResponse
	if err := json.Unmarshal(tickerBody, &rawTickers); err != nil {
		return nil, fmt.Errorf("%w: decode tickers: %v", domain.ErrUpstream, err)
	}
	bySymbol := make(map[string]ticker24hResponse, len(rawTickers))
	for _, t := range rawTickers {
		bySymbol[t.Symbol] = t
	}

	out := make([]domain.SpotMarket, 0, len(symbols))
	for _, s := range symbols {
		base := strings.ToUpper(strings.TrimSpace(s.BaseAsset))
		if _, skip := nonCrypto[base]; skip {
			continue
		}
		m := domain.SpotMarket{
			Symbol:     s.Symbol,
			BaseAsset:  s.BaseAsset,
			QuoteAsset: s.QuoteAsset,
			Status:     s.Status,
		}
		if tags, ok := tagsByBase[base]; ok && len(tags) > 0 {
			m.Tags = append([]string(nil), tags...)
		}
		if t, ok := bySymbol[s.Symbol]; ok {
			m.LastPrice = t.LastPrice
			m.PriceChange = t.PriceChange
			m.PriceChangePercent = t.PriceChangePercent
			m.HighPrice = t.HighPrice
			m.LowPrice = t.LowPrice
			m.Volume = t.Volume
			m.QuoteVolume = t.QuoteVolume
			m.TradeCount = t.Count
		}
		out = append(out, m)
	}

	if c.spotMarkets != nil {
		c.spotMarkets.Set(cacheKey, append([]domain.SpotMarket(nil), out...))
	}
	return out, nil
}

// TagsByBase returns product-catalog tags keyed by uppercased base asset.
func (c *Client) TagsByBase(ctx context.Context) (map[string][]string, error) {
	meta, err := c.getProductMeta(ctx)
	if err != nil {
		if stale, ok := c.productMeta.GetStale("all"); ok && stale != nil && len(stale.TagsByBase) > 0 {
			return copyTagsByBase(stale.TagsByBase), nil
		}
		return nil, err
	}
	if meta == nil {
		return map[string][]string{}, nil
	}
	return copyTagsByBase(meta.TagsByBase), nil
}

func copyTagsByBase(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// ListProductTags returns sorted unique Binance product-catalog tags for crypto bases.
func (c *Client) ListProductTags(ctx context.Context) ([]string, error) {
	meta, err := c.getProductMeta(ctx)
	if err != nil {
		// Last-good tags if catalog is temporarily down.
		if stale, ok := c.productMeta.GetStale("all"); ok && stale != nil && len(stale.AllTags) > 0 {
			return append([]string(nil), stale.AllTags...), nil
		}
		return nil, err
	}
	if meta == nil {
		return []string{}, nil
	}
	return append([]string(nil), meta.AllTags...), nil
}

// getExchangeSpotSymbols returns SPOT-tradable symbols (not yet filtered by bStocks).
// Cached ~10m — exchangeInfo is large and rarely needs a full refresh for a live dashboard.
func (c *Client) getExchangeSpotSymbols(ctx context.Context) ([]spotSymbolMeta, error) {
	const key = "all"
	if c.exchangeSpot != nil {
		if hit, ok := c.exchangeSpot.Get(key); ok {
			return hit, nil
		}
	}
	v, err, _ := c.metaSF.Do("exchangeInfo", func() (any, error) {
		if c.exchangeSpot != nil {
			if hit, ok := c.exchangeSpot.Get(key); ok {
				return hit, nil
			}
		}
		fetchCtx, cancel := context.WithTimeout(context.Background(), multiCallTimeout)
		defer cancel()
		infoBody, err := c.get(fetchCtx, "/api/v3/exchangeInfo", nil)
		if err != nil {
			return nil, err
		}
		var info exchangeInfoResponse
		if err := json.Unmarshal(infoBody, &info); err != nil {
			return nil, fmt.Errorf("%w: decode exchangeInfo: %v", domain.ErrUpstream, err)
		}
		out := make([]spotSymbolMeta, 0, len(info.Symbols))
		for _, s := range info.Symbols {
			if !s.IsSpotTradingAllowed {
				continue
			}
			if len(s.Permissions) > 0 && !hasPermission(s.Permissions, "SPOT") {
				continue
			}
			out = append(out, spotSymbolMeta{
				Symbol:     s.Symbol,
				BaseAsset:  s.BaseAsset,
				QuoteAsset: s.QuoteAsset,
				Status:     s.Status,
			})
		}
		if c.exchangeSpot != nil {
			c.exchangeSpot.Set(key, append([]spotSymbolMeta(nil), out...))
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return v.([]spotSymbolMeta), nil
}

// getProductMeta returns cached product-catalog meta (non-crypto bases + tags).
func (c *Client) getProductMeta(ctx context.Context) (*productMetaSnapshot, error) {
	const key = "all"
	if c.productMeta != nil {
		if hit, ok := c.productMeta.Get(key); ok {
			return hit, nil
		}
	}
	v, err, _ := c.metaSF.Do("productMeta", func() (any, error) {
		if c.productMeta != nil {
			if hit, ok := c.productMeta.Get(key); ok {
				return hit, nil
			}
		}
		// Detach from caller ctx so a cancelled request cannot poison concurrent waiters.
		fetchCtx, cancel := context.WithTimeout(context.Background(), multiCallTimeout)
		defer cancel()
		params := url.Values{}
		params.Set("includeEtf", "true")
		body, err := c.getProduct(fetchCtx, productCatalogPath, params)
		if err != nil {
			return nil, err
		}
		var envelope productCatalogResponse
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, fmt.Errorf("%w: decode product catalog: %v", domain.ErrUpstream, err)
		}
		if !productEnvelopeOK(envelope.Success, envelope.Code) {
			return nil, fmt.Errorf("%w: binance product catalog code=%s msg=%s success=%v",
				domain.ErrUpstream, envelope.Code, envelope.Message, envelope.Success)
		}
		if len(envelope.Data) == 0 {
			// Fail closed: never cache an empty catalog (would include equities/commodities).
			return nil, fmt.Errorf("%w: empty product catalog", domain.ErrUpstream)
		}
		snap := buildProductMetaSnapshot(envelope.Data)
		if c.productMeta != nil {
			c.productMeta.Set(key, snap)
		}
		return snap, nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return v.(*productMetaSnapshot), nil
}

// buildProductMetaSnapshot extracts non-crypto bases and crypto tags from catalog rows.
func buildProductMetaSnapshot(rows []productCatalogRow) *productMetaSnapshot {
	nonCryptoSeen := map[string]struct{}{}
	var nonCrypto []string
	// tag sets per base (crypto only)
	tagSets := map[string]map[string]struct{}{}
	globalTags := map[string]struct{}{}

	for _, row := range rows {
		base := strings.ToUpper(strings.TrimSpace(row.BaseAsset))
		if base == "" {
			continue
		}
		if hasNonCryptoTag(row.Tags) {
			if _, ok := nonCryptoSeen[base]; !ok {
				nonCryptoSeen[base] = struct{}{}
				nonCrypto = append(nonCrypto, base)
			}
			continue
		}
		if tagSets[base] == nil {
			tagSets[base] = map[string]struct{}{}
		}
		for _, raw := range row.Tags {
			t := strings.TrimSpace(raw)
			if t == "" {
				continue
			}
			// Preserve original casing from first sighting; match case-insensitively via lower key.
			// Store display form as-is from Binance.
			tagSets[base][t] = struct{}{}
			globalTags[t] = struct{}{}
		}
	}

	tagsByBase := make(map[string][]string, len(tagSets))
	for base, set := range tagSets {
		tags := make([]string, 0, len(set))
		for t := range set {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		tagsByBase[base] = tags
	}
	allTags := make([]string, 0, len(globalTags))
	for t := range globalTags {
		allTags = append(allTags, t)
	}
	sort.Strings(allTags)

	return &productMetaSnapshot{
		NonCryptoBases: nonCrypto,
		TagsByBase:     tagsByBase,
		AllTags:        allTags,
	}
}

// productCatalogResponse is the www.binance.com product list envelope.
type productCatalogResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Success bool                `json:"success"`
	Data    []productCatalogRow `json:"data"`
}

type productCatalogRow struct {
	Symbol    string   `json:"s"`
	BaseAsset string   `json:"b"`
	Tags      []string `json:"tags"`
}

func hasPermission(perms []string, want string) bool {
	for _, p := range perms {
		if p == want {
			return true
		}
	}
	return false
}

type exchangeInfoResponse struct {
	Symbols []exchangeSymbol `json:"symbols"`
}

type exchangeSymbol struct {
	Symbol               string   `json:"symbol"`
	Status               string   `json:"status"`
	BaseAsset            string   `json:"baseAsset"`
	QuoteAsset           string   `json:"quoteAsset"`
	IsSpotTradingAllowed bool     `json:"isSpotTradingAllowed"`
	Permissions          []string `json:"permissions"`
}

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", domain.ErrUpstream, err)
	}

	// 429 = weight rate limit; 418 = IP ban / forced ban style response from Binance.
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusTeapot {
		return nil, fmt.Errorf("%w: binance status %d", domain.ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: symbol or resource", domain.ErrNotFound)
	}
	if resp.StatusCode >= 400 {
		var er binanceError
		_ = json.Unmarshal(body, &er)
		if er.Msg != "" {
			return nil, mapBinanceError(er.Code, er.Msg)
		}
		return nil, fmt.Errorf("%w: status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return body, nil
}

type binanceError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type ticker24hResponse struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	LastPrice          string `json:"lastPrice"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	Count              int64  `json:"count"`
	// Error fields when Binance returns an error body with 200 (rare) or mixed.
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func parseKline(row []json.RawMessage) (domain.Candle, error) {
	if len(row) < 9 {
		return domain.Candle{}, fmt.Errorf("expected >=9 fields, got %d", len(row))
	}
	openTime, err := unmarshalInt64(row[0])
	if err != nil {
		return domain.Candle{}, err
	}
	open, err := unmarshalString(row[1])
	if err != nil {
		return domain.Candle{}, err
	}
	high, err := unmarshalString(row[2])
	if err != nil {
		return domain.Candle{}, err
	}
	low, err := unmarshalString(row[3])
	if err != nil {
		return domain.Candle{}, err
	}
	closePx, err := unmarshalString(row[4])
	if err != nil {
		return domain.Candle{}, err
	}
	vol, err := unmarshalString(row[5])
	if err != nil {
		return domain.Candle{}, err
	}
	closeTime, err := unmarshalInt64(row[6])
	if err != nil {
		return domain.Candle{}, err
	}
	quoteVol, err := unmarshalString(row[7])
	if err != nil {
		return domain.Candle{}, err
	}
	trades, err := unmarshalInt64(row[8])
	if err != nil {
		return domain.Candle{}, err
	}
	return domain.Candle{
		OpenTime:    time.UnixMilli(openTime),
		Open:        open,
		High:        high,
		Low:         low,
		Close:       closePx,
		Volume:      vol,
		CloseTime:   time.UnixMilli(closeTime),
		QuoteVolume: quoteVol,
		TradeCount:  trades,
	}, nil
}

func unmarshalString(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.String(), nil
	}
	return "", fmt.Errorf("not a string: %s", string(raw))
}

func unmarshalInt64(raw json.RawMessage) (int64, error) {
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strconv.ParseInt(s, 10, 64)
	}
	return 0, fmt.Errorf("not an int: %s", string(raw))
}

func normalizeSymbol(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func mapBinanceError(code int, msg string) error {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "invalid symbol"), code == -1121:
		return fmt.Errorf("%w: %s", domain.ErrNotFound, msg)
	case strings.Contains(lower, "invalid interval"), code == -1120:
		return fmt.Errorf("%w: %s", domain.ErrInvalidArgument, msg)
	case code == -1003:
		return fmt.Errorf("%w: %s", domain.ErrRateLimited, msg)
	default:
		return fmt.Errorf("%w: binance %d: %s", domain.ErrUpstream, code, msg)
	}
}
