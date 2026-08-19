package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOpenInterestSeries loads Binance USD-M current OI plus 5m history (~25h).
func (c *Client) GetOpenInterestSeries(ctx context.Context, symbol string) (*domain.OpenInterestSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
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
	var (
		cur  binanceCurrentOI
		mark float64
		hist []domain.OpenInterestPoint
		errC error
		errM error
		errH error
		wg   sync.WaitGroup
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		cur, errC = c.fetchCurrentOI(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		mark, errM = c.fetchMarkPrice(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		hist, errH = c.fetchOIHist(ctx, symbol)
	}()
	wg.Wait()
	if errC != nil {
		return nil, errC
	}
	if errH != nil {
		return nil, errH
	}
	if errM != nil || mark <= 0 {
		if n := len(hist); n > 0 && hist[n-1].Contracts > 0 {
			mark = hist[n-1].Value / hist[n-1].Contracts
		}
	}
	out := &domain.OpenInterestSeries{
		Exchange: domain.ExchangeBinance,
		Symbol:   symbol,
		Current: domain.OpenInterestPoint{
			Time:      time.UnixMilli(cur.Time).UTC(),
			Contracts: cur.OpenInterest,
			Value:     cur.OpenInterest * mark,
		},
		History: hist,
	}
	if out.Current.Time.IsZero() && len(hist) > 0 {
		out.Current.Time = hist[len(hist)-1].Time
	}
	out.History = domain.SortOpenInterestHistory(out.History)
	return out, nil
}

type binanceCurrentOI struct {
	OpenInterest float64
	Time         int64
}

func (c *Client) fetchCurrentOI(ctx context.Context, symbol string) (binanceCurrentOI, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	body, err := c.getFutures(ctx, "/fapi/v1/openInterest", q)
	if err != nil {
		return binanceCurrentOI{}, err
	}
	var raw struct {
		OpenInterest string `json:"openInterest"`
		Time         int64  `json:"time"`
		Code         int    `json:"code"`
		Msg          string `json:"msg"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return binanceCurrentOI{}, fmt.Errorf("%w: open interest decode: %v", domain.ErrUpstream, err)
	}
	if raw.Code != 0 && raw.Msg != "" {
		return binanceCurrentOI{}, mapBinanceError(raw.Code, raw.Msg)
	}
	oi, err := strconv.ParseFloat(raw.OpenInterest, 64)
	if err != nil || oi < 0 {
		return binanceCurrentOI{}, fmt.Errorf("%w: open interest value", domain.ErrUpstream)
	}
	return binanceCurrentOI{OpenInterest: oi, Time: raw.Time}, nil
}

func (c *Client) fetchMarkPrice(ctx context.Context, symbol string) (float64, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	body, err := c.getFutures(ctx, "/fapi/v1/premiumIndex", q)
	if err != nil {
		return 0, err
	}
	var raw struct {
		MarkPrice string `json:"markPrice"`
		Code      int    `json:"code"`
		Msg       string `json:"msg"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, fmt.Errorf("%w: mark price decode: %v", domain.ErrUpstream, err)
	}
	if raw.Code != 0 && raw.Msg != "" {
		return 0, mapBinanceError(raw.Code, raw.Msg)
	}
	px, err := strconv.ParseFloat(raw.MarkPrice, 64)
	if err != nil || px <= 0 {
		return 0, fmt.Errorf("%w: mark price", domain.ErrUpstream)
	}
	return px, nil
}

func (c *Client) fetchOIHist(ctx context.Context, symbol string) ([]domain.OpenInterestPoint, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("period", "5m")
	q.Set("limit", "300")
	body, err := c.getFutures(ctx, "/futures/data/openInterestHist", q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		SumOpenInterest      string `json:"sumOpenInterest"`
		SumOpenInterestValue string `json:"sumOpenInterestValue"`
		Timestamp            int64  `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: open interest hist decode: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.OpenInterestPoint, 0, len(rows))
	for _, r := range rows {
		contracts, err1 := strconv.ParseFloat(r.SumOpenInterest, 64)
		value, err2 := strconv.ParseFloat(r.SumOpenInterestValue, 64)
		if err1 != nil || err2 != nil || contracts < 0 {
			continue
		}
		out = append(out, domain.OpenInterestPoint{
			Time:      time.UnixMilli(r.Timestamp).UTC(),
			Contracts: contracts,
			Value:     value,
		})
	}
	return domain.SortOpenInterestHistory(out), nil
}

func (c *Client) getFutures(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u := c.futuresBase + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", domain.ErrUpstream, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", domain.ErrUpstream, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusTeapot {
		return nil, fmt.Errorf("%w: binance status %d", domain.ErrRateLimited, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: symbol or resource", domain.ErrNotFound)
	}
	if resp.StatusCode >= 400 {
		var er binanceError
		_ = json.Unmarshal(body, &er)
		if er.Msg != "" {
			return nil, mapBinanceError(er.Code, er.Msg)
		}
		return nil, fmt.Errorf("%w: status %d", domain.ErrUpstream, resp.StatusCode)
	}
	return body, nil
}
