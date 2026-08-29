package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetRSIHeatmap_OK(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/rsi-heatmap?interval=1h&limit=5", nil)
	rr := httptest.NewRecorder()
	h.GetRSIHeatmap(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body rsiHeatmapResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Exchange != "binance" || body.Period != 14 {
		t.Fatalf("%+v", body)
	}
	if body.Interval != "1h" {
		t.Fatalf("interval=%q", body.Interval)
	}
	if body.Note == "" {
		t.Fatal("expected disclaimer")
	}
}

func TestGetRSIHeatmap_BadLimit(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/rsi-heatmap?limit=nope", nil)
	rr := httptest.NewRecorder()
	h.GetRSIHeatmap(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetRSIHeatmap_BadPeriod(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/rsi-heatmap?period=nope", nil)
	rr := httptest.NewRecorder()
	h.GetRSIHeatmap(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetRSIHeatmap_IntervalsAliasAndPeriod(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/rsi-heatmap?intervals=4h&period=14&sort=quoteVolume&limit=2", nil)
	rr := httptest.NewRecorder()
	h.GetRSIHeatmap(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body rsiHeatmapResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Interval != "4h" || body.Period != 14 || body.Sort != "quoteVolume" {
		t.Fatalf("%+v", body)
	}
	if len(body.Items) == 0 {
		t.Fatal("expected items")
	}
}

func TestGetRSIHeatmap_ServiceError(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/rsi-heatmap?exchange=not-a-venue", nil)
	rr := httptest.NewRecorder()
	h.GetRSIHeatmap(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("want error, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestRSIHeatmapToDTO_NilAndStaleRow(t *testing.T) {
	empty := rsiHeatmapToDTO(nil)
	if empty.Exchange != "" || empty.Items != nil {
		t.Fatalf("%+v", empty)
	}
	rsi := 22.5
	mcap := 1e9
	got := rsiHeatmapToDTO(&domain.RSIHeatmap{
		Exchange: domain.ExchangeBinance, Quote: "USDT", Interval: "1h", Period: 14,
		Oversold: 30, Overbought: 70, SortBy: "quoteVolume", AverageRSI: &rsi,
		OversoldCount: 1, NeutralCount: 0, OverboughtCount: 0,
		AsOf: time.Unix(1_700_000_000, 0).UTC(), Stale: true, Note: "n",
		Items: []domain.RSIHeatmapRow{{
			Rank: 1, Symbol: "BTCUSDT", Base: "BTC", LastPrice: "1",
			PriceChangePercent: "2", QuoteVolume: "3", MarketCapCirculating: &mcap,
			RSI: &rsi, Zone: domain.RSIZoneOversold, Error: "",
		}},
	})
	if !got.Stale || got.Items[0].Base != "BTC" || got.Items[0].MarketCapCirculating == nil {
		t.Fatalf("%+v", got)
	}
}
