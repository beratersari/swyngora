package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOrderBook fetches a raw spot order book (ungrouped).
func (c *Client) GetOrderBook(ctx context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	symbol := strings.ToUpper(strings.TrimSpace(q.Symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	limit := domain.ClampOrderBookRawLimit(q.Limit)
	if limit > 200 {
		limit = 200 // Bybit spot max
	}
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
		params.Set("category", "spot")
		params.Set("symbol", symbol)
		params.Set("limit", strconv.Itoa(limit))
		body, err := c.get(fetchCtx, "/v5/market/orderbook", params)
		if err != nil {
			return nil, err
		}
		var resp bybitOrderBookResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("%w: decode bybit orderbook: %v", domain.ErrUpstream, err)
		}
		if resp.RetCode != 0 {
			return nil, mapBybitError(resp.RetCode, resp.RetMsg)
		}
		book := &domain.RawOrderBook{
			Symbol:    symbol,
			UpdateID:  resp.Result.UpdateID,
			FetchedAt: time.Now().UTC(),
			Bids:      parseSide(resp.Result.Bids),
			Asks:      parseSide(resp.Result.Asks),
		}
		if resp.Result.Ts > 0 {
			book.FetchedAt = time.UnixMilli(resp.Result.Ts).UTC()
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

type bybitOrderBookResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Symbol   string     `json:"s"`
		Bids     [][]string `json:"b"`
		Asks     [][]string `json:"a"`
		Ts       int64      `json:"ts"`
		UpdateID int64      `json:"u"`
	} `json:"result"`
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
