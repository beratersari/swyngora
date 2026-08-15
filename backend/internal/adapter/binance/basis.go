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

// GetBasisQuote loads Binance USD-M last/mark vs spot index, plus 5m history.
func (c *Client) GetBasisQuote(ctx context.Context, symbol string) (*domain.BasisQuote, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	if cached, ok := c.basisCache.Get(symbol); ok && cached != nil {
		return cached, nil
	}
	v, err, _ := c.basisSF.Do(symbol, func() (any, error) {
		if cached, ok := c.basisCache.Get(symbol); ok && cached != nil {
			return cached, nil
		}
		got, err := c.fetchBasis(ctx, symbol)
		if err != nil {
			return nil, err
		}
		c.basisCache.Set(symbol, got)
		return got, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.(*domain.BasisQuote)
	return got, nil
}

func (c *Client) fetchBasis(ctx context.Context, symbol string) (*domain.BasisQuote, error) {
	var (
		mark, index, last, spot      float64
		at                           time.Time
		marks, indexes               []domain.PriceSample
		errP, errL, errS, errM, errI error
		wg                           sync.WaitGroup
	)
	wg.Add(5)
	go func() {
		defer wg.Done()
		mark, index, at, errP = c.fetchPremiumPrices(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		last, errL = c.fetchFuturesLast(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		spot, errS = c.fetchSpotLast(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		marks, errM = c.fetchFuturesKlineCloses(ctx, "/fapi/v1/markPriceKlines", symbol)
	}()
	go func() {
		defer wg.Done()
		indexes, errI = c.fetchFuturesKlineCloses(ctx, "/fapi/v1/indexPriceKlines", symbol)
	}()
	wg.Wait()
	if errP != nil {
		return nil, errP
	}
	if index <= 0 || mark <= 0 {
		return nil, fmt.Errorf("%w: premium index prices", domain.ErrUpstream)
	}
	_ = errL
	_ = errS
	if errM != nil {
		marks = nil
	}
	if errI != nil {
		indexes = nil
	}
	out := &domain.BasisQuote{
		Exchange:    domain.ExchangeBinance,
		Symbol:      symbol,
		FuturesLast: last,
		FuturesMark: mark,
		SpotIndex:   index,
		SpotLast:    spot,
		Time:        at,
		History:     domain.BuildBasisHistory(marks, indexes),
	}
	if out.Time.IsZero() {
		out.Time = time.Now().UTC()
	}
	return out, nil
}

func (c *Client) fetchPremiumPrices(ctx context.Context, symbol string) (mark, index float64, at time.Time, err error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	body, err := c.getFutures(ctx, "/fapi/v1/premiumIndex", q)
	if err != nil {
		return 0, 0, time.Time{}, err
	}
	var raw struct {
		MarkPrice  string `json:"markPrice"`
		IndexPrice string `json:"indexPrice"`
		Time       int64  `json:"time"`
		Code       int    `json:"code"`
		Msg        string `json:"msg"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, 0, time.Time{}, fmt.Errorf("%w: premium decode: %v", domain.ErrUpstream, err)
	}
	if raw.Code != 0 && raw.Msg != "" {
		return 0, 0, time.Time{}, mapBinanceError(raw.Code, raw.Msg)
	}
	mark, _ = strconv.ParseFloat(raw.MarkPrice, 64)
	index, _ = strconv.ParseFloat(raw.IndexPrice, 64)
	if raw.Time > 0 {
		at = time.UnixMilli(raw.Time).UTC()
	}
	return mark, index, at, nil
}

func (c *Client) fetchFuturesLast(ctx context.Context, symbol string) (float64, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	body, err := c.getFutures(ctx, "/fapi/v1/ticker/price", q)
	if err != nil {
		return 0, err
	}
	var raw struct {
		Price string `json:"price"`
	}
	if json.Unmarshal(body, &raw) != nil {
		return 0, fmt.Errorf("%w: last price decode", domain.ErrUpstream)
	}
	return strconv.ParseFloat(raw.Price, 64)
}

func (c *Client) fetchSpotLast(ctx context.Context, symbol string) (float64, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	body, err := c.get(ctx, "/api/v3/ticker/price", q)
	if err != nil {
		return 0, err
	}
	var raw struct {
		Price string `json:"price"`
	}
	if json.Unmarshal(body, &raw) != nil {
		return 0, fmt.Errorf("%w: spot last decode", domain.ErrUpstream)
	}
	return strconv.ParseFloat(raw.Price, 64)
}

func (c *Client) fetchFuturesKlineCloses(ctx context.Context, path, symbol string) ([]domain.PriceSample, error) {
	q := url.Values{}
	// indexPriceKlines is keyed by pair; markPriceKlines uses symbol.
	if path == "/fapi/v1/indexPriceKlines" {
		q.Set("pair", symbol)
	} else {
		q.Set("symbol", symbol)
	}
	q.Set("interval", "5m")
	q.Set("limit", "60")
	body, err := c.getFutures(ctx, path, q)
	if err != nil {
		return nil, err
	}
	var rows [][]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: kline decode: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.PriceSample, 0, len(rows))
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		ms, ok := jsonInt64(row[0])
		if !ok || ms <= 0 {
			continue
		}
		closeStr, _ := row[4].(string)
		px, err := strconv.ParseFloat(closeStr, 64)
		if err != nil || px <= 0 {
			continue
		}
		out = append(out, domain.PriceSample{Time: time.UnixMilli(ms).UTC(), Price: px})
	}
	return out, nil
}

func jsonInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
