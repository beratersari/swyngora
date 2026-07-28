package portfoliostore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	id           TEXT PRIMARY KEY NOT NULL,
	client_id    TEXT NOT NULL,
	exchange     TEXT NOT NULL,
	symbol       TEXT NOT NULL,
	side         TEXT NOT NULL,
	quantity     REAL NOT NULL,
	price        REAL NOT NULL,
	notional     REAL NOT NULL,
	realized_pnl REAL NOT NULL DEFAULT 0,
	created_at   TEXT NOT NULL,
	FOREIGN KEY (client_id) REFERENCES portfolios(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_trades_client ON trades(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_positions_client ON positions(client_id);
`
	_, err := s.db.Exec(schema)
	return err
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
		INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL,
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
		SELECT id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, created_at
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
		if err := rows.Scan(&t.ID, &t.ClientID, &ex, &t.Symbol, &side, &t.Quantity, &t.Price, &t.Notional, &t.RealizedPnL, &cAt); err != nil {
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
		INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL,
		at.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
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