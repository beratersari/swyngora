package alertstore

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

// SQLite is a file-backed price alert store implementing domain.PriceAlertPort.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens (or creates) a SQLite database at path and migrates schema.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("alerts sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create alerts db dir: %w", err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite abs path: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(abs) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open alerts sqlite: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("alerts sqlite wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("alerts sqlite foreign_keys: %w", err)
	}
	s := &SQLite{db: db, path: abs}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS price_alerts (
	id              TEXT PRIMARY KEY NOT NULL,
	client_id       TEXT NOT NULL,
	exchange        TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	condition       TEXT NOT NULL,
	target_price    REAL NOT NULL,
	status          TEXT NOT NULL,
	created_at      TEXT NOT NULL,
	triggered_at    TEXT,
	triggered_price REAL NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_price_alerts_client ON price_alerts(client_id);
CREATE INDEX IF NOT EXISTS idx_price_alerts_active ON price_alerts(status) WHERE status = 'active';
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("alerts sqlite migrate: %w", err)
	}
	return nil
}

// Path returns the absolute database file path.
func (s *SQLite) Path() string { return s.path }

// Close releases the database handle.
func (s *SQLite) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Create inserts a new alert. Caller supplies a unique ID and Active status.
func (s *SQLite) Create(ctx context.Context, alert domain.PriceAlert) (*domain.PriceAlert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}
	if alert.Status == "" {
		alert.Status = domain.AlertStatusActive
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO price_alerts (
			id, client_id, exchange, symbol, condition, target_price,
			status, created_at, triggered_at, triggered_price
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		alert.ID, alert.ClientID, string(alert.Exchange), alert.Symbol,
		string(alert.Condition), alert.TargetPrice, string(alert.Status),
		alert.CreatedAt.UTC().Format(time.RFC3339Nano),
		nullTime(alert.TriggeredAt), alert.TriggeredPrice,
	)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite insert: %w", err)
	}
	return cloneAlert(&alert), nil
}

// Get returns one alert for the client or ErrNotFound.
func (s *SQLite) Get(ctx context.Context, clientID, id string) (*domain.PriceAlert, error) {
	a, err := s.scanOne(ctx, s.db, `
		SELECT id, client_id, exchange, symbol, condition, target_price,
		       status, created_at, triggered_at, triggered_price
		FROM price_alerts WHERE id = ? AND client_id = ?
	`, id, clientID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// ListByClient returns all alerts for a client (newest first).
func (s *SQLite) ListByClient(ctx context.Context, clientID string) ([]domain.PriceAlert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, exchange, symbol, condition, target_price,
		       status, created_at, triggered_at, triggered_price
		FROM price_alerts WHERE client_id = ?
		ORDER BY created_at DESC
	`, clientID)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite list: %w", err)
	}
	defer rows.Close()
	return scanAll(rows)
}

// ListActive returns every active alert (any client).
func (s *SQLite) ListActive(ctx context.Context) ([]domain.PriceAlert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, exchange, symbol, condition, target_price,
		       status, created_at, triggered_at, triggered_price
		FROM price_alerts WHERE status = ?
		ORDER BY created_at ASC
	`, string(domain.AlertStatusActive))
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite list active: %w", err)
	}
	defer rows.Close()
	return scanAll(rows)
}

// MarkTriggered transitions active → triggered exactly once.
func (s *SQLite) MarkTriggered(ctx context.Context, id string, price float64, at time.Time) (*domain.PriceAlert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE price_alerts
		SET status = ?, triggered_at = ?, triggered_price = ?
		WHERE id = ? AND status = ?
	`, string(domain.AlertStatusTriggered), at.UTC().Format(time.RFC3339Nano), price,
		id, string(domain.AlertStatusActive))
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite mark triggered: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	a, err := s.scanOne(ctx, s.db, `
		SELECT id, client_id, exchange, symbol, condition, target_price,
		       status, created_at, triggered_at, triggered_price
		FROM price_alerts WHERE id = ?
	`, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// Delete removes an alert owned by clientID.
func (s *SQLite) Delete(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM price_alerts WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return fmt.Errorf("alerts sqlite delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CountByClient returns how many alerts the client owns.
func (s *SQLite) CountByClient(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM price_alerts WHERE client_id = ?`, clientID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("alerts sqlite count: %w", err)
	}
	return n, nil
}

type rowScanner interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *SQLite) scanOne(ctx context.Context, q rowScanner, query string, args ...any) (*domain.PriceAlert, error) {
	var (
		a           domain.PriceAlert
		ex, cond, st, createdRaw string
		trigRaw     sql.NullString
	)
	err := q.QueryRowContext(ctx, query, args...).Scan(
		&a.ID, &a.ClientID, &ex, &a.Symbol, &cond, &a.TargetPrice,
		&st, &createdRaw, &trigRaw, &a.TriggeredPrice,
	)
	if err != nil {
		return nil, err
	}
	a.Exchange = domain.Exchange(ex)
	a.Condition = domain.AlertCondition(cond)
	a.Status = domain.AlertStatus(st)
	a.CreatedAt = parseTime(createdRaw)
	if trigRaw.Valid && trigRaw.String != "" {
		t := parseTime(trigRaw.String)
		a.TriggeredAt = &t
	}
	return &a, nil
}

func scanAll(rows *sql.Rows) ([]domain.PriceAlert, error) {
	out := make([]domain.PriceAlert, 0)
	for rows.Next() {
		var (
			a                      domain.PriceAlert
			ex, cond, st, createdRaw string
			trigRaw                sql.NullString
		)
		if err := rows.Scan(
			&a.ID, &a.ClientID, &ex, &a.Symbol, &cond, &a.TargetPrice,
			&st, &createdRaw, &trigRaw, &a.TriggeredPrice,
		); err != nil {
			return nil, fmt.Errorf("alerts sqlite scan: %w", err)
		}
		a.Exchange = domain.Exchange(ex)
		a.Condition = domain.AlertCondition(cond)
		a.Status = domain.AlertStatus(st)
		a.CreatedAt = parseTime(createdRaw)
		if trigRaw.Valid && trigRaw.String != "" {
			t := parseTime(trigRaw.String)
			a.TriggeredAt = &t
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseTime(raw string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}
		}
	}
	return t.UTC()
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func cloneAlert(a *domain.PriceAlert) *domain.PriceAlert {
	if a == nil {
		return nil
	}
	cp := *a
	if a.TriggeredAt != nil {
		t := a.TriggeredAt.UTC()
		cp.TriggeredAt = &t
	}
	return &cp
}