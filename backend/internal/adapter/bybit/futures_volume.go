package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ListFuturesVolumeBars loads Bybit linear 5m quote-volume klines (turnover).
func (c *Client) ListFuturesVolumeBars(ctx context.Context, symbol string) ([]domain.FuturesVolumeBar, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
	}
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	q.Set("interval", "5")
	q.Set("limit", strconv.Itoa(domain.FuturesVolumeBars))
	body, err := c.get(ctx, "/v5/market/kline", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List [][]string `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: futures volume decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return nil, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	out := make([]domain.FuturesVolumeBar, 0, len(raw.Result.List))
	for _, row := range raw.Result.List {
		if len(row) < 7 {
			continue
		}
		ms, err1 := strconv.ParseInt(row[0], 10, 64)
		vol, err2 := strconv.ParseFloat(row[6], 64)
		if err1 != nil || err2 != nil || ms <= 0 || vol <= 0 {
			continue
		}
		out = append(out, domain.FuturesVolumeBar{Time: time.UnixMilli(ms).UTC(), QuoteVolume: vol})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: futures volume", domain.ErrNotFound)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
