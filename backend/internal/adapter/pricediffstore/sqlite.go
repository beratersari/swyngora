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
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE price_diff_watches ADD COLUMN notional REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE price_diff_watches ADD COLUMN min_profit REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if err := sqliteutil.SetUserVersion(s.db, 2); err != nil {
			return err
		}
	}
	if v < 3 {
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE price_diff_watches ADD COLUMN min_duration_sec REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS price_diff_route_arms (
	watch_id      TEXT NOT NULL,
	buy_exchange  TEXT NOT NULL,
	sell_exchange TEXT NOT NULL,
	armed_at      TEXT NOT NULL,
	PRIMARY KEY (watch_id, buy_exchange, sell_exchange),
	FOREIGN KEY (watch_id) REFERENCES price_diff_watches(id) ON DELETE CASCADE
)`); err != nil {
			return err
		}
		if err := sqliteutil.SetUserVersion(s.db, 3); err != nil {
			return err
		}
	}
	if v < 4 {
		if err := sqliteutil.ExecAllowExists(s.db, `ALTER TABLE price_diff_watches ADD COLUMN exchanges TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
		if err := sqliteutil.SetUserVersion(s.db, 4); err != nil {
			return err
		}
		v = 4
	}
	if v < 5 {
		for _, stmt := range []string{
			`ALTER TABLE price_diff_opportunities ADD COLUMN open_buy_price REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN open_sell_price REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN open_gross_diff_pct REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN open_net_diff_pct REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN open_profit REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN peak_buy_price REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN peak_sell_price REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN peak_gross_diff_pct REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN peak_net_diff_pct REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN peak_profit REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN peak_at TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN fade_started_at TEXT`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN close_reason TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN close_net_diff_pct REAL NOT NULL DEFAULT 0`,
			`ALTER TABLE price_diff_opportunities ADD COLUMN close_profit REAL NOT NULL DEFAULT 0`,
		} {
			if err := sqliteutil.ExecAllowExists(s.db, stmt); err != nil {
				return err
			}
		}
		if _, err := s.db.Exec(`
UPDATE price_diff_opportunities SET
	open_buy_price = buy_price,
	open_sell_price = sell_price,
	open_gross_diff_pct = gross_diff_pct,
	open_net_diff_pct = net_diff_pct,
	peak_buy_price = buy_price,
	peak_sell_price = sell_price,
	peak_gross_diff_pct = gross_diff_pct,
	peak_net_diff_pct = net_diff_pct,
	peak_at = opened_at
WHERE peak_at = '' OR peak_at IS NULL
`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS price_diff_opp_ticks (
	opportunity_id TEXT NOT NULL,
	at TEXT NOT NULL,
	buy_price REAL NOT NULL,
	sell_price REAL NOT NULL,
	gross_diff_pct REAL NOT NULL,
	net_diff_pct REAL NOT NULL,
	profit REAL NOT NULL,
	executable INTEGER NOT NULL,
	PRIMARY KEY (opportunity_id, at),
	FOREIGN KEY (opportunity_id) REFERENCES price_diff_opportunities(id) ON DELETE CASCADE
)`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_pd_opp_ticks ON price_diff_opp_ticks(opportunity_id, at)`); err != nil {
			return err
		}
		if err := sqliteutil.SetUserVersion(s.db, 5); err != nil {
			return err
		}
	}
	return nil
}

const watchCols = `id, client_id, symbol, min_net_diff_pct, fee_binance_pct, fee_coinbase_pct,
	fee_bybit_pct, status, created_at, updated_at, notional, min_profit, min_duration_sec, exchanges`

type scannable interface {
	Scan(dest ...any) error
}

func scanWatch(row scannable) (*domain.PriceDiffWatch, error) {
	var w domain.PriceDiffWatch
	var st, cAt, uAt, exchanges string
	if err := row.Scan(
		&w.ID, &w.ClientID, &w.Symbol, &w.MinNetDiffPct, &w.FeeBinancePct, &w.FeeCoinbasePct,
		&w.FeeBybitPct, &st, &cAt, &uAt, &w.Notional, &w.MinProfit, &w.MinDurationSec, &exchanges,
	); err != nil {
		return nil, err
	}
	w.Status = domain.PriceDiffWatchStatus(st)
	w.CreatedAt = parseTime(cAt)
	w.UpdatedAt = parseTime(uAt)
	w.Exchanges = domain.ParsePriceDiffExchanges(exchanges)
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, w.ID, w.ClientID, w.Symbol, w.MinNetDiffPct, w.FeeBinancePct, w.FeeCoinbasePct, w.FeeBybitPct,
		string(w.Status), w.CreatedAt.UTC().Format(time.RFC3339Nano), w.UpdatedAt.UTC().Format(time.RFC3339Nano),
		w.Notional, w.MinProfit, w.MinDurationSec, domain.EncodePriceDiffExchanges(w.Exchanges))
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

// UpdateWatch writes mutable settings and status.
func (s *SQLite) UpdateWatch(ctx context.Context, w domain.PriceDiffWatch) (*domain.PriceDiffWatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE price_diff_watches SET
			notional = ?, min_profit = ?, min_net_diff_pct = ?, min_duration_sec = ?,
			fee_binance_pct = ?, fee_coinbase_pct = ?, fee_bybit_pct = ?,
			status = ?, updated_at = ?, exchanges = ?
		WHERE id = ? AND client_id = ?
	`, w.Notional, w.MinProfit, w.MinNetDiffPct, w.MinDurationSec,
		w.FeeBinancePct, w.FeeCoinbasePct, w.FeeBybitPct,
		string(w.Status), w.UpdatedAt.UTC().Format(time.RFC3339Nano),
		domain.EncodePriceDiffExchanges(w.Exchanges), w.ID, w.ClientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getWatchUnlocked(ctx, w.ClientID, w.ID)
}

// CountWatches counts watches for a client.
func (s *SQLite) CountWatches(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM price_diff_watches WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

const oppCols = `id, watch_id, client_id, symbol, buy_exchange, sell_exchange, buy_price, sell_price,
	gross_diff_pct, net_diff_pct, min_net_diff_pct, status, opened_at, last_seen_at, closed_at,
	open_buy_price, open_sell_price, open_gross_diff_pct, open_net_diff_pct, open_profit,
	peak_buy_price, peak_sell_price, peak_gross_diff_pct, peak_net_diff_pct, peak_profit, peak_at,
	fade_started_at, close_reason, close_net_diff_pct, close_profit`

func scanOpp(row scannable) (*domain.PriceDiffOpportunity, error) {
	var o domain.PriceDiffOpportunity
	var buyEx, sellEx, st, opened, last, peakAt, closeReason string
	var closed, fade sql.NullString
	if err := row.Scan(
		&o.ID, &o.WatchID, &o.ClientID, &o.Symbol, &buyEx, &sellEx, &o.BuyPrice, &o.SellPrice,
		&o.GrossDiffPct, &o.NetDiffPct, &o.MinNetDiffPct, &st, &opened, &last, &closed,
		&o.OpenBuyPrice, &o.OpenSellPrice, &o.OpenGrossDiffPct, &o.OpenNetDiffPct, &o.OpenProfit,
		&o.PeakBuyPrice, &o.PeakSellPrice, &o.PeakGrossDiffPct, &o.PeakNetDiffPct, &o.PeakProfit, &peakAt,
		&fade, &closeReason, &o.CloseNetDiffPct, &o.CloseProfit,
	); err != nil {
		return nil, err
	}
	o.BuyExchange = domain.Exchange(buyEx)
	o.SellExchange = domain.Exchange(sellEx)
	o.Status = domain.PriceDiffOppStatus(st)
	o.OpenedAt = parseTime(opened)
	o.LastSeenAt = parseTime(last)
	o.PeakAt = parseTime(peakAt)
	o.CloseReason = domain.PriceDiffCloseReason(closeReason)
	if closed.Valid && closed.String != "" {
		t := parseTime(closed.String)
		o.ClosedAt = &t
	}
	if fade.Valid && fade.String != "" {
		t := parseTime(fade.String)
		o.FadeStartedAt = &t
	}
	return &o, nil
}

func oppInsertArgs(o domain.PriceDiffOpportunity) []any {
	return []any{
		o.ID, o.WatchID, o.ClientID, o.Symbol, string(o.BuyExchange), string(o.SellExchange),
		o.BuyPrice, o.SellPrice, o.GrossDiffPct, o.NetDiffPct, o.MinNetDiffPct, string(o.Status),
		o.OpenedAt.UTC().Format(time.RFC3339Nano), o.LastSeenAt.UTC().Format(time.RFC3339Nano), nullTime(o.ClosedAt),
		o.OpenBuyPrice, o.OpenSellPrice, o.OpenGrossDiffPct, o.OpenNetDiffPct, o.OpenProfit,
		o.PeakBuyPrice, o.PeakSellPrice, o.PeakGrossDiffPct, o.PeakNetDiffPct, o.PeakProfit, textTime(o.PeakAt),
		nullTime(o.FadeStartedAt), string(o.CloseReason), o.CloseNetDiffPct, o.CloseProfit,
	}
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
	args := append(oppInsertArgs(o), o.WatchID)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO price_diff_opportunities (`+oppCols+`)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		WHERE EXISTS (SELECT 1 FROM price_diff_watches WHERE id = ? AND status = 'active')
	`, args...)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unique") {
			// Concurrent create — treat as race; return existing
			return s.GetOpenOpportunity(ctx, o.WatchID, o.BuyExchange, o.SellExchange)
		}
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrConflict
	}
	_ = s.appendTickUnlocked(ctx, o.ID, domain.PriceDiffOppSample{
		At: o.OpenedAt, BuyPrice: o.BuyPrice, SellPrice: o.SellPrice,
		GrossDiffPct: o.GrossDiffPct, NetDiffPct: o.NetDiffPct, Profit: o.OpenProfit, Executable: true,
	})
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

func textTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// TouchOpportunity records a live sample on an open opportunity.
func (s *SQLite) TouchOpportunity(ctx context.Context, id string, sample domain.PriceDiffOppSample) (*domain.PriceDiffOpportunity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := s.getOppByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.Status != domain.PriceDiffOppOpen {
		return nil, domain.ErrNotFound
	}
	domain.ApplyPriceDiffOppTick(o, sample)
	if err := s.updateOppUnlocked(ctx, *o); err != nil {
		return nil, err
	}
	_ = s.appendTickUnlocked(ctx, o.ID, sample)
	return s.getOppByID(ctx, id)
}

// CloseOpportunity marks an open opportunity closed.
func (s *SQLite) CloseOpportunity(ctx context.Context, id string, at time.Time, reason domain.PriceDiffCloseReason, last *domain.PriceDiffOppSample) (*domain.PriceDiffOpportunity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := s.getOppByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if o.Status != domain.PriceDiffOppOpen {
		return nil, domain.ErrNotFound
	}
	domain.ClosePriceDiffOpportunity(o, at, reason, last)
	if err := s.updateOppUnlocked(ctx, *o); err != nil {
		return nil, err
	}
	if last != nil {
		_ = s.appendTickUnlocked(ctx, o.ID, *last)
	}
	return s.getOppByID(ctx, id)
}

func (s *SQLite) updateOppUnlocked(ctx context.Context, o domain.PriceDiffOpportunity) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE price_diff_opportunities SET
			buy_price = ?, sell_price = ?, gross_diff_pct = ?, net_diff_pct = ?,
			last_seen_at = ?, closed_at = ?, status = ?,
			open_buy_price = ?, open_sell_price = ?, open_gross_diff_pct = ?, open_net_diff_pct = ?, open_profit = ?,
			peak_buy_price = ?, peak_sell_price = ?, peak_gross_diff_pct = ?, peak_net_diff_pct = ?, peak_profit = ?, peak_at = ?,
			fade_started_at = ?, close_reason = ?, close_net_diff_pct = ?, close_profit = ?
		WHERE id = ?
	`, o.BuyPrice, o.SellPrice, o.GrossDiffPct, o.NetDiffPct,
		textTime(o.LastSeenAt), nullTime(o.ClosedAt), string(o.Status),
		o.OpenBuyPrice, o.OpenSellPrice, o.OpenGrossDiffPct, o.OpenNetDiffPct, o.OpenProfit,
		o.PeakBuyPrice, o.PeakSellPrice, o.PeakGrossDiffPct, o.PeakNetDiffPct, o.PeakProfit, textTime(o.PeakAt),
		nullTime(o.FadeStartedAt), string(o.CloseReason), o.CloseNetDiffPct, o.CloseProfit, o.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLite) appendTickUnlocked(ctx context.Context, oppID string, sample domain.PriceDiffOppSample) error {
	if oppID == "" || sample.At.IsZero() {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO price_diff_opp_ticks
			(opportunity_id, at, buy_price, sell_price, gross_diff_pct, net_diff_pct, profit, executable)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, oppID, sample.At.UTC().Format(time.RFC3339Nano), sample.BuyPrice, sample.SellPrice,
		sample.GrossDiffPct, sample.NetDiffPct, sample.Profit, boolInt(sample.Executable)); err != nil {
		return err
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM price_diff_opp_ticks WHERE opportunity_id = ?`, oppID).Scan(&n); err != nil {
		return err
	}
	if n <= domain.MaxPriceDiffOppTicks {
		return nil
	}
	drop := n - domain.MaxPriceDiffOppTicks
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM price_diff_opp_ticks WHERE opportunity_id = ? AND at IN (
			SELECT at FROM price_diff_opp_ticks WHERE opportunity_id = ? ORDER BY at ASC LIMIT ?
		)
	`, oppID, oppID, drop)
	return err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ListOpportunityTicks returns oldest-first samples for one opportunity.
func (s *SQLite) ListOpportunityTicks(ctx context.Context, opportunityID string, limit int) ([]domain.PriceDiffOppTick, error) {
	if limit <= 0 || limit > domain.MaxPriceDiffOppTicks {
		limit = domain.MaxPriceDiffOppTicks
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT opportunity_id, at, buy_price, sell_price, gross_diff_pct, net_diff_pct, profit, executable
		FROM price_diff_opp_ticks WHERE opportunity_id = ? ORDER BY at ASC LIMIT ?
	`, opportunityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PriceDiffOppTick
	for rows.Next() {
		var t domain.PriceDiffOppTick
		var at string
		var exec int
		if err := rows.Scan(&t.OpportunityID, &at, &t.BuyPrice, &t.SellPrice, &t.GrossDiffPct, &t.NetDiffPct, &t.Profit, &exec); err != nil {
			return nil, err
		}
		t.At = parseTime(at)
		t.Executable = exec != 0
		out = append(out, t)
	}
	return out, rows.Err()
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

// ListRouteArms returns first-seen times for qualifying routes on a watch.
func (s *SQLite) ListRouteArms(ctx context.Context, watchID string) ([]domain.PriceDiffRouteArm, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT watch_id, buy_exchange, sell_exchange, armed_at
		FROM price_diff_route_arms WHERE watch_id = ?
	`, watchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PriceDiffRouteArm
	for rows.Next() {
		var a domain.PriceDiffRouteArm
		var buy, sell, at string
		if err := rows.Scan(&a.WatchID, &buy, &sell, &at); err != nil {
			return nil, err
		}
		a.BuyExchange = domain.Exchange(buy)
		a.SellExchange = domain.Exchange(sell)
		a.ArmedAt = parseTime(at)
		out = append(out, a)
	}
	return out, rows.Err()
}

// SetRouteArm records first-seen time; an existing row is left unchanged.
func (s *SQLite) SetRouteArm(ctx context.Context, arm domain.PriceDiffRouteArm) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if arm.ArmedAt.IsZero() {
		arm.ArmedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO price_diff_route_arms (watch_id, buy_exchange, sell_exchange, armed_at)
		VALUES (?, ?, ?, ?)
	`, arm.WatchID, string(arm.BuyExchange), string(arm.SellExchange), arm.ArmedAt.UTC().Format(time.RFC3339Nano))
	return err
}

// ClearRouteArm drops a pending arm when the condition breaks or an opportunity opens.
func (s *SQLite) ClearRouteArm(ctx context.Context, watchID string, buy, sell domain.Exchange) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM price_diff_route_arms
		WHERE watch_id = ? AND buy_exchange = ? AND sell_exchange = ?
	`, watchID, string(buy), string(sell))
	return err
}

// ClearRouteArms drops every pending duration timer for a watch.
func (s *SQLite) ClearRouteArms(ctx context.Context, watchID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM price_diff_route_arms WHERE watch_id = ?`, watchID)
	return err
}
