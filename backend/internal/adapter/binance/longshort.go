package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
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

// ListLongShortHistory pages 1-hour account, top-account, and top-position ratios.
func (c *Client) ListLongShortHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.LongShortHistoryPoint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	from, to = from.UTC(), to.UTC()
	if !to.After(from) {
		return nil, fmt.Errorf("%w: to must be after from", domain.ErrInvalidArgument)
	}
	global, err := c.pageLongShort(ctx, symbol, "/futures/data/globalLongShortAccountRatio", from, to)
	if err != nil {
		return nil, err
	}
	topAcc, errA := c.pageLongShort(ctx, symbol, "/futures/data/topLongShortAccountRatio", from, to)
	topPos, errP := c.pageLongShort(ctx, symbol, "/futures/data/topLongShortPositionRatio", from, to)
	accAt := longShortByTime(topAcc)
	posAt := longShortByTime(topPos)
	_ = errA
	_ = errP
	out := make([]domain.LongShortHistoryPoint, 0, len(global))
	for _, p := range global {
		pt := domain.NormalizeLongShortPoint(p)
		row := domain.LongShortHistoryPoint{
			Time: pt.Time, LongShare: pt.LongShare, ShortShare: pt.ShortShare, Ratio: pt.Ratio,
		}
		if acc, ok := nearestLongShort(accAt, pt.Time, time.Hour); ok {
			row.TopAccLongShare = domain.NormalizeLongShortPoint(acc).LongShare
		}
		if pos, ok := nearestLongShort(posAt, pt.Time, time.Hour); ok {
			row.SizeLongShare = domain.NormalizeLongShortPoint(pos).LongShare
		}
		out = append(out, row)
	}
	return out, nil
}

func (c *Client) pageLongShort(ctx context.Context, symbol, path string, from, to time.Time) ([]domain.LongShortPoint, error) {
	// Same newest-page behavior as open-interest history: walk endTime backward.
	const page = 500
	var out []domain.LongShortPoint
	end := to
	for i := 0; i < domain.FuturesHistoryPageBudget(page); i++ {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		q := url.Values{}
		q.Set("symbol", symbol)
		q.Set("period", "1h")
		q.Set("limit", strconv.Itoa(page))
		q.Set("startTime", strconv.FormatInt(from.UnixMilli(), 10))
		q.Set("endTime", strconv.FormatInt(end.UnixMilli(), 10))
		body, err := c.getFutures(ctx, path, q)
		if err != nil {
			return nil, err
		}
		batch, err := decodeLongShortRows(body)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		out = append(out, batch...)
		oldest := batch[0].Time
		for _, p := range batch[1:] {
			if p.Time.Before(oldest) {
				oldest = p.Time
			}
		}
		if !oldest.After(from) || len(batch) < page {
			break
		}
		next := oldest.Add(-time.Millisecond)
		if !next.Before(end) {
			break
		}
		end = next
	}
	return out, nil
}

func decodeLongShortRows(body []byte) ([]domain.LongShortPoint, error) {
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
	sort.SliceStable(pts, func(i, j int) bool { return pts[i].Time.Before(pts[j].Time) })
	return pts, nil
}

func longShortByTime(pts []domain.LongShortPoint) []domain.LongShortPoint {
	return pts
}

func nearestLongShort(pts []domain.LongShortPoint, at time.Time, slack time.Duration) (domain.LongShortPoint, bool) {
	var best domain.LongShortPoint
	found := false
	var bestDt time.Duration
	for _, p := range pts {
		dt := at.Sub(p.Time)
		if dt < 0 {
			dt = -dt
		}
		if dt > slack {
			continue
		}
		if !found || dt < bestDt {
			best = p
			bestDt = dt
			found = true
		}
	}
	return best, found
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
