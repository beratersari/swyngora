package domain

import (
	"testing"
	"time"
)

func TestNormalizeAlertKind_SetupRadar(t *testing.T) {
	k, ok := NormalizeAlertKind("setup_radar")
	if !ok || k != AlertKindSetupRadar {
		t.Fatalf("%s %v", k, ok)
	}
	if !IsSetupRadarAlert("setup_radar") {
		t.Fatal("IsSetupRadarAlert")
	}
}

func TestValidateAlertSpec_SetupRadar(t *testing.T) {
	if err := ValidateAlertSpec(AlertKindSetupRadar, "", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindSetupRadar, "any", 60, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindSetupRadar, "up", 80, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindSetupRadar, "down", 40, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindSetupRadar, "above", 60, 0); err == nil {
		t.Fatal("bad condition")
	}
	if err := ValidateAlertSpec(AlertKindSetupRadar, "any", 10, 0); err == nil {
		t.Fatal("tiny similarity")
	}
	if err := ValidateAlertSpec(AlertKindSetupRadar, "any", 101, 0); err == nil {
		t.Fatal("huge similarity")
	}
}

func TestSetupRadarMinSimilarity(t *testing.T) {
	if SetupRadarMinSimilarity(PriceAlert{}) != DefaultSetupRadarMinSimilarity {
		t.Fatal("default")
	}
	if SetupRadarMinSimilarity(PriceAlert{TargetPrice: 75}) != 75 {
		t.Fatal("explicit")
	}
}

func TestSetupRadarObservationFromSimilar(t *testing.T) {
	rep := &AroundSimilarReport{
		Symbol: "BTCUSDT", Exchange: "binance", Events: 3,
		UpAfter: 2, DownAfter: 1,
		Matches: []AroundSimilarHit{{
			Similarity: 78, Coverage: 90, AfterDirection: CVDDirUp, AfterReturnPct: 3.2,
			Move: AroundMove{At: time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)},
		}},
		AfterHorizons: []AroundSimilarHorizonStat{
			{Horizon: "1h", Up: 3, Down: 0, Sample: 3, Events: 3, MedianPct: 2.1},
		},
	}
	a := PriceAlert{
		Kind: AlertKindSetupRadar, Exchange: ExchangeBinance, Symbol: "BTCUSDT",
		Condition: AlertRadarAny, TargetPrice: 60,
		Mode: AlertModeRepeating, Armed: true, Status: AlertStatusActive,
	}
	met, sim, d := SetupRadarObservationFromSimilar(a, rep)
	if !met || sim != 78 || d.Lean != CVDDirUp || d.Symbol != "BTCUSDT" {
		t.Fatalf("met=%v sim=%v %+v", met, sim, d)
	}
	a.TargetPrice = 90
	met, _, _ = SetupRadarObservationFromSimilar(a, rep)
	if met {
		t.Fatal("below floor")
	}
	a.TargetPrice = 60
	a.Condition = AlertRadarDown
	met, _, _ = SetupRadarObservationFromSimilar(a, rep)
	if met {
		t.Fatal("lean filter")
	}
	if met, _, _ = SetupRadarObservationFromSimilar(a, &AroundSimilarReport{}); met {
		t.Fatal("empty")
	}
}

func TestSetupRadarObservationFromScan(t *testing.T) {
	scan := &ForensicsScanReport{
		Exchange: "all",
		Hits: []ForensicsScanHit{
			{Symbol: "ETHUSDT", Exchange: "binance", Similarity: 55, Lean: CVDDirUp, AfterStory: "rose"},
			{Symbol: "BTCUSDT", Exchange: "binance", Similarity: 82, Lean: CVDDirDown, AfterStory: "fell", Matches: 3},
		},
	}
	a := PriceAlert{
		Kind: AlertKindSetupRadar, Exchange: Exchange("all"), Symbol: LiqAlertSymbolAll,
		Condition: AlertRadarAny, TargetPrice: 60,
	}
	met, sim, d := SetupRadarObservationFromScan(a, scan)
	if !met || sim != 82 || d.Symbol != "BTCUSDT" || len(d.Symbols) != 1 {
		t.Fatalf("met=%v sim=%v %+v", met, sim, d)
	}
	// ETH is under the floor; only BTC passes — still met (radar stays hot).
	scan.Hits = append(scan.Hits, ForensicsScanHit{
		Symbol: "SOLUSDT", Exchange: "binance", Similarity: 70, Lean: CVDDirUp, AfterStory: "rose",
	})
	met2, _, d2 := SetupRadarObservationFromScan(a, scan)
	if !met2 || len(d2.Symbols) != 2 {
		t.Fatalf("still hot %+v", d2)
	}
	a.Condition = AlertRadarUp
	met, _, d = SetupRadarObservationFromScan(a, scan)
	if !met || d.Symbol != "SOLUSDT" || len(d.Symbols) != 1 {
		t.Fatalf("up lean %+v", d)
	}
	empty := &ForensicsScanReport{Hits: []ForensicsScanHit{}}
	if met, _, _ = SetupRadarObservationFromScan(a, empty); met {
		t.Fatal("no hits")
	}
}
