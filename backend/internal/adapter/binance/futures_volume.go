package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const futuresVolumeLimit = domain.FuturesVolumeBars

// ListFuturesVolumeBars loads Binance USD-M 5m quote-volume klines.
func (c *Client) ListFuturesVolumeBars(ctx context.Context, symbol string) ([]domain.FuturesVolumeBar, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("interval", "5m")
	q.Set("limit", strconv.Itoa(futuresVolumeLimit))
	body, err := c.getFutures(ctx, "/fapi/v1/klines", q)
	if err != nil {
		return nil, err
	}
	var rows [][]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: futures volume decode: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.FuturesVolumeBar, 0, len(rows))
	for _, row := range rows {
		if len(row) < 8 {
			continue
		}
		ms, ok := jsonInt64(row[0])
		if !ok || ms <= 0 {
			continue
		}
		vol, ok := jsonFloat(row[7])
		if !ok || vol <= 0 {
			continue
		}
		out = append(out, domain.FuturesVolumeBar{Time: time.UnixMilli(ms).UTC(), QuoteVolume: vol})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: futures volume", domain.ErrNotFound)
	}
	return out, nil
}

func jsonFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		n, err := t.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(t, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
