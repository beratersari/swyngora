package okx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// okxCandleBar maps a Swyngora interval onto an OKX SWAP bar.
// Daily and weekly bars use the UTC open so they line up with UTC columns.
func okxCandleBar(interval string) (string, bool) {
	switch interval {
	case "1m", "3m", "5m", "15m", "30m":
		return interval, true
	case "1h":
		return "1H", true
	case "2h":
		return "2H", true
	case "4h":
		return "4H", true
	case "6h":
		return "6H", true
	case "12h":
		return "12H", true
	case "1d":
		return "1Dutc", true
	case "1w":
		return "1Wutc", true
	default:
		return "", false
	}
}

// GetFuturesCandles loads OKX USDT-swap OHLC. The public candles route
// returns the newest rows first and at most 300 of them.
func (c *Client) GetFuturesCandles(ctx context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(q.Symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	bar, ok := okxCandleBar(string(q.Interval))
	if !ok {
		return nil, fmt.Errorf("%w: okx candle interval %q", domain.ErrInvalidArgument, q.Interval)
	}
	inst := SwapInstID(symbol)
	if inst == "" {
		return nil, fmt.Errorf("%w: okx swap symbol", domain.ErrInvalidArgument)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 300 {
		limit = 300
	}
	params := url.Values{}
	params.Set("instId", inst)
	params.Set("bar", bar)
	params.Set("limit", strconv.Itoa(limit))
	body, err := c.get(ctx, "/api/v5/market/candles", params)
	if err != nil {
		return nil, err
	}
	var rows [][]string
	if err := decodeOKX(body, &rows); err != nil {
		return nil, err
	}
	dur := domain.IntervalDuration(q.Interval)
	out := make([]domain.Candle, 0, len(rows))
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		open, ok := parseMillis(row[0])
		if !ok || row[4] == "" {
			continue
		}
		open = open.UTC()
		closeAt := open
		if dur > 0 {
			closeAt = open.Add(dur)
		}
		if !q.StartTime.IsZero() && closeAt.Before(q.StartTime) {
			continue
		}
		if !q.EndTime.IsZero() && open.After(q.EndTime) {
			continue
		}
		out = append(out, domain.Candle{
			OpenTime:  open,
			Open:      row[1],
			High:      row[2],
			Low:       row[3],
			Close:     row[4],
			CloseTime: closeAt,
		})
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: okx candles", domain.ErrNotFound)
	}
	return out, nil
}
