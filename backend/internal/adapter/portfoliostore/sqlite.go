package portfoliostore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/sqliteutil"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists paper portfolios.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the portfolio database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("portfolio sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create portfolio db dir: %w", err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	// foreign_keys off: child rows key by book id while owner lives on portfolios.client_id.
	dsn := "file:" + filepath.ToSlash(abs) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(0)"
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

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS portfolios (
	id                TEXT PRIMARY KEY NOT NULL,
	client_id         TEXT NOT NULL,
	name              TEXT NOT NULL DEFAULT 'Main',
	currency          TEXT NOT NULL,
	starting_balance  REAL NOT NULL,
	cash_balance      REAL NOT NULL,
	realized_pnl_total REAL NOT NULL DEFAULT 0,
	net_deposits      REAL NOT NULL DEFAULT 0,
	margin_mode       TEXT NOT NULL DEFAULT 'isolated',
	created_at        TEXT NOT NULL,
	updated_at        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_portfolios_client ON portfolios(client_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_portfolios_client_name ON portfolios(client_id, name);
CREATE TABLE IF NOT EXISTS positions (
	client_id  TEXT NOT NULL,
	exchange   TEXT NOT NULL,
	symbol     TEXT NOT NULL,
	quantity   REAL NOT NULL,
	avg_cost   REAL NOT NULL,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (client_id, exchange, symbol),
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS trades (
	id               TEXT PRIMARY KEY NOT NULL,
	client_id        TEXT NOT NULL,
	exchange         TEXT NOT NULL,
	symbol           TEXT NOT NULL,
	side             TEXT NOT NULL,
	quantity         REAL NOT NULL,
	price            REAL NOT NULL,
	notional         REAL NOT NULL,
	realized_pnl     REAL NOT NULL DEFAULT 0,
	pending_order_id TEXT NOT NULL DEFAULT '',
	created_at       TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_trades_client ON trades(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_positions_client ON positions(client_id);
CREATE TABLE IF NOT EXISTS pending_orders (
	id                 TEXT PRIMARY KEY NOT NULL,
	client_id          TEXT NOT NULL,
	exchange           TEXT NOT NULL,
	symbol             TEXT NOT NULL,
	order_type         TEXT NOT NULL,
	side               TEXT NOT NULL,
	quantity           REAL NOT NULL,
	filled_quantity    REAL NOT NULL DEFAULT 0,
	remaining_quantity REAL NOT NULL DEFAULT 0,
	trigger_price      REAL NOT NULL,
	reserved_cash      REAL NOT NULL DEFAULT 0,
	reserved_quantity  REAL NOT NULL DEFAULT 0,
	time_in_force      TEXT NOT NULL DEFAULT 'gtc',
	expires_at         TEXT,
	status             TEXT NOT NULL,
	created_at         TEXT NOT NULL,
	updated_at         TEXT NOT NULL,
	filled_at          TEXT,
	canceled_at        TEXT,
	fill_trade_id      TEXT NOT NULL DEFAULT '',
	fill_price         REAL NOT NULL DEFAULT 0,
	reject_reason      TEXT NOT NULL DEFAULT '',
	cancel_reason      TEXT NOT NULL DEFAULT '',
	oco_group_id       TEXT NOT NULL DEFAULT '',
	oco_peer_id        TEXT NOT NULL DEFAULT '',
	trail_type         TEXT NOT NULL DEFAULT '',
	trail_value        REAL NOT NULL DEFAULT 0,
	trail_peak         REAL NOT NULL DEFAULT 0,
	bracket_id         TEXT NOT NULL DEFAULT '',
	bracket_role       TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pending_orders_client ON pending_orders(client_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pending_orders_open ON pending_orders(status) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS idx_pending_orders_oco ON pending_orders(oco_group_id) WHERE oco_group_id != '';

CREATE TABLE IF NOT EXISTS recurring_buy_plans (
	id              TEXT PRIMARY KEY NOT NULL,
	client_id       TEXT NOT NULL,
	exchange        TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	name            TEXT NOT NULL DEFAULT '',
	amount          REAL NOT NULL,
	frequency       TEXT NOT NULL,
	weekday         TEXT NOT NULL DEFAULT '',
	day_of_month    INTEGER NOT NULL DEFAULT 0,
	interval_hours  INTEGER NOT NULL DEFAULT 0,
	status          TEXT NOT NULL,
	next_run_at     TEXT NOT NULL,
	last_run_at     TEXT,
	last_period_key TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_recurring_buy_client ON recurring_buy_plans(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recurring_buy_due ON recurring_buy_plans(status, next_run_at);

CREATE TABLE IF NOT EXISTS recurring_buy_runs (
	id            TEXT PRIMARY KEY NOT NULL,
	plan_id       TEXT NOT NULL,
	client_id     TEXT NOT NULL,
	period_key    TEXT NOT NULL,
	status        TEXT NOT NULL,
	amount        REAL NOT NULL DEFAULT 0,
	quantity      REAL NOT NULL DEFAULT 0,
	price         REAL NOT NULL DEFAULT 0,
	trade_id      TEXT NOT NULL DEFAULT '',
	fail_reason   TEXT NOT NULL DEFAULT '',
	scheduled_for TEXT NOT NULL,
	executed_at   TEXT NOT NULL,
	UNIQUE (plan_id, period_key),
	FOREIGN KEY (plan_id) REFERENCES recurring_buy_plans(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_recurring_buy_runs_plan ON recurring_buy_runs(plan_id, executed_at DESC);

CREATE TABLE IF NOT EXISTS margin_positions (
	id                TEXT PRIMARY KEY NOT NULL,
	client_id         TEXT NOT NULL,
	exchange          TEXT NOT NULL,
	symbol            TEXT NOT NULL,
	side              TEXT NOT NULL,
	mode              TEXT NOT NULL DEFAULT 'isolated',
	quantity          REAL NOT NULL,
	entry_price       REAL NOT NULL,
	leverage          INTEGER NOT NULL,
	margin            REAL NOT NULL,
	debt_principal    REAL NOT NULL DEFAULT 0,
	debt_interest     REAL NOT NULL DEFAULT 0,
	debt_asset        TEXT NOT NULL DEFAULT 'quote',
	last_interest_at  TEXT,
	liquidation_price REAL NOT NULL,
	stop_loss         REAL,
	take_profit       REAL,
	status            TEXT NOT NULL,
	realized_pnl      REAL NOT NULL DEFAULT 0,
	close_reason      TEXT NOT NULL DEFAULT '',
	opened_at         TEXT NOT NULL,
	updated_at        TEXT NOT NULL,
	closed_at         TEXT,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_margin_pos_client ON margin_positions(client_id, status, opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_margin_pos_open ON margin_positions(status) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS margin_orders (
	id               TEXT PRIMARY KEY NOT NULL,
	client_id        TEXT NOT NULL,
	exchange         TEXT NOT NULL,
	symbol           TEXT NOT NULL,
	side             TEXT NOT NULL,
	order_type       TEXT NOT NULL,
	quantity         REAL NOT NULL,
	leverage         INTEGER NOT NULL,
	limit_price      REAL NOT NULL DEFAULT 0,
	reserved_margin  REAL NOT NULL DEFAULT 0,
	stop_loss        REAL,
	take_profit      REAL,
	status           TEXT NOT NULL,
	position_id      TEXT NOT NULL DEFAULT '',
	reject_reason    TEXT NOT NULL DEFAULT '',
	cancel_reason    TEXT NOT NULL DEFAULT '',
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL,
	filled_at        TEXT,
	canceled_at      TEXT,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_margin_orders_client ON margin_orders(client_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_margin_orders_open ON margin_orders(status) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS margin_trades (
	id            TEXT PRIMARY KEY NOT NULL,
	client_id     TEXT NOT NULL,
	position_id   TEXT NOT NULL,
	exchange      TEXT NOT NULL,
	symbol        TEXT NOT NULL,
	side          TEXT NOT NULL,
	action        TEXT NOT NULL,
	quantity      REAL NOT NULL,
	price         REAL NOT NULL,
	notional      REAL NOT NULL,
	realized_pnl    REAL NOT NULL DEFAULT 0,
	margin_delta    REAL NOT NULL DEFAULT 0,
	principal_paid  REAL NOT NULL DEFAULT 0,
	interest_paid   REAL NOT NULL DEFAULT 0,
	leverage        INTEGER NOT NULL DEFAULT 1,
	created_at      TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_margin_trades_client ON margin_trades(client_id, created_at DESC);
-- At most one SL / TP trade per position. Liquidation may be partial (multiple rows);
-- full liquidation uses deterministic trade id (primary key) for restart safety.
CREATE UNIQUE INDEX IF NOT EXISTS idx_margin_trades_sl_tp
	ON margin_trades(position_id, action)
	WHERE action IN ('stop_loss', 'take_profit');

CREATE TABLE IF NOT EXISTS allocation_baskets (
	id         TEXT PRIMARY KEY NOT NULL,
	client_id  TEXT NOT NULL,
	name       TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_allocation_baskets_client ON allocation_baskets(client_id, created_at DESC);
CREATE TABLE IF NOT EXISTS allocation_targets (
	basket_id  TEXT NOT NULL,
	asset      TEXT NOT NULL,
	exchange   TEXT NOT NULL DEFAULT '',
	weight_pct REAL NOT NULL,
	PRIMARY KEY (basket_id, asset),
	FOREIGN KEY (basket_id) REFERENCES allocation_baskets(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS risk_limits (
	client_id             TEXT PRIMARY KEY NOT NULL,
	max_daily_loss_pct    REAL,
	max_asset_weight_pct  REAL,
	day_key               TEXT NOT NULL DEFAULT '',
	day_start_equity      REAL NOT NULL DEFAULT 0,
	updated_at            TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS portfolio_equity_snapshots (
	client_id        TEXT NOT NULL,
	bucket_at        TEXT NOT NULL,
	taken_at         TEXT NOT NULL,
	equity           REAL NOT NULL,
	cash_balance     REAL NOT NULL,
	positions_value  REAL NOT NULL,
	margin_equity    REAL NOT NULL,
	unrealized_pnl   REAL NOT NULL,
	realized_pnl     REAL NOT NULL,
	PRIMARY KEY (client_id, bucket_at),
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pf_equity_snap_time ON portfolio_equity_snapshots(bucket_at);
CREATE TABLE IF NOT EXISTS cash_movements (
	id                  TEXT PRIMARY KEY NOT NULL,
	client_id           TEXT NOT NULL,
	kind                TEXT NOT NULL,
	amount              REAL NOT NULL,
	cash_after          REAL NOT NULL,
	net_deposits_after  REAL NOT NULL,
	note                TEXT NOT NULL DEFAULT '',
	counterparty_portfolio_id   TEXT NOT NULL DEFAULT '',
	counterparty_portfolio_name TEXT NOT NULL DEFAULT '',
	peer_movement_id            TEXT NOT NULL DEFAULT '',
	created_at          TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cash_movements_client ON cash_movements(client_id, created_at DESC);
CREATE TABLE IF NOT EXISTS portfolio_shares (
	portfolio_id      TEXT NOT NULL,
	owner_client_id   TEXT NOT NULL,
	grantee_client_id TEXT NOT NULL,
	role              TEXT NOT NULL,
	created_at        TEXT NOT NULL,
	updated_at        TEXT NOT NULL,
	PRIMARY KEY (portfolio_id, grantee_client_id)
);
CREATE INDEX IF NOT EXISTS idx_portfolio_shares_grantee ON portfolio_shares(grantee_client_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_shares_owner ON portfolio_shares(owner_client_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	if err := s.migrateMultiBook(); err != nil {
		return err
	}
	if err := s.migrateTaxLots(); err != nil {
		return err
	}
	if err := s.migrateIdempotency(); err != nil {
		return err
	}
	// Additive migrations for DBs created before reservations/partial fills.
	alters := []string{
		`ALTER TABLE trades ADD COLUMN pending_order_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pending_orders ADD COLUMN filled_quantity REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pending_orders ADD COLUMN remaining_quantity REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pending_orders ADD COLUMN reserved_cash REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pending_orders ADD COLUMN reserved_quantity REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pending_orders ADD COLUMN time_in_force TEXT NOT NULL DEFAULT 'gtc'`,
		`ALTER TABLE pending_orders ADD COLUMN expires_at TEXT`,
		`ALTER TABLE pending_orders ADD COLUMN cancel_reason TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pending_orders ADD COLUMN oco_group_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pending_orders ADD COLUMN oco_peer_id TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_pending_orders_oco ON pending_orders(oco_group_id) WHERE oco_group_id != ''`,
		`ALTER TABLE pending_orders ADD COLUMN trail_type TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pending_orders ADD COLUMN trail_value REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pending_orders ADD COLUMN trail_peak REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pending_orders ADD COLUMN bracket_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pending_orders ADD COLUMN bracket_role TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_pending_orders_bracket ON pending_orders(bracket_id) WHERE bracket_id != ''`,
		`ALTER TABLE portfolios ADD COLUMN margin_mode TEXT NOT NULL DEFAULT 'isolated'`,
		`ALTER TABLE margin_positions ADD COLUMN mode TEXT NOT NULL DEFAULT 'isolated'`,
		`ALTER TABLE margin_positions ADD COLUMN debt_principal REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE margin_positions ADD COLUMN debt_interest REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE margin_positions ADD COLUMN debt_asset TEXT NOT NULL DEFAULT 'quote'`,
		`ALTER TABLE margin_positions ADD COLUMN last_interest_at TEXT`,
		`ALTER TABLE margin_trades ADD COLUMN principal_paid REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE margin_trades ADD COLUMN interest_paid REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE recurring_buy_plans ADD COLUMN name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE recurring_buy_plans ADD COLUMN weekday TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE recurring_buy_plans ADD COLUMN day_of_month INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE recurring_buy_plans ADD COLUMN interval_hours INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE portfolios ADD COLUMN net_deposits REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE pending_orders ADD COLUMN lot_method TEXT NOT NULL DEFAULT 'fifo'`,
		`ALTER TABLE trades ADD COLUMN lot_method TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trades ADD COLUMN fee REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE trades ADD COLUMN last_price REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE margin_trades ADD COLUMN fee REAL NOT NULL DEFAULT 0`,
		`CREATE TABLE IF NOT EXISTS cash_movements (
	id                  TEXT PRIMARY KEY NOT NULL,
	client_id           TEXT NOT NULL,
	kind                TEXT NOT NULL,
	amount              REAL NOT NULL,
	cash_after          REAL NOT NULL,
	net_deposits_after  REAL NOT NULL,
	note                TEXT NOT NULL DEFAULT '',
	counterparty_portfolio_id   TEXT NOT NULL DEFAULT '',
	counterparty_portfolio_name TEXT NOT NULL DEFAULT '',
	peer_movement_id            TEXT NOT NULL DEFAULT '',
	created_at          TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
)`,
		`CREATE INDEX IF NOT EXISTS idx_cash_movements_client ON cash_movements(client_id, created_at DESC)`,
		`ALTER TABLE cash_movements ADD COLUMN counterparty_portfolio_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cash_movements ADD COLUMN counterparty_portfolio_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE cash_movements ADD COLUMN peer_movement_id TEXT NOT NULL DEFAULT ''`,
		// Prefer SL/TP uniqueness only; drop older liquidation-inclusive unique index if present.
		`DROP INDEX IF EXISTS idx_margin_trades_forced_close`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_margin_trades_sl_tp ON margin_trades(position_id, action) WHERE action IN ('stop_loss', 'take_profit')`,
	}
	for _, q := range alters {
		if err := sqliteutil.ExecAllowExists(s.db, q); err != nil {
			return err
		}
	}
	if err := sqliteutil.SetUserVersion(s.db, 1); err != nil {
		return err
	}
	// Backfill remaining_quantity for legacy open orders that only had quantity.
	_, _ = s.db.Exec(`
		UPDATE pending_orders
		SET remaining_quantity = quantity - COALESCE(filled_quantity, 0)
		WHERE remaining_quantity = 0 AND status = 'open' AND quantity > 0
	`)
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

func scanPortfolio(sc interface{ Scan(dest ...any) error }) (*domain.Portfolio, error) {
	var p domain.Portfolio
	var cAt, uAt, mode string
	err := sc.Scan(&p.ID, &p.ClientID, &p.Name, &p.Currency, &p.StartingBalance, &p.CashBalance, &p.RealizedPnLTotal,
		&p.NetDeposits, &mode, &cAt, &uAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.MarginMode = domain.MarginMode(mode)
	if p.MarginMode == "" {
		p.MarginMode = domain.MarginModeIsolated
	}
	if p.Name == "" {
		p.Name = domain.DefaultPortfolioName
	}
	p.CreatedAt = parseTime(cAt)
	p.UpdatedAt = parseTime(uAt)
	return &p, nil
}

const portfolioCols = `id, client_id, name, currency, starting_balance, cash_balance, realized_pnl_total,
			COALESCE(net_deposits, 0), COALESCE(margin_mode, 'isolated'), created_at, updated_at`

// GetPortfolio returns a book by id.
func (s *SQLite) GetPortfolio(ctx context.Context, id string) (*domain.Portfolio, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+portfolioCols+` FROM portfolios WHERE id = ?`, id)
	return scanPortfolio(row)
}

// ListPortfolios lists books for an owner.
func (s *SQLite) ListPortfolios(ctx context.Context, clientID string) ([]domain.Portfolio, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+portfolioCols+` FROM portfolios WHERE client_id = ? ORDER BY created_at ASC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Portfolio
	for rows.Next() {
		p, err := scanPortfolio(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// CountPortfolios counts books for an owner.
func (s *SQLite) CountPortfolios(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM portfolios WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// CreatePortfolio inserts a new book.
func (s *SQLite) CreatePortfolio(ctx context.Context, p domain.Portfolio) (*domain.Portfolio, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = p.ClientID
	}
	if p.Name == "" {
		p.Name = domain.DefaultPortfolioName
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.CreatedAt
	}
	if p.MarginMode == "" {
		p.MarginMode = domain.MarginModeIsolated
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO portfolios (id, client_id, name, currency, starting_balance, cash_balance, realized_pnl_total, net_deposits, margin_mode, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.ClientID, p.Name, p.Currency, p.StartingBalance, p.CashBalance, p.RealizedPnLTotal, p.NetDeposits, string(p.MarginMode),
		p.CreatedAt.UTC().Format(time.RFC3339Nano), p.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, fmt.Errorf("%w: a portfolio named %q already exists", domain.ErrInvalidArgument, p.Name)
		}
		return nil, fmt.Errorf("portfolio create: %w", err)
	}
	cp := p
	return &cp, nil
}

// UpdatePortfolioName renames a book owned by clientID.
func (s *SQLite) UpdatePortfolioName(ctx context.Context, clientID, id, name string, at time.Time) (*domain.Portfolio, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE portfolios SET name = ?, updated_at = ? WHERE id = ? AND client_id = ?
	`, name, at.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetPortfolio(ctx, id)
}

// DeletePortfolio removes a book and child rows keyed by book id.
func (s *SQLite) DeletePortfolio(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var owner string
	err := s.db.QueryRowContext(ctx, `SELECT client_id FROM portfolios WHERE id = ?`, id).Scan(&owner)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if owner != clientID {
		return domain.ErrNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, q := range []string{
		`DELETE FROM idempotency_keys WHERE client_id = ?`,
		`DELETE FROM cash_movements WHERE client_id = ?`,
		`DELETE FROM portfolio_equity_snapshots WHERE client_id = ?`,
		`DELETE FROM risk_limits WHERE client_id = ?`,
		`DELETE FROM allocation_targets WHERE basket_id IN (SELECT id FROM allocation_baskets WHERE client_id = ?)`,
		`DELETE FROM allocation_baskets WHERE client_id = ?`,
		`DELETE FROM recurring_buy_runs WHERE client_id = ?`,
		`DELETE FROM recurring_buy_plans WHERE client_id = ?`,
		`DELETE FROM pending_orders WHERE client_id = ?`,
		`DELETE FROM tax_lot_fills WHERE lot_id IN (SELECT id FROM tax_lots WHERE client_id = ?)`,
		`DELETE FROM tax_lots WHERE client_id = ?`,
		`DELETE FROM trades WHERE client_id = ?`,
		`DELETE FROM positions WHERE client_id = ?`,
		`DELETE FROM margin_trades WHERE client_id = ?`,
		`DELETE FROM margin_orders WHERE client_id = ?`,
		`DELETE FROM margin_positions WHERE client_id = ?`,
		`DELETE FROM portfolio_shares WHERE portfolio_id = ?`,
		`DELETE FROM portfolios WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateCashAndRealized updates balances.
func (s *SQLite) UpdateCashAndRealized(ctx context.Context, clientID string, cash, realizedTotal float64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, realized_pnl_total = ?, updated_at = ?
		WHERE id = ?
	`, cash, realizedTotal, at.UTC().Format(time.RFC3339Nano), clientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetPosition returns one position or ErrNotFound.
func (s *SQLite) GetPosition(ctx context.Context, clientID string, exchange domain.Exchange, symbol string) (*domain.Position, error) {
	var p domain.Position
	var ex, uAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT client_id, exchange, symbol, quantity, avg_cost, updated_at
		FROM positions WHERE client_id = ? AND exchange = ? AND symbol = ?
	`, clientID, string(exchange), symbol).Scan(&p.ClientID, &ex, &p.Symbol, &p.Quantity, &p.AvgCost, &uAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Exchange = domain.Exchange(ex)
	p.UpdatedAt = parseTime(uAt)
	return &p, nil
}

// ListPositions lists open positions (qty > 0).
func (s *SQLite) ListPositions(ctx context.Context, clientID string) ([]domain.Position, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT client_id, exchange, symbol, quantity, avg_cost, updated_at
		FROM positions WHERE client_id = ? AND quantity > 0
		ORDER BY symbol ASC
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Position, 0)
	for rows.Next() {
		var p domain.Position
		var ex, uAt string
		if err := rows.Scan(&p.ClientID, &ex, &p.Symbol, &p.Quantity, &p.AvgCost, &uAt); err != nil {
			return nil, err
		}
		p.Exchange = domain.Exchange(ex)
		p.UpdatedAt = parseTime(uAt)
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertPosition writes or deletes a position.
func (s *SQLite) UpsertPosition(ctx context.Context, pos domain.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pos.UpdatedAt.IsZero() {
		pos.UpdatedAt = time.Now().UTC()
	}
	if pos.Quantity <= domain.PositionEpsilon {
		_, err := s.db.ExecContext(ctx, `
			DELETE FROM positions WHERE client_id = ? AND exchange = ? AND symbol = ?
		`, pos.ClientID, string(pos.Exchange), pos.Symbol)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO positions (client_id, exchange, symbol, quantity, avg_cost, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id, exchange, symbol) DO UPDATE SET
			quantity = excluded.quantity,
			avg_cost = excluded.avg_cost,
			updated_at = excluded.updated_at
	`, pos.ClientID, string(pos.Exchange), pos.Symbol, pos.Quantity, pos.AvgCost,
		pos.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

// InsertTrade records a fill.
func (s *SQLite) InsertTrade(ctx context.Context, t domain.Trade) (*domain.Trade, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.PendingOrderID,
		t.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	cp := t
	return &cp, nil
}

// ListTrades returns trade history newest first.
func (s *SQLite) ListTrades(ctx context.Context, clientID string, limit, offset int) ([]domain.Trade, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id,
		       COALESCE(lot_method, ''), COALESCE(fee, 0), COALESCE(last_price, 0), created_at
		FROM trades WHERE client_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Trade, 0)
	for rows.Next() {
		var t domain.Trade
		var ex, side, cAt, lotMethod string
		if err := rows.Scan(&t.ID, &t.ClientID, &ex, &t.Symbol, &side, &t.Quantity, &t.Price, &t.Notional, &t.RealizedPnL, &t.PendingOrderID, &lotMethod, &t.Fee, &t.LastPrice, &cAt); err != nil {
			return nil, err
		}
		t.LotMethod, _ = domain.NormalizeLotMethod(lotMethod)
		t.Exchange = domain.Exchange(ex)
		t.Side = domain.TradeSide(side)
		t.CreatedAt = parseTime(cAt)
		out = append(out, t)
	}
	return out, rows.Err()
}

// CountTrades returns trade count for a client.
func (s *SQLite) CountTrades(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM trades WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// ExecuteTrade applies portfolio + position + trade insert under one transaction.
func (s *SQLite) ExecuteTrade(ctx context.Context, p *domain.Portfolio, pos *domain.Position, t domain.Trade, lots *domain.LotOps) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	at := t.CreatedAt
	if at.IsZero() {
		at = time.Now().UTC()
		t.CreatedAt = at
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.txInsertIdempotency(ctx, tx); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, realized_pnl_total = ?, updated_at = ?
		WHERE id = ?
	`, p.CashBalance, p.RealizedPnLTotal, at.UTC().Format(time.RFC3339Nano), p.BookID()); err != nil {
		return err
	}

	if pos != nil {
		if pos.Quantity <= domain.PositionEpsilon {
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM positions WHERE client_id = ? AND exchange = ? AND symbol = ?
			`, pos.ClientID, string(pos.Exchange), pos.Symbol); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO positions (client_id, exchange, symbol, quantity, avg_cost, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT(client_id, exchange, symbol) DO UPDATE SET
					quantity = excluded.quantity,
					avg_cost = excluded.avg_cost,
					updated_at = excluded.updated_at
			`, pos.ClientID, string(pos.Exchange), pos.Symbol, pos.Quantity, pos.AvgCost,
				at.UTC().Format(time.RFC3339Nano)); err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id, lot_method, fee, last_price, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.PendingOrderID,
		string(t.LotMethod), t.Fee, t.LastPrice, at.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err := applyLotOps(ctx, tx, lots, t.ID, at); err != nil {
		return err
	}
	return tx.Commit()
}

// CreatePendingOrder inserts an open resting order with reservations.
func (s *SQLite) CreatePendingOrder(ctx context.Context, o domain.PendingOrder) (*domain.PendingOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = o.CreatedAt
	}
	if o.Status == "" {
		o.Status = domain.PendingStatusOpen
	}
	if o.RemainingQuantity <= 0 {
		o.RemainingQuantity = o.Quantity
	}
	if o.TimeInForce == "" {
		o.TimeInForce = domain.TimeInForceGTC
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.txInsertIdempotency(ctx, tx); err != nil {
		return nil, err
	}
	if err := s.txInsertPendingOrder(ctx, tx, o); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w", domain.ErrIdempotencyHit)
		}
		return nil, fmt.Errorf("pending order create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	cp := o
	return &cp, nil
}

// CreateOCOPair inserts take-profit + stop-loss legs atomically.
func (s *SQLite) CreateOCOPair(ctx context.Context, takeProfit, stopLoss domain.PendingOrder) (*domain.PendingOrder, *domain.PendingOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, o := range []*domain.PendingOrder{&takeProfit, &stopLoss} {
		if o.CreatedAt.IsZero() {
			o.CreatedAt = now
		}
		if o.UpdatedAt.IsZero() {
			o.UpdatedAt = o.CreatedAt
		}
		if o.Status == "" {
			o.Status = domain.PendingStatusOpen
		}
		if o.RemainingQuantity <= 0 {
			o.RemainingQuantity = o.Quantity
		}
		if o.TimeInForce == "" {
			o.TimeInForce = domain.TimeInForceGTC
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.txInsertIdempotency(ctx, tx); err != nil {
		return nil, nil, err
	}
	for _, o := range []domain.PendingOrder{takeProfit, stopLoss} {
		if err := s.txInsertPendingOrder(ctx, tx, o); err != nil {
			return nil, nil, fmt.Errorf("oco pair create: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	tp, sl := takeProfit, stopLoss
	return &tp, &sl, nil
}

const pendingOrderInsertSQL = `
		INSERT INTO pending_orders (
			id, client_id, exchange, symbol, order_type, side, quantity, filled_quantity, remaining_quantity,
			trigger_price, reserved_cash, reserved_quantity, time_in_force, expires_at, status,
			created_at, updated_at, filled_at, canceled_at, fill_trade_id, fill_price, reject_reason, cancel_reason,
			oco_group_id, oco_peer_id, trail_type, trail_value, trail_peak, bracket_id, bracket_role, lot_method
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *SQLite) txInsertPendingOrder(ctx context.Context, tx *sql.Tx, o domain.PendingOrder) error {
	_, err := tx.ExecContext(ctx, pendingOrderInsertSQL,
		o.ID, o.ClientID, string(o.Exchange), o.Symbol, string(o.Type), string(o.Side), o.Quantity, o.FilledQuantity, o.RemainingQuantity,
		o.TriggerPrice, o.ReservedCash, o.ReservedQuantity, string(o.TimeInForce), nullTime(o.ExpiresAt), string(o.Status),
		o.CreatedAt.UTC().Format(time.RFC3339Nano), o.UpdatedAt.UTC().Format(time.RFC3339Nano),
		nullTime(o.FilledAt), nullTime(o.CanceledAt), o.FillTradeID, o.FillPrice, o.RejectReason, o.CancelReason,
		o.OCOGroupID, o.OCOPeerID, o.TrailType, o.TrailValue, o.TrailPeak, o.BracketID, o.BracketRole, lotMethodOrDefault(o.LotMethod),
	)
	return err
}

func lotMethodOrDefault(m domain.LotMethod) string {
	if m == "" {
		return string(domain.DefaultLotMethod)
	}
	return string(m)
}

// CreateBracket inserts entry (open) and TP/SL exits (pending/inactive) in one transaction.
func (s *SQLite) CreateBracket(ctx context.Context, entry, takeProfit, stopLoss domain.PendingOrder) (*domain.PendingOrder, *domain.PendingOrder, *domain.PendingOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, o := range []*domain.PendingOrder{&entry, &takeProfit, &stopLoss} {
		if o.CreatedAt.IsZero() {
			o.CreatedAt = now
		}
		if o.UpdatedAt.IsZero() {
			o.UpdatedAt = o.CreatedAt
		}
		if o.TimeInForce == "" {
			o.TimeInForce = domain.TimeInForceGTC
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.txInsertIdempotency(ctx, tx); err != nil {
		return nil, nil, nil, err
	}
	for _, o := range []domain.PendingOrder{entry, takeProfit, stopLoss} {
		if err := s.txInsertPendingOrder(ctx, tx, o); err != nil {
			return nil, nil, nil, fmt.Errorf("bracket create: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, nil, err
	}
	e, tp, sl := entry, takeProfit, stopLoss
	return &e, &tp, &sl, nil
}

// SyncBracketExitsToFilled activates or grows exit legs so their open size matches entryFilled.
// remaining = entryFilled - filled_on_exit; status open when entryFilled > 0 and leg not canceled/filled.
func (s *SQLite) SyncBracketExitsToFilled(ctx context.Context, clientID, bracketID string, entryFilled float64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if entryFilled < 0 {
		entryFilled = 0
	}
	atStr := at.UTC().Format(time.RFC3339Nano)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, quantity, filled_quantity, remaining_quantity, status, bracket_role
		FROM pending_orders
		WHERE client_id = ? AND bracket_id = ? AND bracket_role IN ('take_profit', 'stop_loss')
	`, clientID, bracketID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type leg struct {
		id, status, role string
		qty, filled, rem float64
	}
	var legs []leg
	for rows.Next() {
		var L leg
		if err := rows.Scan(&L.id, &L.qty, &L.filled, &L.rem, &L.status, &L.role); err != nil {
			return err
		}
		legs = append(legs, L)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, L := range legs {
		// Do not resurrect canceled/rejected/filled exits.
		if L.status != string(domain.PendingStatusPending) && L.status != string(domain.PendingStatusOpen) {
			continue
		}
		// Target open size is entry filled amount; already-exited qty stays filled.
		if L.filled > entryFilled+1e-12 {
			// Exit already sold more than new target (should not happen); clamp remaining 0.
			_, err = s.db.ExecContext(ctx, `
				UPDATE pending_orders
				SET quantity = ?, remaining_quantity = 0, reserved_quantity = 0,
				    status = CASE WHEN ? > 0 THEN 'filled' ELSE status END,
				    updated_at = ?
				WHERE id = ?
			`, math.Max(L.qty, entryFilled), L.filled, atStr, L.id)
			if err != nil {
				return err
			}
			continue
		}
		newQty := entryFilled
		if newQty < L.filled {
			newQty = L.filled
		}
		newRem := newQty - L.filled
		if newRem < domain.PositionEpsilon {
			newRem = 0
		}
		newStatus := string(domain.PendingStatusPending)
		reserved := 0.0
		if entryFilled > domain.PositionEpsilon && newRem > domain.PositionEpsilon {
			newStatus = string(domain.PendingStatusOpen)
			reserved = newRem
		} else if entryFilled > domain.PositionEpsilon && newRem <= domain.PositionEpsilon && L.filled > domain.PositionEpsilon {
			newStatus = string(domain.PendingStatusFilled)
		} else if entryFilled <= domain.PositionEpsilon {
			newStatus = string(domain.PendingStatusPending)
			newQty = 0
			newRem = 0
		}
		_, err = s.db.ExecContext(ctx, `
			UPDATE pending_orders
			SET quantity = ?, remaining_quantity = ?, reserved_quantity = ?, reserved_cash = 0,
			    status = ?, updated_at = ?
			WHERE id = ?
		`, newQty, newRem, reserved, newStatus, atStr, L.id)
		if err != nil {
			return err
		}
	}
	return nil
}

// CancelBracket cancels all open or pending legs of a bracket.
func (s *SQLite) CancelBracket(ctx context.Context, clientID, bracketID string, at time.Time, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if reason == "" {
		reason = domain.CancelReasonBracketEntry
	}
	atStr := at.UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		UPDATE pending_orders
		SET status = 'canceled', canceled_at = ?, updated_at = ?,
		    reserved_cash = 0, reserved_quantity = 0, cancel_reason = ?
		WHERE client_id = ? AND bracket_id = ? AND status IN ('open', 'pending')
	`, atStr, atStr, reason, clientID, bracketID)
	return err
}

// GetPendingOrder returns one order or ErrNotFound.
func (s *SQLite) GetPendingOrder(ctx context.Context, clientID, id string) (*domain.PendingOrder, error) {
	row := s.db.QueryRowContext(ctx, pendingOrderSelect+` WHERE client_id = ? AND id = ?`, clientID, id)
	o, err := scanPendingOrder(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return o, nil
}

// AmendPendingOrder updates an open order in place when remaining+trigger still match the CAS snapshot.
func (s *SQLite) AmendPendingOrder(ctx context.Context, clientID, id string, a domain.PendingOrderAmend) (*domain.PendingOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at := a.At; at.IsZero() {
		a.At = time.Now().UTC()
	}
	atStr := a.At.UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `
		UPDATE pending_orders
		SET remaining_quantity = ?, trigger_price = ?, quantity = ?,
		    reserved_cash = ?, reserved_quantity = ?, updated_at = ?
		WHERE id = ? AND client_id = ? AND status = 'open'
		  AND ABS(remaining_quantity - ?) < 1e-9
		  AND ABS(trigger_price - ?) < 1e-9
	`, a.RemainingQuantity, a.TriggerPrice, a.Quantity, a.ReservedCash, a.ReservedQuantity, atStr,
		id, clientID, a.ExpectedRemaining, a.ExpectedTrigger)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		cur, gerr := scanPendingOrder(s.db.QueryRowContext(ctx, pendingOrderSelect+` WHERE client_id = ? AND id = ?`, clientID, id))
		if gerr == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		if gerr != nil {
			return nil, gerr
		}
		if cur.Status != domain.PendingStatusOpen {
			return nil, fmt.Errorf("%w: order is no longer open", domain.ErrConflict)
		}
		return nil, fmt.Errorf("%w: order changed; refetch and retry", domain.ErrConflict)
	}
	return scanPendingOrder(s.db.QueryRowContext(ctx, pendingOrderSelect+` WHERE client_id = ? AND id = ?`, clientID, id))
}

// ListPendingOrders lists client orders, optional status filter.
func (s *SQLite) ListPendingOrders(ctx context.Context, clientID string, status domain.PendingOrderStatus, limit, offset int) ([]domain.PendingOrder, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var rows *sql.Rows
	var err error
	if status == "" {
		rows, err = s.db.QueryContext(ctx, pendingOrderSelect+`
			WHERE client_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, clientID, limit, offset)
	} else {
		rows, err = s.db.QueryContext(ctx, pendingOrderSelect+`
			WHERE client_id = ? AND status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, clientID, string(status), limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPendingOrders(rows)
}

// CountOpenPendingOrders counts open orders for a client.
func (s *SQLite) CountOpenPendingOrders(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pending_orders WHERE client_id = ? AND status = 'open'
	`, clientID).Scan(&n)
	return n, err
}

// SumReservedCash totals reserved cash on open buy orders.
func (s *SQLite) SumReservedCash(ctx context.Context, clientID string) (float64, error) {
	var n sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(reserved_cash), 0) FROM pending_orders
		WHERE client_id = ? AND status = 'open' AND reserved_cash > 0
	`, clientID).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n.Float64, nil
}

// SumReservedQuantity totals reserved sell qty for a symbol.
// Standalone orders contribute their reserved_quantity; each OCO group contributes
// max(remaining) once so take-profit + stop-loss do not double-lock the position.
func (s *SQLite) SumReservedQuantity(ctx context.Context, clientID string, exchange domain.Exchange, symbol string) (float64, error) {
	var standalone sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(reserved_quantity), 0) FROM pending_orders
		WHERE client_id = ? AND status = 'open' AND exchange = ? AND symbol = ?
		  AND reserved_quantity > 0
		  AND (oco_group_id IS NULL OR oco_group_id = '')
	`, clientID, string(exchange), symbol).Scan(&standalone)
	if err != nil {
		return 0, err
	}
	var oco sql.NullFloat64
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(gmax), 0) FROM (
			SELECT MAX(remaining_quantity) AS gmax FROM pending_orders
			WHERE client_id = ? AND status = 'open' AND exchange = ? AND symbol = ?
			  AND oco_group_id IS NOT NULL AND oco_group_id != ''
			GROUP BY oco_group_id
		)
	`, clientID, string(exchange), symbol).Scan(&oco)
	if err != nil {
		return 0, err
	}
	return standalone.Float64 + oco.Float64, nil
}

// ListAllOpenPendingOrders returns all open orders for the background filler.
func (s *SQLite) ListAllOpenPendingOrders(ctx context.Context) ([]domain.PendingOrder, error) {
	rows, err := s.db.QueryContext(ctx, pendingOrderSelect+` WHERE status = 'open' ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPendingOrders(rows)
}

// CancelPendingOrder cancels only if still open and releases remaining reservation.
// If the order is an OCO leg and the user cancels it, the peer leg is canceled too.
func (s *SQLite) CancelPendingOrder(ctx context.Context, clientID, id string, at time.Time, reason string) (*domain.PendingOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if reason == "" {
		reason = domain.CancelReasonUser
	}
	atStr := at.UTC().Format(time.RFC3339Nano)
	// Load open row for OCO peer linkage.
	var ocoGroup, ocoPeer string
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(oco_group_id,''), COALESCE(oco_peer_id,'') FROM pending_orders
		WHERE id = ? AND client_id = ? AND status = 'open'
	`, id, clientID).Scan(&ocoGroup, &ocoPeer)

	res, err := s.db.ExecContext(ctx, `
		UPDATE pending_orders
		SET status = 'canceled', canceled_at = ?, updated_at = ?,
		    reserved_cash = 0, reserved_quantity = 0, cancel_reason = ?
		WHERE id = ? AND client_id = ? AND status = 'open'
	`, atStr, atStr, reason, id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	// User cancel of one OCO leg cancels the peer (shared size).
	if ocoGroup != "" && ocoPeer != "" && reason == domain.CancelReasonUser {
		_, _ = s.db.ExecContext(ctx, `
			UPDATE pending_orders
			SET status = 'canceled', canceled_at = ?, updated_at = ?,
			    reserved_cash = 0, reserved_quantity = 0, cancel_reason = ?
			WHERE id = ? AND client_id = ? AND status = 'open'
		`, atStr, atStr, domain.CancelReasonOCOGroup, ocoPeer, clientID)
	}
	// Canceling bracket entry (user/expiry/IOC/…) cancels pending/open exits.
	var bracketID, bracketRole string
	_ = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(bracket_id,''), COALESCE(bracket_role,'') FROM pending_orders WHERE id = ? AND client_id = ?
	`, id, clientID).Scan(&bracketID, &bracketRole)
	if bracketID != "" && bracketRole == domain.BracketRoleEntry {
		exitReason := domain.CancelReasonBracketEntry
		if reason != domain.CancelReasonUser {
			exitReason = reason
		}
		_, _ = s.db.ExecContext(ctx, `
			UPDATE pending_orders
			SET status = 'canceled', canceled_at = ?, updated_at = ?,
			    reserved_cash = 0, reserved_quantity = 0, cancel_reason = ?
			WHERE client_id = ? AND bracket_id = ? AND id != ? AND status IN ('open', 'pending')
		`, atStr, atStr, exitReason, clientID, bracketID, id)
	}
	row := s.db.QueryRowContext(ctx, pendingOrderSelect+` WHERE id = ? AND client_id = ?`, id, clientID)
	return scanPendingOrder(row)
}

// CancelOpenPendingOrders cancels matching open/pending orders in one transaction and releases reservations.
func (s *SQLite) CancelOpenPendingOrders(ctx context.Context, clientID string, exchange domain.Exchange, symbol string, at time.Time, reason string) ([]domain.PendingOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if reason == "" {
		reason = domain.CancelReasonUser
	}
	atStr := at.UTC().Format(time.RFC3339Nano)

	where := `client_id = ? AND status IN ('open', 'pending')`
	args := []any{clientID}
	if symbol != "" {
		where += ` AND exchange = ? AND symbol = ?`
		args = append(args, string(exchange), symbol)
	} else if exchange != "" {
		where += ` AND exchange = ?`
		args = append(args, string(exchange))
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, pendingOrderSelect+` WHERE `+where+` ORDER BY created_at ASC`, args...)
	if err != nil {
		return nil, err
	}
	list, err := scanPendingOrders(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return []domain.PendingOrder{}, nil
	}

	placeholders := make([]string, 0, len(list))
	updArgs := []any{atStr, atStr, reason}
	for i := range list {
		placeholders = append(placeholders, "?")
		updArgs = append(updArgs, list[i].ID)
	}
	q := fmt.Sprintf(`
		UPDATE pending_orders
		SET status = 'canceled', canceled_at = ?, updated_at = ?,
		    reserved_cash = 0, reserved_quantity = 0, cancel_reason = ?
		WHERE id IN (%s) AND status IN ('open', 'pending')
	`, strings.Join(placeholders, ","))
	if _, err := tx.ExecContext(ctx, q, updArgs...); err != nil {
		return nil, err
	}

	selArgs := make([]any, 0, len(list))
	for i := range list {
		selArgs = append(selArgs, list[i].ID)
	}
	rows, err = tx.QueryContext(ctx, pendingOrderSelect+`
		WHERE id IN (`+strings.Join(placeholders, ",")+`) AND status = 'canceled'
		ORDER BY created_at ASC`, selArgs...)
	if err != nil {
		return nil, err
	}
	out, err := scanPendingOrders(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdatePendingTrail ratchets peak and stop for an open trailing_stop only when peak rises.
func (s *SQLite) UpdatePendingTrail(ctx context.Context, id string, newPeak, newTrigger float64, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	atStr := at.UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `
		UPDATE pending_orders
		SET trail_peak = ?, trigger_price = ?, updated_at = ?
		WHERE id = ? AND status = 'open' AND order_type = 'trailing_stop'
		  AND trail_peak <= ?
	`, newPeak, newTrigger, atStr, id, newPeak)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// CancelOCOGroup cancels all open legs in the group.
func (s *SQLite) CancelOCOGroup(ctx context.Context, clientID, groupID string, at time.Time, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if reason == "" {
		reason = domain.CancelReasonOCOGroup
	}
	atStr := at.UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		UPDATE pending_orders
		SET status = 'canceled', canceled_at = ?, updated_at = ?,
		    reserved_cash = 0, reserved_quantity = 0, cancel_reason = ?
		WHERE client_id = ? AND oco_group_id = ? AND status = 'open'
	`, atStr, atStr, reason, clientID, groupID)
	return err
}

// ExecutePendingFill applies a partial or full fill for an open order.
// order must contain updated filled/remaining/reserved/status fields after the fill.
func (s *SQLite) ExecutePendingFill(ctx context.Context, order *domain.PendingOrder, p *domain.Portfolio, pos *domain.Position, t domain.Trade, at time.Time, lots *domain.LotOps) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if order == nil {
		return fmt.Errorf("pending order required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
		t.CreatedAt = at
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var remaining float64
	err = tx.QueryRowContext(ctx, `
		SELECT status, remaining_quantity FROM pending_orders WHERE id = ?
	`, order.ID).Scan(&status, &remaining)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != string(domain.PendingStatusOpen) || remaining <= domain.PositionEpsilon {
		return domain.ErrNotFound
	}
	if t.Quantity > remaining+1e-9 {
		return fmt.Errorf("%w: fill quantity exceeds remaining", domain.ErrInvalidArgument)
	}

	atStr := at.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, realized_pnl_total = ?, updated_at = ?
		WHERE id = ?
	`, p.CashBalance, p.RealizedPnLTotal, atStr, p.BookID()); err != nil {
		return err
	}

	if pos != nil {
		if pos.Quantity <= domain.PositionEpsilon {
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM positions WHERE client_id = ? AND exchange = ? AND symbol = ?
			`, pos.ClientID, string(pos.Exchange), pos.Symbol); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO positions (client_id, exchange, symbol, quantity, avg_cost, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT(client_id, exchange, symbol) DO UPDATE SET
					quantity = excluded.quantity,
					avg_cost = excluded.avg_cost,
					updated_at = excluded.updated_at
			`, pos.ClientID, string(pos.Exchange), pos.Symbol, pos.Quantity, pos.AvgCost, atStr); err != nil {
				return err
			}
		}
	}

	if t.PendingOrderID == "" {
		t.PendingOrderID = order.ID
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id, lot_method, fee, last_price, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.PendingOrderID, string(t.LotMethod), t.Fee, t.LastPrice, atStr); err != nil {
		return err
	}
	if err := applyLotOps(ctx, tx, lots, t.ID, at); err != nil {
		return err
	}

	var filledAt any
	if order.Status == domain.PendingStatusFilled {
		filledAt = atStr
	} else {
		filledAt = nil
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE pending_orders
		SET filled_quantity = ?, remaining_quantity = ?, reserved_cash = ?, reserved_quantity = ?,
		    status = ?, updated_at = ?, fill_trade_id = ?, fill_price = ?, filled_at = COALESCE(?, filled_at)
		WHERE id = ? AND status = 'open'
	`, order.FilledQuantity, order.RemainingQuantity, order.ReservedCash, order.ReservedQuantity,
		string(order.Status), atStr, t.ID, t.Price, filledAt, order.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

// ExecuteOCOFill fills one OCO leg and syncs or cancels the peer in the same transaction.
func (s *SQLite) ExecuteOCOFill(ctx context.Context, filled *domain.PendingOrder, peer *domain.PendingOrder, p *domain.Portfolio, pos *domain.Position, t domain.Trade, at time.Time, lots *domain.LotOps) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if filled == nil {
		return fmt.Errorf("filled order required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
		t.CreatedAt = at
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var remaining float64
	err = tx.QueryRowContext(ctx, `
		SELECT status, remaining_quantity FROM pending_orders WHERE id = ?
	`, filled.ID).Scan(&status, &remaining)
	if err == sql.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != string(domain.PendingStatusOpen) || remaining <= domain.PositionEpsilon {
		return domain.ErrNotFound
	}
	if t.Quantity > remaining+1e-9 {
		return fmt.Errorf("%w: fill quantity exceeds remaining", domain.ErrInvalidArgument)
	}

	atStr := at.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, realized_pnl_total = ?, updated_at = ?
		WHERE id = ?
	`, p.CashBalance, p.RealizedPnLTotal, atStr, p.BookID()); err != nil {
		return err
	}
	if pos != nil {
		if pos.Quantity <= domain.PositionEpsilon {
			if _, err := tx.ExecContext(ctx, `
				DELETE FROM positions WHERE client_id = ? AND exchange = ? AND symbol = ?
			`, pos.ClientID, string(pos.Exchange), pos.Symbol); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO positions (client_id, exchange, symbol, quantity, avg_cost, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT(client_id, exchange, symbol) DO UPDATE SET
					quantity = excluded.quantity,
					avg_cost = excluded.avg_cost,
					updated_at = excluded.updated_at
			`, pos.ClientID, string(pos.Exchange), pos.Symbol, pos.Quantity, pos.AvgCost, atStr); err != nil {
				return err
			}
		}
	}
	if t.PendingOrderID == "" {
		t.PendingOrderID = filled.ID
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id, lot_method, fee, last_price, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.PendingOrderID, string(t.LotMethod), t.Fee, t.LastPrice, atStr); err != nil {
		return err
	}
	if err := applyLotOps(ctx, tx, lots, t.ID, at); err != nil {
		return err
	}

	var filledAt any
	if filled.Status == domain.PendingStatusFilled {
		filledAt = atStr
	} else {
		filledAt = nil
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE pending_orders
		SET filled_quantity = ?, remaining_quantity = ?, reserved_cash = ?, reserved_quantity = ?,
		    status = ?, updated_at = ?, fill_trade_id = ?, fill_price = ?, filled_at = COALESCE(?, filled_at)
		WHERE id = ? AND status = 'open'
	`, filled.FilledQuantity, filled.RemainingQuantity, filled.ReservedCash, filled.ReservedQuantity,
		string(filled.Status), atStr, t.ID, t.Price, filledAt, filled.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}

	// Peer: cancel if this leg fully filled; otherwise shrink remaining to match (no second cash move).
	if peer != nil && peer.ID != "" {
		if filled.Status == domain.PendingStatusFilled || filled.RemainingQuantity <= domain.PositionEpsilon {
			_, err = tx.ExecContext(ctx, `
				UPDATE pending_orders
				SET status = 'canceled', canceled_at = ?, updated_at = ?,
				    reserved_cash = 0, reserved_quantity = 0, cancel_reason = ?
				WHERE id = ? AND status = 'open'
			`, atStr, atStr, domain.CancelReasonOCOPeerFilled, peer.ID)
			if err != nil {
				return err
			}
		} else {
			// Align peer remaining/filled so filled+remaining = quantity; reserved = remaining.
			peerRem := filled.RemainingQuantity
			peerFilled := peer.Quantity - peerRem
			if peerFilled < 0 {
				peerFilled = 0
			}
			_, err = tx.ExecContext(ctx, `
				UPDATE pending_orders
				SET filled_quantity = ?, remaining_quantity = ?, reserved_quantity = ?,
				    reserved_cash = 0, updated_at = ?
				WHERE id = ? AND status = 'open'
			`, peerFilled, peerRem, peerRem, atStr, peer.ID)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// RejectPendingOrder marks open order rejected and releases reservation.
func (s *SQLite) RejectPendingOrder(ctx context.Context, orderID, reason string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	atStr := at.UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `
		UPDATE pending_orders
		SET status = 'rejected', reject_reason = ?, updated_at = ?,
		    reserved_cash = 0, reserved_quantity = 0
		WHERE id = ? AND status = 'open'
	`, reason, atStr, orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const pendingOrderSelect = `
	SELECT id, client_id, exchange, symbol, order_type, side, quantity, filled_quantity, remaining_quantity,
		trigger_price, reserved_cash, reserved_quantity, time_in_force, expires_at, status,
		created_at, updated_at, filled_at, canceled_at, fill_trade_id, fill_price, reject_reason, cancel_reason,
		COALESCE(oco_group_id, ''), COALESCE(oco_peer_id, ''),
		COALESCE(trail_type, ''), COALESCE(trail_value, 0), COALESCE(trail_peak, 0),
		COALESCE(bracket_id, ''), COALESCE(bracket_role, ''), COALESCE(lot_method, 'fifo')
	FROM pending_orders`

type scannable interface {
	Scan(dest ...any) error
}

func scanPendingOrder(row scannable) (*domain.PendingOrder, error) {
	var o domain.PendingOrder
	var ex, typ, side, tif, st, cAt, uAt, lotMethod string
	var filledAt, canceledAt, expiresAt sql.NullString
	if err := row.Scan(
		&o.ID, &o.ClientID, &ex, &o.Symbol, &typ, &side, &o.Quantity, &o.FilledQuantity, &o.RemainingQuantity,
		&o.TriggerPrice, &o.ReservedCash, &o.ReservedQuantity, &tif, &expiresAt, &st,
		&cAt, &uAt, &filledAt, &canceledAt, &o.FillTradeID, &o.FillPrice, &o.RejectReason, &o.CancelReason,
		&o.OCOGroupID, &o.OCOPeerID, &o.TrailType, &o.TrailValue, &o.TrailPeak,
		&o.BracketID, &o.BracketRole, &lotMethod,
	); err != nil {
		return nil, err
	}
	o.LotMethod, _ = domain.NormalizeLotMethod(lotMethod)
	o.Exchange = domain.Exchange(ex)
	o.Type = domain.PendingOrderType(typ)
	o.Side = domain.TradeSide(side)
	o.TimeInForce = domain.TimeInForce(tif)
	if o.TimeInForce == "" {
		o.TimeInForce = domain.TimeInForceGTC
	}
	o.Status = domain.PendingOrderStatus(st)
	o.CreatedAt = parseTime(cAt)
	o.UpdatedAt = parseTime(uAt)
	if expiresAt.Valid && expiresAt.String != "" {
		t := parseTime(expiresAt.String)
		o.ExpiresAt = &t
	}
	if filledAt.Valid && filledAt.String != "" {
		t := parseTime(filledAt.String)
		o.FilledAt = &t
	}
	if canceledAt.Valid && canceledAt.String != "" {
		t := parseTime(canceledAt.String)
		o.CanceledAt = &t
	}
	return &o, nil
}

func scanPendingOrders(rows *sql.Rows) ([]domain.PendingOrder, error) {
	out := make([]domain.PendingOrder, 0)
	for rows.Next() {
		o, err := scanPendingOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
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

const recurringPlanCols = `id, client_id, exchange, symbol, name, amount, frequency, weekday, day_of_month, interval_hours, status,
	next_run_at, last_run_at, last_period_key, created_at, updated_at`

func scanRecurringPlan(row scannable) (*domain.RecurringBuyPlan, error) {
	var p domain.RecurringBuyPlan
	var ex, freq, st, next, cAt, uAt string
	var lastRun sql.NullString
	if err := row.Scan(
		&p.ID, &p.ClientID, &ex, &p.Symbol, &p.Name, &p.Amount, &freq, &p.Weekday, &p.DayOfMonth, &p.IntervalHours, &st,
		&next, &lastRun, &p.LastPeriodKey, &cAt, &uAt,
	); err != nil {
		return nil, err
	}
	p.Exchange = domain.Exchange(ex)
	p.Frequency = domain.RecurringBuyFrequency(freq)
	p.Status = domain.RecurringBuyPlanStatus(st)
	p.NextRunAt = parseTime(next)
	p.CreatedAt = parseTime(cAt)
	p.UpdatedAt = parseTime(uAt)
	if lastRun.Valid && lastRun.String != "" {
		t := parseTime(lastRun.String)
		p.LastRunAt = &t
	}
	return &p, nil
}

// CreateRecurringBuyPlan inserts a plan.
func (s *SQLite) CreateRecurringBuyPlan(ctx context.Context, p domain.RecurringBuyPlan) (*domain.RecurringBuyPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recurring_buy_plans (`+recurringPlanCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.ClientID, string(p.Exchange), p.Symbol, p.Name, p.Amount, string(p.Frequency), p.Weekday, p.DayOfMonth, p.IntervalHours, string(p.Status),
		p.NextRunAt.UTC().Format(time.RFC3339Nano), nullTime(p.LastRunAt), p.LastPeriodKey,
		p.CreatedAt.UTC().Format(time.RFC3339Nano), p.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.GetRecurringBuyPlan(ctx, p.ClientID, p.ID)
}

// GetRecurringBuyPlan returns one plan.
func (s *SQLite) GetRecurringBuyPlan(ctx context.Context, clientID, id string) (*domain.RecurringBuyPlan, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+recurringPlanCols+` FROM recurring_buy_plans WHERE id = ? AND client_id = ?
	`, id, clientID)
	p, err := scanRecurringPlan(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// ListRecurringBuyPlans lists plans for a client.
func (s *SQLite) ListRecurringBuyPlans(ctx context.Context, clientID string) ([]domain.RecurringBuyPlan, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+recurringPlanCols+` FROM recurring_buy_plans
		WHERE client_id = ? ORDER BY created_at DESC
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecurringBuyPlan
	for rows.Next() {
		p, err := scanRecurringPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// CountRecurringBuyPlans counts plans for a client.
func (s *SQLite) CountRecurringBuyPlans(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM recurring_buy_plans WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// UpdateRecurringBuyPlanStatus sets status and next_run_at.
func (s *SQLite) UpdateRecurringBuyPlanStatus(ctx context.Context, clientID, id string, status domain.RecurringBuyPlanStatus, nextRunAt, at time.Time) (*domain.RecurringBuyPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE recurring_buy_plans SET status = ?, next_run_at = ?, updated_at = ?
		WHERE id = ? AND client_id = ?
	`, string(status), nextRunAt.UTC().Format(time.RFC3339Nano), at.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetRecurringBuyPlan(ctx, clientID, id)
}

// UpdateRecurringBuyPlan updates name, amount, and schedule (status unchanged).
func (s *SQLite) UpdateRecurringBuyPlan(ctx context.Context, clientID, id string, p domain.RecurringBuyPlan) (*domain.RecurringBuyPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE recurring_buy_plans
		SET name = ?, amount = ?, frequency = ?, weekday = ?, day_of_month = ?, interval_hours = ?,
		    next_run_at = ?, updated_at = ?
		WHERE id = ? AND client_id = ?
	`, p.Name, p.Amount, string(p.Frequency), p.Weekday, p.DayOfMonth, p.IntervalHours,
		p.NextRunAt.UTC().Format(time.RFC3339Nano), p.UpdatedAt.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetRecurringBuyPlan(ctx, clientID, id)
}

// DeleteRecurringBuyPlan removes a plan and cascaded runs.
func (s *SQLite) DeleteRecurringBuyPlan(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM recurring_buy_plans WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListDueRecurringBuyPlans returns active due plans.
func (s *SQLite) ListDueRecurringBuyPlans(ctx context.Context, now time.Time, limit int) ([]domain.RecurringBuyPlan, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+recurringPlanCols+` FROM recurring_buy_plans
		WHERE status = 'active' AND next_run_at <= ?
		ORDER BY next_run_at ASC LIMIT ?
	`, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecurringBuyPlan
	for rows.Next() {
		p, err := scanRecurringPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ClaimRecurringBuyRun inserts a run; unique (plan_id, period_key) prevents double execution.
func (s *SQLite) ClaimRecurringBuyRun(ctx context.Context, run domain.RecurringBuyRun) (bool, *domain.RecurringBuyRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recurring_buy_runs (
			id, plan_id, client_id, period_key, status, amount, quantity, price,
			trade_id, fail_reason, scheduled_for, executed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.PlanID, run.ClientID, run.PeriodKey, string(run.Status),
		run.Amount, run.Quantity, run.Price, run.TradeID, run.FailReason,
		run.ScheduledFor.UTC().Format(time.RFC3339Nano), run.ExecutedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unique") || strings.Contains(msg, "constraint") {
			return false, nil, nil
		}
		return false, nil, err
	}
	cp := run
	return true, &cp, nil
}

// FinishRecurringBuyRun updates the run and advances the plan.
func (s *SQLite) FinishRecurringBuyRun(ctx context.Context, planID string, run domain.RecurringBuyRun, nextRunAt time.Time, lastPeriodKey string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE recurring_buy_runs SET status = ?, amount = ?, quantity = ?, price = ?,
			trade_id = ?, fail_reason = ?, executed_at = ?
		WHERE id = ?
	`, string(run.Status), run.Amount, run.Quantity, run.Price, run.TradeID, run.FailReason,
		run.ExecutedAt.UTC().Format(time.RFC3339Nano), run.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE recurring_buy_plans SET next_run_at = ?, last_run_at = ?, last_period_key = ?, updated_at = ?
		WHERE id = ?
	`, nextRunAt.UTC().Format(time.RFC3339Nano), at.UTC().Format(time.RFC3339Nano), lastPeriodKey,
		at.UTC().Format(time.RFC3339Nano), planID); err != nil {
		return err
	}
	return tx.Commit()
}

// AdvanceRecurringBuyPlan updates schedule when a period was already claimed.
func (s *SQLite) AdvanceRecurringBuyPlan(ctx context.Context, planID string, nextRunAt time.Time, lastPeriodKey string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		UPDATE recurring_buy_plans SET next_run_at = ?, last_period_key = ?, updated_at = ?
		WHERE id = ?
	`, nextRunAt.UTC().Format(time.RFC3339Nano), lastPeriodKey, at.UTC().Format(time.RFC3339Nano), planID)
	return err
}

// ListRecurringBuyRuns lists runs for a plan.
func (s *SQLite) ListRecurringBuyRuns(ctx context.Context, clientID, planID string, limit, offset int) ([]domain.RecurringBuyRun, error) {
	if limit <= 0 {
		limit = 50
	}
	// Ensure plan ownership
	if _, err := s.GetRecurringBuyPlan(ctx, clientID, planID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, plan_id, client_id, period_key, status, amount, quantity, price,
			trade_id, fail_reason, scheduled_for, executed_at
		FROM recurring_buy_runs WHERE plan_id = ?
		ORDER BY executed_at DESC LIMIT ? OFFSET ?
	`, planID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecurringBuyRun
	for rows.Next() {
		var r domain.RecurringBuyRun
		var st, sched, exec string
		if err := rows.Scan(
			&r.ID, &r.PlanID, &r.ClientID, &r.PeriodKey, &st, &r.Amount, &r.Quantity, &r.Price,
			&r.TradeID, &r.FailReason, &sched, &exec,
		); err != nil {
			return nil, err
		}
		r.Status = domain.RecurringBuyRunStatus(st)
		r.ScheduledFor = parseTime(sched)
		r.ExecutedAt = parseTime(exec)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLite) loadAllocationTargets(ctx context.Context, basketID string) ([]domain.AllocationTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT asset, exchange, weight_pct FROM allocation_targets WHERE basket_id = ? ORDER BY asset ASC
	`, basketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AllocationTarget
	for rows.Next() {
		var t domain.AllocationTarget
		var ex string
		if err := rows.Scan(&t.Asset, &ex, &t.WeightPct); err != nil {
			return nil, err
		}
		t.Exchange = domain.Exchange(ex)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLite) replaceAllocationTargetsTx(ctx context.Context, tx *sql.Tx, basketID string, targets []domain.AllocationTarget) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM allocation_targets WHERE basket_id = ?`, basketID); err != nil {
		return err
	}
	for _, t := range targets {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO allocation_targets (basket_id, asset, exchange, weight_pct) VALUES (?, ?, ?, ?)
		`, basketID, t.Asset, string(t.Exchange), t.WeightPct); err != nil {
			return err
		}
	}
	return nil
}

// CreateAllocationBasket inserts a basket and its targets.
func (s *SQLite) CreateAllocationBasket(ctx context.Context, b domain.AllocationBasket) (*domain.AllocationBasket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = b.CreatedAt
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO allocation_baskets (id, client_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
	`, b.ID, b.ClientID, b.Name, b.CreatedAt.UTC().Format(time.RFC3339Nano), b.UpdatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	if err := s.replaceAllocationTargetsTx(ctx, tx, b.ID, b.Targets); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetAllocationBasket(ctx, b.ClientID, b.ID)
}

// GetAllocationBasket returns one basket with targets.
func (s *SQLite) GetAllocationBasket(ctx context.Context, clientID, id string) (*domain.AllocationBasket, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client_id, name, created_at, updated_at FROM allocation_baskets WHERE id = ? AND client_id = ?
	`, id, clientID)
	var b domain.AllocationBasket
	var cAt, uAt string
	if err := row.Scan(&b.ID, &b.ClientID, &b.Name, &cAt, &uAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	b.CreatedAt = parseTime(cAt)
	b.UpdatedAt = parseTime(uAt)
	tg, err := s.loadAllocationTargets(ctx, b.ID)
	if err != nil {
		return nil, err
	}
	b.Targets = tg
	return &b, nil
}

// ListAllocationBaskets lists baskets for a client.
func (s *SQLite) ListAllocationBaskets(ctx context.Context, clientID string) ([]domain.AllocationBasket, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, name, created_at, updated_at FROM allocation_baskets
		WHERE client_id = ? ORDER BY created_at DESC
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AllocationBasket
	for rows.Next() {
		var b domain.AllocationBasket
		var cAt, uAt string
		if err := rows.Scan(&b.ID, &b.ClientID, &b.Name, &cAt, &uAt); err != nil {
			return nil, err
		}
		b.CreatedAt = parseTime(cAt)
		b.UpdatedAt = parseTime(uAt)
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		tg, err := s.loadAllocationTargets(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Targets = tg
	}
	return out, nil
}

// CountAllocationBaskets counts baskets for a client.
func (s *SQLite) CountAllocationBaskets(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM allocation_baskets WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// UpdateAllocationBasket replaces name and targets.
func (s *SQLite) UpdateAllocationBasket(ctx context.Context, clientID, id string, b domain.AllocationBasket) (*domain.AllocationBasket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = time.Now().UTC()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
		UPDATE allocation_baskets SET name = ?, updated_at = ? WHERE id = ? AND client_id = ?
	`, b.Name, b.UpdatedAt.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	if err := s.replaceAllocationTargetsTx(ctx, tx, id, b.Targets); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetAllocationBasket(ctx, clientID, id)
}

// DeleteAllocationBasket removes a basket and its targets.
func (s *SQLite) DeleteAllocationBasket(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM allocation_baskets WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanRiskLimits(row scannable) (*domain.RiskLimits, error) {
	var lim domain.RiskLimits
	var loss, weight sql.NullFloat64
	var updated string
	if err := row.Scan(&lim.ClientID, &loss, &weight, &lim.DayKey, &lim.DayStartEquity, &updated); err != nil {
		return nil, err
	}
	if loss.Valid {
		v := loss.Float64
		lim.MaxDailyLossPct = &v
	}
	if weight.Valid {
		v := weight.Float64
		lim.MaxAssetWeightPct = &v
	}
	lim.UpdatedAt = parseTime(updated)
	return &lim, nil
}

func nullFloatPtr(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// GetRiskLimits returns saved limits or ErrNotFound.
func (s *SQLite) GetRiskLimits(ctx context.Context, clientID string) (*domain.RiskLimits, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT client_id, max_daily_loss_pct, max_asset_weight_pct, day_key, day_start_equity, updated_at
		FROM risk_limits WHERE client_id = ?
	`, clientID)
	lim, err := scanRiskLimits(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return lim, err
}

// UpsertRiskLimits inserts or updates the single risk-limits row for a client.
func (s *SQLite) UpsertRiskLimits(ctx context.Context, lim domain.RiskLimits) (*domain.RiskLimits, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lim.UpdatedAt.IsZero() {
		lim.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO risk_limits (client_id, max_daily_loss_pct, max_asset_weight_pct, day_key, day_start_equity, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET
			max_daily_loss_pct = excluded.max_daily_loss_pct,
			max_asset_weight_pct = excluded.max_asset_weight_pct,
			day_key = excluded.day_key,
			day_start_equity = excluded.day_start_equity,
			updated_at = excluded.updated_at
	`, lim.ClientID, nullFloatPtr(lim.MaxDailyLossPct), nullFloatPtr(lim.MaxAssetWeightPct),
		lim.DayKey, lim.DayStartEquity, lim.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.GetRiskLimits(ctx, lim.ClientID)
}

// DeleteRiskLimits removes all risk rules for a client.
func (s *SQLite) DeleteRiskLimits(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM risk_limits WHERE client_id = ?`, clientID)
	return err
}