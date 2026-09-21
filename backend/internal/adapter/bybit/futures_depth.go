package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const linearDepthLimit = 500

// GetFuturesDepth loads a Bybit linear perpetual book snapshot.
func (c *Client) GetFuturesDepth(ctx context.Context, symbol string) (*domain.RawOrderBook, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
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
		got, err := c.fetchLinearDepth(ctx, symbol)
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

func (c *Client) fetchLinearDepth(ctx context.Context, symbol string) (*domain.RawOrderBook, error) {
	params := url.Values{}
	params.Set("category", "linear")
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(linearDepthLimit))
	body, err := c.get(ctx, "/v5/market/orderbook", params)
	if err != nil {
		return nil, err
	}
	var resp bybitRESTBookResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("%w: decode linear orderbook: %v", domain.ErrUpstream, err)
	}
	if resp.RetCode != 0 {
		return nil, mapBybitError(resp.RetCode, resp.RetMsg)
	}
	bids := make([]domain.DepthLevel, 0, len(resp.Result.Bids))
	for _, row := range resp.Result.Bids {
		if len(row) < 2 {
			continue
		}
		qty, err := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
		if err != nil || qty <= 0 {
			continue
		}
		bids = append(bids, domain.DepthLevel{Price: row[0], Quantity: qty})
	}
	asks := make([]domain.DepthLevel, 0, len(resp.Result.Asks))
	for _, row := range resp.Result.Asks {
		if len(row) < 2 {
			continue
		}
		qty, err := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
		if err != nil || qty <= 0 {
			continue
		}
		asks = append(asks, domain.DepthLevel{Price: row[0], Quantity: qty})
	}
	if len(bids) == 0 && len(asks) == 0 {
		return nil, fmt.Errorf("%w: empty perpetual depth for %s", domain.ErrNotFound, symbol)
	}
	return domain.RawBookFromDepthLevels(symbol, resp.Result.UpdateID, bids, asks), nil
}
