package domain

import (
	"math"
	"testing"
)

// When accrued interest exceeds (margin − maint), long liq must rise above entry
// so ShouldLiquidate can fire in that band (false-safe if clamped to entry).
func TestLiquidationPriceWithDebt_InterestExceedsBuffer(t *testing.T) {
	const entry, qty, margin, principal, interest, mmr = 100.0, 1.0, 20.0, 80.0, 20.0, 0.005
	want := entry - (margin - MaintenanceMargin(qty, entry, mmr) - interest) / qty
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
