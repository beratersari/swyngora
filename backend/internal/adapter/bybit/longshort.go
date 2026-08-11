package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLongShortSeries loads Bybit linear account long/short ratio (5min).
func (c *Client) GetLongShortSeries(ctx context.Context, symbol string, limit int) (*domain.LongShortSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
	}
	limit = domain.ClampLongShortHistoryLimit(limit)
	key := symbol + "|" + strconv.Itoa(limit)
	if cached, ok := c.lsCache.Get(key); ok && cached != nil {
		return cached, nil
	}
	v, err, _ := c.lsSF.Do(key, func() (any, error) {
		if cached, ok := c.lsCache.Get(key); ok && cached != nil {
			return cached, nil
		}
		ser, err := c.fetchLongShort(ctx, symbol, limit)
		if err != nil {
			return nil, err
		}
		c.lsCache.Set(key, ser)
		return ser, nil
	})
	if err != nil {
		return nil, err
	}
	ser, _ := v.(*domain.LongShortSeries)
	return ser, nil
}

func (c *Client) fetchLongShort(ctx context.Context, symbol string, limit int) (*domain.LongShortSeries, error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	q.Set("period", "5min")
	q.Set("limit", strconv.Itoa(limit+1))
	body, err := c.get(ctx, "/v5/market/account-ratio", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				BuyRatio  string `json:"buyRatio"`
				SellRatio string `json:"sellRatio"`
				Timestamp string `json:"timestamp"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: long short decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return nil, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	pts := make([]domain.LongShortPoint, 0, len(raw.Result.List))
	for _, row := range raw.Result.List {
		longShare, err1 := strconv.ParseFloat(strings.TrimSpace(row.BuyRatio), 64)
		shortShare, err2 := strconv.ParseFloat(strings.TrimSpace(row.SellRatio), 64)
		ms, err3 := strconv.ParseInt(strings.TrimSpace(row.Timestamp), 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || ms <= 0 {
			continue
		}
		p := domain.NormalizeLongShortPoint(domain.LongShortPoint{
			Time:       time.UnixMilli(ms).UTC(),
			LongShare:  longShare,
			ShortShare: shortShare,
		})
		if p.Ratio <= 0 {
			continue
		}
		pts = append(pts, p)
	}
	pts = domain.SortLongShortNewestFirst(pts)
	if len(pts) == 0 {
		return nil, fmt.Errorf("%w: long short ratio", domain.ErrNotFound)
	}
	return &domain.LongShortSeries{
		Exchange: domain.ExchangeBybit,
		Symbol:   symbol,
		Kind:     domain.LongShortKindAccounts,
		Period:   domain.LongShortPeriod5m,
		Current:  pts[0],
		History:  pts[1:],
	}, nil
}
