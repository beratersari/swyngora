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