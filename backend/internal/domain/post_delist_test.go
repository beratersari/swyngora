package domain

import (
	"math"
	"testing"
)

func TestFillChangeAbsFromPct(t *testing.T) {
	pct := -12.5
	q := &OffVenueQuote{LastUSD: 0.03, ChangePct: &pct}
	q.FillChangeAbs()
	if q.ChangeAbs == nil {
		t.Fatal("expected delta")
	}
	want := 0.03 * (-0.125) / (1 - 0.125)
	if math.Abs(*q.ChangeAbs-want) > 1e-12 {
		t.Fatalf("got %v want %v", *q.ChangeAbs, want)
	}
}
