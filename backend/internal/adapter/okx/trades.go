package okx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const swapTradeLimit = 500

// SetTakerWatch subscribes a symbol on the swap trade hub.
func (c *Client) SetTakerWatch(fn func(string)) {
	if c != nil {
		c.takerWatch = fn
	}
}

// TakerBook is the rolling 1-minute measured tape.
func (c *Client) TakerBook() *domain.TakerBook {
	if c == nil {
		return nil
	}
	return c.taker
}

// PrintTape is the recent individual fills behind that minute book.
func (c *Client) PrintTape() *domain.PrintTape {
	if c == nil {
		return nil
	}
	return c.tape
}

// MemoryPrints returns fills already on the in-memory tape.
// The boolean is false when that tape does not reach from.
func (c *Client) MemoryPrints(symbol string, from, to time.Time) ([]domain.TakerPrint, bool) {
	if c == nil || c.tape == nil {
		return nil, false
	}
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, false
	}
	if from.IsZero() || !c.tape.Covers(domain.ExchangeOKX, symbol, from) {
		return nil, false
	}
	return domain.FilterPrintsInRange(c.tape.Prints(domain.ExchangeOKX, symbol), from, to), true
}

// GetRecentPrints returns the newest OKX USDT-swap aggressive fills.
// The live tape wins once it has prints. Size is contracts × ctVal.
func (c *Client) GetRecentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	if err := c.ensureMeasuredTape(ctx, symbol); err != nil && len(c.tape.Prints(domain.ExchangeOKX, symbol)) == 0 {
		return nil, err
	}
	if c.tape != nil {
		if got := c.tape.Prints(domain.ExchangeOKX, symbol); len(got) > 0 {
			return got, nil
		}
	}
	return c.recentPrints(ctx, symbol)
}

// GetTakerFlow aggregates measured prints into 5m / 1h / 4h windows.
func (c *Client) GetTakerFlow(ctx context.Context, symbol string) (*domain.TakerVenueFlow, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil || c.taker == nil {
		return nil, fmt.Errorf("%w: okx taker book", domain.ErrUpstream)
	}
	if err := c.ensureMeasuredTape(ctx, symbol); err != nil && len(c.taker.Buckets(domain.ExchangeOKX, symbol)) == 0 {
		return nil, err
	}
	out := c.taker.Snapshot(domain.ExchangeOKX, symbol)
	return &out, nil
}

// GetTakerBuckets returns measured 1-minute buy/sell bars.
func (c *Client) GetTakerBuckets(ctx context.Context, symbol string) ([]domain.TakerBucket, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil || c.taker == nil {
		return nil, fmt.Errorf("%w: okx taker book", domain.ErrUpstream)
	}
	if err := c.ensureMeasuredTape(ctx, symbol); err != nil && len(c.taker.Buckets(domain.ExchangeOKX, symbol)) == 0 {
		return nil, err
	}
	return c.taker.Buckets(domain.ExchangeOKX, symbol), nil
}

func (c *Client) ensureMeasuredTape(ctx context.Context, symbol string) error {
	if c == nil {
		return fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	if c.takerWatch != nil {
		c.takerWatch(symbol)
	}
	if c.taker != nil && len(c.taker.Buckets(domain.ExchangeOKX, symbol)) > 0 {
		return nil
	}
	c.seedMu.Lock()
	if c.seeded == nil {
		c.seeded = map[string]struct{}{}
	}
	if _, ok := c.seeded[symbol]; ok {
		c.seedMu.Unlock()
		return nil
	}
	c.seeded[symbol] = struct{}{}
	c.seedMu.Unlock()
	prints, err := c.recentPrints(ctx, symbol)
	if err != nil {
		c.seedMu.Lock()
		delete(c.seeded, symbol)
		c.seedMu.Unlock()
		return err
	}
	for _, p := range prints {
		domain.RecordMeasuredPrint(c.taker, c.tape, p)
	}
	return nil
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
