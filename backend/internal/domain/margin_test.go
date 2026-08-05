package domain

import (
	"math"
	"testing"
)

func TestInitialMargin(t *testing.T) {
	m, err := InitialMargin(2, 100, 5)
	if err != nil || math.Abs(m-40) > 1e-9 {
		t.Fatalf("m=%v err=%v", m, err)
	}
	if _, err := InitialMargin(1, 1, 0); err == nil {
		t.Fatal("expected leverage error")
	}
}

func TestLiquidationPriceIsolated(t *testing.T) {
	// long 10x entry 100, mmr 0.5% → 100*(1-0.1+0.005)=90.5
	liq, err := LiquidationPriceIsolated(MarginLong, 100, 10, 0.005)
	if err != nil || math.Abs(liq-90.5) > 1e-9 {
		t.Fatalf("long liq=%v err=%v", liq, err)
	}
	// short 10x → 100*(1+0.1-0.005)=109.5
	liq, err = LiquidationPriceIsolated(MarginShort, 100, 10, 0.005)
	if err != nil || math.Abs(liq-109.5) > 1e-9 {
		t.Fatalf("short liq=%v err=%v", liq, err)
	}
}

func TestMarginPnL(t *testing.T) {
	// long: entry 100 mark 110 qty 2 → +20
	if u := MarginUnrealizedPnL(MarginLong, 2, 100, 110); math.Abs(u-20) > 1e-9 {
		t.Fatalf("%v", u)
	}
	// short: entry 100 mark 90 qty 2 → +20
	if u := MarginUnrealizedPnL(MarginShort, 2, 100, 90); math.Abs(u-20) > 1e-9 {
		t.Fatalf("%v", u)
	}
}

func TestShouldLiquidateAndBrackets(t *testing.T) {
	if !ShouldLiquidate(MarginLong, 90, 90.5) {
		t.Fatal("long should liq")
	}
	if ShouldLiquidate(MarginLong, 95, 90.5) {
		t.Fatal("long should not liq")
	}
	if !ShouldLiquidate(MarginShort, 110, 109.5) {
		t.Fatal("short should liq")
	}
	sl := 95.0
	tp := 110.0
	if !ShouldTriggerStopLoss(MarginLong, 94, &sl) {
		t.Fatal("long sl")
	}
	if !ShouldTriggerTakeProfit(MarginLong, 111, &tp) {
		t.Fatal("long tp")
	}
	if !ShouldTriggerStopLoss(MarginShort, 96, &sl) {
		t.Fatal("short sl")
	}
	if err := ValidateMarginBrackets(MarginLong, 100, &sl, &tp); err != nil {
		t.Fatal(err)
	}
	bad := 105.0
	if err := ValidateMarginBrackets(MarginLong, 100, &bad, nil); err == nil {
		t.Fatal("expected bad long sl")
	}
}

func TestMarginLimitTriggered(t *testing.T) {
	if !MarginLimitTriggered(MarginLong, 100, 99) {
		t.Fatal("long limit")
	}
	if MarginLimitTriggered(MarginLong, 100, 101) {
		t.Fatal("long limit no fill")
	}
	if !MarginLimitTriggered(MarginShort, 100, 101) {
		t.Fatal("short limit")
	}
}
