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

func TestGetUnusual_NotConfigured(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/unusual?symbol=BTCUSDT", nil)
	rr := httptest.NewRecorder()
	h.GetUnusual(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body domain.UnusualReport
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Symbol != "BTCUSDT" || body.Exchange != "all" || len(body.Venues) != 3 {
		t.Fatalf("%+v", body)
	}
	if body.Venues[2].Exchange != domain.ExchangeOKX {
		t.Fatalf("venues %+v", body.Venues)
	}
	printSize := readingByMetric(t, body.Venues[0], domain.UnusualMetricPrintSize)
	if printSize.Reason != domain.UnusualReasonNoStoredHistory {
		t.Fatalf("%+v", printSize)
	}
	okxSpread := readingByMetric(t, body.Venues[2], domain.UnusualMetricSpread)
	if okxSpread.Reason != domain.UnusualReasonNoBook {
		t.Fatalf("%+v", okxSpread)
	}
}

func TestGetUnusual_BadSymbol(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/unusual", nil)
	rr := httptest.NewRecorder()
	h.GetUnusual(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetUnusual_RejectsSpotVenue(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/unusual?symbol=BTCUSDT&exchange=coinbase", nil)
	rr := httptest.NewRecorder()
	h.GetUnusual(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

type stubUnusualHist struct{}

func (stubUnusualHist) History(context.Context, domain.FuturesHistoryQuery) (any, error) {
	return nil, nil
}

func (stubUnusualHist) HourlyOpenInterest(context.Context, string, string, time.Time) ([]domain.UnusualSample, error) {
	return nil, nil
}

func (stubUnusualHist) SettledFunding(context.Context, string, string, time.Time) ([]domain.UnusualSample, error) {
	now := time.Now().UTC()
	out := make([]domain.UnusualSample, 21)
	for i := range out {
		out[i] = domain.UnusualSample{
			At:    now.Add(-time.Duration(21-i) * 8 * time.Hour),
			Value: float64(i+1) / 100000,
		}
	}
	return out, nil
}

func (stubUnusualHist) LatestPredictedFunding(context.Context, string, string) (domain.UnusualSample, bool, error) {
	return domain.UnusualSample{}, false, nil
}

func (stubUnusualHist) HourlyLiquidationNotional(context.Context, string, string, time.Time) ([]domain.UnusualSample, error) {
	return nil, nil
}

func TestGetUnusual_ScoresFunding(t *testing.T) {
	svc := market.New(stubMarket{}, stubSupply{})
	svc.SetFuturesHistory(stubUnusualHist{})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/unusual?symbol=BTCUSDT&exchange=binance", nil)
	rr := httptest.NewRecorder()
	h.GetUnusual(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body domain.UnusualReport
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Venues) != 1 || body.Venues[0].Exchange != domain.ExchangeBinance {
		t.Fatalf("%+v", body.Venues)
	}
	funding := readingByMetric(t, body.Venues[0], domain.UnusualMetricFunding)
	if funding.Band != domain.UnusualBandExtremeHigh || funding.Samples != 20 || funding.Label != domain.UnusualLabelSettled {
		t.Fatalf("%+v", funding)
	}
	if funding.Summary == "" || body.Note == "" {
		t.Fatalf("summary %q note %q", funding.Summary, body.Note)
	}
}

func TestScanUnusual_ScoresFunding(t *testing.T) {
	svc := market.New(stubMarket{}, stubSupply{})
	svc.SetFuturesHistory(stubUnusualHist{})
	h := NewMarketHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/unusual/scan?exchange=binance&metric=funding&limit=5", nil)
	rr := httptest.NewRecorder()
	h.ScanUnusual(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body domain.UnusualScan
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Metric != "funding" || body.Universe != 5 || len(body.Hits) == 0 {
		t.Fatalf("%+v", body)
	}
	for _, hit := range body.Hits {
		if hit.Metric != domain.UnusualMetricFunding || hit.Band != domain.UnusualBandExtremeHigh || hit.Exchange != "binance" {
			t.Fatalf("%+v", hit)
		}
	}
}

func TestScanUnusual_RejectsPrintSize(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/unusual/scan?metric=print_size", nil)
	rr := httptest.NewRecorder()
	h.ScanUnusual(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestScanUnusual_RejectsBadLimit(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/unusual/scan?limit=nope", nil)
	rr := httptest.NewRecorder()
	h.ScanUnusual(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func readingByMetric(t *testing.T, venue domain.UnusualVenue, metric string) domain.UnusualReading {
	t.Helper()
	for _, r := range venue.Readings {
		if r.Metric == metric {
			return r
		}
	}
	t.Fatalf("missing %s in %+v", metric, venue.Readings)
	return domain.UnusualReading{}
}
