package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetRecentPrints loads the newest Binance USD-M aggTrades (taker side).
func (c *Client) GetRecentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
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
		got, err := c.fetchAggTrades(ctx, symbol)
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

func (c *Client) fetchAggTrades(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("limit", "1000")
	body, err := c.getFutures(ctx, "/fapi/v1/aggTrades", q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Price    string `json:"p"`
		Qty      string `json:"q"`
		Time     int64  `json:"T"`
		MakerBuy bool   `json:"m"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: aggTrades decode: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.TakerPrint, 0, len(rows))
	for _, r := range rows {
		px, e1 := strconv.ParseFloat(r.Price, 64)
		qty, e2 := strconv.ParseFloat(r.Qty, 64)
		if e1 != nil || e2 != nil || px <= 0 || qty <= 0 || r.Time <= 0 {
			continue
		}
		side := domain.TakerSideBuy
		if r.MakerBuy {
			side = domain.TakerSideSell
		}
		out = append(out, domain.TakerPrint{
			Exchange: domain.ExchangeBinance, Symbol: symbol, Side: side,
			Price: px, Quantity: qty, Notional: px * qty,
			Time: time.UnixMilli(r.Time).UTC(),
		})
	}
	return out, nil
}
