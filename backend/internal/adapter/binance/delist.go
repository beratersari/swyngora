package binance

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
	return out, nil
}
