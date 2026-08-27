package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetFundingSeries loads Bybit linear predicted next rate plus settled history.
func (c *Client) GetFundingSeries(ctx context.Context, symbol string, limit int) (*domain.FundingSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
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
	var (
		cur  bybitFundingTicker
		hist []domain.FundingPoint
		errT error
		errH error
		wg   sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		cur, errT = c.fetchFundingTicker(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		hist, errH = c.fetchFundingHist(ctx, symbol, limit)
	}()
	wg.Wait()
	if errT != nil {
		return nil, errT
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
		Exchange: domain.ExchangeBybit,
		Symbol:   symbol,
		Current: domain.FundingPoint{
			Time:      cur.Time,
			Rate:      cur.FundingRate,
			Predicted: true,
		},
		NextFundingTime: cur.NextFundingTime,
		IntervalHours:   domain.InferFundingIntervalHours(cur.IntervalHours, cur.NextFundingTime, last),
		History:         hist,
	}, nil
}

type bybitFundingTicker struct {
	FundingRate     float64
	NextFundingTime time.Time
	IntervalHours   int
	Time            time.Time
}

func (c *Client) fetchFundingTicker(ctx context.Context, symbol string) (bybitFundingTicker, error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	body, err := c.get(ctx, "/v5/market/tickers", q)
	if err != nil {
		return bybitFundingTicker{}, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Time    int64  `json:"time"`
		Result  struct {
			List []struct {
				FundingRate         string `json:"fundingRate"`
				NextFundingTime     string `json:"nextFundingTime"`
				FundingIntervalHour string `json:"fundingIntervalHour"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return bybitFundingTicker{}, fmt.Errorf("%w: ticker decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return bybitFundingTicker{}, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	if len(raw.Result.List) == 0 {
		return bybitFundingTicker{}, fmt.Errorf("%w: linear ticker", domain.ErrNotFound)
	}
	row := raw.Result.List[0]
	rate, err := strconv.ParseFloat(strings.TrimSpace(row.FundingRate), 64)
	if err != nil {
		return bybitFundingTicker{}, fmt.Errorf("%w: funding rate", domain.ErrUpstream)
	}
	out := bybitFundingTicker{FundingRate: rate}
	if ms, err := strconv.ParseInt(strings.TrimSpace(row.NextFundingTime), 10, 64); err == nil && ms > 0 {
		out.NextFundingTime = time.UnixMilli(ms).UTC()
	}
	if h, err := strconv.Atoi(strings.TrimSpace(row.FundingIntervalHour)); err == nil {
		out.IntervalHours = h
	}
	if raw.Time > 0 {
		out.Time = time.UnixMilli(raw.Time).UTC()
	} else {
		out.Time = time.Now().UTC()
	}
	return out, nil
}

func (c *Client) fetchFundingHist(ctx context.Context, symbol string, limit int) ([]domain.FundingPoint, error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	q.Set("limit", strconv.Itoa(limit))
	body, err := c.get(ctx, "/v5/market/funding/history", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				FundingRate          string `json:"fundingRate"`
				FundingRateTimestamp string `json:"fundingRateTimestamp"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: funding hist decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return nil, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	out := make([]domain.FundingPoint, 0, len(raw.Result.List))
	for _, row := range raw.Result.List {
		rate, err := strconv.ParseFloat(strings.TrimSpace(row.FundingRate), 64)
		if err != nil {
			continue
		}
		ms, err := strconv.ParseInt(strings.TrimSpace(row.FundingRateTimestamp), 10, 64)
		if err != nil || ms <= 0 {
			continue
		}
		out = append(out, domain.FundingPoint{
			Time: time.UnixMilli(ms).UTC(),
			Rate: rate,
		})
	}
	return out, nil
}

// ListFundingHistory returns settled linear funding prints in [from, to].
func (c *Client) ListFundingHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.FundingPoint, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
	}
	from, to = from.UTC(), to.UTC()
	if !to.After(from) {
		return nil, fmt.Errorf("%w: to must be after from", domain.ErrInvalidArgument)
	}
	const page = 200
	var out []domain.FundingPoint
	cursor := from
	for i := 0; i < 40; i++ {
		q := url.Values{}
		q.Set("category", "linear")
		q.Set("symbol", symbol)
		q.Set("startTime", strconv.FormatInt(cursor.UnixMilli(), 10))
		q.Set("endTime", strconv.FormatInt(to.UnixMilli(), 10))
		q.Set("limit", strconv.Itoa(page))
		batch, err := c.fetchFundingHistRange(ctx, q)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		out = append(out, batch...)
		last := batch[len(batch)-1].Time
		if !last.After(cursor) || last.After(to) || len(batch) < page {
			break
		}
		cursor = last.Add(time.Millisecond)
	}
	return domain.SortFundingHistoryOldestFirst(out), nil
}

func (c *Client) fetchFundingHistRange(ctx context.Context, q url.Values) ([]domain.FundingPoint, error) {
	body, err := c.get(ctx, "/v5/market/funding/history", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				FundingRate          string `json:"fundingRate"`
				FundingRateTimestamp string `json:"fundingRateTimestamp"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: funding hist decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return nil, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	out := make([]domain.FundingPoint, 0, len(raw.Result.List))
	for _, row := range raw.Result.List {
		rate, err := strconv.ParseFloat(strings.TrimSpace(row.FundingRate), 64)
		if err != nil {
			continue
		}
		ms, err := strconv.ParseInt(strings.TrimSpace(row.FundingRateTimestamp), 10, 64)
		if err != nil || ms <= 0 {
			continue
		}
		out = append(out, domain.FundingPoint{
			Time: time.UnixMilli(ms).UTC(),
			Rate: rate,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}
