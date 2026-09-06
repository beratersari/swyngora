package optionsstore

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

// SQLite persists options desk samples across restarts.
type SQLite struct {
	db   *sql.DB
	path string
}

// Open opens or creates the options history database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("options sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create options db dir: %w", err)
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
CREATE TABLE IF NOT EXISTS options_samples (
	asset              TEXT NOT NULL,
	sampled_at_ms      INTEGER NOT NULL,
	open_interest      REAL NOT NULL DEFAULT 0,
	open_interest_usd  REAL NOT NULL DEFAULT 0,
	call_oi            REAL NOT NULL DEFAULT 0,
	put_oi             REAL NOT NULL DEFAULT 0,
	put_call_oi        REAL NOT NULL DEFAULT 0,
	has_put_call       INTEGER NOT NULL DEFAULT 0,
	dvol               REAL NOT NULL DEFAULT 0,
	has_dvol           INTEGER NOT NULL DEFAULT 0,
	atm_iv             REAL NOT NULL DEFAULT 0,
	has_atm            INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (asset, sampled_at_ms)
);
CREATE INDEX IF NOT EXISTS idx_options_samples_lookup
	ON options_samples(asset, sampled_at_ms);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("options schema: %w", err)
	}
	return nil
}

// InsertSample upserts one 5-minute bucket.
func (s *SQLite) InsertSample(ctx context.Context, rec domain.OptionsSample) (bool, error) {
	if s == nil || s.db == nil || rec.Asset == "" || rec.SampledAt.IsZero() {
		return false, nil
	}
	asset, err := domain.ParseOptionsAsset(rec.Asset)
	if err != nil {
		return false, err
	}
	ms := rec.SampledAt.UTC().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO options_samples (
	asset, sampled_at_ms, open_interest, open_interest_usd, call_oi, put_oi,
	put_call_oi, has_put_call, dvol, has_dvol, atm_iv, has_atm
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, asset, ms, rec.OpenInterest, rec.OpenInterestUSD,
		rec.CallOI, rec.PutOI, rec.PutCallOI, boolInt(rec.HasPutCallOI),
		rec.DVOL, boolInt(rec.HasDVOL), rec.AtmIV, boolInt(rec.HasAtmIV))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 1 {
		return true, nil
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE options_samples SET
	open_interest=?, open_interest_usd=?, call_oi=?, put_oi=?,
	put_call_oi=?, has_put_call=?, dvol=?, has_dvol=?, atm_iv=?, has_atm=?
WHERE asset=? AND sampled_at_ms=?
`, rec.OpenInterest, rec.OpenInterestUSD, rec.CallOI, rec.PutOI,
		rec.PutCallOI, boolInt(rec.HasPutCallOI), rec.DVOL, boolInt(rec.HasDVOL),
		rec.AtmIV, boolInt(rec.HasAtmIV), asset, ms)
	return false, err
}

// ListSamples returns oldest-first prints in [from, to].
func (s *SQLite) ListSamples(ctx context.Context, asset string, from, to time.Time, limit int) ([]domain.OptionsSample, error) {
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > domain.OptionsHistoryListLimit {
		limit = domain.OptionsHistoryListLimit
	}
	fromMs, toMs := int64(0), time.Now().UTC().Add(time.Hour).UnixMilli()
	if !from.IsZero() {
		fromMs = from.UTC().UnixMilli()
	}
	if !to.IsZero() {
		toMs = to.UTC().UnixMilli()
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT sampled_at_ms, open_interest, open_interest_usd, call_oi, put_oi,
	put_call_oi, has_put_call, dvol, has_dvol, atm_iv, has_atm
FROM options_samples
WHERE asset = ? AND sampled_at_ms >= ? AND sampled_at_ms <= ?
ORDER BY sampled_at_ms ASC
LIMIT ?
`, asset, fromMs, toMs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.OptionsSample, 0, 64)
	for rows.Next() {
		var rec domain.OptionsSample
		var ms int64
		var hasPCR, hasDVOL, hasATM int
		if err := rows.Scan(&ms, &rec.OpenInterest, &rec.OpenInterestUSD, &rec.CallOI, &rec.PutOI,
			&rec.PutCallOI, &hasPCR, &rec.DVOL, &hasDVOL, &rec.AtmIV, &hasATM); err != nil {
			return nil, err
		}
		rec.Asset = asset
		rec.SampledAt = time.UnixMilli(ms).UTC()
		rec.HasPutCallOI = hasPCR != 0
		rec.HasDVOL = hasDVOL != 0
		rec.HasAtmIV = hasATM != 0
		out = append(out, rec)
	}
	return out, rows.Err()
}

// PurgeOlderThan deletes samples before cutoff.
func (s *SQLite) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.db == nil || cutoff.IsZero() {
		return 0, nil
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM options_samples WHERE sampled_at_ms < ?`, cutoff.UTC().UnixMilli())
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
