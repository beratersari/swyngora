package portfoliostore

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *SQLite) migrateTaxLots() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS tax_lots (
	id                 TEXT PRIMARY KEY NOT NULL,
	client_id          TEXT NOT NULL,
	exchange           TEXT NOT NULL,
	symbol             TEXT NOT NULL,
	quantity           REAL NOT NULL,
	original_quantity  REAL NOT NULL,
	price              REAL NOT NULL,
	opened_at          TEXT NOT NULL,
	source_trade_id    TEXT NOT NULL DEFAULT '',
	closed_at          TEXT
);
CREATE INDEX IF NOT EXISTS idx_tax_lots_book_sym ON tax_lots(client_id, exchange, symbol, opened_at);
CREATE TABLE IF NOT EXISTS tax_lot_fills (
	id           TEXT PRIMARY KEY NOT NULL,
	trade_id     TEXT NOT NULL,
	lot_id       TEXT NOT NULL,
	quantity     REAL NOT NULL,
	cost_price   REAL NOT NULL,
	sell_price   REAL NOT NULL,
	realized_pnl REAL NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tax_lot_fills_trade ON tax_lot_fills(trade_id);
`
	if _, err := s.db.Exec(ddl); err != nil {
		return err
	}
	// One synthetic lot per existing position that has no open lots.
	rows, err := s.db.Query(`
		SELECT p.client_id, p.exchange, p.symbol, p.quantity, p.avg_cost, p.updated_at
		FROM positions p
		WHERE p.quantity > 1e-12
		  AND NOT EXISTS (
			SELECT 1 FROM tax_lots t
			WHERE t.client_id = p.client_id AND t.exchange = p.exchange AND t.symbol = p.symbol
			  AND t.quantity > 1e-12 AND t.closed_at IS NULL
		  )`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		book, ex, sym, at string
		qty, avg          float64
	}
	var need []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.book, &r.ex, &r.sym, &r.qty, &r.avg, &r.at); err != nil {
			return err
		}
		need = append(need, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range need {
		id := "legacy-" + r.book + "-" + r.ex + "-" + r.sym
		if _, err := s.db.Exec(`
			INSERT OR IGNORE INTO tax_lots (
				id, client_id, exchange, symbol, quantity, original_quantity, price, opened_at, source_trade_id, closed_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', NULL)
		`, id, r.book, r.ex, r.sym, r.qty, r.qty, r.avg, r.at); err != nil {
			return err
		}
	}
	return nil
}

// ListOpenTaxLots returns remaining lots for a pair, oldest first.
func (s *SQLite) ListOpenTaxLots(ctx context.Context, clientID string, exchange domain.Exchange, symbol string) ([]domain.TaxLot, error) {
	return s.ListTaxLots(ctx, clientID, exchange, symbol, true)
}

// ListTaxLots lists lots; openOnly skips closed rows.
func (s *SQLite) ListTaxLots(ctx context.Context, clientID string, exchange domain.Exchange, symbol string, openOnly bool) ([]domain.TaxLot, error) {
	q := `
		SELECT id, client_id, exchange, symbol, quantity, original_quantity, price, opened_at, source_trade_id, closed_at
		FROM tax_lots WHERE client_id = ?`
	args := []any{clientID}
	if strings.TrimSpace(string(exchange)) != "" {
		q += ` AND exchange = ?`
		args = append(args, string(exchange))
	}
	if symbol = strings.TrimSpace(symbol); symbol != "" {
		q += ` AND symbol = ?`
		args = append(args, symbol)
	}
	if openOnly {
		q += ` AND quantity > 1e-12 AND closed_at IS NULL`
	}
	q += ` ORDER BY opened_at ASC, id ASC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.TaxLot, 0)
	for rows.Next() {
		l, err := scanTaxLot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ListTaxLotFillsForTrades returns fill legs for the given trade ids.
func (s *SQLite) ListTaxLotFillsForTrades(ctx context.Context, tradeIDs []string) ([]domain.TaxLotFill, error) {
	if len(tradeIDs) == 0 {
		return nil, nil
	}
	ph := strings.Repeat("?,", len(tradeIDs))
	ph = ph[:len(ph)-1]
	args := make([]any, len(tradeIDs))
	for i, id := range tradeIDs {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, trade_id, lot_id, quantity, cost_price, sell_price, realized_pnl
		FROM tax_lot_fills WHERE trade_id IN (`+ph+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.TaxLotFill, 0)
	for rows.Next() {
		var f domain.TaxLotFill
		if err := rows.Scan(&f.ID, &f.TradeID, &f.LotID, &f.Quantity, &f.CostPrice, &f.SellPrice, &f.RealizedPnL); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// InsertTaxLot inserts one lot (used by tests / rare backfill outside a trade tx).
func (s *SQLite) InsertTaxLot(ctx context.Context, lot domain.TaxLot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, taxLotInsertSQL,
		lot.ID, lot.ClientID, string(lot.Exchange), lot.Symbol, lot.Quantity, lot.OriginalQuantity, lot.Price,
		lot.OpenedAt.UTC().Format(time.RFC3339Nano), lot.SourceTradeID, nullTime(lot.ClosedAt))
	return err
}

const taxLotInsertSQL = `
	INSERT INTO tax_lots (id, client_id, exchange, symbol, quantity, original_quantity, price, opened_at, source_trade_id, closed_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func applyLotOps(ctx context.Context, tx *sql.Tx, ops *domain.LotOps, tradeID string, at time.Time) error {
	if ops == nil {
		return nil
	}
	atStr := at.UTC().Format(time.RFC3339Nano)
	for _, lot := range ops.Created {
		if lot.OpenedAt.IsZero() {
			lot.OpenedAt = at
		}
		if lot.SourceTradeID == "" {
			lot.SourceTradeID = tradeID
		}
		if _, err := tx.ExecContext(ctx, taxLotInsertSQL,
			lot.ID, lot.ClientID, string(lot.Exchange), lot.Symbol, lot.Quantity, lot.OriginalQuantity, lot.Price,
			lot.OpenedAt.UTC().Format(time.RFC3339Nano), lot.SourceTradeID, nullTime(lot.ClosedAt)); err != nil {
			return err
		}
	}
	for _, lot := range ops.Updated {
		if _, err := tx.ExecContext(ctx, `
			UPDATE tax_lots SET quantity = ?, closed_at = ? WHERE id = ?
		`, lot.Quantity, nullTime(lot.ClosedAt), lot.ID); err != nil {
			return err
		}
	}
	for i, f := range ops.Fills {
		if f.ID == "" {
			f.ID = tradeID + "-f" + strconv.Itoa(i)
		}
		if f.TradeID == "" {
			f.TradeID = tradeID
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tax_lot_fills (id, trade_id, lot_id, quantity, cost_price, sell_price, realized_pnl)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, f.ID, f.TradeID, f.LotID, f.Quantity, f.CostPrice, f.SellPrice, f.RealizedPnL); err != nil {
			return err
		}
	}
	_ = atStr
	return nil
}

func scanTaxLot(row scannable) (domain.TaxLot, error) {
	var l domain.TaxLot
	var ex, opened string
	var closed sql.NullString
	err := row.Scan(&l.ID, &l.ClientID, &ex, &l.Symbol, &l.Quantity, &l.OriginalQuantity, &l.Price, &opened, &l.SourceTradeID, &closed)
	if err != nil {
		return l, err
	}
	l.Exchange = domain.Exchange(ex)
	l.OpenedAt = parseTime(opened)
	if closed.Valid && closed.String != "" {
		t := parseTime(closed.String)
		l.ClosedAt = &t
	}
	return l, nil
}
