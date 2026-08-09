package accountstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists account close/reopen state.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the accounts database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("account sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(abs) + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &SQLite{db: db, path: abs}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Path() string { return s.path }

// CloseDB releases the database handle (named to avoid clashing with AccountPort.Close).
func (s *SQLite) CloseDB() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLite) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS accounts (
	client_id   TEXT PRIMARY KEY NOT NULL,
	status      TEXT NOT NULL,
	closed_at   TEXT,
	purge_at    TEXT,
	reopened_at TEXT,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_accounts_purge ON accounts(status, purge_at);
CREATE TABLE IF NOT EXISTS api_keys (
	id          TEXT PRIMARY KEY NOT NULL,
	client_id   TEXT NOT NULL,
	name        TEXT NOT NULL,
	prefix      TEXT NOT NULL,
	hash        TEXT NOT NULL UNIQUE,
	permission  TEXT NOT NULL,
	created_at  TEXT NOT NULL,
	last_used_at TEXT,
	revoked_at  TEXT
);
CREATE INDEX IF NOT EXISTS idx_api_keys_client ON api_keys(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(hash);
`)
	return err
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseNullTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, ns.String)
	if err != nil {
		t2, err2 := time.Parse(time.RFC3339, ns.String)
		if err2 != nil {
			return nil
		}
		t = t2
	}
	u := t.UTC()
	return &u
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t.UTC()
}

const cols = `client_id, status, closed_at, purge_at, reopened_at, created_at, updated_at`

func scanAcc(sc interface{ Scan(dest ...any) error }) (*domain.Account, error) {
	var a domain.Account
	var closed, purge, reopened sql.NullString
	var created, updated string
	if err := sc.Scan(&a.ClientID, &a.Status, &closed, &purge, &reopened, &created, &updated); err != nil {
		return nil, err
	}
	a.ClosedAt = parseNullTime(closed)
	a.PurgeAt = parseNullTime(purge)
	a.ReopenedAt = parseNullTime(reopened)
	a.CreatedAt = parseTime(created)
	a.UpdatedAt = parseTime(updated)
	return &a, nil
}

func (s *SQLite) Get(ctx context.Context, clientID string) (*domain.Account, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM accounts WHERE client_id = ?`, clientID)
	a, err := scanAcc(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return a, err
}

func (s *SQLite) UpsertActive(ctx context.Context, clientID string, at time.Time) (*domain.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := at.UTC().Format(time.RFC3339Nano)
	// Do not overwrite a closed account.
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (client_id, status, created_at, updated_at)
		VALUES (?, 'active', ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET
			updated_at = excluded.updated_at
		WHERE accounts.status = 'active'
	`, clientID, now, now)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, clientID)
}

func (s *SQLite) Close(ctx context.Context, clientID string, closedAt, purgeAt time.Time) (*domain.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := closedAt.UTC().Format(time.RFC3339Nano)
	p := purgeAt.UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (client_id, status, closed_at, purge_at, created_at, updated_at)
		VALUES (?, 'closed', ?, ?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET
			status = 'closed', closed_at = excluded.closed_at, purge_at = excluded.purge_at,
			reopened_at = NULL, updated_at = excluded.updated_at
	`, clientID, c, p, c, c)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, clientID)
}

func (s *SQLite) Reopen(ctx context.Context, clientID string, at time.Time) (*domain.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := at.UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `
		UPDATE accounts SET status = 'active', closed_at = NULL, purge_at = NULL,
			reopened_at = ?, updated_at = ?
		WHERE client_id = ? AND status = 'closed'
	`, t, t, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.Get(ctx, clientID)
}

func (s *SQLite) ListDueForPurge(ctx context.Context, now time.Time, limit int) ([]domain.Account, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+cols+` FROM accounts
		WHERE status = 'closed' AND purge_at IS NOT NULL AND purge_at <= ?
		ORDER BY purge_at ASC LIMIT ?
	`, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Account
	for rows.Next() {
		a, err := scanAcc(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *SQLite) MarkPurged(ctx context.Context, clientID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE accounts SET status = 'purged', updated_at = ? WHERE client_id = ?
	`, at.UTC().Format(time.RFC3339Nano), clientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLite) Delete(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE client_id = ?`, clientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
