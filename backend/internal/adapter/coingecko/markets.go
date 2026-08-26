// Package coingecko is a delist-only supply fallback (public /coins/markets).
// Live tape and the default supply snapshot stay on Binance (ADR 0001).
package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL = "https://api.coingecko.com"
	marketsPath    = "/api/v3/coins/markets"
	httpTimeout    = 18 * time.Second
	cacheTTL       = 6 * time.Hour
)

// Client implements domain.SymbolSupplyFallback.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	cache       *cache.TTL[*domain.AssetSupply]
	ohlcCache     *cache.TTL[[]domain.Candle]
	changeCache    *cache.TTL[*float64]
	changeAbsCache *cache.TTL[*float64]
	contractCache *cache.TTL[[]domain.AssetContract]
}

// Options configures the public markets client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
	Cache      *cache.TTL[*domain.AssetSupply]
}

// New returns a CoinGecko markets client.
func New(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: httpTimeout}
	}
	cch := opts.Cache
	if cch == nil {
		cch = cache.New[*domain.AssetSupply](cacheTTL)
	}
	return &Client{
		baseURL:     base,
		httpClient:  hc,
		cache:       cch,
		ohlcCache:     cache.New[[]domain.Candle](ohlcCacheTTL),
		changeCache:    cache.New[*float64](cacheTTL),
		changeAbsCache: cache.New[*float64](cacheTTL),
		contractCache: cache.New[[]domain.AssetContract](cacheTTL),
	}
}

// SupplyBySymbols returns the best exact-symbol match per ticker (highest mcap first).
func (c *Client) SupplyBySymbols(ctx context.Context, symbols []string) (map[string]*domain.AssetSupply, error) {
	out := make(map[string]*domain.AssetSupply)
	var missing []string
	seen := map[string]struct{}{}
	for _, raw := range symbols {
		sym := strings.ToUpper(strings.TrimSpace(raw))
		if sym == "" {
			continue
		}
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}
		if hit, ok := c.cache.Get(sym); ok && hit != nil {
			out[sym] = cloneSupply(hit)
			continue
		}
		missing = append(missing, strings.ToLower(sym))
	}
	if len(missing) == 0 {
		return out, nil
	}
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	rows, err := c.fetchMarkets(ctx, missing)
	if err != nil {
		if len(out) > 0 {
			return out, nil
		}
		return nil, err
	}
	now := time.Now().UTC()
	applyRows := func(rows []marketRow) {
		for _, row := range rows {
			sym := strings.ToUpper(strings.TrimSpace(row.Symbol))
			if sym == "" {
				continue
			}
			if _, ok := out[sym]; ok {
				continue // already took the higher-ranked match
			}
			sup := row.toSupply(now)
			if sup.CirculatingSupply == nil && sup.TotalSupply == nil && sup.MaxSupply == nil {
				continue
			}
			c.cache.Set(sym, sup)
			if row.PriceChangePct24h != nil {
				c.changeCache.Set(sym, cloneF(row.PriceChangePct24h))
			}
			if row.PriceChange24h != nil {
				c.changeAbsCache.Set(sym, cloneF(row.PriceChange24h))
			}
			out[sym] = cloneSupply(sup)
		}
	}
	applyRows(rows)
	var leftover []string
	for _, raw := range missing {
		sym := strings.ToUpper(raw)
		if _, ok := out[sym]; !ok {
			leftover = append(leftover, sym)
		}
	}
	for _, sym := range leftover {
		id, ok := c.searchID(ctx, sym)
		if !ok {
			continue
		}
		extra, err := c.fetchMarketsByIDs(ctx, []string{id})
		if err != nil {
			continue
		}
		applyRows(extra)
	}
	return out, nil
}

type marketRow struct {
	ID                string   `json:"id"`
	Symbol            string   `json:"symbol"`
	Name              string   `json:"name"`
	CurrentPrice      *float64 `json:"current_price"`
	PriceChange24h    *float64 `json:"price_change_24h"`
	PriceChangePct24h *float64 `json:"price_change_percentage_24h"`
	CirculatingSupply *float64 `json:"circulating_supply"`
	TotalSupply       *float64 `json:"total_supply"`
	MaxSupply         *float64 `json:"max_supply"`
	MarketCap         *float64 `json:"market_cap"`
}

func (r marketRow) toSupply(asOf time.Time) *domain.AssetSupply {
	return &domain.AssetSupply{
		Asset:             strings.ToUpper(strings.TrimSpace(r.Symbol)),
		Name:              strings.TrimSpace(r.Name),
		ProviderID:        r.ID,
		CirculatingSupply: cloneF(r.CirculatingSupply),
		TotalSupply:       cloneF(r.TotalSupply),
		MaxSupply:         cloneF(r.MaxSupply),
		CurrentPriceUSD:   cloneF(r.CurrentPrice),
		AsOf:              asOf,
		Source:            "coingecko",
	}
}

func (c *Client) fetchMarketsByIDs(ctx context.Context, ids []string) ([]marketRow, error) {
	u, err := url.Parse(c.baseURL + marketsPath)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("vs_currency", "usd")
	q.Set("ids", strings.Join(ids, ","))
	q.Set("order", "market_cap_desc")
	q.Set("per_page", "250")
	q.Set("page", "1")
	q.Set("sparkline", "false")
	u.RawQuery = q.Encode()
	return c.getMarkets(ctx, u.String())
}

type searchResponse struct {
	Coins []struct {
		ID     string `json:"id"`
		Symbol string `json:"symbol"`
	} `json:"coins"`
}

func (c *Client) searchID(ctx context.Context, symbol string) (string, bool) {
	u := c.baseURL + "/api/v3/search?query=" + url.QueryEscape(symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", false
	}
	var parsed searchResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&parsed); err != nil {
		return "", false
	}
	want := strings.ToLower(symbol)
	for _, coin := range parsed.Coins {
		if strings.ToLower(strings.TrimSpace(coin.Symbol)) == want && coin.ID != "" {
			return coin.ID, true
		}
	}
	return "", false
}

func (c *Client) fetchMarkets(ctx context.Context, symbols []string) ([]marketRow, error) {
	u, err := url.Parse(c.baseURL + marketsPath)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("vs_currency", "usd")
	q.Set("symbols", strings.Join(symbols, ","))
	q.Set("order", "market_cap_desc")
	q.Set("per_page", "250")
	q.Set("page", "1")
	q.Set("sparkline", "false")
	u.RawQuery = q.Encode()
	return c.getMarkets(ctx, u.String())
}

func (c *Client) getMarkets(ctx context.Context, rawURL string) ([]marketRow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build coingecko request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: coingecko request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read coingecko body: %v", domain.ErrUpstream, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: coingecko status 429", domain.ErrRateLimited)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: coingecko status %d: %s", domain.ErrUpstream, resp.StatusCode, truncate(string(body), 160))
	}
	var rows []marketRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: decode coingecko markets: %v", domain.ErrUpstream, err)
	}
	return rows, nil
}

func cloneSupply(s *domain.AssetSupply) *domain.AssetSupply {
	if s == nil {
		return nil
	}
	cp := *s
	cp.CirculatingSupply = domain.CloneFloatPtr(s.CirculatingSupply)
	cp.TotalSupply = domain.CloneFloatPtr(s.TotalSupply)
	cp.MaxSupply = domain.CloneFloatPtr(s.MaxSupply)
	cp.CurrentPriceUSD = domain.CloneFloatPtr(s.CurrentPriceUSD)
	return &cp
}

func cloneF(p *float64) *float64 { return domain.CloneFloatPtr(p) }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
