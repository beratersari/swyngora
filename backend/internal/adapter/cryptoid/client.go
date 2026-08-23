// Package cryptoid fetches UTXO holder snapshots from the public Chainz CryptoID API.
package cryptoid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL = "https://chainz.cryptoid.info"
	maxBody        = 2 << 20
)

// Client implements domain.HoldersPort for coins CryptoID indexes (PIVX, DASH, …).
type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      *cache.TTL[*domain.AssetHolders]
	missing    *cache.TTL[struct{}]
	sf         singleflight.Group
}

// Options configures the CryptoID client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
	Cache      *cache.TTL[*domain.AssetHolders]
}

// New constructs a CryptoID holders client.
func New(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	cch := opts.Cache
	if cch == nil {
		cch = cache.New[*domain.AssetHolders](time.Hour)
	}
	return &Client{
		baseURL:    base,
		httpClient: hc,
		cache:      cch,
		missing:    cache.New[struct{}](time.Hour),
	}
}

// GetHolders returns nonzero-address count and top wallets for a UTXO ticker.
func (c *Client) GetHolders(ctx context.Context, asset string) (*domain.AssetHolders, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: cryptoid not configured", domain.ErrUpstream)
	}
	key := domain.NormalizeAssetKey(asset)
	if key == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	coin := strings.ToLower(key)
	if !isCryptoIDTicker(coin) {
		return nil, fmt.Errorf("%w: cryptoid ticker %q", domain.ErrNotFound, key)
	}
	if hit, ok := c.cache.Get(key); ok {
		return domain.CloneHolders(hit), nil
	}
	if _, ok := c.missing.Get(key); ok {
		return nil, fmt.Errorf("%w: cryptoid has no %q", domain.ErrNotFound, key)
	}

	v, err, _ := c.sf.Do(key, func() (any, error) {
		if hit, ok := c.cache.Get(key); ok {
			return hit, nil
		}
		if _, ok := c.missing.Get(key); ok {
			return nil, fmt.Errorf("%w: cryptoid has no %q", domain.ErrNotFound, key)
		}
		snap, fetchErr := c.fetch(ctx, key, coin)
		if fetchErr != nil {
			if stale, ok := c.cache.GetStale(key); ok && stale != nil {
				cp := domain.CloneHolders(stale)
				cp.Stale = true
				return cp, nil
			}
			if isCryptoIDMiss(fetchErr) {
				c.missing.Set(key, struct{}{})
			}
			return nil, fetchErr
		}
		c.missing.Delete(key)
		c.cache.Set(key, snap)
		return snap, nil
	})
	if err != nil {
		return nil, err
	}
	return domain.CloneHolders(v.(*domain.AssetHolders)), nil
}

func (c *Client) fetch(ctx context.Context, asset, coin string) (*domain.AssetHolders, error) {
	addrBody, err := c.get(ctx, fmt.Sprintf("%s/%s/api.dws?q=addresses", c.baseURL, coin))
	if err != nil {
		return nil, err
	}
	var addr struct {
		Known   int64 `json:"known"`
		Nonzero int64 `json:"nonzero"`
	}
	if err := json.Unmarshal(addrBody, &addr); err != nil {
		return nil, fmt.Errorf("%w: decode cryptoid addresses: %v", domain.ErrUpstream, err)
	}
	if addr.Nonzero <= 0 && addr.Known <= 0 {
		return nil, fmt.Errorf("%w: cryptoid empty %q", domain.ErrNotFound, asset)
	}

	count := addr.Nonzero
	if count <= 0 {
		count = addr.Known
	}

	out := &domain.AssetHolders{
		Asset:       asset,
		Name:        asset,
		HolderCount: count,
		AsOf:        time.Now().UTC(),
		Source:      "cryptoid",
	}

	richBody, err := c.get(ctx, fmt.Sprintf("%s/%s/api.dws?q=rich", c.baseURL, coin))
	if err != nil {
		return out, nil
	}
	var rich struct {
		Total    float64 `json:"total"`
		Rich1000 []struct {
			Amount float64 `json:"amount"`
			Addr   string  `json:"addr"`
		} `json:"rich1000"`
	}
	if json.Unmarshal(richBody, &rich) != nil {
		return out, nil
	}
	list := make([]domain.AssetHolder, 0, domain.MaxHolderList)
	for _, row := range rich.Rich1000 {
		a := strings.TrimSpace(row.Addr)
		if a == "" || strings.EqualFold(a, "Anonymous") || row.Amount <= 0 {
			continue
		}
		h := domain.AssetHolder{Address: a, Balance: row.Amount}
		if rich.Total > 0 {
			h.SharePct = row.Amount / rich.Total * 100
		}
		list = append(list, h)
		if len(list) >= domain.MaxHolderList {
			break
		}
	}
	out.TopHolders = list
	if len(list) >= 10 {
		top := 0.0
		for i := 0; i < 10 && i < len(list); i++ {
			top += list[i].SharePct
		}
		out.TopTenSharePct = &top
	}
	return out, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build cryptoid request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: cryptoid request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: read cryptoid body: %v", domain.ErrUpstream, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: cryptoid status %d", domain.ErrRateLimited, resp.StatusCode)
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest:
		return nil, fmt.Errorf("%w: cryptoid status %d", domain.ErrNotFound, resp.StatusCode)
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: cryptoid status %d: %s", domain.ErrUpstream, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if len(body) > 0 && body[0] == '<' {
		return nil, fmt.Errorf("%w: cryptoid html", domain.ErrNotFound)
	}
	return body, nil
}

func isCryptoIDTicker(coin string) bool {
	if len(coin) < 2 || len(coin) > 16 {
		return false
	}
	for _, r := range coin {
		if r < 'a' || r > 'z' {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func isCryptoIDMiss(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "cryptoid has no") ||
		strings.Contains(err.Error(), "cryptoid status 404") ||
		strings.Contains(err.Error(), "cryptoid status 400") ||
		strings.Contains(err.Error(), "cryptoid empty") ||
		strings.Contains(err.Error(), "cryptoid html"))
}
