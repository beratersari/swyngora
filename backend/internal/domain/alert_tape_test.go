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
