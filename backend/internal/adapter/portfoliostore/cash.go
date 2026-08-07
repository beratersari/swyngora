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
	res, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, net_deposits = ?, updated_at = ?
		WHERE client_id = ?
	`, p.CashBalance, p.NetDeposits, p.UpdatedAt.UTC().Format(time.RFC3339Nano), p.ClientID)
	if err != nil {
		return nil, fmt.Errorf("cash movement update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO cash_movements (id, client_id, kind, amount, cash_after, net_deposits_after, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, p.ClientID, string(m.Kind), m.Amount, m.CashAfter, m.NetDepositsAfter, m.Note,
		m.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("cash movement insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	cp := m
	cp.ClientID = p.ClientID
	return &cp, nil
}

// ListCashMovements returns newest-first ledger rows.
func (s *SQLite) ListCashMovements(ctx context.Context, clientID string, limit, offset int) ([]domain.CashMovement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, kind, amount, cash_after, net_deposits_after, note, created_at
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
		if err := rows.Scan(&m.ID, &m.ClientID, &kind, &m.Amount, &m.CashAfter, &m.NetDepositsAfter, &m.Note, &cAt); err != nil {
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
