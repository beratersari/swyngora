package coingecko

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

const ohlcCacheTTL = 15 * time.Minute

// QuoteByBase returns the CoinGecko USD last and 24h change for a ticker.
func (c *Client) QuoteByBase(ctx context.Context, base string) (*domain.OffVenueQuote, error) {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		return nil, fmt.Errorf("%w: base is required", domain.ErrInvalidArgument)
	}
	got, err := c.SupplyBySymbols(ctx, []string{base})
	if err == nil {
		if q := quoteFromSupply(c, base, got[base]); q != nil {
			return q, nil
		}
	}
	if q := c.quoteFromMarkets(ctx, base); q != nil {
		return q, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: no public price for %s", domain.ErrNotFound, base)
}

func quoteFromSupply(c *Client, base string, sup *domain.AssetSupply) *domain.OffVenueQuote {
	if sup == nil || sup.CurrentPriceUSD == nil {
		return nil
	}
	q := &domain.OffVenueQuote{
		Symbol:     base,
		Name:       sup.Name,
		ProviderID: sup.ProviderID,
		LastUSD:    *sup.CurrentPriceUSD,
		AsOf:       sup.AsOf,
	}
	if c != nil && c.changeCache != nil {
		if pct, ok := c.changeCache.Get(base); ok {
			q.ChangePct = pct
		}
	}
	if c != nil && c.changeAbsCache != nil {
		if abs, ok := c.changeAbsCache.Get(base); ok {
			q.ChangeAbs = abs
		}
	}
	q.FillChangeAbs()
	return q
}

func (c *Client) quoteFromMarkets(ctx context.Context, base string) *domain.OffVenueQuote {
	rows, err := c.fetchMarkets(ctx, []string{strings.ToLower(base)})
	if err != nil {
		rows = nil
	}
	if q := c.quoteFromRows(base, rows); q != nil {
		return q
	}
	id, ok := c.searchID(ctx, base)
	if !ok {
		return nil
	}
	extra, err := c.fetchMarketsByIDs(ctx, []string{id})
	if err != nil {
		return nil
	}
	return c.quoteFromRows(base, extra)
}

func (c *Client) quoteFromRows(base string, rows []marketRow) *domain.OffVenueQuote {
	for _, row := range rows {
		if row.CurrentPrice == nil {
			continue
		}
		if row.Symbol != "" && !strings.EqualFold(row.Symbol, base) {
			continue
		}
		if row.PriceChangePct24h != nil && c.changeCache != nil {
			c.changeCache.Set(base, cloneF(row.PriceChangePct24h))
		}
		if row.PriceChange24h != nil && c.changeAbsCache != nil {
			c.changeAbsCache.Set(base, cloneF(row.PriceChange24h))
		}
		q := &domain.OffVenueQuote{
			Symbol:     base,
			Name:       strings.TrimSpace(row.Name),
			ProviderID: row.ID,
			LastUSD:    *row.CurrentPrice,
			ChangePct:  cloneF(row.PriceChangePct24h),
			ChangeAbs:  cloneF(row.PriceChange24h),
			AsOf:       time.Now().UTC(),
		}
		q.FillChangeAbs()
		return q
	}
	return nil
}

// OHLCByBase returns CoinGecko USD OHLC (daily for days>=30).
func (c *Client) OHLCByBase(ctx context.Context, base string, days int) ([]domain.Candle, error) {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		return nil, fmt.Errorf("%w: base is required", domain.ErrInvalidArgument)
	}
	if days <= 0 {
		days = 30
	}
	key := base + "|" + strconv.Itoa(days)
	if c.ohlcCache != nil {
		if hit, ok := c.ohlcCache.Get(key); ok {
			return append([]domain.Candle(nil), hit...), nil
		}
	}
	id := ""
	if got, err := c.SupplyBySymbols(ctx, []string{base}); err == nil && got[base] != nil {
		id = strings.TrimSpace(got[base].ProviderID)
	}
	if id == "" {
		found, ok := c.searchID(ctx, base)
		if !ok {
			return nil, fmt.Errorf("%w: no public ohlc for %s", domain.ErrNotFound, base)
		}
		id = found
	}
	bars, err := c.fetchOHLC(ctx, id, days)
	if err != nil {
		return nil, err
	}
	if c.ohlcCache != nil {
		c.ohlcCache.Set(key, append([]domain.Candle(nil), bars...))
	}
	return bars, nil
}

func (c *Client) fetchOHLC(ctx context.Context, id string, days int) ([]domain.Candle, error) {
	u, err := url.Parse(c.baseURL + "/api/v3/coins/" + url.PathEscape(id) + "/ohlc")
	if err != nil {
		return nil, fmt.Errorf("%w: build ohlc url: %v", domain.ErrUpstream, err)
	}
	q := u.Query()
	q.Set("vs_currency", "usd")
	q.Set("days", strconv.Itoa(days))
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build ohlc request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: ohlc request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read ohlc body: %v", domain.ErrUpstream, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: coingecko status 429", domain.ErrRateLimited)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: coingecko ohlc status %d: %s", domain.ErrUpstream, resp.StatusCode, truncate(string(body), 160))
	}
	var raw [][]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode ohlc: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.Candle, 0, len(raw))
	for _, row := range raw {
		if len(row) < 5 {
			continue
		}
		ms := int64(row[0])
		open := time.UnixMilli(ms).UTC()
		out = append(out, domain.Candle{
			OpenTime:  open,
			CloseTime: open.Add(24 * time.Hour).Add(-time.Millisecond),
			Open:      formatFloat(row[1]),
			High:      formatFloat(row[2]),
			Low:       formatFloat(row[3]),
			Close:     formatFloat(row[4]),
		})
	}
	return out, nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}


