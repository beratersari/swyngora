package portfoliostore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const snapTimeLayout = time.RFC3339

func fmtSnapTime(t time.Time) string {
	return t.UTC().Format(snapTimeLayout)
}

func parseSnapTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	return time.Time{}
}

// UpsertEquitySnapshot writes or replaces the bucket row.
func (s *SQLite) UpsertEquitySnapshot(ctx context.Context, snap domain.EquitySnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if snap.BucketAt.IsZero() {
		snap.BucketAt = domain.SnapshotBucket(snap.TakenAt, domain.DefaultSnapshotInterval)
	}
	if snap.TakenAt.IsZero() {
		snap.TakenAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO portfolio_equity_snapshots (
			client_id, bucket_at, taken_at, equity, cash_balance, positions_value,
			margin_equity, unrealized_pnl, realized_pnl
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id, bucket_at) DO UPDATE SET
			taken_at = excluded.taken_at,
			equity = excluded.equity,
			cash_balance = excluded.cash_balance,
			positions_value = excluded.positions_value,
			margin_equity = excluded.margin_equity,
			unrealized_pnl = excluded.unrealized_pnl,
			realized_pnl = excluded.realized_pnl
	`, snap.ClientID, fmtSnapTime(snap.BucketAt), snap.TakenAt.UTC().Format(time.RFC3339Nano),
		snap.Equity, snap.CashBalance, snap.PositionsValue, snap.MarginEquity,
		snap.UnrealizedPnL, snap.RealizedPnL)
	if err != nil {
		return fmt.Errorf("upsert equity snapshot: %w", err)
	}
	return nil
}

// ListEquitySnapshots returns samples in [from, to] by bucket (inclusive).
func (s *SQLite) ListEquitySnapshots(ctx context.Context, clientID string, from, to time.Time) ([]domain.EquitySnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `
		SELECT client_id, bucket_at, taken_at, equity, cash_balance, positions_value,
			margin_equity, unrealized_pnl, realized_pnl
		FROM portfolio_equity_snapshots
		WHERE client_id = ? AND bucket_at >= ? AND bucket_at <= ?
		ORDER BY bucket_at ASC
	`, clientID, fmtSnapTime(from), fmtSnapTime(to))
	if err != nil {
		return nil, fmt.Errorf("list equity snapshots: %w", err)
	}
	defer rows.Close()
	return scanSnapshots(rows)
}

// LatestEquitySnapshotBefore returns the newest snapshot with bucket_at < before.
func (s *SQLite) LatestEquitySnapshotBefore(ctx context.Context, clientID string, before time.Time) (*domain.EquitySnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.db.QueryRowContext(ctx, `
		SELECT client_id, bucket_at, taken_at, equity, cash_balance, positions_value,
			margin_equity, unrealized_pnl, realized_pnl
		FROM portfolio_equity_snapshots
		WHERE client_id = ? AND bucket_at < ?
		ORDER BY bucket_at DESC
		LIMIT 1
	`, clientID, fmtSnapTime(before))
	snap, err := scanSnapshot(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// DeleteEquitySnapshotsBefore removes history older than before (retention).
func (s *SQLite) DeleteEquitySnapshotsBefore(ctx context.Context, before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM portfolio_equity_snapshots WHERE bucket_at < ?
	`, fmtSnapTime(before))
	if err != nil {
		return 0, fmt.Errorf("delete equity snapshots: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ListPortfolioClientIDs lists all paper accounts (snapshot worker).
func (s *SQLite) ListPortfolioClientIDs(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT client_id FROM portfolios ORDER BY client_id`)
	if err != nil {
		return nil, fmt.Errorf("list portfolio clients: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

type snapshotRow interface {
	Scan(dest ...any) error
}

func scanSnapshot(row snapshotRow) (*domain.EquitySnapshot, error) {
	var snap domain.EquitySnapshot
	var bucket, taken string
	if err := row.Scan(&snap.ClientID, &bucket, &taken, &snap.Equity, &snap.CashBalance,
		&snap.PositionsValue, &snap.MarginEquity, &snap.UnrealizedPnL, &snap.RealizedPnL); err != nil {
		return nil, err
	}
	snap.BucketAt = parseSnapTime(bucket)
	snap.TakenAt = parseSnapTime(taken)
	return &snap, nil
}

func scanSnapshots(rows *sql.Rows) ([]domain.EquitySnapshot, error) {
	var out []domain.EquitySnapshot
	for rows.Next() {
		snap, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *snap)
	}
	return out, rows.Err()
}
