package okx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const swapTradeLimit = 500

// GetRecentPrints returns the newest OKX USDT-swap aggressive fills.
// Size is contracts; notional is contracts × ctVal × price.
func (c *Client) GetRecentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	return c.recentPrints(ctx, symbol)
}

// GetTakerFlow aggregates those prints into 5m / 1h / 4h windows.
// The public trade list is only the newest 500 fills, so longer windows
// stay incomplete until that list covers them.
func (c *Client) GetTakerFlow(ctx context.Context, symbol string) (*domain.TakerVenueFlow, error) {
	prints, err := c.recentPrints(ctx, symbol)
	if err != nil {
		return nil, err
	}
	book := domain.NewTakerBook()
	for _, p := range prints {
		book.Record(p)
	}
	out := book.Snapshot(domain.ExchangeOKX, symbol)
	return &out, nil
}

// GetTakerBuckets returns 1-minute buy/sell bars from the same recent prints.
func (c *Client) GetTakerBuckets(ctx context.Context, symbol string) ([]domain.TakerBucket, error) {
	prints, err := c.recentPrints(ctx, symbol)
	if err != nil {
		return nil, err
	}
	book := domain.NewTakerBook()
	for _, p := range prints {
		book.Record(p)
	}
	return book.Buckets(domain.ExchangeOKX, symbol), nil
}

func (c *Client) recentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	inst := SwapInstID(symbol)
	if inst == "" {
		return nil, fmt.Errorf("%w: okx swap symbol", domain.ErrInvalidArgument)
	}
	ctVal := c.contractValue(ctx, inst)
	if ctVal <= 0 {
		return nil, fmt.Errorf("%w: okx contract size for %s", domain.ErrUpstream, inst)
	}
	q := url.Values{}
	q.Set("instId", inst)
	q.Set("limit", strconv.Itoa(swapTradeLimit))
	body, err := c.get(ctx, "/api/v5/market/trades", q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Side string `json:"side"`
		Px   string `json:"px"`
		Sz   string `json:"sz"`
		Ts   string `json:"ts"`
	}
	if err := decodeOKX(body, &rows); err != nil {
		return nil, err
	}
	out := make([]domain.TakerPrint, 0, len(rows))
	for _, row := range rows {
		px, ok := parseFloat(row.Px)
		if !ok || px <= 0 {
			continue
		}
		contracts, ok := parseFloat(row.Sz)
		if !ok || contracts <= 0 {
			continue
		}
		at, ok := parseMillis(row.Ts)
		if !ok {
			continue
		}
		side := domain.TakerSideBuy
		if row.Side == "sell" {
			side = domain.TakerSideSell
		} else if row.Side != "buy" {
			continue
		}
		qty := contracts * ctVal
		out = append(out, domain.TakerPrint{
			Exchange: domain.ExchangeOKX,
			Symbol:   symbol,
			Side:     side,
			Price:    px,
			Quantity: qty,
			Notional: qty * px,
			Time:     at,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: okx trades for %s", domain.ErrNotFound, symbol)
	}
	return out, nil
}
