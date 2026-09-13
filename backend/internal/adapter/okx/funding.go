package okx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetFundingSeries loads OKX SWAP predicted next rate plus settled history.
func (c *Client) GetFundingSeries(ctx context.Context, symbol string, limit int) (*domain.FundingSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	limit = domain.ClampFundingHistoryLimit(limit)
	key := symbol + "|" + strconv.Itoa(limit)
	if cached, ok := c.fundingCache.Get(key); ok && cached != nil {
		return cached, nil
	}
	v, err, _ := c.fundingSF.Do(key, func() (any, error) {
		if cached, ok := c.fundingCache.Get(key); ok && cached != nil {
			return cached, nil
		}
		ser, err := c.fetchFunding(ctx, symbol, limit)
		if err != nil {
			return nil, err
		}
		c.fundingCache.Set(key, ser)
		return ser, nil
	})
	if err != nil {
		return nil, err
	}
	ser, _ := v.(*domain.FundingSeries)
	return ser, nil
}

func (c *Client) fetchFunding(ctx context.Context, symbol string, limit int) (*domain.FundingSeries, error) {
	inst := SwapInstID(symbol)
	if inst == "" {
		return nil, fmt.Errorf("%w: okx swap symbol", domain.ErrInvalidArgument)
	}
	var (
		cur  okxFundingNow
		hist []domain.FundingPoint
		errC error
		errH error
		wg   sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		cur, errC = c.fetchFundingNow(ctx, inst)
	}()
	go func() {
		defer wg.Done()
		hist, errH = c.fetchFundingHist(ctx, inst, limit)
	}()
	wg.Wait()
	if errC != nil {
		return nil, errC
	}
	if errH != nil {
		return nil, errH
	}
	hist = domain.SortFundingHistoryNewestFirst(hist)
	var last time.Time
	if len(hist) > 0 {
		last = hist[0].Time
	}
	return &domain.FundingSeries{
		Exchange: domain.ExchangeOKX,
		Symbol:   symbol,
		Current: domain.FundingPoint{
			Time:      cur.Time,
			Rate:      cur.Rate,
			Predicted: true,
		},
		NextFundingTime: cur.NextTime,
		IntervalHours:   domain.InferFundingIntervalHours(cur.IntervalHours, cur.NextTime, last),
		History:         hist,
	}, nil
}

type okxFundingNow struct {
	Rate          float64
	NextTime      time.Time
	IntervalHours int
	Time          time.Time
}

func (c *Client) fetchFundingNow(ctx context.Context, instID string) (okxFundingNow, error) {
	q := url.Values{}
	q.Set("instId", instID)
	body, err := c.get(ctx, "/api/v5/public/funding-rate", q)
	if err != nil {
		return okxFundingNow{}, err
	}
	var rows []struct {
		FundingRate     string `json:"fundingRate"`
		FundingTime     string `json:"fundingTime"`
		NextFundingTime string `json:"nextFundingTime"`
		PrevFundingTime string `json:"prevFundingTime"`
		Ts              string `json:"ts"`
	}
	if err := decodeOKX(body, &rows); err != nil {
		return okxFundingNow{}, err
	}
	if len(rows) == 0 {
		return okxFundingNow{}, fmt.Errorf("%w: funding rate", domain.ErrNotFound)
	}
	row := rows[0]
	rate, ok := parseFloat(row.FundingRate)
	if !ok {
		return okxFundingNow{}, fmt.Errorf("%w: funding rate", domain.ErrUpstream)
	}
	out := okxFundingNow{Rate: rate}
	out.NextTime, _ = parseMillis(row.NextFundingTime)
	out.Time, _ = parseMillis(row.Ts)
	if out.Time.IsZero() {
		out.Time = time.Now().UTC()
	}
	if cur, ok := parseMillis(row.FundingTime); ok && !out.NextTime.IsZero() && out.NextTime.After(cur) {
		out.IntervalHours = int(out.NextTime.Sub(cur).Hours() + 0.5)
	} else if prev, ok := parseMillis(row.PrevFundingTime); ok {
		if cur, ok := parseMillis(row.FundingTime); ok && cur.After(prev) {
			out.IntervalHours = int(cur.Sub(prev).Hours() + 0.5)
		}
	}
	return out, nil
}

func (c *Client) fetchFundingHist(ctx context.Context, instID string, limit int) ([]domain.FundingPoint, error) {
	q := url.Values{}
	q.Set("instId", instID)
	q.Set("limit", strconv.Itoa(limit))
	body, err := c.get(ctx, "/api/v5/public/funding-rate-history", q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		FundingRate  string `json:"fundingRate"`
		RealizedRate string `json:"realizedRate"`
		FundingTime  string `json:"fundingTime"`
	}
	if err := decodeOKX(body, &rows); err != nil {
		return nil, err
	}
	out := make([]domain.FundingPoint, 0, len(rows))
	for _, row := range rows {
		rate, ok := parseFloat(row.RealizedRate)
		if !ok {
			rate, ok = parseFloat(row.FundingRate)
		}
		if !ok {
			continue
		}
		ts, ok := parseMillis(row.FundingTime)
		if !ok {
			continue
		}
		out = append(out, domain.FundingPoint{Time: ts, Rate: rate})
	}
	return out, nil
}

// ListFundingHistory returns settled SWAP funding prints in [from, to].
func (c *Client) ListFundingHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.FundingPoint, error) {
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
		return nil, nil
	}
	var out []domain.FundingPoint
	after := ""
	for i := 0; i < 20; i++ {
		q := url.Values{}
		q.Set("instId", inst)
		q.Set("limit", "100")
		if after != "" {
			q.Set("after", after)
		}
		body, err := c.get(ctx, "/api/v5/public/funding-rate-history", q)
		if err != nil {
			return nil, err
		}
		var rows []struct {
			FundingRate  string `json:"fundingRate"`
			RealizedRate string `json:"realizedRate"`
			FundingTime  string `json:"fundingTime"`
		}
		if err := decodeOKX(body, &rows); err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		var oldest time.Time
		for _, row := range rows {
			ts, ok := parseMillis(row.FundingTime)
			if !ok {
				continue
			}
			if oldest.IsZero() || ts.Before(oldest) {
				oldest = ts
			}
			if ts.Before(from) || ts.After(to) {
				continue
			}
			rate, ok := parseFloat(row.RealizedRate)
			if !ok {
				rate, ok = parseFloat(row.FundingRate)
			}
			if !ok {
				continue
			}
			out = append(out, domain.FundingPoint{Time: ts, Rate: rate})
		}
		if oldest.IsZero() || !oldest.After(from) || len(rows) < 100 {
			break
		}
		after = strconv.FormatInt(oldest.UnixMilli(), 10)
	}
	return domain.SortFundingHistoryOldestFirst(out), nil
}
