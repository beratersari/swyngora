// Package routescan fetches EVM token holders from the public Routescan explorer API.
package routescan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL = "https://api.routescan.io"
	maxBody        = 2 << 20
)

// Client implements holder snapshots via Routescan's etherscan-compatible API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Options configures the public Routescan client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a Routescan holders client.
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

// FromContracts uses the first mapped EVM contract that has holders.
func (c *Client) FromContracts(ctx context.Context, asset string, contracts []domain.AssetContract) (*domain.AssetHolders, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: routescan not configured", domain.ErrUpstream)
	}
	asset = domain.NormalizeAssetKey(asset)
	var last error
	for _, con := range contracts {
		chain := domain.InferContractChain(con.Chain, con.Address)
		id := evmChainID(chain)
		if id == 0 || strings.TrimSpace(con.Address) == "" {
			continue
		}
		snap, err := c.fetchToken(ctx, asset, id, con.Address)
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
	return nil, fmt.Errorf("%w: routescan holders for %q", domain.ErrHoldersUnpublished, asset)
}

func (c *Client) fetchToken(ctx context.Context, asset string, chainID int, addr string) (*domain.AssetHolders, error) {
	countRaw, err := c.api(ctx, chainID, map[string]string{
		"module": "token", "action": "tokenholdercount", "contractaddress": addr,
	})
	if err != nil {
		return nil, err
	}
	var countWrap struct {
		Status string `json:"status"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(countRaw, &countWrap); err != nil || countWrap.Status != "1" {
		return nil, fmt.Errorf("%w: routescan holdercount", domain.ErrHoldersUnpublished)
	}
	count, err := strconv.ParseInt(strings.TrimSpace(countWrap.Result), 10, 64)
	if err != nil || count <= 0 {
		return nil, fmt.Errorf("%w: routescan holdercount empty", domain.ErrHoldersUnpublished)
	}

	infoRaw, _ := c.api(ctx, chainID, map[string]string{
		"module": "token", "action": "tokeninfo", "contractaddress": addr,
	})
	name, symbol, decimals, supply := parseTokenInfo(infoRaw)
	if name == "" {
		name = asset
	}

	listRaw, err := c.api(ctx, chainID, map[string]string{
		"module": "token", "action": "tokenholderlist", "contractaddress": addr,
		"page": "1", "offset": "20",
	})
	if err != nil {
		return &domain.AssetHolders{
			Asset: asset, Name: name, ProviderID: addr,
			HolderCount: count, AsOf: time.Now().UTC(), Source: "routescan",
		}, nil
	}
	var listWrap struct {
		Status string `json:"status"`
		Result []struct {
			Address  string `json:"TokenHolderAddress"`
			Quantity string `json:"TokenHolderQuantity"`
		} `json:"result"`
	}
	_ = json.Unmarshal(listRaw, &listWrap)
	var top []domain.AssetHolder
	var top10 float64
	for i, row := range listWrap.Result {
		addr := strings.TrimSpace(row.Address)
		if addr == "" {
			continue
		}
		bal := tokenAmount(row.Quantity, decimals)
		share := sharePct(row.Quantity, supply)
		top = append(top, domain.AssetHolder{Address: addr, Balance: bal, SharePct: share})
		if i < 10 {
			top10 += share
		}
	}
	out := &domain.AssetHolders{
		Asset:       asset,
		Name:        firstNonEmpty(name, symbol, asset),
		ProviderID:  addr,
		HolderCount: count,
		TopHolders:  domain.CapHolderList(top, domain.MaxHolderList),
		AsOf:        time.Now().UTC(),
		Source:      "routescan",
	}
	if top10 > 0 {
		out.TopTenSharePct = &top10
	}
	return out, nil
}

func (c *Client) api(ctx context.Context, chainID int, params map[string]string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	u, err := url.Parse(fmt.Sprintf("%s/v2/network/mainnet/evm/%d/etherscan/api", c.baseURL, chainID))
	if err != nil {
		return nil, fmt.Errorf("%w: routescan url: %v", domain.ErrUpstream, err)
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build routescan request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: routescan request: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: read routescan: %v", domain.ErrUpstream, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: routescan 429", domain.ErrRateLimited)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: routescan status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return body, nil
}

func parseTokenInfo(raw []byte) (name, symbol string, decimals int, supply string) {
	var wrap struct {
		Status string `json:"status"`
		Result []struct {
			TokenName   string `json:"tokenName"`
			Symbol      string `json:"symbol"`
			Divisor     string `json:"divisor"`
			TotalSupply string `json:"totalSupply"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &wrap) != nil || len(wrap.Result) == 0 {
		return "", "", 18, ""
	}
	row := wrap.Result[0]
	decimals, _ = strconv.Atoi(strings.TrimSpace(row.Divisor))
	if decimals <= 0 {
		decimals = 18
	}
	return strings.TrimSpace(row.TokenName), strings.TrimSpace(row.Symbol), decimals, strings.TrimSpace(row.TotalSupply)
}

func tokenAmount(qty string, decimals int) float64 {
	n, ok := new(big.Float).SetString(strings.TrimSpace(qty))
	if !ok {
		return 0
	}
	den := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	out, _ := new(big.Float).Quo(n, den).Float64()
	return out
}

func sharePct(qty, supply string) float64 {
	q, ok1 := new(big.Float).SetString(strings.TrimSpace(qty))
	s, ok2 := new(big.Float).SetString(strings.TrimSpace(supply))
	if !ok1 || !ok2 || s.Sign() <= 0 {
		return 0
	}
	pct, _ := new(big.Float).Quo(new(big.Float).Mul(q, big.NewFloat(100)), s).Float64()
	return pct
}

func evmChainID(chain string) int {
	switch domain.CanonicalChain(chain) {
	case "ethereum":
		return 1
	case "bsc":
		return 56
	case "base":
		return 8453
	case "arbitrum":
		return 42161
	case "optimism":
		return 10
	case "polygon":
		return 137
	case "avalanche":
		return 43114
	case "chiliz":
		return 88888
	default:
		return 0
	}
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
