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
func (pumpStubPort) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{}, nil
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

// scanStubPort returns several symbols and multi-pump candles for scan handler tests.
type scanStubPort struct{}

func (scanStubPort) GetCandles(_ context.Context, q domain.CandleQuery) ([]domain.Candle, error) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	// multiple +20% jumps so MaxEventsPerSymbol can yield several events per symbol
	price := 100.0
	out := make([]domain.Candle, 0, 9)
	t0 := base
	out = append(out, domain.Candle{
		OpenTime: t0, CloseTime: t0.Add(15 * time.Minute),
		Open: "100", High: "100", Low: "100", Close: "100", Volume: "10",
	})
	t0 = t0.Add(15 * time.Minute)
	for i := 0; i < 4; i++ {
		next := price * 1.20
		out = append(out, domain.Candle{
			OpenTime: t0, CloseTime: t0.Add(15 * time.Minute),
			Open: f64s(price), High: f64s(next), Low: f64s(price), Close: f64s(next), Volume: "50",
		})
		price = next
		t0 = t0.Add(15 * time.Minute)
		out = append(out, domain.Candle{
			OpenTime: t0, CloseTime: t0.Add(15 * time.Minute),
			Open: f64s(price), High: f64s(price), Low: f64s(price), Close: f64s(price), Volume: "10",
		})
		t0 = t0.Add(15 * time.Minute)
	}
	return out, nil
}
func (scanStubPort) GetTicker24h(context.Context, string) (*domain.Ticker24h, error) {
	return nil, domain.ErrNotFound
}
func (scanStubPort) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{}, nil
}
func (scanStubPort) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) {
	return []domain.SpotMarket{
		{Symbol: "AAAUSDT", BaseAsset: "AAA", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "3000", Volume: "10", LastPrice: "1"},
		{Symbol: "BBBUSDT", BaseAsset: "BBB", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "2000", Volume: "10", LastPrice: "1"},
		{Symbol: "CCCUSDT", BaseAsset: "CCC", QuoteAsset: "USDT", Status: "TRADING", QuoteVolume: "1000", Volume: "10", LastPrice: "1"},
	}, nil
}
func (scanStubPort) ListProductTags(context.Context) ([]string, error) { return nil, nil }
func (scanStubPort) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

func f64s(f float64) string {
	return market.FormatPumpReturn(f)
}

func TestScanPumpEvents_DefaultsInMetadataWhenParamsOmitted(t *testing.T) {
	svc := market.New(scanStubPort{}, nil)
	h := NewMarketHandler(svc)
	// No optional query params — response must echo resolved defaults, not zeros/empties.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/pumps/scan", nil)
	rr := httptest.NewRecorder()
	h.ScanPumpEvents(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["exchange"] != "binance" {
		t.Fatalf("exchange=%v want binance", body["exchange"])
	}
	if body["quote"] != "USDT" {
		t.Fatalf("quote=%v", body["quote"])
	}
	if body["interval"] != "15m" {
		t.Fatalf("interval=%v want 15m", body["interval"])
	}
	if body["lookbackHours"].(float64) != 24 {
		t.Fatalf("lookbackHours=%v want 24", body["lookbackHours"])
	}
	if body["minReturnPct"].(float64) != 8 {
		t.Fatalf("minReturnPct=%v want 8", body["minReturnPct"])
	}
	if body["windowBars"].(float64) != 1 {
		t.Fatalf("windowBars=%v", body["windowBars"])
	}
	if body["mode"] != "close_return" {
		t.Fatalf("mode=%v", body["mode"])
	}
	if body["direction"] != "up" {
		t.Fatalf("direction=%v", body["direction"])
	}
	if body["symbolLimit"].(float64) != 15 {
		t.Fatalf("symbolLimit=%v", body["symbolLimit"])
	}
	if body["maxTotalEvents"].(float64) != 30 {
		t.Fatalf("maxTotalEvents=%v want 30", body["maxTotalEvents"])
	}
}

func TestScanPumpEvents_MaxTotalEventsCapsEventCount(t *testing.T) {
	svc := market.New(scanStubPort{}, nil)
	h := NewMarketHandler(svc)
	// minReturnPct=5 so multi-pump candles qualify; maxTotalEvents=4 must limit aggregate events.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/pumps/scan?minReturnPct=5&symbolLimit=3&maxTotalEvents=4", nil)
	rr := httptest.NewRecorder()
	h.ScanPumpEvents(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["maxTotalEvents"].(float64) != 4 {
		t.Fatalf("maxTotalEvents metadata=%v", body["maxTotalEvents"])
	}
	eventCount := int(body["eventCount"].(float64))
	if eventCount > 4 {
		t.Fatalf("eventCount=%d exceeds maxTotalEvents=4 body=%s", eventCount, rr.Body.String())
	}
	if eventCount < 1 {
		t.Fatalf("expected at least one event: %s", rr.Body.String())
	}
	// Count events in hits as well
	hits, _ := body["hits"].([]any)
	total := 0
	for _, raw := range hits {
		m := raw.(map[string]any)
		evs, _ := m["events"].([]any)
		total += len(evs)
	}
	if total != eventCount {
		t.Fatalf("sum events in hits=%d eventCount=%d", total, eventCount)
	}
	if total > 4 {
		t.Fatalf("hits contain %d events > maxTotalEvents", total)
	}
}
