package optionsstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/sqliteutil"
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
	const strikes = `
CREATE TABLE IF NOT EXISTS options_strike_oi (
	asset          TEXT NOT NULL,
	sampled_at_ms  INTEGER NOT NULL,
	expiry_ms      INTEGER NOT NULL,
	strike         REAL NOT NULL,
	call_oi        REAL NOT NULL DEFAULT 0,
	put_oi         REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (asset, sampled_at_ms, expiry_ms, strike)
);
CREATE INDEX IF NOT EXISTS idx_options_strike_oi_lookup
	ON options_strike_oi(asset, sampled_at_ms);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("options schema: %w", err)
	}
	if _, err := s.db.Exec(strikes); err != nil {
		return fmt.Errorf("options strike schema: %w", err)
	}
	alters := []string{
		`ALTER TABLE options_samples ADD COLUMN index_price REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE options_samples ADD COLUMN has_index INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE options_samples ADD COLUMN gamma_net_usd REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE options_samples ADD COLUMN has_gamma INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE options_samples ADD COLUMN gamma_regime TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE options_samples ADD COLUMN gamma_flip REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE options_samples ADD COLUMN has_gamma_flip INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE options_samples ADD COLUMN gamma_flip_side TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alters {
		if err := sqliteutil.ExecAllowExists(s.db, stmt); err != nil {
			return err
		}
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
	put_call_oi, has_put_call, dvol, has_dvol, atm_iv, has_atm,
	index_price, has_index, gamma_net_usd, has_gamma, gamma_regime,
	gamma_flip, has_gamma_flip, gamma_flip_side
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, asset, ms, rec.OpenInterest, rec.OpenInterestUSD,
		rec.CallOI, rec.PutOI, rec.PutCallOI, boolInt(rec.HasPutCallOI),
		rec.DVOL, boolInt(rec.HasDVOL), rec.AtmIV, boolInt(rec.HasAtmIV),
		rec.IndexPrice, boolInt(rec.HasIndex), rec.GammaNetUSD, boolInt(rec.HasGamma), rec.GammaRegime,
		rec.GammaFlip, boolInt(rec.HasGammaFlip), rec.GammaFlipSide)
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
	put_call_oi=?, has_put_call=?, dvol=?, has_dvol=?, atm_iv=?, has_atm=?,
	index_price=?, has_index=?, gamma_net_usd=?, has_gamma=?, gamma_regime=?,
	gamma_flip=?, has_gamma_flip=?, gamma_flip_side=?
WHERE asset=? AND sampled_at_ms=?
`, rec.OpenInterest, rec.OpenInterestUSD, rec.CallOI, rec.PutOI,
		rec.PutCallOI, boolInt(rec.HasPutCallOI), rec.DVOL, boolInt(rec.HasDVOL),
		rec.AtmIV, boolInt(rec.HasAtmIV),
		rec.IndexPrice, boolInt(rec.HasIndex), rec.GammaNetUSD, boolInt(rec.HasGamma), rec.GammaRegime,
		rec.GammaFlip, boolInt(rec.HasGammaFlip), rec.GammaFlipSide, asset, ms)
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
	put_call_oi, has_put_call, dvol, has_dvol, atm_iv, has_atm,
	index_price, has_index, gamma_net_usd, has_gamma, gamma_regime,
	gamma_flip, has_gamma_flip, gamma_flip_side
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
		var hasPCR, hasDVOL, hasATM, hasIndex, hasGamma, hasFlip int
		if err := rows.Scan(&ms, &rec.OpenInterest, &rec.OpenInterestUSD, &rec.CallOI, &rec.PutOI,
			&rec.PutCallOI, &hasPCR, &rec.DVOL, &hasDVOL, &rec.AtmIV, &hasATM,
			&rec.IndexPrice, &hasIndex, &rec.GammaNetUSD, &hasGamma, &rec.GammaRegime,
			&rec.GammaFlip, &hasFlip, &rec.GammaFlipSide); err != nil {
			return nil, err
		}
		rec.Asset = asset
		rec.SampledAt = time.UnixMilli(ms).UTC()
		rec.HasPutCallOI = hasPCR != 0
		rec.HasDVOL = hasDVOL != 0
		rec.HasAtmIV = hasATM != 0
		rec.HasIndex = hasIndex != 0
		rec.HasGamma = hasGamma != 0
		rec.HasGammaFlip = hasFlip != 0
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
	if _, err := s.db.ExecContext(ctx, `DELETE FROM options_strike_oi WHERE sampled_at_ms < ?`, cutoff.UTC().UnixMilli()); err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// ReplaceStrikeBook replaces the call/put ladder for one asset and bucket.
func (s *SQLite) ReplaceStrikeBook(ctx context.Context, asset string, at time.Time, points []domain.OptionsStrikeOIPoint) error {
	if s == nil || s.db == nil || asset == "" || at.IsZero() {
		return nil
	}
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return err
	}
	ms := at.UTC().UnixMilli()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM options_strike_oi WHERE asset=? AND sampled_at_ms=?`, asset, ms); err != nil {
		return err
	}
	for _, p := range points {
		if p.Expiry.IsZero() || (p.CallOI <= 0 && p.PutOI <= 0) {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO options_strike_oi (asset, sampled_at_ms, expiry_ms, strike, call_oi, put_oi)
VALUES (?, ?, ?, ?, ?, ?)
`, asset, ms, p.Expiry.UTC().UnixMilli(), p.Strike, p.CallOI, p.PutOI); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// StrikeBookAt returns the latest ladder at or before at.
func (s *SQLite) StrikeBookAt(ctx context.Context, asset string, at time.Time) (domain.OptionsStrikeBook, bool, error) {
	var book domain.OptionsStrikeBook
	if s == nil || s.db == nil || at.IsZero() {
		return book, false, nil
	}
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return book, false, err
	}
	var ms sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `
SELECT MAX(sampled_at_ms) FROM options_strike_oi
WHERE asset=? AND sampled_at_ms<=?
`, asset, at.UTC().UnixMilli()).Scan(&ms); err != nil {
		return book, false, err
	}
	if !ms.Valid {
		return book, false, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT expiry_ms, strike, call_oi, put_oi
FROM options_strike_oi
WHERE asset=? AND sampled_at_ms=?
`, asset, ms.Int64)
	if err != nil {
		return book, false, err
	}
	defer rows.Close()
	book.SampledAt = time.UnixMilli(ms.Int64).UTC()
	for rows.Next() {
		var expiryMs int64
		var p domain.OptionsStrikeOIPoint
		if err := rows.Scan(&expiryMs, &p.Strike, &p.CallOI, &p.PutOI); err != nil {
			return book, false, err
		}
		p.Expiry = time.UnixMilli(expiryMs).UTC()
		book.Points = append(book.Points, p)
	}
	return book, true, rows.Err()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
