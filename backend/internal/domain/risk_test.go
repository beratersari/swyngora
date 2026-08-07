package domain

import (
	"errors"
	"math"
	"testing"
)

func TestDailyLossLimitHit(t *testing.T) {
	if !DailyLossLimitHit(10000, 9400, 5) { // -6%
		t.Fatal("expected hit")
	}
	if DailyLossLimitHit(10000, 9600, 5) { // -4%
		t.Fatal("4% should pass 5% cap")
	}
	if DailyLossLimitHit(10000, 10500, 5) {
		t.Fatal("profit is not a loss limit")
	}
}

func TestDailyLossLimitHitExact(t *testing.T) {
	if !DailyLossLimitHit(10000, 9500, 5) {
		t.Fatal("exactly 5% loss should hit")
	}
}

func TestAssetWeightLimitHit(t *testing.T) {
	// equity 10k, BTC 2k, buy 1.2k → 32%
	if !AssetWeightLimitHit(10000, 2000, 1200, 30) {
		t.Fatal("expected concentration hit")
	}
	if AssetWeightLimitHit(10000, 2000, 500, 30) { // 25%
		t.Fatal("under limit")
	}
	// already over: any new buy hits
	if !AssetWeightLimitHit(10000, 6500, 1, 30) {
		t.Fatal("already over 30%")
	}
}

func TestValidateOptionalRiskPct(t *testing.T) {
	if err := ValidateOptionalRiskPct(nil, "x"); err != nil {
		t.Fatal(err)
	}
	v := 5.0
	if err := ValidateOptionalRiskPct(&v, "x"); err != nil {
		t.Fatal(err)
	}
	bad := 0.0
	if err := ValidateOptionalRiskPct(&bad, "maxDailyLossPct"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestRiskLimitMessages(t *testing.T) {
	d, w := 5.0, 30.0
	lim := RiskLimits{MaxDailyLossPct: &d, MaxAssetWeightPct: &w}
	rs := RiskLimitMessages(lim, 10000, 9400, "BTC", 2000, 100)
	if len(rs) != 1 {
		t.Fatalf("%v", rs)
	}
	rs = RiskLimitMessages(lim, 10000, 10000, "BTC", 2000, 2000)
	if len(rs) != 1 || rs[0] == "" {
		t.Fatalf("%v", rs)
	}
	_ = math.Abs
}
