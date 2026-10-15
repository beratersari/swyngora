package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestDerivativesGridMap_KeepsVenueZeroAndOmitsMissing(t *testing.T) {
	at := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	raw, err := json.Marshal(derivativesGridMap(&domain.DerivativesGrid{
		Exchange: "all",
		Metric:   domain.GridMetricLiquidations,
		Range:    domain.GridRange24h,
		AsOf:     at,
		Columns:  []time.Time{at},
		Rows: []domain.GridRow{{
			Symbol: "ETHUSDT",
			Base:   "ETH",
			Cells: []domain.GridCell{{
				Time: at, Definite: false, Missing: []string{"okx"},
				Venues: []domain.GridVenue{
					{Exchange: "bybit", Definite: true, HasValue: true, Value: 0, Text: "0"},
					{Exchange: "okx", Definite: false, Value: 9, Text: "leak"},
				},
			}},
		}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Rows []struct {
			Cells []struct {
				Venues []struct {
					Exchange string   `json:"exchange"`
					Value    *float64 `json:"value"`
					Text     string   `json:"text"`
					Definite bool     `json:"definite"`
				} `json:"venues"`
			} `json:"cells"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	venues := body.Rows[0].Cells[0].Venues
	if len(venues) != 2 || venues[0].Exchange != "bybit" || !venues[0].Definite || venues[0].Value == nil || *venues[0].Value != 0 || venues[0].Text != "0" {
		t.Fatalf("zero dropped %+v", venues)
	}
	if venues[1].Definite || venues[1].Value != nil || venues[1].Text != "" {
		t.Fatalf("missing leaked %+v", venues[1])
	}
}
