package fundingarbstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/sqliteutil"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists funding-arb watches and signals.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the funding-arb watch database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("funding-arb sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create funding-arb db dir: %w", err)
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
CREATE TABLE IF NOT EXISTS funding_arb_watches (
	id               TEXT PRIMARY KEY NOT NULL,
	client_id        TEXT NOT NULL,
	symbol           TEXT NOT NULL,
	notional         REAL NOT NULL,
	hold_hours       REAL NOT NULL,
	min_profit       REAL NOT NULL,
	fee_binance_pct  REAL NOT NULL DEFAULT 0,
	fee_bybit_pct    REAL NOT NULL DEFAULT 0,
	status           TEXT NOT NULL,
	armed            INTEGER NOT NULL DEFAULT 1,
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_fa_watches_client ON funding_arb_watches(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_fa_watches_active ON funding_arb_watches(status) WHERE status = 'active';

CREATE TABLE IF NOT EXISTS funding_arb_signals (
	id              TEXT PRIMARY KEY NOT NULL,
	watch_id        TEXT NOT NULL,
	client_id       TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	long_exchange   TEXT NOT NULL,
	short_exchange  TEXT NOT NULL,
	net_after_fees  REAL NOT NULL,
	min_profit      REAL NOT NULL,
	status          TEXT NOT NULL,
	opened_at       TEXT NOT NULL,
	last_seen_at    TEXT NOT NULL,
	closed_at       TEXT,
	FOREIGN KEY (watch_id) REFERENCES funding_arb_watches(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fa_sig_client ON funding_arb_signals(client_id, opened_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_fa_sig_open ON funding_arb_signals(watch_id) WHERE status = 'open';
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
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
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE funding_arb_watches ADD COLUMN quote TEXT NOT NULL DEFAULT 'USDT'`); err != nil {
			return err
		}
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE funding_arb_watches ADD COLUMN symbol_limit INTEGER NOT NULL DEFAULT 15`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`DROP INDEX IF EXISTS idx_fa_sig_open`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_fa_sig_open ON funding_arb_signals(watch_id, symbol) WHERE status = 'open'`); err != nil {
			return err
		}
		if err := sqliteutil.SetUserVersion(s.db, 2); err != nil {
			return err
		}
		v = 2
	}
	if v < 3 {
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE funding_arb_watches ADD COLUMN expires_at TEXT`); err != nil {
			return err
		}
		if err := sqliteutil.SetUserVersion(s.db, 3); err != nil {
			return err
		}
	}
	return nil
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t.UTC()
}

const watchCols = `id, client_id, symbol, notional, hold_hours, min_profit, fee_binance_pct, fee_bybit_pct, quote, symbol_limit, status, armed, expires_at, created_at, updated_at`

type scannable interface {
	Scan(dest ...any) error
}

func scanWatch(row scannable) (*domain.FundingArbWatch, error) {
	var w domain.FundingArbWatch
	var st, cAt, uAt string
	var armed int
	var exp sql.NullString
	if err := row.Scan(&w.ID, &w.ClientID, &w.Symbol, &w.Notional, &w.HoldHours, &w.MinProfit,
		&w.FeeBinancePct, &w.FeeBybitPct, &w.Quote, &w.SymbolLimit, &st, &armed, &exp, &cAt, &uAt); err != nil {
		return nil, err
	}
	w.Status = domain.FundingArbWatchStatus(st)
	w.Armed = armed != 0
	if exp.Valid && exp.String != "" {
		t := parseTime(exp.String)
		w.ExpiresAt = &t
	}
	w.CreatedAt = parseTime(cAt)
	w.UpdatedAt = parseTime(uAt)
	return &w, nil
}

// CreateWatch inserts a watch.
func (s *SQLite) CreateWatch(ctx context.Context, w domain.FundingArbWatch) (*domain.FundingArbWatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	armed := 0
	if w.Armed {
		armed = 1
	}
	if w.Quote == "" {
		w.Quote = "USDT"
	}
	if w.SymbolLimit <= 0 {
		w.SymbolLimit = domain.FundingArbScanDefault
	}
	var exp any
	if w.ExpiresAt != nil && !w.ExpiresAt.IsZero() {
		exp = w.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO funding_arb_watches (`+watchCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, w.ID, w.ClientID, w.Symbol, w.Notional, w.HoldHours, w.MinProfit,
		w.FeeBinancePct, w.FeeBybitPct, w.Quote, w.SymbolLimit, string(w.Status), armed, exp,
		w.CreatedAt.UTC().Format(time.RFC3339Nano), w.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.getWatchUnlocked(ctx, w.ClientID, w.ID)
}

func (s *SQLite) getWatchUnlocked(ctx context.Context, clientID, id string) (*domain.FundingArbWatch, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+watchCols+` FROM funding_arb_watches WHERE id = ? AND client_id = ?`, id, clientID)
	w, err := scanWatch(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return w, err
}

// GetWatch returns one watch.
func (s *SQLite) GetWatch(ctx context.Context, clientID, id string) (*domain.FundingArbWatch, error) {
	return s.getWatchUnlocked(ctx, clientID, id)
}

func listWatchesQuery(ctx context.Context, db *sql.DB, q string, args ...any) ([]domain.FundingArbWatch, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FundingArbWatch{}
	for rows.Next() {
		w, err := scanWatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

// ListWatches lists watches for a client.
func (s *SQLite) ListWatches(ctx context.Context, clientID string) ([]domain.FundingArbWatch, error) {
	return listWatchesQuery(ctx, s.db, `SELECT `+watchCols+` FROM funding_arb_watches WHERE client_id = ? ORDER BY created_at DESC`, clientID)
}

// ListActiveWatches returns all active watches.
func (s *SQLite) ListActiveWatches(ctx context.Context) ([]domain.FundingArbWatch, error) {
	return listWatchesQuery(ctx, s.db, `SELECT `+watchCols+` FROM funding_arb_watches WHERE status = 'active' ORDER BY created_at ASC`)
}

// DeleteWatch removes a watch and cascaded signals.
func (s *SQLite) DeleteWatch(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM funding_arb_watches WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CountWatches counts watches for a client.
func (s *SQLite) CountWatches(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_arb_watches WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// UpdateWatch writes mutable watch fields.
func (s *SQLite) UpdateWatch(ctx context.Context, w domain.FundingArbWatch) (*domain.FundingArbWatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	armed := 0
	if w.Armed {
		armed = 1
	}
	if w.Quote == "" {
		w.Quote = "USDT"
	}
	if w.SymbolLimit <= 0 {
		w.SymbolLimit = domain.FundingArbScanDefault
	}
	var exp any
	if w.ExpiresAt != nil && !w.ExpiresAt.IsZero() {
		exp = w.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE funding_arb_watches SET
			notional = ?, hold_hours = ?, min_profit = ?,
			fee_binance_pct = ?, fee_bybit_pct = ?,
			quote = ?, symbol_limit = ?, status = ?, armed = ?, expires_at = ?, updated_at = ?
		WHERE id = ? AND client_id = ?
	`, w.Notional, w.HoldHours, w.MinProfit, w.FeeBinancePct, w.FeeBybitPct,
		w.Quote, w.SymbolLimit, string(w.Status), armed, exp,
		w.UpdatedAt.UTC().Format(time.RFC3339Nano), w.ID, w.ClientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getWatchUnlocked(ctx, w.ClientID, w.ID)
}

// SetWatchArmed updates the re-arm flag.
func (s *SQLite) SetWatchArmed(ctx context.Context, id string, armed bool, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := 0
	if armed {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE funding_arb_watches SET armed = ?, updated_at = ? WHERE id = ?`,
		v, at.UTC().Format(time.RFC3339Nano), id)
	return err
}

const sigCols = `id, watch_id, client_id, symbol, long_exchange, short_exchange, net_after_fees, min_profit, status, opened_at, last_seen_at, closed_at`

func scanSignal(row scannable) (*domain.FundingArbSignal, error) {
	var s domain.FundingArbSignal
	var longEx, shortEx, st, opened, last string
	var closed sql.NullString
	if err := row.Scan(&s.ID, &s.WatchID, &s.ClientID, &s.Symbol, &longEx, &shortEx,
		&s.NetAfterFees, &s.MinProfit, &st, &opened, &last, &closed); err != nil {
		return nil, err
	}
	s.LongExchange = domain.Exchange(longEx)
	s.ShortExchange = domain.Exchange(shortEx)
	s.Status = domain.FundingArbSignalStatus(st)
	s.OpenedAt = parseTime(opened)
	s.LastSeenAt = parseTime(last)
	if closed.Valid && closed.String != "" {
		t := parseTime(closed.String)
		s.ClosedAt = &t
	}
	return &s, nil
}

// GetOpenSignal returns the open signal for a watch+symbol or ErrNotFound.
func (s *SQLite) GetOpenSignal(ctx context.Context, watchID, symbol string) (*domain.FundingArbSignal, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+sigCols+` FROM funding_arb_signals WHERE watch_id = ? AND symbol = ? AND status = 'open'`, watchID, symbol)
	sig, err := scanSignal(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return sig, err
}

// ListOpenSignals returns every open signal for a watch.
func (s *SQLite) ListOpenSignals(ctx context.Context, watchID string) ([]domain.FundingArbSignal, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+sigCols+` FROM funding_arb_signals WHERE watch_id = ? AND status = 'open'`, watchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FundingArbSignal{}
	for rows.Next() {
		sig, err := scanSignal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sig)
	}
	return out, rows.Err()
}

// CreateSignal inserts an open signal.
func (s *SQLite) CreateSignal(ctx context.Context, sig domain.FundingArbSignal) (*domain.FundingArbSignal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO funding_arb_signals (`+sigCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sig.ID, sig.WatchID, sig.ClientID, sig.Symbol, string(sig.LongExchange), string(sig.ShortExchange),
		sig.NetAfterFees, sig.MinProfit, string(sig.Status),
		sig.OpenedAt.UTC().Format(time.RFC3339Nano), sig.LastSeenAt.UTC().Format(time.RFC3339Nano), nil)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+sigCols+` FROM funding_arb_signals WHERE id = ?`, sig.ID)
	return scanSignal(row)
}

// TouchSignal updates last net/time on an open signal.
func (s *SQLite) TouchSignal(ctx context.Context, id string, net float64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `UPDATE funding_arb_signals SET net_after_fees = ?, last_seen_at = ? WHERE id = ? AND status = 'open'`,
		net, at.UTC().Format(time.RFC3339Nano), id)
	return err
}

// CloseSignal marks an open signal closed.
func (s *SQLite) CloseSignal(ctx context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `UPDATE funding_arb_signals SET status = 'closed', closed_at = ? WHERE id = ? AND status = 'open'`,
		at.UTC().Format(time.RFC3339Nano), id)
	return err
}

// ListSignals lists signals for a client.
func (s *SQLite) ListSignals(ctx context.Context, clientID string, status domain.FundingArbSignalStatus, limit int) ([]domain.FundingArbSignal, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT ` + sigCols + ` FROM funding_arb_signals WHERE client_id = ?`
	args := []any{clientID}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, string(status))
	}
	q += ` ORDER BY opened_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FundingArbSignal{}
	for rows.Next() {
		sig, err := scanSignal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sig)
	}
	return out, rows.Err()
}

// PurgeClient deletes all watches (and signals) for a tenant.
func (s *SQLite) PurgeClient(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM funding_arb_watches WHERE client_id = ?`, clientID)
	return err
}
