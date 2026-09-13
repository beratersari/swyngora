package portfoliostore

import (
	"context"
	"database/sql"
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
	if err := s.txInsertIdempotency(ctx, tx); err != nil {
		return nil, nil, err
	}
	if err := txInsertTransferAttempt(ctx, tx); err != nil {
		return nil, nil, err
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

func (s *SQLite) migrateTransferAttempts() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS transfer_attempts (
	id                   TEXT PRIMARY KEY NOT NULL,
	owner_client_id      TEXT NOT NULL,
	idempotency_key      TEXT NOT NULL,
	request_hash         TEXT NOT NULL,
	from_portfolio_id    TEXT NOT NULL,
	to_portfolio_id      TEXT NOT NULL,
	from_portfolio_name  TEXT NOT NULL DEFAULT '',
	to_portfolio_name    TEXT NOT NULL DEFAULT '',
	amount               REAL NOT NULL,
	note                 TEXT NOT NULL DEFAULT '',
	status               TEXT NOT NULL,
	attempt              INTEGER NOT NULL,
	out_movement_id      TEXT NOT NULL DEFAULT '',
	in_movement_id       TEXT NOT NULL DEFAULT '',
	created_at           TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_transfer_attempts_owner ON transfer_attempts(owner_client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfer_attempts_key ON transfer_attempts(owner_client_id, idempotency_key, attempt);
`)
	return err
}

func txInsertTransferAttempt(ctx context.Context, tx *sql.Tx) error {
	a := domain.TransferAttemptFromContext(ctx)
	if a == nil || a.IdempotencyKey == "" {
		return nil
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if a.Attempt <= 0 {
		a.Attempt = 1
	}
	if a.ID == "" {
		return fmt.Errorf("transfer attempt id required")
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO transfer_attempts (
			id, owner_client_id, idempotency_key, request_hash, from_portfolio_id, to_portfolio_id,
			from_portfolio_name, to_portfolio_name, amount, note, status, attempt,
			out_movement_id, in_movement_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.OwnerClientID, a.IdempotencyKey, a.RequestHash, a.FromPortfolioID, a.ToPortfolioID,
		a.FromPortfolioName, a.ToPortfolioName, a.Amount, a.Note, a.Status, a.Attempt,
		a.OutMovementID, a.InMovementID, fmtTime(a.CreatedAt))
	return err
}

// GetCashMovement returns one ledger row for a book.
func (s *SQLite) GetCashMovement(ctx context.Context, bookID, id string) (*domain.CashMovement, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client_id, kind, amount, cash_after, net_deposits_after, note,
			COALESCE(counterparty_portfolio_id, ''), COALESCE(counterparty_portfolio_name, ''),
			COALESCE(peer_movement_id, ''), created_at
		FROM cash_movements WHERE client_id = ? AND id = ?
	`, bookID, id)
	m, err := scanCashMovement(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return m, err
}

func scanCashMovement(row scannable) (*domain.CashMovement, error) {
	var m domain.CashMovement
	var kind, cAt string
	if err := row.Scan(&m.ID, &m.ClientID, &kind, &m.Amount, &m.CashAfter, &m.NetDepositsAfter, &m.Note,
		&m.CounterpartyPortfolioID, &m.CounterpartyPortfolioName, &m.PeerMovementID, &cAt); err != nil {
		return nil, err
	}
	m.Kind = domain.CashMovementKind(kind)
	m.CreatedAt = parseTime(cAt)
	return &m, nil
}

// InsertTransferAttempt records a completed, replayed, or conflict try.
func (s *SQLite) InsertTransferAttempt(ctx context.Context, a domain.TransferAttempt) (*domain.TransferAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if a.Attempt <= 0 {
		a.Attempt = 1
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO transfer_attempts (
			id, owner_client_id, idempotency_key, request_hash, from_portfolio_id, to_portfolio_id,
			from_portfolio_name, to_portfolio_name, amount, note, status, attempt,
			out_movement_id, in_movement_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.OwnerClientID, a.IdempotencyKey, a.RequestHash, a.FromPortfolioID, a.ToPortfolioID,
		a.FromPortfolioName, a.ToPortfolioName, a.Amount, a.Note, a.Status, a.Attempt,
		a.OutMovementID, a.InMovementID, fmtTime(a.CreatedAt)); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListTransferAttempts returns newest-first tries for an owner.
func (s *SQLite) ListTransferAttempts(ctx context.Context, ownerClientID string, limit, offset int) ([]domain.TransferAttempt, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_client_id, idempotency_key, request_hash, from_portfolio_id, to_portfolio_id,
			from_portfolio_name, to_portfolio_name, amount, note, status, attempt,
			out_movement_id, in_movement_id, created_at
		FROM transfer_attempts WHERE owner_client_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, ownerClientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TransferAttempt
	for rows.Next() {
		a, err := scanTransferAttempt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// CountTransferAttempts counts tries for an owner.
func (s *SQLite) CountTransferAttempts(ctx context.Context, ownerClientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transfer_attempts WHERE owner_client_id = ?`, ownerClientID).Scan(&n)
	return n, err
}

// NextTransferAttempt is 1 + max attempt for this owner key.
func (s *SQLite) NextTransferAttempt(ctx context.Context, ownerClientID, key string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(attempt), 0) FROM transfer_attempts WHERE owner_client_id = ? AND idempotency_key = ?
	`, ownerClientID, key).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n + 1, nil
}

func scanTransferAttempt(row scannable) (*domain.TransferAttempt, error) {
	var a domain.TransferAttempt
	var created string
	if err := row.Scan(
		&a.ID, &a.OwnerClientID, &a.IdempotencyKey, &a.RequestHash, &a.FromPortfolioID, &a.ToPortfolioID,
		&a.FromPortfolioName, &a.ToPortfolioName, &a.Amount, &a.Note, &a.Status, &a.Attempt,
		&a.OutMovementID, &a.InMovementID, &created,
	); err != nil {
		return nil, err
	}
	a.CreatedAt = parseTime(created)
	return &a, nil
}
