package portfoliostore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *SQLite) migrateSpreadOrders() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS spread_orders (
	id               TEXT PRIMARY KEY NOT NULL,
	client_id        TEXT NOT NULL,
	symbol           TEXT NOT NULL,
	side             TEXT NOT NULL,
	quantity         REAL NOT NULL,
	filled_quantity  REAL NOT NULL DEFAULT 0,
	remaining_quantity REAL NOT NULL DEFAULT 0,
	slice_count      INTEGER NOT NULL,
	slices_done      INTEGER NOT NULL DEFAULT 0,
	duration_sec     INTEGER NOT NULL DEFAULT 600,
	next_slice_at    TEXT NOT NULL,
	ends_at          TEXT NOT NULL,
	status           TEXT NOT NULL,
	max_slippage     REAL NOT NULL DEFAULT 0,
	reserved_cash    REAL NOT NULL DEFAULT 0,
	filled_notional  REAL NOT NULL DEFAULT 0,
	filled_fee       REAL NOT NULL DEFAULT 0,
	lot_method       TEXT NOT NULL DEFAULT '',
	reason_set       INTEGER NOT NULL DEFAULT 0,
	reason_json      TEXT NOT NULL DEFAULT '',
	cancel_reason    TEXT NOT NULL DEFAULT '',
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_spread_orders_client ON spread_orders(client_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_spread_orders_open ON spread_orders(status, next_slice_at);
CREATE TABLE IF NOT EXISTS spread_holds (
	order_id TEXT NOT NULL,
	exchange TEXT NOT NULL,
	symbol   TEXT NOT NULL,
	quantity REAL NOT NULL,
	PRIMARY KEY (order_id, exchange, symbol)
);
CREATE INDEX IF NOT EXISTS idx_spread_holds_order ON spread_holds(order_id);
`)
	return err
}

const spreadOrderSelect = `
SELECT id, client_id, symbol, side, quantity, filled_quantity, remaining_quantity,
	slice_count, slices_done, duration_sec, next_slice_at, ends_at, status, max_slippage, reserved_cash,
	filled_notional, filled_fee, lot_method, reason_set, reason_json, cancel_reason, created_at, updated_at
FROM spread_orders `

func (s *SQLite) CreateSpreadOrder(ctx context.Context, o domain.SpreadOrder) (*domain.SpreadOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = o.CreatedAt
	}
	if o.Status == "" {
		o.Status = domain.SpreadStatusOpen
	}
	if o.RemainingQuantity <= 0 {
		o.RemainingQuantity = o.Quantity
	}
	if o.SliceCount <= 0 {
		o.SliceCount = domain.SpreadDefaultSlices
	}
	if o.Duration <= 0 {
		o.Duration = time.Duration(domain.SpreadDefaultMinutes) * time.Minute
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.txInsertIdempotency(ctx, tx); err != nil {
		return nil, err
	}
	if err := insertSpreadOrder(ctx, tx, o); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w", domain.ErrIdempotencyHit)
		}
		return nil, fmt.Errorf("spread order create: %w", err)
	}
	if err := replaceSpreadHolds(ctx, tx, o.ID, o.Holds); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	cp := o
	return &cp, nil
}

func (s *SQLite) GetSpreadOrder(ctx context.Context, clientID, id string) (*domain.SpreadOrder, error) {
	row := s.db.QueryRowContext(ctx, spreadOrderSelect+`WHERE client_id = ? AND id = ?`, clientID, id)
	o, err := scanSpreadOrder(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.attachSpreadHolds(ctx, []*domain.SpreadOrder{o}); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *SQLite) ListSpreadOrders(ctx context.Context, clientID string, status domain.SpreadStatus, limit, offset int) ([]domain.SpreadOrder, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if status == "" {
		rows, err = s.db.QueryContext(ctx, spreadOrderSelect+`
			WHERE client_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, clientID, limit, offset)
	} else {
		rows, err = s.db.QueryContext(ctx, spreadOrderSelect+`
			WHERE client_id = ? AND status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, clientID, string(status), limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanSpreadOrders(ctx, rows)
}

func (s *SQLite) ListOpenSpreadOrders(ctx context.Context) ([]domain.SpreadOrder, error) {
	rows, err := s.db.QueryContext(ctx, spreadOrderSelect+`WHERE status = ? ORDER BY next_slice_at ASC`, string(domain.SpreadStatusOpen))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanSpreadOrders(ctx, rows)
}

func (s *SQLite) SaveSpreadWorking(ctx context.Context, p *domain.Portfolio, o domain.SpreadOrder, fills []domain.RoutedFill) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o.ID == "" {
		return fmt.Errorf("%w: spread order is missing", domain.ErrInvalidArgument)
	}
	if len(fills) > 0 && p == nil {
		return fmt.Errorf("%w: spread fill is missing the book", domain.ErrInvalidArgument)
	}
	at := o.UpdatedAt
	if at.IsZero() {
		at = time.Now().UTC()
	}
	o.UpdatedAt = at
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if len(fills) > 0 {
		if err := applyRoutedFillsTx(ctx, tx, p, fills, at); err != nil {
			return err
		}
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE spread_orders SET
			filled_quantity = ?, remaining_quantity = ?, slices_done = ?, next_slice_at = ?, ends_at = ?,
			status = ?, reserved_cash = ?, filled_notional = ?, filled_fee = ?, cancel_reason = ?, updated_at = ?
		WHERE id = ? AND client_id = ? AND status = ?
	`, o.FilledQuantity, o.RemainingQuantity, o.SlicesDone, formatSpreadTime(o.NextSliceAt), formatSpreadTime(o.EndsAt),
		string(o.Status), o.ReservedCash, o.FilledNotional, o.FilledFee, o.CancelReason, formatSpreadTime(at),
		o.ID, o.ClientID, string(domain.SpreadStatusOpen))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	if err := replaceSpreadHolds(ctx, tx, o.ID, o.Holds); err != nil {
		return err
	}
	return tx.Commit()
}

func applyRoutedFillsTx(ctx context.Context, tx *sql.Tx, p *domain.Portfolio, fills []domain.RoutedFill, at time.Time) error {
	bookAt := p.UpdatedAt
	if bookAt.IsZero() {
		bookAt = at
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, realized_pnl_total = ?, updated_at = ?
		WHERE id = ?
	`, p.CashBalance, p.RealizedPnLTotal, bookAt.UTC().Format(time.RFC3339Nano), p.BookID()); err != nil {
		return err
	}
	for _, fill := range fills {
		tradeAt := fill.Trade.CreatedAt
		if tradeAt.IsZero() {
			tradeAt = at
		}
		pos := fill.Position
		if pos != nil {
			if pos.Quantity <= domain.PositionEpsilon {
				if _, err := tx.ExecContext(ctx, `
					DELETE FROM positions WHERE client_id = ? AND exchange = ? AND symbol = ?
				`, pos.ClientID, string(pos.Exchange), pos.Symbol); err != nil {
					return err
				}
			} else if _, err := tx.ExecContext(ctx, `
				INSERT INTO positions (client_id, exchange, symbol, quantity, avg_cost, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT(client_id, exchange, symbol) DO UPDATE SET
					quantity = excluded.quantity,
					avg_cost = excluded.avg_cost,
					updated_at = excluded.updated_at
			`, pos.ClientID, string(pos.Exchange), pos.Symbol, pos.Quantity, pos.AvgCost,
				tradeAt.UTC().Format(time.RFC3339Nano)); err != nil {
				return err
			}
		}
		if err := execInsertTrade(ctx, tx, fill.Trade, tradeAt); err != nil {
			return err
		}
		if err := applyLotOps(ctx, tx, fill.Lots, fill.Trade.ID, tradeAt); err != nil {
			return err
		}
	}
	return nil
}

func insertSpreadOrder(ctx context.Context, tx *sql.Tx, o domain.SpreadOrder) error {
	reason, err := marshalSpreadReason(o)
	if err != nil {
		return err
	}
	set := 0
	if o.ReasonSet {
		set = 1
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO spread_orders (
			id, client_id, symbol, side, quantity, filled_quantity, remaining_quantity,
			slice_count, slices_done, duration_sec, next_slice_at, ends_at, status, max_slippage, reserved_cash,
			filled_notional, filled_fee, lot_method, reason_set, reason_json, cancel_reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, o.ID, o.ClientID, o.Symbol, string(o.Side), o.Quantity, o.FilledQuantity, o.RemainingQuantity,
		o.SliceCount, o.SlicesDone, int(o.Duration/time.Second), formatSpreadTime(o.NextSliceAt), formatSpreadTime(o.EndsAt), string(o.Status),
		o.MaxSlippage, o.ReservedCash, o.FilledNotional, o.FilledFee, string(o.LotMethod), set, reason, o.CancelReason,
		formatSpreadTime(o.CreatedAt), formatSpreadTime(o.UpdatedAt))
	return err
}

func replaceSpreadHolds(ctx context.Context, tx *sql.Tx, orderID string, holds []domain.SpreadHold) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM spread_holds WHERE order_id = ?`, orderID); err != nil {
		return err
	}
	for _, h := range holds {
		if h.Quantity <= domain.PositionEpsilon || h.Symbol == "" || h.Exchange == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO spread_holds (order_id, exchange, symbol, quantity) VALUES (?, ?, ?, ?)
		`, orderID, string(h.Exchange), h.Symbol, h.Quantity); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLite) scanSpreadOrders(ctx context.Context, rows *sql.Rows) ([]domain.SpreadOrder, error) {
	var list []domain.SpreadOrder
	ptrs := make([]*domain.SpreadOrder, 0)
	for rows.Next() {
		o, err := scanSpreadOrder(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []domain.SpreadOrder{}, nil
	}
	for i := range list {
		ptrs = append(ptrs, &list[i])
	}
	if err := s.attachSpreadHolds(ctx, ptrs); err != nil {
		return nil, err
	}
	return list, nil
}

type spreadScanner interface {
	Scan(dest ...any) error
}

func scanSpreadOrder(row spreadScanner) (*domain.SpreadOrder, error) {
	var (
		o        domain.SpreadOrder
		side     string
		status   string
		nextAt   string
		endsAt   string
		lot      string
		reason   string
		set      int
		created  string
		updated  string
		duration int
	)
	err := row.Scan(
		&o.ID, &o.ClientID, &o.Symbol, &side, &o.Quantity, &o.FilledQuantity, &o.RemainingQuantity,
		&o.SliceCount, &o.SlicesDone, &duration, &nextAt, &endsAt, &status, &o.MaxSlippage, &o.ReservedCash,
		&o.FilledNotional, &o.FilledFee, &lot, &set, &reason, &o.CancelReason, &created, &updated,
	)
	if err != nil {
		return nil, err
	}
	o.Side = domain.TradeSide(side)
	o.Status = domain.SpreadStatus(status)
	o.LotMethod = domain.LotMethod(lot)
	o.ReasonSet = set != 0
	o.Duration = time.Duration(duration) * time.Second
	o.NextSliceAt = parseSpreadTime(nextAt)
	o.EndsAt = parseSpreadTime(endsAt)
	o.CreatedAt = parseSpreadTime(created)
	o.UpdatedAt = parseSpreadTime(updated)
	if o.ReasonSet && strings.TrimSpace(reason) != "" {
		_ = json.Unmarshal([]byte(reason), &o.Reason)
		if o.Reason == nil {
			o.Reason = []string{}
		}
	}
	o.Holds = []domain.SpreadHold{}
	return &o, nil
}

func (s *SQLite) attachSpreadHolds(ctx context.Context, orders []*domain.SpreadOrder) error {
	if len(orders) == 0 {
		return nil
	}
	holders := make([]any, 0, len(orders))
	marks := make([]string, len(orders))
	byID := map[string]*domain.SpreadOrder{}
	for i, o := range orders {
		if o == nil {
			continue
		}
		marks[i] = "?"
		holders = append(holders, o.ID)
		byID[o.ID] = o
		o.Holds = []domain.SpreadHold{}
	}
	if len(holders) == 0 {
		return nil
	}
	q := `SELECT order_id, exchange, symbol, quantity FROM spread_holds WHERE order_id IN (` + strings.Join(marks[:len(holders)], ",") + `)`
	rows, err := s.db.QueryContext(ctx, q, holders...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var h domain.SpreadHold
		var id, ex string
		if err := rows.Scan(&id, &ex, &h.Symbol, &h.Quantity); err != nil {
			return err
		}
		h.Exchange = domain.Exchange(ex)
		if o := byID[id]; o != nil {
			o.Holds = append(o.Holds, h)
		}
	}
	return rows.Err()
}

func marshalSpreadReason(o domain.SpreadOrder) (string, error) {
	if !o.ReasonSet {
		return "", nil
	}
	raw := o.Reason
	if raw == nil {
		raw = []string{}
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func formatSpreadTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseSpreadTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t
	}
	t, _ := time.Parse(time.RFC3339, raw)
	return t
}
