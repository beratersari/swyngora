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

// marketingSymbolListPath is a public Binance web API used by the marketing/price UI.
// Official Spot REST does not expose supply; this list includes circulating, total, and max.
const marketingSymbolListPath = "/bapi/composite/v1/public/marketing/symbol/list"

// productCatalogPath lists spot products with tags (used to exclude bStocks / commodities).
const productCatalogPath = "/bapi/asset/v2/public/asset-service/product/get-products"

// GetSupply returns a cached supply snapshot for an asset ticker (e.g. BTC).
// It does **not** call Binance on the request path — populate via Refresh.
//
// Lookup order: exact uppercased ticker first (so RLUSD/BFUSD are not mangled),
// then NormalizeAssetKey (BTCUSDT / BTC-USD / ETHTRY → BTC). Expired entries
// are served last-good with Stale=true until the next successful Refresh.
func (c *Client) GetSupply(ctx context.Context, asset string) (*domain.AssetSupply, error) {
	_ = ctx
	key := strings.ToUpper(strings.TrimSpace(asset))
	if key == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	if c.supply == nil {
		return nil, fmt.Errorf("%w: supply cache not configured", domain.ErrUpstream)
	}
	if hit, stale, ok := lookupSupply(c.supply, key); ok {
		return stampSupply(hit, stale), nil
	}
	return nil, fmt.Errorf("%w: supply for %q not in Binance supply snapshot cache", domain.ErrSupplyUnmapped, key)
}

func lookupSupply(store *cache.TTL[*domain.AssetSupply], key string) (*domain.AssetSupply, bool, bool) {
	if hit, ok := store.Get(key); ok {
		return hit, false, true
	}
	if stale, ok := store.GetStale(key); ok && stale != nil {
		return stale, true, true
	}
	base := domain.NormalizeAssetKey(key)
	if base != "" && base != key {
		if hit, ok := store.Get(base); ok {
			return hit, false, true
		}
		if stale, ok := store.GetStale(base); ok && stale != nil {
			return stale, true, true
		}
	}
	return nil, false, false
}

func stampSupply(hit *domain.AssetSupply, stale bool) *domain.AssetSupply {
	cp := cloneSupply(hit)
	cp.Stale = stale
	return cp
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
// On failure the previous snapshot is left intact (atomic replace only on success).
func (c *Client) Refresh(ctx context.Context) (int, error) {
	if c.supply == nil {
		return 0, fmt.Errorf("%w: supply cache not configured", domain.ErrUpstream)
	}
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	v, err, _ := c.supplySF.Do("refresh", func() (any, error) {
		// Detached context with deadline: shared work must not hang forever.
		fetchCtx, cancel := context.WithTimeout(context.Background(), multiCallTimeout)
		defer cancel()
		return c.fetchAndStoreSupply(fetchCtx)
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
	if !productEnvelopeOK(envelope.Success, envelope.Code) {
		return 0, fmt.Errorf("%w: binance marketing symbol list code=%s msg=%s success=%v",
			domain.ErrUpstream, envelope.Code, envelope.Message, envelope.Success)
	}
	if len(envelope.Data) == 0 {
		return 0, fmt.Errorf("%w: empty marketing symbol list", domain.ErrUpstream)
	}

	asOf := time.Now().UTC()
	// First pass: pick best supply row and best CMC-id row independently per base.
	// A USDT winner with no cmcUniqueId must not drop an id on a lesser quote.
	type rowScore struct {
		row   marketingSymbolRow
		score int
	}
	bestSupply := map[string]rowScore{}
	bestCatalog := map[string]rowScore{}
	bestSlug := map[string]rowScore{}
	for _, row := range envelope.Data {
		if hasNonCryptoTag(row.Tags) {
			continue
		}
		base := strings.ToUpper(strings.TrimSpace(row.Name))
		if base == "" {
			base = domain.NormalizeAssetKey(strings.ToUpper(strings.TrimSpace(row.Symbol)))
		}
		if base == "" {
			continue
		}
		circ, hasCirc := parseOptionalFloat(row.CirculatingSupply)
		total, hasTotal := parseOptionalFloat(row.TotalSupply)
		max, hasMax := parseOptionalFloat(row.MaxSupply)
		hasSupply := (hasCirc && circ > 0) || (hasTotal && total > 0) || (hasMax && max > 0)
		id, hasID := parseOptionalInt(row.CMCUniqueID)
		quoteScore := pairQuoteScore(row.Symbol)
		if hasSupply {
			score := quoteScore + completenessScore(hasCirc && circ > 0, hasTotal && total > 0, hasMax && max > 0)
			if prev, ok := bestSupply[base]; !ok || score > prev.score {
				bestSupply[base] = rowScore{row: row, score: score}
			}
		}
		if hasID && id > 0 {
			if prev, ok := bestCatalog[base]; !ok || quoteScore > prev.score {
				bestCatalog[base] = rowScore{row: row, score: quoteScore}
			}
		}
		if slug := strings.TrimSpace(row.Slug); slug != "" {
			if prev, ok := bestSlug[base]; !ok || quoteScore > prev.score {
				bestSlug[base] = rowScore{row: row, score: quoteScore}
			}
		}
	}

	next := make(map[string]*domain.AssetSupply, len(bestSupply))
	for base, rs := range bestSupply {
		row := rs.row
		circ, hasCirc := parseOptionalFloat(row.CirculatingSupply)
		total, hasTotal := parseOptionalFloat(row.TotalSupply)
		max, hasMax := parseOptionalFloat(row.MaxSupply)

		name := marketingRowName(row, base)

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
		next[base] = sup
	}

	catalog := make(map[string]*domain.AssetCatalogEntry, len(bestCatalog)+len(bestSlug))
	for base, rs := range bestCatalog {
		id, ok := parseOptionalInt(rs.row.CMCUniqueID)
		if !ok || id <= 0 {
			continue
		}
		name := marketingRowName(rs.row, base)
		if sup, has := next[base]; has && strings.TrimSpace(sup.Name) != "" {
			name = sup.Name
		}
		catalog[base] = &domain.AssetCatalogEntry{
			Asset: base,
			Name:  name,
			CMCID: id,
			Slug:  strings.TrimSpace(rs.row.Slug),
		}
	}
	// Slug-only rows: no cmcUniqueId, but CMC still serves detail?slug=.
	for base, rs := range bestSlug {
		if _, exists := catalog[base]; exists {
			continue
		}
		slug := strings.TrimSpace(rs.row.Slug)
		if slug == "" {
			continue
		}
		name := marketingRowName(rs.row, base)
		if sup, has := next[base]; has && strings.TrimSpace(sup.Name) != "" {
			name = sup.Name
		}
		catalog[base] = &domain.AssetCatalogEntry{
			Asset: base,
			Name:  name,
			Slug:  slug,
		}
	}
	if len(next) == 0 {
		return 0, fmt.Errorf("%w: no supply rows stored from marketing symbol list", domain.ErrUpstream)
	}

	// Atomic swap: only replace previous snapshot after a full successful build.
	// defaultTTL for supply is configured long; entries also never vanish on failed refresh.
	c.supply.ReplaceAll(next)
	if c.catalog != nil {
		c.catalog.ReplaceAll(catalog)
	}
	return len(next), nil
}

func marketingRowName(row marketingSymbolRow, fallback string) string {
	name := strings.TrimSpace(row.FullName)
	if name == "" {
		name = strings.TrimSpace(row.LocalFullName)
	}
	if name == "" {
		name = fallback
	}
	return name
}

// LookupAsset resolves a base ticker or pair to the CoinMarketCap id stored
// with the daily marketing snapshot. Cache-only — same lookup rules as GetSupply.
func (c *Client) LookupAsset(ctx context.Context, asset string) (*domain.AssetCatalogEntry, error) {
	_ = ctx
	key := strings.ToUpper(strings.TrimSpace(asset))
	if key == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	if c.catalog == nil {
		return nil, fmt.Errorf("%w: asset catalog not configured", domain.ErrUpstream)
	}
	if hit, ok := c.catalog.Get(key); ok {
		return cloneCatalog(hit), nil
	}
	if stale, ok := c.catalog.GetStale(key); ok && stale != nil {
		return cloneCatalog(stale), nil
	}
	base := domain.NormalizeAssetKey(key)
	if base != "" && base != key {
		if hit, ok := c.catalog.Get(base); ok {
			return cloneCatalog(hit), nil
		}
		if stale, ok := c.catalog.GetStale(base); ok && stale != nil {
			return cloneCatalog(stale), nil
		}
		key = base
	}
	return nil, fmt.Errorf("%w: catalog for %q not in Binance marketing snapshot", domain.ErrCatalogUnmapped, key)
}

func cloneCatalog(hit *domain.AssetCatalogEntry) *domain.AssetCatalogEntry {
	if hit == nil {
		return nil
	}
	cp := *hit
	return &cp
}

// pairQuoteScore ranks a marketing pair symbol by preferred USD-stable quote.
func pairQuoteScore(symbol string) int {
	u := strings.ToUpper(strings.TrimSpace(symbol))
	// Longest first.
	for _, q := range []struct {
		suffix string
		score  int
	}{
		{"FDUSD", 80},
		{"USDT", 100},
		{"USDC", 90},
		{"BUSD", 70},
		{"TUSD", 60},
		{"DAI", 50},
	} {
		if strings.HasSuffix(u, q.suffix) && len(u) > len(q.suffix) {
			return q.score
		}
	}
	return 0
}

func completenessScore(hasCirc, hasTotal, hasMax bool) int {
	n := 0
	if hasCirc {
		n += 3
	}
	if hasTotal {
		n += 2
	}
	if hasMax {
		n += 1
	}
	return n
}

// productEnvelopeOK requires a clear success signal from Binance bapi envelopes.
func productEnvelopeOK(success bool, code string) bool {
	if success {
		return true
	}
	return code == "000000"
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
		return nil, fmt.Errorf("%w: binance product path not found (%s)", domain.ErrNotFound, path)
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
	CMCUniqueID       json.RawMessage `json:"cmcUniqueId"`
	Slug              string          `json:"slug"`
}

// parseOptionalInt decodes a JSON number or numeric string as int64.
func parseOptionalInt(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, true
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int64(f), true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
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
