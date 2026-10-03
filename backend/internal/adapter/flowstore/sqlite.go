package flowstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/sqliteutil"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists minute buy/sell rows at each stored price.
type SQLite struct {
	db   *sql.DB
	path string
}

// Open opens or creates the flow-at-prices history database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("flow history sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create flow history db dir: %w", err)
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
CREATE TABLE IF NOT EXISTS flow_price_cells (
	exchange  TEXT NOT NULL,
	symbol    TEXT NOT NULL,
	minute_ms INTEGER NOT NULL,
	price_e8  INTEGER NOT NULL,
	buy       REAL NOT NULL,
	sell      REAL NOT NULL,
	buy_n     INTEGER NOT NULL,
	sell_n    INTEGER NOT NULL,
	PRIMARY KEY (exchange, symbol, minute_ms, price_e8)
);
CREATE TABLE IF NOT EXISTS flow_cursors (
	exchange TEXT NOT NULL,
	symbol   TEXT NOT NULL,
	last_ms  INTEGER NOT NULL,
	PRIMARY KEY (exchange, symbol)
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("flow history schema: %w", err)
	}
	return nil
}

// AddFlowCells adds buy/sell size onto existing minute rows.
func (s *SQLite) AddFlowCells(ctx context.Context, cells []domain.FlowAtPricesCell) error {
	if s == nil || s.db == nil || len(cells) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO flow_price_cells (exchange, symbol, minute_ms, price_e8, buy, sell, buy_n, sell_n)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(exchange, symbol, minute_ms, price_e8) DO UPDATE SET
	buy = buy + excluded.buy,
	sell = sell + excluded.sell,
	buy_n = buy_n + excluded.buy_n,
	sell_n = sell_n + excluded.sell_n
`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, c := range cells {
		if c.Price <= 0 || c.Minute.IsZero() || c.Symbol == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx,
			string(c.Exchange), c.Symbol, c.Minute.UTC().UnixMilli(), priceE8(c.Price),
			c.Buy, c.Sell, c.BuyN, c.SellN,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListFlowCells returns rows whose minute overlaps [from, to).
func (s *SQLite) ListFlowCells(ctx context.Context, exchange domain.Exchange, symbol string, from, to time.Time) ([]domain.FlowAtPricesCell, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	rows, err := s.db.QueryContext(ctx, `
SELECT minute_ms, price_e8, buy, sell, buy_n, sell_n
FROM flow_price_cells
WHERE exchange = ? AND symbol = ? AND minute_ms >= ? AND minute_ms < ?
ORDER BY minute_ms, price_e8
`, string(exchange), symbol, from.UTC().Add(-time.Minute).UnixMilli(), to.UTC().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FlowAtPricesCell
	for rows.Next() {
		var ms, px int64
		var c domain.FlowAtPricesCell
		if err := rows.Scan(&ms, &px, &c.Buy, &c.Sell, &c.BuyN, &c.SellN); err != nil {
			return nil, err
		}
		c.Exchange = exchange
		c.Symbol = symbol
		c.Minute = time.UnixMilli(ms).UTC()
		c.Price = float64(px) / 1e8
		out = append(out, c)
	}
	return out, rows.Err()
}

// EarliestFlow is the oldest stored minute for one venue and symbol.
func (s *SQLite) EarliestFlow(ctx context.Context, exchange domain.Exchange, symbol string) (time.Time, bool, error) {
	if s == nil || s.db == nil {
		return time.Time{}, false, nil
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if exchange == "" || symbol == "" {
		return time.Time{}, false, fmt.Errorf("%w: incomplete flow query", domain.ErrInvalidArgument)
	}
	var ms sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
SELECT MIN(minute_ms) FROM flow_price_cells
WHERE exchange = ? AND symbol = ?`, string(exchange), symbol).Scan(&ms)
	if err != nil {
		return time.Time{}, false, err
	}
	if !ms.Valid || ms.Int64 <= 0 {
		return time.Time{}, false, nil
	}
	return time.UnixMilli(ms.Int64).UTC(), true, nil
}

// EarliestFlowSpans is the oldest stored minute for each symbol and venue.
func (s *SQLite) EarliestFlowSpans(ctx context.Context, symbols []string) ([]domain.StoredSince, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	symbols = domain.SpanQuerySymbols(symbols)
	if len(symbols) == 0 {
		return nil, nil
	}
	in, args := sqliteutil.InArgs(symbols)
	rows, err := s.db.QueryContext(ctx, `
SELECT symbol, exchange, MIN(minute_ms) FROM flow_price_cells
WHERE symbol IN (`+in+`) GROUP BY symbol, exchange`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StoredSince
	for rows.Next() {
		var sym, ex string
		var ms sql.NullInt64
		if err := rows.Scan(&sym, &ex, &ms); err != nil {
			return nil, err
		}
		if !ms.Valid || ms.Int64 <= 0 || sym == "" || ex == "" {
			continue
		}
		out = append(out, domain.StoredSince{
			Exchange: domain.Exchange(ex),
			Symbol:   sym,
			Since:    time.UnixMilli(ms.Int64).UTC(),
		})
	}
	return out, rows.Err()
}

// FlowCursor is the newest print time already stored.
func (s *SQLite) FlowCursor(ctx context.Context, exchange domain.Exchange, symbol string) (time.Time, error) {
	if s == nil || s.db == nil {
		return time.Time{}, nil
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	var ms int64
	err := s.db.QueryRowContext(ctx, `
SELECT last_ms FROM flow_cursors WHERE exchange = ? AND symbol = ?
`, string(exchange), symbol).Scan(&ms)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(ms).UTC(), nil
}

// SetFlowCursor records the newest print time stored for this pair.
func (s *SQLite) SetFlowCursor(ctx context.Context, exchange domain.Exchange, symbol string, at time.Time) error {
	if s == nil || s.db == nil || at.IsZero() {
		return nil
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO flow_cursors (exchange, symbol, last_ms) VALUES (?, ?, ?)
ON CONFLICT(exchange, symbol) DO UPDATE SET last_ms = excluded.last_ms
WHERE excluded.last_ms > flow_cursors.last_ms
`, string(exchange), symbol, at.UTC().UnixMilli())
	return err
}

// PurgeFlowCells drops minutes older than cutoff.
func (s *SQLite) PurgeFlowCells(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.db == nil || cutoff.IsZero() {
		return 0, nil
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM flow_price_cells WHERE minute_ms < ?`, cutoff.UTC().UnixMilli())
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func priceE8(price float64) int64 {
	return int64(mathRound(price * 1e8))
}

func mathRound(v float64) float64 {
	if v < 0 {
		return float64(int64(v - 0.5))
	}
	return float64(int64(v + 0.5))
}
