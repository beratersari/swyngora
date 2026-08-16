package cmc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	detailPath     = "/data-api/v3/cryptocurrency/detail"
	defaultBaseURL = "https://api.coinmarketcap.com"
	maxBody        = 8 << 20
)

// Client implements domain.HoldersPort against CoinMarketCap’s public data-api.
type Client struct {
	baseURL    string
	httpClient *http.Client
	catalog    domain.AssetCatalogPort
	holders    *cache.TTL[*domain.AssetHolders]
	sf         singleflight.Group
}

// Options configures the CMC holders client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
	Catalog    domain.AssetCatalogPort
	Cache      *cache.TTL[*domain.AssetHolders]
}

// New constructs a holders client. Cache and Catalog are required.
func New(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    base,
		httpClient: hc,
		catalog:    opts.Catalog,
		holders:    opts.Cache,
	}
}

// GetHolders returns a holder snapshot for a base asset or pair (BTC / BTCUSDT).
func (c *Client) GetHolders(ctx context.Context, asset string) (*domain.AssetHolders, error) {
	if c == nil || c.catalog == nil {
		return nil, fmt.Errorf("%w: holders catalog not configured", domain.ErrUpstream)
	}
	if c.holders == nil {
		return nil, fmt.Errorf("%w: holders cache not configured", domain.ErrUpstream)
	}
	entry, err := c.catalog.LookupAsset(ctx, asset)
	if err != nil {
		return nil, err
	}
	key := entry.Asset
	if hit, ok := c.holders.Get(key); ok {
		return domain.CloneHolders(hit), nil
	}

	v, err, _ := c.sf.Do(key, func() (any, error) {
		if hit, ok := c.holders.Get(key); ok {
			return hit, nil
		}
		snap, fetchErr := c.fetchDetail(ctx, entry)
		if fetchErr != nil {
			if stale, ok := c.holders.GetStale(key); ok &&
				(errors.Is(fetchErr, domain.ErrRateLimited) || errors.Is(fetchErr, domain.ErrUpstream)) {
				return stale, nil
			}
			return nil, fetchErr
		}
		c.holders.Set(key, snap)
		return snap, nil
	})
	if err != nil {
		return nil, err
	}
	return domain.CloneHolders(v.(*domain.AssetHolders)), nil
}

func (c *Client) fetchDetail(ctx context.Context, entry *domain.AssetCatalogEntry) (*domain.AssetHolders, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	u := fmt.Sprintf("%s%s?id=%d", c.baseURL, detailPath, entry.CMCID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build cmc request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: cmc request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: read cmc body: %v", domain.ErrUpstream, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: cmc status %d", domain.ErrRateLimited, resp.StatusCode)
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("%w: holders for %q", domain.ErrNotFound, entry.Asset)
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: cmc status %d: %s", domain.ErrUpstream, resp.StatusCode, truncate(string(body), 200))
	}

	snap, err := parseDetail(body, entry)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

type detailEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type detailData struct {
	ID      json.RawMessage `json:"id"`
	Name    string          `json:"name"`
	Symbol  string          `json:"symbol"`
	Holders *holdersBlock   `json:"holders"`
}

type holdersBlock struct {
	HolderCount           json.RawMessage `json:"holderCount"`
	DailyActive           json.RawMessage `json:"dailyActive"`
	TopTenHolderRatio     json.RawMessage `json:"topTenHolderRatio"`
	TopTwentyHolderRatio  json.RawMessage `json:"topTwentyHolderRatio"`
	TopFiftyHolderRatio   json.RawMessage `json:"topFiftyHolderRatio"`
	TopHundredHolderRatio json.RawMessage `json:"topHundredHolderRatio"`
	HolderList            []holderRow     `json:"holderList"`
}

type holderRow struct {
	Address string          `json:"address"`
	Balance json.RawMessage `json:"balance"`
	Share   json.RawMessage `json:"share"`
}

func parseDetail(body []byte, entry *domain.AssetCatalogEntry) (*domain.AssetHolders, error) {
	var env detailEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("%w: decode cmc detail: %v", domain.ErrUpstream, err)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil, fmt.Errorf("%w: holders for %q", domain.ErrNotFound, entry.Asset)
	}
	var data detailData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("%w: decode cmc detail data: %v", domain.ErrUpstream, err)
	}
	if data.Holders == nil {
		return nil, fmt.Errorf("%w: holders for %q not published", domain.ErrNotFound, entry.Asset)
	}

	count, hasCount := parseOptionalInt(data.Holders.HolderCount)
	list := make([]domain.AssetHolder, 0, len(data.Holders.HolderList))
	for _, row := range data.Holders.HolderList {
		addr := strings.TrimSpace(row.Address)
		if addr == "" {
			continue
		}
		bal, _ := parseOptionalFloat(row.Balance)
		share, _ := parseOptionalFloat(row.Share)
		list = append(list, domain.AssetHolder{Address: addr, Balance: bal, SharePct: share})
	}
	list = domain.CapHolderList(list, domain.MaxHolderList)
	if (!hasCount || count <= 0) && len(list) == 0 {
		return nil, fmt.Errorf("%w: holders for %q empty", domain.ErrNotFound, entry.Asset)
	}

	name := strings.TrimSpace(data.Name)
	if name == "" {
		name = entry.Name
	}
	if name == "" {
		name = entry.Asset
	}

	out := &domain.AssetHolders{
		Asset:       entry.Asset,
		Name:        name,
		ProviderID:  strconv.FormatInt(entry.CMCID, 10),
		HolderCount: count,
		TopHolders:  list,
		AsOf:        time.Now().UTC(),
		Source:      "coinmarketcap",
	}
	if active, ok := parseOptionalInt(data.Holders.DailyActive); ok && active >= 0 {
		out.DailyActive = &active
	}
	if v, ok := parseOptionalFloat(data.Holders.TopTenHolderRatio); ok {
		out.TopTenSharePct = &v
	}
	if v, ok := parseOptionalFloat(data.Holders.TopTwentyHolderRatio); ok {
		out.TopTwentySharePct = &v
	}
	if v, ok := parseOptionalFloat(data.Holders.TopFiftyHolderRatio); ok {
		out.TopFiftySharePct = &v
	}
	if v, ok := parseOptionalFloat(data.Holders.TopHundredHolderRatio); ok {
		out.TopHundredSharePct = &v
	}
	return out, nil
}

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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
