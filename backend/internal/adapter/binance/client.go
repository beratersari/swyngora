package binance

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

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Client implements domain.MarketDataPort against Binance public REST APIs.
// No API key is required for market data endpoints used here.
type Client struct {
	baseURL    string
	httpClient *http.Client
	candles    *cache.TTL[[]domain.Candle]
	tickers    *cache.TTL[*domain.Ticker24h]
}

// Options configures the Binance client.
type Options struct {
	BaseURL       string
	HTTPClient    *http.Client
	CandleCache   *cache.TTL[[]domain.Candle]
	TickerCache   *cache.TTL[*domain.Ticker24h]
}

// NewClient constructs a Binance market-data client.
func NewClient(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = "https://api.binance.com"
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    base,
		httpClient: hc,
		candles:    opts.CandleCache,
		tickers:    opts.TickerCache,
	}
}

// GetCandles fetches OHLCV klines for the given query.
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

	cacheKey := fmt.Sprintf("%s|%s|%d|%d|%d",
		symbol, q.Interval, limit,
		q.StartTime.UnixMilli(), q.EndTime.UnixMilli())
	if c.candles != nil {
		if hit, ok := c.candles.Get(cacheKey); ok {
			return append([]domain.Candle(nil), hit...), nil
		}
	}

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

	body, err := c.get(ctx, "/api/v3/klines", params)
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

	if c.candles != nil {
		c.candles.Set(cacheKey, append([]domain.Candle(nil), out...))
	}
	return out, nil
}

// GetTicker24h fetches 24-hour rolling statistics for a symbol.
func (c *Client) GetTicker24h(ctx context.Context, symbol string) (*domain.Ticker24h, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}

	if c.tickers != nil {
		if hit, ok := c.tickers.Get(symbol); ok {
			return hit, nil
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	body, err := c.get(ctx, "/api/v3/ticker/24hr", params)
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
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
