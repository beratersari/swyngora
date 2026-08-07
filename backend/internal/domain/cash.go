package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Paper cash movement limits (simulated only).
const (
	MinCashMovement     = 0.01
	MaxCashMovement     = MaxStartingBalance
	MaxCashBalance      = MaxStartingBalance * 10
	MaxCashMovementNote = 200
)

// CashMovementKind is a user cash transfer in or out of the paper book.
type CashMovementKind string

const (
	CashMovementDeposit    CashMovementKind = "deposit"
	CashMovementWithdrawal CashMovementKind = "withdrawal"
)

// IsValidCashMovementKind reports deposit|withdrawal.
func IsValidCashMovementKind(s string) bool {
	switch CashMovementKind(strings.ToLower(strings.TrimSpace(s))) {
	case CashMovementDeposit, CashMovementWithdrawal:
		return true
	default:
		return false
	}
}

// CashMovement is one deposit or withdrawal ledger row.
type CashMovement struct {
	ID               string
	ClientID         string
	Kind             CashMovementKind
	Amount           float64 // always positive
	CashAfter        float64
	NetDepositsAfter float64
	Note             string
	CreatedAt        time.Time
}

// ContributedCapital is opening cash plus later net deposits (external money in).
func ContributedCapital(startingBalance, netDeposits float64) float64 {
	return startingBalance + netDeposits
}

// PortfolioTotalPnL is equity minus money the user put in (not trading P&L from deposits).
func PortfolioTotalPnL(equity, startingBalance, netDeposits float64) float64 {
	return equity - ContributedCapital(startingBalance, netDeposits)
}

// ValidateCashMovementAmount checks a deposit/withdraw size.
func ValidateCashMovementAmount(amount float64) error {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < MinCashMovement || amount > MaxCashMovement {
		return fmt.Errorf("%w: amount must be between %g and %g", ErrInvalidArgument, MinCashMovement, MaxCashMovement)
	}
	return nil
}
