package okx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultBaseURL   = "https://www.okx.com"
	multiCallTimeout = 45 * time.Second
	maxBody          = 32 << 20
)

// Client is the public OKX v5 adapter for USDT-margined SWAP futures.
type Client struct {
	baseURL      string
	httpClient   *http.Client
	oiCache      *cache.TTL[*domain.OpenInterestSeries]
	oiSF         singleflight.Group
	fundingCache *cache.TTL[*domain.FundingSeries]
	fundingSF    singleflight.Group
	depthCache   *cache.TTL[*domain.RawOrderBook]
	depthSF      singleflight.Group

	instMu   sync.RWMutex
	ctVal    map[string]float64
	instOnce sync.Once
}

// Options configures the OKX client.
type Options struct {
	BaseURL           string
	HTTPClient        *http.Client
	OpenInterestCache *cache.TTL[*domain.OpenInterestSeries]
}

// NewClient constructs an OKX SWAP market client.
func NewClient(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	c := &Client{
		baseURL:    base,
		httpClient: hc,
		oiCache:    opts.OpenInterestCache,
		ctVal:      map[string]float64{},
	}
	if c.oiCache == nil {
		c.oiCache = cache.New[*domain.OpenInterestSeries](30 * time.Second)
	}
	if c.fundingCache == nil {
		c.fundingCache = cache.New[*domain.FundingSeries](30 * time.Second)
	}
	if c.depthCache == nil {
		c.depthCache = cache.New[*domain.RawOrderBook](5 * time.Second)
	}
	return c
}

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", domain.ErrUpstream, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: okx status %d", domain.ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: okx status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return body, nil
}

type okxEnvelope struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func decodeOKX(body []byte, dest any) error {
	var env okxEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("%w: okx decode: %v", domain.ErrUpstream, err)
	}
	if env.Code != "0" {
		return mapOKXError(env.Code, env.Msg)
	}
	if dest == nil || len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, dest); err != nil {
		return fmt.Errorf("%w: okx data: %v", domain.ErrUpstream, err)
	}
	return nil
}

func mapOKXError(code, msg string) error {
	lower := strings.ToLower(msg + " " + code)
	switch {
	case strings.Contains(lower, "too many") || code == "50011":
		return fmt.Errorf("%w: %s", domain.ErrRateLimited, msg)
	case strings.Contains(lower, "doesn't exist") || strings.Contains(lower, "not exist") ||
		strings.Contains(lower, "instrument") && strings.Contains(lower, "invalid") ||
		code == "51001":
		return fmt.Errorf("%w: %s", domain.ErrNotFound, msg)
	default:
		return fmt.Errorf("%w: okx %s: %s", domain.ErrUpstream, code, msg)
	}
}

func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func parseMillis(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil || ms <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(ms).UTC(), true
}
