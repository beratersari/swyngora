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

// GetOrderBook returns the live local Bybit spot book (orderbook.200 websocket).
// SnapshotOnly uses REST /v5/market/orderbook and does not attach a stream.
func (c *Client) GetOrderBook(ctx context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	symbol := strings.ToUpper(strings.TrimSpace(q.Symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	limit := domain.ClampOrderBookRawLimit(q.Limit)
	if q.SnapshotOnly {
		return c.fetchRESTBook(ctx, symbol, limit)
	}
	c.ensureDepth()
	if c.depth == nil {
		return nil, fmt.Errorf("%w: bybit depth hub not configured", domain.ErrUpstream)
	}
	wait, cancel := context.WithTimeout(ctx, c.depthWait)
	defer cancel()
	return c.depth.Get(wait, symbol, limit)
}

func (c *Client) fetchRESTBook(ctx context.Context, symbol string, limit int) (*domain.RawOrderBook, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	params := url.Values{}
	params.Set("category", "spot")
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(limit))
	body, err := c.get(ctx, "/v5/market/orderbook", params)
	if err != nil {
		return nil, err
	}
	var resp bybitRESTBookResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("%w: decode bybit orderbook: %v", domain.ErrUpstream, err)
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
		return nil, fmt.Errorf("%w: empty order book for %s", domain.ErrNotFound, symbol)
	}
	return domain.RawBookFromDepthLevels(symbol, resp.Result.UpdateID, bids, asks), nil
}

type bybitRESTBookResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		UpdateID int64      `json:"u"`
		Bids     [][]string `json:"b"`
		Asks     [][]string `json:"a"`
	} `json:"result"`
}
