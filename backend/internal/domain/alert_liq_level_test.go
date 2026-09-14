package domain

import (
	"testing"
	"time"
)

func TestNormalizeAlertKind_LiquidationLevel(t *testing.T) {
	k, ok := NormalizeAlertKind("liquidation_level")
	if !ok || k != AlertKindLiqLevel {
		t.Fatal(k, ok)
	}
	if err := ValidateAlertSpec(AlertKindLiqLevel, "high", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindLiqLevel, "any", 0.4, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindLiqLevel, "above", 0, 0); err == nil {
		t.Fatal("expected bad condition")
	}
	if err := ValidateAlertSpec(AlertKindLiqLevel, "any", 2, 0); err == nil {
		t.Fatal("expected bad confidence")
	}
}

func TestLevelAlertObservation_HighConfidenceOnly(t *testing.T) {
	t0 := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	bands := []LevelAlertBand{
		{Leverage: 100, Side: LiquidationSideShort, Price: 100.6, Lo: 100.3, Hi: 100.9, Confidence: 0.92, Reliability: LiquidationLevelsReliabilityHigh},
		{Leverage: 10, Side: LiquidationSideShort, Price: 109.6, Lo: 107, Hi: 112, Confidence: 0.15, Reliability: LiquidationLevelsReliabilityLow},
	}
	a := PriceAlert{Kind: AlertKindLiqLevel, Exchange: ExchangeBinance, Symbol: "BTCUSDT", Condition: "high"}
	met, conf, d := LevelAlertObservation(a, 100.6, bands, nil, t0)
	if !met || d.Leverage != 100 || d.Importance != AlertImportanceHigh || conf < 0.9 {
		t.Fatalf("want high-confidence 100x, got met=%v %+v", met, d)
	}

	// Price in the weak 10x band only.
	met, _, d = LevelAlertObservation(a, 110, bands, nil, t0)
	if met {
		t.Fatalf("high-min must ignore low-confidence 10x %+v", d)
	}

	anyA := PriceAlert{Kind: AlertKindLiqLevel, Exchange: ExchangeBinance, Symbol: "BTCUSDT", Condition: "any"}
	met, _, d = LevelAlertObservation(anyA, 110, bands, nil, t0)
	if met {
		t.Fatalf("default min 0.35 must still skip 0.15 confidence %+v", d)
	}

	lowOK := PriceAlert{Kind: AlertKindLiqLevel, Exchange: ExchangeBinance, Symbol: "BTCUSDT", Condition: "any", TargetPrice: 0.10}
	met, _, d = LevelAlertObservation(lowOK, 110, bands, nil, t0)
	if !met || d.Leverage != 10 || d.Importance != AlertImportanceLow {
		t.Fatalf("explicit low floor should fire 10x %+v", d)
	}
}

func TestAnnotateNotionalWithLevels_ImportanceFollowsBand(t *testing.T) {
	t0 := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	bands := []LevelAlertBand{
		{Leverage: 100, Side: LiquidationSideShort, Price: 100.6, Lo: 100.3, Hi: 100.9, Confidence: 0.9, Reliability: LiquidationLevelsReliabilityHigh},
		{Leverage: 10, Side: LiquidationSideShort, Price: 110, Lo: 108, Hi: 112, Confidence: 0.15, Reliability: LiquidationLevelsReliabilityLow},
	}
	detail := LiqNotionalAlertDetail{Side: "both", Window: "5m"}
	ev := []LiquidationEvent{{
		Side: LiquidationSideShort, Price: 100.6, Notional: 50_000, Time: t0.Add(-time.Minute),
	}}
	AnnotateNotionalWithLevels(&detail, 100, ev, bands, t0)
	if detail.Importance != AlertImportanceHigh || detail.Level == nil || detail.Level.Leverage != 100 {
		t.Fatalf("%+v", detail)
	}

	miss := LiqNotionalAlertDetail{Side: "both", Window: "5m"}
	AnnotateNotionalWithLevels(&miss, 100, []LiquidationEvent{{
		Side: LiquidationSideShort, Price: 150, Notional: 50_000, Time: t0.Add(-time.Minute),
	}}, bands, t0)
	if miss.Importance != AlertImportanceLow || miss.Level != nil {
		t.Fatalf("unmapped wave must stay low %+v", miss)
	}
}

func TestLevelAlertImportance(t *testing.T) {
	if LevelAlertImportance(0.9) != AlertImportanceHigh {
		t.Fatal("high")
	}
	if LevelAlertImportance(0.5) != AlertImportanceMedium {
		t.Fatal("medium")
	}
	if LevelAlertImportance(0.2) != AlertImportanceLow {
		t.Fatal("low")
	}
}
