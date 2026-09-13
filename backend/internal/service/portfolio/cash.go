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
		if err := s.guardCrossWithdrawMaint(ctx, p, in.Amount); err != nil {
			return nil, nil, err
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

// guardCrossWithdrawMaint rejects a withdraw that would leave cross equity under total maintenance.
func (s *Service) guardCrossWithdrawMaint(ctx context.Context, p *domain.Portfolio, amount float64) error {
	if p == nil || p.MarginMode != domain.MarginModeCross {
		return nil
	}
	risk, err := s.loadCrossAccountRisk(ctx, p.BookID())
	if err != nil {
		return err
	}
	if risk == nil || risk.totalMaint <= 0 {
		return nil
	}
	after := risk.equity - amount
	if after+1e-9 < risk.totalMaint {
		return fmt.Errorf("%w: withdrawal would leave equity %g below maintenance %g", domain.ErrInvalidArgument, after, risk.totalMaint)
	}
	return nil
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
	IdempotencyKey  string
}

// TransferResult is one transfer or a replay of the same client key.
type TransferResult struct {
	Out            *domain.CashMovement
	In             *domain.CashMovement
	From           *domain.PortfolioView
	To             *domain.PortfolioView
	Replayed       bool
	Attempt        int
	IdempotencyKey string
}

// Transfer moves available cash from one owned book to another. Not a deposit/withdrawal.
// Contributed capital moves with the cash so neither book's trading P&L changes.
// The same Idempotency-Key + same request returns the original pair instead of a second move.
func (s *Service) Transfer(ctx context.Context, in TransferInput) (*TransferResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	from, err := s.requireBook(ctx, in.ClientID, in.FromPortfolioID)
	if err != nil {
		return nil, err
	}
	to, err := s.requireBook(ctx, in.ClientID, in.ToPortfolioID)
	if err != nil {
		return nil, err
	}
	if from.BookID() == to.BookID() {
		return nil, fmt.Errorf("%w: from and to portfolios must be different", domain.ErrInvalidArgument)
	}
	if from.ClientID != to.ClientID {
		return nil, fmt.Errorf("%w: can only transfer between your own portfolios", domain.ErrForbidden)
	}
	if domain.AliasFxCode(from.Currency) != domain.AliasFxCode(to.Currency) {
		return nil, fmt.Errorf("%w: cannot transfer %s to a %s book", domain.ErrInvalidArgument, from.Currency, to.Currency)
	}
	idempKey, err := domain.NormalizeIdempotencyKey(in.IdempotencyKey)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	to, err = s.requireBook(ctx, in.ClientID, to.ID)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateCashMovementAmount(in.Amount); err != nil {
		return nil, err
	}
	note, err := normalizeCashNote(in.Note)
	if err != nil {
		return nil, err
	}
	idempHash := hashParts("transfer", from.BookID(), to.BookID(), in.Amount, note)
	if rec, err := s.checkIdempotency(ctx, from.BookID(), idempKey, idempHash); err != nil {
		if idempKey != "" {
			s.recordTransferAttempt(ctx, in, from, to, note, idempKey, idempHash, domain.TransferAttemptConflict, "", "")
		}
		return nil, err
	} else if rec != nil {
		return s.replayTransfer(ctx, rec, from, to, in, note, idempKey, idempHash)
	}
	reservedCash, err := s.store.SumReservedCash(ctx, from.BookID())
	if err != nil {
		return nil, err
	}
	reservedMargin, err := s.store.SumReservedMargin(ctx, from.BookID())
	if err != nil {
		return nil, err
	}
	avail := domain.AvailableCash(from.CashBalance, reservedCash+reservedMargin)
	if in.Amount > avail+1e-9 {
		return nil, fmt.Errorf("%w: insufficient available cash (have %g)", domain.ErrInvalidArgument, avail)
	}
	if err := s.guardCrossWithdrawMaint(ctx, from, in.Amount); err != nil {
		return nil, err
	}
	if to.CashBalance+in.Amount > domain.MaxCashBalance {
		return nil, fmt.Errorf("%w: destination cash balance would exceed %g", domain.ErrInvalidArgument, domain.MaxCashBalance)
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
	if idempKey != "" {
		ctx = s.withIdempotency(ctx, from.BookID(), idempKey, idempHash, domain.IdempotencyKindTransfer, idempIDs{
			OutID: outID, InID: inID,
		})
		ctx = domain.ContextWithTransferAttempt(ctx, &domain.TransferAttempt{
			ID: uuid.NewString(), OwnerClientID: from.ClientID, IdempotencyKey: idempKey, RequestHash: idempHash,
			FromPortfolioID: from.BookID(), ToPortfolioID: to.BookID(),
			FromPortfolioName: from.Name, ToPortfolioName: to.Name, Amount: in.Amount, Note: note,
			Status: domain.TransferAttemptCompleted, Attempt: 1, OutMovementID: outID, InMovementID: inID, CreatedAt: now,
		})
	}
	fromMov, toMov, err := s.store.ApplyInternalTransfer(ctx, from, to, out, inRow)
	if err != nil {
		return nil, err
	}
	fromView, err := s.View(ctx, from.ClientID, from.ID)
	if err != nil {
		return &TransferResult{Out: fromMov, In: toMov, IdempotencyKey: idempKey, Attempt: 1}, err
	}
	toView, err := s.View(ctx, to.ClientID, to.ID)
	if err != nil {
		return &TransferResult{Out: fromMov, In: toMov, From: fromView, IdempotencyKey: idempKey, Attempt: 1}, err
	}
	_ = s.recordViewSnapshot(ctx, fromView, now)
	_ = s.recordViewSnapshot(ctx, toView, now)
	s.notifyChange(ctx, from.ID, domain.PortfolioChangeCash, nil, nil, fromView)
	s.notifyChange(ctx, to.ID, domain.PortfolioChangeCash, nil, nil, toView)
	return &TransferResult{
		Out: fromMov, In: toMov, From: fromView, To: toView,
		Attempt: 1, IdempotencyKey: idempKey,
	}, nil
}

func (s *Service) replayTransfer(ctx context.Context, rec *domain.IdempotencyRecord, from, to *domain.Portfolio, in TransferInput, note, key, hash string) (*TransferResult, error) {
	ids := parseIdempIDs(rec.ResultJSON)
	outMov, err := s.store.GetCashMovement(ctx, from.BookID(), ids.OutID)
	if err != nil {
		return nil, err
	}
	inMov, err := s.store.GetCashMovement(ctx, to.BookID(), ids.InID)
	if err != nil {
		return nil, err
	}
	n := s.recordTransferAttempt(ctx, in, from, to, note, key, hash, domain.TransferAttemptReplayed, ids.OutID, ids.InID)
	fromView, err := s.View(ctx, from.ClientID, from.ID)
	if err != nil {
		return &TransferResult{Out: outMov, In: inMov, Replayed: true, Attempt: n, IdempotencyKey: key}, err
	}
	toView, err := s.View(ctx, to.ClientID, to.ID)
	if err != nil {
		return &TransferResult{Out: outMov, In: inMov, From: fromView, Replayed: true, Attempt: n, IdempotencyKey: key}, err
	}
	return &TransferResult{
		Out: outMov, In: inMov, From: fromView, To: toView,
		Replayed: true, Attempt: n, IdempotencyKey: key,
	}, nil
}

func (s *Service) recordTransferAttempt(ctx context.Context, in TransferInput, from, to *domain.Portfolio, note, key, hash, status, outID, inID string) int {
	n, err := s.store.NextTransferAttempt(ctx, from.ClientID, key)
	if err != nil || n <= 0 {
		n = 2
	}
	_, _ = s.store.InsertTransferAttempt(ctx, domain.TransferAttempt{
		ID: uuid.NewString(), OwnerClientID: from.ClientID, IdempotencyKey: key, RequestHash: hash,
		FromPortfolioID: from.BookID(), ToPortfolioID: to.BookID(),
		FromPortfolioName: from.Name, ToPortfolioName: to.Name, Amount: in.Amount, Note: note,
		Status: status, Attempt: n, OutMovementID: outID, InMovementID: inID, CreatedAt: time.Now().UTC(),
	})
	return n
}

// ListTransferAttempts returns newest-first retry history for the owner.
func (s *Service) ListTransferAttempts(ctx context.Context, clientID string, limit, offset int, portfolioID ...string) ([]domain.TransferAttempt, int, error) {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	owner := p.ClientID
	total, err := s.store.CountTransferAttempts(ctx, owner)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.store.ListTransferAttempts(ctx, owner, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
