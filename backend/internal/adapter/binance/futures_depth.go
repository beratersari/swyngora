package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const futuresDepthLimit = 1000

// GetFuturesDepth loads a Binance USD-M perpetual book snapshot.
func (c *Client) GetFuturesDepth(ctx context.Context, symbol string) (*domain.RawOrderBook, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	if c.futDepthCache != nil {
		if hit, ok := c.futDepthCache.Get(symbol); ok {
			cp := *hit
			return &cp, nil
		}
	}
	v, err, _ := c.futDepthSF.Do(symbol, func() (any, error) {
		if c.futDepthCache != nil {
			if hit, ok := c.futDepthCache.Get(symbol); ok {
				return hit, nil
			}
		}
		got, err := c.fetchFuturesDepth(ctx, symbol)
		if err != nil {
			return nil, err
		}
		if c.futDepthCache != nil {
			c.futDepthCache.Set(symbol, got)
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

func (c *Client) fetchFuturesDepth(ctx context.Context, symbol string) (*domain.RawOrderBook, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("limit", strconv.Itoa(futuresDepthLimit))
	body, err := c.getFutures(ctx, "/fapi/v1/depth", q)
	if err != nil {
		return nil, err
	}
	var resp binanceDepthResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("%w: decode futures depth: %v", domain.ErrUpstream, err)
	}
	if resp.Code != 0 && resp.Msg != "" {
		return nil, mapBinanceError(resp.Code, resp.Msg)
	}
	_, bids := parseSideKeep(resp.Bids)
	_, asks := parseSideKeep(resp.Asks)
	if len(bids) == 0 && len(asks) == 0 {
		return nil, fmt.Errorf("%w: empty perpetual depth for %s", domain.ErrNotFound, symbol)
	}
	return domain.RawBookFromDepthLevels(symbol, resp.LastUpdateID, bids, asks), nil
}
