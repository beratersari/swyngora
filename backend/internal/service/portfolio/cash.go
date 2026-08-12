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
	unlock := s.lockClient(clientID)
	defer unlock()
	// Re-load under lock so concurrent trades cannot last-write-wins cash.
	p, err = s.requireBook(ctx, in.ClientID, p.ID)
	if err != nil {
		return nil, nil, err
	}
	clientID = p.BookID()
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
	s.notifyChange(ctx, p.ID, domain.PortfolioChangeCash, nil, nil, view)
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

// TransferInput moves virtual cash between two books the same owner controls.
type TransferInput struct {
	ClientID        string
	FromPortfolioID string
	ToPortfolioID   string
	Amount          float64
	Note            string
}

// Transfer moves available cash from one owned book to another. Not a deposit/withdrawal.
// Contributed capital moves with the cash so neither book's trading P&L changes.
func (s *Service) Transfer(ctx context.Context, in TransferInput) (fromMov, toMov *domain.CashMovement, fromView, toView *domain.PortfolioView, err error) {
	if s.store == nil {
		return nil, nil, nil, nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	from, err := s.requireBook(ctx, in.ClientID, in.FromPortfolioID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	to, err := s.requireBook(ctx, in.ClientID, in.ToPortfolioID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if from.BookID() == to.BookID() {
		return nil, nil, nil, nil, fmt.Errorf("%w: from and to portfolios must be different", domain.ErrInvalidArgument)
	}
	if from.ClientID != to.ClientID {
		return nil, nil, nil, nil, fmt.Errorf("%w: can only transfer between your own portfolios", domain.ErrForbidden)
	}
	// Ordered locks avoid deadlock when two transfers reverse the book pair.
	idA, idB := from.BookID(), to.BookID()
	if idA > idB {
		idA, idB = idB, idA
	}
	unlockA := s.lockClient(idA)
	defer unlockA()
	unlockB := s.lockClient(idB)
	defer unlockB()
	from, err = s.requireBook(ctx, in.ClientID, from.ID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	to, err = s.requireBook(ctx, in.ClientID, to.ID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := domain.ValidateCashMovementAmount(in.Amount); err != nil {
		return nil, nil, nil, nil, err
	}
	note, err := normalizeCashNote(in.Note)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	reservedCash, err := s.store.SumReservedCash(ctx, from.BookID())
	if err != nil {
		return nil, nil, nil, nil, err
	}
	reservedMargin, err := s.store.SumReservedMargin(ctx, from.BookID())
	if err != nil {
		return nil, nil, nil, nil, err
	}
	avail := domain.AvailableCash(from.CashBalance, reservedCash+reservedMargin)
	if in.Amount > avail+1e-9 {
		return nil, nil, nil, nil, fmt.Errorf("%w: insufficient available cash (have %g)", domain.ErrInvalidArgument, avail)
	}
	if to.CashBalance+in.Amount > domain.MaxCashBalance {
		return nil, nil, nil, nil, fmt.Errorf("%w: destination cash balance would exceed %g", domain.ErrInvalidArgument, domain.MaxCashBalance)
	}
	now := time.Now().UTC()
	from.CashBalance -= in.Amount
	from.NetDeposits -= in.Amount
	from.UpdatedAt = now
	to.CashBalance += in.Amount
	to.NetDeposits += in.Amount
	to.UpdatedAt = now
	outID, inID := uuid.NewString(), uuid.NewString()
	out := domain.CashMovement{
		ID: outID, Kind: domain.CashMovementTransferOut, Amount: in.Amount,
		CashAfter: from.CashBalance, NetDepositsAfter: from.NetDeposits, Note: note,
		CounterpartyPortfolioID: to.BookID(), CounterpartyPortfolioName: to.Name,
		PeerMovementID: inID, CreatedAt: now,
	}
	inRow := domain.CashMovement{
		ID: inID, Kind: domain.CashMovementTransferIn, Amount: in.Amount,
		CashAfter: to.CashBalance, NetDepositsAfter: to.NetDeposits, Note: note,
		CounterpartyPortfolioID: from.BookID(), CounterpartyPortfolioName: from.Name,
		PeerMovementID: outID, CreatedAt: now,
	}
	fromMov, toMov, err = s.store.ApplyInternalTransfer(ctx, from, to, out, inRow)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	fromView, err = s.View(ctx, from.ClientID, from.ID)
	if err != nil {
		return fromMov, toMov, nil, nil, err
	}
	toView, err = s.View(ctx, to.ClientID, to.ID)
	if err != nil {
		return fromMov, toMov, fromView, nil, err
	}
	_ = s.recordViewSnapshot(ctx, fromView, now)
	_ = s.recordViewSnapshot(ctx, toView, now)
	s.notifyChange(ctx, from.ID, domain.PortfolioChangeCash, nil, nil, fromView)
	s.notifyChange(ctx, to.ID, domain.PortfolioChangeCash, nil, nil, toView)
	return fromMov, toMov, fromView, toView, nil
}
