package okx

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	okxLSAccounts = "/api/v5/rubik/stat/contracts/long-short-account-ratio-contract"
	okxLSTopAcc   = "/api/v5/rubik/stat/contracts/long-short-account-ratio-contract-top-trader"
	okxLSTopPos   = "/api/v5/rubik/stat/contracts/long-short-position-ratio-contract-top-trader"
	okxLSPage     = 100
)

// rubikGap keeps hourly backfill under OKX's 5 requests / 2 seconds per instrument.
var rubikGap = 400 * time.Millisecond

// GetLongShortSeries loads the OKX USDT-swap account ratio plus top-trader
// account and position ratios. Top traders are OKX's top 5% by position value.
func (c *Client) GetLongShortSeries(ctx context.Context, symbol string, limit int) (*domain.LongShortSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
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
	inst := SwapInstID(symbol)
	if inst == "" {
		return nil, fmt.Errorf("%w: okx swap symbol", domain.ErrInvalidArgument)
	}
	global, err := c.fetchRatioSeries(ctx, inst, "5m", limit+1, okxLSAccounts, domain.LongShortKindAccounts)
	if err != nil {
		return nil, err
	}
	if topAcc, errA := c.fetchRatioSeries(ctx, inst, "5m", limit+1, okxLSTopAcc, domain.LongShortKindTopAccounts); errA == nil {
		global.TopAccounts = topAcc
	}
	if topPos, errP := c.fetchRatioSeries(ctx, inst, "5m", limit+1, okxLSTopPos, domain.LongShortKindTopPositions); errP == nil {
		global.TopPositions = topPos
	}
	global.Exchange = domain.ExchangeOKX
	global.Symbol = symbol
	domain.AttachLongShortSplit(global)
	return global, nil
}

// ListLongShortHistory pages 1-hour account, top-account, and top-position ratios.
// Each rubik endpoint returns at most the latest 1,440 entries.
func (c *Client) ListLongShortHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.LongShortHistoryPoint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	inst := SwapInstID(symbol)
	if inst == "" {
		return nil, fmt.Errorf("%w: okx swap symbol", domain.ErrInvalidArgument)
	}
	from, to = from.UTC(), to.UTC()
	if !to.After(from) {
		return nil, fmt.Errorf("%w: to must be after from", domain.ErrInvalidArgument)
	}
	from = domain.ClampPublicHistoryFrom(domain.ExchangeOKX, domain.FuturesMetricLongShort, from, to)
	global, err := c.pageRatio(ctx, inst, okxLSAccounts, from, to)
	if err != nil {
		return nil, err
	}
	topAcc, _ := c.pageRatio(ctx, inst, okxLSTopAcc, from, to)
	topPos, _ := c.pageRatio(ctx, inst, okxLSTopPos, from, to)
	out := make([]domain.LongShortHistoryPoint, 0, len(global))
	for _, p := range global {
		pt := domain.NormalizeLongShortPoint(p)
		row := domain.LongShortHistoryPoint{
			Time: pt.Time, LongShare: pt.LongShare, ShortShare: pt.ShortShare, Ratio: pt.Ratio,
		}
		if acc, ok := nearestOKXRatio(topAcc, pt.Time, time.Hour); ok {
			row.TopAccLongShare = domain.NormalizeLongShortPoint(acc).LongShare
		}
		if pos, ok := nearestOKXRatio(topPos, pt.Time, time.Hour); ok {
			row.SizeLongShare = domain.NormalizeLongShortPoint(pos).LongShare
		}
		out = append(out, row)
	}
	return out, nil
}

func (c *Client) fetchRatioSeries(ctx context.Context, inst, period string, limit int, path, kind string) (*domain.LongShortSeries, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > okxLSPage {
		limit = okxLSPage
	}
	q := url.Values{}
	q.Set("instId", inst)
	q.Set("period", period)
	q.Set("limit", strconv.Itoa(limit))
	pts, err := c.fetchRatioQuery(ctx, path, q)
	if err != nil {
		return nil, err
	}
	pts = domain.SortLongShortNewestFirst(pts)
	if len(pts) == 0 {
		return nil, fmt.Errorf("%w: long short ratio", domain.ErrNotFound)
	}
	return &domain.LongShortSeries{
		Exchange: domain.ExchangeOKX,
		Kind:     kind,
		Period:   domain.LongShortPeriod5m,
		Current:  pts[0],
		History:  pts[1:],
	}, nil
}

func (c *Client) pageRatio(ctx context.Context, inst, path string, from, to time.Time) ([]domain.LongShortPoint, error) {
	var out []domain.LongShortPoint
	end := to.UnixMilli()
	for page := 0; page < domain.FuturesHistoryPageBudget(okxLSPage); page++ {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		if page > 0 {
			if err := waitRubik(ctx); err != nil {
				return out, err
			}
		}
		q := url.Values{}
		q.Set("instId", inst)
		q.Set("period", "1H")
		q.Set("limit", strconv.Itoa(okxLSPage))
		q.Set("end", strconv.FormatInt(end, 10))
		batch, err := c.fetchRatioQuery(ctx, path, q)
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
		if !oldest.After(from) || len(batch) < okxLSPage {
			break
		}
		next := oldest.UnixMilli() - 1
		if next >= end {
			break
		}
		end = next
	}
	kept := out[:0]
	for _, p := range out {
		if p.Time.Before(from) || p.Time.After(to) {
			continue
		}
		kept = append(kept, p)
	}
	return kept, nil
}

func (c *Client) fetchRatioQuery(ctx context.Context, path string, q url.Values) ([]domain.LongShortPoint, error) {
	body, err := c.get(ctx, path, q)
	if err != nil {
		return nil, err
	}
	var rows [][]string
	if err := decodeOKX(body, &rows); err != nil {
		return nil, err
	}
	out := make([]domain.LongShortPoint, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		ts, ok := parseMillis(row[0])
		if !ok {
			continue
		}
		ratio, ok := parseFloat(row[1])
		if !ok {
			continue
		}
		longShare, shortShare, ok := sharesFromOKXRatio(ratio)
		if !ok {
			continue
		}
		p := domain.NormalizeLongShortPoint(domain.LongShortPoint{
			Time: ts, LongShare: longShare, ShortShare: shortShare, Ratio: ratio,
		})
		if p.Ratio <= 0 {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func sharesFromOKXRatio(ratio float64) (longShare, shortShare float64, ok bool) {
	if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 0, 0, false
	}
	return ratio / (1 + ratio), 1 / (1 + ratio), true
}

func nearestOKXRatio(pts []domain.LongShortPoint, at time.Time, slack time.Duration) (domain.LongShortPoint, bool) {
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

func waitRubik(ctx context.Context) error {
	if rubikGap <= 0 {
		return nil
	}
	timer := time.NewTimer(rubikGap)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
