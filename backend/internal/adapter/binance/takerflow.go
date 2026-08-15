package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetTakerFlow loads Binance USD-M taker buy/sell volume (5m bars).
func (c *Client) GetTakerFlow(ctx context.Context, symbol string) (*domain.TakerVenueFlow, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	if cached, ok := c.takerCache.Get(symbol); ok && cached != nil {
		return cached, nil
	}
	v, err, _ := c.takerSF.Do(symbol, func() (any, error) {
		if cached, ok := c.takerCache.Get(symbol); ok && cached != nil {
			return cached, nil
		}
		got, err := c.fetchTakerFlow(ctx, symbol)
		if err != nil {
			return nil, err
		}
		c.takerCache.Set(symbol, got)
		return got, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.(*domain.TakerVenueFlow)
	return got, nil
}

func (c *Client) fetchTakerFlow(ctx context.Context, symbol string) (*domain.TakerVenueFlow, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("period", "5m")
	q.Set("limit", "50") // 50*5m = ~4h
	body, err := c.getFutures(ctx, "/futures/data/takerlongshortRatio", q)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		BuyVol    string `json:"buyVol"`
		SellVol   string `json:"sellVol"`
		Timestamp int64  `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%w: taker flow decode: %v", domain.ErrUpstream, err)
	}
	px := c.futuresMarkPrice(ctx, symbol)
	now := time.Now().UTC()
	buckets := make([]domain.TakerBucket, 0, len(rows))
	var oldest time.Time
	for _, r := range rows {
		buy, e1 := strconv.ParseFloat(r.BuyVol, 64)
		sell, e2 := strconv.ParseFloat(r.SellVol, 64)
		if e1 != nil || e2 != nil || r.Timestamp <= 0 {
			continue
		}
		if px > 0 {
			buy *= px
			sell *= px
		}
		at := time.UnixMilli(r.Timestamp).UTC()
		buckets = append(buckets, domain.TakerBucket{
			Exchange: domain.ExchangeBinance, Symbol: symbol, Start: at,
			BuyNotional: buy, SellNotional: sell,
		})
		if oldest.IsZero() || at.Before(oldest) {
			oldest = at
		}
	}
	if len(buckets) == 0 {
		return nil, fmt.Errorf("%w: taker buy/sell volume", domain.ErrNotFound)
	}
	out := domain.BuildTakerVenueFlowBucket(domain.ExchangeBinance, symbol, buckets, now, oldest, 5*time.Minute)
	out.Price = px
	return &out, nil
}

func (c *Client) futuresMarkPrice(ctx context.Context, symbol string) float64 {
	q := url.Values{}
	q.Set("symbol", symbol)
	body, err := c.getFutures(ctx, "/fapi/v1/premiumIndex", q)
	if err != nil {
		return 0
	}
	var raw struct {
		MarkPrice string `json:"markPrice"`
	}
	if json.Unmarshal(body, &raw) != nil {
		return 0
	}
	px, err := strconv.ParseFloat(raw.MarkPrice, 64)
	if err != nil || px <= 0 {
		return 0
	}
	return px
}
