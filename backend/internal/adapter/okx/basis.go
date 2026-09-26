package okx

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBasisQuote loads OKX USDT-swap last and mark versus the USDT index.
func (c *Client) GetBasisQuote(ctx context.Context, symbol string) (*domain.BasisQuote, error) {
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
	var (
		last, mark, index float64
		at                time.Time
		errL, errM, errI  error
	)
	errL = c.onePrice(ctx, "/api/v5/market/ticker", url.Values{"instId": {inst}}, "last", &last, &at)
	errM = c.onePrice(ctx, "/api/v5/public/mark-price", url.Values{"instId": {inst}}, "markPx", &mark, nil)
	idx := inst
	if len(inst) > len("-SWAP") {
		idx = inst[:len(inst)-len("-SWAP")]
	}
	errI = c.onePrice(ctx, "/api/v5/market/index-tickers", url.Values{"instId": {idx}}, "idxPx", &index, nil)
	if errM != nil {
		return nil, errM
	}
	if errI != nil {
		return nil, errI
	}
	if mark <= 0 || index <= 0 {
		return nil, fmt.Errorf("%w: okx basis prices", domain.ErrUpstream)
	}
	if errL != nil || last <= 0 {
		last = mark
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return &domain.BasisQuote{
		Exchange:    domain.ExchangeOKX,
		Symbol:      symbol,
		FuturesLast: last,
		FuturesMark: mark,
		SpotIndex:   index,
		Time:        at.UTC(),
	}, nil
}

func (c *Client) onePrice(ctx context.Context, path string, q url.Values, field string, dest *float64, at *time.Time) error {
	body, err := c.get(ctx, path, q)
	if err != nil {
		return err
	}
	var rows []map[string]string
	if err := decodeOKX(body, &rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("%w: empty %s", domain.ErrNotFound, path)
	}
	v, ok := parseFloat(rows[0][field])
	if !ok || v <= 0 {
		return fmt.Errorf("%w: %s %s", domain.ErrUpstream, path, field)
	}
	*dest = v
	if at != nil {
		if ts, ok := parseMillis(rows[0]["ts"]); ok {
			*at = ts
		}
	}
	return nil
}
