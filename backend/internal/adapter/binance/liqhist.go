package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	liqHistLimit    = 1000
	liqHistMaxPages = 20
)

// ListLiquidationHistory loads USD-M force orders for a disconnect window.
// Venue-wide first (no symbol); falls back to per-symbol when the API requires it.
// CoveredFrom/To is only the stretch the pages actually reached.
// A walk that stops at the page cap leaves the earlier gap in place.
func (c *Client) ListLiquidationHistory(ctx context.Context, q domain.LiquidationHistoryQuery) (domain.LiquidationHistoryResult, error) {
	var out domain.LiquidationHistoryResult
	q, ok := domain.NormalizeHistoryQuery(q)
	if !ok || c == nil {
		return out, nil
	}
	if q.Exchange != "" && q.Exchange != domain.ExchangeBinance {
		return out, fmt.Errorf("%w: binance history for %s", domain.ErrInvalidArgument, q.Exchange)
	}
	events, reached, oldest, err := c.fetchForceOrdersRange(ctx, "", q.From, q.To)
	if err == nil {
		out.Events = events
		if from, to, ok := domain.HistoryCoveredSpan(q.From, q.To, oldest, reached); ok {
			out.CoveredFrom, out.CoveredTo = from, to
		}
		return out, nil
	}
	if !forceOrderNeedsSymbol(err) && !errors.Is(err, domain.ErrInvalidArgument) && !errors.Is(err, domain.ErrNotFound) {
		return out, err
	}
	if len(q.Symbols) == 0 {
		return out, nil
	}
	var all []domain.LiquidationEvent
	var spans []domain.LiquidationHistoryResult
	for _, sym := range q.Symbols {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		ev, reached, oldest, err := c.fetchForceOrdersRange(ctx, sym, q.From, q.To)
		if err != nil {
			out.Events = all
			return out, nil
		}
		all = append(all, ev...)
		span := domain.LiquidationHistoryResult{}
		if from, to, ok := domain.HistoryCoveredSpan(q.From, q.To, oldest, reached); ok {
			span.CoveredFrom, span.CoveredTo = from, to
		}
		spans = append(spans, span)
	}
	out.Events = all
	if from, to, ok := domain.MergeHistoryCoverage(spans); ok {
		out.CoveredFrom, out.CoveredTo = from, to
	}
	return out, nil
}

func (c *Client) fetchForceOrdersRange(ctx context.Context, symbol string, from, to time.Time) ([]domain.LiquidationEvent, bool, time.Time, error) {
	end := to
	var out []domain.LiquidationEvent
	for page := 0; page < liqHistMaxPages; page++ {
		q := url.Values{}
		if symbol != "" {
			q.Set("symbol", symbol)
		}
		q.Set("startTime", strconv.FormatInt(from.UTC().UnixMilli(), 10))
		q.Set("endTime", strconv.FormatInt(end.UTC().UnixMilli(), 10))
		q.Set("limit", strconv.Itoa(liqHistLimit))
		body, err := c.getFutures(ctx, "/fapi/v1/allForceOrders", q)
		if err != nil {
			return out, false, time.Time{}, err
		}
		batch, err := parseForceOrderHistory(body)
		if err != nil {
			return out, false, time.Time{}, err
		}
		if len(batch) == 0 {
			return out, true, domain.OldestLiquidationTime(out), nil
		}
		out = append(out, batch...)
		oldest := domain.OldestLiquidationTime(batch)
		if !oldest.After(from) || len(batch) < liqHistLimit {
			return out, true, domain.OldestLiquidationTime(out), nil
		}
		end = oldest.Add(-time.Millisecond)
		if !end.After(from) {
			return out, true, domain.OldestLiquidationTime(out), nil
		}
	}
	return out, false, domain.OldestLiquidationTime(out), nil
}

type binanceForceOrderHist struct {
	Symbol       string `json:"symbol"`
	Price        string `json:"price"`
	OrigQty      string `json:"origQty"`
	ExecutedQty  string `json:"executedQty"`
	AveragePrice string `json:"averagePrice"`
	Side         string `json:"side"`
	Time         int64  `json:"time"`
}

func parseForceOrderHistory(body []byte) ([]domain.LiquidationEvent, error) {
	var rows []binanceForceOrderHist
	if err := json.Unmarshal(body, &rows); err != nil {
		var er struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if json.Unmarshal(body, &er) == nil && er.Msg != "" {
			return nil, mapBinanceError(er.Code, er.Msg)
		}
		return nil, fmt.Errorf("%w: force-order history decode: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.LiquidationEvent, 0, len(rows))
	for _, r := range rows {
		ev, ok := forceOrderHistToEvent(r)
		if ok {
			out = append(out, ev)
		}
	}
	return out, nil
}

func forceOrderNeedsSymbol(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "symbol") && (strings.Contains(s, "mandatory") || strings.Contains(s, "required") || strings.Contains(s, "not sent"))
}

func forceOrderHistToEvent(r binanceForceOrderHist) (domain.LiquidationEvent, bool) {
	side, err := domain.LiquidationSideFromBinanceOrder(r.Side)
	if err != nil {
		return domain.LiquidationEvent{}, false
	}
	px := parseLiqFloat(r.AveragePrice)
	if px <= 0 {
		px = parseLiqFloat(r.Price)
	}
	qty := parseLiqFloat(r.ExecutedQty)
	if qty <= 0 {
		qty = parseLiqFloat(r.OrigQty)
	}
	if px <= 0 || qty <= 0 || r.Time <= 0 || r.Symbol == "" {
		return domain.LiquidationEvent{}, false
	}
	return domain.LiquidationEvent{
		Exchange: domain.ExchangeBinance,
		Symbol:   r.Symbol,
		Side:     side,
		Price:    px,
		Quantity: qty,
		Notional: px * qty,
		Time:     time.UnixMilli(r.Time).UTC(),
	}, true
}
