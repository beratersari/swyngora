package radarstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/sqliteutil"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists Setup Radar hits and after-fills.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the radar history database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("radar sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create radar db dir: %w", err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(abs) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
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

// Path returns the absolute database path.
func (s *SQLite) Path() string { return s.path }

// Close closes the database.
func (s *SQLite) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS radar_cases (
	id           TEXT PRIMARY KEY NOT NULL,
	exchange     TEXT NOT NULL,
	symbol       TEXT NOT NULL,
	found_at     TEXT NOT NULL,
	found_price  REAL NOT NULL DEFAULT 0,
	similarity   REAL NOT NULL DEFAULT 0,
	coverage     REAL NOT NULL DEFAULT 0,
	lean         TEXT NOT NULL DEFAULT '',
	why_similar  TEXT NOT NULL DEFAULT '',
	after_story  TEXT NOT NULL DEFAULT '',
	summary      TEXT NOT NULL DEFAULT '',
	lookback     TEXT NOT NULL DEFAULT '',
	window       TEXT NOT NULL DEFAULT '',
	compared_json TEXT NOT NULL DEFAULT '[]',
	created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_radar_cases_found ON radar_cases(found_at DESC);
CREATE INDEX IF NOT EXISTS idx_radar_cases_sym ON radar_cases(exchange, symbol, found_at DESC);

CREATE TABLE IF NOT EXISTS radar_afters (
	case_id     TEXT NOT NULL,
	horizon     TEXT NOT NULL,
	due_at      TEXT NOT NULL,
	filled_at   TEXT,
	return_pct  REAL NOT NULL DEFAULT 0,
	direction   TEXT NOT NULL DEFAULT '',
	price       REAL NOT NULL DEFAULT 0,
	useful      INTEGER,
	PRIMARY KEY (case_id, horizon),
	FOREIGN KEY (case_id) REFERENCES radar_cases(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_radar_afters_due ON radar_afters(due_at) WHERE filled_at IS NULL;
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	return sqliteutil.SetUserVersion(s.db, 1)
}

// InsertIfQuiet stores a case when that coin has no hit inside cooldown.
func (s *SQLite) InsertIfQuiet(ctx context.Context, c domain.RadarCase, cooldown time.Duration) (*domain.RadarCase, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" || c.Symbol == "" {
		return nil, false, fmt.Errorf("radar case id and symbol are required")
	}
	if cooldown <= 0 {
		cooldown = domain.DefaultRadarCooldown
	}
	cut := c.FoundAt.Add(-cooldown).UTC().Format(time.RFC3339Nano)
	var n int
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(1) FROM radar_cases
WHERE symbol = ? AND found_at > ? AND (? = '' OR ? = 'all' OR exchange = ?)`,
		c.Symbol, cut, c.Exchange, c.Exchange, c.Exchange,
	).Scan(&n)
	if err != nil {
		return nil, false, err
	}
	if n > 0 {
		return nil, false, nil
	}
	cmp, err := json.Marshal(c.Compared)
	if err != nil {
		return nil, false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
INSERT INTO radar_cases(
	id, exchange, symbol, found_at, found_price, similarity, coverage, lean,
	why_similar, after_story, summary, lookback, window, compared_json, created_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.Exchange, c.Symbol, c.FoundAt.UTC().Format(time.RFC3339Nano), c.FoundPrice,
		c.Similarity, c.Coverage, c.Lean, c.WhySimilar, c.AfterStory, c.Summary,
		c.Lookback, c.Window, string(cmp), time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, false, err
	}
	for _, a := range c.After {
		if a.Horizon == "" {
			continue
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO radar_afters(case_id, horizon, due_at) VALUES (?,?,?)`,
			c.ID, a.Horizon, a.DueAt.UTC().Format(time.RFC3339Nano),
		)
		if err != nil {
			return nil, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	out := c
	return &out, true, nil
}

// List returns newest cases first.
func (s *SQLite) List(ctx context.Context, q domain.RadarListQuery) ([]domain.RadarCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := domain.ClampRadarListLimit(q.Limit)
	args := []any{}
	where := []string{"1=1"}
	if ex := strings.ToLower(strings.TrimSpace(q.Exchange)); ex != "" && ex != "all" {
		where = append(where, "exchange = ?")
		args = append(args, ex)
	}
	if sym := strings.ToUpper(strings.TrimSpace(q.Symbol)); sym != "" && sym != "ALL" {
		where = append(where, "symbol = ?")
		args = append(args, sym)
	}
	if !q.Since.IsZero() {
		where = append(where, "found_at >= ?")
		args = append(args, q.Since.UTC().Format(time.RFC3339Nano))
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
SELECT id, exchange, symbol, found_at, found_price, similarity, coverage, lean,
	why_similar, after_story, summary, lookback, window, compared_json
FROM radar_cases
WHERE `+strings.Join(where, " AND ")+`
ORDER BY found_at DESC
LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RadarCase
	ids := []string{}
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
		ids = append(ids, c.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return []domain.RadarCase{}, nil
	}
	afters, err := s.loadAfters(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].After = afters[out[i].ID]
		if out[i].After == nil {
			out[i].After = []domain.RadarAfter{}
		}
	}
	return out, nil
}

// ListDueAfters returns unfilled horizons that are due.
func (s *SQLite) ListDueAfters(ctx context.Context, now time.Time, limit int) ([]domain.RadarAfterDue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT a.case_id, c.exchange, c.symbol, c.found_at, c.found_price, c.lean, a.horizon, a.due_at
FROM radar_afters a
JOIN radar_cases c ON c.id = a.case_id
WHERE a.filled_at IS NULL AND a.due_at <= ?
ORDER BY a.due_at ASC
LIMIT ?`, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RadarAfterDue
	for rows.Next() {
		var d domain.RadarAfterDue
		var foundAt, dueAt string
		if err := rows.Scan(&d.CaseID, &d.Exchange, &d.Symbol, &foundAt, &d.FoundPrice, &d.Lean, &d.Horizon, &dueAt); err != nil {
			return nil, err
		}
		d.FoundAt, _ = time.Parse(time.RFC3339Nano, foundAt)
		d.DueAt, _ = time.Parse(time.RFC3339Nano, dueAt)
		out = append(out, d)
	}
	return out, rows.Err()
}

// FillAfter writes one horizon outcome.
func (s *SQLite) FillAfter(ctx context.Context, caseID, horizon string, fill domain.RadarAfter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fill.FilledAt == nil {
		now := time.Now().UTC()
		fill.FilledAt = &now
	}
	var useful any
	if fill.Useful != nil {
		if *fill.Useful {
			useful = 1
		} else {
			useful = 0
		}
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE radar_afters
SET filled_at = ?, return_pct = ?, direction = ?, price = ?, useful = ?
WHERE case_id = ? AND horizon = ? AND filled_at IS NULL`,
		fill.FilledAt.UTC().Format(time.RFC3339Nano), fill.ReturnPct, fill.Direction, fill.Price, useful,
		caseID, horizon,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SetFoundPrice records the print at found time when it was missing.
func (s *SQLite) SetFoundPrice(ctx context.Context, caseID string, price float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `UPDATE radar_cases SET found_price = ? WHERE id = ?`, price, caseID)
	return err
}

func (s *SQLite) loadAfters(ctx context.Context, ids []string) (map[string][]domain.RadarAfter, error) {
	out := map[string][]domain.RadarAfter{}
	if len(ids) == 0 {
		return out, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = "?"
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT case_id, horizon, due_at, filled_at, return_pct, direction, price, useful
FROM radar_afters
WHERE case_id IN (`+strings.Join(ph, ",")+`)
ORDER BY due_at ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id, horizon, dueAt string
			filledAt           sql.NullString
			ret, price         float64
			dir                string
			useful             sql.NullInt64
		)
		if err := rows.Scan(&id, &horizon, &dueAt, &filledAt, &ret, &dir, &price, &useful); err != nil {
			return nil, err
		}
		a := domain.RadarAfter{Horizon: horizon, ReturnPct: ret, Direction: dir, Price: price}
		a.DueAt, _ = time.Parse(time.RFC3339Nano, dueAt)
		if filledAt.Valid && filledAt.String != "" {
			t, err := time.Parse(time.RFC3339Nano, filledAt.String)
			if err == nil {
				a.FilledAt = &t
			}
		}
		if useful.Valid {
			v := useful.Int64 == 1
			a.Useful = &v
		}
		out[id] = append(out[id], a)
	}
	return out, rows.Err()
}

type caseScanner interface {
	Scan(dest ...any) error
}

func scanCase(row caseScanner) (domain.RadarCase, error) {
	var c domain.RadarCase
	var foundAt, cmp string
	if err := row.Scan(&c.ID, &c.Exchange, &c.Symbol, &foundAt, &c.FoundPrice, &c.Similarity, &c.Coverage, &c.Lean,
		&c.WhySimilar, &c.AfterStory, &c.Summary, &c.Lookback, &c.Window, &cmp); err != nil {
		return c, err
	}
	c.FoundAt, _ = time.Parse(time.RFC3339Nano, foundAt)
	if cmp != "" && cmp != "[]" {
		_ = json.Unmarshal([]byte(cmp), &c.Compared)
	}
	return c, nil
}
