package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

type pumpStubPort struct{}

func (pumpStubPort) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return []domain.Candle{
		{OpenTime: base, CloseTime: base.Add(time.Hour), Open: "100", High: "100", Low: "100", Close: "100", Volume: "10"},
		{OpenTime: base.Add(time.Hour), CloseTime: base.Add(2 * time.Hour), Open: "100", High: "120", Low: "100", Close: "115", Volume: "50"},
	}, nil
}
func (pumpStubPort) GetTicker24h(context.Context, string) (*domain.Ticker24h, error) {
	return nil, domain.ErrNotFound
}
func (pumpStubPort) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) {
	return []domain.SpotMarket{{Symbol: "BTCUSDT", QuoteVolume: "1"}}, nil
}
func (pumpStubPort) ListProductTags(context.Context) ([]string, error) { return nil, nil }
func (pumpStubPort) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func TestGetPumpEvents(t *testing.T) {
	svc := market.New(pumpStubPort{}, nil)
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/pumps?symbol=BTCUSDT&minReturnPct=5&interval=1h&limit=10", nil)
	rr := httptest.NewRecorder()
	h.GetPumpEvents(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["eventCount"].(float64) < 1 {
		t.Fatalf("expected events: %v", body)
	}
}
