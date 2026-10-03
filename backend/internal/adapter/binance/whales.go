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

// GetRecentPrints loads the newest Binance USD-M aggressive fills.
// The live tape wins once it has prints; otherwise this is REST aggTrades.
func (c *Client) GetRecentPrints(ctx context.Context, symbol string) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	_ = c.ensureMeasuredTape(ctx, symbol)
	if c.tape != nil {
		if got := c.tape.Prints(domain.ExchangeBinance, symbol); len(got) > 0 {
			return got, nil
		}
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
	return c.decodeAggTrades(ctx, symbol, q, false)
}

// ListSpotPrints pages spot aggTrades in [from, to] (inclusive).
func (c *Client) ListSpotPrints(ctx context.Context, symbol string, from, to time.Time) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if from.IsZero() || to.IsZero() || !to.After(from) {
		q := url.Values{}
		q.Set("symbol", symbol)
		q.Set("limit", "1000")
		return c.decodeAggTrades(ctx, symbol, q, true)
	}
	from, to = from.UTC(), to.UTC()
	start := from.UnixMilli()
	end := to.UnixMilli()
	var out []domain.TakerPrint
	for page := 0; page < domain.PrintSizeMaxPages(); page++ {
		if start > end {
			break
		}
		q := url.Values{}
		q.Set("symbol", symbol)
		q.Set("limit", "1000")
		q.Set("startTime", strconv.FormatInt(start, 10))
		q.Set("endTime", strconv.FormatInt(end, 10))
		chunk, err := c.decodeAggTrades(ctx, symbol, q, true)
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}
		out = append(out, chunk...)
		last := chunk[len(chunk)-1].Time.UnixMilli()
		if last < start {
			break
		}
		start = last + 1
		if len(chunk) < 1000 {
			break
		}
	}
	return domain.FilterPrintsInRange(out, from, to), nil
}

// ListPrints pages USD-M aggTrades in [from, to] (inclusive).
func (c *Client) ListPrints(ctx context.Context, symbol string, from, to time.Time) ([]domain.TakerPrint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if from.IsZero() || to.IsZero() || !to.After(from) {
		return c.GetRecentPrints(ctx, symbol)
	}
	from, to = from.UTC(), to.UTC()
	if c.tape != nil && c.tape.Covers(domain.ExchangeBinance, symbol, from) {
		return domain.FilterPrintsInRange(c.tape.Prints(domain.ExchangeBinance, symbol), from, to), nil
	}
	key := "range:" + symbol
	if c.printCache != nil {
		if hit, ok := c.printCache.Get(key); ok {
			return domain.FilterPrintsInRange(append([]domain.TakerPrint(nil), hit...), from, to), nil
		}
	}
	v, err, _ := c.printSF.Do(key, func() (any, error) {
		if c.printCache != nil {
			if hit, ok := c.printCache.Get(key); ok {
				return hit, nil
			}
		}
		got, err := c.fetchAggTradesRange(ctx, symbol, from, to)
		if err != nil {
			return nil, err
		}
		if c.printCache != nil {
			c.printCache.Set(key, got)
		}
		return got, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.([]domain.TakerPrint)
	return domain.FilterPrintsInRange(append([]domain.TakerPrint(nil), got...), from, to), nil
}

func (c *Client) fetchAggTradesRange(ctx context.Context, symbol string, from, to time.Time) ([]domain.TakerPrint, error) {
	start := from.UnixMilli()
	end := to.UnixMilli()
	var out []domain.TakerPrint
	for page := 0; page < domain.PrintSizeMaxPages(); page++ {
		if start > end {
			break
		}
		q := url.Values{}
		q.Set("symbol", symbol)
		q.Set("limit", "1000")
		q.Set("startTime", strconv.FormatInt(start, 10))
		q.Set("endTime", strconv.FormatInt(end, 10))
		chunk, err := c.decodeAggTrades(ctx, symbol, q, false)
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}
		out = append(out, chunk...)
		last := chunk[len(chunk)-1].Time.UnixMilli()
		if last < start {
			break
		}
		start = last + 1
		if len(chunk) < 1000 {
			break
		}
	}
	return out, nil
}

func (c *Client) decodeAggTrades(ctx context.Context, symbol string, q url.Values, spot bool) ([]domain.TakerPrint, error) {
	var (
		body []byte
		err  error
	)
	if spot {
		body, err = c.get(ctx, "/api/v3/aggTrades", q)
	} else {
		body, err = c.getFutures(ctx, "/fapi/v1/aggTrades", q)
	}
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
