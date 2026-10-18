package portfoliostore

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *SQLite) migrateFillCosts() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS execution_quotes (
	id         TEXT PRIMARY KEY NOT NULL,
	client_id  TEXT NOT NULL,
	rows_json  TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_execution_quotes_client ON execution_quotes(client_id, created_at);
CREATE TABLE IF NOT EXISTS fill_cost_checks (
	id                 TEXT PRIMARY KEY NOT NULL,
	client_id          TEXT NOT NULL,
	style              TEXT NOT NULL,
	symbol             TEXT NOT NULL,
	side               TEXT NOT NULL,
	expected_fee       REAL NOT NULL,
	expected_slippage  REAL NOT NULL,
	touch_price        REAL NOT NULL,
	quoted_quantity    REAL NOT NULL,
	quoted_at          TEXT NOT NULL,
	actual_fee         REAL NOT NULL DEFAULT 0,
	actual_slippage    REAL NOT NULL DEFAULT 0,
	filled_quantity    REAL NOT NULL DEFAULT 0,
	status             TEXT NOT NULL,
	lead_trade_id      TEXT NOT NULL DEFAULT '',
	spread_order_id    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_fill_cost_lead ON fill_cost_checks(client_id, lead_trade_id);
CREATE INDEX IF NOT EXISTS idx_fill_cost_spread ON fill_cost_checks(client_id, spread_order_id);
`)
	return err
}

func (s *SQLite) SaveExecutionQuote(ctx context.Context, q domain.StoredExecutionQuote) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(q.Compare)
	if err != nil {
		return err
	}
	if q.CreatedAt.IsZero() {
		q.CreatedAt = time.Now().UTC()
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO execution_quotes (id, client_id, rows_json, created_at) VALUES (?, ?, ?, ?)`,
		q.ID, q.ClientID, string(raw), q.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *SQLite) GetExecutionQuote(ctx context.Context, clientID, id string) (*domain.StoredExecutionQuote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var raw, at string
	err := s.db.QueryRowContext(ctx, `
SELECT rows_json, created_at FROM execution_quotes WHERE id = ? AND client_id = ?`, id, clientID).Scan(&raw, &at)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var cmp domain.ExecutionCompare
	if err := json.Unmarshal([]byte(raw), &cmp); err != nil {
		return nil, err
	}
	return &domain.StoredExecutionQuote{
		ID: id, ClientID: clientID, Compare: cmp, CreatedAt: parseTime(at),
	}, nil
}

func (s *SQLite) DeleteExpiredExecutionQuotes(ctx context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM execution_quotes WHERE created_at < ?`, before.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *SQLite) SaveFillCostCheck(ctx context.Context, c domain.FillCostCheck) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.QuotedAt.IsZero() {
		c.QuotedAt = time.Now().UTC()
	}
	if c.Status == "" {
		c.Status = domain.FillCostOpen
	}
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO fill_cost_checks (
	id, client_id, style, symbol, side, expected_fee, expected_slippage, touch_price,
	quoted_quantity, quoted_at, actual_fee, actual_slippage, filled_quantity, status,
	lead_trade_id, spread_order_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.ClientID, string(c.Style), c.Symbol, string(c.Side), c.ExpectedFee, c.ExpectedSlippage, c.TouchPrice,
		c.QuotedQuantity, c.QuotedAt.UTC().Format(time.RFC3339Nano), c.ActualFee, c.ActualSlippage, c.FilledQuantity, c.Status,
		c.LeadTradeID, c.SpreadOrderID,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (s *SQLite) GetFillCostBySpread(ctx context.Context, clientID, spreadOrderID string) (*domain.FillCostCheck, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	spreadOrderID = strings.TrimSpace(spreadOrderID)
	if spreadOrderID == "" {
		return nil, domain.ErrNotFound
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, client_id, style, symbol, side, expected_fee, expected_slippage, touch_price,
       quoted_quantity, quoted_at, actual_fee, actual_slippage, filled_quantity, status, lead_trade_id, spread_order_id
FROM fill_cost_checks WHERE client_id = ? AND spread_order_id = ?`, clientID, spreadOrderID)
	c, err := scanFillCost(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *SQLite) AddFillCostActual(ctx context.Context, clientID, spreadOrderID, leadTradeID string, fee, slippage, qty float64, complete bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	done := 0
	if complete {
		done = 1
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE fill_cost_checks
SET actual_fee = actual_fee + ?,
    actual_slippage = actual_slippage + ?,
    filled_quantity = filled_quantity + ?,
    lead_trade_id = CASE WHEN ? <> '' THEN ? ELSE lead_trade_id END,
    status = CASE WHEN ? <> 0 THEN ? ELSE status END
WHERE client_id = ? AND spread_order_id = ? AND spread_order_id <> ''`,
		fee, slippage, qty, leadTradeID, leadTradeID, done, domain.FillCostComplete, clientID, spreadOrderID)
	return err
}

func (s *SQLite) ListCompleteFillCosts(ctx context.Context, clientID string, tradeIDs []string) (map[string]domain.FillCostCheck, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(tradeIDs))
	for _, id := range tradeIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	marks := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, clientID)
	for i, id := range ids {
		marks[i] = "?"
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, client_id, style, symbol, side, expected_fee, expected_slippage, touch_price,
       quoted_quantity, quoted_at, actual_fee, actual_slippage, filled_quantity, status, lead_trade_id, spread_order_id
FROM fill_cost_checks
WHERE client_id = ? AND status = ? AND lead_trade_id IN (`+strings.Join(marks, ",")+`)`,
		append([]any{args[0], domain.FillCostComplete}, args[1:]...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.FillCostCheck{}
	for rows.Next() {
		c, err := scanFillCost(rows)
		if err != nil {
			return nil, err
		}
		out[c.LeadTradeID] = c
	}
	return out, rows.Err()
}

func scanFillCost(row scannable) (domain.FillCostCheck, error) {
	var c domain.FillCostCheck
	var style, side, at string
	if err := row.Scan(
		&c.ID, &c.ClientID, &style, &c.Symbol, &side, &c.ExpectedFee, &c.ExpectedSlippage, &c.TouchPrice,
		&c.QuotedQuantity, &at, &c.ActualFee, &c.ActualSlippage, &c.FilledQuantity, &c.Status, &c.LeadTradeID, &c.SpreadOrderID,
	); err != nil {
		return domain.FillCostCheck{}, err
	}
	c.Style = domain.ExecutionStyle(style)
	c.Side = domain.TradeSide(side)
	c.QuotedAt = parseTime(at)
	return c, nil
}
