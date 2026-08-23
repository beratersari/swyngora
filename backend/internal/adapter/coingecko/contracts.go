package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const coinsPath = "/api/v3/coins/"

// LookupContracts resolves published token addresses via CoinGecko search + coin detail.
func (c *Client) LookupContracts(ctx context.Context, asset string) ([]domain.AssetContract, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: coingecko client not configured", domain.ErrUpstream)
	}
	key := domain.NormalizeAssetKey(asset)
	if key == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	if c.contractCache != nil {
		if hit, ok := c.contractCache.Get(key); ok {
			return cloneContracts(hit), nil
		}
	}
	id, ok := c.searchID(ctx, key)
	if !ok {
		return nil, fmt.Errorf("%w: no coingecko id for %q", domain.ErrNotFound, key)
	}
	plats, name, err := c.fetchPlatforms(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = name
	if c.contractCache != nil {
		c.contractCache.Set(key, plats)
	}
	return cloneContracts(plats), nil
}

func (c *Client) fetchPlatforms(ctx context.Context, id string) ([]domain.AssetContract, string, error) {
	if ctx.Err() != nil {
		return nil, "", ctx.Err()
	}
	u := c.baseURL + coinsPath + strings.ReplaceAll(id, "/", "") + "?localization=false&tickers=false&market_data=false&community_data=false&developer_data=false&sparkline=false"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", fmt.Errorf("%w: build coingecko coin request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: coingecko coin request: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, "", fmt.Errorf("%w: read coingecko coin: %v", domain.ErrUpstream, err)
	}
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("%w: coingecko coin status %d", domain.ErrUpstream, resp.StatusCode)
	}
	var parsed struct {
		Name      string            `json:"name"`
		Platforms map[string]string `json:"platforms"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("%w: decode coingecko coin: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.AssetContract, 0, len(parsed.Platforms))
	for chain, addr := range parsed.Platforms {
		addr = strings.TrimSpace(addr)
		chain = strings.TrimSpace(chain)
		if addr == "" || chain == "" {
			continue
		}
		out = append(out, domain.AssetContract{Chain: chain, Address: addr})
	}
	if len(out) == 0 {
		return nil, parsed.Name, fmt.Errorf("%w: no contracts for %q", domain.ErrNotFound, id)
	}
	return out, parsed.Name, nil
}

func cloneContracts(in []domain.AssetContract) []domain.AssetContract {
	if len(in) == 0 {
		return nil
	}
	out := make([]domain.AssetContract, len(in))
	copy(out, in)
	return out
}
