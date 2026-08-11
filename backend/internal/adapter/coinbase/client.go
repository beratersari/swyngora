package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL     = "https://api.coinbase.com"
	defaultExchangeURL = "https://api.exchange.coinbase.com"
	multiCallTimeout   = 45 * time.Second
	productsPath       = "/api/v3/brokerage/market/products"
)

// Client implements domain.MarketDataPort against Coinbase public market APIs.
// Spot list uses Advanced Trade public products; candles use Exchange public candles.
type Client struct {
	baseURL     string // api.coinbase.com — products/ticker bulk
	exchangeURL string // api.exchange.coinbase.com — candles
	httpClient  *http.Client
	candles     *cache.TTL[[]domain.Candle]
	tickers     *cache.TTL[*domain.Ticker24h]
	orderBooks  *cache.TTL[*domain.RawOrderBook]
	spotMarkets *cache.TTL[[]domain.SpotMarket]
	spotSF      singleflight.Group
	candleSF    singleflight.Group
	tickerSF    singleflight.Group
	orderBookSF singleflight.Group
	depth       *DepthHub
	depthOnce   sync.Once
	depthWait   time.Duration
	wsURL       string
	wsDial      wsDialer
	depthIdle   time.Duration
}

// Options configures the Coinbase client.
type Options struct {
	BaseURL         string
	ExchangeURL     string
	HTTPClient      *http.Client
	CandleCache     *cache.TTL[[]domain.Candle]
	TickerCache     *cache.TTL[*domain.Ticker24h]
	OrderBookCache  *cache.TTL[*domain.RawOrderBook]
	SpotMarketCache *cache.TTL[[]domain.SpotMarket]
	WSURL           string
	WSDial          wsDialer
	DepthIdle       time.Duration
	DepthWait       time.Duration
}

// NewClient constructs a Coinbase market-data client.
func NewClient(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	ex := strings.TrimRight(opts.ExchangeURL, "/")
	if ex == "" {
		ex = defaultExchangeURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	c := &Client{
		baseURL:     base,
		exchangeURL: ex,
		httpClient:  hc,
		candles:     opts.CandleCache,
		tickers:     opts.TickerCache,
		orderBooks:  opts.OrderBookCache,
		spotMarkets: opts.SpotMarketCache,
		depthWait:   opts.DepthWait,
		wsURL:       opts.WSURL,
		wsDial:      opts.WSDial,
		depthIdle:   opts.DepthIdle,
	}
	if c.depthWait <= 0 {
		c.depthWait = 8 * time.Second
	}
	return c
}

func (c *Client) ensureDepth() {
	c.depthOnce.Do(func() {
		c.depth = newDepthHub(c.wsURL, c.wsDial, c.depthIdle, c.checksumTop)
	})
}

// Close stops live order-book streams.
func (c *Client) Close() {
	if c == nil || c.depth == nil {
		return
	}
	c.depth.Close()
}

// ListProductTags is not available from Coinbase public product catalog.
// Service layer may fall back to the Binance catalog for the filter UI.
func (c *Client) ListProductTags(context.Context) ([]string, error) {
	return []string{}, nil
}

// TagsByBase — Coinbase has no product-tag catalog (service enriches from Binance).
func (c *Client) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

// ListSpotMarkets returns online SPOT products with 24h metrics.
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
	var out []domain.SpotMarket
	cursor := ""
	const maxPages = 20
	for page := 0; page < maxPages; page++ {
		params := url.Values{}
		params.Set("product_type", "SPOT")
		params.Set("limit", "250")
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		body, err := c.get(ctx, c.baseURL, productsPath, params)
		if err != nil {
			return nil, err
		}
		var resp productsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("%w: decode coinbase products: %v", domain.ErrUpstream, err)
		}
		for _, p := range resp.Products {
			if strings.ToLower(p.Status) != "online" || p.TradingDisabled || p.IsDisabled {
				continue
			}
			if strings.ToUpper(p.ProductType) != "" && strings.ToUpper(p.ProductType) != "SPOT" {
				continue
			}
			m := domain.SpotMarket{
				Symbol:             p.ProductID,
				BaseAsset:          p.BaseCurrencyID,
				QuoteAsset:         p.QuoteCurrencyID,
				Status:             "TRADING",
				LastPrice:          p.Price,
				PriceChangePercent: p.PricePercentageChange24h,
				Volume:             p.Volume24h,
				QuoteVolume:        p.ApproximateQuote24hVolume,
			}
			// Coinbase % is already percent (e.g. 0.09); keep as string for consistency with other venues.
			out = append(out, m)
		}
		if !resp.Pagination.HasNext || resp.Pagination.NextCursor == "" {
			break
		}
		if page == maxPages-1 {
			return nil, fmt.Errorf("%w: coinbase products pagination exceeded %d pages", domain.ErrUpstream, maxPages)
		}
		cursor = resp.Pagination.NextCursor
	}
	if c.spotMarkets != nil {
		c.spotMarkets.Set("all", append([]domain.SpotMarket(nil), out...))
	}
	return out, nil
}

// GetTicker24h returns 24h stats for a Coinbase product id (e.g. BTC-USD).
//
// Advanced Trade public products expose last price / volume / % change, but
// high_24h and low_24h are typically empty on that endpoint. We fill high/low/open
// from the classic Exchange public stats API (api.exchange.coinbase.com), which
// does publish those fields.
func (c *Client) GetTicker24h(ctx context.Context, symbol string) (*domain.Ticker24h, error) {
	symbol = normalizeProductID(symbol)
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

		// Product row: last, %, base/quote volume (Advanced Trade public).
		params := url.Values{}
		params.Set("product_ids", symbol)
		body, err := c.get(fetchCtx, c.baseURL, productsPath, params)
		if err != nil {
			return nil, err
		}
		var resp productsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("%w: decode coinbase product: %v", domain.ErrUpstream, err)
		}
		var p *productRow
		for i := range resp.Products {
			if strings.EqualFold(resp.Products[i].ProductID, symbol) {
				p = &resp.Products[i]
				break
			}
		}
		if p == nil {
			return nil, fmt.Errorf("%w: product %s", domain.ErrNotFound, symbol)
		}

		now := time.Now().UTC()
		t := &domain.Ticker24h{
			Symbol:             p.ProductID,
			LastPrice:          p.Price,
			PriceChangePercent: p.PricePercentageChange24h,
			Volume:             p.Volume24h,
			QuoteVolume:        p.ApproximateQuote24hVolume,
			// high/low often empty on Advanced Trade public products — fill from stats.
			HighPrice: strings.TrimSpace(p.High24h),
			LowPrice:  strings.TrimSpace(p.Low24h),
			OpenTime:  now.Add(-24 * time.Hour),
			// CloseTime stays zero unless Exchange /ticker provides an event time.
		}

		// Exchange stats: open, high, low, last, volume (public, no auth).
		statsPath := "/products/" + url.PathEscape(symbol) + "/stats"
		statsBody, statsErr := c.get(fetchCtx, c.exchangeURL, statsPath, nil)
		if statsErr == nil {
			var st productStats
			if err := json.Unmarshal(statsBody, &st); err == nil {
				if st.High != "" {
					t.HighPrice = st.High
				}
				if st.Low != "" {
					t.LowPrice = st.Low
				}
				if st.Open != "" {
					t.OpenPrice = st.Open
				}
				if st.Last != "" {
					t.LastPrice = st.Last
				}
				if st.Volume != "" {
					t.Volume = st.Volume
				}
			}
		}
		// If % change empty but open+last known, derive approximate 24h change.
		if t.PriceChangePercent == "" && t.OpenPrice != "" && t.LastPrice != "" {
			if open, e1 := strconv.ParseFloat(t.OpenPrice, 64); e1 == nil && open != 0 {
				if last, e2 := strconv.ParseFloat(t.LastPrice, 64); e2 == nil {
					t.PriceChangePercent = strconv.FormatFloat((last-open)/open*100, 'f', -1, 64)
					t.PriceChange = strconv.FormatFloat(last-open, 'f', -1, 64)
				}
			}
		}

		// Exchange ticker carries last-trade time (stats do not). Missing/unparseable → not fresh.
		tickerPath := "/products/" + url.PathEscape(symbol) + "/ticker"
		if tickBody, tickErr := c.get(fetchCtx, c.exchangeURL, tickerPath, nil); tickErr == nil {
			var tick productTicker
			if err := json.Unmarshal(tickBody, &tick); err == nil {
				if ts := strings.TrimSpace(tick.Time); ts != "" {
					if parsed, perr := time.Parse(time.RFC3339Nano, ts); perr == nil {
						t.CloseTime = parsed.UTC()
					} else if parsed, perr := time.Parse(time.RFC3339, ts); perr == nil {
						t.CloseTime = parsed.UTC()
					}
				}
			}
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

// GetCandles fetches OHLCV from Coinbase Exchange public candles API.
func (c *Client) GetCandles(ctx context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	symbol := normalizeProductID(q.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	gran, ok := intervalToGranularity(q.Interval)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported interval %q for coinbase", domain.ErrInvalidArgument, q.Interval)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 300 {
		limit = 300 // Coinbase max candles per request
	}
	cacheable := q.StartTime.IsZero() && q.EndTime.IsZero()
	cacheKey := fmt.Sprintf("%s|%s|%d", symbol, q.Interval, limit)
	if cacheable && c.candles != nil {
		if hit, ok := c.candles.Get(cacheKey); ok {
			return append([]domain.Candle(nil), hit...), nil
		}
	}

	doFetch := func() ([]domain.Candle, error) {
		params := url.Values{}
		params.Set("granularity", strconv.Itoa(gran))
		// Coinbase returns newest first up to ~300 bars; optional start/end as unix.
		if !q.StartTime.IsZero() {
			params.Set("start", q.StartTime.UTC().Format(time.RFC3339))
		}
		if !q.EndTime.IsZero() {
			params.Set("end", q.EndTime.UTC().Format(time.RFC3339))
		}
		path := "/products/" + url.PathEscape(symbol) + "/candles"
		fetchCtx := ctx
		if cacheable {
			var cancel context.CancelFunc
			fetchCtx, cancel = context.WithTimeout(context.Background(), multiCallTimeout)
			defer cancel()
		}
		body, err := c.get(fetchCtx, c.exchangeURL, path, params)
		if err != nil {
			return nil, err
		}
		var raw [][]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, fmt.Errorf("%w: decode coinbase candles: %v", domain.ErrUpstream, err)
		}
		// Coinbase: [time, low, high, open, close, volume] newest first.
		out := make([]domain.Candle, 0, len(raw))
		for i, row := range raw {
			if len(row) < 6 {
				return nil, fmt.Errorf("%w: coinbase candle row %d too short", domain.ErrUpstream, i)
			}
			ts, err := unmarshalFloat(row[0])
			if err != nil {
				return nil, fmt.Errorf("%w: candle time: %v", domain.ErrUpstream, err)
			}
			low, err := unmarshalStringNum(row[1])
			if err != nil {
				return nil, fmt.Errorf("%w: candle low: %v", domain.ErrUpstream, err)
			}
			high, err := unmarshalStringNum(row[2])
			if err != nil {
				return nil, fmt.Errorf("%w: candle high: %v", domain.ErrUpstream, err)
			}
			open, err := unmarshalStringNum(row[3])
			if err != nil {
				return nil, fmt.Errorf("%w: candle open: %v", domain.ErrUpstream, err)
			}
			closePx, err := unmarshalStringNum(row[4])
			if err != nil {
				return nil, fmt.Errorf("%w: candle close: %v", domain.ErrUpstream, err)
			}
			vol, err := unmarshalStringNum(row[5])
			if err != nil {
				return nil, fmt.Errorf("%w: candle volume: %v", domain.ErrUpstream, err)
			}
			openTime := time.Unix(int64(ts), 0).UTC()
			closeTime := openTime.Add(time.Duration(gran) * time.Second).Add(-time.Millisecond)
			out = append(out, domain.Candle{
				OpenTime:  openTime,
				Open:      open,
				High:      high,
				Low:       low,
				Close:     closePx,
				Volume:    vol,
				CloseTime: closeTime,
			})
		}
		// Return chronological order (oldest first) to match Binance.
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
		if limit < len(out) {
			out = out[len(out)-limit:]
		}
		if cacheable && c.candles != nil {
			c.candles.Set(cacheKey, append([]domain.Candle(nil), out...))
		}
		return out, nil
	}

	if !cacheable {
		return doFetch()
	}
	v, err, _ := c.candleSF.Do(cacheKey, func() (any, error) {
		if c.candles != nil {
			if hit, ok := c.candles.Get(cacheKey); ok {
				return hit, nil
			}
		}
		return doFetch()
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	out := v.([]domain.Candle)
	return append([]domain.Candle(nil), out...), nil
}

func intervalToGranularity(iv domain.CandleInterval) (int, bool) {
	switch iv {
	case domain.Interval1m:
		return 60, true
	case domain.Interval5m:
		return 300, true
	case domain.Interval15m:
		return 900, true
	case domain.Interval1h:
		return 3600, true
	case domain.Interval6h:
		return 21600, true
	case domain.Interval1d:
		return 86400, true
	default:
		return 0, false
	}
}

func normalizeProductID(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	return s
}

func (c *Client) get(ctx context.Context, host, path string, params url.Values) ([]byte, error) {
	u := host + path
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
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: coinbase status %d", domain.ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: resource", domain.ErrNotFound)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: coinbase status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return body, nil
}

type productsResponse struct {
	Products   []productRow `json:"products"`
	Pagination struct {
		NextCursor string `json:"next_cursor"`
		HasNext    bool   `json:"has_next"`
	} `json:"pagination"`
}

type productRow struct {
	ProductID                 string `json:"product_id"`
	Price                     string `json:"price"`
	PricePercentageChange24h  string `json:"price_percentage_change_24h"`
	Volume24h                 string `json:"volume_24h"`
	ApproximateQuote24hVolume string `json:"approximate_quote_24h_volume"`
	BaseCurrencyID            string `json:"base_currency_id"`
	QuoteCurrencyID           string `json:"quote_currency_id"`
	Status                    string `json:"status"`
	TradingDisabled           bool   `json:"trading_disabled"`
	IsDisabled                bool   `json:"is_disabled"`
	ProductType               string `json:"product_type"`
	// High24h / Low24h are documented on Advanced Trade products but often empty
	// for unauthenticated public market endpoints — use productStats instead.
	High24h string `json:"high_24h"`
	Low24h  string `json:"low_24h"`
}

// productStats is Coinbase Exchange public GET /products/{id}/stats.
type productStats struct {
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Last   string `json:"last"`
	Volume string `json:"volume"`
}

// productTicker is Coinbase Exchange public GET /products/{id}/ticker.
type productTicker struct {
	Time string `json:"time"`
}

func unmarshalFloat(raw json.RawMessage) (float64, error) {
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strconv.ParseFloat(s, 64)
	}
	return 0, fmt.Errorf("not a number: %s", string(raw))
}

func unmarshalStringNum(raw json.RawMessage) (string, error) {
	// Reject JSON null / empty explicitly (do not treat as zero candle).
	trim := strings.TrimSpace(string(raw))
	if trim == "" || trim == "null" {
		return "", fmt.Errorf("not a number: %s", string(raw))
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return "", fmt.Errorf("not a finite number: %s", string(raw))
		}
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return "", fmt.Errorf("empty number string")
		}
		// Must be a real number string — reject "not-a-number".
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return "", fmt.Errorf("not a number: %q", s)
		}
		return s, nil
	}
	return "", fmt.Errorf("not a number: %s", string(raw))
}
