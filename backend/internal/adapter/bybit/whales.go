package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetRecentPrints loads Bybit linear public recent trades.
func (c *Client) GetRecentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
	}
	if c.printCache != nil {
		if hit, ok := c.printCache.Get(symbol); ok {
			return append([]domain.TakerPrint(nil), hit...), nil
		}
	}
	v, err, _ := c.printSF.Do(symbol, func() (any, error) {
		if c.printCache != nil {
			if hit, ok := c.printCache.Get(symbol); ok {
				return hit, nil
			}
		}
		got, err := c.fetchRecentPrints(ctx, symbol)
		if err != nil {
			return nil, err
		}
		if c.printCache != nil {
			c.printCache.Set(symbol, got)
		}
		return got, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.([]domain.TakerPrint)
	return append([]domain.TakerPrint(nil), got...), nil
}

func (c *Client) fetchRecentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	q := url.Values{}
	q.Set("category", "linear")
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
		return nil, fmt.Errorf("%w: recent-trade decode: %v", domain.ErrUpstream, err)
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
