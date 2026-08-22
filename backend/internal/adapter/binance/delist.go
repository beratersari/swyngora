package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// delistSchedulePath is Binance Wallet MARKET_DATA: spot pair delist times.
// Requires API key header (no HMAC for this MARKET_DATA route in Binance connector).
const delistSchedulePath = "/sapi/v1/spot/delist-schedule"

type delistScheduleRow struct {
	DelistTime int64 `json:"delistTime"`
	// Binance currently returns "symbols"; older docs/examples used "symbol".
	Symbols []string `json:"symbols"`
	Symbol  []string `json:"symbol"`
}

// FetchSpotDelistSchedule implements domain.SpotDelistSchedulePort.
// Returns domain.ErrInvalidArgument when API key is not configured.
func (c *Client) FetchSpotDelistSchedule(ctx context.Context) ([]domain.SpotDelistEntry, error) {
	key := strings.TrimSpace(c.apiKey)
	if key == "" {
		return nil, fmt.Errorf("%w: BINANCE_API_KEY is required for spot delist schedule", domain.ErrInvalidArgument)
	}

	u := c.baseURL + delistSchedulePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	req.Header.Set("X-MBX-APIKEY", key)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", domain.ErrUpstream, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusTeapot {
		return nil, fmt.Errorf("%w: binance status %d", domain.ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: binance delist schedule auth failed (%d): %s",
			domain.ErrUpstream, resp.StatusCode, truncate(string(body), 200))
	}
	if resp.StatusCode >= 400 {
		var er binanceError
		_ = json.Unmarshal(body, &er)
		if er.Msg != "" {
			return nil, mapBinanceError(er.Code, er.Msg)
		}
		return nil, fmt.Errorf("%w: status %d: %s", domain.ErrUpstream, resp.StatusCode, truncate(string(body), 200))
	}

	var rows []delistScheduleRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: decode delist schedule: %v", domain.ErrUpstream, err)
	}

	out := make([]domain.SpotDelistEntry, 0, 32)
	for _, row := range rows {
		if row.DelistTime <= 0 {
			continue
		}
		t := time.UnixMilli(row.DelistTime).UTC()
		syms := row.Symbols
		if len(syms) == 0 {
			syms = row.Symbol
		}
		for _, sym := range syms {
			sym = strings.ToUpper(strings.TrimSpace(sym))
			if sym == "" {
				continue
			}
			out = append(out, domain.SpotDelistEntry{
				Exchange:   domain.ExchangeBinance,
				Symbol:     sym,
				DelistTime: t,
			})
		}
	}
	if extra, err := c.fetchCMSWillDelistEntries(ctx); err == nil {
		out = mergeDelistEntriesPreferFirst(out, extra)
	}
	return out, nil
}

const cmsDelistCatalogPath = "/bapi/apex/v1/public/apex/cms/article/list/query"

var willDelistTitle = regexp.MustCompile(`(?i)^Binance Will Delist (.+) on (20\d{2}-\d{2}-\d{2})\s*$`)

type cmsArticleListResponse struct {
	Code string `json:"code"`
	Data struct {
		Catalogs []struct {
			CatalogID int `json:"catalogId"`
			Articles  []struct {
				Title string `json:"title"`
			} `json:"articles"`
		} `json:"catalogs"`
	} `json:"data"`
}

// fetchCMSWillDelistEntries reads public Delisting catalog titles so last-30-day
// full-token removals still appear after they drop off the official schedule.
func (c *Client) fetchCMSWillDelistEntries(ctx context.Context) ([]domain.SpotDelistEntry, error) {
	base := strings.TrimRight(c.productBaseURL, "/")
	if base == "" {
		return nil, nil
	}
	u := base + cmsDelistCatalogPath + "?type=1&catalogId=161&pageNo=1&pageSize=20"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cms status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	var parsed cmsArticleListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]domain.SpotDelistEntry, 0, 16)
	for _, cat := range parsed.Data.Catalogs {
		for _, art := range cat.Articles {
			tokens, when, ok := parseBinanceWillDelistTitle(art.Title)
			if !ok || !domain.DelistVisibleOnTradingList(when, now) {
				continue
			}
			for _, tok := range tokens {
				out = append(out, domain.SpotDelistEntry{
					Exchange:   domain.ExchangeBinance,
					Symbol:     tok + "USDT",
					DelistTime: when,
				})
			}
		}
	}
	return out, nil
}

func parseBinanceWillDelistTitle(title string) (tokens []string, when time.Time, ok bool) {
	m := willDelistTitle.FindStringSubmatch(strings.TrimSpace(title))
	if len(m) != 3 {
		return nil, time.Time{}, false
	}
	when, err := time.ParseInLocation("2006-01-02", m[2], time.UTC)
	if err != nil {
		return nil, time.Time{}, false
	}
	blob := strings.ToLower(m[1])
	if strings.Contains(blob, "margin") || strings.Contains(blob, "futures") ||
		strings.Contains(blob, "alpha") || strings.Contains(blob, "loan") {
		return nil, time.Time{}, false
	}
	for _, part := range strings.Split(m[1], ",") {
		tok := strings.ToUpper(strings.TrimSpace(part))
		tok = strings.TrimPrefix(tok, "AND ")
		tok = strings.TrimSpace(tok)
		if tok == "" || strings.ContainsAny(tok, " /") {
			continue
		}
		tokens = append(tokens, tok)
	}
	if len(tokens) == 0 {
		return nil, time.Time{}, false
	}
	return tokens, when, true
}

func mergeDelistEntriesPreferFirst(primary, extra []domain.SpotDelistEntry) []domain.SpotDelistEntry {
	seen := make(map[string]struct{}, len(primary)+len(extra))
	out := make([]domain.SpotDelistEntry, 0, len(primary)+len(extra))
	for _, e := range primary {
		sym := strings.ToUpper(strings.TrimSpace(e.Symbol))
		if sym == "" {
			continue
		}
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}
		out = append(out, e)
	}
	for _, e := range extra {
		sym := strings.ToUpper(strings.TrimSpace(e.Symbol))
		if sym == "" {
			continue
		}
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}
		out = append(out, e)
	}
	return out
}
