package portfoliostore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ExportOwnedPortfolios dumps every book the owner controls.
func (s *SQLite) ExportOwnedPortfolios(ctx context.Context, ownerClientID string) ([]domain.PortfolioSnapshot, error) {
	books, err := s.ListPortfolios(ctx, ownerClientID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PortfolioSnapshot, 0, len(books))
	for i := range books {
		snap, err := s.exportBook(ctx, books[i])
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, nil
}

func (s *SQLite) exportBook(ctx context.Context, book domain.Portfolio) (domain.PortfolioSnapshot, error) {
	id := book.BookID()
	snap := domain.PortfolioSnapshot{Book: book}
	var err error
	if snap.Positions, err = s.ListPositions(ctx, id); err != nil {
		return snap, err
	}
	if snap.Trades, err = s.listAllTrades(ctx, id); err != nil {
		return snap, err
	}
	open, err := s.ListPendingOrders(ctx, id, domain.PendingStatusOpen, 500, 0)
	if err != nil {
		return snap, err
	}
	pending, err := s.ListPendingOrders(ctx, id, domain.PendingStatusPending, 500, 0)
	if err != nil {
		return snap, err
	}
	snap.OpenOrders = append(open, pending...)
	if snap.Lots, err = s.ListTaxLots(ctx, id, "", "", false); err != nil {
		return snap, err
	}
	tradeIDs := make([]string, 0, len(snap.Trades))
	for _, t := range snap.Trades {
		tradeIDs = append(tradeIDs, t.ID)
	}
	if snap.LotFills, err = s.ListTaxLotFillsForTrades(ctx, tradeIDs); err != nil {
		return snap, err
	}
	if snap.RecurringPlans, err = s.ListRecurringBuyPlans(ctx, id); err != nil {
		return snap, err
	}
	for _, p := range snap.RecurringPlans {
		runs, rerr := s.ListRecurringBuyRuns(ctx, id, p.ID, 500, 0)
		if rerr != nil {
			return snap, rerr
		}
		snap.RecurringRuns = append(snap.RecurringRuns, runs...)
	}
	if snap.MarginPositions, err = s.listAllMarginPositions(ctx, id); err != nil {
		return snap, err
	}
	if snap.MarginOrders, err = s.ListMarginOrders(ctx, id, domain.MarginOrderOpen, 200, 0); err != nil {
		return snap, err
	}
	if snap.MarginTrades, err = s.listAllMarginTrades(ctx, id); err != nil {
		return snap, err
	}
	if snap.Shares, err = s.ListPortfolioSharesByBook(ctx, id); err != nil {
		return snap, err
	}
	return snap, nil
}

func (s *SQLite) listAllTrades(ctx context.Context, bookID string) ([]domain.Trade, error) {
	var all []domain.Trade
	off := 0
	for {
		page, err := s.ListTrades(ctx, bookID, 200, off)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < 200 {
			return all, nil
		}
		off += 200
	}
}

func (s *SQLite) listAllMarginTrades(ctx context.Context, bookID string) ([]domain.MarginTrade, error) {
	var all []domain.MarginTrade
	off := 0
	for {
		page, err := s.ListMarginTrades(ctx, bookID, 200, off)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < 200 {
			return all, nil
		}
		off += 200
	}
}

func (s *SQLite) listAllMarginPositions(ctx context.Context, bookID string) ([]domain.MarginPosition, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+marginPosCols+` FROM margin_positions WHERE client_id = ? ORDER BY opened_at ASC
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

// ImportOwnedPortfolios restores remapped snapshots. replace wipes the owner's books first.
func (s *SQLite) ImportOwnedPortfolios(ctx context.Context, ownerClientID string, snaps []domain.PortfolioSnapshot, replace bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if replace {
		ids, err := s.txListBookIDs(ctx, tx, ownerClientID)
		if err != nil {
			return 0, err
		}
		for _, id := range ids {
			if err := s.txDeleteBook(ctx, tx, id); err != nil {
				return 0, err
			}
		}
	}

	existingIDs := map[string]struct{}{}
	existingNames := map[string]struct{}{}
	if !replace {
		ids, err := s.txListBookIDs(ctx, tx, ownerClientID)
		if err != nil {
			return 0, err
		}
		for _, id := range ids {
			existingIDs[id] = struct{}{}
		}
		rows, err := tx.QueryContext(ctx, `SELECT lower(name) FROM portfolios WHERE client_id = ?`, ownerClientID)
		if err != nil {
			return 0, err
		}
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				rows.Close()
				return 0, err
			}
			existingNames[n] = struct{}{}
		}
		rows.Close()
	}

	var nBooks int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM portfolios WHERE client_id = ?`, ownerClientID).Scan(&nBooks); err != nil {
		return 0, err
	}

	added := 0
	for _, snap := range snaps {
		if err := domain.ValidatePortfolioSnapshot(snap); err != nil {
			continue
		}
		if snap.Book.ClientID != ownerClientID {
			snap.Book.ClientID = ownerClientID
		}
		if !replace {
			if _, ok := existingIDs[snap.Book.ID]; ok {
				continue
			}
			if _, ok := existingNames[strings.ToLower(snap.Book.Name)]; ok {
				continue
			}
		}
		taken, takenOwner, err := s.txBookOwner(ctx, tx, snap.Book.ID)
		if err != nil {
			return added, err
		}
		if domain.MustRemintImportedBookID(snap.Book.ID, ownerClientID) || (taken && takenOwner != ownerClientID) {
			snap = domain.RebindPortfolioSnapshotBookID(snap, uuid.NewString())
		}
		if nBooks >= domain.MaxPortfoliosPerClient {
			break
		}
		if err := s.txInsertSnapshot(ctx, tx, snap); err != nil {
			return added, err
		}
		existingIDs[snap.Book.ID] = struct{}{}
		existingNames[strings.ToLower(snap.Book.Name)] = struct{}{}
		nBooks++
		added++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return added, nil
}

func (s *SQLite) txBookOwner(ctx context.Context, tx *sql.Tx, id string) (bool, string, error) {
	var owner string
	err := tx.QueryRowContext(ctx, `SELECT client_id FROM portfolios WHERE id = ?`, id).Scan(&owner)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, owner, nil
}

func (s *SQLite) txListBookIDs(ctx context.Context, tx *sql.Tx, owner string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM portfolios WHERE client_id = ?`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *SQLite) txDeleteBook(ctx context.Context, tx *sql.Tx, id string) error {
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
	return nil
}

func (s *SQLite) txInsertSnapshot(ctx context.Context, tx *sql.Tx, snap domain.PortfolioSnapshot) error {
	p := snap.Book
	if p.Name == "" {
		p.Name = domain.DefaultPortfolioName
	}
	if p.MarginMode == "" {
		p.MarginMode = domain.MarginModeIsolated
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO portfolios (id, client_id, name, currency, starting_balance, cash_balance, realized_pnl_total, net_deposits, margin_mode, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.ClientID, p.Name, p.Currency, p.StartingBalance, p.CashBalance, p.RealizedPnLTotal, p.NetDeposits, string(p.MarginMode),
		fmtTime(p.CreatedAt), fmtTime(p.UpdatedAt)); err != nil {
		return fmt.Errorf("import portfolio: %w", err)
	}
	for _, pos := range snap.Positions {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO positions (client_id, exchange, symbol, quantity, avg_cost, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, pos.ClientID, string(pos.Exchange), pos.Symbol, pos.Quantity, pos.AvgCost, fmtTime(pos.UpdatedAt)); err != nil {
			return err
		}
	}
	for _, lot := range snap.Lots {
		if lot.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tax_lots (id, client_id, exchange, symbol, quantity, original_quantity, price, opened_at, source_trade_id, closed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, lot.ID, lot.ClientID, string(lot.Exchange), lot.Symbol, lot.Quantity, lot.OriginalQuantity, lot.Price,
			fmtTime(lot.OpenedAt), lot.SourceTradeID, nullTime(lot.ClosedAt)); err != nil {
			return err
		}
	}
	for _, t := range snap.Trades {
		if t.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO trades (id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id, lot_method, fee, last_price, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL,
			t.PendingOrderID, string(t.LotMethod), t.Fee, t.LastPrice, fmtTime(t.CreatedAt)); err != nil {
			return err
		}
	}
	for _, f := range snap.LotFills {
		if f.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tax_lot_fills (id, trade_id, lot_id, quantity, cost_price, sell_price, realized_pnl)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, f.ID, f.TradeID, f.LotID, f.Quantity, f.CostPrice, f.SellPrice, f.RealizedPnL); err != nil {
			return err
		}
	}
	for _, o := range snap.OpenOrders {
		if o.ID == "" {
			continue
		}
		if err := s.txInsertPendingOrder(ctx, tx, o); err != nil {
			return err
		}
	}
	for _, plan := range snap.RecurringPlans {
		if plan.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO recurring_buy_plans (`+recurringPlanCols+`)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, plan.ID, plan.ClientID, string(plan.Exchange), plan.Symbol, plan.Name, plan.Amount, string(plan.Frequency),
			plan.Weekday, plan.DayOfMonth, plan.IntervalHours,
			plan.TimeZone, boolToInt(plan.HasLocalTime), plan.Hour, plan.Minute, plan.MaxPrice, string(plan.Status),
			fmtTime(plan.NextRunAt), nullTime(plan.LastRunAt), plan.LastPeriodKey,
			fmtTime(plan.CreatedAt), fmtTime(plan.UpdatedAt)); err != nil {
			return err
		}
	}
	for _, run := range snap.RecurringRuns {
		if run.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO recurring_buy_runs (
				id, plan_id, client_id, period_key, status, amount, quantity, price,
				trade_id, fail_reason, scheduled_for, executed_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, run.ID, run.PlanID, run.ClientID, run.PeriodKey, string(run.Status),
			run.Amount, run.Quantity, run.Price, run.TradeID, run.FailReason,
			fmtTime(run.ScheduledFor), fmtTime(run.ExecutedAt)); err != nil {
			return err
		}
	}
	for _, pos := range snap.MarginPositions {
		if pos.ID == "" {
			continue
		}
		if pos.Mode == "" {
			pos.Mode = domain.MarginModeIsolated
		}
		if pos.DebtAsset == "" {
			pos.DebtAsset = domain.DebtAssetQuote
		}
		lastInt := nullTime(&pos.LastInterestAt)
		if pos.LastInterestAt.IsZero() {
			lastInt = nil
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO margin_positions (`+marginPosInsertCols+`)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, pos.ID, pos.ClientID, string(pos.Exchange), pos.Symbol, string(pos.Side), string(pos.Mode), pos.Quantity, pos.EntryPrice,
			pos.Leverage, pos.Margin, pos.DebtPrincipal, pos.DebtInterest, string(pos.DebtAsset), lastInt,
			pos.LiquidationPrice, nullFloat(pos.StopLoss), nullFloat(pos.TakeProfit),
			string(pos.Status), pos.RealizedPnL, pos.CloseReason,
			fmtTime(pos.OpenedAt), fmtTime(pos.UpdatedAt), nullTime(pos.ClosedAt)); err != nil {
			return err
		}
	}
	for _, o := range snap.MarginOrders {
		if o.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO margin_orders (`+marginOrderCols+`)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, o.ID, o.ClientID, string(o.Exchange), o.Symbol, string(o.Side), string(o.Type), o.Quantity, o.Leverage, o.LimitPrice,
			o.ReservedMargin, nullFloat(o.StopLoss), nullFloat(o.TakeProfit), string(o.Status), o.PositionID,
			o.RejectReason, o.CancelReason, fmtTime(o.CreatedAt), fmtTime(o.UpdatedAt), nullTime(o.FilledAt), nullTime(o.CanceledAt)); err != nil {
			return err
		}
	}
	for _, t := range snap.MarginTrades {
		if t.ID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO margin_trades (
				id, client_id, position_id, exchange, symbol, side, action, quantity, price,
				notional, realized_pnl, margin_delta, principal_paid, interest_paid, leverage, fee, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, t.ID, t.ClientID, t.PositionID, string(t.Exchange), t.Symbol, string(t.Side), t.Action,
			t.Quantity, t.Price, t.Notional, t.RealizedPnL, t.MarginDelta, t.PrincipalPaid, t.InterestPaid, t.Leverage, t.Fee,
			fmtTime(t.CreatedAt)); err != nil {
			return err
		}
	}
	for _, sh := range snap.Shares {
		if sh.GranteeClientID == "" || sh.GranteeClientID == p.ClientID {
			continue
		}
		if sh.Role == "" {
			sh.Role = domain.PortfolioRoleViewer
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO portfolio_shares (portfolio_id, owner_client_id, grantee_client_id, role, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, sh.PortfolioID, p.ClientID, sh.GranteeClientID, string(sh.Role), fmtTime(sh.CreatedAt), fmtTime(sh.UpdatedAt)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				continue
			}
			return err
		}
	}
	return nil
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.UTC().Format(time.RFC3339Nano)
}
