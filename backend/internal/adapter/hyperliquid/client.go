package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
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
	defaultBaseURL = "https://api.hyperliquid.xyz"
	infoPath       = "/info"
	bookTTL        = 20 * time.Second
	maxBody        = 8 << 20
	fundingPageCap = 500
	fundingPages   = 8
)

// Client reads Hyperliquid's public info endpoint.
// One metaAndAssetCtxs call covers mark, open interest, and the current funding rate.
// There is no liquidation stream, trade tape, or order book in this client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	books      *cache.TTL[assetBook]
	bookSF     singleflight.Group
}

// Options configures the public info client.
type Options struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient constructs a Hyperliquid info client.
func NewClient(opts Options) *Client {
	base := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if base == "" {
		base = defaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    base,
		httpClient: hc,
		books:      cache.New[assetBook](bookTTL),
	}
}

type asset struct {
	mark     float64
	oi       float64
	funding  float64
	notional float64
}

type assetBook struct {
	at    time.Time
	coins map[string]asset
}

// Coin maps a USDT perpetual name onto Hyperliquid's coin id.
// BTCUSDT is BTC. A name Hyperliquid does not list stays unmatched.
func Coin(symbol string) (string, bool) {
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	base, quote := domain.SplitBaseQuote(domain.ExchangeBinance, symbol)
	if base == "" || quote != "USDT" {
		return "", false
	}
	return base, true
}

// GetOpenInterestSeries returns the current coin open interest and its mark notional.
// Hyperliquid does not publish an open-interest history, so History is empty.
func (c *Client) GetOpenInterestSeries(ctx context.Context, symbol string) (*domain.OpenInterestSeries, error) {
	coin, book, row, err := c.lookup(ctx, symbol)
	if err != nil {
		return nil, err
	}
	_ = coin
	return &domain.OpenInterestSeries{
		Exchange: domain.ExchangeHyperliquid,
		Symbol:   domain.NormalizeLiquidationSymbol(symbol),
		Current: domain.OpenInterestPoint{
			Time:      book.at,
			Contracts: row.oi,
			Value:     row.notional,
		},
	}, nil
}

// GetFundingSeries returns the current hourly funding rate and the mark.
// Settled history is loaded by ListFundingHistory so a live sample does not page the archive.
func (c *Client) GetFundingSeries(ctx context.Context, symbol string, _ int) (*domain.FundingSeries, error) {
	_, book, row, err := c.lookup(ctx, symbol)
	if err != nil {
		return nil, err
	}
	next := book.at.UTC().Truncate(time.Hour).Add(time.Hour)
	return &domain.FundingSeries{
		Exchange: domain.ExchangeHyperliquid,
		Symbol:   domain.NormalizeLiquidationSymbol(symbol),
		Current: domain.FundingPoint{
			Time:      book.at,
			Rate:      row.funding,
			MarkPrice: row.mark,
			Predicted: true,
		},
		NextFundingTime: next,
		IntervalHours:   1,
	}, nil
}

// ListFundingHistory pages settled hourly funding inside [from, to].
func (c *Client) ListFundingHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.FundingPoint, error) {
	coin, ok := Coin(symbol)
	if !ok {
		return nil, fmt.Errorf("%w: hyperliquid symbol", domain.ErrInvalidArgument)
	}
	if c == nil {
		return nil, fmt.Errorf("%w: hyperliquid client", domain.ErrUpstream)
	}
	from, to = from.UTC(), to.UTC()
	if !to.After(from) {
		return nil, nil
	}
	start := from.UnixMilli()
	end := to.UnixMilli()
	var out []domain.FundingPoint
	for page := 0; page < fundingPages; page++ {
		rows, err := c.fundingPage(ctx, coin, start, end)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		var maxMs int64
		for _, row := range rows {
			if row.Time.Before(from) || row.Time.After(to) {
				continue
			}
			out = append(out, row)
			if ms := row.Time.UnixMilli(); ms > maxMs {
				maxMs = ms
			}
		}
		if len(rows) < fundingPageCap || maxMs <= start {
			break
		}
		start = maxMs + 1
		if start > end {
			break
		}
	}
	return out, nil
}

func (c *Client) lookup(ctx context.Context, symbol string) (string, assetBook, asset, error) {
	coin, ok := Coin(symbol)
	if !ok {
		return "", assetBook{}, asset{}, fmt.Errorf("%w: hyperliquid symbol", domain.ErrInvalidArgument)
	}
	book, err := c.book(ctx)
	if err != nil {
		return "", assetBook{}, asset{}, err
	}
	row, ok := book.coins[coin]
	if !ok {
		return "", assetBook{}, asset{}, fmt.Errorf("%w: hyperliquid %s", domain.ErrNotFound, coin)
	}
	return coin, book, row, nil
}

func (c *Client) book(ctx context.Context) (assetBook, error) {
	if c == nil {
		return assetBook{}, fmt.Errorf("%w: hyperliquid client", domain.ErrUpstream)
	}
	if hit, ok := c.books.Get("book"); ok {
		return hit, nil
	}
	v, err, _ := c.bookSF.Do("book", func() (any, error) {
		if hit, ok := c.books.Get("book"); ok {
			return hit, nil
		}
		got, err := c.fetchBook(ctx)
		if err != nil {
			return assetBook{}, err
		}
		c.books.Set("book", got)
		return got, nil
	})
	if err != nil {
		return assetBook{}, err
	}
	book, _ := v.(assetBook)
	return book, nil
}

func (c *Client) fetchBook(ctx context.Context) (assetBook, error) {
	body, err := c.post(ctx, map[string]string{"type": "metaAndAssetCtxs"})
	if err != nil {
		return assetBook{}, err
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) < 2 {
		return assetBook{}, fmt.Errorf("%w: hyperliquid asset contexts", domain.ErrUpstream)
	}
	var meta struct {
		Universe []struct {
			Name string `json:"name"`
		} `json:"universe"`
	}
	if err := json.Unmarshal(raw[0], &meta); err != nil {
		return assetBook{}, fmt.Errorf("%w: hyperliquid universe", domain.ErrUpstream)
	}
	var ctxs []struct {
		Funding      string `json:"funding"`
		OpenInterest string `json:"openInterest"`
		MarkPx       string `json:"markPx"`
	}
	if err := json.Unmarshal(raw[1], &ctxs); err != nil {
		return assetBook{}, fmt.Errorf("%w: hyperliquid contexts", domain.ErrUpstream)
	}
	out := assetBook{at: time.Now().UTC(), coins: map[string]asset{}}
	n := len(meta.Universe)
	if len(ctxs) < n {
		n = len(ctxs)
	}
	for i := 0; i < n; i++ {
		name := strings.TrimSpace(meta.Universe[i].Name)
		mark, markOK := parsePositive(ctxs[i].MarkPx)
		oi, oiOK := parsePositive(ctxs[i].OpenInterest)
		rate, rateOK := parseFloat(ctxs[i].Funding)
		if name == "" || !markOK || !oiOK || !rateOK {
			continue
		}
		out.coins[name] = asset{
			mark:     mark,
			oi:       oi,
			funding:  rate,
			notional: oi * mark,
		}
	}
	if len(out.coins) == 0 {
		return assetBook{}, fmt.Errorf("%w: hyperliquid asset contexts empty", domain.ErrUpstream)
	}
	return out, nil
}

func (c *Client) fundingPage(ctx context.Context, coin string, start, end int64) ([]domain.FundingPoint, error) {
	body, err := c.post(ctx, map[string]any{
		"type":      "fundingHistory",
		"coin":      coin,
		"startTime": start,
		"endTime":   end,
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		FundingRate string `json:"fundingRate"`
		Time        int64  `json:"time"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: hyperliquid funding history", domain.ErrUpstream)
	}
	out := make([]domain.FundingPoint, 0, len(rows))
	for _, row := range rows {
		rate, ok := parseFloat(row.FundingRate)
		if !ok || row.Time <= 0 {
			continue
		}
		out = append(out, domain.FundingPoint{
			Time: time.UnixMilli(row.Time).UTC(),
			Rate: rate,
		})
	}
	return out, nil
}

func (c *Client) post(ctx context.Context, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+infoPath, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: hyperliquid %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	buf, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%w: hyperliquid body", domain.ErrUpstream)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: hyperliquid", domain.ErrRateLimited)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: hyperliquid status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return buf, nil
}

func parsePositive(raw string) (float64, bool) {
	v, ok := parseFloat(raw)
	return v, ok && v > 0
}

func parseFloat(raw string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || v != v {
		return 0, false
	}
	return v, true
}
