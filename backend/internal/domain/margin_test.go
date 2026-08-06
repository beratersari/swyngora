package domain

import (
	"math"
	"testing"
	"time"
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
	// FromMargin with IM matches
	liq2, err := LiquidationPriceFromMargin(MarginLong, 100, 1, 10, 0.005)
	if err != nil || math.Abs(liq2-90.5) > 1e-9 {
		t.Fatalf("from margin=%v err=%v", liq2, err)
	}
	// Extra margin moves long liq lower
	liq3, _ := LiquidationPriceFromMargin(MarginLong, 100, 1, 20, 0.005)
	if liq3 >= liq2 {
		t.Fatalf("extra margin should lower long liq: %v vs %v", liq3, liq2)
	}
}

func TestCrossLiquidationPrice(t *testing.T) {
	// Single long: equityExcl = cash+margin = 100 (if cash=90 margin=10, no other U)
	// totalMaint = 0.5, uNeed = 0.5-100 = -99.5, mark = 100 + (-99.5)/1 = 0.5
	liq, err := CrossLiquidationPrice(MarginLong, 100, 1, 100, 0.5)
	if err != nil || math.Abs(liq-0.5) > 1e-9 {
		t.Fatalf("liq=%v err=%v", liq, err)
	}
}

func TestBorrowedPrincipalOnOpen(t *testing.T) {
	// long 1 @ 100, 5x → margin 20, borrow 80 cash
	p, a, err := BorrowedPrincipalOnOpen(MarginLong, 1, 100, 5)
	if err != nil || a != DebtAssetQuote || math.Abs(p-80) > 1e-9 {
		t.Fatalf("p=%v a=%s err=%v", p, a, err)
	}
	// short 2 @ 50, any lev → borrow 2 coins
	p, a, err = BorrowedPrincipalOnOpen(MarginShort, 2, 50, 10)
	if err != nil || a != DebtAssetBase || math.Abs(p-2) > 1e-9 {
		t.Fatalf("p=%v a=%s err=%v", p, a, err)
	}
}

func TestAllocateRepaymentInterestFirst(t *testing.T) {
	pp, ip, np, ni := AllocateRepayment(100, 10, 5)
	if ip != 5 || pp != 0 || ni != 5 || np != 100 {
		t.Fatalf("%v %v %v %v", pp, ip, np, ni)
	}
	pp, ip, np, ni = AllocateRepayment(100, 10, 30)
	if math.Abs(ip-10) > 1e-12 || math.Abs(pp-20) > 1e-12 || math.Abs(np-80) > 1e-12 || math.Abs(ni) > 1e-12 {
		t.Fatalf("%v %v %v %v", pp, ip, np, ni)
	}
}

func TestSystemCloseTradeID(t *testing.T) {
	id := "pos-abc"
	if got := SystemCloseTradeID(MarginCloseLiquidation, id); got != "margin-liq:pos-abc" {
		t.Fatalf("liq id=%q", got)
	}
	if got := SystemCloseTradeID(MarginCloseStopLoss, id); got != "margin-sl:pos-abc" {
		t.Fatalf("sl id=%q", got)
	}
	if got := SystemCloseTradeID("close", id); got != "" {
		t.Fatalf("user close should not get system id, got %q", got)
	}
	if !IsSystemForcedCloseAction(MarginCloseLiquidation) || IsSystemForcedCloseAction("repay") {
		t.Fatal("IsSystemForcedCloseAction")
	}
}

func TestAccrueInterestHours(t *testing.T) {
	last := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	now := last.Add(2*time.Hour + 30*time.Minute)
	ni, nl, h := AccrueInterestHours(1000, 0, last, now, 0.01)
	if h != 2 || math.Abs(ni-20) > 1e-9 {
		t.Fatalf("ni=%v h=%d nl=%v", ni, h, nl)
	}
	if !nl.Equal(last.Add(2 * time.Hour)) {
		t.Fatalf("last=%v", nl)
	}
	// Long offline catch-up is O(1) product, not a loop of 100 hours.
	ni2, nl2, h2 := AccrueInterestHours(100, 0, last, last.Add(100*time.Hour), 0.01)
	if h2 != 100 || math.Abs(ni2-100) > 1e-9 {
		t.Fatalf("catch-up ni=%v h=%d", ni2, h2)
	}
	if !nl2.Equal(last.Add(100 * time.Hour)) {
		t.Fatalf("nl2=%v", nl2)
	}
	// Clock backward: no change
	ni3, nl3, h3 := AccrueInterestHours(1000, 5, last, last.Add(-time.Hour), 0.01)
	if h3 != 0 || ni3 != 5 || !nl3.Equal(last) {
		t.Fatalf("backward h=%d ni=%v nl=%v", h3, ni3, nl3)
	}
	// Same last: no double period
	ni4, _, h4 := AccrueInterestHours(1000, 5, last, last, 0.01)
	if h4 != 0 || ni4 != 5 {
		t.Fatalf("same instant h=%d ni=%v", h4, ni4)
	}
	// Fully paid principal: stop
	_, _, h5 := AccrueInterestHours(0, 1, last, now, 0.01)
	if h5 != 0 {
		t.Fatalf("zero principal hours=%d", h5)
	}
}

func TestLiquidationPriceWithDebtInterest(t *testing.T) {
	// long 1@100, margin 20, no interest → classic ~80.5
	liq0, err := LiquidationPriceWithDebt(MarginLong, 100, 1, 20, 80, 0, 0.005)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(liq0-80.5) > 1e-9 {
		t.Fatalf("liq0=%v", liq0)
	}
	// with interest 5: liq higher (worse for long)
	liq1, err := LiquidationPriceWithDebt(MarginLong, 100, 1, 20, 80, 5, 0.005)
	if err != nil || liq1 <= liq0 {
		t.Fatalf("liq0=%v liq1=%v", liq0, liq1)
	}
	// add margin without interest: liq lower
	liq2, err := LiquidationPriceWithDebt(MarginLong, 100, 1, 40, 80, 0, 0.005)
	if err != nil || liq2 >= liq0 {
		t.Fatalf("liq0=%v liq2=%v", liq0, liq2)
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
