package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestFearGreedToDTO_IncludesPullAndPast(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	rep := &domain.FearGreedReport{
		Score: 62, Level: domain.FearGreedGreed, Label: "Greed", AsOf: now,
		Used: 7, Total: 8, Complete: false,
		Top:     &domain.FearGreedFactor{ID: domain.FearGreedFactorFunding, Name: "Funding", Reading: 80, Pull: 8.2, Toward: domain.FearGreedTowardGreed},
		Factors: []domain.FearGreedFactor{{ID: domain.FearGreedFactorFunding, Name: "Funding", Reading: 80, Pull: 8.2, Toward: domain.FearGreedTowardGreed}},
		Missing: []string{domain.FearGreedFactorTaker},
		History: []domain.FearGreedSample{{Time: past, Score: 55, Level: domain.FearGreedNeutral}},
		Windows: []domain.FearGreedWindow{{
			Window: "1h", PastScore: 55, PastLevel: domain.FearGreedNeutral,
			Score: 62, Level: domain.FearGreedGreed, Change: 7, HasPast: true, Complete: true, SampleTime: past,
		}},
		Summary: "Greed 62.",
		Note:    "own tape",
	}
	body := fearGreedToDTO(rep)
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["score"].(float64) != 62 || got["level"] != "greed" {
		t.Fatalf("%s", raw)
	}
	top := got["top"].(map[string]any)
	if top["id"] != "funding" || top["pull"] != "+8.2" {
		t.Fatalf("top %v", top)
	}
	wins := got["windows"].([]any)
	w0 := wins[0].(map[string]any)
	if w0["pastScore"].(float64) != 55 || w0["change"].(float64) != 7 {
		t.Fatalf("window %v", w0)
	}
}

func TestGetFearGreed_NeedsInputs(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/fear-greed", nil)
	rr := httptest.NewRecorder()
	h.GetFearGreed(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
