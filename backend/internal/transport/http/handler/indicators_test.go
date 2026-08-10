package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type indFakeMarket struct{}

func (indFakeMarket) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	n := q.Limit
	if n <= 0 {
		n = 40
	}
	out := make([]domain.Candle, n)
	base := time.Unix(1_700_000_000, 0).UTC()
	for i := 0; i < n; i++ {
		out[i] = domain.Candle{
			OpenTime: base.Add(time.Duration(i) * time.Hour),
			Close:    fmt.Sprintf("%g", 100+float64(i)),
		}
	}
	return out, nil
}
func (indFakeMarket) GetTicker24h(context.Context, string) (*domain.Ticker24h, error) {
	return nil, domain.ErrNotFound
}
func (indFakeMarket) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{}, nil
}
func (indFakeMarket) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) { return nil, nil }
func (indFakeMarket) ListProductTags(context.Context) ([]string, error)            { return nil, nil }
func (indFakeMarket) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func TestGetIndicators_InvalidEMAPeriodsCSV(t *testing.T) {
	svc := market.New(indFakeMarket{}, nil)
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/indicators?symbol=BTCUSDT&emaPeriods=12,nope", nil)
	rr := httptest.NewRecorder()
	h.GetIndicators(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPostIndicatorsBatch_OK(t *testing.T) {
	svc := market.New(indFakeMarket{}, nil)
	h := NewMarketHandler(svc)
	body, _ := json.Marshal(map[string]any{
		"exchange": "binance", "interval": "1h",
		"symbols": []string{"BTCUSDT", "ETHUSDT"},
		"emaPeriods": "12,26",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/market/indicators/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.PostIndicatorsBatch(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d %s", rr.Code, rr.Body.String())
	}
}

func TestPostIndicatorsBatch_TooManySymbols(t *testing.T) {
	svc := market.New(indFakeMarket{}, nil)
	h := NewMarketHandler(svc)
	syms := make([]string, 501)
	for i := range syms {
		syms[i] = fmt.Sprintf("S%d", i)
	}
	body, _ := json.Marshal(map[string]any{"symbols": syms})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/market/indicators/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.PostIndicatorsBatch(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d %s", rr.Code, rr.Body.String())
	}
}
