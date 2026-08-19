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

// SetTakerWatch is called when a symbol should be subscribed on the trade hub.
func (c *Client) SetTakerWatch(fn func(string)) {
	if c != nil {
		c.takerWatch = fn
	}
}

// TakerBook exposes the rolling Bybit taker store (for restore / hub).
func (c *Client) TakerBook() *domain.TakerBook {
	if c == nil {
		return nil
	}
	return c.taker
}

// GetTakerFlow returns collected Bybit linear taker buy/sell (plus a REST seed).
func (c *Client) GetTakerFlow(ctx context.Context, symbol string) (*domain.TakerVenueFlow, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil || c.taker == nil {
		return nil, fmt.Errorf("%w: bybit taker book", domain.ErrUpstream)
	}
	if c.takerWatch != nil {
		c.takerWatch(symbol)
	}
	_ = c.seedRecentTrades(ctx, symbol)
	out := c.taker.Snapshot(domain.ExchangeBybit, symbol)
	return &out, nil
}

// GetTakerBuckets returns collected Bybit 1-minute taker bars (seeded from recent trades).
func (c *Client) GetTakerBuckets(ctx context.Context, symbol string) ([]domain.TakerBucket, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil || c.taker == nil {
		return nil, fmt.Errorf("%w: bybit taker book", domain.ErrUpstream)
	}
	if c.takerWatch != nil {
		c.takerWatch(symbol)
	}
	_ = c.seedRecentTrades(ctx, symbol)
	return c.taker.Buckets(domain.ExchangeBybit, symbol), nil
}

// GetSpotTakerBuckets returns recent Bybit spot aggressive buy/sell bars.
func (c *Client) GetSpotTakerBuckets(ctx context.Context, symbol string) ([]domain.TakerBucket, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	prints, err := c.fetchRecentTrades(ctx, symbol, "spot")
	if err != nil {
		return nil, err
	}
	book := domain.NewTakerBook()
	for _, p := range prints {
		book.Record(p)
	}
	return book.Buckets(domain.ExchangeBybit, symbol), nil
}

func (c *Client) seedRecentTrades(ctx context.Context, symbol string) error {
	prints, err := c.fetchRecentTrades(ctx, symbol, "linear")
	if err != nil {
		return err
	}
	for _, p := range prints {
		if p.Notional > 0 {
			c.taker.Record(p)
		}
	}
	return nil
}

func (c *Client) fetchRecentTrades(ctx context.Context, symbol, category string) ([]domain.TakerPrint, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
	}
	q := url.Values{}
	q.Set("category", category)
	q.Set("symbol", symbol)
	q.Set("limit", "1000")
	body, err := c.get(ctx, "/v5/market/recent-trade", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Side  string `json:"side"`
				Price string `json:"price"`
				Size  string `json:"size"`
				Time  string `json:"time"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if raw.RetCode != 0 {
		return nil, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	out := make([]domain.TakerPrint, 0, len(raw.Result.List))
	for _, row := range raw.Result.List {
		p := parseBybitTaker(domain.ExchangeBybit, symbol, row.Side, row.Price, row.Size, row.Time)
		if p.Notional > 0 {
			out = append(out, p)
		}
	}
	return out, nil
}

func parseBybitTaker(ex domain.Exchange, symbol, side, price, size, ts string) domain.TakerPrint {
	out := domain.TakerPrint{Exchange: ex, Symbol: domain.NormalizeLiquidationSymbol(symbol)}
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy":
		out.Side = domain.TakerSideBuy
	case "sell":
		out.Side = domain.TakerSideSell
	default:
		return out
	}
	px, err1 := strconv.ParseFloat(strings.TrimSpace(price), 64)
	qty, err2 := strconv.ParseFloat(strings.TrimSpace(size), 64)
	ms, err3 := strconv.ParseInt(strings.TrimSpace(ts), 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || px <= 0 || qty <= 0 {
		return out
	}
	out.Price = px
	out.Quantity = qty
	out.Notional = px * qty
	out.Time = time.UnixMilli(ms).UTC()
	return out
}
