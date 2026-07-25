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

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// marketingSymbolListPath is a public Binance web API used by the marketing/price UI.
// Official Spot REST does not expose supply; this list includes circulating, total, and max.
const marketingSymbolListPath = "/bapi/composite/v1/public/marketing/symbol/list"

// productCatalogPath lists spot products with tags (used to exclude bStocks / commodities).
const productCatalogPath = "/bapi/asset/v2/public/asset-service/product/get-products"

// GetSupply returns a cached supply snapshot for an asset ticker (e.g. BTC).
// It does **not** call Binance on the request path — populate via Refresh.
//
// Lookup order: exact uppercased ticker first (so RLUSD/BFUSD are not mangled),
// then strip a USD-stable quote suffix for pair forms (BTCUSDT → BTC).
func (c *Client) GetSupply(ctx context.Context, asset string) (*domain.AssetSupply, error) {
	_ = ctx
	key := strings.ToUpper(strings.TrimSpace(asset))
	if key == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	if c.supply == nil {
		return nil, fmt.Errorf("%w: supply cache not configured", domain.ErrUpstream)
	}
	if hit, ok := c.supply.Get(key); ok {
		return cloneSupply(hit), nil
	}
	if base := stripStableQuoteSuffix(key); base != key {
		if hit, ok := c.supply.Get(base); ok {
			return cloneSupply(hit), nil
		}
		key = base // for clearer not-found message
	}
	return nil, fmt.Errorf("%w: supply for %q not in Binance supply snapshot cache", domain.ErrNotFound, key)
}

func cloneSupply(hit *domain.AssetSupply) *domain.AssetSupply {
	cp := *hit
	cp.CirculatingSupply = domain.CloneFloatPtr(hit.CirculatingSupply)
	cp.TotalSupply = domain.CloneFloatPtr(hit.TotalSupply)
	cp.MaxSupply = domain.CloneFloatPtr(hit.MaxSupply)
	cp.CurrentPriceUSD = domain.CloneFloatPtr(hit.CurrentPriceUSD)
	return &cp
}

// Refresh bulk-loads circulating / total / max supply from Binance marketing symbol list.
// Safe to call from the daily supply job. Returns the number of unique base assets stored.
func (c *Client) Refresh(ctx context.Context) (int, error) {
	if c.supply == nil {
		return 0, fmt.Errorf("%w: supply cache not configured", domain.ErrUpstream)
	}
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	v, err, _ := c.supplySF.Do("refresh", func() (any, error) {
		// Detached context: one job cancel mid-flight must not poison shared work
		// if another refresh waiter exists (same pattern as ListSpotMarkets).
		return c.fetchAndStoreSupply(context.Background())
	})
	if err != nil {
		return 0, err
	}
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	return v.(int), nil
}

func (c *Client) fetchAndStoreSupply(ctx context.Context) (int, error) {
	body, err := c.getProduct(ctx, marketingSymbolListPath, nil)
	if err != nil {
		return 0, err
	}

	var envelope marketingSymbolListResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return 0, fmt.Errorf("%w: decode marketing symbol list: %v", domain.ErrUpstream, err)
	}
	if !envelope.Success && envelope.Code != "" && envelope.Code != "000000" {
		return 0, fmt.Errorf("%w: binance marketing symbol list code=%s msg=%s",
			domain.ErrUpstream, envelope.Code, envelope.Message)
	}
	if len(envelope.Data) == 0 {
		return 0, fmt.Errorf("%w: empty marketing symbol list", domain.ErrUpstream)
	}

	asOf := time.Now().UTC()
	stored := 0
	for _, row := range envelope.Data {
		// Crypto-only: drop tokenized stocks (bStocks) and commodity wrappers.
		if hasNonCryptoTag(row.Tags) {
			continue
		}
		base := strings.ToUpper(strings.TrimSpace(row.Name))
		if base == "" {
			// Fallback: strip quote from pair symbol (e.g. BTCUSDT).
			base = stripStableQuoteSuffix(strings.ToUpper(strings.TrimSpace(row.Symbol)))
		}
		if base == "" {
			continue
		}

		circ, hasCirc := parseOptionalFloat(row.CirculatingSupply)
		total, hasTotal := parseOptionalFloat(row.TotalSupply)
		max, hasMax := parseOptionalFloat(row.MaxSupply)
		// Skip rows with no usable supply metrics at all.
		if (!hasCirc || circ <= 0) && (!hasTotal || total <= 0) && (!hasMax || max <= 0) {
			continue
		}

		name := strings.TrimSpace(row.FullName)
		if name == "" {
			name = strings.TrimSpace(row.LocalFullName)
		}
		if name == "" {
			name = base
		}

		var usd *float64
		if px, ok := parseOptionalFloat(row.Price); ok && px > 0 {
			usd = &px
		}

		sup := &domain.AssetSupply{
			Asset:      base,
			Name:       name,
			ProviderID: base,
			AsOf:       asOf,
			Source:     "binance",
		}
		if hasCirc && circ > 0 {
			sup.CirculatingSupply = &circ
		}
		if hasTotal && total > 0 {
			sup.TotalSupply = &total
		}
		// maxSupply null / infiniteSupply → leave MaxSupply nil (UI may show ∞ for max mcap).
		if hasMax && max > 0 && !row.InfiniteSupply {
			sup.MaxSupply = &max
		}
		if usd != nil {
			sup.CurrentPriceUSD = usd
		}
		c.supply.Set(base, sup)
		stored++
	}
	if stored == 0 {
		return 0, fmt.Errorf("%w: no supply rows stored from marketing symbol list", domain.ErrUpstream)
	}
	return stored, nil
}

func (c *Client) getProduct(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u := c.productBaseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build product request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: product request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read product body: %v", domain.ErrUpstream, err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusTeapot {
		return nil, fmt.Errorf("%w: binance product status %d", domain.ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: marketing symbol list", domain.ErrNotFound)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: product status %d: %s", domain.ErrUpstream, resp.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

type marketingSymbolListResponse struct {
	Code    string               `json:"code"`
	Message string               `json:"message"`
	Success bool                 `json:"success"`
	Data    []marketingSymbolRow `json:"data"`
}

// marketingSymbolRow matches Binance marketing symbol/list JSON.
type marketingSymbolRow struct {
	Name              string          `json:"name"`     // base ticker, e.g. BTC
	FullName          string          `json:"fullName"` // e.g. Bitcoin
	LocalFullName     string          `json:"localFullName"`
	Symbol            string          `json:"symbol"` // pair, e.g. BTCUSDT
	Tags              []string        `json:"tags"`
	CirculatingSupply json.RawMessage `json:"circulatingSupply"`
	TotalSupply       json.RawMessage `json:"totalSupply"`
	MaxSupply         json.RawMessage `json:"maxSupply"`
	Price             json.RawMessage `json:"price"`
	InfiniteSupply    bool            `json:"infiniteSupply"`
}

// parseOptionalFloat decodes a JSON number or numeric string; false if null/missing/invalid.
func parseOptionalFloat(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

// usdStableQuotes stripped from trading pairs only after an exact cache miss.
// Longest first so USDT/FDUSD win over shorter USD tails.
// Bare "USD" is intentionally omitted — it would turn RLUSD→RL, BFUSD→BF, etc.
var usdStableQuotes = []string{"FDUSD", "USDT", "USDC", "BUSD", "TUSD", "DAI"}

// stripStableQuoteSuffix removes a trailing USD-stable quote for pair forms
// (BTCUSDT → BTC). Callers should try the full ticker as a cache key first.
func stripStableQuoteSuffix(upper string) string {
	for _, q := range usdStableQuotes {
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
