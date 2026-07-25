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

// Client implements domain.SupplyPort.
//
// User-facing GetSupply is **cache-only** (no live CoinGecko call).
// Refresh bulk-loads CoinGecko /coins/markets pages into the cache (daily job).
type Client struct {
	baseURL    string
	httpClient *http.Client
	supply     *cache.TTL[*domain.AssetSupply]
	idBySymbol *cache.TTL[string]

	// Refresh tuning (set via Options).
	refreshPages     int
	refreshPageDelay time.Duration
}

// Options configures the CoinGecko client.
type Options struct {
	BaseURL          string
	HTTPClient       *http.Client
	SupplyCache      *cache.TTL[*domain.AssetSupply]
	SymbolCache      *cache.TTL[string]
	RefreshPages     int           // default 4 (×250 coins)
	RefreshPageDelay time.Duration // delay between market pages
}

// NewClient constructs a CoinGecko supply client.
func NewClient(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = "https://api.coingecko.com"
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	pages := opts.RefreshPages
	if pages <= 0 {
		pages = 4
	}
	delay := opts.RefreshPageDelay
	if delay <= 0 {
		delay = 2 * time.Second
	}
	return &Client{
		baseURL:          base,
		httpClient:       hc,
		supply:           opts.SupplyCache,
		idBySymbol:       opts.SymbolCache,
		refreshPages:     pages,
		refreshPageDelay: delay,
	}
}

// GetSupply returns a cached supply snapshot for an asset ticker (e.g. BTC).
// It does **not** call CoinGecko — populate the cache via Refresh (daily job).
func (c *Client) GetSupply(ctx context.Context, asset string) (*domain.AssetSupply, error) {
	_ = ctx
	asset = normalizeAsset(asset)
	if asset == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	if c.supply == nil {
		return nil, fmt.Errorf("%w: supply cache not configured", domain.ErrUpstream)
	}
	if hit, ok := c.supply.Get(asset); ok {
		// Deep copy of pointer fields so callers cannot mutate the cache entry
		// (or each other) by writing through the returned *float64s.
		cp := *hit
		cp.CirculatingSupply = domain.CloneFloatPtr(hit.CirculatingSupply)
		cp.TotalSupply = domain.CloneFloatPtr(hit.TotalSupply)
		cp.MaxSupply = domain.CloneFloatPtr(hit.MaxSupply)
		cp.CurrentPriceUSD = domain.CloneFloatPtr(hit.CurrentPriceUSD)
		return &cp, nil
	}
	return nil, fmt.Errorf("%w: supply for %q not in daily snapshot cache", domain.ErrNotFound, asset)
}

// Refresh bulk-loads top coins by market cap from CoinGecko into the supply cache.
// Safe to call from a background daily job. Returns the number of assets written.
func (c *Client) Refresh(ctx context.Context) (int, error) {
	if c.supply == nil {
		return 0, fmt.Errorf("%w: supply cache not configured", domain.ErrUpstream)
	}
	seen := map[string]struct{}{}
	stored := 0
	asOf := time.Now().UTC()
	for page := 1; page <= c.refreshPages; page++ {
		if err := ctx.Err(); err != nil {
			return stored, err
		}
		n, err := c.refreshMarketsPage(ctx, page, asOf, seen)
		if err != nil {
			return stored, err
		}
		stored += n
		if page < c.refreshPages && c.refreshPageDelay > 0 {
			select {
			case <-ctx.Done():
				return stored, ctx.Err()
			case <-time.After(c.refreshPageDelay):
			}
		}
	}
	// Ensure important tickers (e.g. WBTC) are present even if outside top pages.
	// We always add what we successfully fetched from well-known (even if a later
	// batch fails) and treat the whole well-known step as non-fatal so a transient
	// failure for a few wrapped assets does not invalidate the daily snapshot.
	n, err := c.refreshByIDs(ctx, wellKnownCoinIDs(), asOf, seen)
	stored += n
	if err != nil {
		// Non-fatal: markets pages already loaded.
		return stored, nil
	}
	return stored, nil
}

func (c *Client) refreshMarketsPage(ctx context.Context, page int, asOf time.Time, seen map[string]struct{}) (int, error) {
	params := url.Values{}
	params.Set("vs_currency", "usd")
	params.Set("order", "market_cap_desc")
	params.Set("per_page", "250")
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("sparkline", "false")

	body, err := c.get(ctx, "/api/v3/coins/markets", params)
	if err != nil {
		return 0, err
	}
	var rows []marketRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return 0, fmt.Errorf("%w: decode markets: %v", domain.ErrUpstream, err)
	}

	// First occurrence wins (highest mcap first via order=market_cap_desc).
	written := 0
	for _, row := range rows {
		asset := strings.ToUpper(strings.TrimSpace(row.Symbol))
		if asset == "" {
			continue
		}
		if _, ok := seen[asset]; ok {
			continue // keep higher-ranked entry from earlier pages/rows
		}
		seen[asset] = struct{}{}
		sup := &domain.AssetSupply{
			Asset:             asset,
			Name:              row.Name,
			ProviderID:        row.ID,
			CirculatingSupply: row.CirculatingSupply,
			TotalSupply:       row.TotalSupply,
			MaxSupply:         row.MaxSupply,
			CurrentPriceUSD:   row.CurrentPrice,
			AsOf:              asOf,
			Source:            "coingecko",
		}
		c.supply.Set(asset, sup)
		if c.idBySymbol != nil && row.ID != "" {
			c.idBySymbol.Set(asset, row.ID)
		}
		written++
	}
	return written, nil
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", domain.ErrUpstream, err)
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: coingecko", domain.ErrRateLimited)
	case http.StatusNotFound:
		return nil, fmt.Errorf("%w: coin", domain.ErrNotFound)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: status %d: %s", domain.ErrUpstream, resp.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

type marketRow struct {
	ID                string   `json:"id"`
	Symbol            string   `json:"symbol"`
	Name              string   `json:"name"`
	CurrentPrice      *float64 `json:"current_price"`
	CirculatingSupply *float64 `json:"circulating_supply"`
	TotalSupply       *float64 `json:"total_supply"`
	MaxSupply         *float64 `json:"max_supply"`
}


// wellKnownCoinIDs are always refreshed so high-signal assets (including wrapped
// majors that may fall outside top market-cap pages) stay in the daily cache.
func wellKnownCoinIDs() []string {
	return []string{
		"bitcoin", "ethereum", "binancecoin", "ripple", "solana", "cardano",
		"dogecoin", "tron", "polkadot", "avalanche-2", "chainlink",
		"matic-network", "polygon-ecosystem-token", "litecoin", "bitcoin-cash",
		"cosmos", "uniswap", "stellar", "near", "aptos", "arbitrum", "optimism",
		"sui", "pepe", "shiba-inu", "the-open-network", "tether", "usd-coin",
		"wrapped-bitcoin", "wrapped-beacon-eth", "staked-ether",
	}
}

func (c *Client) refreshByIDs(ctx context.Context, ids []string, asOf time.Time, seen map[string]struct{}) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	const batch = 50
	written := 0
	for i := 0; i < len(ids); i += batch {
		end := i + batch
		if end > len(ids) {
			end = len(ids)
		}
		params := url.Values{}
		params.Set("vs_currency", "usd")
		params.Set("ids", strings.Join(ids[i:end], ","))
		params.Set("sparkline", "false")
		body, err := c.get(ctx, "/api/v3/coins/markets", params)
		if err != nil {
			return written, err
		}
		var rows []marketRow
		if err := json.Unmarshal(body, &rows); err != nil {
			return written, fmt.Errorf("%w: decode markets by id: %v", domain.ErrUpstream, err)
		}
		for _, row := range rows {
			asset := strings.ToUpper(strings.TrimSpace(row.Symbol))
			if asset == "" {
				continue
			}
			sup := &domain.AssetSupply{
				Asset:             asset,
				Name:              row.Name,
				ProviderID:        row.ID,
				CirculatingSupply: row.CirculatingSupply,
				TotalSupply:       row.TotalSupply,
				MaxSupply:         row.MaxSupply,
				CurrentPriceUSD:   row.CurrentPrice,
				AsOf:              asOf,
				Source:            "coingecko",
			}
			c.supply.Set(asset, sup)
			if c.idBySymbol != nil && row.ID != "" {
				c.idBySymbol.Set(asset, row.ID)
			}
			if _, ok := seen[asset]; !ok {
				seen[asset] = struct{}{}
				written++
			}
		}
	}
	return written, nil
}

// normalizeAsset maps a user-facing ticker or pair to a base asset key for the supply cache.
//
// Only USD-stable quote suffixes are stripped (BTCUSDT → BTC). Crypto quote suffixes
// like BTC/ETH/BNB are NOT stripped — otherwise WBTC becomes "W", RENBTC becomes "REN",
// and market caps are computed against the wrong supply.
func normalizeAsset(s string) string {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)
	for _, q := range []string{"USDT", "USDC", "BUSD", "FDUSD", "TUSD", "DAI", "USD"} {
		if len(upper) > len(q) && strings.HasSuffix(upper, q) {
			base := strings.TrimSuffix(upper, q)
			if base != "" && base != q {
				return base
			}
		}
	}
	return upper
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
