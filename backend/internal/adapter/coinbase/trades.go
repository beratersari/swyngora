package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ListSpotPrints loads recent Coinbase Exchange trades (maker side flipped to taker).
func (c *Client) ListSpotPrints(ctx context.Context, symbol string, from, to time.Time) ([]domain.TakerPrint, error) {
	symbol = domain.NormalizeSymbol(domain.ExchangeCoinbase, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if c == nil {
		return nil, fmt.Errorf("%w: coinbase client", domain.ErrUpstream)
	}
	body, err := c.get(ctx, c.exchangeURL, "/products/"+symbol+"/trades", nil)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Time  time.Time `json:"time"`
		Price string    `json:"price"`
		Size  string    `json:"size"`
		Side  string    `json:"side"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: trades decode: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.TakerPrint, 0, len(rows))
	for _, r := range rows {
		px, e1 := strconv.ParseFloat(r.Price, 64)
		qty, e2 := strconv.ParseFloat(r.Size, 64)
		if e1 != nil || e2 != nil || px <= 0 || qty <= 0 || r.Time.IsZero() {
			continue
		}
		side := domain.TakerSideBuy
		if r.Side == "buy" {
			// Coinbase `side` is the maker. Maker buy ⇒ taker sell (hits bids).
			side = domain.TakerSideSell
		}
		out = append(out, domain.TakerPrint{
			Exchange: domain.ExchangeCoinbase, Symbol: symbol, Side: side,
			Price: px, Quantity: qty, Notional: px * qty, Time: r.Time.UTC(),
		})
	}
	if from.IsZero() || to.IsZero() {
		return out, nil
	}
	return domain.FilterPrintsInRange(out, from, to), nil
}
