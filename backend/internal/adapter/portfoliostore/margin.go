package portfoliostore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const marginPosCols = `id, client_id, exchange, symbol, side, COALESCE(mode, 'isolated'), quantity, entry_price, leverage, margin,
	liquidation_price, stop_loss, take_profit, status, realized_pnl, close_reason, opened_at, updated_at, closed_at`

const marginPosInsertCols = `id, client_id, exchange, symbol, side, mode, quantity, entry_price, leverage, margin,
	liquidation_price, stop_loss, take_profit, status, realized_pnl, close_reason, opened_at, updated_at, closed_at`

func scanMarginPos(row scannable) (*domain.MarginPosition, error) {
	var p domain.MarginPosition
	var ex, side, mode, st, opened, updated string
	var sl, tp sql.NullFloat64
	var closed sql.NullString
	if err := row.Scan(
		&p.ID, &p.ClientID, &ex, &p.Symbol, &side, &mode, &p.Quantity, &p.EntryPrice, &p.Leverage, &p.Margin,
		&p.LiquidationPrice, &sl, &tp, &st, &p.RealizedPnL, &p.CloseReason, &opened, &updated, &closed,
	); err != nil {
		return nil, err
	}
	p.Exchange = domain.Exchange(ex)
	p.Side = domain.MarginSide(side)
	p.Mode = domain.MarginMode(mode)
	if p.Mode == "" {
		p.Mode = domain.MarginModeIsolated
	}
	p.Status = domain.MarginPositionStatus(st)
	p.OpenedAt = parseTime(opened)
	p.UpdatedAt = parseTime(updated)
	if sl.Valid {
		v := sl.Float64
		p.StopLoss = &v
	}
	if tp.Valid {
		v := tp.Float64
		p.TakeProfit = &v
	}
	if closed.Valid && closed.String != "" {
		t := parseTime(closed.String)
		p.ClosedAt = &t
	}
	return &p, nil
}

func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// CreateMarginPosition inserts a position row.
func (s *SQLite) CreateMarginPosition(ctx context.Context, pos domain.MarginPosition) (*domain.MarginPosition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pos.Mode == "" {
		pos.Mode = domain.MarginModeIsolated
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO margin_positions (`+marginPosInsertCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, pos.ID, pos.ClientID, string(pos.Exchange), pos.Symbol, string(pos.Side), string(pos.Mode), pos.Quantity, pos.EntryPrice,
		pos.Leverage, pos.Margin, pos.LiquidationPrice, nullFloat(pos.StopLoss), nullFloat(pos.TakeProfit),
		string(pos.Status), pos.RealizedPnL, pos.CloseReason,
		pos.OpenedAt.UTC().Format(time.RFC3339Nano), pos.UpdatedAt.UTC().Format(time.RFC3339Nano), nullTime(pos.ClosedAt))
	if err != nil {
		return nil, err
	}
	return s.getMarginPosByID(ctx, pos.ID)
}

func (s *SQLite) getMarginPosByID(ctx context.Context, id string) (*domain.MarginPosition, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+marginPosCols+` FROM margin_positions WHERE id = ?`, id)
	p, err := scanMarginPos(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// GetMarginPosition returns one position for the client.
func (s *SQLite) GetMarginPosition(ctx context.Context, clientID, id string) (*domain.MarginPosition, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+marginPosCols+` FROM margin_positions WHERE id = ? AND client_id = ?
	`, id, clientID)
	p, err := scanMarginPos(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// ListOpenMarginPositions lists open positions for a client.
func (s *SQLite) ListOpenMarginPositions(ctx context.Context, clientID string) ([]domain.MarginPosition, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+marginPosCols+` FROM margin_positions
		WHERE client_id = ? AND status = 'open' ORDER BY opened_at DESC
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMarginPositions(rows)
}

// ListAllOpenMarginPositions lists every open margin position (worker).
func (s *SQLite) ListAllOpenMarginPositions(ctx context.Context) ([]domain.MarginPosition, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+marginPosCols+` FROM margin_positions WHERE status = 'open' ORDER BY opened_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMarginPositions(rows)
}

// CountOpenMarginPositions counts open positions.
func (s *SQLite) CountOpenMarginPositions(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM margin_positions WHERE client_id = ? AND status = 'open'
	`, clientID).Scan(&n)
	return n, err
}

// UpdateMarginPosition updates an open position's mutable fields.
func (s *SQLite) UpdateMarginPosition(ctx context.Context, pos domain.MarginPosition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE margin_positions SET quantity = ?, margin = ?, liquidation_price = ?,
			stop_loss = ?, take_profit = ?, realized_pnl = ?, updated_at = ?
		WHERE id = ? AND status = 'open'
	`, pos.Quantity, pos.Margin, pos.LiquidationPrice, nullFloat(pos.StopLoss), nullFloat(pos.TakeProfit),
		pos.RealizedPnL, pos.UpdatedAt.UTC().Format(time.RFC3339Nano), pos.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CloseMarginPosition marks a position closed.
func (s *SQLite) CloseMarginPosition(ctx context.Context, pos domain.MarginPosition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE margin_positions SET quantity = ?, margin = ?, realized_pnl = ?, status = 'closed',
			close_reason = ?, updated_at = ?, closed_at = ?, stop_loss = NULL, take_profit = NULL
		WHERE id = ? AND status = 'open'
	`, pos.Quantity, pos.Margin, pos.RealizedPnL, pos.CloseReason,
		pos.UpdatedAt.UTC().Format(time.RFC3339Nano), nullTime(pos.ClosedAt), pos.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanMarginPositions(rows *sql.Rows) ([]domain.MarginPosition, error) {
	var out []domain.MarginPosition
	for rows.Next() {
		p, err := scanMarginPos(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

const marginOrderCols = `id, client_id, exchange, symbol, side, order_type, quantity, leverage, limit_price,
	reserved_margin, stop_loss, take_profit, status, position_id, reject_reason, cancel_reason,
	created_at, updated_at, filled_at, canceled_at`

func scanMarginOrder(row scannable) (*domain.MarginOrder, error) {
	var o domain.MarginOrder
	var ex, side, typ, st, cAt, uAt string
	var sl, tp sql.NullFloat64
	var filled, canceled sql.NullString
	if err := row.Scan(
		&o.ID, &o.ClientID, &ex, &o.Symbol, &side, &typ, &o.Quantity, &o.Leverage, &o.LimitPrice,
		&o.ReservedMargin, &sl, &tp, &st, &o.PositionID, &o.RejectReason, &o.CancelReason,
		&cAt, &uAt, &filled, &canceled,
	); err != nil {
		return nil, err
	}
	o.Exchange = domain.Exchange(ex)
	o.Side = domain.MarginSide(side)
	o.Type = domain.MarginOrderType(typ)
	o.Status = domain.MarginOrderStatus(st)
	o.CreatedAt = parseTime(cAt)
	o.UpdatedAt = parseTime(uAt)
	if sl.Valid {
		v := sl.Float64
		o.StopLoss = &v
	}
	if tp.Valid {
		v := tp.Float64
		o.TakeProfit = &v
	}
	if filled.Valid && filled.String != "" {
		t := parseTime(filled.String)
		o.FilledAt = &t
	}
	if canceled.Valid && canceled.String != "" {
		t := parseTime(canceled.String)
		o.CanceledAt = &t
	}
	return &o, nil
}

// CreateMarginOrder inserts a margin order.
func (s *SQLite) CreateMarginOrder(ctx context.Context, o domain.MarginOrder) (*domain.MarginOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO margin_orders (`+marginOrderCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, o.ID, o.ClientID, string(o.Exchange), o.Symbol, string(o.Side), string(o.Type), o.Quantity, o.Leverage,
		o.LimitPrice, o.ReservedMargin, nullFloat(o.StopLoss), nullFloat(o.TakeProfit), string(o.Status),
		o.PositionID, o.RejectReason, o.CancelReason,
		o.CreatedAt.UTC().Format(time.RFC3339Nano), o.UpdatedAt.UTC().Format(time.RFC3339Nano),
		nullTime(o.FilledAt), nullTime(o.CanceledAt))
	if err != nil {
		return nil, err
	}
	return s.getMarginOrderByID(ctx, o.ID)
}

func (s *SQLite) getMarginOrderByID(ctx context.Context, id string) (*domain.MarginOrder, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+marginOrderCols+` FROM margin_orders WHERE id = ?`, id)
	o, err := scanMarginOrder(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return o, err
}

// GetMarginOrder returns one order for the client.
func (s *SQLite) GetMarginOrder(ctx context.Context, clientID, id string) (*domain.MarginOrder, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+marginOrderCols+` FROM margin_orders WHERE id = ? AND client_id = ?
	`, id, clientID)
	o, err := scanMarginOrder(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return o, err
}

// ListMarginOrders lists margin orders.
func (s *SQLite) ListMarginOrders(ctx context.Context, clientID string, status domain.MarginOrderStatus, limit, offset int) ([]domain.MarginOrder, error) {
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
			SELECT `+marginOrderCols+` FROM margin_orders WHERE client_id = ?
			ORDER BY created_at DESC LIMIT ? OFFSET ?
		`, clientID, limit, offset)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT `+marginOrderCols+` FROM margin_orders WHERE client_id = ? AND status = ?
			ORDER BY created_at DESC LIMIT ? OFFSET ?
		`, clientID, string(status), limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMarginOrders(rows)
}

// ListAllOpenMarginOrders returns all open margin orders.
func (s *SQLite) ListAllOpenMarginOrders(ctx context.Context) ([]domain.MarginOrder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+marginOrderCols+` FROM margin_orders WHERE status = 'open' ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMarginOrders(rows)
}

// CountOpenMarginOrders counts open orders.
func (s *SQLite) CountOpenMarginOrders(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM margin_orders WHERE client_id = ? AND status = 'open'
	`, clientID).Scan(&n)
	return n, err
}

// SumReservedMargin sums reserved margin for open limit orders.
func (s *SQLite) SumReservedMargin(ctx context.Context, clientID string) (float64, error) {
	var n sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(reserved_margin), 0) FROM margin_orders
		WHERE client_id = ? AND status = 'open'
	`, clientID).Scan(&n)
	if err != nil {
		return 0, err
	}
	if !n.Valid {
		return 0, nil
	}
	return n.Float64, nil
}

// CancelMarginOrder cancels an open order and clears reservation.
func (s *SQLite) CancelMarginOrder(ctx context.Context, clientID, id string, at time.Time, reason string) (*domain.MarginOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE margin_orders SET status = 'canceled', cancel_reason = ?, reserved_margin = 0,
			updated_at = ?, canceled_at = ?
		WHERE id = ? AND client_id = ? AND status = 'open'
	`, reason, at.UTC().Format(time.RFC3339Nano), at.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getMarginOrderByID(ctx, id)
}

// FillMarginOrder marks order filled.
func (s *SQLite) FillMarginOrder(ctx context.Context, id, positionID string, at time.Time) (*domain.MarginOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE margin_orders SET status = 'filled', position_id = ?, reserved_margin = 0,
			updated_at = ?, filled_at = ?
		WHERE id = ? AND status = 'open'
	`, positionID, at.UTC().Format(time.RFC3339Nano), at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getMarginOrderByID(ctx, id)
}

// RejectMarginOrder marks rejected and releases reservation.
func (s *SQLite) RejectMarginOrder(ctx context.Context, id, reason string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE margin_orders SET status = 'rejected', reject_reason = ?, reserved_margin = 0,
			updated_at = ?, canceled_at = ?
		WHERE id = ? AND status = 'open'
	`, reason, at.UTC().Format(time.RFC3339Nano), at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanMarginOrders(rows *sql.Rows) ([]domain.MarginOrder, error) {
	var out []domain.MarginOrder
	for rows.Next() {
		o, err := scanMarginOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

// InsertMarginTrade inserts a margin trade row.
func (s *SQLite) InsertMarginTrade(ctx context.Context, t domain.MarginTrade) (*domain.MarginTrade, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO margin_trades (
			id, client_id, position_id, exchange, symbol, side, action, quantity, price,
			notional, realized_pnl, margin_delta, leverage, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, t.PositionID, string(t.Exchange), t.Symbol, string(t.Side), t.Action,
		t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.MarginDelta, t.Leverage,
		t.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	cp := t
	return &cp, nil
}

// ListMarginTrades lists margin trades.
func (s *SQLite) ListMarginTrades(ctx context.Context, clientID string, limit, offset int) ([]domain.MarginTrade, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, position_id, exchange, symbol, side, action, quantity, price,
			notional, realized_pnl, margin_delta, leverage, created_at
		FROM margin_trades WHERE client_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MarginTrade
	for rows.Next() {
		var t domain.MarginTrade
		var ex, side, cAt string
		if err := rows.Scan(
			&t.ID, &t.ClientID, &t.PositionID, &ex, &t.Symbol, &side, &t.Action, &t.Quantity, &t.Price,
			&t.Notional, &t.RealizedPnL, &t.MarginDelta, &t.Leverage, &cAt,
		); err != nil {
			return nil, err
		}
		t.Exchange = domain.Exchange(ex)
		t.Side = domain.MarginSide(side)
		t.CreatedAt = parseTime(cAt)
		out = append(out, t)
	}
	return out, rows.Err()
}

// ApplyMarginOpen debits cash, inserts position and trade.
func (s *SQLite) ApplyMarginOpen(ctx context.Context, p *domain.Portfolio, pos domain.MarginPosition, t domain.MarginTrade) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.txUpdatePortfolioCash(ctx, tx, p); err != nil {
		return err
	}
	if err := s.txInsertMarginPos(ctx, tx, pos); err != nil {
		return err
	}
	if err := s.txInsertMarginTrade(ctx, tx, t); err != nil {
		return err
	}
	return tx.Commit()
}

// ApplyMarginClose credits cash, updates or closes position, inserts trade.
func (s *SQLite) ApplyMarginClose(ctx context.Context, p *domain.Portfolio, pos domain.MarginPosition, t domain.MarginTrade, fullClose bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.txUpdatePortfolioCash(ctx, tx, p); err != nil {
		return err
	}
	if fullClose {
		res, err := tx.ExecContext(ctx, `
			UPDATE margin_positions SET quantity = 0, margin = 0, realized_pnl = ?, status = 'closed',
				close_reason = ?, updated_at = ?, closed_at = ?, stop_loss = NULL, take_profit = NULL
			WHERE id = ? AND status = 'open'
		`, pos.RealizedPnL, pos.CloseReason, pos.UpdatedAt.UTC().Format(time.RFC3339Nano),
			nullTime(pos.ClosedAt), pos.ID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return domain.ErrNotFound
		}
	} else {
		res, err := tx.ExecContext(ctx, `
			UPDATE margin_positions SET quantity = ?, margin = ?, liquidation_price = ?,
				realized_pnl = ?, updated_at = ?
			WHERE id = ? AND status = 'open'
		`, pos.Quantity, pos.Margin, pos.LiquidationPrice, pos.RealizedPnL,
			pos.UpdatedAt.UTC().Format(time.RFC3339Nano), pos.ID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return domain.ErrNotFound
		}
	}
	if err := s.txInsertMarginTrade(ctx, tx, t); err != nil {
		return err
	}
	return tx.Commit()
}

// ApplyMarginOpenFromOrder fills limit order + opens position + debits cash in one tx.
func (s *SQLite) ApplyMarginOpenFromOrder(ctx context.Context, p *domain.Portfolio, orderID string, pos domain.MarginPosition, t domain.MarginTrade, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
		UPDATE margin_orders SET status = 'filled', position_id = ?, reserved_margin = 0,
			updated_at = ?, filled_at = ?
		WHERE id = ? AND status = 'open'
	`, pos.ID, at.UTC().Format(time.RFC3339Nano), at.UTC().Format(time.RFC3339Nano), orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	if err := s.txUpdatePortfolioCash(ctx, tx, p); err != nil {
		return err
	}
	if err := s.txInsertMarginPos(ctx, tx, pos); err != nil {
		return err
	}
	if err := s.txInsertMarginTrade(ctx, tx, t); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLite) txUpdatePortfolioCash(ctx context.Context, tx *sql.Tx, p *domain.Portfolio) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE portfolios SET cash_balance = ?, realized_pnl_total = ?, updated_at = ?
		WHERE client_id = ?
	`, p.CashBalance, p.RealizedPnLTotal, p.UpdatedAt.UTC().Format(time.RFC3339Nano), p.ClientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("portfolio update: %w", domain.ErrNotFound)
	}
	return nil
}

func (s *SQLite) txInsertMarginPos(ctx context.Context, tx *sql.Tx, pos domain.MarginPosition) error {
	if pos.Mode == "" {
		pos.Mode = domain.MarginModeIsolated
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO margin_positions (`+marginPosInsertCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, pos.ID, pos.ClientID, string(pos.Exchange), pos.Symbol, string(pos.Side), string(pos.Mode), pos.Quantity, pos.EntryPrice,
		pos.Leverage, pos.Margin, pos.LiquidationPrice, nullFloat(pos.StopLoss), nullFloat(pos.TakeProfit),
		string(pos.Status), pos.RealizedPnL, pos.CloseReason,
		pos.OpenedAt.UTC().Format(time.RFC3339Nano), pos.UpdatedAt.UTC().Format(time.RFC3339Nano), nullTime(pos.ClosedAt))
	return err
}

// ApplyMarginAdjust updates portfolio cash and position margin/liq atomically.
func (s *SQLite) ApplyMarginAdjust(ctx context.Context, p *domain.Portfolio, pos domain.MarginPosition, t domain.MarginTrade) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.txUpdatePortfolioCash(ctx, tx, p); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE margin_positions SET margin = ?, liquidation_price = ?, updated_at = ?
		WHERE id = ? AND status = 'open'
	`, pos.Margin, pos.LiquidationPrice, pos.UpdatedAt.UTC().Format(time.RFC3339Nano), pos.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	if err := s.txInsertMarginTrade(ctx, tx, t); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdatePortfolioMarginMode sets the account margin mode.
func (s *SQLite) UpdatePortfolioMarginMode(ctx context.Context, clientID string, mode domain.MarginMode, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE portfolios SET margin_mode = ?, updated_at = ? WHERE client_id = ?
	`, string(mode), at.UTC().Format(time.RFC3339Nano), clientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}


func (s *SQLite) txInsertMarginTrade(ctx context.Context, tx *sql.Tx, t domain.MarginTrade) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO margin_trades (
			id, client_id, position_id, exchange, symbol, side, action, quantity, price,
			notional, realized_pnl, margin_delta, leverage, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ClientID, t.PositionID, string(t.Exchange), t.Symbol, string(t.Side), t.Action,
		t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.MarginDelta, t.Leverage,
		t.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}
