package futuresstore

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

// SQLite persists futures snapshots and liquidation events.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the futures history database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("futures sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create futures db dir: %w", err)
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
CREATE TABLE IF NOT EXISTS futures_snapshots (
	metric          TEXT NOT NULL,
	exchange        TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	sampled_at_ms   INTEGER NOT NULL,
	predicted       INTEGER NOT NULL DEFAULT 0,
	contracts       REAL NOT NULL DEFAULT 0,
	value           REAL NOT NULL DEFAULT 0,
	funding_rate    REAL NOT NULL DEFAULT 0,
	interval_hours  INTEGER NOT NULL DEFAULT 0,
	next_funding_ms INTEGER NOT NULL DEFAULT 0,
	long_share      REAL NOT NULL DEFAULT 0,
	short_share     REAL NOT NULL DEFAULT 0,
	ratio           REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (metric, exchange, symbol, sampled_at_ms, predicted)
);
CREATE INDEX IF NOT EXISTS idx_futures_snapshots_lookup
	ON futures_snapshots(metric, symbol, sampled_at_ms);

CREATE TABLE IF NOT EXISTS futures_liquidations (
	exchange    TEXT NOT NULL,
	symbol      TEXT NOT NULL,
	side        TEXT NOT NULL,
	time_ms     INTEGER NOT NULL,
	price       REAL NOT NULL,
	quantity    REAL NOT NULL,
	notional    REAL NOT NULL,
	PRIMARY KEY (exchange, symbol, side, time_ms, price, quantity)
);
CREATE INDEX IF NOT EXISTS idx_futures_liq_lookup
	ON futures_liquidations(symbol, time_ms);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("futures schema: %w", err)
	}
	v, err := sqliteutil.UserVersion(s.db)
	if err != nil {
		return err
	}
	if v < 1 {
		if err := sqliteutil.SetUserVersion(s.db, 1); err != nil {
			return err
		}
		v = 1
	}
	if v < 2 {
		const taker = `
CREATE TABLE IF NOT EXISTS taker_buckets (
	exchange      TEXT NOT NULL,
	symbol        TEXT NOT NULL,
	start_ms      INTEGER NOT NULL,
	buy_notional  REAL NOT NULL DEFAULT 0,
	sell_notional REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (exchange, symbol, start_ms)
);
CREATE INDEX IF NOT EXISTS idx_taker_buckets_lookup
	ON taker_buckets(exchange, symbol, start_ms);
`
		if _, err := s.db.Exec(taker); err != nil {
			return fmt.Errorf("taker bucket schema: %w", err)
		}
		if err := sqliteutil.SetUserVersion(s.db, 2); err != nil {
			return err
		}
		v = 2
	}
	if v < 3 {
		const cov = `
CREATE TABLE IF NOT EXISTS liquidation_coverage (
	exchange       TEXT NOT NULL,
	symbol         TEXT NOT NULL DEFAULT '',
	first_watch_ms INTEGER NOT NULL,
	live_ms        INTEGER NOT NULL,
	PRIMARY KEY (exchange, symbol)
);`
		if _, err := s.db.Exec(cov); err != nil {
			return fmt.Errorf("liquidation coverage schema: %w", err)
		}
		if err := sqliteutil.SetUserVersion(s.db, 3); err != nil {
			return err
		}
		v = 3
	}
	if v < 4 {
		for _, stmt := range []string{
			`ALTER TABLE liquidation_coverage ADD COLUMN last_event_ms INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE liquidation_coverage ADD COLUMN last_seen_ms INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE liquidation_coverage ADD COLUMN last_saved_ms INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE liquidation_coverage ADD COLUMN gaps_json TEXT NOT NULL DEFAULT ''`,
		} {
			if err := sqliteutil.ExecAllowExists(s.db, stmt); err != nil {
				return err
			}
		}
		if err := sqliteutil.SetUserVersion(s.db, 4); err != nil {
			return err
		}
	}
	return nil
}

// InsertSnapshot stores one sample. Duplicate keys are ignored.
func (s *SQLite) InsertSnapshot(ctx context.Context, rec domain.FuturesSnapshot) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("futures store is nil")
	}
	rec.Symbol = domain.NormalizeLiquidationSymbol(rec.Symbol)
	if rec.Symbol == "" || rec.Metric == "" || rec.Exchange == "" || rec.SampledAt.IsZero() {
		return false, fmt.Errorf("%w: incomplete futures snapshot", domain.ErrInvalidArgument)
	}
	pred := 0
	if rec.Predicted {
		pred = 1
	}
	nextMS := int64(0)
	if !rec.NextFunding.IsZero() {
		nextMS = rec.NextFunding.UTC().UnixMilli()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO futures_snapshots (
	metric, exchange, symbol, sampled_at_ms, predicted,
	contracts, value, funding_rate, interval_hours, next_funding_ms,
	long_share, short_share, ratio
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.Metric, string(rec.Exchange), rec.Symbol, rec.SampledAt.UTC().UnixMilli(), pred,
		rec.Contracts, rec.Value, rec.FundingRate, rec.IntervalHours, nextMS,
		rec.LongShare, rec.ShortShare, rec.Ratio,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// InsertLiquidation stores one event. Duplicate keys are ignored.
func (s *SQLite) InsertLiquidation(ctx context.Context, e domain.LiquidationEvent) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("futures store is nil")
	}
	e.Symbol = domain.NormalizeLiquidationSymbol(e.Symbol)
	if e.Symbol == "" || e.Exchange == "" || e.Time.IsZero() || e.Notional <= 0 {
		return false, fmt.Errorf("%w: incomplete liquidation", domain.ErrInvalidArgument)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO futures_liquidations (
	exchange, symbol, side, time_ms, price, quantity, notional
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(e.Exchange), e.Symbol, e.Side, e.Time.UTC().UnixMilli(), e.Price, e.Quantity, e.Notional,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ListSnapshots returns samples newest first.
func (s *SQLite) ListSnapshots(ctx context.Context, q domain.FuturesHistoryQuery) ([]domain.FuturesSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	q.Symbol = domain.NormalizeLiquidationSymbol(q.Symbol)
	limit := domain.ClampFuturesHistoryLimit(q.Limit)
	var b strings.Builder
	args := []any{q.Metric, q.Symbol}
	b.WriteString(`SELECT metric, exchange, symbol, sampled_at_ms, predicted,
		contracts, value, funding_rate, interval_hours, next_funding_ms,
		long_share, short_share, ratio
		FROM futures_snapshots WHERE metric = ? AND symbol = ?`)
	if q.Exchange != "" && q.Exchange != "all" {
		b.WriteString(` AND exchange = ?`)
		args = append(args, q.Exchange)
	}
	if !q.From.IsZero() {
		b.WriteString(` AND sampled_at_ms >= ?`)
		args = append(args, q.From.UTC().UnixMilli())
	}
	if !q.To.IsZero() {
		b.WriteString(` AND sampled_at_ms <= ?`)
		args = append(args, q.To.UTC().UnixMilli())
	}
	b.WriteString(` ORDER BY sampled_at_ms DESC LIMIT ?`)
	args = append(args, limit)

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FuturesSnapshot
	for rows.Next() {
		var rec domain.FuturesSnapshot
		var ex string
		var ms, nextMS int64
		var pred int
		if err := rows.Scan(&rec.Metric, &ex, &rec.Symbol, &ms, &pred,
			&rec.Contracts, &rec.Value, &rec.FundingRate, &rec.IntervalHours, &nextMS,
			&rec.LongShare, &rec.ShortShare, &rec.Ratio); err != nil {
			return nil, err
		}
		rec.Exchange = domain.Exchange(ex)
		rec.SampledAt = time.UnixMilli(ms).UTC()
		rec.Predicted = pred == 1
		if nextMS > 0 {
			rec.NextFunding = time.UnixMilli(nextMS).UTC()
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ListLiquidations returns events newest first.
func (s *SQLite) ListLiquidations(ctx context.Context, exchange, symbol string, from, to time.Time, limit int) ([]domain.LiquidationEvent, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	limit = domain.ClampFuturesHistoryLimit(limit)
	var b strings.Builder
	args := []any{symbol}
	b.WriteString(`SELECT exchange, symbol, side, time_ms, price, quantity, notional
		FROM futures_liquidations WHERE symbol = ?`)
	if exchange != "" && exchange != "all" {
		b.WriteString(` AND exchange = ?`)
		args = append(args, exchange)
	}
	if !from.IsZero() {
		b.WriteString(` AND time_ms >= ?`)
		args = append(args, from.UTC().UnixMilli())
	}
	if !to.IsZero() {
		b.WriteString(` AND time_ms <= ?`)
		args = append(args, to.UTC().UnixMilli())
	}
	b.WriteString(` ORDER BY time_ms DESC LIMIT ?`)
	args = append(args, limit)

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LiquidationEvent
	for rows.Next() {
		var e domain.LiquidationEvent
		var ex string
		var ms int64
		if err := rows.Scan(&ex, &e.Symbol, &e.Side, &ms, &e.Price, &e.Quantity, &e.Notional); err != nil {
			return nil, err
		}
		e.Exchange = domain.Exchange(ex)
		e.Time = time.UnixMilli(ms).UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListLiquidationsSince returns every stored print at or after from, oldest first.
// limit <= 0 means no cap (pages through the 24h restore set).
func (s *SQLite) ListLiquidationsSince(ctx context.Context, from time.Time, limit int) ([]domain.LiquidationEvent, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	const page = 5000
	var out []domain.LiquidationEvent
	for {
		want := page
		if limit > 0 {
			left := limit - len(out)
			if left <= 0 {
				break
			}
			if left < want {
				want = left
			}
		}
		batch, err := s.listLiquidationsPage(ctx, from, len(out), want)
		if err != nil {
			return out, err
		}
		out = append(out, batch...)
		if len(batch) < want {
			break
		}
	}
	return out, nil
}

func (s *SQLite) listLiquidationsPage(ctx context.Context, from time.Time, offset, limit int) ([]domain.LiquidationEvent, error) {
	args := []any{}
	q := `SELECT exchange, symbol, side, time_ms, price, quantity, notional
		FROM futures_liquidations`
	if !from.IsZero() {
		q += ` WHERE time_ms >= ?`
		args = append(args, from.UTC().UnixMilli())
	}
	q += ` ORDER BY time_ms ASC, exchange ASC, symbol ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LiquidationEvent
	for rows.Next() {
		var e domain.LiquidationEvent
		var ex string
		var ms int64
		if err := rows.Scan(&ex, &e.Symbol, &e.Side, &ms, &e.Price, &e.Quantity, &e.Notional); err != nil {
			return nil, err
		}
		e.Exchange = domain.Exchange(ex)
		e.Time = time.UnixMilli(ms).UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpsertLiquidationCoverage stores venue or per-pair live coverage.
func (s *SQLite) UpsertLiquidationCoverage(ctx context.Context, rows []domain.LiquidationCoverage) error {
	if s == nil || s.db == nil || len(rows) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO liquidation_coverage (exchange, symbol, first_watch_ms, live_ms, last_event_ms, last_seen_ms, last_saved_ms, gaps_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(exchange, symbol) DO UPDATE SET
	first_watch_ms = MIN(first_watch_ms, excluded.first_watch_ms),
	live_ms = MAX(live_ms, excluded.live_ms),
	last_event_ms = MAX(last_event_ms, excluded.last_event_ms),
	last_seen_ms = MAX(last_seen_ms, excluded.last_seen_ms),
	last_saved_ms = excluded.last_saved_ms,
	gaps_json = excluded.gaps_json`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, row := range rows {
		if row.Exchange == "" || row.FirstWatch.IsZero() {
			continue
		}
		sym := domain.NormalizeLiquidationSymbol(row.Symbol)
		gaps, err := marshalLiqGaps(row.Gaps)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := stmt.ExecContext(ctx, string(row.Exchange), sym, row.FirstWatch.UTC().UnixMilli(), row.Live.Milliseconds(),
			unixMS(row.LastEvent), unixMS(row.LastSeen), unixMS(row.LastSaved), gaps); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ListLiquidationCoverage returns every stored coverage row.
func (s *SQLite) ListLiquidationCoverage(ctx context.Context) ([]domain.LiquidationCoverage, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `
SELECT exchange, symbol, first_watch_ms, live_ms, last_event_ms, last_seen_ms, last_saved_ms, gaps_json
FROM liquidation_coverage`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LiquidationCoverage
	for rows.Next() {
		var row domain.LiquidationCoverage
		var ex string
		var firstMS, liveMS, eventMS, seenMS, savedMS int64
		var gaps string
		if err := rows.Scan(&ex, &row.Symbol, &firstMS, &liveMS, &eventMS, &seenMS, &savedMS, &gaps); err != nil {
			return nil, err
		}
		row.Exchange = domain.Exchange(ex)
		row.FirstWatch = time.UnixMilli(firstMS).UTC()
		row.Live = time.Duration(liveMS) * time.Millisecond
		row.LastEvent = fromUnixMS(eventMS)
		row.LastSeen = fromUnixMS(seenMS)
		row.LastSaved = fromUnixMS(savedMS)
		row.Gaps = unmarshalLiqGaps(gaps)
		out = append(out, row)
	}
	return out, rows.Err()
}

// PurgeOlderThan deletes samples and events older than cutoff.
func (s *SQLite) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int, int, error) {
	if s == nil || s.db == nil || cutoff.IsZero() {
		return 0, 0, nil
	}
	ms := cutoff.UTC().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	r1, err := s.db.ExecContext(ctx, `DELETE FROM futures_snapshots WHERE sampled_at_ms < ?`, ms)
	if err != nil {
		return 0, 0, err
	}
	r2, err := s.db.ExecContext(ctx, `DELETE FROM futures_liquidations WHERE time_ms < ?`, ms)
	if err != nil {
		return 0, 0, err
	}
	n1, _ := r1.RowsAffected()
	n2, _ := r2.RowsAffected()
	return int(n1), int(n2), nil
}

type liqGapRow struct {
	From    int64 `json:"from"`
	To      int64 `json:"to,omitempty"`
	Seconds int64 `json:"seconds,omitempty"`
}

func unixMS(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UTC().UnixMilli()
}

func fromUnixMS(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func marshalLiqGaps(in []domain.LiquidationGap) (string, error) {
	if len(in) == 0 {
		return "", nil
	}
	rows := make([]liqGapRow, 0, len(in))
	for _, g := range in {
		rows = append(rows, liqGapRow{From: unixMS(g.From), To: unixMS(g.To), Seconds: g.Seconds})
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalLiqGaps(raw string) []domain.LiquidationGap {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var rows []liqGapRow
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil
	}
	out := make([]domain.LiquidationGap, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.LiquidationGap{
			From: fromUnixMS(r.From), To: fromUnixMS(r.To), Seconds: r.Seconds,
		})
	}
	return out
}

// UpsertTakerBuckets inserts or replaces buy/sell bars.
func (s *SQLite) UpsertTakerBuckets(ctx context.Context, recs []domain.TakerBucket) (int, error) {
	if s == nil || s.db == nil || len(recs) == 0 {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO taker_buckets (exchange, symbol, start_ms, buy_notional, sell_notional)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(exchange, symbol, start_ms) DO UPDATE SET
	buy_notional = excluded.buy_notional,
	sell_notional = excluded.sell_notional`)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	defer stmt.Close()
	n := 0
	for _, rec := range recs {
		rec.Symbol = domain.NormalizeLiquidationSymbol(rec.Symbol)
		if rec.Symbol == "" || rec.Exchange == "" || rec.Start.IsZero() {
			continue
		}
		if _, err := stmt.ExecContext(ctx, string(rec.Exchange), rec.Symbol, rec.Start.UTC().UnixMilli(), rec.BuyNotional, rec.SellNotional); err != nil {
			_ = tx.Rollback()
			return n, err
		}
		n++
	}
	if err := tx.Commit(); err != nil {
		return n, err
	}
	return n, nil
}

// ListTakerBuckets returns bars oldest first.
func (s *SQLite) ListTakerBuckets(ctx context.Context, exchange, symbol string, from, to time.Time) ([]domain.TakerBucket, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	ex := domain.ParseExchange(exchange)
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if symbol == "" || ex == "" {
		return nil, fmt.Errorf("%w: exchange and symbol are required", domain.ErrInvalidArgument)
	}
	args := []any{string(ex), symbol}
	q := `SELECT exchange, symbol, start_ms, buy_notional, sell_notional
		FROM taker_buckets WHERE exchange = ? AND symbol = ?`
	if !from.IsZero() {
		q += ` AND start_ms >= ?`
		args = append(args, from.UTC().UnixMilli())
	}
	if !to.IsZero() {
		q += ` AND start_ms <= ?`
		args = append(args, to.UTC().UnixMilli())
	}
	q += ` ORDER BY start_ms ASC`
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TakerBucket
	for rows.Next() {
		var rec domain.TakerBucket
		var exs string
		var ms int64
		if err := rows.Scan(&exs, &rec.Symbol, &ms, &rec.BuyNotional, &rec.SellNotional); err != nil {
			return nil, err
		}
		rec.Exchange = domain.Exchange(exs)
		rec.Start = time.UnixMilli(ms).UTC()
		out = append(out, rec)
	}
	return out, rows.Err()
}

// PurgeTakerBuckets deletes bars older than cutoff.
func (s *SQLite) PurgeTakerBuckets(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.db == nil || cutoff.IsZero() {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM taker_buckets WHERE start_ms < ?`, cutoff.UTC().UnixMilli())
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
