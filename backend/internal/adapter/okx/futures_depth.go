package okx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const swapDepthLimit = 400

// GetFuturesDepth loads an OKX USDT-swap book. Contract size is converted to
// base quantity with ctVal so notional matches Binance and Bybit (price × base).
func (c *Client) GetFuturesDepth(ctx context.Context, symbol string) (*domain.RawOrderBook, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	if c.depthCache != nil {
		if hit, ok := c.depthCache.Get(symbol); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}
	v, err, _ := c.depthSF.Do(symbol, func() (any, error) {
		if c.depthCache != nil {
			if hit, ok := c.depthCache.Get(symbol); ok && hit != nil {
				return hit, nil
			}
		}
		got, ferr := c.fetchSwapDepth(ctx, symbol)
		if ferr != nil {
			return nil, ferr
		}
		if c.depthCache != nil {
			c.depthCache.Set(symbol, got)
		}
		return got, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.(*domain.RawOrderBook)
	if got == nil {
		return nil, fmt.Errorf("%w: empty perpetual depth", domain.ErrUpstream)
	}
	cp := *got
	return &cp, nil
}

func (c *Client) fetchSwapDepth(ctx context.Context, symbol string) (*domain.RawOrderBook, error) {
	inst := SwapInstID(symbol)
	if inst == "" {
		return nil, fmt.Errorf("%w: okx swap symbol", domain.ErrInvalidArgument)
	}
	ctVal := c.contractValue(ctx, inst)
	if ctVal <= 0 {
		return nil, fmt.Errorf("%w: okx contract size for %s", domain.ErrUpstream, inst)
	}
	q := url.Values{}
	q.Set("instId", inst)
	q.Set("sz", strconv.Itoa(swapDepthLimit))
	body, err := c.get(ctx, "/api/v5/market/books", q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
		Ts   string     `json:"ts"`
	}
	if err := decodeOKX(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: empty perpetual depth for %s", domain.ErrNotFound, symbol)
	}
	bids := okxDepthSide(rows[0].Bids, ctVal)
	asks := okxDepthSide(rows[0].Asks, ctVal)
	if len(bids) == 0 && len(asks) == 0 {
		return nil, fmt.Errorf("%w: empty perpetual depth for %s", domain.ErrNotFound, symbol)
	}
	ts, _ := strconv.ParseInt(rows[0].Ts, 10, 64)
	return domain.RawBookFromDepthLevels(symbol, ts, bids, asks), nil
}

func okxDepthSide(rows [][]string, ctVal float64) []domain.DepthLevel {
	out := make([]domain.DepthLevel, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		contracts, ok := parseFloat(row[1])
		if !ok || contracts <= 0 {
			continue
		}
		out = append(out, domain.DepthLevel{Price: row[0], Quantity: contracts * ctVal})
	}
	return out
}
