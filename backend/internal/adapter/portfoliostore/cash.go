package portfoliostore

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ApplyCashMovement updates cash + net_deposits and inserts a ledger row atomically.
func (s *SQLite) ApplyCashMovement(ctx context.Context, p *domain.Portfolio, m domain.CashMovement) (*domain.CashMovement, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: portfolio required", domain.ErrInvalidArgument)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = m.CreatedAt
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	bookID := p.BookID()
	res, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, net_deposits = ?, updated_at = ?
		WHERE id = ?
	`, p.CashBalance, p.NetDeposits, p.UpdatedAt.UTC().Format(time.RFC3339Nano), bookID)
	if err != nil {
		return nil, fmt.Errorf("cash movement update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO cash_movements (id, client_id, kind, amount, cash_after, net_deposits_after, note,
			counterparty_portfolio_id, counterparty_portfolio_name, peer_movement_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, bookID, string(m.Kind), m.Amount, m.CashAfter, m.NetDepositsAfter, m.Note,
		m.CounterpartyPortfolioID, m.CounterpartyPortfolioName, m.PeerMovementID,
		m.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("cash movement insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	cp := m
	cp.ClientID = bookID
	return &cp, nil
}

// ApplyInternalTransfer updates both books and writes paired ledger rows in one transaction.
func (s *SQLite) ApplyInternalTransfer(ctx context.Context, from, to *domain.Portfolio, out, in domain.CashMovement) (*domain.CashMovement, *domain.CashMovement, error) {
	if from == nil || to == nil {
		return nil, nil, fmt.Errorf("%w: both portfolios required", domain.ErrInvalidArgument)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if out.CreatedAt.IsZero() {
		out.CreatedAt = time.Now().UTC()
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = out.CreatedAt
	}
	if from.UpdatedAt.IsZero() {
		from.UpdatedAt = out.CreatedAt
	}
	if to.UpdatedAt.IsZero() {
		to.UpdatedAt = in.CreatedAt
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()
	fromID, toID := from.BookID(), to.BookID()
	res, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, net_deposits = ?, updated_at = ? WHERE id = ?
	`, from.CashBalance, from.NetDeposits, from.UpdatedAt.UTC().Format(time.RFC3339Nano), fromID)
	if err != nil {
		return nil, nil, fmt.Errorf("transfer from update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil, domain.ErrNotFound
	}
	res, err = tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, net_deposits = ?, updated_at = ? WHERE id = ?
	`, to.CashBalance, to.NetDeposits, to.UpdatedAt.UTC().Format(time.RFC3339Nano), toID)
	if err != nil {
		return nil, nil, fmt.Errorf("transfer to update: %w", err)
	}
	n, _ = res.RowsAffected()
	if n == 0 {
		return nil, nil, domain.ErrNotFound
	}
	at := out.CreatedAt.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO cash_movements (id, client_id, kind, amount, cash_after, net_deposits_after, note,
			counterparty_portfolio_id, counterparty_portfolio_name, peer_movement_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, out.ID, fromID, string(out.Kind), out.Amount, out.CashAfter, out.NetDepositsAfter, out.Note,
		out.CounterpartyPortfolioID, out.CounterpartyPortfolioName, out.PeerMovementID, at); err != nil {
		return nil, nil, fmt.Errorf("transfer out insert: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO cash_movements (id, client_id, kind, amount, cash_after, net_deposits_after, note,
			counterparty_portfolio_id, counterparty_portfolio_name, peer_movement_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, in.ID, toID, string(in.Kind), in.Amount, in.CashAfter, in.NetDepositsAfter, in.Note,
		in.CounterpartyPortfolioID, in.CounterpartyPortfolioName, in.PeerMovementID, at); err != nil {
		return nil, nil, fmt.Errorf("transfer in insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	out.ClientID = fromID
	in.ClientID = toID
	return &out, &in, nil
}

// ListCashMovements returns newest-first ledger rows.
func (s *SQLite) ListCashMovements(ctx context.Context, clientID string, limit, offset int) ([]domain.CashMovement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, kind, amount, cash_after, net_deposits_after, note,
			COALESCE(counterparty_portfolio_id, ''), COALESCE(counterparty_portfolio_name, ''),
			COALESCE(peer_movement_id, ''), created_at
		FROM cash_movements
		WHERE client_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, clientID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list cash movements: %w", err)
	}
	defer rows.Close()
	var out []domain.CashMovement
	for rows.Next() {
		var m domain.CashMovement
		var kind, cAt string
		if err := rows.Scan(&m.ID, &m.ClientID, &kind, &m.Amount, &m.CashAfter, &m.NetDepositsAfter, &m.Note,
			&m.CounterpartyPortfolioID, &m.CounterpartyPortfolioName, &m.PeerMovementID, &cAt); err != nil {
			return nil, err
		}
		m.Kind = domain.CashMovementKind(kind)
		m.CreatedAt = parseTime(cAt)
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountCashMovements counts ledger rows for a client.
func (s *SQLite) CountCashMovements(ctx context.Context, clientID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cash_movements WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}
