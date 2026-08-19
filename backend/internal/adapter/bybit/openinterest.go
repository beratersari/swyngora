package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOpenInterestSeries loads Bybit linear perpetual current OI plus 5m history.
func (c *Client) GetOpenInterestSeries(ctx context.Context, symbol string) (*domain.OpenInterestSeries, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
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
		cur  bybitLinearTicker
		hist []domain.OpenInterestPoint
		px   map[int64]float64
		errT error
		errH error
		errK error
		wg   sync.WaitGroup
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		cur, errT = c.fetchLinearTicker(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		hist, errH = c.fetchLinearOIHist(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		px, errK = c.fetchLinearCloses(ctx, symbol)
	}()
	wg.Wait()
	if errT != nil {
		return nil, errT
	}
	if errH != nil {
		return nil, errH
	}
	value := cur.OpenInterestValue
	if value <= 0 && cur.LastPrice > 0 {
		value = cur.OpenInterest * cur.LastPrice
	}
	for i := range hist {
		if hist[i].Value > 0 {
			continue
		}
		if closePx := closeAt(px, hist[i].Time); closePx > 0 {
			hist[i].Value = hist[i].Contracts * closePx
		} else if cur.LastPrice > 0 {
			hist[i].Value = hist[i].Contracts * cur.LastPrice
		}
	}
	_ = errK // klines are best-effort for hist notional
	out := &domain.OpenInterestSeries{
		Exchange: domain.ExchangeBybit,
		Symbol:   symbol,
		Current: domain.OpenInterestPoint{
			Time:      cur.Time,
			Contracts: cur.OpenInterest,
			Value:     value,
		},
		History: domain.SortOpenInterestHistory(hist),
	}
	if out.Current.Time.IsZero() {
		out.Current.Time = time.Now().UTC()
	}
	return out, nil
}

type bybitLinearTicker struct {
	OpenInterest      float64
	OpenInterestValue float64
	LastPrice         float64
	Time              time.Time
}

func (c *Client) fetchLinearTicker(ctx context.Context, symbol string) (bybitLinearTicker, error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	body, err := c.get(ctx, "/v5/market/tickers", q)
	if err != nil {
		return bybitLinearTicker{}, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				OpenInterest            string `json:"openInterest"`
				OpenInterestValue       string `json:"openInterestValue"`
				SingleOpenInterest      string `json:"singleOpenInterest"`
				SingleOpenInterestValue string `json:"singleOpenInterestValue"`
				LastPrice               string `json:"lastPrice"`
			} `json:"list"`
		} `json:"result"`
		Time int64 `json:"time"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return bybitLinearTicker{}, fmt.Errorf("%w: ticker decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return bybitLinearTicker{}, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	if len(raw.Result.List) == 0 {
		return bybitLinearTicker{}, fmt.Errorf("%w: linear ticker", domain.ErrNotFound)
	}
	row := raw.Result.List[0]
	oi, ok := bybitSingleSideOI(row.SingleOpenInterest, row.OpenInterest)
	if !ok {
		return bybitLinearTicker{}, fmt.Errorf("%w: open interest value", domain.ErrUpstream)
	}
	px, _ := strconv.ParseFloat(strings.TrimSpace(row.LastPrice), 64)
	val := bybitSingleSideValue(row.SingleOpenInterestValue, row.OpenInterestValue, oi, px)
	ts := time.Now().UTC()
	if raw.Time > 0 {
		ts = time.UnixMilli(raw.Time).UTC()
	}
	return bybitLinearTicker{
		OpenInterest: oi, OpenInterestValue: val, LastPrice: px, Time: ts,
	}, nil
}

func (c *Client) fetchLinearOIHist(ctx context.Context, symbol string) ([]domain.OpenInterestPoint, error) {
	// 200 * 5min = 16.6h; second page covers the rest of 24h.
	first, err := c.fetchOIPage(ctx, symbol, 0)
	if err != nil {
		return nil, err
	}
	out := append([]domain.OpenInterestPoint(nil), first...)
	if oldest, ok := oldestPoint(first); ok && time.Since(oldest) < 23*time.Hour {
		more, err := c.fetchOIPage(ctx, symbol, oldest.UnixMilli())
		if err == nil {
			out = append(out, more...)
		}
	}
	return domain.SortOpenInterestHistory(out), nil
}

func (c *Client) fetchOIPage(ctx context.Context, symbol string, endTimeMs int64) ([]domain.OpenInterestPoint, error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	q.Set("intervalTime", "5min")
	q.Set("limit", "200")
	if endTimeMs > 0 {
		q.Set("endTime", strconv.FormatInt(endTimeMs, 10))
	}
	body, err := c.get(ctx, "/v5/market/open-interest", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				OpenInterest       string `json:"openInterest"`
				SingleOpenInterest string `json:"singleOpenInterest"`
				Timestamp          string `json:"timestamp"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: open interest decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return nil, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	out := make([]domain.OpenInterestPoint, 0, len(raw.Result.List))
	for _, row := range raw.Result.List {
		oi, ok := bybitSingleSideOI(row.SingleOpenInterest, row.OpenInterest)
		if !ok {
			continue
		}
		ms, err := strconv.ParseInt(row.Timestamp, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, domain.OpenInterestPoint{
			Time:      time.UnixMilli(ms).UTC(),
			Contracts: oi,
		})
	}
	return out, nil
}

func (c *Client) fetchLinearCloses(ctx context.Context, symbol string) (map[int64]float64, error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	q.Set("interval", "5")
	q.Set("limit", "300")
	body, err := c.get(ctx, "/v5/market/kline", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List [][]string `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: kline decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return nil, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	out := make(map[int64]float64, len(raw.Result.List))
	for _, row := range raw.Result.List {
		if len(row) < 5 {
			continue
		}
		ms, err1 := strconv.ParseInt(row[0], 10, 64)
		closePx, err2 := strconv.ParseFloat(row[4], 64)
		if err1 != nil || err2 != nil || closePx <= 0 {
			continue
		}
		out[ms] = closePx
	}
	return out, nil
}

func closeAt(px map[int64]float64, at time.Time) float64 {
	if len(px) == 0 || at.IsZero() {
		return 0
	}
	ms := at.UnixMilli()
	if v, ok := px[ms]; ok {
		return v
	}
	// 5m buckets: align down to 5 minutes.
	bucket := at.UTC().Truncate(5 * time.Minute).UnixMilli()
	if v, ok := px[bucket]; ok {
		return v
	}
	var best int64
	var found bool
	for k, v := range px {
		if k > ms {
			continue
		}
		if !found || k > best {
			best = k
			found = true
			_ = v
		}
	}
	if !found || ms-best > int64((7*time.Minute)/time.Millisecond) {
		return 0
	}
	return px[best]
}

// bybitSingleSideOI prefers Bybit's unilateral field. openInterest is still the
// sum of both sides (~2×); after 2026-06-11 the UI and singleOpenInterest are
// one side. If only the bilateral field is present, halve it.
func bybitSingleSideOI(single, both string) (float64, bool) {
	if v, ok := parseOptionalFloat(single); ok {
		return v, true
	}
	if v, ok := parseOptionalFloat(both); ok {
		return v / 2, true
	}
	return 0, false
}

func bybitSingleSideValue(singleVal, bothVal string, contracts, lastPrice float64) float64 {
	if v, ok := parseOptionalFloat(singleVal); ok {
		return v
	}
	if v, ok := parseOptionalFloat(bothVal); ok {
		return v / 2
	}
	if lastPrice > 0 && contracts > 0 {
		return contracts * lastPrice
	}
	return 0
}

func parseOptionalFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}

func oldestPoint(pts []domain.OpenInterestPoint) (time.Time, bool) {
	var t time.Time
	for _, p := range pts {
		if p.Time.IsZero() {
			continue
		}
		if t.IsZero() || p.Time.Before(t) {
			t = p.Time
		}
	}
	return t, !t.IsZero()
}
