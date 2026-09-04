package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeRadarReview struct {
	rev *domain.RadarReview
}

func (f *fakeRadarReview) Review(context.Context, string, string, string, int) (*domain.RadarReview, error) {
	return f.rev, nil
}
func (f *fakeRadarReview) RecordHits(context.Context, *domain.ForensicsScanReport) (int, error) {
	return 0, nil
}

func TestReviewRadar_OK(t *testing.T) {
	now := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	yes := true
	h := NewMarketHandler(nil)
	h.Radar = &fakeRadarReview{rev: &domain.RadarReview{
		Exchange: "all", Since: "7d", AsOf: now,
		Cases: []domain.RadarCase{{
			ID: "c1", Exchange: "binance", Symbol: "BTCUSDT", FoundAt: now.Add(-2 * time.Hour),
			Similarity: 80, Lean: domain.CVDDirUp, WhySimilar: "OI matched",
			After: []domain.RadarAfter{{
				Horizon: "1h", FilledAt: &now, ReturnPct: 1.2, Direction: domain.CVDDirUp, Useful: &yes,
			}},
		}},
		Summary: "Saved 1 setup(s).",
		Note:    domain.RadarCaseDisclaimer,
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/forensics/radar?since=7d", nil)
	rr := httptest.NewRecorder()
	h.ReviewRadar(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body radarReviewDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Summary == "" || len(body.Cases) != 1 || body.Cases[0].Symbol != "BTCUSDT" {
		t.Fatalf("%+v", body)
	}
	if body.Cases[0].After[0].Useful == nil || !*body.Cases[0].After[0].Useful {
		t.Fatalf("useful %+v", body.Cases[0].After)
	}
}
