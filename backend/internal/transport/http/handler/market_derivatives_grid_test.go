package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

func TestGetDerivativesGrid_OmitsWithheldValue(t *testing.T) {
	at := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	dto := derivativesGridToDTO(&domain.DerivativesGrid{
		Exchange: "all",
		Metric:   domain.GridMetricFunding,
		Range:    domain.GridRange24h,
		StepSec:  3600,
		AsOf:     at,
		Columns:  []time.Time{at},
		Rows: []domain.GridRow{{
			Symbol: "BTCUSDT",
			Base:   "BTC",
			Cells: []domain.GridCell{
				{Time: at, Definite: false, Missing: []string{"okx"}, Value: 99, Text: "nope"},
				{Time: at.Add(time.Hour), Definite: true, Value: 0, Text: "0%", Weight: "equal"},
			},
		}},
		Universe: 1,
	})
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Rows []struct {
			Cells []struct {
				Value    *float64 `json:"value"`
				Text     string   `json:"text"`
				Definite bool     `json:"definite"`
				Missing  []string `json:"missing"`
			} `json:"cells"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.Rows[0].Cells[0].Value != nil || body.Rows[0].Cells[0].Text != "" || body.Rows[0].Cells[0].Missing[0] != "okx" {
		t.Fatalf("withheld %+v", body.Rows[0].Cells[0])
	}
	if body.Rows[0].Cells[1].Value == nil || *body.Rows[0].Cells[1].Value != 0 || body.Rows[0].Cells[1].Text != "0%" {
		t.Fatalf("zero kept %+v", body.Rows[0].Cells[1])
	}
}

func TestGetDerivativesGrid_BadMetric(t *testing.T) {
	h := NewMarketHandler(market.New(nil, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/derivatives-grid?metric=price", nil)
	rr := httptest.NewRecorder()
	h.GetDerivativesGrid(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}
