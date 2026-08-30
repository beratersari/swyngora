package domain

import (
	"testing"
	"time"
)

func TestNormalizeAlertKind_Liquidation(t *testing.T) {
	k, ok := NormalizeAlertKind("liquidation_feed")
	if !ok || k != AlertKindLiqFeed {
		t.Fatalf("%s %v", k, ok)
	}
	k, ok = NormalizeAlertKind("liquidation_cascade")
	if !ok || k != AlertKindLiqCascade {
		t.Fatalf("%s %v", k, ok)
	}
}

func TestValidateAlertSpec_Liquidation(t *testing.T) {
	if err := ValidateAlertSpec(AlertKindLiqFeed, "", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindLiqFeed, "down", 120, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindLiqFeed, "down", 10, 0); err == nil {
		t.Fatal("tiny threshold")
	}
	if err := ValidateAlertSpec(AlertKindLiqCascade, "cascade", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindLiqCascade, "above", 0, 0); err == nil {
		t.Fatal("bad cascade condition")
	}
}

func TestFeedAlertObservation_NoRetriggerWhileDown(t *testing.T) {
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	a := PriceAlert{
		Kind: AlertKindLiqFeed, Exchange: ExchangeBybit, Symbol: LiqAlertSymbolAll,
		TargetPrice: 300, CreatedAt: now.Add(-time.Hour), Mode: AlertModeRepeating, Armed: true,
		Status: AlertStatusActive,
	}
	feed := LiquidationFeed{Venues: []LiquidationVenueHealth{{
		Exchange: "bybit", Live: false, LastSeenAt: now.Add(-10 * time.Minute),
	}}}
	met, sec, d := FeedAlertObservation(a, feed, now)
	if !met || sec < 599 || d.Exchange != "bybit" {
		t.Fatalf("met=%v sec=%v %+v", met, sec, d)
	}
	ev := EvaluateAlertState(a, met)
	if !ev.Fire {
		t.Fatal("first fire")
	}
	a.Armed = false
	ev = EvaluateAlertState(a, true)
	if ev.Fire {
		t.Fatal("must not re-fire while still down")
	}
	ev = EvaluateAlertState(a, false)
	if !ev.NewArmed {
		t.Fatal("re-arm when live")
	}
}

func TestFeedAlertObservation_LiveWithLeftoverGapIsHealthy(t *testing.T) {
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	a := PriceAlert{
		Kind: AlertKindLiqFeed, Exchange: ExchangeBinance, Symbol: LiqAlertSymbolAll,
		TargetPrice: 300, CreatedAt: now.Add(-time.Hour), Status: AlertStatusActive,
	}
	feed := LiquidationFeed{Venues: []LiquidationVenueHealth{{
		Exchange: "binance", Live: true, LastSeenAt: now,
		MissingSeconds: 900,
		Gaps:           []LiquidationGap{{From: now.Add(-20 * time.Minute), To: now.Add(-5 * time.Minute), Seconds: 900, MissingSeconds: 900}},
	}}}
	met, _, _ := FeedAlertObservation(a, feed, now)
	if met {
		t.Fatal("reconnected feed must not stay alerting on an old unfilled hole")
	}
}

func TestFeedAlertObservation_AllAnyVenue(t *testing.T) {
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	a := PriceAlert{
		Kind: AlertKindLiqFeed, Exchange: Exchange("all"), Symbol: LiqAlertSymbolAll,
		TargetPrice: 60, CreatedAt: now.Add(-time.Hour), Status: AlertStatusActive,
	}
	feed := LiquidationFeed{Venues: []LiquidationVenueHealth{
		{Exchange: "binance", Live: true, LastSeenAt: now},
		{Exchange: "bybit", Live: false, LastSeenAt: now.Add(-2 * time.Minute)},
	}}
	met, _, d := FeedAlertObservation(a, feed, now)
	if !met || len(d.Missing) != 1 || d.Missing[0] != "bybit" {
		t.Fatalf("%v %+v", met, d)
	}
}

func TestCascadeAlertObservation_CoinAndVenue(t *testing.T) {
	a := PriceAlert{
		Kind: AlertKindLiqCascade, Exchange: Exchange("all"), Symbol: "BTCUSDT",
		Condition: "cascade", Status: AlertStatusActive,
	}
	rep := &CascadeReport{
		Symbol: "BTCUSDT",
		Venues: []CascadeVenue{
			{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Grade: CascadeGradeElevated, Score: 2, Side: "long"},
			{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Grade: CascadeGradeCascade, Score: 4.5, Side: "short", Hottest: "1m", Summary: "bybit short cascade"},
		},
	}
	met, score, d := CascadeAlertObservation(a, rep)
	if !met || score != 4.5 || d.Exchange != "bybit" || d.Symbol != "BTCUSDT" || d.Grade != CascadeGradeCascade {
		t.Fatalf("%v %v %+v", met, score, d)
	}
	a.Condition = CascadeGradeExtreme
	met, _, _ = CascadeAlertObservation(a, rep)
	if met {
		t.Fatal("extreme should not fire on cascade")
	}
}

func TestCascadeScanAlertObservation_HottestCoin(t *testing.T) {
	a := PriceAlert{
		Kind: AlertKindLiqCascade, Exchange: Exchange("all"), Symbol: LiqAlertSymbolAll,
		Condition: "cascade", Status: AlertStatusActive,
	}
	scan := &CascadeScan{Hits: []CascadeHit{
		{Symbol: "ETHUSDT", Grade: CascadeGradeCascade, Score: 3, Side: "long", Summary: "eth"},
		{Symbol: "BTCUSDT", Grade: CascadeGradeExtreme, Score: 8, Side: "short", Hottest: "5m", Summary: "btc"},
	}}
	met, score, d := CascadeScanAlertObservation(a, scan)
	if !met || score != 8 || d.Symbol != "BTCUSDT" || d.Grade != CascadeGradeExtreme {
		t.Fatalf("%v %v %+v", met, score, d)
	}
}
