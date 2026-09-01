package bybit

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

// ListLiquidationHistory loads linear liquidations for a disconnect window.
// Bybit has no official all-market REST tape; each watched symbol is queried
// separately. A missing endpoint (404) leaves the gap uncovered.
func (c *Client) ListLiquidationHistory(ctx context.Context, q domain.LiquidationHistoryQuery) (domain.LiquidationHistoryResult, error) {
	var out domain.LiquidationHistoryResult
	q, ok := domain.NormalizeHistoryQuery(q)
	if !ok || c == nil {
		return out, nil
	}
	if q.Exchange != "" && q.Exchange != domain.ExchangeBybit {
		return out, fmt.Errorf("%w: bybit history for %s", domain.ErrInvalidArgument, q.Exchange)
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

func (c *Client) fetchRecentLiquidations(ctx context.Context, symbol string, from, to time.Time) ([]domain.LiquidationEvent, error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	q.Set("startTime", strconv.FormatInt(from.UTC().UnixMilli(), 10))
	q.Set("endTime", strconv.FormatInt(to.UTC().UnixMilli(), 10))
	q.Set("limit", "1000")
	body, err := c.get(ctx, "/v5/market/recent-liquidation", q)
	if err != nil {
		return nil, err
	}
	return parseRecentLiquidation(body)
}

type bybitLiqHistMsg struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			T     int64  `json:"T"`
			S     string `json:"s"`
			Side  string `json:"S"`
			V     string `json:"v"`
			P     string `json:"p"`
			Time  int64  `json:"time"`
			Sym   string `json:"symbol"`
			Side2 string `json:"side"`
			Size  string `json:"size"`
			Px    string `json:"price"`
		} `json:"list"`
	} `json:"result"`
}

func parseRecentLiquidation(body []byte) ([]domain.LiquidationEvent, error) {
	var msg bybitLiqHistMsg
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("%w: liquidation history decode: %v", domain.ErrUpstream, err)
	}
	if msg.RetCode != 0 {
		return nil, mapBybitError(msg.RetCode, msg.RetMsg)
	}
	out := make([]domain.LiquidationEvent, 0, len(msg.Result.List))
	for _, d := range msg.Result.List {
		sideRaw := d.Side
		if sideRaw == "" {
			sideRaw = d.Side2
		}
		side, err := domain.LiquidationSideFromBybit(sideRaw)
		if err != nil {
			continue
		}
		pxStr, qtyStr := d.P, d.V
		if pxStr == "" {
			pxStr = d.Px
		}
		if qtyStr == "" {
			qtyStr = d.Size
		}
		px, err1 := strconv.ParseFloat(strings.TrimSpace(pxStr), 64)
		qty, err2 := strconv.ParseFloat(strings.TrimSpace(qtyStr), 64)
		if err1 != nil || err2 != nil || px <= 0 || qty <= 0 {
			continue
		}
		ts := d.T
		if ts == 0 {
			ts = d.Time
		}
		if ts <= 0 {
			continue
		}
		sym := d.S
		if sym == "" {
			sym = d.Sym
		}
		out = append(out, domain.LiquidationEvent{
			Exchange: domain.ExchangeBybit,
			Symbol:   sym,
			Side:     side,
			Price:    px,
			Quantity: qty,
			Notional: px * qty,
			Time:     time.UnixMilli(ts).UTC(),
		})
	}
	return out, nil
}

func historyUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, domain.ErrNotFound) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "status 404") || strings.Contains(s, "not found")
}
