package equities

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultYahooBase = "https://query1.finance.yahoo.com"
	spotCacheTTL     = 2 * time.Minute
	tickerCacheTTL   = 20 * time.Second
	candleCacheTTL   = 60 * time.Second
	quoteBatchSize   = 12
	httpTimeout      = 18 * time.Second
)

// Client implements domain.MarketDataPort for a cash equity venue (Nasdaq or BIST)
// using Yahoo Finance's public quote/chart endpoints (no API key).
type Client struct {
	exchange        domain.Exchange
	quoteAsset      string
	yahooSfx        string // e.g. ".IS" for BIST
	universe        []string
	baseURL         string
	nasdaqURL       string
	bistListURL     string
	bistScreenerURL string
	httpClient      *http.Client
	spot            *cache.TTL[[]domain.SpotMarket]
	tickers         *cache.TTL[*domain.Ticker24h]
	candles         *cache.TTL[[]domain.Candle]
}

// Options configures an equity venue client.
type Options struct {
	Exchange       domain.Exchange
	QuoteAsset     string
	YahooSuffix    string
	Universe       []string
	BaseURL        string
	NasdaqScreener string
	BistListURL    string
	BistScreener   string
	HTTPClient     *http.Client
	SpotCache      *cache.TTL[[]domain.SpotMarket]
	TickerCache    *cache.TTL[*domain.Ticker24h]
	CandleCache    *cache.TTL[[]domain.Candle]
}

// NewNasdaq returns a Nasdaq cash-equity adapter (USD).
func NewNasdaq(opts Options) *Client {
	opts.Exchange = domain.ExchangeNasdaq
	opts.QuoteAsset = "USD"
	opts.YahooSuffix = ""
	if len(opts.Universe) == 0 {
		opts.Universe = uniqueUpper(nasdaqUniverse)
	}
	return newClient(opts)
}

// NewBist returns a Borsa Istanbul cash-equity adapter (TRY).
func NewBist(opts Options) *Client {
	opts.Exchange = domain.ExchangeBist
	opts.QuoteAsset = "TRY"
	opts.YahooSuffix = ".IS"
	if len(opts.Universe) == 0 {
		opts.Universe = uniqueUpper(bistUniverse)
	}
	return newClient(opts)
}

func newClient(opts Options) *Client {
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: httpTimeout}
	}
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultYahooBase
	}
	spot := opts.SpotCache
	if spot == nil {
		spot = cache.New[[]domain.SpotMarket](spotCacheTTL)
	}
	tick := opts.TickerCache
	if tick == nil {
		tick = cache.New[*domain.Ticker24h](tickerCacheTTL)
	}
	cand := opts.CandleCache
	if cand == nil {
		cand = cache.New[[]domain.Candle](candleCacheTTL)
	}
	bistScreener := strings.TrimSpace(opts.BistScreener)
	if bistScreener == "" {
		bistScreener = defaultBistScreener
	}
	return &Client{
		exchange:        opts.Exchange,
		quoteAsset:      strings.ToUpper(strings.TrimSpace(opts.QuoteAsset)),
		yahooSfx:        opts.YahooSuffix,
		universe:        uniqueUpper(opts.Universe),
		baseURL:         base,
		nasdaqURL:       strings.TrimSpace(opts.NasdaqScreener),
		bistListURL:     strings.TrimSpace(opts.BistListURL),
		bistScreenerURL: bistScreener,
		httpClient:      hc,
		spot:            spot,
		tickers:         tick,
		candles:         cand,
	}
}

func (c *Client) yahooSymbol(local string) string {
	local = domain.NormalizeSymbol(c.exchange, local)
	if local == "" {
		return ""
	}
	if c.yahooSfx != "" && !strings.HasSuffix(local, c.yahooSfx) {
		return local + c.yahooSfx
	}
	return local
}

func (c *Client) localSymbol(yahoo string) string {
	s := strings.ToUpper(strings.TrimSpace(yahoo))
	if c.yahooSfx != "" {
		s = strings.TrimSuffix(s, strings.ToUpper(c.yahooSfx))
	}
	return s
}

// ListSpotMarkets returns the full venue tape with last/change/volume (and mcap when the feed has it).
func (c *Client) ListSpotMarkets(ctx context.Context) ([]domain.SpotMarket, error) {
	if hit, ok := c.spot.Get("all"); ok {
		return append([]domain.SpotMarket(nil), hit...), nil
	}
	var (
		out []domain.SpotMarket
		err error
	)
	switch c.exchange {
	case domain.ExchangeNasdaq:
		out, err = c.fetchNasdaqScreener(ctx)
	default:
		out, err = c.fetchBistScreener(ctx)
		if err != nil || len(out) == 0 {
			syms := c.fetchBistUniverse(ctx)
			quotes, qerr := c.fetchQuotes(ctx, syms)
			err = qerr
			if err == nil {
				out = make([]domain.SpotMarket, 0, len(quotes))
				for _, q := range quotes {
					if row := c.spotFromQuote(q); row != nil {
						out = append(out, *row)
					}
				}
			}
		}
	}
	if err != nil {
		if stale, ok := c.spot.GetStale("all"); ok {
			return append([]domain.SpotMarket(nil), stale...), nil
		}
		return nil, err
	}
	c.spot.Set("all", out)
	return append([]domain.SpotMarket(nil), out...), nil
}

// ListProductTags returns unique sector tags from the last quote snapshot.
func (c *Client) ListProductTags(ctx context.Context) ([]string, error) {
	rows, err := c.ListSpotMarkets(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var out []string
	for _, r := range rows {
		for _, t := range r.Tags {
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out, nil
}

// TagsByBase maps ticker → sector tags.
func (c *Client) TagsByBase(ctx context.Context) (map[string][]string, error) {
	rows, err := c.ListSpotMarkets(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(rows))
	for _, r := range rows {
		if r.BaseAsset == "" || len(r.Tags) == 0 {
			continue
		}
		out[r.BaseAsset] = append([]string(nil), r.Tags...)
	}
	return out, nil
}

// GetTicker24h returns the session quote for one symbol.
func (c *Client) GetTicker24h(ctx context.Context, symbol string) (*domain.Ticker24h, error) {
	local := domain.NormalizeSymbol(c.exchange, symbol)
	if local == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if hit, ok := c.tickers.Get(local); ok {
		cp := *hit
		return &cp, nil
	}
	quotes, err := c.fetchQuotes(ctx, []string{local})
	if err != nil {
		// Do not serve an expired last-good ticker as live: paper fills and
		// price alerts treat LastPrice as current (AGENTS.md §6.6).
		return nil, err
	}
	if len(quotes) == 0 {
		return nil, fmt.Errorf("%w: no quote for %s", domain.ErrNotFound, local)
	}
	tk := c.tickerFromQuote(quotes[0])
	if tk == nil {
		return nil, fmt.Errorf("%w: no quote for %s", domain.ErrNotFound, local)
	}
	c.tickers.Set(local, tk)
	cp := *tk
	return &cp, nil
}

// GetCandles fetches Yahoo chart bars and maps them onto Swyngora intervals.
func (c *Client) GetCandles(ctx context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	local := domain.NormalizeSymbol(c.exchange, q.Symbol)
	if local == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if !domain.IsValidIntervalFor(c.exchange, string(q.Interval)) {
		return nil, fmt.Errorf("%w: unsupported interval %q", domain.ErrInvalidArgument, q.Interval)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	cacheable := q.StartTime.IsZero() && q.EndTime.IsZero()
	key := fmt.Sprintf("%s|%s|%d", local, q.Interval, limit)
	if cacheable {
		if hit, ok := c.candles.Get(key); ok {
			return append([]domain.Candle(nil), hit...), nil
		}
	}
	iv, rng := yahooChartWindow(q.Interval)
	u, _ := url.Parse(c.baseURL + "/v8/finance/chart/" + url.PathEscape(c.yahooSymbol(local)))
	qs := u.Query()
	qs.Set("interval", iv)
	qs.Set("range", rng)
	qs.Set("includePrePost", "false")
	u.RawQuery = qs.Encode()
	body, err := c.get(ctx, u.String())
	if err != nil {
		return nil, err
	}
	bars, err := parseChart(body, limit)
	if err != nil {
		return nil, err
	}
	if cacheable {
		c.candles.Set(key, bars)
	}
	return bars, nil
}

// Cleanup drops expired spot/ticker/candle entries so last-good lists cannot live for the process lifetime.
func (c *Client) Cleanup() {
	if c.spot != nil {
		c.spot.Cleanup()
	}
	if c.tickers != nil {
		c.tickers.Cleanup()
	}
	if c.candles != nil {
		c.candles.Cleanup()
	}
}

// GetOrderBook is not available for public cash-equity quotes.
func (c *Client) GetOrderBook(_ context.Context, _ domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return nil, fmt.Errorf("%w: order book is not available for %s", domain.ErrNotFound, c.exchange)
}

func (c *Client) fetchQuotes(ctx context.Context, locals []string) ([]yahooQuote, error) {
	var batches [][]string
	for i := 0; i < len(locals); i += quoteBatchSize {
		end := i + quoteBatchSize
		if end > len(locals) {
			end = len(locals)
		}
		syms := make([]string, 0, end-i)
		for _, loc := range locals[i:end] {
			if y := c.yahooSymbol(loc); y != "" {
				syms = append(syms, y)
			}
		}
		if len(syms) > 0 {
			batches = append(batches, syms)
		}
	}
	if len(batches) == 0 {
		return nil, fmt.Errorf("%w: no symbols to quote", domain.ErrInvalidArgument)
	}
	type result struct {
		quotes []yahooQuote
		err    error
	}
	outCh := make(chan result, len(batches))
	sem := make(chan struct{}, 6)
	for _, batch := range batches {
		batch := batch
		go func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			u, _ := url.Parse(c.baseURL + "/v7/finance/spark")
			qs := u.Query()
			qs.Set("symbols", strings.Join(batch, ","))
			qs.Set("range", "1d")
			qs.Set("interval", "1d")
			u.RawQuery = qs.Encode()
			body, err := c.get(ctx, u.String())
			if err != nil {
				outCh <- result{err: err}
				return
			}
			quotes, err := parseQuotes(body)
			outCh <- result{quotes: quotes, err: err}
		}()
	}
	var all []yahooQuote
	var firstErr error
	for i := 0; i < len(batches); i++ {
		r := <-outCh
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		all = append(all, r.quotes...)
	}
	if len(all) == 0 {
		return nil, firstErr
	}
	return all, nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) postJSON(ctx context.Context, rawURL string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json,text/plain,*/*")
	}
	rawURL := req.URL.String()
	switch {
	case strings.Contains(rawURL, "nasdaq.com"):
		req.Header.Set("Referer", "https://www.nasdaq.com/")
	case strings.Contains(rawURL, "tradingview.com"):
		req.Header.Set("Origin", "https://www.tradingview.com")
		req.Header.Set("Referer", "https://www.tradingview.com/")
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: equity request: %v", domain.ErrUpstream, err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: equity read: %v", domain.ErrUpstream, err)
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: equity status %d", domain.ErrUpstream, res.StatusCode)
	}
	return b, nil
}

func (c *Client) spotFromQuote(q yahooQuote) *domain.SpotMarket {
	local := c.localSymbol(q.Symbol)
	if local == "" || q.RegularMarketPrice == 0 {
		return nil
	}
	vol := q.RegularMarketVolume
	last := q.RegularMarketPrice
	quoteVol := last * vol
	mcap := q.MarketCap
	var tags []string
	if s := strings.TrimSpace(q.Sector); s != "" {
		tags = []string{s}
	}
	status := "TRADING"
	if q.QuoteType != "" && !strings.EqualFold(q.QuoteType, "EQUITY") && !strings.EqualFold(q.QuoteType, "ETF") {
		status = "TRADING"
	}
	row := &domain.SpotMarket{
		Symbol:             local,
		BaseAsset:          local,
		QuoteAsset:         c.quoteAsset,
		Status:             status,
		LastPrice:          fmtFloat(last),
		PriceChange:        fmtFloat(q.RegularMarketChange),
		PriceChangePercent: fmtFloat(q.RegularMarketChangePercent),
		HighPrice:          fmtFloat(q.RegularMarketDayHigh),
		LowPrice:           fmtFloat(q.RegularMarketDayLow),
		Volume:             fmtFloat(vol),
		QuoteVolume:        fmtFloat(quoteVol),
		Tags:               tags,
	}
	if mcap > 0 {
		row.MarketCapCirculating = &mcap
		row.MarketCapTotal = &mcap
		if last > 0 {
			circ := mcap / last
			row.CirculatingSupply = &circ
		}
	}
	return row
}

func (c *Client) tickerFromQuote(q yahooQuote) *domain.Ticker24h {
	local := c.localSymbol(q.Symbol)
	if local == "" || q.RegularMarketPrice == 0 {
		return nil
	}
	now := time.Now().UTC()
	openT := now.Add(-24 * time.Hour)
	if q.RegularMarketTime > 0 {
		now = time.Unix(q.RegularMarketTime, 0).UTC()
		openT = now.Add(-24 * time.Hour)
	}
	return &domain.Ticker24h{
		Symbol:             local,
		PriceChange:        fmtFloat(q.RegularMarketChange),
		PriceChangePercent: fmtFloat(q.RegularMarketChangePercent),
		LastPrice:          fmtFloat(q.RegularMarketPrice),
		OpenPrice:          fmtFloat(q.RegularMarketOpen),
		HighPrice:          fmtFloat(q.RegularMarketDayHigh),
		LowPrice:           fmtFloat(q.RegularMarketDayLow),
		Volume:             fmtFloat(q.RegularMarketVolume),
		QuoteVolume:        fmtFloat(q.RegularMarketPrice * q.RegularMarketVolume),
		OpenTime:           openT,
		CloseTime:          now,
	}
}

func fmtFloat(v float64) string {
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
