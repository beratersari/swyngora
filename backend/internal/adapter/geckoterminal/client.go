// Package geckoterminal fetches public token holder counts (no API key).
package geckoterminal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL = "https://api.geckoterminal.com"
	infoPathTpl    = "/api/v2/networks/%s/tokens/%s/info"
	maxBody        = 2 << 20
)

// Client implements holder snapshots from GeckoTerminal token info.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Options configures the public GeckoTerminal client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a GeckoTerminal holders client.
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

// FromContracts walks published token addresses until one has a holder count.
func (c *Client) FromContracts(ctx context.Context, asset string, contracts []domain.AssetContract) (*domain.AssetHolders, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: geckoterminal not configured", domain.ErrUpstream)
	}
	asset = domain.NormalizeAssetKey(asset)
	var last error
	for _, con := range preferContracts(contracts) {
		net := geckoNetwork(domain.InferContractChain(con.Chain, con.Address))
		if net == "" || strings.TrimSpace(con.Address) == "" {
			continue
		}
		snap, err := c.fetchInfo(ctx, asset, net, con)
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
	return nil, fmt.Errorf("%w: geckoterminal holders for %q", domain.ErrHoldersUnpublished, asset)
}

func (c *Client) fetchInfo(ctx context.Context, asset, network string, con domain.AssetContract) (*domain.AssetHolders, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	addr := strings.TrimSpace(con.Address)
	u := c.baseURL + fmt.Sprintf(infoPathTpl, network, addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build geckoterminal request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: geckoterminal request: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: read geckoterminal: %v", domain.ErrUpstream, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: geckoterminal 429", domain.ErrRateLimited)
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("%w: geckoterminal %s %s", domain.ErrNotFound, network, addr)
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: geckoterminal status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return parseTokenInfo(body, asset, addr)
}

func parseTokenInfo(body []byte, asset, addr string) (*domain.AssetHolders, error) {
	var env struct {
		Data struct {
			Attributes struct {
				Name    string `json:"name"`
				Symbol  string `json:"symbol"`
				Holders *struct {
					Count        json.RawMessage `json:"count"`
					Distribution struct {
						Top10 string `json:"top_10"`
					} `json:"distribution_percentage"`
				} `json:"holders"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("%w: decode geckoterminal: %v", domain.ErrUpstream, err)
	}
	h := env.Data.Attributes.Holders
	if h == nil {
		return nil, fmt.Errorf("%w: geckoterminal holders missing", domain.ErrHoldersUnpublished)
	}
	count, ok := parseJSONInt(h.Count)
	if !ok || count <= 0 {
		return nil, fmt.Errorf("%w: geckoterminal holders empty", domain.ErrHoldersUnpublished)
	}
	name := strings.TrimSpace(env.Data.Attributes.Name)
	if name == "" {
		name = asset
	}
	out := &domain.AssetHolders{
		Asset:       asset,
		Name:        name,
		ProviderID:  addr,
		HolderCount: count,
		AsOf:        time.Now().UTC(),
		Source:      "geckoterminal",
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(h.Distribution.Top10), 64); err == nil {
		out.TopTenSharePct = &v
	}
	return out, nil
}

func parseJSONInt(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		return n, true
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return int64(f), true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		return n, err == nil
	}
	return 0, false
}

func geckoNetwork(chain string) string {
	switch domain.CanonicalChain(chain) {
	case "ethereum":
		return "eth"
	case "bsc":
		return "bsc"
	case "solana":
		return "solana"
	case "base":
		return "base"
	case "arbitrum":
		return "arbitrum"
	case "optimism":
		return "optimism"
	case "polygon":
		return "polygon_pos"
	case "avalanche":
		return "avax"
	case "chiliz":
		return "chiliz-chain"
	case "scroll":
		return "scroll"
	case "zksync":
		return "zksync"
	case "kaia":
		return "kaia"
	case "manta":
		return "manta-pacific"
	case "sonic":
		return "sonic"
	case "ronin":
		return "ronin"
	case "celo":
		return "celo"
	case "tron":
		return "tron"
	default:
		s := strings.ToLower(strings.TrimSpace(chain))
		s = strings.ReplaceAll(s, " ", "-")
		return s
	}
}

func preferContracts(in []domain.AssetContract) []domain.AssetContract {
	rank := func(chain string) int {
		switch geckoNetwork(chain) {
		case "eth":
			return 0
		case "solana":
			return 1
		case "bsc":
			return 2
		case "base":
			return 3
		default:
			return 9
		}
	}
	out := append([]domain.AssetContract(nil), in...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if rank(out[j].Chain) < rank(out[i].Chain) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
