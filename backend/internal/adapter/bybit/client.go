package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL   = "https://api.bybit.com"
	multiCallTimeout = 45 * time.Second
)

// Client implements domain.MarketDataPort against Bybit v5 public market APIs (spot).
type Client struct {
	baseURL     string
	httpClient  *http.Client
	candles     *cache.TTL[[]domain.Candle]
	tickers     *cache.TTL[*domain.Ticker24h]
	spotMarkets *cache.TTL[[]domain.SpotMarket]
	spotSF      singleflight.Group
	candleSF    singleflight.Group
	tickerSF    singleflight.Group
}

// Options configures the Bybit client.
type Options struct {
	BaseURL         string
	HTTPClient      *http.Client
	CandleCache     *cache.TTL[[]domain.Candle]
	TickerCache     *cache.TTL[*domain.Ticker24h]
	SpotMarketCache *cache.TTL[[]domain.SpotMarket]
}

// NewClient constructs a Bybit spot market-data client.
func NewClient(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:     base,
		httpClient:  hc,
		candles:     opts.CandleCache,
		tickers:     opts.TickerCache,
		spotMarkets: opts.SpotMarketCache,
	}
}

// ListProductTags — Bybit does not expose Binance-style product tags.
func (c *Client) ListProductTags(context.Context) ([]string, error) {
	return []string{}, nil
}

// ListSpotMarkets joins instruments-info + tickers for spot category.
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
	// Instruments (may paginate via nextPageCursor).
	type inst struct {
		Symbol    string
		BaseCoin  string
		QuoteCoin string
		Status    string
	}
	var instruments []inst
	cursor := ""
	for page := 0; page < 20; page++ {
		params := url.Values{}
		params.Set("category", "spot")
		params.Set("limit", "1000")
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		body, err := c.get(ctx, "/v5/market/instruments-info", params)
		if err != nil {
			return nil, err
		}
		var resp instrumentsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("%w: decode bybit instruments: %v", domain.ErrUpstream, err)
		}
		if resp.RetCode != 0 {
			return nil, mapBybitError(resp.RetCode, resp.RetMsg)
		}
		for _, row := range resp.Result.List {
			instruments = append(instruments, inst{
				Symbol:    row.Symbol,
				BaseCoin:  row.BaseCoin,
				QuoteCoin: row.QuoteCoin,
				Status:    row.Status,
			})
		}
		if resp.Result.NextPageCursor == "" {
			break
		}
		cursor = resp.Result.NextPageCursor
	}

	// All tickers (single call for spot).
	tBody, err := c.get(ctx, "/v5/market/tickers", url.Values{"category": []string{"spot"}})
	if err != nil {
		return nil, err
	}
	var tResp tickersResponse
	if err := json.Unmarshal(tBody, &tResp); err != nil {
		return nil, fmt.Errorf("%w: decode bybit tickers: %v", domain.ErrUpstream, err)
	}
	if tResp.RetCode != 0 {
		return nil, mapBybitError(tResp.RetCode, tResp.RetMsg)
	}
	bySym := make(map[string]tickerRow, len(tResp.Result.List))
	for _, t := range tResp.Result.List {
		bySym[t.Symbol] = t
	}

	out := make([]domain.SpotMarket, 0, len(instruments))
	for _, in := range instruments {
		// Bybit uses "Trading" status for active spot.
		status := strings.ToUpper(in.Status)
		if status == "TRADING" {
			status = "TRADING"
		}
		m := domain.SpotMarket{
			Symbol:     in.Symbol,
			BaseAsset:  in.BaseCoin,
			QuoteAsset: in.QuoteCoin,
			Status:     status,
		}
		if t, ok := bySym[in.Symbol]; ok {
			m.LastPrice = t.LastPrice
			m.HighPrice = t.HighPrice24h
			m.LowPrice = t.LowPrice24h
			m.Volume = t.Volume24h
			m.QuoteVolume = t.Turnover24h
			// price24hPcnt is a fraction (e.g. -0.0238); convert to percent string like Binance.
			if t.Price24hPcnt != "" {
				if f, err := strconv.ParseFloat(t.Price24hPcnt, 64); err == nil {
					m.PriceChangePercent = strconv.FormatFloat(f*100, 'f', -1, 64)
				} else {
					m.PriceChangePercent = t.Price24hPcnt
				}
			}
		}
		out = append(out, m)
	}
	if c.spotMarkets != nil {
		c.spotMarkets.Set("all", append([]domain.SpotMarket(nil), out...))
	}
	return out, nil
}

// GetTicker24h fetches a single spot ticker.
func (c *Client) GetTicker24h(ctx context.Context, symbol string) (*domain.Ticker24h, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
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
		params.Set("category", "spot")
		params.Set("symbol", symbol)
		body, err := c.get(fetchCtx, "/v5/market/tickers", params)
		if err != nil {
			return nil, err
		}
		var resp tickersResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("%w: decode bybit ticker: %v", domain.ErrUpstream, err)
		}
		if resp.RetCode != 0 {
			return nil, mapBybitError(resp.RetCode, resp.RetMsg)
		}
		if len(resp.Result.List) == 0 {
			return nil, fmt.Errorf("%w: symbol %s", domain.ErrNotFound, symbol)
		}
		t := resp.Result.List[0]
		now := time.Now().UTC()
		pct := t.Price24hPcnt
		if f, err := strconv.ParseFloat(pct, 64); err == nil {
			pct = strconv.FormatFloat(f*100, 'f', -1, 64)
		}
		out := &domain.Ticker24h{
			Symbol:             t.Symbol,
			LastPrice:          t.LastPrice,
			HighPrice:          t.HighPrice24h,
			LowPrice:           t.LowPrice24h,
			Volume:             t.Volume24h,
			QuoteVolume:        t.Turnover24h,
			PriceChangePercent: pct,
			OpenTime:           now.Add(-24 * time.Hour),
			CloseTime:          now,
		}
		if c.tickers != nil {
			c.tickers.Set(symbol, out)
		}
		return out, nil
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

// GetCandles fetches spot klines.
func (c *Client) GetCandles(ctx context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	symbol := strings.ToUpper(strings.TrimSpace(q.Symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	iv, ok := intervalToBybit(q.Interval)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported interval %q for bybit", domain.ErrInvalidArgument, q.Interval)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
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
		params.Set("category", "spot")
		params.Set("symbol", symbol)
		params.Set("interval", iv)
		params.Set("limit", strconv.Itoa(limit))
		if !q.StartTime.IsZero() {
			params.Set("start", strconv.FormatInt(q.StartTime.UnixMilli(), 10))
		}
		if !q.EndTime.IsZero() {
			params.Set("end", strconv.FormatInt(q.EndTime.UnixMilli(), 10))
		}
		fetchCtx := ctx
		if cacheable {
			var cancel context.CancelFunc
			fetchCtx, cancel = context.WithTimeout(context.Background(), multiCallTimeout)
			defer cancel()
		}
		body, err := c.get(fetchCtx, "/v5/market/kline", params)
		if err != nil {
			return nil, err
		}
		var resp klineResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("%w: decode bybit kline: %v", domain.ErrUpstream, err)
		}
		if resp.RetCode != 0 {
			return nil, mapBybitError(resp.RetCode, resp.RetMsg)
		}
		// Bybit list: [start, open, high, low, close, volume, turnover] newest first.
		out := make([]domain.Candle, 0, len(resp.Result.List))
		for _, row := range resp.Result.List {
			if len(row) < 7 {
				continue
			}
			ms, err := strconv.ParseInt(row[0], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: kline time: %v", domain.ErrUpstream, err)
			}
			openTime := time.UnixMilli(ms).UTC()
			// close time approx by interval length — use open for display consistency when unknown
			out = append(out, domain.Candle{
				OpenTime:    openTime,
				Open:        row[1],
				High:        row[2],
				Low:         row[3],
				Close:       row[4],
				Volume:      row[5],
				QuoteVolume: row[6],
				CloseTime:   openTime,
			})
		}
		// Chronological oldest-first
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
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

func intervalToBybit(iv domain.CandleInterval) (string, bool) {
	switch iv {
	case domain.Interval1m:
		return "1", true
	case domain.Interval3m:
		return "3", true
	case domain.Interval5m:
		return "5", true
	case domain.Interval15m:
		return "15", true
	case domain.Interval30m:
		return "30", true
	case domain.Interval1h:
		return "60", true
	case domain.Interval2h:
		return "120", true
	case domain.Interval4h:
		return "240", true
	case domain.Interval6h:
		return "360", true
	case domain.Interval12h:
		return "720", true
	case domain.Interval1d:
		return "D", true
	case domain.Interval1w:
		return "W", true
	case domain.Interval1M:
		return "M", true
	default:
		return "", false
	}
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
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: bybit status %d", domain.ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: bybit status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return body, nil
}

func mapBybitError(code int, msg string) error {
	lower := strings.ToLower(msg)
	switch {
	case code == 10006 || strings.Contains(lower, "too many"):
		return fmt.Errorf("%w: %s", domain.ErrRateLimited, msg)
	case strings.Contains(lower, "not exist") || strings.Contains(lower, "invalid symbol") || code == 10001:
		return fmt.Errorf("%w: %s", domain.ErrNotFound, msg)
	default:
		return fmt.Errorf("%w: bybit %d: %s", domain.ErrUpstream, code, msg)
	}
}

type instrumentsResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List           []instrumentRow `json:"list"`
		NextPageCursor string          `json:"nextPageCursor"`
	} `json:"result"`
}

type instrumentRow struct {
	Symbol    string `json:"symbol"`
	BaseCoin  string `json:"baseCoin"`
	QuoteCoin string `json:"quoteCoin"`
	Status    string `json:"status"`
}

type tickersResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []tickerRow `json:"list"`
	} `json:"result"`
}

type tickerRow struct {
	Symbol        string `json:"symbol"`
	LastPrice     string `json:"lastPrice"`
	HighPrice24h  string `json:"highPrice24h"`
	LowPrice24h   string `json:"lowPrice24h"`
	Volume24h     string `json:"volume24h"`
	Turnover24h   string `json:"turnover24h"`
	Price24hPcnt  string `json:"price24hPcnt"`
}

type klineResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List [][]string `json:"list"`
	} `json:"result"`
}
