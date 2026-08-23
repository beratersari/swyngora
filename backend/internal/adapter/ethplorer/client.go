// Package ethplorer fetches ERC-20 holder snapshots from the public freekey API.
package ethplorer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL = "https://api.ethplorer.io"
	defaultAPIKey  = "freekey"
	maxBody        = 2 << 20
)

// Client implements Ethereum token holders via Ethplorer's public API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Options configures the Ethplorer client.
type Options struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// New returns an Ethplorer holders client. APIKey defaults to the public freekey.
func New(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	key := strings.TrimSpace(opts.APIKey)
	if key == "" {
		key = defaultAPIKey
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: base, apiKey: key, httpClient: hc}
}

// FromContracts uses the first Ethereum contract that has a holder count.
func (c *Client) FromContracts(ctx context.Context, asset string, contracts []domain.AssetContract) (*domain.AssetHolders, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: ethplorer not configured", domain.ErrUpstream)
	}
	asset = domain.NormalizeAssetKey(asset)
	var last error
	for _, con := range contracts {
		chain := domain.InferContractChain(con.Chain, con.Address)
		if !isETH(chain) || strings.TrimSpace(con.Address) == "" {
			continue
		}
		snap, err := c.fetchToken(ctx, asset, con.Address)
		if err != nil {
			last = err
			continue
		}
		if domain.HoldersUseful(snap) {
			return snap, nil
		}
	}
	if last != nil {
		return nil, last
	}
	return nil, fmt.Errorf("%w: ethplorer holders for %q", domain.ErrHoldersUnpublished, asset)
}

func (c *Client) fetchToken(ctx context.Context, asset, addr string) (*domain.AssetHolders, error) {
	info, err := c.getJSON(ctx, "/getTokenInfo/"+url.PathEscape(addr), nil)
	if err != nil {
		return nil, err
	}
	var meta struct {
		Name         string  `json:"name"`
		Symbol       string  `json:"symbol"`
		Decimals     jsonInt `json:"decimals"`
		HoldersCount int64   `json:"holdersCount"`
		Error        any     `json:"error"`
	}
	if err := json.Unmarshal(info, &meta); err != nil {
		return nil, fmt.Errorf("%w: decode ethplorer info: %v", domain.ErrUpstream, err)
	}
	if meta.Error != nil || meta.HoldersCount <= 0 {
		return nil, fmt.Errorf("%w: ethplorer holders empty", domain.ErrHoldersUnpublished)
	}
	name := strings.TrimSpace(meta.Name)
	if name == "" {
		name = asset
	}
	out := &domain.AssetHolders{
		Asset:       asset,
		Name:        name,
		ProviderID:  addr,
		HolderCount: meta.HoldersCount,
		AsOf:        time.Now().UTC(),
		Source:      "ethplorer",
	}
	top, err := c.getJSON(ctx, "/getTopTokenHolders/"+url.PathEscape(addr), map[string]string{"limit": "20"})
	if err == nil {
		var parsed struct {
			Holders []struct {
				Address    string  `json:"address"`
				Balance    float64 `json:"balance"`
				Share      float64 `json:"share"`
				RawBalance string  `json:"rawBalance"`
			} `json:"holders"`
		}
		if json.Unmarshal(top, &parsed) == nil {
			dec := int(meta.Decimals)
			if dec < 0 || dec > 36 {
				dec = 0
			}
			div := math.Pow10(dec)
			var top10 float64
			for i, row := range parsed.Holders {
				addr := strings.TrimSpace(row.Address)
				if addr == "" {
					continue
				}
				bal := row.Balance
				if div > 1 {
					bal = row.Balance / div
				}
				out.TopHolders = append(out.TopHolders, domain.AssetHolder{
					Address:  addr,
					Balance:  bal,
					SharePct: row.Share,
				})
				if i < 10 {
					top10 += row.Share
				}
			}
			out.TopHolders = domain.CapHolderList(out.TopHolders, domain.MaxHolderList)
			if top10 > 0 {
				out.TopTenSharePct = &top10
			}
		}
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, extra map[string]string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("%w: ethplorer url: %v", domain.ErrUpstream, err)
	}
	q := u.Query()
	q.Set("apiKey", c.apiKey)
	for k, v := range extra {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build ethplorer request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: ethplorer request: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: read ethplorer: %v", domain.ErrUpstream, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: ethplorer 429", domain.ErrRateLimited)
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: ethplorer status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return body, nil
}

type jsonInt int

func (j *jsonInt) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*j = jsonInt(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		var parsed int
		_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &parsed)
		if err != nil {
			return nil
		}
		*j = jsonInt(parsed)
	}
	return nil
}

func isETH(chain string) bool {
	s := strings.ToLower(strings.TrimSpace(chain))
	return s == "eth" || s == "ethereum" || s == "erc20"
}
