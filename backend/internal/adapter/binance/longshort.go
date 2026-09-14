package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLongShortSeries loads Binance USD-M global long/short account ratio (5m).
func (c *Client) GetLongShortSeries(ctx context.Context, symbol string, limit int) (*domain.LongShortSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
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
	var (
		global, topAcc, topPos *domain.LongShortSeries
		errG, errA, errP       error
		wg                     sync.WaitGroup
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		global, errG = c.fetchLongShortPath(ctx, symbol, limit, "/futures/data/globalLongShortAccountRatio", domain.LongShortKindAccounts)
	}()
	go func() {
		defer wg.Done()
		topAcc, errA = c.fetchLongShortPath(ctx, symbol, limit, "/futures/data/topLongShortAccountRatio", domain.LongShortKindTopAccounts)
	}()
	go func() {
		defer wg.Done()
		topPos, errP = c.fetchLongShortPath(ctx, symbol, limit, "/futures/data/topLongShortPositionRatio", domain.LongShortKindTopPositions)
	}()
	wg.Wait()
	if errG != nil {
		return nil, errG
	}
	if errA == nil {
		global.TopAccounts = topAcc
	}
	if errP == nil {
		global.TopPositions = topPos
	}
	domain.AttachLongShortSplit(global)
	return global, nil
}

func (c *Client) fetchLongShortPath(ctx context.Context, symbol string, limit int, path, kind string) (*domain.LongShortSeries, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("period", "5m")
	q.Set("limit", strconv.Itoa(limit+1)) // +1 so current can be split off history
	body, err := c.getFutures(ctx, path, q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		LongAccount    string `json:"longAccount"`
		ShortAccount   string `json:"shortAccount"`
		LongShortRatio string `json:"longShortRatio"`
		Timestamp      int64  `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: long short decode: %v", domain.ErrUpstream, err)
	}
	pts := make([]domain.LongShortPoint, 0, len(rows))
	for _, r := range rows {
		p, ok := parseLongShortRow(r.LongAccount, r.ShortAccount, r.LongShortRatio, r.Timestamp)
		if !ok {
			continue
		}
		pts = append(pts, p)
	}
	pts = domain.SortLongShortNewestFirst(pts)
	if len(pts) == 0 {
		return nil, fmt.Errorf("%w: long short ratio", domain.ErrNotFound)
	}
	return &domain.LongShortSeries{
		Exchange: domain.ExchangeBinance,
		Symbol:   symbol,
		Kind:     kind,
		Period:   domain.LongShortPeriod5m,
		Current:  pts[0],
		History:  pts[1:],
	}, nil
}

func parseLongShortRow(longS, shortS, ratioS string, ts int64) (domain.LongShortPoint, bool) {
	longShare, err1 := strconv.ParseFloat(longS, 64)
	shortShare, err2 := strconv.ParseFloat(shortS, 64)
	if err1 != nil || err2 != nil || ts <= 0 {
		return domain.LongShortPoint{}, false
	}
	ratio, err := strconv.ParseFloat(ratioS, 64)
	p := domain.LongShortPoint{
		Time:       time.UnixMilli(ts).UTC(),
		LongShare:  longShare,
		ShortShare: shortShare,
		Ratio:      ratio,
	}
	if err != nil || p.Ratio <= 0 {
		p.Ratio = 0
	}
	p = domain.NormalizeLongShortPoint(p)
	if p.Ratio <= 0 {
		return domain.LongShortPoint{}, false
	}
	return p, true
}
