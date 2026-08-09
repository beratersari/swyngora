package portfoliostore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func scanShare(sc interface{ Scan(dest ...any) error }) (*domain.PortfolioShare, error) {
	var sh domain.PortfolioShare
	var role, cAt, uAt string
	err := sc.Scan(&sh.PortfolioID, &sh.OwnerClientID, &sh.GranteeClientID, &role, &cAt, &uAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sh.Role = domain.PortfolioShareRole(role)
	sh.CreatedAt = parseTime(cAt)
	sh.UpdatedAt = parseTime(uAt)
	return &sh, nil
}

const shareCols = `portfolio_id, owner_client_id, grantee_client_id, role, created_at, updated_at`

func (s *SQLite) CreatePortfolioShare(ctx context.Context, share domain.PortfolioShare) (*domain.PortfolioShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if share.CreatedAt.IsZero() {
		share.CreatedAt = time.Now().UTC()
	}
	if share.UpdatedAt.IsZero() {
		share.UpdatedAt = share.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO portfolio_shares (portfolio_id, owner_client_id, grantee_client_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, share.PortfolioID, share.OwnerClientID, share.GranteeClientID, string(share.Role),
		share.CreatedAt.UTC().Format(time.RFC3339Nano), share.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, fmt.Errorf("%w: portfolio already shared with this user", domain.ErrInvalidArgument)
		}
		return nil, fmt.Errorf("portfolio share create: %w", err)
	}
	cp := share
	return &cp, nil
}

func (s *SQLite) UpdatePortfolioShareRole(ctx context.Context, portfolioID, granteeClientID string, role domain.PortfolioShareRole, at time.Time) (*domain.PortfolioShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE portfolio_shares SET role = ?, updated_at = ?
		WHERE portfolio_id = ? AND grantee_client_id = ?
	`, string(role), at.UTC().Format(time.RFC3339Nano), portfolioID, granteeClientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetPortfolioShare(ctx, portfolioID, granteeClientID)
}

func (s *SQLite) GetPortfolioShare(ctx context.Context, portfolioID, granteeClientID string) (*domain.PortfolioShare, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+shareCols+` FROM portfolio_shares WHERE portfolio_id = ? AND grantee_client_id = ?`,
		portfolioID, granteeClientID)
	return scanShare(row)
}

func (s *SQLite) listShares(ctx context.Context, q string, args ...any) ([]domain.PortfolioShare, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PortfolioShare
	for rows.Next() {
		sh, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sh)
	}
	return out, rows.Err()
}

func (s *SQLite) ListPortfolioSharesByBook(ctx context.Context, portfolioID string) ([]domain.PortfolioShare, error) {
	return s.listShares(ctx, `SELECT `+shareCols+` FROM portfolio_shares WHERE portfolio_id = ? ORDER BY created_at ASC`, portfolioID)
}

func (s *SQLite) ListPortfolioSharesByOwner(ctx context.Context, ownerClientID string) ([]domain.PortfolioShare, error) {
	return s.listShares(ctx, `SELECT `+shareCols+` FROM portfolio_shares WHERE owner_client_id = ? ORDER BY created_at ASC`, ownerClientID)
}

func (s *SQLite) ListPortfolioSharesForGrantee(ctx context.Context, granteeClientID string) ([]domain.PortfolioShare, error) {
	return s.listShares(ctx, `SELECT `+shareCols+` FROM portfolio_shares WHERE grantee_client_id = ? ORDER BY created_at ASC`, granteeClientID)
}

func (s *SQLite) DeletePortfolioShare(ctx context.Context, portfolioID, granteeClientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM portfolio_shares WHERE portfolio_id = ? AND grantee_client_id = ?
	`, portfolioID, granteeClientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLite) CountPortfolioShares(ctx context.Context, portfolioID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM portfolio_shares WHERE portfolio_id = ?`, portfolioID).Scan(&n)
	return n, err
}
