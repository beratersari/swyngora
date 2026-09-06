package deribit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL = "https://www.deribit.com"
	maxBody        = 8 << 20
	bookPath       = "/api/v2/public/get_book_summary_by_currency"
	tickerPath     = "/api/v2/public/ticker"
	indexPath      = "/api/v2/public/get_index_price"
	dvolHistPath   = "/api/v2/public/get_volatility_index_data"
)

// Client implements domain.OptionsChainPort against Deribit public REST.
type Client struct {
	baseURL    string
	httpClient *http.Client
	quotes     *cache.TTL[[]domain.OptionQuote]
	dvol       *cache.TTL[domain.OptionsDVOL]
	dvolHist   *cache.TTL[[]domain.OptionsDVOLPoint]
	index      *cache.TTL[float64]
	sf         singleflight.Group
}

// Options configures the Deribit client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
	CacheTTL   time.Duration
}

// New constructs a Deribit options client. No API key is required.
func New(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{
			Timeout: 20 * time.Second,
			// Cloudflare sometimes resets the default Go HTTP/2 fingerprint.
			Transport: &http.Transport{ForceAttemptHTTP2: false, Proxy: http.ProxyFromEnvironment},
		}
	}
	ttl := opts.CacheTTL
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &Client{
		baseURL:    base,
		httpClient: hc,
		quotes:     cache.New[[]domain.OptionQuote](ttl),
		dvol:       cache.New[domain.OptionsDVOL](ttl),
		dvolHist:   cache.New[[]domain.OptionsDVOLPoint](5 * time.Minute),
		index:      cache.New[float64](ttl),
	}
}

// Cleanup evicts expired cache entries.
func (c *Client) Cleanup() {
	if c == nil {
		return
	}
	c.quotes.Cleanup()
	c.dvol.Cleanup()
	c.dvolHist.Cleanup()
	c.index.Cleanup()
}

// GetOptionQuotes returns listed inverse options for BTC or ETH.
func (c *Client) GetOptionQuotes(ctx context.Context, asset string) ([]domain.OptionQuote, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: deribit client not configured", domain.ErrUpstream)
	}
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return nil, err
	}
	if hit, ok := c.quotes.Get(asset); ok {
		return hit, nil
	}
	v, err, _ := c.sf.Do("q:"+asset, func() (any, error) {
		if hit, ok := c.quotes.Get(asset); ok {
			return hit, nil
		}
		rows, err := c.fetchQuotes(ctx, asset)
		if err != nil {
			return nil, err
		}
		c.quotes.Set(asset, rows)
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]domain.OptionQuote), nil
}

// GetOptionsDVOL returns the 30-day implied vol index and optional 24h change.
func (c *Client) GetOptionsDVOL(ctx context.Context, asset string) (domain.OptionsDVOL, error) {
	if c == nil {
		return domain.OptionsDVOL{}, fmt.Errorf("%w: deribit client not configured", domain.ErrUpstream)
	}
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return domain.OptionsDVOL{}, err
	}
	if hit, ok := c.dvol.Get(asset); ok {
		return hit, nil
	}
	v, err, _ := c.sf.Do("d:"+asset, func() (any, error) {
		if hit, ok := c.dvol.Get(asset); ok {
			return hit, nil
		}
		got, err := c.fetchDVOL(ctx, asset)
		if err != nil {
			return domain.OptionsDVOL{}, err
		}
		c.dvol.Set(asset, got)
		return got, nil
	})
	if err != nil {
		return domain.OptionsDVOL{}, err
	}
	return v.(domain.OptionsDVOL), nil
}

// GetOptionsIndex returns the USD index used as the options underlying.
func (c *Client) GetOptionsIndex(ctx context.Context, asset string) (float64, error) {
	if c == nil {
		return 0, fmt.Errorf("%w: deribit client not configured", domain.ErrUpstream)
	}
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return 0, err
	}
	if hit, ok := c.index.Get(asset); ok {
		return hit, nil
	}
	v, err, _ := c.sf.Do("i:"+asset, func() (any, error) {
		if hit, ok := c.index.Get(asset); ok {
			return hit, nil
		}
		px, err := c.fetchIndex(ctx, asset)
		if err != nil {
			return 0.0, err
		}
		c.index.Set(asset, px)
		return px, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(float64), nil
}

type rpcEnvelope struct {
	Error  *rpcError       `json:"error"`
	Result json.RawMessage `json:"result"`
}

type rpcError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type bookRow struct {
	InstrumentName  string   `json:"instrument_name"`
	OpenInterest    float64  `json:"open_interest"`
	Volume          float64  `json:"volume"`
	VolumeUSD       float64  `json:"volume_usd"`
	MarkIV          *float64 `json:"mark_iv"`
	UnderlyingPrice float64  `json:"underlying_price"`
}

type tickerResult struct {
	LastPrice  *float64 `json:"last_price"`
	MarkPrice  float64  `json:"mark_price"`
	IndexPrice float64  `json:"index_price"`
	Stats      struct {
		PriceChange *float64 `json:"price_change"`
	} `json:"stats"`
}

type indexResult struct {
	IndexPrice float64 `json:"index_price"`
}

type volIndexResult struct {
	Data [][]float64 `json:"data"`
}

func (c *Client) fetchQuotes(ctx context.Context, asset string) ([]domain.OptionQuote, error) {
	q := url.Values{}
	q.Set("currency", asset)
	q.Set("kind", "option")
	body, err := c.get(ctx, bookPath, q)
	if err != nil {
		return nil, err
	}
	var rows []bookRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: deribit book json: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.OptionQuote, 0, len(rows))
	for _, r := range rows {
		a, exp, strike, side, ok := domain.ParseOptionInstrument(r.InstrumentName)
		if !ok || a != asset {
			continue
		}
		q := domain.OptionQuote{
			Instrument:   r.InstrumentName,
			Asset:        a,
			Expiry:       exp,
			Strike:       strike,
			Side:         side,
			OpenInterest: r.OpenInterest,
			Volume:       r.Volume,
			VolumeUSD:    r.VolumeUSD,
			Underlying:   r.UnderlyingPrice,
		}
		if r.MarkIV != nil && *r.MarkIV > 0 {
			q.MarkIV = *r.MarkIV
		}
		out = append(out, q)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no deribit options for %s", domain.ErrNotFound, asset)
	}
	return out, nil
}

func (c *Client) fetchDVOL(ctx context.Context, asset string) (domain.OptionsDVOL, error) {
	q := url.Values{}
	q.Set("instrument_name", asset+"-DVOL")
	body, err := c.get(ctx, tickerPath, q)
	if err != nil {
		return domain.OptionsDVOL{}, err
	}
	var tick tickerResult
	if err := json.Unmarshal(body, &tick); err != nil {
		return domain.OptionsDVOL{}, fmt.Errorf("%w: deribit dvol json: %v", domain.ErrUpstream, err)
	}
	val := tick.MarkPrice
	if tick.LastPrice != nil && *tick.LastPrice > 0 {
		val = *tick.LastPrice
	}
	if val <= 0 && tick.IndexPrice > 0 {
		val = tick.IndexPrice
	}
	if val <= 0 {
		return domain.OptionsDVOL{}, fmt.Errorf("%w: deribit dvol empty", domain.ErrUpstream)
	}
	out := domain.OptionsDVOL{Value: val}
	if chg, ok := c.fetchDVOLChange(ctx, asset, val); ok {
		out.Change24h = chg
		out.HasChange = true
	} else if tick.Stats.PriceChange != nil {
		// Deribit ticker price_change is percent of the index, not points.
		out.Change24h = val * (*tick.Stats.PriceChange) / 100
		out.HasChange = true
	}
	return out, nil
}

// GetOptionsDVOLHistory returns hourly DVOL closes from Deribit public REST.
func (c *Client) GetOptionsDVOLHistory(ctx context.Context, asset string, from, to time.Time) ([]domain.OptionsDVOLPoint, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: deribit client not configured", domain.ErrUpstream)
	}
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return nil, err
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() || to.Sub(from) > 8*24*time.Hour {
		from = to.Add(-domain.OptionsHistoryLookback)
	}
	key := "h:" + asset
	if hit, ok := c.dvolHist.Get(key); ok {
		return hit, nil
	}
	v, err, _ := c.sf.Do(key, func() (any, error) {
		if hit, ok := c.dvolHist.Get(key); ok {
			return hit, nil
		}
		rows, err := c.fetchDVOLHistory(ctx, asset, from, to)
		if err != nil {
			return nil, err
		}
		c.dvolHist.Set(key, rows)
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]domain.OptionsDVOLPoint), nil
}

func (c *Client) fetchDVOLHistory(ctx context.Context, asset string, from, to time.Time) ([]domain.OptionsDVOLPoint, error) {
	q := url.Values{}
	q.Set("currency", asset)
	q.Set("resolution", "3600")
	q.Set("start_timestamp", fmt.Sprintf("%d", from.UTC().UnixMilli()))
	q.Set("end_timestamp", fmt.Sprintf("%d", to.UTC().UnixMilli()))
	body, err := c.get(ctx, dvolHistPath, q)
	if err != nil {
		return nil, err
	}
	var hist volIndexResult
	if err := json.Unmarshal(body, &hist); err != nil {
		return nil, fmt.Errorf("%w: deribit dvol history json: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.OptionsDVOLPoint, 0, len(hist.Data))
	for _, row := range hist.Data {
		if len(row) < 5 || row[4] <= 0 {
			continue
		}
		out = append(out, domain.OptionsDVOLPoint{
			Time:  time.UnixMilli(int64(row[0])).UTC(),
			Value: row[4],
		})
	}
	return out, nil
}

func (c *Client) fetchDVOLChange(ctx context.Context, asset string, current float64) (float64, bool) {
	end := time.Now().UTC()
	start := end.Add(-26 * time.Hour)
	q := url.Values{}
	q.Set("currency", asset)
	q.Set("resolution", "3600")
	q.Set("start_timestamp", fmt.Sprintf("%d", start.UnixMilli()))
	q.Set("end_timestamp", fmt.Sprintf("%d", end.UnixMilli()))
	body, err := c.get(ctx, dvolHistPath, q)
	if err != nil {
		return 0, false
	}
	var hist volIndexResult
	if err := json.Unmarshal(body, &hist); err != nil || len(hist.Data) < 2 {
		return 0, false
	}
	// Each row: [timestamp, open, high, low, close]
	target := end.Add(-24 * time.Hour).UnixMilli()
	var past float64
	found := false
	for _, row := range hist.Data {
		if len(row) < 5 {
			continue
		}
		if int64(row[0]) <= target && row[4] > 0 {
			past = row[4]
			found = true
		}
	}
	if !found || past <= 0 {
		return 0, false
	}
	return current - past, true
}

func (c *Client) fetchIndex(ctx context.Context, asset string) (float64, error) {
	q := url.Values{}
	q.Set("index_name", strings.ToLower(asset)+"_usd")
	body, err := c.get(ctx, indexPath, q)
	if err != nil {
		return 0, err
	}
	var idx indexResult
	if err := json.Unmarshal(body, &idx); err != nil {
		return 0, fmt.Errorf("%w: deribit index json: %v", domain.ErrUpstream, err)
	}
	if idx.IndexPrice <= 0 {
		return 0, fmt.Errorf("%w: deribit index empty", domain.ErrUpstream)
	}
	return idx.IndexPrice, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values) (json.RawMessage, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: deribit request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora/0.1 (options-desk)")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: deribit fetch: %v", domain.ErrUpstream, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: deribit read: %v", domain.ErrUpstream, err)
	}
	if res.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: deribit", domain.ErrRateLimited)
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: deribit status %d", domain.ErrUpstream, res.StatusCode)
	}
	var env rpcEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("%w: deribit rpc json: %v", domain.ErrUpstream, err)
	}
	if env.Error != nil {
		return nil, fmt.Errorf("%w: deribit: %s", domain.ErrUpstream, env.Error.Message)
	}
	if len(env.Result) == 0 || string(env.Result) == "null" {
		return nil, fmt.Errorf("%w: deribit empty result", domain.ErrUpstream)
	}
	return env.Result, nil
}
