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

func TestTradingPnLChange_IgnoresCashMoves(t *testing.T) {
	// Deposit 5000 on a flat book: equity 10000 → 15000, net 0 → 5000.
	if got := TradingPnLChange(10000, 15000, 0, 5000); got != 0 {
		t.Fatalf("deposit change %v", got)
	}
	// Withdraw 1000 of profit that was already there: equity 11000 → 10000, net 0 → -1000.
	if got := TradingPnLChange(11000, 10000, 0, -1000); got != 0 {
		t.Fatalf("withdraw change %v", got)
	}
	// Earn 200, then deposit 5000: equity 10000 → 15200, net 0 → 5000.
	if got := TradingPnLChange(10000, 15200, 0, 5000); got != 200 {
		t.Fatalf("earn then deposit %v", got)
	}
	if got := EquityExcludingFlows(15200, 0, 5000, true); got != 10200 {
		t.Fatalf("mark %v", got)
	}
	if got := EquityExcludingFlows(15200, 0, 5000, false); got != 15200 {
		t.Fatalf("unknown baseline should keep equity %v", got)
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
