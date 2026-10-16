package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SetTakerWatch subscribes a symbol on the public trade hub.
func (c *Client) SetTakerWatch(fn func(string)) {
	if c != nil {
		c.takerWatch = fn
	}
}

// TakerBook is the rolling measured tape.
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
	if from.IsZero() || !c.tape.Covers(domain.ExchangeHyperliquid, symbol, from) {
		return nil, false
	}
	return domain.FilterPrintsInRange(c.tape.Prints(domain.ExchangeHyperliquid, symbol), from, to), true
}

// GetRecentPrints returns aggressive fills already on the tape.
// A tape that has not started yet is seeded from the newest public trades.
// There is no paged trade history, so a window the tape does not cover stays uncovered.
func (c *Client) GetRecentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: hyperliquid client", domain.ErrUpstream)
	}
	if err := c.ensureMeasuredTape(ctx, symbol); err != nil && len(c.tapePrints(symbol)) == 0 {
		return nil, err
	}
	if got := c.tapePrints(symbol); len(got) > 0 {
		return got, nil
	}
	return c.recentPrints(ctx, symbol)
}

// GetTakerFlow aggregates measured prints into 5m / 1h / 4h windows.
// A window this process has not recorded stays incomplete.
func (c *Client) GetTakerFlow(ctx context.Context, symbol string) (*domain.TakerVenueFlow, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil || c.taker == nil {
		return nil, fmt.Errorf("%w: hyperliquid taker book", domain.ErrUpstream)
	}
	if err := c.ensureMeasuredTape(ctx, symbol); err != nil && len(c.taker.Buckets(domain.ExchangeHyperliquid, symbol)) == 0 {
		return nil, err
	}
	out := c.taker.Snapshot(domain.ExchangeHyperliquid, symbol)
	return &out, nil
}

// GetTakerBuckets returns measured 1-minute buy/sell bars.
func (c *Client) GetTakerBuckets(ctx context.Context, symbol string) ([]domain.TakerBucket, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil || c.taker == nil {
		return nil, fmt.Errorf("%w: hyperliquid taker book", domain.ErrUpstream)
	}
	if err := c.ensureMeasuredTape(ctx, symbol); err != nil && len(c.taker.Buckets(domain.ExchangeHyperliquid, symbol)) == 0 {
		return nil, err
	}
	return c.taker.Buckets(domain.ExchangeHyperliquid, symbol), nil
}

func (c *Client) tapePrints(symbol string) []domain.TakerPrint {
	if c == nil || c.tape == nil {
		return nil
	}
	return c.tape.Prints(domain.ExchangeHyperliquid, symbol)
}

func (c *Client) ensureMeasuredTape(ctx context.Context, symbol string) error {
	if c == nil {
		return fmt.Errorf("%w: hyperliquid client", domain.ErrUpstream)
	}
	if c.takerWatch != nil {
		c.takerWatch(symbol)
	}
	if c.taker != nil && len(c.taker.Buckets(domain.ExchangeHyperliquid, symbol)) > 0 {
		return nil
	}
	_, err, _ := c.seedSF.Do("tape:"+symbol, func() (any, error) {
		if c.taker != nil && len(c.taker.Buckets(domain.ExchangeHyperliquid, symbol)) > 0 {
			return nil, nil
		}
		prints, err := c.recentPrints(ctx, symbol)
		if err != nil {
			return nil, err
		}
		for _, p := range prints {
			domain.RecordMeasuredPrint(c.taker, c.tape, p)
		}
		return nil, nil
	})
	return err
}

func (c *Client) recentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	coin, ok := Coin(symbol)
	if !ok {
		return nil, fmt.Errorf("%w: hyperliquid symbol", domain.ErrInvalidArgument)
	}
	if c == nil {
		return nil, fmt.Errorf("%w: hyperliquid client", domain.ErrUpstream)
	}
	if err := c.hasCoin(ctx, coin); err != nil {
		return nil, err
	}
	body, err := c.post(ctx, map[string]string{"type": "recentTrades", "coin": coin})
	if err != nil {
		return nil, err
	}
	var rows []hlTrade
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: hyperliquid trades", domain.ErrUpstream)
	}
	out := make([]domain.TakerPrint, 0, len(rows))
	for _, row := range rows {
		p, ok := printFromTrade(row)
		if ok {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: hyperliquid trades for %s", domain.ErrNotFound, coin)
	}
	return out, nil
}

type hlTrade struct {
	Coin string          `json:"coin"`
	Side string          `json:"side"`
	Px   string          `json:"px"`
	Sz   string          `json:"sz"`
	Time int64           `json:"time"`
	Tid  json.RawMessage `json:"tid"`
}

func printFromTrade(row hlTrade) (domain.TakerPrint, bool) {
	if !perpCoin(row.Coin) {
		return domain.TakerPrint{}, false
	}
	side, ok := takerSide(row.Side)
	if !ok || row.Time <= 0 {
		return domain.TakerPrint{}, false
	}
	px, ok := parsePositive(row.Px)
	if !ok {
		return domain.TakerPrint{}, false
	}
	qty, ok := parsePositive(row.Sz)
	if !ok {
		return domain.TakerPrint{}, false
	}
	return domain.TakerPrint{
		Exchange: domain.ExchangeHyperliquid,
		Symbol:   row.Coin + "USDT",
		Side:     side,
		Price:    px,
		Quantity: qty,
		Notional: qty * px,
		Time:     time.UnixMilli(row.Time).UTC(),
	}, true
}

func takerSide(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "B", "BUY":
		return domain.TakerSideBuy, true
	case "A", "SELL":
		return domain.TakerSideSell, true
	default:
		return "", false
	}
}

// perpCoin keeps USDT-perp names. Spot pairs use @index or BASE/USDC.
func perpCoin(coin string) bool {
	coin = strings.TrimSpace(coin)
	if coin == "" || strings.Contains(coin, "@") || strings.Contains(coin, "/") {
		return false
	}
	return true
}
