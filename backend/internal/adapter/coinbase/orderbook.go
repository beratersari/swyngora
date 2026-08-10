package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOrderBook fetches Exchange level-2 spot depth (already price-aggregated by Coinbase).
func (c *Client) GetOrderBook(ctx context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	symbol := normalizeProductID(q.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	cacheKey := symbol + "|l2"
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
		path := "/products/" + symbol + "/book"
		params := url.Values{}
		params.Set("level", "2")
		body, err := c.get(fetchCtx, c.exchangeURL, path, params)
		if err != nil {
			return nil, err
		}
		var resp coinbaseBookResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("%w: decode coinbase book: %v", domain.ErrUpstream, err)
		}
		book := &domain.RawOrderBook{
			Symbol:    symbol,
			UpdateID:  resp.Sequence,
			FetchedAt: time.Now().UTC(),
			Bids:      parseCoinbaseSide(resp.Bids),
			Asks:      parseCoinbaseSide(resp.Asks),
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

type coinbaseBookResponse struct {
	Sequence int64      `json:"sequence"`
	Bids     [][]string `json:"bids"`
	Asks     [][]string `json:"asks"`
}

func parseCoinbaseSide(rows [][]string) []domain.PriceLevel {
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
