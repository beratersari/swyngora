package scannerstore

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

// SQLite persists scanner rules and results.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the scanner database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("scanner sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create scanner db dir: %w", err)
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

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS scanner_rules (
	id               TEXT PRIMARY KEY NOT NULL,
	client_id        TEXT NOT NULL,
	rule_type        TEXT NOT NULL,
	interval         TEXT NOT NULL,
	enabled          INTEGER NOT NULL DEFAULT 1,
	rsi_period       INTEGER NOT NULL DEFAULT 0,
	rsi_condition    TEXT NOT NULL DEFAULT '',
	rsi_threshold    REAL NOT NULL DEFAULT 0,
	ma_fast_period   INTEGER NOT NULL DEFAULT 0,
	ma_slow_period   INTEGER NOT NULL DEFAULT 0,
	ma_direction     TEXT NOT NULL DEFAULT '',
	volume_lookback  INTEGER NOT NULL DEFAULT 0,
	volume_min_ratio REAL NOT NULL DEFAULT 0,
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_scanner_rules_client ON scanner_rules(client_id);
CREATE INDEX IF NOT EXISTS idx_scanner_rules_enabled ON scanner_rules(enabled) WHERE enabled = 1;

CREATE TABLE IF NOT EXISTS scanner_results (
	id              TEXT PRIMARY KEY NOT NULL,
	client_id       TEXT NOT NULL,
	rule_id         TEXT NOT NULL,
	exchange        TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	rule_type       TEXT NOT NULL,
	interval        TEXT NOT NULL,
	market_data_key TEXT NOT NULL,
	matched_at      TEXT NOT NULL,
	summary         TEXT NOT NULL,
	metrics_json    TEXT NOT NULL DEFAULT '{}',
	UNIQUE(rule_id, exchange, symbol, market_data_key),
	FOREIGN KEY (rule_id) REFERENCES scanner_rules(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_scanner_results_client ON scanner_results(client_id, matched_at DESC);

CREATE TABLE IF NOT EXISTS scanner_backtests (
	id              TEXT PRIMARY KEY NOT NULL,
	client_id       TEXT NOT NULL,
	rule_id         TEXT NOT NULL,
	exchange        TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	interval        TEXT NOT NULL,
	range_start     TEXT NOT NULL,
	range_end       TEXT NOT NULL,
	status          TEXT NOT NULL,
	progress_pct    REAL NOT NULL DEFAULT 0,
	processed_bars  INTEGER NOT NULL DEFAULT 0,
	total_bars      INTEGER NOT NULL DEFAULT 0,
	signal_count    INTEGER NOT NULL DEFAULT 0,
	error_message   TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL,
	started_at      TEXT,
	finished_at     TEXT,
	FOREIGN KEY (rule_id) REFERENCES scanner_rules(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_scanner_backtests_client ON scanner_backtests(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scanner_backtests_pending ON scanner_backtests(status) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_scanner_backtests_fingerprint ON scanner_backtests(client_id, rule_id, exchange, symbol, range_start, range_end);

CREATE TABLE IF NOT EXISTS scanner_backtest_signals (
	id           TEXT PRIMARY KEY NOT NULL,
	backtest_id  TEXT NOT NULL,
	signal_at    TEXT NOT NULL,
	close_price  REAL NOT NULL,
	summary      TEXT NOT NULL,
	return_1d    REAL,
	return_5d    REAL,
	return_20d   REAL,
	metrics_json TEXT NOT NULL DEFAULT '{}',
	FOREIGN KEY (backtest_id) REFERENCES scanner_backtests(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_scanner_backtest_signals ON scanner_backtest_signals(backtest_id, signal_at ASC);
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
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE scanner_rules ADD COLUMN conditions TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE scanner_rules ADD COLUMN match_mode TEXT NOT NULL DEFAULT 'all'`); err != nil {
			return err
		}
		if err := sqliteutil.SetUserVersion(s.db, 2); err != nil {
			return err
		}
	}
	if v < 3 {
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE scanner_backtests ADD COLUMN rule_json TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
		if err := sqliteutil.SetUserVersion(s.db, 3); err != nil {
			return err
		}
	}
	return nil
}

// Path returns absolute DB path.
func (s *SQLite) Path() string { return s.path }

// Close closes the DB.
func (s *SQLite) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// CreateRule inserts a rule.
func (s *SQLite) CreateRule(ctx context.Context, r domain.ScannerRule) (*domain.ScannerRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = r.CreatedAt
	}
	en := 0
	if r.Enabled {
		en = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO scanner_rules (
			id, client_id, rule_type, interval, enabled,
			rsi_period, rsi_condition, rsi_threshold,
			ma_fast_period, ma_slow_period, ma_direction,
			volume_lookback, volume_min_ratio, conditions, match_mode,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.ClientID, string(r.Type), r.Interval, en,
		r.RSIPeriod, string(r.RSICondition), r.RSIThreshold,
		r.MAFastPeriod, r.MASlowPeriod, r.MADirection,
		r.VolumeLookback, r.VolumeMinRatio,
		encodeScannerConditions(r.Conditions), string(r.MatchMode),
		r.CreatedAt.UTC().Format(time.RFC3339Nano), r.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	cp := r
	return &cp, nil
}

// UpdateRule persists enable/disable and parameter edits.
func (s *SQLite) UpdateRule(ctx context.Context, r domain.ScannerRule) (*domain.ScannerRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = time.Now().UTC()
	}
	en := 0
	if r.Enabled {
		en = 1
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE scanner_rules SET
			rule_type = ?, interval = ?, enabled = ?,
			rsi_period = ?, rsi_condition = ?, rsi_threshold = ?,
			ma_fast_period = ?, ma_slow_period = ?, ma_direction = ?,
			volume_lookback = ?, volume_min_ratio = ?, conditions = ?, match_mode = ?,
			updated_at = ?
		WHERE client_id = ? AND id = ?
	`, string(r.Type), r.Interval, en,
		r.RSIPeriod, string(r.RSICondition), r.RSIThreshold,
		r.MAFastPeriod, r.MASlowPeriod, r.MADirection,
		r.VolumeLookback, r.VolumeMinRatio,
		encodeScannerConditions(r.Conditions), string(r.MatchMode),
		r.UpdatedAt.UTC().Format(time.RFC3339Nano), r.ClientID, r.ID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	cp := r
	return &cp, nil
}

// GetRule returns one rule or ErrNotFound.
func (s *SQLite) GetRule(ctx context.Context, clientID, id string) (*domain.ScannerRule, error) {
	row := s.db.QueryRowContext(ctx, ruleSelect+` WHERE client_id = ? AND id = ?`, clientID, id)
	r, err := scanRule(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return r, err
}

// ListRules lists rules for a client.
func (s *SQLite) ListRules(ctx context.Context, clientID string) ([]domain.ScannerRule, error) {
	rows, err := s.db.QueryContext(ctx, ruleSelect+` WHERE client_id = ? ORDER BY created_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

// ListEnabledRules returns all enabled rules.
func (s *SQLite) ListEnabledRules(ctx context.Context) ([]domain.ScannerRule, error) {
	rows, err := s.db.QueryContext(ctx, ruleSelect+` WHERE enabled = 1 ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

// DeleteRule removes a rule and cascades results.
func (s *SQLite) DeleteRule(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM scanner_rules WHERE client_id = ? AND id = ?`, clientID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CountRules counts rules for a client.
func (s *SQLite) CountRules(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scanner_rules WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// InsertResult inserts a result; returns inserted=false on unique conflict.
func (s *SQLite) InsertResult(ctx context.Context, res domain.ScannerResult) (*domain.ScannerResult, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if res.MatchedAt.IsZero() {
		res.MatchedAt = time.Now().UTC()
	}
	mj, err := json.Marshal(res.Metrics)
	if err != nil {
		return nil, false, err
	}
	if mj == nil {
		mj = []byte("{}")
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO scanner_results (
			id, client_id, rule_id, exchange, symbol, rule_type, interval,
			market_data_key, matched_at, summary, metrics_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, res.ID, res.ClientID, res.RuleID, string(res.Exchange), res.Symbol, string(res.RuleType), res.Interval,
		res.MarketDataKey, res.MatchedAt.UTC().Format(time.RFC3339Nano), res.Summary, string(mj))
	if err != nil {
		// UNIQUE violation → duplicate for same market data
		if isUniqueErr(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	cp := res
	return &cp, true, nil
}

// ListResults returns newest-first results for a client.
func (s *SQLite) ListResults(ctx context.Context, clientID string, limit, offset int) ([]domain.ScannerResult, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, rule_id, exchange, symbol, rule_type, interval,
			market_data_key, matched_at, summary, metrics_json
		FROM scanner_results WHERE client_id = ?
		ORDER BY matched_at DESC LIMIT ? OFFSET ?
	`, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.ScannerResult, 0)
	for rows.Next() {
		r, err := scanResult(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// CountResults counts results for a client.
func (s *SQLite) CountResults(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scanner_results WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

const backtestSelect = `
	SELECT id, client_id, rule_id, exchange, symbol, interval, range_start, range_end,
		status, progress_pct, processed_bars, total_bars, signal_count, error_message,
		created_at, started_at, finished_at, rule_json
	FROM scanner_backtests`

// CreateBacktest inserts a pending job.
func (s *SQLite) CreateBacktest(ctx context.Context, b domain.ScannerBacktest) (*domain.ScannerBacktest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}
	if b.Status == "" {
		b.Status = domain.BacktestPending
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO scanner_backtests (
			id, client_id, rule_id, exchange, symbol, interval, range_start, range_end,
			status, progress_pct, processed_bars, total_bars, signal_count, error_message,
			created_at, started_at, finished_at, rule_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, b.ID, b.ClientID, b.RuleID, string(b.Exchange), b.Symbol, b.Interval,
		b.RangeStart.UTC().Format(time.RFC3339Nano), b.RangeEnd.UTC().Format(time.RFC3339Nano),
		string(b.Status), b.ProgressPct, b.ProcessedBars, b.TotalBars, b.SignalCount, b.ErrorMessage,
		b.CreatedAt.UTC().Format(time.RFC3339Nano), nullTime(b.StartedAt), nullTime(b.FinishedAt), b.RuleJSON)
	if err != nil {
		return nil, err
	}
	cp := b
	return &cp, nil
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// GetBacktest returns one job for the client.
func (s *SQLite) GetBacktest(ctx context.Context, clientID, id string) (*domain.ScannerBacktest, error) {
	row := s.db.QueryRowContext(ctx, backtestSelect+` WHERE client_id = ? AND id = ?`, clientID, id)
	b, err := scanBacktest(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return b, err
}

// FindActiveBacktest finds pending/running/completed job with same params.
func (s *SQLite) FindActiveBacktest(ctx context.Context, clientID, ruleID string, exchange domain.Exchange, symbol string, rangeStart, rangeEnd time.Time) (*domain.ScannerBacktest, error) {
	row := s.db.QueryRowContext(ctx, backtestSelect+`
		WHERE client_id = ? AND rule_id = ? AND exchange = ? AND symbol = ?
		  AND range_start = ? AND range_end = ?
		  AND status IN ('pending', 'running', 'completed')
		ORDER BY created_at DESC LIMIT 1
	`, clientID, ruleID, string(exchange), symbol,
		rangeStart.UTC().Format(time.RFC3339Nano), rangeEnd.UTC().Format(time.RFC3339Nano))
	b, err := scanBacktest(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return b, err
}

// ListBacktests lists jobs for a client newest first.
func (s *SQLite) ListBacktests(ctx context.Context, clientID string, limit, offset int) ([]domain.ScannerBacktest, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, backtestSelect+`
		WHERE client_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBacktests(rows)
}

// CountBacktests counts jobs for a client.
func (s *SQLite) CountBacktests(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scanner_backtests WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// ListPendingBacktests returns pending jobs oldest first.
func (s *SQLite) ListPendingBacktests(ctx context.Context) ([]domain.ScannerBacktest, error) {
	rows, err := s.db.QueryContext(ctx, backtestSelect+` WHERE status = 'pending' ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBacktests(rows)
}

// ClaimBacktest transitions pending → running.
func (s *SQLite) ClaimBacktest(ctx context.Context, id string, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE scanner_backtests
		SET status = 'running', started_at = ?, progress_pct = 0
		WHERE id = ? AND status = 'pending'
	`, at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// UpdateBacktestProgress updates progress counters.
func (s *SQLite) UpdateBacktestProgress(ctx context.Context, id string, processed, total, signalCount int, progressPct float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		UPDATE scanner_backtests
		SET processed_bars = ?, total_bars = ?, signal_count = ?, progress_pct = ?
		WHERE id = ? AND status = 'running'
	`, processed, total, signalCount, progressPct, id)
	return err
}

// FinishBacktest sets a terminal status.
func (s *SQLite) FinishBacktest(ctx context.Context, id string, status domain.ScannerBacktestStatus, signalCount int, errMsg string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	pct := 100.0
	if status != domain.BacktestCompleted {
		// keep last progress for canceled/failed
		pct = -1
	}
	if pct < 0 {
		_, err := s.db.ExecContext(ctx, `
			UPDATE scanner_backtests
			SET status = ?, signal_count = ?, error_message = ?, finished_at = ?
			WHERE id = ? AND status IN ('pending', 'running')
		`, string(status), signalCount, errMsg, at.UTC().Format(time.RFC3339Nano), id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE scanner_backtests
		SET status = ?, signal_count = ?, error_message = ?, finished_at = ?, progress_pct = 100
		WHERE id = ? AND status IN ('pending', 'running')
	`, string(status), signalCount, errMsg, at.UTC().Format(time.RFC3339Nano), id)
	return err
}

// CancelBacktest cancels pending or running job.
func (s *SQLite) CancelBacktest(ctx context.Context, clientID, id string, at time.Time) (*domain.ScannerBacktest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE scanner_backtests
		SET status = 'canceled', finished_at = ?
		WHERE id = ? AND client_id = ? AND status IN ('pending', 'running')
	`, at.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	row := s.db.QueryRowContext(ctx, backtestSelect+` WHERE id = ? AND client_id = ?`, id, clientID)
	return scanBacktest(row)
}

// GetBacktestStatus returns status only.
func (s *SQLite) GetBacktestStatus(ctx context.Context, id string) (domain.ScannerBacktestStatus, error) {
	var st string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM scanner_backtests WHERE id = ?`, id).Scan(&st)
	if err == sql.ErrNoRows {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return domain.ScannerBacktestStatus(st), nil
}

// InsertBacktestSignal stores one signal row.
func (s *SQLite) InsertBacktestSignal(ctx context.Context, sig domain.ScannerBacktestSignal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mj, err := json.Marshal(sig.Metrics)
	if err != nil {
		return err
	}
	if mj == nil {
		mj = []byte("{}")
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO scanner_backtest_signals (
			id, backtest_id, signal_at, close_price, summary,
			return_1d, return_5d, return_20d, metrics_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sig.ID, sig.BacktestID, sig.SignalAt.UTC().Format(time.RFC3339Nano), sig.ClosePrice, sig.Summary,
		nullFloat(sig.Return1d), nullFloat(sig.Return5d), nullFloat(sig.Return20d), string(mj))
	return err
}

func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// ListBacktestSignals lists signals for a job.
func (s *SQLite) ListBacktestSignals(ctx context.Context, backtestID string, limit, offset int) ([]domain.ScannerBacktestSignal, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, backtest_id, signal_at, close_price, summary, return_1d, return_5d, return_20d, metrics_json
		FROM scanner_backtest_signals WHERE backtest_id = ?
		ORDER BY signal_at ASC LIMIT ? OFFSET ?
	`, backtestID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.ScannerBacktestSignal, 0)
	for rows.Next() {
		sig, err := scanBacktestSignal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sig)
	}
	return out, rows.Err()
}

// CountBacktestSignals counts signals for a job.
func (s *SQLite) CountBacktestSignals(ctx context.Context, backtestID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scanner_backtest_signals WHERE backtest_id = ?`, backtestID).Scan(&n)
	return n, err
}

// DeleteBacktest removes a backtest and cascaded signals for the owning client.
func (s *SQLite) DeleteBacktest(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM scanner_backtests WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// PurgeClient deletes rules, results, and backtests for clientID.
func (s *SQLite) PurgeClient(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// signals cascade with backtests if FK on; delete backtests first
	if _, err := s.db.ExecContext(ctx, `DELETE FROM scanner_backtests WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM scanner_results WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM scanner_rules WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	return nil
}

func scanBacktest(row scannable) (*domain.ScannerBacktest, error) {
	var b domain.ScannerBacktest
	var ex, st, rStart, rEnd, cAt string
	var started, finished sql.NullString
	if err := row.Scan(
		&b.ID, &b.ClientID, &b.RuleID, &ex, &b.Symbol, &b.Interval, &rStart, &rEnd,
		&st, &b.ProgressPct, &b.ProcessedBars, &b.TotalBars, &b.SignalCount, &b.ErrorMessage,
		&cAt, &started, &finished, &b.RuleJSON,
	); err != nil {
		return nil, err
	}
	b.Exchange = domain.Exchange(ex)
	b.Status = domain.ScannerBacktestStatus(st)
	b.RangeStart = parseTime(rStart)
	b.RangeEnd = parseTime(rEnd)
	b.CreatedAt = parseTime(cAt)
	if started.Valid && started.String != "" {
		t := parseTime(started.String)
		b.StartedAt = &t
	}
	if finished.Valid && finished.String != "" {
		t := parseTime(finished.String)
		b.FinishedAt = &t
	}
	return &b, nil
}

func scanBacktests(rows *sql.Rows) ([]domain.ScannerBacktest, error) {
	out := make([]domain.ScannerBacktest, 0)
	for rows.Next() {
		b, err := scanBacktest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func scanBacktestSignal(row scannable) (*domain.ScannerBacktestSignal, error) {
	var sig domain.ScannerBacktestSignal
	var sAt, metrics string
	var r1, r5, r20 sql.NullFloat64
	if err := row.Scan(
		&sig.ID, &sig.BacktestID, &sAt, &sig.ClosePrice, &sig.Summary, &r1, &r5, &r20, &metrics,
	); err != nil {
		return nil, err
	}
	sig.SignalAt = parseTime(sAt)
	if r1.Valid {
		v := r1.Float64
		sig.Return1d = &v
	}
	if r5.Valid {
		v := r5.Float64
		sig.Return5d = &v
	}
	if r20.Valid {
		v := r20.Float64
		sig.Return20d = &v
	}
	sig.Metrics = map[string]float64{}
	if metrics != "" && metrics != "{}" {
		_ = json.Unmarshal([]byte(metrics), &sig.Metrics)
	}
	return &sig, nil
}

func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsFold(msg, "unique") || containsFold(msg, "constraint")
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (indexFold(s, sub) >= 0)
}

func indexFold(s, sub string) int {
	// simple case-insensitive search
	ls, lsub := len(s), len(sub)
	for i := 0; i+lsub <= ls; i++ {
		ok := true
		for j := 0; j < lsub; j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

const ruleSelect = `
	SELECT id, client_id, rule_type, interval, enabled,
		rsi_period, rsi_condition, rsi_threshold,
		ma_fast_period, ma_slow_period, ma_direction,
		volume_lookback, volume_min_ratio, conditions, match_mode,
		created_at, updated_at
	FROM scanner_rules`

type scannable interface {
	Scan(dest ...any) error
}

func scanRule(row scannable) (*domain.ScannerRule, error) {
	var r domain.ScannerRule
	var typ, cond, dir, cAt, uAt, conds, mode string
	var en int
	if err := row.Scan(
		&r.ID, &r.ClientID, &typ, &r.Interval, &en,
		&r.RSIPeriod, &cond, &r.RSIThreshold,
		&r.MAFastPeriod, &r.MASlowPeriod, &dir,
		&r.VolumeLookback, &r.VolumeMinRatio, &conds, &mode, &cAt, &uAt,
	); err != nil {
		return nil, err
	}
	r.Type = domain.ScannerRuleType(typ)
	r.Enabled = en != 0
	r.RSICondition = domain.AlertCondition(cond)
	r.MADirection = dir
	r.Conditions = decodeScannerConditions(conds)
	r.MatchMode = domain.ScannerMatchMode(mode)
	if r.MatchMode == "" {
		r.MatchMode = domain.ScannerMatchAll
	}
	r.CreatedAt = parseTime(cAt)
	r.UpdatedAt = parseTime(uAt)
	return &r, nil
}

func encodeScannerConditions(in []domain.ScannerRuleType) string {
	if len(in) == 0 {
		return ""
	}
	parts := make([]string, 0, len(in))
	for _, c := range in {
		parts = append(parts, string(c))
	}
	return strings.Join(parts, ",")
}

func decodeScannerConditions(raw string) []domain.ScannerRuleType {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]domain.ScannerRuleType, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, domain.ScannerRuleType(p))
	}
	return out
}

func scanRules(rows *sql.Rows) ([]domain.ScannerRule, error) {
	out := make([]domain.ScannerRule, 0)
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func scanResult(row scannable) (*domain.ScannerResult, error) {
	var r domain.ScannerResult
	var ex, typ, mAt, metrics string
	if err := row.Scan(
		&r.ID, &r.ClientID, &r.RuleID, &ex, &r.Symbol, &typ, &r.Interval,
		&r.MarketDataKey, &mAt, &r.Summary, &metrics,
	); err != nil {
		return nil, err
	}
	r.Exchange = domain.Exchange(ex)
	r.RuleType = domain.ScannerRuleType(typ)
	r.MatchedAt = parseTime(mAt)
	r.Metrics = map[string]float64{}
	if metrics != "" && metrics != "{}" {
		_ = json.Unmarshal([]byte(metrics), &r.Metrics)
	}
	return &r, nil
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
