package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetFuturesDepth loads the public l2Book. Hyperliquid returns at most 20
// levels a side. A band those levels do not reach stays incomplete upstream.
func (c *Client) GetFuturesDepth(ctx context.Context, symbol string) (*domain.RawOrderBook, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	coin, ok := Coin(symbol)
	if !ok {
		return nil, fmt.Errorf("%w: hyperliquid symbol", domain.ErrInvalidArgument)
	}
	if c == nil {
		return nil, fmt.Errorf("%w: hyperliquid client", domain.ErrUpstream)
	}
	if c.depth != nil {
		if hit, ok := c.depth.Get(symbol); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}
	v, err, _ := c.depthSF.Do(symbol, func() (any, error) {
		if c.depth != nil {
			if hit, ok := c.depth.Get(symbol); ok && hit != nil {
				return hit, nil
			}
		}
		got, ferr := c.fetchDepth(ctx, symbol, coin)
		if ferr != nil {
			return nil, ferr
		}
		if c.depth != nil {
			c.depth.Set(symbol, got)
		}
		return got, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.(*domain.RawOrderBook)
	if got == nil {
		return nil, fmt.Errorf("%w: empty hyperliquid book", domain.ErrUpstream)
	}
	cp := *got
	return &cp, nil
}

func (c *Client) fetchDepth(ctx context.Context, symbol, coin string) (*domain.RawOrderBook, error) {
	if err := c.hasCoin(ctx, coin); err != nil {
		return nil, err
	}
	body, err := c.post(ctx, map[string]string{"type": "l2Book", "coin": coin})
	if err != nil {
		return nil, err
	}
	var raw struct {
		Time   int64 `json:"time"`
		Levels [][]struct {
			Px string `json:"px"`
			Sz string `json:"sz"`
		} `json:"levels"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: hyperliquid book", domain.ErrUpstream)
	}
	if len(raw.Levels) < 2 {
		return nil, fmt.Errorf("%w: hyperliquid book for %s", domain.ErrNotFound, coin)
	}
	bids := depthSide(raw.Levels[0])
	asks := depthSide(raw.Levels[1])
	if len(bids) == 0 && len(asks) == 0 {
		return nil, fmt.Errorf("%w: hyperliquid book for %s", domain.ErrNotFound, coin)
	}
	book := domain.RawBookFromDepthLevels(symbol, raw.Time, bids, asks)
	sort.Slice(book.Bids, func(i, j int) bool { return book.Bids[i].Price > book.Bids[j].Price })
	sort.Slice(book.Asks, func(i, j int) bool { return book.Asks[i].Price < book.Asks[j].Price })
	return book, nil
}

func depthSide(rows []struct {
	Px string `json:"px"`
	Sz string `json:"sz"`
}) []domain.DepthLevel {
	out := make([]domain.DepthLevel, 0, len(rows))
	for _, row := range rows {
		qty, ok := parsePositive(row.Sz)
		if !ok {
			continue
		}
		if _, ok := parsePositive(row.Px); !ok {
			continue
		}
		out = append(out, domain.DepthLevel{Price: row.Px, Quantity: qty})
	}
	return out
}
