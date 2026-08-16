package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOrderBook returns the live local Coinbase book (level2 websocket).
// SnapshotOnly uses the public REST book and does not attach a stream.
func (c *Client) GetOrderBook(ctx context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	symbol := normalizeProductID(q.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if q.SnapshotOnly {
		book, err := c.fetchRESTBook(ctx, symbol)
		if err != nil {
			return nil, err
		}
		if limit := domain.ClampOrderBookRawLimit(q.Limit); limit > 0 {
			if len(book.Bids) > limit {
				book.Bids = book.Bids[:limit]
			}
			if len(book.Asks) > limit {
				book.Asks = book.Asks[:limit]
			}
		}
		return book, nil
	}
	limit := domain.ClampOrderBookRawLimit(q.Limit)
	c.ensureDepth()
	if c.depth == nil {
		return nil, fmt.Errorf("%w: coinbase depth hub not configured", domain.ErrUpstream)
	}
	wait, cancel := context.WithTimeout(ctx, c.depthWait)
	defer cancel()
	return c.depth.Get(wait, symbol, limit)
}

func (c *Client) checksumTop(ctx context.Context, symbol string) (bid, ask domain.PriceLevel, ok bool, err error) {
	book, err := c.fetchRESTBook(ctx, symbol)
	if err != nil || book == nil || len(book.Bids) == 0 || len(book.Asks) == 0 {
		return domain.PriceLevel{}, domain.PriceLevel{}, false, err
	}
	return book.Bids[0], book.Asks[0], true, nil
}

func (c *Client) fetchRESTBook(ctx context.Context, symbol string) (*domain.RawOrderBook, error) {
	path := "/products/" + symbol + "/book"
	params := url.Values{}
	params.Set("level", "2")
	body, err := c.get(ctx, c.exchangeURL, path, params)
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
		Source:    domain.OrderBookSourceREST,
	}
	if len(book.Bids) == 0 && len(book.Asks) == 0 {
		return nil, fmt.Errorf("%w: empty order book for %s", domain.ErrNotFound, symbol)
	}
	return book, nil
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
