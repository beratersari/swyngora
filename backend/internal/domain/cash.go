package domain

import (
	"context"
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
	CashMovementDeposit     CashMovementKind = "deposit"
	CashMovementWithdrawal  CashMovementKind = "withdrawal"
	CashMovementTransferOut CashMovementKind = "transfer_out"
	CashMovementTransferIn  CashMovementKind = "transfer_in"
)

// IsValidCashMovementKind reports known ledger kinds.
func IsValidCashMovementKind(s string) bool {
	switch CashMovementKind(strings.ToLower(strings.TrimSpace(s))) {
	case CashMovementDeposit, CashMovementWithdrawal, CashMovementTransferOut, CashMovementTransferIn:
		return true
	default:
		return false
	}
}

// IsInternalTransfer reports a move between the owner's own paper books.
func (k CashMovementKind) IsInternalTransfer() bool {
	return k == CashMovementTransferOut || k == CashMovementTransferIn
}

// CashMovement is one deposit, withdrawal, or internal transfer ledger row.
type CashMovement struct {
	ID               string
	ClientID         string
	Kind             CashMovementKind
	Amount           float64 // always positive
	CashAfter        float64
	NetDepositsAfter float64
	Note             string
	// CounterpartyPortfolioID is the other book on an internal transfer.
	CounterpartyPortfolioID string
	// CounterpartyPortfolioName is the other book's name at transfer time.
	CounterpartyPortfolioName string
	// PeerMovementID is the matching ledger row on the other book.
	PeerMovementID string
	CreatedAt      time.Time
}

// Transfer attempt outcomes for a client retry key.
const (
	TransferAttemptCompleted = "completed"
	TransferAttemptReplayed  = "replayed"
	TransferAttemptConflict  = "conflict"
)

// TransferAttempt is one POST /transfers try (first fill or a later retry).
type TransferAttempt struct {
	ID                string
	OwnerClientID     string
	IdempotencyKey    string
	RequestHash       string
	FromPortfolioID   string
	ToPortfolioID     string
	FromPortfolioName string
	ToPortfolioName   string
	Amount            float64
	Note              string
	Status            string
	Attempt           int
	OutMovementID     string
	InMovementID      string
	CreatedAt         time.Time
}

type transferAttemptCtxKey struct{}

// ContextWithTransferAttempt attaches a completed-attempt row for the transfer write tx.
func ContextWithTransferAttempt(ctx context.Context, a *TransferAttempt) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if a == nil || a.IdempotencyKey == "" {
		return ctx
	}
	return context.WithValue(ctx, transferAttemptCtxKey{}, a)
}

// TransferAttemptFromContext returns the attempt attached for the current write, if any.
func TransferAttemptFromContext(ctx context.Context) *TransferAttempt {
	if ctx == nil {
		return nil
	}
	a, _ := ctx.Value(transferAttemptCtxKey{}).(*TransferAttempt)
	return a
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
