package pricediffstore

import (
	"context"
	"database/sql"
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

// SQLite persists price-diff watches and opportunities.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the price-diff database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("pricediff sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create pricediff db dir: %w", err)
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
CREATE TABLE IF NOT EXISTS price_diff_watches (
	id                TEXT PRIMARY KEY NOT NULL,
	client_id         TEXT NOT NULL,
	symbol            TEXT NOT NULL,
	min_net_diff_pct  REAL NOT NULL,
	fee_binance_pct   REAL NOT NULL DEFAULT 0,
	fee_coinbase_pct  REAL NOT NULL DEFAULT 0,
	fee_bybit_pct     REAL NOT NULL DEFAULT 0,
	status            TEXT NOT NULL,
	created_at        TEXT NOT NULL,
	updated_at        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pd_watches_client ON price_diff_watches(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pd_watches_active ON price_diff_watches(status) WHERE status = 'active';

CREATE TABLE IF NOT EXISTS price_diff_opportunities (
	id               TEXT PRIMARY KEY NOT NULL,
	watch_id         TEXT NOT NULL,
	client_id        TEXT NOT NULL,
	symbol           TEXT NOT NULL,
	buy_exchange     TEXT NOT NULL,
	sell_exchange    TEXT NOT NULL,
	buy_price        REAL NOT NULL,
	sell_price       REAL NOT NULL,
	gross_diff_pct   REAL NOT NULL,
	net_diff_pct     REAL NOT NULL,
	min_net_diff_pct REAL NOT NULL,
	status           TEXT NOT NULL,
	opened_at        TEXT NOT NULL,
	last_seen_at     TEXT NOT NULL,
	closed_at        TEXT,
	FOREIGN KEY (watch_id) REFERENCES price_diff_watches(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pd_opp_client ON price_diff_opportunities(client_id, opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_pd_opp_open ON price_diff_opportunities(watch_id, status)
	WHERE status = 'open';
CREATE UNIQUE INDEX IF NOT EXISTS idx_pd_opp_open_route
	ON price_diff_opportunities(watch_id, buy_exchange, sell_exchange)
	WHERE status = 'open';
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	return sqliteutil.SetUserVersion(s.db, 1)
}

const watchCols = `id, client_id, symbol, min_net_diff_pct, fee_binance_pct, fee_coinbase_pct,
	fee_bybit_pct, status, created_at, updated_at`

type scannable interface {
	Scan(dest ...any) error
}

func scanWatch(row scannable) (*domain.PriceDiffWatch, error) {
	var w domain.PriceDiffWatch
	var st, cAt, uAt string
	if err := row.Scan(
		&w.ID, &w.ClientID, &w.Symbol, &w.MinNetDiffPct, &w.FeeBinancePct, &w.FeeCoinbasePct,
		&w.FeeBybitPct, &st, &cAt, &uAt,
	); err != nil {
		return nil, err
	}
	w.Status = domain.PriceDiffWatchStatus(st)
	w.CreatedAt = parseTime(cAt)
	w.UpdatedAt = parseTime(uAt)
	return &w, nil
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

// CreateWatch inserts a watch.
func (s *SQLite) CreateWatch(ctx context.Context, w domain.PriceDiffWatch) (*domain.PriceDiffWatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO price_diff_watches (`+watchCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, w.ID, w.ClientID, w.Symbol, w.MinNetDiffPct, w.FeeBinancePct, w.FeeCoinbasePct, w.FeeBybitPct,
		string(w.Status), w.CreatedAt.UTC().Format(time.RFC3339Nano), w.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.getWatchUnlocked(ctx, w.ClientID, w.ID)
}

func (s *SQLite) getWatchUnlocked(ctx context.Context, clientID, id string) (*domain.PriceDiffWatch, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+watchCols+` FROM price_diff_watches WHERE id = ? AND client_id = ?
	`, id, clientID)
	w, err := scanWatch(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return w, err
}

// GetWatch returns one watch.
func (s *SQLite) GetWatch(ctx context.Context, clientID, id string) (*domain.PriceDiffWatch, error) {
	return s.getWatchUnlocked(ctx, clientID, id)
}

// ListWatches lists watches for a client.
func (s *SQLite) ListWatches(ctx context.Context, clientID string) ([]domain.PriceDiffWatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+watchCols+` FROM price_diff_watches WHERE client_id = ? ORDER BY created_at DESC
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PriceDiffWatch
	for rows.Next() {
		w, err := scanWatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

// ListActiveWatches returns all active watches.
func (s *SQLite) ListActiveWatches(ctx context.Context) ([]domain.PriceDiffWatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+watchCols+` FROM price_diff_watches WHERE status = 'active' ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PriceDiffWatch
	for rows.Next() {
		w, err := scanWatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

// DeleteWatch removes a watch and cascaded opportunities.
func (s *SQLite) DeleteWatch(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM price_diff_watches WHERE id = ? AND client_id = ?`, id, clientID)
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
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM price_diff_watches WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

const oppCols = `id, watch_id, client_id, symbol, buy_exchange, sell_exchange, buy_price, sell_price,
	gross_diff_pct, net_diff_pct, min_net_diff_pct, status, opened_at, last_seen_at, closed_at`

func scanOpp(row scannable) (*domain.PriceDiffOpportunity, error) {
	var o domain.PriceDiffOpportunity
	var buyEx, sellEx, st, opened, last string
	var closed sql.NullString
	if err := row.Scan(
		&o.ID, &o.WatchID, &o.ClientID, &o.Symbol, &buyEx, &sellEx, &o.BuyPrice, &o.SellPrice,
		&o.GrossDiffPct, &o.NetDiffPct, &o.MinNetDiffPct, &st, &opened, &last, &closed,
	); err != nil {
		return nil, err
	}
	o.BuyExchange = domain.Exchange(buyEx)
	o.SellExchange = domain.Exchange(sellEx)
	o.Status = domain.PriceDiffOppStatus(st)
	o.OpenedAt = parseTime(opened)
	o.LastSeenAt = parseTime(last)
	if closed.Valid && closed.String != "" {
		t := parseTime(closed.String)
		o.ClosedAt = &t
	}
	return &o, nil
}

// GetOpenOpportunity returns the open opportunity for a route.
func (s *SQLite) GetOpenOpportunity(ctx context.Context, watchID string, buy, sell domain.Exchange) (*domain.PriceDiffOpportunity, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+oppCols+` FROM price_diff_opportunities
		WHERE watch_id = ? AND buy_exchange = ? AND sell_exchange = ? AND status = 'open'
	`, watchID, string(buy), string(sell))
	o, err := scanOpp(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return o, err
}

// CreateOpportunity inserts an open opportunity.
func (s *SQLite) CreateOpportunity(ctx context.Context, o domain.PriceDiffOpportunity) (*domain.PriceDiffOpportunity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO price_diff_opportunities (`+oppCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, o.ID, o.WatchID, o.ClientID, o.Symbol, string(o.BuyExchange), string(o.SellExchange),
		o.BuyPrice, o.SellPrice, o.GrossDiffPct, o.NetDiffPct, o.MinNetDiffPct, string(o.Status),
		o.OpenedAt.UTC().Format(time.RFC3339Nano), o.LastSeenAt.UTC().Format(time.RFC3339Nano), nullTime(o.ClosedAt))
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unique") {
			// Concurrent create — treat as race; return existing
			return s.GetOpenOpportunity(ctx, o.WatchID, o.BuyExchange, o.SellExchange)
		}
		return nil, err
	}
	return s.getOppByID(ctx, o.ID)
}

func (s *SQLite) getOppByID(ctx context.Context, id string) (*domain.PriceDiffOpportunity, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+oppCols+` FROM price_diff_opportunities WHERE id = ?`, id)
	o, err := scanOpp(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return o, err
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// TouchOpportunity updates live fields on an open opportunity.
func (s *SQLite) TouchOpportunity(ctx context.Context, id string, buyPrice, sellPrice, grossPct, netPct float64, at time.Time) (*domain.PriceDiffOpportunity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE price_diff_opportunities
		SET buy_price = ?, sell_price = ?, gross_diff_pct = ?, net_diff_pct = ?, last_seen_at = ?
		WHERE id = ? AND status = 'open'
	`, buyPrice, sellPrice, grossPct, netPct, at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getOppByID(ctx, id)
}

// CloseOpportunity marks an open opportunity closed.
func (s *SQLite) CloseOpportunity(ctx context.Context, id string, at time.Time) (*domain.PriceDiffOpportunity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE price_diff_opportunities
		SET status = 'closed', closed_at = ?, last_seen_at = ?
		WHERE id = ? AND status = 'open'
	`, at.UTC().Format(time.RFC3339Nano), at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getOppByID(ctx, id)
}

// ListOpenOpportunitiesForWatch lists open opps for a watch.
func (s *SQLite) ListOpenOpportunitiesForWatch(ctx context.Context, watchID string) ([]domain.PriceDiffOpportunity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+oppCols+` FROM price_diff_opportunities
		WHERE watch_id = ? AND status = 'open'
	`, watchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOpps(rows)
}

// ListOpportunities lists client opportunities with optional status filter.
func (s *SQLite) ListOpportunities(ctx context.Context, clientID string, status domain.PriceDiffOppStatus, limit, offset int) ([]domain.PriceDiffOpportunity, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var rows *sql.Rows
	var err error
	if status == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT `+oppCols+` FROM price_diff_opportunities
			WHERE client_id = ? ORDER BY opened_at DESC LIMIT ? OFFSET ?
		`, clientID, limit, offset)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT `+oppCols+` FROM price_diff_opportunities
			WHERE client_id = ? AND status = ? ORDER BY opened_at DESC LIMIT ? OFFSET ?
		`, clientID, string(status), limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOpps(rows)
}

// GetOpportunity returns one opportunity by id for the client.
func (s *SQLite) GetOpportunity(ctx context.Context, clientID, id string) (*domain.PriceDiffOpportunity, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+oppCols+` FROM price_diff_opportunities WHERE id = ? AND client_id = ?
	`, id, clientID)
	o, err := scanOpp(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return o, err
}

func scanOpps(rows *sql.Rows) ([]domain.PriceDiffOpportunity, error) {
	var out []domain.PriceDiffOpportunity
	for rows.Next() {
		o, err := scanOpp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}
