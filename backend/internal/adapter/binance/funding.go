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

// GetFundingSeries loads Binance USD-M predicted next rate plus settled history.
func (c *Client) GetFundingSeries(ctx context.Context, symbol string, limit int) (*domain.FundingSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
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
		cur  binancePremium
		hist []domain.FundingPoint
		errC error
		errH error
		wg   sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		cur, errC = c.fetchPremiumIndex(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		hist, errH = c.fetchFundingHist(ctx, symbol, limit)
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
		Exchange: domain.ExchangeBinance,
		Symbol:   symbol,
		Current: domain.FundingPoint{
			Time:      cur.Time,
			Rate:      cur.LastFundingRate,
			MarkPrice: cur.MarkPrice,
			Predicted: true,
		},
		NextFundingTime: cur.NextFundingTime,
		IntervalHours:   domain.InferFundingIntervalHours(0, cur.NextFundingTime, last),
		History:         hist,
	}, nil
}

type binancePremium struct {
	LastFundingRate float64
	MarkPrice       float64
	NextFundingTime time.Time
	Time            time.Time
}

func (c *Client) fetchPremiumIndex(ctx context.Context, symbol string) (binancePremium, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	body, err := c.getFutures(ctx, "/fapi/v1/premiumIndex", q)
	if err != nil {
		return binancePremium{}, err
	}
	var raw struct {
		LastFundingRate string `json:"lastFundingRate"`
		MarkPrice       string `json:"markPrice"`
		NextFundingTime int64  `json:"nextFundingTime"`
		Time            int64  `json:"time"`
		Code            int    `json:"code"`
		Msg             string `json:"msg"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return binancePremium{}, fmt.Errorf("%w: premium index decode: %v", domain.ErrUpstream, err)
	}
	if raw.Code != 0 && raw.Msg != "" {
		return binancePremium{}, mapBinanceError(raw.Code, raw.Msg)
	}
	rate, err := strconv.ParseFloat(raw.LastFundingRate, 64)
	if err != nil {
		return binancePremium{}, fmt.Errorf("%w: funding rate", domain.ErrUpstream)
	}
	mark, _ := strconv.ParseFloat(raw.MarkPrice, 64)
	out := binancePremium{LastFundingRate: rate, MarkPrice: mark}
	if raw.NextFundingTime > 0 {
		out.NextFundingTime = time.UnixMilli(raw.NextFundingTime).UTC()
	}
	if raw.Time > 0 {
		out.Time = time.UnixMilli(raw.Time).UTC()
	}
	return out, nil
}

func (c *Client) fetchFundingHist(ctx context.Context, symbol string, limit int) ([]domain.FundingPoint, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("limit", strconv.Itoa(limit))
	body, err := c.getFutures(ctx, "/fapi/v1/fundingRate", q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		FundingRate string `json:"fundingRate"`
		FundingTime int64  `json:"fundingTime"`
		MarkPrice   string `json:"markPrice"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: funding hist decode: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.FundingPoint, 0, len(rows))
	for _, r := range rows {
		rate, err := strconv.ParseFloat(r.FundingRate, 64)
		if err != nil {
			continue
		}
		mark, _ := strconv.ParseFloat(r.MarkPrice, 64)
		out = append(out, domain.FundingPoint{
			Time:      time.UnixMilli(r.FundingTime).UTC(),
			Rate:      rate,
			MarkPrice: mark,
		})
	}
	return out, nil
}
