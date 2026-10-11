package okx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// okxLiqHistLimit is the public liquidation-orders page size. The route has
// no start or end, so a full page is the newest rows only.
const okxLiqHistLimit = 100

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
	var spans []domain.LiquidationHistoryResult
	for i, sym := range q.Symbols {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		raw, err := c.fetchRecentLiquidations(ctx, sym)
		if err != nil {
			if i == 0 && historyUnavailable(err) {
				return out, nil
			}
			out.Events = all
			return out, nil
		}
		oldest := domain.OldestLiquidationTime(raw)
		newest := domain.NewestLiquidationTime(raw)
		reached := okxHistoryReachedStart(raw, oldest, newest, q.From)
		for _, e := range raw {
			if e.Time.Before(q.From) || !e.Time.Before(q.To.Add(time.Millisecond)) {
				continue
			}
			all = append(all, e)
		}
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

// okxHistoryReachedStart reports whether the newest public page contains
// every print from reqFrom forward. An empty page does not. A page whose
// newest print is still before reqFrom has nothing inside the window.
func okxHistoryReachedStart(raw []domain.LiquidationEvent, oldest, newest, reqFrom time.Time) bool {
	if len(raw) == 0 {
		return false
	}
	if !newest.After(reqFrom) || !oldest.After(reqFrom) {
		return true
	}
	return false
}

func historyUnavailable(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "404") || strings.Contains(s, "not found") || strings.Contains(s, "doesn't exist")
}

func (c *Client) fetchRecentLiquidations(ctx context.Context, symbol string) ([]domain.LiquidationEvent, error) {
	uly := UnderlyingFromSymbol(symbol)
	if uly == "" {
		return nil, nil
	}
	inst := SwapInstID(symbol)
	ctVal := c.contractValue(ctx, inst)
	if ctVal <= 0 {
		// Leave this symbol uncovered. The next pass can store the page
		// once the instrument cache has a positive contract size.
		return nil, nil
	}
	q := url.Values{}
	q.Set("instType", "SWAP")
	q.Set("uly", uly)
	q.Set("state", "filled")
	q.Set("limit", strconv.Itoa(okxLiqHistLimit))
	body, err := c.get(ctx, "/api/v5/public/liquidation-orders", q)
	if err != nil {
		return nil, err
	}
	return parseLiquidationOrders(body, ctVal)
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
	// A missing contract size is not one contract. BTC-USDT-SWAP is 0.01,
	// so treating an unknown size as 1 stores about 100× the notional.
	// The trade tape drops the same print.
	if ctVal <= 0 {
		return domain.LiquidationEvent{}, false
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
