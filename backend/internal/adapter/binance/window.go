package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const windowTickerBatch = 100

// GetWindowChanges loads Binance rolling-window percent changes (1h / 4h / 24h).
func (c *Client) GetWindowChanges(ctx context.Context, window string, symbols []string) ([]domain.WindowChange, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	size, err := mapBreadthWindow(window)
	if err != nil {
		return nil, err
	}
	syms := uniqueSymbols(symbols)
	if len(syms) == 0 {
		return nil, nil
	}
	key := size + "|" + strings.Join(syms, ",")
	if c.windowCache != nil {
		if hit, ok := c.windowCache.Get(key); ok {
			return append([]domain.WindowChange(nil), hit...), nil
		}
	}
	v, err, _ := c.windowSF.Do(key, func() (any, error) {
		if c.windowCache != nil {
			if hit, ok := c.windowCache.Get(key); ok {
				return hit, nil
			}
		}
		out := make([]domain.WindowChange, 0, len(syms))
		for i := 0; i < len(syms); i += windowTickerBatch {
			j := i + windowTickerBatch
			if j > len(syms) {
				j = len(syms)
			}
			part, err := c.fetchWindowTickers(ctx, size, syms[i:j])
			if err != nil {
				return nil, err
			}
			out = append(out, part...)
		}
		if c.windowCache != nil {
			c.windowCache.Set(key, out)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.([]domain.WindowChange)
	return append([]domain.WindowChange(nil), got...), nil
}

func mapBreadthWindow(window string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(window)) {
	case domain.BreadthWindow1h:
		return "1h", nil
	case domain.BreadthWindow4h:
		return "4h", nil
	case domain.BreadthWindow24h:
		return "1d", nil
	default:
		return "", fmt.Errorf("%w: window must be 1h, 4h, or 24h", domain.ErrInvalidArgument)
	}
}

func uniqueSymbols(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = normalizeSymbol(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func (c *Client) fetchWindowTickers(ctx context.Context, windowSize string, symbols []string) ([]domain.WindowChange, error) {
	rawSyms, err := json.Marshal(symbols)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("symbols", string(rawSyms))
	q.Set("windowSize", windowSize)
	q.Set("type", "MINI")
	body, err := c.get(ctx, "/api/v3/ticker", q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Symbol             string `json:"symbol"`
		PriceChangePercent string `json:"priceChangePercent"`
		Code               int    `json:"code"`
		Msg                string `json:"msg"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		var one struct {
			Symbol             string `json:"symbol"`
			PriceChangePercent string `json:"priceChangePercent"`
			Code               int    `json:"code"`
			Msg                string `json:"msg"`
		}
		if json.Unmarshal(body, &one) != nil {
			return nil, fmt.Errorf("%w: window ticker decode: %v", domain.ErrUpstream, err)
		}
		if one.Code != 0 && one.Msg != "" {
			return nil, mapBinanceError(one.Code, one.Msg)
		}
		pct, ok := domain.ParseChangePct(one.PriceChangePercent)
		if !ok {
			return nil, nil
		}
		return []domain.WindowChange{{Symbol: one.Symbol, ChangePct: pct}}, nil
	}
	out := make([]domain.WindowChange, 0, len(rows))
	for _, r := range rows {
		if r.Code != 0 && r.Msg != "" {
			return nil, mapBinanceError(r.Code, r.Msg)
		}
		pct, ok := domain.ParseChangePct(r.PriceChangePercent)
		if !ok || r.Symbol == "" {
			continue
		}
		out = append(out, domain.WindowChange{Symbol: r.Symbol, ChangePct: pct})
	}
	return out, nil
}
