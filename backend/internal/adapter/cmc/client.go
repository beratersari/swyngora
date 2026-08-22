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
	baseURL     string
	httpClient  *http.Client
	catalog     domain.AssetCatalogPort
	holders     *cache.TTL[*domain.AssetHolders]
	unpublished *cache.TTL[struct{}]
	sf          singleflight.Group
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
		baseURL:     base,
		httpClient:  hc,
		catalog:     opts.Catalog,
		holders:     opts.Cache,
		unpublished: cache.New[struct{}](time.Hour),
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
	if c.unpublished != nil {
		if _, ok := c.unpublished.Get(key); ok {
			return nil, fmt.Errorf("%w: holders for %q not published", domain.ErrHoldersUnpublished, key)
		}
	}

	v, err, _ := c.sf.Do(key, func() (any, error) {
		if hit, ok := c.holders.Get(key); ok {
			return hit, nil
		}
		if c.unpublished != nil {
			if _, ok := c.unpublished.Get(key); ok {
				return nil, fmt.Errorf("%w: holders for %q not published", domain.ErrHoldersUnpublished, key)
			}
		}
		snap, fetchErr := c.fetchDetail(ctx, entry)
		if fetchErr != nil {
			if stale, ok := c.holders.GetStale(key); ok && stale != nil && isHoldersLastGood(fetchErr) {
				cp := domain.CloneHolders(stale)
				cp.Stale = true
				return cp, nil
			}
			if isHoldersUnpublished(fetchErr) && c.unpublished != nil {
				c.unpublished.Set(key, struct{}{})
			}
			return nil, fetchErr
		}
		if c.unpublished != nil {
			c.unpublished.Delete(key)
		}
		c.holders.Set(key, snap)
		return snap, nil
	})
	if err != nil {
		return nil, err
	}
	return domain.CloneHolders(v.(*domain.AssetHolders)), nil
}

// GetAssetProfile returns logo (always from the public CMC static CDN when an
// id exists) plus listing date and contracts when the detail payload has them.
func (c *Client) GetAssetProfile(ctx context.Context, asset string) (*domain.AssetProfile, error) {
	if c == nil || c.catalog == nil {
		return nil, fmt.Errorf("%w: holders catalog not configured", domain.ErrUpstream)
	}
	entry, err := c.catalog.LookupAsset(ctx, asset)
	if err != nil {
		return nil, err
	}
	out := profileFromCatalog(entry)
	if c.holders != nil {
		if hit, ok := c.holders.Get(entry.Asset); ok && hit != nil {
			mergeHoldersName(out, hit)
		}
	}
	body, fetchErr := c.fetchDetailBody(ctx, entry)
	if fetchErr != nil {
		return out, nil
	}
	applyProfileJSON(out, body)
	return out, nil
}

func profileFromCatalog(entry *domain.AssetCatalogEntry) *domain.AssetProfile {
	if entry == nil {
		return &domain.AssetProfile{Source: "coinmarketcap"}
	}
	return &domain.AssetProfile{
		Asset:      entry.Asset,
		Name:       entry.Name,
		Slug:       entry.Slug,
		ProviderID: strconv.FormatInt(entry.CMCID, 10),
		LogoURL:    domain.CMCLogoURL(entry.CMCID),
		AsOf:       time.Now().UTC(),
		Source:     "coinmarketcap",
	}
}

func mergeHoldersName(out *domain.AssetProfile, hit *domain.AssetHolders) {
	if out == nil || hit == nil {
		return
	}
	if strings.TrimSpace(hit.Name) != "" {
		out.Name = hit.Name
	}
}

func isHoldersLastGood(err error) bool {
	return errors.Is(err, domain.ErrRateLimited) ||
		errors.Is(err, domain.ErrUpstream) ||
		errors.Is(err, domain.ErrHoldersUnpublished) ||
		errors.Is(err, domain.ErrNotFound)
}

func isHoldersUnpublished(err error) bool {
	return errors.Is(err, domain.ErrHoldersUnpublished) || errors.Is(err, domain.ErrNotFound)
}

func (c *Client) fetchDetail(ctx context.Context, entry *domain.AssetCatalogEntry) (*domain.AssetHolders, error) {
	body, err := c.fetchDetailBody(ctx, entry)
	if err != nil {
		return nil, err
	}
	return parseDetail(body, entry)
}

func (c *Client) fetchDetailBody(ctx context.Context, entry *domain.AssetCatalogEntry) ([]byte, error) {
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
		return nil, fmt.Errorf("%w: holders for %q", domain.ErrHoldersUnpublished, entry.Asset)
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: cmc status %d: %s", domain.ErrUpstream, resp.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

func applyProfileJSON(out *domain.AssetProfile, body []byte) {
	if out == nil || len(body) == 0 {
		return
	}
	var env detailEnvelope
	if err := json.Unmarshal(body, &env); err != nil || len(env.Data) == 0 {
		return
	}
	var extra struct {
		Name         string          `json:"name"`
		DateAdded    string          `json:"dateAdded"`
		DateLaunched string          `json:"dateLaunched"`
		LaunchDate   string          `json:"launchDate"`
		Statistics   json.RawMessage `json:"statistics"`
		ContractAddr json.RawMessage `json:"contractAddress"`
		Platforms    json.RawMessage `json:"platforms"`
	}
	if err := json.Unmarshal(env.Data, &extra); err != nil {
		return
	}
	if strings.TrimSpace(extra.Name) != "" {
		out.Name = extra.Name
	}
	for _, raw := range []string{extra.DateLaunched, extra.LaunchDate, extra.DateAdded} {
		if t, ok := parseCMCTime(raw); ok {
			out.ListingDate = &t
			break
		}
	}
	if out.ListingDate == nil && len(extra.Statistics) > 0 {
		var stats struct {
			DateAdded    string `json:"dateAdded"`
			DateLaunched string `json:"dateLaunched"`
		}
		if json.Unmarshal(extra.Statistics, &stats) == nil {
			for _, raw := range []string{stats.DateLaunched, stats.DateAdded} {
				if t, ok := parseCMCTime(raw); ok {
					out.ListingDate = &t
					break
				}
			}
		}
	}
	out.Contracts = parseCMCContracts(extra.ContractAddr)
	if len(out.Contracts) == 0 {
		out.Contracts = parseCMCContracts(extra.Platforms)
	}
	out.AsOf = time.Now().UTC()
}

func parseCMCTime(raw string) (time.Time, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func parseCMCContracts(raw json.RawMessage) []domain.AssetContract {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var rows []struct {
		ContractAddress string `json:"contractAddress"`
		Address         string `json:"address"`
		Platform        struct {
			Name string `json:"name"`
			Coin struct {
				Symbol string `json:"symbol"`
			} `json:"coin"`
		} `json:"platform"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil
	}
	out := make([]domain.AssetContract, 0, len(rows))
	for _, row := range rows {
		addr := strings.TrimSpace(row.ContractAddress)
		if addr == "" {
			addr = strings.TrimSpace(row.Address)
		}
		if addr == "" {
			continue
		}
		chain := strings.TrimSpace(row.Platform.Name)
		if chain == "" {
			chain = strings.TrimSpace(row.Platform.Coin.Symbol)
		}
		if chain == "" {
			chain = strings.TrimSpace(row.Name)
		}
		out = append(out, domain.AssetContract{Chain: chain, Address: addr})
	}
	return out
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
		return nil, fmt.Errorf("%w: holders for %q", domain.ErrHoldersUnpublished, entry.Asset)
	}
	var data detailData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("%w: decode cmc detail data: %v", domain.ErrUpstream, err)
	}
	if data.Holders == nil {
		return nil, fmt.Errorf("%w: holders for %q not published", domain.ErrHoldersUnpublished, entry.Asset)
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
		return nil, fmt.Errorf("%w: holders for %q empty", domain.ErrHoldersUnpublished, entry.Asset)
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
