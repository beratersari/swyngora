package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/swing"
)

type swingPort struct{}

func (swingPort) GetCandles(_ context.Context, _ domain.CandleQuery) ([]domain.Candle, error) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	px := 80.0
	out := make([]domain.Candle, 90)
	for i := 0; i < 90; i++ {
		o := px
		px += 0.5
		out[i] = domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * 4 * time.Hour),
			CloseTime: t0.Add(time.Duration(i+1) * 4 * time.Hour),
			Open: f(o), High: f(px + 0.2), Low: f(o - 0.2), Close: f(px), Volume: "1000",
		}
	}
	out[89].CloseTime = time.Now().UTC().Add(-time.Minute)
	return out, nil
}
func (swingPort) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "120", QuoteVolume: "9000000"}, nil
}
func (swingPort) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{}, nil
}
func (swingPort) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) { return nil, nil }
func (swingPort) ListProductTags(context.Context) ([]string, error)            { return nil, nil }
func (swingPort) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

type swingSupply struct{}

func (swingSupply) GetSupply(context.Context, string) (*domain.AssetSupply, error) {
	return &domain.AssetSupply{}, nil
}
func (swingSupply) Refresh(context.Context) (int, error) { return 0, nil }

func f(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func TestSwingAnalyzeHTTP(t *testing.T) {
	m := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{domain.ExchangeBinance: swingPort{}}, swingSupply{})
	h := NewSwingHandler(swing.New(m, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/swing?exchange=binance&symbol=ETHUSDT", nil)
	rec := httptest.NewRecorder()
	h.Analyze(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["symbol"] != "ETHUSDT" {
		t.Fatalf("%v", body)
	}
	if body["note"] == "" {
		t.Fatal("expected disclaimer note")
	}
}
