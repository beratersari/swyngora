package perpdepthstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists market-wide perp bid/ask samples across restarts.
type SQLite struct {
	db   *sql.DB
	path string
}

// Open opens or creates the perp-depth history database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("perp depth sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create perp depth db dir: %w", err)
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
CREATE TABLE IF NOT EXISTS perp_depth_samples (
	exchange       TEXT NOT NULL,
	range_mpct     INTEGER NOT NULL,
	sampled_at_ms  INTEGER NOT NULL,
	symbol         TEXT NOT NULL,
	base           TEXT NOT NULL DEFAULT '',
	bid            REAL NOT NULL DEFAULT 0,
	ask            REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (exchange, range_mpct, sampled_at_ms, symbol)
);
CREATE INDEX IF NOT EXISTS idx_perp_depth_samples_lookup
	ON perp_depth_samples(exchange, range_mpct, sampled_at_ms);
CREATE TABLE IF NOT EXISTS perp_depth_gaps (
	exchange       TEXT NOT NULL,
	range_mpct     INTEGER NOT NULL,
	sampled_at_ms  INTEGER NOT NULL,
	missing        TEXT NOT NULL DEFAULT '',
	stale          TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (exchange, range_mpct, sampled_at_ms)
);
CREATE TABLE IF NOT EXISTS perp_book_bands (
	exchange       TEXT NOT NULL,
	symbol         TEXT NOT NULL,
	range_mpct     INTEGER NOT NULL,
	sampled_at_ms  INTEGER NOT NULL,
	bid            REAL NOT NULL DEFAULT 0,
	ask            REAL NOT NULL DEFAULT 0,
	mid            REAL NOT NULL DEFAULT 0,
	cover_bid      REAL NOT NULL DEFAULT 0,
	cover_ask      REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (exchange, symbol, range_mpct, sampled_at_ms)
);
CREATE INDEX IF NOT EXISTS idx_perp_book_bands_lookup
	ON perp_book_bands(exchange, symbol, sampled_at_ms);
CREATE TABLE IF NOT EXISTS perp_book_gaps (
	symbol         TEXT NOT NULL,
	sampled_at_ms  INTEGER NOT NULL,
	missing        TEXT NOT NULL DEFAULT '',
	stale          TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (symbol, sampled_at_ms)
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("perp depth schema: %w", err)
	}
	for _, stmt := range []string{
		`ALTER TABLE perp_book_bands ADD COLUMN cover_bid REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE perp_book_bands ADD COLUMN cover_ask REAL NOT NULL DEFAULT 0`,
	} {
		if _, err := s.db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("perp book band column: %w", err)
		}
	}
	return nil
}

func rangeMilliPct(v float64) int {
	if v <= 0 {
		v = domain.DefaultPerpDepthRangePct
	}
	return int(math.Round(v * 100))
}

// InsertSamples upserts one-minute buckets.
func (s *SQLite) InsertSamples(ctx context.Context, rows []domain.PerpDepthSample) (int, error) {
	if s == nil || s.db == nil || len(rows) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO perp_depth_samples (exchange, range_mpct, sampled_at_ms, symbol, base, bid, ask)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (exchange, range_mpct, sampled_at_ms, symbol) DO UPDATE SET
	base=excluded.base, bid=excluded.bid, ask=excluded.ask
`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	n := 0
	for _, r := range rows {
		if r.Symbol == "" || r.Time.IsZero() {
			continue
		}
		ex := r.Exchange
		if ex == "" {
			ex = "all"
		}
		ms := domain.BucketPerpDepthSampleTime(r.Time).UnixMilli()
		if _, err := stmt.ExecContext(ctx, ex, rangeMilliPct(r.RangePct), ms, r.Symbol, r.Base, r.Bid, r.Ask); err != nil {
			return n, err
		}
		n++
	}
	if err := tx.Commit(); err != nil {
		return n, err
	}
	return n, nil
}

// ListSamples returns rows in [from, to], oldest first.
func (s *SQLite) ListSamples(ctx context.Context, exchange string, rangePct float64, from, to time.Time) ([]domain.PerpDepthSample, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	if to.IsZero() {
		to = time.Now().UTC().Add(time.Hour)
	}
	fromMs := int64(0)
	if !from.IsZero() {
		fromMs = from.UTC().UnixMilli()
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT sampled_at_ms, exchange, symbol, base, bid, ask
FROM perp_depth_samples
WHERE exchange = ? AND range_mpct = ? AND sampled_at_ms >= ? AND sampled_at_ms <= ?
ORDER BY sampled_at_ms ASC
LIMIT ?
`, exchange, rangeMilliPct(rangePct), fromMs, to.UTC().UnixMilli(), domain.PerpDepthHistoryListLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PerpDepthSample, 0, 64)
	for rows.Next() {
		var rec domain.PerpDepthSample
		var ms int64
		if err := rows.Scan(&ms, &rec.Exchange, &rec.Symbol, &rec.Base, &rec.Bid, &rec.Ask); err != nil {
			return nil, err
		}
		rec.Time = time.UnixMilli(ms).UTC()
		rec.RangePct = rangePct
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ListMinuteTotals returns summed bid+ask per minute, oldest first.
func (s *SQLite) ListMinuteTotals(ctx context.Context, exchange string, rangePct float64, from, to time.Time) ([]domain.PerpDepthTotalPoint, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	if to.IsZero() {
		to = time.Now().UTC().Add(time.Hour)
	}
	fromMs := int64(0)
	if !from.IsZero() {
		fromMs = from.UTC().UnixMilli()
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT sampled_at_ms, SUM(bid + ask)
FROM perp_depth_samples
WHERE exchange = ? AND range_mpct = ? AND sampled_at_ms >= ? AND sampled_at_ms <= ?
GROUP BY sampled_at_ms
ORDER BY sampled_at_ms ASC
`, exchange, rangeMilliPct(rangePct), fromMs, to.UTC().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PerpDepthTotalPoint, 0, 64)
	for rows.Next() {
		var ms int64
		var total float64
		if err := rows.Scan(&ms, &total); err != nil {
			return nil, err
		}
		if total <= 0 {
			continue
		}
		out = append(out, domain.PerpDepthTotalPoint{Time: time.UnixMilli(ms).UTC(), Total: total})
	}
	return out, rows.Err()
}

// InsertGap records a minute that has no total. A minute that already has samples is left alone.
func (s *SQLite) InsertGap(ctx context.Context, gap domain.PerpDepthHistoryGap) error {
	if s == nil || s.db == nil || gap.Time.IsZero() || (len(gap.Missing) == 0 && len(gap.Stale) == 0) {
		return nil
	}
	ex := gap.Exchange
	if ex == "" {
		ex = "all"
	}
	ms := domain.BucketPerpDepthSampleTime(gap.Time).UnixMilli()
	var n int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM perp_depth_samples
WHERE exchange = ? AND range_mpct = ? AND sampled_at_ms = ?
`, ex, rangeMilliPct(gap.RangePct), ms).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO perp_depth_gaps (exchange, range_mpct, sampled_at_ms, missing, stale)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (exchange, range_mpct, sampled_at_ms) DO UPDATE SET
	missing=excluded.missing, stale=excluded.stale
`, ex, rangeMilliPct(gap.RangePct), ms, strings.Join(gap.Missing, ","), strings.Join(gap.Stale, ","))
	return err
}

// ListGaps returns gap minutes in [from, to], oldest first.
func (s *SQLite) ListGaps(ctx context.Context, exchange string, rangePct float64, from, to time.Time) ([]domain.PerpDepthHistoryGap, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	if to.IsZero() {
		to = time.Now().UTC().Add(time.Hour)
	}
	fromMs := int64(0)
	if !from.IsZero() {
		fromMs = from.UTC().UnixMilli()
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT sampled_at_ms, missing, stale
FROM perp_depth_gaps
WHERE exchange = ? AND range_mpct = ? AND sampled_at_ms >= ? AND sampled_at_ms <= ?
ORDER BY sampled_at_ms ASC
`, exchange, rangeMilliPct(rangePct), fromMs, to.UTC().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PerpDepthHistoryGap, 0)
	for rows.Next() {
		var ms int64
		var missing, stale string
		if err := rows.Scan(&ms, &missing, &stale); err != nil {
			return nil, err
		}
		out = append(out, domain.PerpDepthHistoryGap{
			Time: time.UnixMilli(ms).UTC(), Exchange: exchange, RangePct: rangePct,
			Missing: splitVenues(missing), Stale: splitVenues(stale),
		})
	}
	return out, rows.Err()
}

// ClearGap drops a gap once that minute has a complete total.
func (s *SQLite) ClearGap(ctx context.Context, exchange string, rangePct float64, at time.Time) error {
	if s == nil || s.db == nil || at.IsZero() {
		return nil
	}
	if exchange == "" {
		exchange = "all"
	}
	_, err := s.db.ExecContext(ctx, `
DELETE FROM perp_depth_gaps
WHERE exchange = ? AND range_mpct = ? AND sampled_at_ms = ?
`, exchange, rangeMilliPct(rangePct), domain.BucketPerpDepthSampleTime(at).UnixMilli())
	return err
}

func splitVenues(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// PurgeOlderThan drops samples before cutoff.
func (s *SQLite) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.db == nil || cutoff.IsZero() {
		return 0, nil
	}
	ms := cutoff.UTC().UnixMilli()
	res, err := s.db.ExecContext(ctx, `DELETE FROM perp_depth_samples WHERE sampled_at_ms < ?`, ms)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	gres, err := s.db.ExecContext(ctx, `DELETE FROM perp_depth_gaps WHERE sampled_at_ms < ?`, ms)
	if err != nil {
		return int(n), err
	}
	gn, _ := gres.RowsAffected()
	n += gn
	bres, err := s.db.ExecContext(ctx, `DELETE FROM perp_book_bands WHERE sampled_at_ms < ?`, ms)
	if err != nil {
		return int(n), err
	}
	bn, _ := bres.RowsAffected()
	n += bn
	bgres, err := s.db.ExecContext(ctx, `DELETE FROM perp_book_gaps WHERE sampled_at_ms < ?`, ms)
	if err != nil {
		return int(n), err
	}
	bgn, _ := bgres.RowsAffected()
	return int(n + bgn), nil
}

// ListBucketTotals returns the newest minute's summed bid+ask inside each step.
func (s *SQLite) ListBucketTotals(ctx context.Context, exchange string, rangePct float64, from, to time.Time, step time.Duration) ([]domain.PerpDepthTotalPoint, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	if step < time.Minute {
		step = time.Minute
	}
	if to.IsZero() {
		to = time.Now().UTC().Add(time.Hour)
	}
	fromMs := int64(0)
	if !from.IsZero() {
		fromMs = from.UTC().UnixMilli()
	}
	stepMs := step.Milliseconds()
	rows, err := s.db.QueryContext(ctx, `
WITH mins AS (
	SELECT sampled_at_ms, SUM(bid + ask) AS total
	FROM perp_depth_samples
	WHERE exchange = ? AND range_mpct = ? AND sampled_at_ms >= ? AND sampled_at_ms <= ?
	GROUP BY sampled_at_ms
),
picked AS (
	SELECT (sampled_at_ms / ?) * ? AS bucket, total,
		ROW_NUMBER() OVER (PARTITION BY (sampled_at_ms / ?) * ? ORDER BY sampled_at_ms DESC) AS rn
	FROM mins
	WHERE total > 0
)
SELECT bucket, total FROM picked WHERE rn = 1 ORDER BY bucket ASC
`, exchange, rangeMilliPct(rangePct), fromMs, to.UTC().UnixMilli(), stepMs, stepMs, stepMs, stepMs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PerpDepthTotalPoint, 0, 64)
	for rows.Next() {
		var ms int64
		var total float64
		if err := rows.Scan(&ms, &total); err != nil {
			return nil, err
		}
		out = append(out, domain.PerpDepthTotalPoint{Time: time.UnixMilli(ms).UTC(), Total: total})
	}
	return out, rows.Err()
}

// InsertBookBands upserts one-minute perp band rows.
func (s *SQLite) InsertBookBands(ctx context.Context, rows []domain.PerpBookBandSample) (int, error) {
	if s == nil || s.db == nil || len(rows) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO perp_book_bands (exchange, symbol, range_mpct, sampled_at_ms, bid, ask, mid, cover_bid, cover_ask)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (exchange, symbol, range_mpct, sampled_at_ms) DO UPDATE SET
	bid=excluded.bid, ask=excluded.ask, mid=excluded.mid, cover_bid=excluded.cover_bid, cover_ask=excluded.cover_ask
`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	n := 0
	for _, r := range rows {
		if r.Symbol == "" || r.Time.IsZero() || r.Bid+r.Ask <= 0 {
			continue
		}
		ex := r.Exchange
		if ex == "" {
			ex = "all"
		}
		ms := domain.BucketPerpDepthSampleTime(r.Time).UnixMilli()
		if _, err := stmt.ExecContext(ctx, ex, r.Symbol, rangeMilliPct(r.RangePct), ms, r.Bid, r.Ask, r.Mid, r.CoverBid, r.CoverAsk); err != nil {
			return n, err
		}
		n++
	}
	if err := tx.Commit(); err != nil {
		return n, err
	}
	return n, nil
}

// ListBookBandBuckets returns the newest sample of each band inside each step.
func (s *SQLite) ListBookBandBuckets(ctx context.Context, exchange, symbol string, from, to time.Time, step time.Duration) ([]domain.PerpBookBandSample, error) {
	if s == nil || s.db == nil || symbol == "" {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	if step < time.Minute {
		step = time.Minute
	}
	if to.IsZero() {
		to = time.Now().UTC().Add(time.Hour)
	}
	fromMs := int64(0)
	if !from.IsZero() {
		fromMs = from.UTC().UnixMilli()
	}
	stepMs := step.Milliseconds()
	rows, err := s.db.QueryContext(ctx, `
WITH ranked AS (
	SELECT sampled_at_ms, range_mpct, bid, ask, mid, cover_bid, cover_ask,
		(sampled_at_ms / ?) * ? AS bucket,
		ROW_NUMBER() OVER (
			PARTITION BY range_mpct, (sampled_at_ms / ?) * ?
			ORDER BY sampled_at_ms DESC
		) AS rn
	FROM perp_book_bands
	WHERE exchange = ? AND symbol = ? AND sampled_at_ms >= ? AND sampled_at_ms <= ?
)
SELECT bucket, range_mpct, bid, ask, mid, cover_bid, cover_ask FROM ranked WHERE rn = 1 ORDER BY bucket ASC, range_mpct ASC
`, stepMs, stepMs, stepMs, stepMs, exchange, symbol, fromMs, to.UTC().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PerpBookBandSample, 0, 64)
	for rows.Next() {
		var rec domain.PerpBookBandSample
		var ms int64
		var mpct int
		if err := rows.Scan(&ms, &mpct, &rec.Bid, &rec.Ask, &rec.Mid, &rec.CoverBid, &rec.CoverAsk); err != nil {
			return nil, err
		}
		rec.Time = time.UnixMilli(ms).UTC()
		rec.Exchange = exchange
		rec.Symbol = symbol
		rec.RangePct = float64(mpct) / 100
		out = append(out, rec)
	}
	return out, rows.Err()
}

// InsertBookGap records why a coin's minute has no combined book.
func (s *SQLite) InsertBookGap(ctx context.Context, gap domain.PerpBookGap) error {
	if s == nil || s.db == nil || gap.Time.IsZero() || gap.Symbol == "" || (len(gap.Missing) == 0 && len(gap.Stale) == 0) {
		return nil
	}
	ms := domain.BucketPerpDepthSampleTime(gap.Time).UnixMilli()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO perp_book_gaps (symbol, sampled_at_ms, missing, stale)
VALUES (?, ?, ?, ?)
ON CONFLICT (symbol, sampled_at_ms) DO UPDATE SET
	missing=excluded.missing, stale=excluded.stale
`, gap.Symbol, ms, strings.Join(gap.Missing, ","), strings.Join(gap.Stale, ","))
	return err
}

// ListBookGaps returns gap minutes for one coin, oldest first.
func (s *SQLite) ListBookGaps(ctx context.Context, symbol string, from, to time.Time) ([]domain.PerpBookGap, error) {
	if s == nil || s.db == nil || symbol == "" {
		return nil, nil
	}
	if to.IsZero() {
		to = time.Now().UTC().Add(time.Hour)
	}
	fromMs := int64(0)
	if !from.IsZero() {
		fromMs = from.UTC().UnixMilli()
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT sampled_at_ms, missing, stale
FROM perp_book_gaps
WHERE symbol = ? AND sampled_at_ms >= ? AND sampled_at_ms <= ?
ORDER BY sampled_at_ms ASC
`, symbol, fromMs, to.UTC().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.PerpBookGap, 0)
	for rows.Next() {
		var ms int64
		var missing, stale string
		if err := rows.Scan(&ms, &missing, &stale); err != nil {
			return nil, err
		}
		out = append(out, domain.PerpBookGap{
			Time: time.UnixMilli(ms).UTC(), Symbol: symbol,
			Missing: splitVenues(missing), Stale: splitVenues(stale),
		})
	}
	return out, rows.Err()
}

// ClearBookGap drops a coin gap once that minute was stored.
func (s *SQLite) ClearBookGap(ctx context.Context, symbol string, at time.Time) error {
	if s == nil || s.db == nil || symbol == "" || at.IsZero() {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM perp_book_gaps WHERE symbol = ? AND sampled_at_ms = ?`, symbol, domain.BucketPerpDepthSampleTime(at).UnixMilli())
	return err
}
