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

// GetOrderBook fetches a raw spot depth snapshot (ungrouped).
func (c *Client) GetOrderBook(ctx context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	symbol := normalizeSymbol(q.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	limit := domain.ClampOrderBookRawLimit(q.Limit)
	// Binance only accepts a fixed set of limits.
	limit = snapBinanceDepthLimit(limit)
	cacheKey := symbol + "|" + strconv.Itoa(limit)
	if c.orderBooks != nil {
		if hit, ok := c.orderBooks.Get(cacheKey); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}
	v, err, _ := c.orderBookSF.Do(cacheKey, func() (any, error) {
		if c.orderBooks != nil {
			if hit, ok := c.orderBooks.Get(cacheKey); ok && hit != nil {
				return hit, nil
			}
		}
		fetchCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		params := url.Values{}
		params.Set("symbol", symbol)
		params.Set("limit", strconv.Itoa(limit))
		body, err := c.get(fetchCtx, "/api/v3/depth", params)
		if err != nil {
			return nil, err
		}
		var resp binanceDepthResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("%w: decode depth: %v", domain.ErrUpstream, err)
		}
		if resp.Code != 0 && resp.Msg != "" {
			return nil, mapBinanceError(resp.Code, resp.Msg)
		}
		book := &domain.RawOrderBook{
			Symbol:    symbol,
			UpdateID:  resp.LastUpdateID,
			FetchedAt: time.Now().UTC(),
			Bids:      parseSide(resp.Bids),
			Asks:      parseSide(resp.Asks),
		}
		if len(book.Bids) == 0 && len(book.Asks) == 0 {
			return nil, fmt.Errorf("%w: empty order book for %s", domain.ErrNotFound, symbol)
		}
		if c.orderBooks != nil {
			c.orderBooks.Set(cacheKey, book)
		}
		return book, nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	hit := v.(*domain.RawOrderBook)
	cp := *hit
	return &cp, nil
}

type binanceDepthResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
	Code         int        `json:"code"`
	Msg          string     `json:"msg"`
}

func parseSide(rows [][]string) []domain.PriceLevel {
	out := make([]domain.PriceLevel, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		lv, ok := domain.ParsePriceQty(row[0], row[1])
		if ok {
			out = append(out, lv)
		}
	}
	return out
}

func snapBinanceDepthLimit(n int) int {
	// Allowed: 5, 10, 20, 50, 100, 500, 1000, 5000. We cap at 500.
	switch {
	case n <= 5:
		return 5
	case n <= 10:
		return 10
	case n <= 20:
		return 20
	case n <= 50:
		return 50
	case n <= 100:
		return 100
	default:
		return 500
	}
}
