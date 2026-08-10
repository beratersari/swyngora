package bybit

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOrderBook returns the live local Bybit spot book (orderbook.200 websocket).
func (c *Client) GetOrderBook(ctx context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	symbol := strings.ToUpper(strings.TrimSpace(q.Symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	limit := domain.ClampOrderBookRawLimit(q.Limit)
	c.ensureDepth()
	if c.depth == nil {
		return nil, fmt.Errorf("%w: bybit depth hub not configured", domain.ErrUpstream)
	}
	wait, cancel := context.WithTimeout(ctx, c.depthWait)
	defer cancel()
	return c.depth.Get(wait, symbol, limit)
}
