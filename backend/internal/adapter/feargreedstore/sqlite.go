package feargreedstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists fear-and-greed samples across restarts.
type SQLite struct {
	db   *sql.DB
	path string
}

// Open opens or creates the fear-and-greed history database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("fear greed sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create fear greed db dir: %w", err)
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
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
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

// Path returns the absolute database path.
func (s *SQLite) Path() string { return s.path }

// Close closes the database.
func (s *SQLite) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS fear_greed_samples (
	sampled_at_ms INTEGER PRIMARY KEY,
	score         INTEGER NOT NULL,
	level         TEXT NOT NULL,
	top_id        TEXT NOT NULL DEFAULT '',
	top_pull      REAL NOT NULL DEFAULT 0
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("fear greed schema: %w", err)
	}
	return nil
}

// SaveFearGreed upserts one 15-minute bucket.
func (s *SQLite) SaveFearGreed(ctx context.Context, sample domain.FearGreedSample) error {
	if s == nil || s.db == nil || sample.Time.IsZero() {
		return nil
	}
	ms := sample.Time.UTC().UnixMilli()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO fear_greed_samples (sampled_at_ms, score, level, top_id, top_pull)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(sampled_at_ms) DO UPDATE SET
	score = excluded.score,
	level = excluded.level,
	top_id = excluded.top_id,
	top_pull = excluded.top_pull
`, ms, sample.Score, sample.Level, sample.TopID, sample.TopPull)
	if err != nil {
		return fmt.Errorf("save fear greed: %w", err)
	}
	return nil
}

// ListFearGreed returns samples at or after since, oldest first.
func (s *SQLite) ListFearGreed(ctx context.Context, since time.Time) ([]domain.FearGreedSample, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT sampled_at_ms, score, level, top_id, top_pull
FROM fear_greed_samples
WHERE sampled_at_ms >= ?
ORDER BY sampled_at_ms ASC
`, since.UTC().UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("list fear greed: %w", err)
	}
	defer rows.Close()
	out := []domain.FearGreedSample{}
	for rows.Next() {
		var ms int64
		var rec domain.FearGreedSample
		if err := rows.Scan(&ms, &rec.Score, &rec.Level, &rec.TopID, &rec.TopPull); err != nil {
			return nil, err
		}
		rec.Time = time.UnixMilli(ms).UTC()
		out = append(out, rec)
	}
	return out, rows.Err()
}

// PurgeFearGreedBefore deletes samples older than the cutoff.
func (s *SQLite) PurgeFearGreedBefore(ctx context.Context, before time.Time) (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM fear_greed_samples WHERE sampled_at_ms < ?`, before.UTC().UnixMilli())
	if err != nil {
		return 0, fmt.Errorf("purge fear greed: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
