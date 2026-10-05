package binance

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ListAggTradeWindow pages USD-M aggTrades inside [from, to).
// The window must be shorter than one hour. Results are not cached.
// A window that still fills the page budget returns ErrTapeWindowFull
// and no prints, so the caller can shrink it without storing a partial hour.
func (c *Client) ListAggTradeWindow(ctx context.Context, symbol string, from, to time.Time) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	from, to = from.UTC(), to.UTC()
	if !to.After(from) {
		return nil, fmt.Errorf("%w: empty aggTrades window", domain.ErrInvalidArgument)
	}
	if to.Sub(from) > domain.BinanceTapeWindow {
		return nil, fmt.Errorf("%w: aggTrades window longer than %s", domain.ErrInvalidArgument, domain.BinanceTapeWindow)
	}
	start := from.UnixMilli()
	end := to.UnixMilli()
	var out []domain.TakerPrint
	for page := 0; page < domain.BinanceTapeMaxPages; page++ {
		if start >= end {
			break
		}
		q := url.Values{}
		q.Set("symbol", symbol)
		q.Set("limit", "1000")
		q.Set("startTime", strconv.FormatInt(start, 10))
		q.Set("endTime", strconv.FormatInt(end, 10))
		chunk, err := c.decodeAggTrades(ctx, symbol, q, false)
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}
		out = append(out, chunk...)
		last := chunk[len(chunk)-1].Time.UnixMilli()
		if last < start {
			break
		}
		next := last + 1
		if next <= start {
			break
		}
		start = next
		if len(chunk) < 1000 {
			break
		}
		if page == domain.BinanceTapeMaxPages-1 {
			return nil, fmt.Errorf("%w", domain.ErrTapeWindowFull)
		}
	}
	return printsBefore(out, from, to), nil
}

func printsBefore(prints []domain.TakerPrint, from, to time.Time) []domain.TakerPrint {
	out := make([]domain.TakerPrint, 0, len(prints))
	for _, p := range prints {
		t := p.Time.UTC()
		if t.Before(from) || !t.Before(to) {
			continue
		}
		p.Time = t
		out = append(out, p)
	}
	return out
}
