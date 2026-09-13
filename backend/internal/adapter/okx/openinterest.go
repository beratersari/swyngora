package okx

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOpenInterestSeries loads OKX USDT-swap current OI plus 5m history.
func (c *Client) GetOpenInterestSeries(ctx context.Context, symbol string) (*domain.OpenInterestSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: okx client", domain.ErrUpstream)
	}
	if cached, ok := c.oiCache.Get(symbol); ok && cached != nil {
		return cached, nil
	}
	v, err, _ := c.oiSF.Do(symbol, func() (any, error) {
		if cached, ok := c.oiCache.Get(symbol); ok && cached != nil {
			return cached, nil
		}
		ser, err := c.fetchOpenInterest(ctx, symbol)
		if err != nil {
			return nil, err
		}
		c.oiCache.Set(symbol, ser)
		return ser, nil
	})
	if err != nil {
		return nil, err
	}
	ser, _ := v.(*domain.OpenInterestSeries)
	return ser, nil
}

func (c *Client) fetchOpenInterest(ctx context.Context, symbol string) (*domain.OpenInterestSeries, error) {
	inst := SwapInstID(symbol)
	if inst == "" {
		return nil, fmt.Errorf("%w: okx swap symbol", domain.ErrInvalidArgument)
	}
	var (
		cur  domain.OpenInterestPoint
		hist []domain.OpenInterestPoint
		errC error
		errH error
		wg   sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		cur, errC = c.fetchCurrentOI(ctx, inst)
	}()
	go func() {
		defer wg.Done()
		hist, errH = c.fetchOIHist(ctx, inst)
	}()
	wg.Wait()
	if errC != nil {
		return nil, errC
	}
	if errH != nil {
		return nil, errH
	}
	out := &domain.OpenInterestSeries{
		Exchange: domain.ExchangeOKX,
		Symbol:   symbol,
		Current:  cur,
		History:  domain.SortOpenInterestHistory(hist),
	}
	if out.Current.Time.IsZero() {
		out.Current.Time = time.Now().UTC()
	}
	return out, nil
}

func (c *Client) fetchCurrentOI(ctx context.Context, instID string) (domain.OpenInterestPoint, error) {
	q := url.Values{}
	q.Set("instId", instID)
	body, err := c.get(ctx, "/api/v5/public/open-interest", q)
	if err != nil {
		return domain.OpenInterestPoint{}, err
	}
	var rows []struct {
		OI    string `json:"oi"`
		OICcy string `json:"oiCcy"`
		OIUsd string `json:"oiUsd"`
		Ts    string `json:"ts"`
	}
	if err := decodeOKX(body, &rows); err != nil {
		return domain.OpenInterestPoint{}, err
	}
	if len(rows) == 0 {
		return domain.OpenInterestPoint{}, fmt.Errorf("%w: open interest", domain.ErrNotFound)
	}
	row := rows[0]
	contracts, ok := parseFloat(row.OICcy)
	if !ok || contracts <= 0 {
		return domain.OpenInterestPoint{}, fmt.Errorf("%w: open interest value", domain.ErrUpstream)
	}
	value, _ := parseFloat(row.OIUsd)
	ts, _ := parseMillis(row.Ts)
	return domain.OpenInterestPoint{Time: ts, Contracts: contracts, Value: value}, nil
}

func (c *Client) fetchOIHist(ctx context.Context, instID string) ([]domain.OpenInterestPoint, error) {
	q := url.Values{}
	q.Set("instId", instID)
	q.Set("period", "5m")
	q.Set("limit", "300")
	body, err := c.get(ctx, "/api/v5/rubik/stat/contracts/open-interest-history", q)
	if err != nil {
		return nil, err
	}
	var rows [][]string
	if err := decodeOKX(body, &rows); err != nil {
		return nil, err
	}
	out := make([]domain.OpenInterestPoint, 0, len(rows))
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		ts, ok := parseMillis(row[0])
		if !ok {
			continue
		}
		contracts, ok := parseFloat(row[2])
		if !ok {
			continue
		}
		value, _ := parseFloat(row[3])
		out = append(out, domain.OpenInterestPoint{Time: ts, Contracts: contracts, Value: value})
	}
	return out, nil
}
