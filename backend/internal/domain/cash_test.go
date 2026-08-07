package domain

import "testing"

func TestPortfolioTotalPnL_IgnoresDeposits(t *testing.T) {
	// Start 10k, deposit 5k, no trading → equity 15k, P&L 0.
	if got := PortfolioTotalPnL(15000, 10000, 5000); got != 0 {
		t.Fatalf("got %v", got)
	}
	// Then +200 trading → equity 15200, P&L 200.
	if got := PortfolioTotalPnL(15200, 10000, 5000); got != 200 {
		t.Fatalf("got %v", got)
	}
	// Withdraw 1k after that → equity 14200, netDeposits 4000, P&L still 200.
	if got := PortfolioTotalPnL(14200, 10000, 4000); got != 200 {
		t.Fatalf("got %v", got)
	}
}

func TestValidateCashMovementAmount(t *testing.T) {
	if err := ValidateCashMovementAmount(100); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCashMovementAmount(0); err == nil {
		t.Fatal("want error")
	}
}
