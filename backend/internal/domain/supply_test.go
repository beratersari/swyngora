package domain

import (
	"testing"
	"time"
)

// Smoke tests for AssetSupply field layout (types-only domain model).

func TestAssetSupply_FieldsAndNullables(t *testing.T) {
	circ := 19_800_000.0
	total := 19_800_000.0
	max := 21_000_000.0
	price := 64000.0
	asOf := time.Unix(1_700_000_000, 0).UTC()

	s := AssetSupply{
		Asset:             "BTC",
		Name:              "Bitcoin",
		ProviderID:        "bitcoin",
		CirculatingSupply: &circ,
		TotalSupply:       &total,
		MaxSupply:         &max,
		CurrentPriceUSD:   &price,
		AsOf:              asOf,
		Source:            "binance",
	}
	if s.Asset != "BTC" || s.Source != "binance" {
		t.Fatalf("supply=%+v", s)
	}
	if s.CirculatingSupply == nil || *s.CirculatingSupply != circ {
		t.Fatalf("circulating=%v", s.CirculatingSupply)
	}
	if s.MaxSupply == nil || *s.MaxSupply != max {
		t.Fatalf("max=%v", s.MaxSupply)
	}

	// Assets without a hard cap leave MaxSupply nil (e.g. ETH-style).
	noCap := AssetSupply{Asset: "ETH", Source: "binance"}
	if noCap.MaxSupply != nil || noCap.CirculatingSupply != nil {
		t.Fatalf("expected nil optional fields, got %+v", noCap)
	}
}
