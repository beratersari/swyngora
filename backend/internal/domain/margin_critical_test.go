package domain

import (
	"math"
	"testing"
)

// When accrued interest exceeds (margin − maint), long liq must rise above entry
// so ShouldLiquidate can fire in that band (false-safe if clamped to entry).
func TestLiquidationPriceWithDebt_InterestExceedsBuffer(t *testing.T) {
	const entry, qty, margin, principal, interest, mmr = 100.0, 1.0, 20.0, 80.0, 20.0, 0.005
	want := entry - (margin-MaintenanceMargin(qty, entry, mmr)-interest)/qty
	if want <= entry {
		t.Fatalf("setup: expected true liq above entry, want=%v", want)
	}
	got, err := LiquidationPriceWithDebt(MarginLong, entry, qty, margin, principal, interest, mmr)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("liq got=%v want=%v", got, want)
	}
	mark := (entry + want) / 2
	if !ShouldLiquidate(MarginLong, mark, got) {
		t.Fatalf("mark=%v should liquidate vs liq=%v", mark, got)
	}
}

func TestApplyMarginCloseCash_IsolatedDoesNotTakeUnassigned(t *testing.T) {
	// Free cash 200, assigned IM already deducted. A −2800 isolated loss
	// must not touch the 200 or go negative.
	got, applied := ApplyMarginCloseCash(MarginModeIsolated, 200, -2800)
	if got != 200 || applied != 0 {
		t.Fatalf("isolated gap: cash=%v applied=%v want 200 / 0", got, applied)
	}
	got, applied = ApplyMarginCloseCash(MarginModeIsolated, 200, 1200)
	if math.Abs(got-1400) > 1e-9 || math.Abs(applied-1200) > 1e-9 {
		t.Fatalf("isolated win: cash=%v applied=%v", got, applied)
	}
}

func TestApplyMarginCloseCash_CrossBankruptcyFloor(t *testing.T) {
	got, applied := ApplyMarginCloseCash(MarginModeCross, 200, -2800)
	if got != 0 || math.Abs(applied+200) > 1e-9 {
		t.Fatalf("cross floor: cash=%v applied=%v want 0 / -200", got, applied)
	}
	got, applied = ApplyMarginCloseCash(MarginModeCross, 900, -100)
	if math.Abs(got-800) > 1e-9 || math.Abs(applied+100) > 1e-9 {
		t.Fatalf("cross loss: cash=%v applied=%v", got, applied)
	}
}
