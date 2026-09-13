package okx

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ListLiquidationHistory loads SWAP liquidations for a disconnect window.
func (c *Client) ListLiquidationHistory(ctx context.Context, q domain.LiquidationHistoryQuery) (domain.LiquidationHistoryResult, error) {
	var out domain.LiquidationHistoryResult
	q, ok := domain.NormalizeHistoryQuery(q)
	if !ok || c == nil {
		return out, nil
	}
	if q.Exchange != "" && q.Exchange != domain.ExchangeOKX {
		return out, fmt.Errorf("%w: okx history for %s", domain.ErrInvalidArgument, q.Exchange)
	}
	if len(q.Symbols) == 0 {
		return out, nil
	}
	var all []domain.LiquidationEvent
	for i, sym := range q.Symbols {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		ev, err := c.fetchRecentLiquidations(ctx, sym, q.From, q.To)
		if err != nil {
			if i == 0 && historyUnavailable(err) {
				return out, nil
			}
			out.Events = all
			return out, nil
		}
		all = append(all, ev...)
	}
	out.Events = all
	out.CoveredFrom = q.From
	out.CoveredTo = q.To
	return out, nil
}

func historyUnavailable(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "404") || strings.Contains(s, "not found") || strings.Contains(s, "doesn't exist")
}

func (c *Client) fetchRecentLiquidations(ctx context.Context, symbol string, from, to time.Time) ([]domain.LiquidationEvent, error) {
	uly := UnderlyingFromSymbol(symbol)
	if uly == "" {
		return nil, nil
	}
	inst := SwapInstID(symbol)
	ctVal := c.contractValue(ctx, inst)
	q := url.Values{}
	q.Set("instType", "SWAP")
	q.Set("uly", uly)
	q.Set("state", "filled")
	q.Set("limit", "100")
	body, err := c.get(ctx, "/api/v5/public/liquidation-orders", q)
	if err != nil {
		return nil, err
	}
	evs, err := parseLiquidationOrders(body, ctVal)
	if err != nil {
		return nil, err
	}
	out := evs[:0]
	for _, e := range evs {
		if e.Time.Before(from) || !e.Time.Before(to.Add(time.Millisecond)) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func parseLiquidationOrders(body []byte, ctVal float64) ([]domain.LiquidationEvent, error) {
	var rows []okxLiqFamily
	if err := decodeOKX(body, &rows); err != nil {
		return nil, err
	}
	var out []domain.LiquidationEvent
	for _, fam := range rows {
		inst := fam.InstID
		sym := SymbolFromInstID(inst)
		if sym == "" {
			continue
		}
		for _, d := range fam.Details {
			ev, ok := detailToEvent(sym, d, ctVal)
			if ok {
				out = append(out, ev)
			}
		}
	}
	return out, nil
}

type okxLiqFamily struct {
	InstID  string        `json:"instId"`
	Details []okxLiqPrint `json:"details"`
}

type okxLiqPrint struct {
	BkPx    string `json:"bkPx"`
	Sz      string `json:"sz"`
	Side    string `json:"side"`
	PosSide string `json:"posSide"`
	Ts      string `json:"ts"`
	Time    int64  `json:"time"`
}

func detailToEvent(symbol string, d okxLiqPrint, ctVal float64) (domain.LiquidationEvent, bool) {
	side, err := domain.LiquidationSideFromOKX(d.PosSide, d.Side)
	if err != nil {
		return domain.LiquidationEvent{}, false
	}
	px, ok := parseFloat(d.BkPx)
	if !ok || px <= 0 {
		return domain.LiquidationEvent{}, false
	}
	sz, ok := parseFloat(d.Sz)
	if !ok || sz <= 0 {
		return domain.LiquidationEvent{}, false
	}
	if ctVal <= 0 {
		ctVal = 1
	}
	qty := sz * ctVal
	ts, ok := parseMillis(d.Ts)
	if !ok && d.Time > 0 {
		ts = time.UnixMilli(d.Time).UTC()
		ok = true
	}
	if !ok {
		return domain.LiquidationEvent{}, false
	}
	return domain.LiquidationEvent{
		Exchange: domain.ExchangeOKX,
		Symbol:   symbol,
		Side:     side,
		Price:    px,
		Quantity: qty,
		Notional: qty * px,
		Time:     ts,
	}, true
}
