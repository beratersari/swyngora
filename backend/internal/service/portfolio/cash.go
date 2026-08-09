package portfolio

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// CashMoveInput is a user deposit or withdrawal.
type CashMoveInput struct {
	ClientID    string
	PortfolioID string
	Amount      float64
	Note        string
}

func normalizeCashNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > domain.MaxCashMovementNote {
		return "", fmt.Errorf("%w: note must be at most %d characters", domain.ErrInvalidArgument, domain.MaxCashMovementNote)
	}
	return note, nil
}

func (s *Service) recordOpeningCashMovement(ctx context.Context, p *domain.Portfolio) {
	if p == nil || s.store == nil {
		return
	}
	at := p.CreatedAt.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, _ = s.store.ApplyCashMovement(ctx, p, domain.CashMovement{
		ID: uuid.NewString(), Kind: domain.CashMovementDeposit,
		Amount: p.StartingBalance, CashAfter: p.CashBalance, NetDepositsAfter: 0,
		Note: "Opening balance", CreatedAt: at,
	})
}

// Deposit adds virtual cash. Does not count as trading P&L.
func (s *Service) Deposit(ctx context.Context, in CashMoveInput) (*domain.CashMovement, *domain.PortfolioView, error) {
	return s.moveCash(ctx, in, domain.CashMovementDeposit)
}

// Withdraw removes available cash. Does not count as trading P&L.
func (s *Service) Withdraw(ctx context.Context, in CashMoveInput) (*domain.CashMovement, *domain.PortfolioView, error) {
	return s.moveCash(ctx, in, domain.CashMovementWithdrawal)
}

func (s *Service) moveCash(ctx context.Context, in CashMoveInput, kind domain.CashMovementKind) (*domain.CashMovement, *domain.PortfolioView, error) {
	if s.store == nil {
		return nil, nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireBook(ctx, in.ClientID, in.PortfolioID)
	if err != nil {
		return nil, nil, err
	}
	clientID := p.BookID()
	if err := domain.ValidateCashMovementAmount(in.Amount); err != nil {
		return nil, nil, err
	}
	note, err := normalizeCashNote(in.Note)
	if err != nil {
		return nil, nil, err
	}
	reservedCash, err := s.store.SumReservedCash(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}
	reservedMargin, err := s.store.SumReservedMargin(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}
	avail := domain.AvailableCash(p.CashBalance, reservedCash+reservedMargin)
	now := time.Now().UTC()
	newCash := p.CashBalance
	newNet := p.NetDeposits
	switch kind {
	case domain.CashMovementDeposit:
		newCash = p.CashBalance + in.Amount
		newNet = p.NetDeposits + in.Amount
		if newCash > domain.MaxCashBalance {
			return nil, nil, fmt.Errorf("%w: cash balance would exceed %g", domain.ErrInvalidArgument, domain.MaxCashBalance)
		}
	case domain.CashMovementWithdrawal:
		if in.Amount > avail+1e-9 {
			return nil, nil, fmt.Errorf("%w: insufficient available cash (have %g)", domain.ErrInvalidArgument, avail)
		}
		newCash = p.CashBalance - in.Amount
		newNet = p.NetDeposits - in.Amount
		if newCash < 0 {
			newCash = 0
		}
	default:
		return nil, nil, fmt.Errorf("%w: kind must be deposit or withdrawal", domain.ErrInvalidArgument)
	}
	p.CashBalance = newCash
	p.NetDeposits = newNet
	p.UpdatedAt = now
	m, err := s.store.ApplyCashMovement(ctx, p, domain.CashMovement{
		ID: uuid.NewString(), Kind: kind, Amount: in.Amount,
		CashAfter: newCash, NetDepositsAfter: newNet, Note: note, CreatedAt: now,
	})
	if err != nil {
		return nil, nil, err
	}
	view, err := s.View(ctx, p.ClientID, p.ID)
	if err != nil {
		return m, nil, err
	}
	_ = s.recordViewSnapshot(ctx, view, now)
	return m, view, nil
}

// ListCashMovements returns deposit/withdraw history (newest first).
func (s *Service) ListCashMovements(ctx context.Context, clientID string, limit, offset int, portfolioID ...string) ([]domain.CashMovement, int, error) {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return nil, 0, err
	}
	clientID = p.BookID()
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	total, err := s.store.CountCashMovements(ctx, clientID)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.store.ListCashMovements(ctx, clientID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
