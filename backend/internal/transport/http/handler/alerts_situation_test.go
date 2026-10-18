package handler

import (
	"encoding/json"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestAlertToDTO_SituationFireAndWatch(t *testing.T) {
	at := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	dto := alertToDTO(&domain.PriceAlert{
		ID: "a1", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Kind: domain.AlertKindSituation, Condition: "leaves", Mode: domain.AlertModeRepeating,
		Armed: true, Status: domain.AlertStatusActive, CreatedAt: at, TriggeredAt: &at,
		SituationWatch: domain.SituationWatchBlank,
		SituationFire: &domain.SituationFire{
			Comparison: domain.SituationComparison{Clauses: []domain.SituationClauseLeft{{
				Clause: "priceCvd", Status: domain.SituationLeftChanged,
				Entry: domain.SituationCVDPriceUp, Live: domain.SituationCVDAgrees,
			}}},
			Entry: &domain.CoinSituation{PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDPriceUp}},
			Live:  &domain.CoinSituation{PriceCVD: &domain.SituationPriceCVD{Kind: domain.SituationCVDAgrees}},
		},
	})
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["situationWatch"] != "blank" {
		t.Fatalf("watch %#v", body["situationWatch"])
	}
	fire, _ := body["situationFire"].(map[string]any)
	cmp, _ := fire["comparison"].(map[string]any)
	clauses, _ := cmp["clauses"].([]any)
	if len(clauses) != 1 {
		t.Fatalf("body %s", raw)
	}
	if body["triggeredAt"] == "" || body["triggeredAt"] == nil {
		t.Fatalf("time %#v", body["triggeredAt"])
	}
}

func TestAlertToDTO_SituationMatch(t *testing.T) {
	dto := alertToDTO(&domain.PriceAlert{
		ID: "a2", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT",
		Kind: domain.AlertKindSituation, Condition: domain.AlertConditionMatches,
		Mode: domain.AlertModeOneTime, Status: domain.AlertStatusActive, CreatedAt: time.Now().UTC(),
		SituationWatch: domain.SituationWatchSelected,
		SituationMatch: &domain.SituationMatch{PriceCVD: domain.SituationCVDPriceUp, Funding: "long"},
	})
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["situationWatch"] != "selected" || body["condition"] != "matches" {
		t.Fatalf("%s", raw)
	}
	match, _ := body["situationMatch"].(map[string]any)
	if match["priceCvd"] != domain.SituationCVDPriceUp || match["funding"] != "long" || match["prints"] != nil {
		t.Fatalf("match %#v", match)
	}
}
