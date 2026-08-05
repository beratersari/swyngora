package portfoliostore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
CREATE TABLE IF NOT EXISTS portfolios (
	client_id         TEXT PRIMARY KEY NOT NULL,
	currency          TEXT NOT NULL,
	starting_balance  REAL NOT NULL,
	cash_balance      REAL NOT NULL,
	realized_pnl_total REAL NOT NULL DEFAULT 0,
	created_at        TEXT NOT NULL,
	updated_at        TEXT NOT NULL
);
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
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_pending_orders_client ON pending_orders(client_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pending_orders_open ON pending_orders(status) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS recurring_buy_plans (
	id              TEXT PRIMARY KEY NOT NULL,
	client_id       TEXT NOT NULL,
	exchange        TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	amount          REAL NOT NULL,
	frequency       TEXT NOT NULL,
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
	quantity          REAL NOT NULL,
	entry_price       REAL NOT NULL,
	leverage          INTEGER NOT NULL,
	margin            REAL NOT NULL,
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
	realized_pnl  REAL NOT NULL DEFAULT 0,
	margin_delta  REAL NOT NULL DEFAULT 0,
	leverage      INTEGER NOT NULL DEFAULT 1,
	created_at    TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_margin_trades_client ON margin_trades(client_id, created_at DESC);
`
	if _, err := s.db.Exec(schema); err != nil {
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
	}
	for _, q := range alters {
		if _, err := s.db.Exec(q); err != nil {
			// SQLite returns error if column already exists — ignore those.
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				// modernc may say "duplicate column name"
				if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
					// continue only on duplicate; other errors are fatal
					msg := strings.ToLower(err.Error())
					if strings.Contains(msg, "duplicate") {
						continue
					}
					return err
				}
			}
		}
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

// GetPortfolio returns portfolio or ErrNotFound.
func (s *SQLite) GetPortfolio(ctx context.Context, clientID string) (*domain.Portfolio, error) {
	var p domain.Portfolio
	var cAt, uAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT client_id, currency, starting_balance, cash_balance, realized_pnl_total, created_at, updated_at
		FROM portfolios WHERE client_id = ?
	`, clientID).Scan(&p.ClientID, &p.Currency, &p.StartingBalance, &p.CashBalance, &p.RealizedPnLTotal, &cAt, &uAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt = parseTime(cAt)
	p.UpdatedAt = parseTime(uAt)
	return &p, nil
}

// CreatePortfolio inserts a new portfolio.
func (s *SQLite) CreatePortfolio(ctx context.Context, p domain.Portfolio) (*domain.Portfolio, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO portfolios (client_id, currency, starting_balance, cash_balance, realized_pnl_total, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, p.ClientID, p.Currency, p.StartingBalance, p.CashBalance, p.RealizedPnLTotal,
		p.CreatedAt.UTC().Format(time.RFC3339Nano), p.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("portfolio create: %w", err)
	}
	cp := p
	return &cp, nil
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
		WHERE client_id = ?
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
		SELECT id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id, created_at
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
		var ex, side, cAt string
		if err := rows.Scan(&t.ID, &t.ClientID, &ex, &t.Symbol, &side, &t.Quantity, &t.Price, &t.Notional, &t.RealizedPnL, &t.PendingOrderID, &cAt); err != nil {
			return nil, err
		}
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
func (s *SQLite) ExecuteTrade(ctx context.Context, p *domain.Portfolio, pos *domain.Position, t domain.Trade) error {
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

	if _, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, realized_pnl_total = ?, updated_at = ?
		WHERE client_id = ?
	`, p.CashBalance, p.RealizedPnLTotal, at.UTC().Format(time.RFC3339Nano), p.ClientID); err != nil {
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
		INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.PendingOrderID,
		at.UTC().Format(time.RFC3339Nano)); err != nil {
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
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pending_orders (
			id, client_id, exchange, symbol, order_type, side, quantity, filled_quantity, remaining_quantity,
			trigger_price, reserved_cash, reserved_quantity, time_in_force, expires_at, status,
			created_at, updated_at, filled_at, canceled_at, fill_trade_id, fill_price, reject_reason, cancel_reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, o.ID, o.ClientID, string(o.Exchange), o.Symbol, string(o.Type), string(o.Side), o.Quantity, o.FilledQuantity, o.RemainingQuantity,
		o.TriggerPrice, o.ReservedCash, o.ReservedQuantity, string(o.TimeInForce), nullTime(o.ExpiresAt), string(o.Status),
		o.CreatedAt.UTC().Format(time.RFC3339Nano), o.UpdatedAt.UTC().Format(time.RFC3339Nano),
		nullTime(o.FilledAt), nullTime(o.CanceledAt), o.FillTradeID, o.FillPrice, o.RejectReason, o.CancelReason)
	if err != nil {
		return nil, fmt.Errorf("pending order create: %w", err)
	}
	cp := o
	return &cp, nil
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
func (s *SQLite) SumReservedQuantity(ctx context.Context, clientID string, exchange domain.Exchange, symbol string) (float64, error) {
	var n sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(reserved_quantity), 0) FROM pending_orders
		WHERE client_id = ? AND status = 'open' AND exchange = ? AND symbol = ? AND reserved_quantity > 0
	`, clientID, string(exchange), symbol).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n.Float64, nil
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
	row := s.db.QueryRowContext(ctx, pendingOrderSelect+` WHERE id = ? AND client_id = ?`, id, clientID)
	return scanPendingOrder(row)
}

// ExecutePendingFill applies a partial or full fill for an open order.
// order must contain updated filled/remaining/reserved/status fields after the fill.
func (s *SQLite) ExecutePendingFill(ctx context.Context, order *domain.PendingOrder, p *domain.Portfolio, pos *domain.Position, t domain.Trade, at time.Time) error {
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
		WHERE client_id = ?
	`, p.CashBalance, p.RealizedPnLTotal, atStr, p.ClientID); err != nil {
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
		INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.PendingOrderID, atStr); err != nil {
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
		created_at, updated_at, filled_at, canceled_at, fill_trade_id, fill_price, reject_reason, cancel_reason
	FROM pending_orders`

type scannable interface {
	Scan(dest ...any) error
}

func scanPendingOrder(row scannable) (*domain.PendingOrder, error) {
	var o domain.PendingOrder
	var ex, typ, side, tif, st, cAt, uAt string
	var filledAt, canceledAt, expiresAt sql.NullString
	if err := row.Scan(
		&o.ID, &o.ClientID, &ex, &o.Symbol, &typ, &side, &o.Quantity, &o.FilledQuantity, &o.RemainingQuantity,
		&o.TriggerPrice, &o.ReservedCash, &o.ReservedQuantity, &tif, &expiresAt, &st,
		&cAt, &uAt, &filledAt, &canceledAt, &o.FillTradeID, &o.FillPrice, &o.RejectReason, &o.CancelReason,
	); err != nil {
		return nil, err
	}
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

const recurringPlanCols = `id, client_id, exchange, symbol, amount, frequency, status,
	next_run_at, last_run_at, last_period_key, created_at, updated_at`

func scanRecurringPlan(row scannable) (*domain.RecurringBuyPlan, error) {
	var p domain.RecurringBuyPlan
	var ex, freq, st, next, cAt, uAt string
	var lastRun sql.NullString
	if err := row.Scan(
		&p.ID, &p.ClientID, &ex, &p.Symbol, &p.Amount, &freq, &st,
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.ClientID, string(p.Exchange), p.Symbol, p.Amount, string(p.Frequency), string(p.Status),
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