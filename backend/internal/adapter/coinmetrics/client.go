// Package coinmetrics fetches public address-count metrics (community API, no key).
package coinmetrics

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

const (
	defaultBaseURL = "https://community-api.coinmetrics.io"
	metricsPath    = "/v4/timeseries/asset-metrics"
	maxBody        = 1 << 20
)

// Client loads AdrBalCnt (addresses with a balance) for a ticker.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Options configures the community client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a CoinMetrics holders client.
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

// GetHolders returns an address-count snapshot when the community API has AdrBalCnt.
func (c *Client) GetHolders(ctx context.Context, asset string) (*domain.AssetHolders, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: coinmetrics not configured", domain.ErrUpstream)
	}
	key := domain.NormalizeAssetKey(asset)
	if key == "" {
		return nil, fmt.Errorf("%w: asset is required", domain.ErrInvalidArgument)
	}
	id := coinmetricsAsset(key)
	if id == "" {
		return nil, fmt.Errorf("%w: coinmetrics has no id for %q", domain.ErrNotFound, key)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	u, err := url.Parse(c.baseURL + metricsPath)
	if err != nil {
		return nil, fmt.Errorf("%w: coinmetrics url: %v", domain.ErrUpstream, err)
	}
	q := u.Query()
	q.Set("assets", id)
	q.Set("metrics", "AdrBalCnt")
	q.Set("frequency", "1d")
	q.Set("paging_from", "end")
	q.Set("page_size", "1")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build coinmetrics request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: coinmetrics request: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: read coinmetrics: %v", domain.ErrUpstream, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: coinmetrics 429", domain.ErrRateLimited)
	case resp.StatusCode == http.StatusBadRequest, resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusNotFound:
		// Community API 400s unknown tickers ("asset not found") the same as a miss.
		return nil, fmt.Errorf("%w: coinmetrics no AdrBalCnt for %q", domain.ErrHoldersUnpublished, key)
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: coinmetrics status %d", domain.ErrUpstream, resp.StatusCode)
	}
	var parsed struct {
		Data []struct {
			Asset     string `json:"asset"`
			Time      string `json:"time"`
			AdrBalCnt string `json:"AdrBalCnt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: decode coinmetrics: %v", domain.ErrUpstream, err)
	}
	if len(parsed.Data) == 0 || strings.TrimSpace(parsed.Data[0].AdrBalCnt) == "" {
		return nil, fmt.Errorf("%w: coinmetrics empty AdrBalCnt for %q", domain.ErrHoldersUnpublished, key)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(parsed.Data[0].AdrBalCnt), 10, 64)
	if err != nil || n <= 0 {
		return nil, fmt.Errorf("%w: coinmetrics bad AdrBalCnt %q", domain.ErrHoldersUnpublished, parsed.Data[0].AdrBalCnt)
	}
	asOf := time.Now().UTC()
	if t, err := time.Parse(time.RFC3339Nano, parsed.Data[0].Time); err == nil {
		asOf = t.UTC()
	}
	return &domain.AssetHolders{
		Asset:       key,
		Name:        key,
		ProviderID:  id,
		HolderCount: n,
		AsOf:        asOf,
		Source:      "coinmetrics",
	}, nil
}

func coinmetricsAsset(ticker string) string {
	s := strings.ToLower(strings.TrimSpace(ticker))
	switch s {
	case "pol", "matic":
		return "matic"
	default:
		return s
	}
}
