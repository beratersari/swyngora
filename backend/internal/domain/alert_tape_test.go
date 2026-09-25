package domain

import "testing"

func TestTapeAlertKinds(t *testing.T) {
	k, ok := NormalizeAlertKind("funding")
	if !ok || k != AlertKindFunding {
		t.Fatal(k, ok)
	}
	if err := ValidateAlertSpec(AlertKindFunding, "above", 0.05, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindFunding, "above", -0.02, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindOpenInterest, "below", 1_000_000_000, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindTaker, "above", 60, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindFearGreed, "below", 25, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindFearGreed, "above", 120, 0); err == nil {
		t.Fatal("score out of range")
	}
}

func TestTapeAlertValues(t *testing.T) {
	fund, ok := FundingAlertPercent(&FundingSnapshot{Current: &FundingCurrent{RatePct: "0.0300%"}})
	if !ok || fund < 0.029 || fund > 0.031 {
		t.Fatalf("funding %v %v", fund, ok)
	}
	oi, ok := OpenInterestAlertUSDT(&OpenInterestSnapshot{Current: OpenInterestLevel{Value: "2500000000"}})
	if !ok || oi != 2_500_000_000 {
		t.Fatalf("oi %v", oi)
	}
	buy, ok := TakerAlertBuyPercent(&TakerFlowReport{Combined: &TakerVenueFlow{Windows: []TakerWindowFlow{
		{Window: "5m", BuyNotional: 70, SellNotional: 30, BuyShare: 0.7},
	}}})
	if !ok || buy < 69 || buy > 71 {
		t.Fatalf("taker %v", buy)
	}
	if !AlertConditionMet(AlertAbove, buy, 60) {
		t.Fatal("60 buy share should be above 60")
	}
}

func TestFundingAlertReading_AnyVenue(t *testing.T) {
	snap := &FundingSnapshot{
		Exchange: "all",
		Venues: []FundingVenueSnap{
			{Exchange: "binance", Current: FundingCurrent{RatePct: "0.0100"}},
			{Exchange: "bybit", Current: FundingCurrent{RatePct: "0.0800"}},
			{Exchange: "okx", Current: FundingCurrent{RatePct: "-0.0200"}},
		},
	}
	pct, venue, ok := FundingAlertReading(snap, AlertAbove)
	if !ok || venue != "bybit" || pct < 0.079 || pct > 0.081 {
		t.Fatalf("above should follow the highest venue, got %v %s %v", pct, venue, ok)
	}
	if !AlertConditionMet(AlertAbove, pct, 0.05) {
		t.Fatal("0.08 should be above 0.05 even though binance is 0.01")
	}
	pct, venue, ok = FundingAlertReading(snap, AlertBelow)
	if !ok || venue != "okx" || pct > -0.019 || pct < -0.021 {
		t.Fatalf("below should follow the lowest venue, got %v %s %v", pct, venue, ok)
	}
	one := &FundingSnapshot{Exchange: "binance", Current: &FundingCurrent{RatePct: "0.0100"}}
	pct, venue, ok = FundingAlertReading(one, AlertAbove)
	if !ok || venue != "binance" || pct < 0.009 || pct > 0.011 {
		t.Fatalf("single venue %v %s %v", pct, venue, ok)
	}
}
