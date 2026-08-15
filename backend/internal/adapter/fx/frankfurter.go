package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const defaultFrankfurterURL = "https://api.frankfurter.app/latest?from=USD"

// Client fetches ECB spot FX from the free Frankfurter API (no key).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New constructs a Frankfurter client.
func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{baseURL: defaultFrankfurterURL, httpClient: httpClient}
}

// WithBaseURL overrides the API URL (tests).
func (c *Client) WithBaseURL(u string) *Client {
	if c != nil && strings.TrimSpace(u) != "" {
		c.baseURL = strings.TrimSpace(u)
	}
	return c
}

type frankfurterLatest struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// LatestUSD returns units of each currency per 1 USD.
func (c *Client) LatestUSD(ctx context.Context) (map[string]float64, time.Time, error) {
	if c == nil || c.httpClient == nil {
		return nil, time.Time{}, fmt.Errorf("%w: fx client not configured", domain.ErrUpstream)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("%w: fx request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("%w: fx fetch: %v", domain.ErrUpstream, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("%w: fx read: %v", domain.ErrUpstream, err)
	}
	if res.StatusCode >= 400 {
		return nil, time.Time{}, fmt.Errorf("%w: fx status %d", domain.ErrUpstream, res.StatusCode)
	}
	var parsed frankfurterLatest
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, time.Time{}, fmt.Errorf("%w: fx json: %v", domain.ErrUpstream, err)
	}
	if len(parsed.Rates) == 0 {
		return nil, time.Time{}, fmt.Errorf("%w: fx empty rates", domain.ErrUpstream)
	}
	asOf := time.Now().UTC()
	if parsed.Date != "" {
		if d, err := time.Parse("2006-01-02", parsed.Date); err == nil {
			asOf = d.UTC()
		}
	}
	out := make(map[string]float64, len(parsed.Rates)+2)
	out[domain.FxBaseUSD] = 1
	out[domain.FxUSDT] = 1
	for code, v := range parsed.Rates {
		code = domain.NormalizeFxCode(code)
		if code == "" || v <= 0 {
			continue
		}
		out[code] = v
	}
	return out, asOf, nil
}
