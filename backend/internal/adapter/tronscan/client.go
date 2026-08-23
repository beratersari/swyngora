// Package tronscan fetches TRC-20 holder counts from the public Tronscan API.
package tronscan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL = "https://apilist.tronscanapi.com"
	maxBody        = 2 << 20
)

// Client implements holder snapshots via Tronscan's public token_trc20 API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Options configures the public Tronscan client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a Tronscan holders client (no API key).
func New(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: base, httpClient: hc}
}

// FromContracts uses the first TRC-20 contract that has a holder count.
func (c *Client) FromContracts(ctx context.Context, asset string, contracts []domain.AssetContract) (*domain.AssetHolders, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: tronscan not configured", domain.ErrUpstream)
	}
	asset = domain.NormalizeAssetKey(asset)
	var last error
	for _, con := range contracts {
		chain := domain.InferContractChain(con.Chain, con.Address)
		if chain != "tron" || strings.TrimSpace(con.Address) == "" {
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
	return nil, fmt.Errorf("%w: tronscan holders for %q", domain.ErrHoldersUnpublished, asset)
}

func (c *Client) fetchToken(ctx context.Context, asset, addr string) (*domain.AssetHolders, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	u, err := url.Parse(c.baseURL + "/api/token_trc20")
	if err != nil {
		return nil, fmt.Errorf("%w: tronscan url: %v", domain.ErrUpstream, err)
	}
	q := u.Query()
	q.Set("contract", addr)
	q.Set("showAll", "1")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build tronscan request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: tronscan request: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: read tronscan: %v", domain.ErrUpstream, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: tronscan 429", domain.ErrRateLimited)
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("%w: tronscan %s", domain.ErrNotFound, addr)
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: tronscan status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return parseToken(body, asset, addr)
}

func parseToken(body []byte, asset, addr string) (*domain.AssetHolders, error) {
	var env struct {
		Tokens []struct {
			Name         string `json:"name"`
			Symbol       string `json:"symbol"`
			HoldersCount int64  `json:"holders_count"`
			Contract     string `json:"contract_address"`
		} `json:"trc20_tokens"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("%w: decode tronscan: %v", domain.ErrUpstream, err)
	}
	if len(env.Tokens) == 0 || env.Tokens[0].HoldersCount <= 0 {
		return nil, fmt.Errorf("%w: tronscan holders empty", domain.ErrHoldersUnpublished)
	}
	row := env.Tokens[0]
	name := strings.TrimSpace(row.Name)
	if name == "" {
		name = firstNonEmpty(row.Symbol, asset)
	}
	provider := strings.TrimSpace(row.Contract)
	if provider == "" {
		provider = addr
	}
	return &domain.AssetHolders{
		Asset:       asset,
		Name:        name,
		ProviderID:  provider,
		HolderCount: row.HoldersCount,
		AsOf:        time.Now().UTC(),
		Source:      "tronscan",
	}, nil
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
