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

// Client implements domain.SupplyPort using the free CoinGecko public API.
// Binance does not publish circulating/total/max supply on public market endpoints.
type Client struct {
	baseURL    string
	httpClient *http.Client
	supply     *cache.TTL[*domain.AssetSupply]
	// idBySymbol caches resolved CoinGecko IDs for tickers (BTC -> bitcoin).
	idBySymbol *cache.TTL[string]
}

// Options configures the CoinGecko client.
type Options struct {
	BaseURL      string
	HTTPClient   *http.Client
	SupplyCache  *cache.TTL[*domain.AssetSupply]
	SymbolCache  *cache.TTL[string]
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
	return &Client{
		baseURL:    base,
		httpClient: hc,
		supply:     opts.SupplyCache,
		idBySymbol: opts.SymbolCache,
	}
}

// GetSupply returns circulating / total / max supply for an asset ticker (e.g. BTC, ETH).
func (c *Client) GetSupply(ctx context.Context, asset string) (*domain.AssetSupply, error) {
	asset = normalizeAsset(asset)
	if asset == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}

	if c.supply != nil {
		if hit, ok := c.supply.Get(asset); ok {
			return hit, nil
		}
	}

	id, err := c.resolveID(ctx, asset)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("localization", "false")
	params.Set("tickers", "false")
	params.Set("market_data", "true")
	params.Set("community_data", "false")
	params.Set("developer_data", "false")
	params.Set("sparkline", "false")

	body, err := c.get(ctx, "/api/v3/coins/"+url.PathEscape(id), params)
	if err != nil {
		return nil, err
	}

	var raw coinResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode coin: %v", domain.ErrUpstream, err)
	}

	sup := &domain.AssetSupply{
		Asset:               strings.ToUpper(raw.Symbol),
		Name:                raw.Name,
		ProviderID:          raw.ID,
		CirculatingSupply:   raw.MarketData.CirculatingSupply,
		TotalSupply:         raw.MarketData.TotalSupply,
		MaxSupply:           raw.MarketData.MaxSupply,
		CurrentPriceUSD:     raw.MarketData.CurrentPrice.USD,
		AsOf:                time.Now().UTC(),
		Source:              "coingecko",
	}
	if sup.Asset == "" {
		sup.Asset = asset
	}

	if c.supply != nil {
		c.supply.Set(asset, sup)
	}
	return sup, nil
}

func (c *Client) resolveID(ctx context.Context, asset string) (string, error) {
	if wellKnown, ok := wellKnownIDs[asset]; ok {
		return wellKnown, nil
	}
	if c.idBySymbol != nil {
		if id, ok := c.idBySymbol.Get(asset); ok {
			return id, nil
		}
	}

	// Search free endpoint — pick the first coin whose symbol matches exactly.
	params := url.Values{}
	params.Set("query", asset)
	body, err := c.get(ctx, "/api/v3/search", params)
	if err != nil {
		return "", err
	}
	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return "", fmt.Errorf("%w: decode search: %v", domain.ErrUpstream, err)
	}
	want := strings.ToLower(asset)
	for _, coin := range sr.Coins {
		if strings.ToLower(coin.Symbol) == want {
			if c.idBySymbol != nil {
				c.idBySymbol.Set(asset, coin.ID)
			}
			return coin.ID, nil
		}
	}
	return "", fmt.Errorf("%w: asset %q", domain.ErrNotFound, asset)
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

// wellKnownIDs avoids an extra search call for the most common assets.
var wellKnownIDs = map[string]string{
	"BTC":  "bitcoin",
	"ETH":  "ethereum",
	"BNB":  "binancecoin",
	"XRP":  "ripple",
	"SOL":  "solana",
	"ADA":  "cardano",
	"DOGE": "dogecoin",
	"TRX":  "tron",
	"DOT":  "polkadot",
	"AVAX": "avalanche-2",
	"LINK": "chainlink",
	"MATIC": "matic-network",
	"POL":  "polygon-ecosystem-token",
	"LTC":  "litecoin",
	"BCH":  "bitcoin-cash",
	"ATOM": "cosmos",
	"UNI":  "uniswap",
	"XLM":  "stellar",
	"NEAR": "near",
	"APT":  "aptos",
	"ARB":  "arbitrum",
	"OP":   "optimism",
	"SUI":  "sui",
	"PEPE": "pepe",
	"SHIB": "shiba-inu",
	"TON":  "the-open-network",
	"USDT": "tether",
	"USDC": "usd-coin",
}

type coinResponse struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	MarketData struct {
		CirculatingSupply *float64 `json:"circulating_supply"`
		TotalSupply       *float64 `json:"total_supply"`
		MaxSupply         *float64 `json:"max_supply"`
		CurrentPrice      struct {
			USD *float64 `json:"usd"`
		} `json:"current_price"`
	} `json:"market_data"`
}

type searchResponse struct {
	Coins []struct {
		ID     string `json:"id"`
		Symbol string `json:"symbol"`
		Name   string `json:"name"`
	} `json:"coins"`
}

func normalizeAsset(s string) string {
	s = strings.TrimSpace(s)
	// Allow pairs like BTCUSDT — strip common quote suffixes for convenience.
	upper := strings.ToUpper(s)
	for _, q := range []string{"USDT", "USDC", "BUSD", "FDUSD", "TUSD", "BTC", "ETH", "BNB"} {
		if len(upper) > len(q) && strings.HasSuffix(upper, q) {
			base := strings.TrimSuffix(upper, q)
			// Avoid stripping when the whole ticker is the quote (e.g. "USDT").
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
