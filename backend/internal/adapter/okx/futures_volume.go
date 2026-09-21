package okx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ListFuturesVolumeBars loads OKX USDT-swap 5m quote-volume candles.
func (c *Client) ListFuturesVolumeBars(ctx context.Context, symbol string) ([]domain.FuturesVolumeBar, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	inst := SwapInstID(symbol)
	if inst == "" {
		return nil, fmt.Errorf("%w: okx swap symbol", domain.ErrInvalidArgument)
	}
	q := url.Values{}
	q.Set("instId", inst)
	q.Set("bar", "5m")
	q.Set("limit", strconv.Itoa(domain.FuturesVolumeBars))
	body, err := c.get(ctx, "/api/v5/market/candles", q)
	if err != nil {
		return nil, err
	}
	var rows [][]string
	if err := decodeOKX(body, &rows); err != nil {
		return nil, err
	}
	out := make([]domain.FuturesVolumeBar, 0, len(rows))
	for _, row := range rows {
		if len(row) < 8 {
			continue
		}
		ms, ok := parseMillis(row[0])
		if !ok {
			continue
		}
		vol, ok := parseFloat(row[7])
		if !ok || vol <= 0 {
			continue
		}
		out = append(out, domain.FuturesVolumeBar{Time: ms.UTC(), QuoteVolume: vol})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: futures volume", domain.ErrNotFound)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
