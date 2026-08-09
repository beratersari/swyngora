package accountstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// CreateAPIKey inserts a new key metadata row.
func (s *SQLite) CreateAPIKey(ctx context.Context, k domain.APIKey) (*domain.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, client_id, name, prefix, hash, permission, created_at, last_used_at, revoked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, k.ID, k.ClientID, k.Name, k.Prefix, k.Hash, string(k.Permission),
		k.CreatedAt.UTC().Format(time.RFC3339Nano), nullTime(k.LastUsedAt), nullTime(k.RevokedAt))
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	cp := k
	return &cp, nil
}

// GetAPIKeyByHash loads by secret hash.
func (s *SQLite) GetAPIKeyByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client_id, name, prefix, hash, permission, created_at, last_used_at, revoked_at
		FROM api_keys WHERE hash = ?
	`, hash)
	return scanAPIKey(row)
}

// GetAPIKey loads one key owned by clientID.
func (s *SQLite) GetAPIKey(ctx context.Context, clientID, id string) (*domain.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client_id, name, prefix, hash, permission, created_at, last_used_at, revoked_at
		FROM api_keys WHERE id = ? AND client_id = ?
	`, id, clientID)
	return scanAPIKey(row)
}

// ListAPIKeys lists keys for a client (newest first).
func (s *SQLite) ListAPIKeys(ctx context.Context, clientID string) ([]domain.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, name, prefix, hash, permission, created_at, last_used_at, revoked_at
		FROM api_keys WHERE client_id = ?
		ORDER BY created_at DESC
	`, clientID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()
	var out []domain.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *k)
	}
	return out, rows.Err()
}

// CountActiveAPIKeys counts non-revoked keys.
func (s *SQLite) CountActiveAPIKeys(ctx context.Context, clientID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM api_keys WHERE client_id = ? AND revoked_at IS NULL
	`, clientID).Scan(&n)
	return n, err
}

// RevokeAPIKey sets revoked_at if still active.
func (s *SQLite) RevokeAPIKey(ctx context.Context, clientID, id string, at time.Time) (*domain.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE api_keys SET revoked_at = COALESCE(revoked_at, ?)
		WHERE id = ? AND client_id = ?
	`, at.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client_id, name, prefix, hash, permission, created_at, last_used_at, revoked_at
		FROM api_keys WHERE id = ? AND client_id = ?
	`, id, clientID)
	return scanAPIKey(row)
}

// TouchAPIKeyLastUsed updates last_used_at.
func (s *SQLite) TouchAPIKeyLastUsed(ctx context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ? AND revoked_at IS NULL`,
		at.UTC().Format(time.RFC3339Nano), id)
	return err
}

// DeleteAPIKeysByClient removes all keys for a purged account.
func (s *SQLite) DeleteAPIKeysByClient(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE client_id = ?`, clientID)
	return err
}

type apiKeyScanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(sc apiKeyScanner) (*domain.APIKey, error) {
	var k domain.APIKey
	var perm string
	var created string
	var lastUsed, revoked sql.NullString
	if err := sc.Scan(&k.ID, &k.ClientID, &k.Name, &k.Prefix, &k.Hash, &perm, &created, &lastUsed, &revoked); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	k.Permission = domain.APIKeyPermission(perm)
	k.CreatedAt = parseTime(created)
	k.LastUsedAt = parseNullTime(lastUsed)
	k.RevokedAt = parseNullTime(revoked)
	return &k, nil
}
