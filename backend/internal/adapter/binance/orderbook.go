package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOrderBook returns the live local Binance book (WebSocket + snapshot).
// It never serves a book after a gap or disconnect; those states wait for resync.
// SnapshotOnly skips the hub and uses REST depth so many pairs can be sampled.
func (c *Client) GetOrderBook(ctx context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	symbol := normalizeSymbol(q.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	limit := domain.ClampOrderBookRawLimit(q.Limit)
	if q.SnapshotOnly {
		lastID, bids, asks, err := c.fetchDepthSnapshot(ctx, symbol, limit)
		if err != nil {
			return nil, err
		}
		return domain.RawBookFromDepthLevels(symbol, lastID, bids, asks), nil
	}
	c.ensureDepth()
	if c.depth == nil {
		return nil, fmt.Errorf("%w: binance depth hub not configured", domain.ErrUpstream)
	}
	wait, cancel := context.WithTimeout(ctx, c.depthWait)
	defer cancel()
	return c.depth.Get(wait, symbol, limit)
}

// fetchDepthSnapshot pulls a REST snapshot used only to seed / resync the local book.
func (c *Client) fetchDepthSnapshot(ctx context.Context, symbol string, limit int) (int64, []domain.DepthLevel, []domain.DepthLevel, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return 0, nil, nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if limit <= 0 {
		limit = defaultDepthSnapshotLim
	}
	limit = snapBinanceDepthLimit(limit)
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(limit))
	body, err := c.get(ctx, "/api/v3/depth", params)
	if err != nil {
		return 0, nil, nil, err
	}
	var resp binanceDepthResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, nil, nil, fmt.Errorf("%w: decode depth: %v", domain.ErrUpstream, err)
	}
	if resp.Code != 0 && resp.Msg != "" {
		return 0, nil, nil, mapBinanceError(resp.Code, resp.Msg)
	}
	_, bidsD := parseSideKeep(resp.Bids)
	_, asksD := parseSideKeep(resp.Asks)
	if len(bidsD) == 0 && len(asksD) == 0 {
		return 0, nil, nil, fmt.Errorf("%w: empty order book for %s", domain.ErrNotFound, symbol)
	}
	return resp.LastUpdateID, bidsD, asksD, nil
}

type binanceDepthResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
	Code         int        `json:"code"`
	Msg          string     `json:"msg"`
}

func parseSide(rows [][]string) []domain.PriceLevel {
	out, _ := parseSideKeep(rows)
	return out
}

func parseSideKeep(rows [][]string) ([]domain.PriceLevel, []domain.DepthLevel) {
	floats := make([]domain.PriceLevel, 0, len(rows))
	depths := make([]domain.DepthLevel, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		lv, ok := domain.ParsePriceQty(row[0], row[1])
		if !ok {
			continue
		}
		floats = append(floats, lv)
		depths = append(depths, domain.DepthLevel{Price: row[0], Quantity: lv.Quantity})
	}
	return floats, depths
}

func snapBinanceDepthLimit(n int) int {
	// Allowed: 5, 10, 20, 50, 100, 500, 1000, 5000.
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
	case n <= 500:
		return 500
	default:
		return 1000
	}
}
