package portfoliostore

import (
	"context"
	"database/sql"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/sqliteutil"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *SQLite) migrateRebalanceAttempts() error {
	for _, q := range []string{
		`ALTER TABLE allocation_rebalances ADD COLUMN updated_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE allocation_rebalances ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE allocation_rebalance_legs ADD COLUMN attempt INTEGER NOT NULL DEFAULT 1`,
		`CREATE TABLE IF NOT EXISTS allocation_rebalance_attempts (
	rebalance_id    TEXT NOT NULL,
	seq             INTEGER NOT NULL,
	status          TEXT NOT NULL,
	planned_legs    INTEGER NOT NULL,
	filled_legs     INTEGER NOT NULL,
	failed_legs     INTEGER NOT NULL,
	before_equity   REAL NOT NULL,
	after_equity    REAL NOT NULL,
	before_cash     REAL NOT NULL,
	after_cash      REAL NOT NULL,
	total_fee       REAL NOT NULL DEFAULT 0,
	total_slippage  REAL NOT NULL DEFAULT 0,
	summary         TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL,
	PRIMARY KEY (rebalance_id, seq)
)`,
	} {
		if err := sqliteutil.ExecAllowExists(s.db, q); err != nil {
			return err
		}
	}
	hasAttempt, err := tableHasColumn(s.db, "allocation_rebalance_lines", "attempt")
	if err != nil {
		return err
	}
	if !hasAttempt {
		if err := rebuildRebalanceLines(s.db); err != nil {
			return err
		}
	}
	return nil
}

func tableHasColumn(db *sql.DB, table, col string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == col {
			return true, nil
		}
	}
	return false, rows.Err()
}

func rebuildRebalanceLines(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS allocation_rebalance_lines_v2 (
	rebalance_id  TEXT NOT NULL,
	attempt       INTEGER NOT NULL DEFAULT 1,
	asset         TEXT NOT NULL,
	exchange      TEXT NOT NULL DEFAULT '',
	symbol        TEXT NOT NULL DEFAULT '',
	is_cash       INTEGER NOT NULL DEFAULT 0,
	target_pct    REAL NOT NULL,
	before_pct    REAL NOT NULL,
	after_pct     REAL NOT NULL,
	before_value  REAL NOT NULL,
	after_value   REAL NOT NULL,
	target_value  REAL NOT NULL,
	after_delta   REAL NOT NULL,
	PRIMARY KEY (rebalance_id, attempt, asset)
)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
INSERT OR IGNORE INTO allocation_rebalance_lines_v2
	(rebalance_id, attempt, asset, exchange, symbol, is_cash, target_pct, before_pct, after_pct, before_value, after_value, target_value, after_delta)
SELECT rebalance_id, 1, asset, exchange, symbol, is_cash, target_pct, before_pct, after_pct, before_value, after_value, target_value, after_delta
FROM allocation_rebalance_lines
`); err != nil {
		return err
	}
	if _, err := db.Exec(`DROP TABLE allocation_rebalance_lines`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE allocation_rebalance_lines_v2 RENAME TO allocation_rebalance_lines`); err != nil {
		return err
	}
	return nil
}

// StartRebalanceResult inserts a running job so a crash still leaves a row to continue.
func (s *SQLite) StartRebalanceResult(ctx context.Context, r domain.RebalanceResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = r.CreatedAt
	}
	if r.Status == "" {
		r.Status = domain.RebalanceResultRunning
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO allocation_rebalances (`+rebalanceCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.ClientID, r.BasketID, r.BasketName, r.Currency, r.Status,
		r.PlannedLegs, r.FilledLegs, r.FailedLegs,
		r.BeforeEquity, r.AfterEquity, r.BeforeCash, r.AfterCash,
		r.RemainingAbsDriftPct, r.MaxSleeveDriftPct, r.RemainingNotional,
		r.TotalFee, r.TotalSlippage, r.Summary, r.Note, r.CreatedAt.UTC().Format(time.RFC3339Nano),
		r.UpdatedAt.UTC().Format(time.RFC3339Nano), maxInt(r.AttemptCount, 1))
	return err
}

// AppendRebalanceLeg stores one planned/filled/failed leg immediately.
func (s *SQLite) AppendRebalanceLeg(ctx context.Context, rebalanceID string, attempt, seq int, leg domain.RebalanceLegOutcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if attempt <= 0 {
		attempt = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO allocation_rebalance_legs (
			rebalance_id, seq, attempt, side, asset, exchange, symbol, reason,
			planned_qty, planned_price, planned_notional, status,
			trade_id, filled_qty, fill_price, fill_notional, last_price, fee, slippage, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rebalanceID, seq, attempt, string(leg.Side), leg.Asset, string(leg.Exchange), leg.Symbol, leg.Reason,
		leg.PlannedQty, leg.PlannedPrice, leg.PlannedNotional, leg.Status,
		leg.TradeID, leg.FilledQty, leg.FillPrice, leg.FillNotional, leg.LastPrice, leg.Fee, leg.Slippage, leg.Error)
	return err
}

// FinishRebalanceResult writes the header, last attempt, and leftover mix.
func (s *SQLite) FinishRebalanceResult(ctx context.Context, r domain.RebalanceResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := updateRebalanceHeaderTx(ctx, tx, r); err != nil {
		return err
	}
	if len(r.Attempts) > 0 {
		att := r.Attempts[len(r.Attempts)-1]
		if err := upsertRebalanceAttemptTx(ctx, tx, r.ID, att); err != nil {
			return err
		}
		if err := replaceAttemptLinesTx(ctx, tx, r.ID, att.Seq, att.Lines); err != nil {
			return err
		}
	} else if err := replaceAttemptLinesTx(ctx, tx, r.ID, 1, r.Lines); err != nil {
		return err
	}
	return tx.Commit()
}

func updateRebalanceHeaderTx(ctx context.Context, tx *sql.Tx, r domain.RebalanceResult) error {
	updated := r.UpdatedAt
	if updated.IsZero() {
		updated = time.Now().UTC()
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE allocation_rebalances SET
			status = ?, planned_legs = ?, filled_legs = ?, failed_legs = ?,
			before_equity = ?, after_equity = ?, before_cash = ?, after_cash = ?,
			remaining_abs_drift_pct = ?, max_sleeve_drift_pct = ?, remaining_notional = ?,
			total_fee = ?, total_slippage = ?, summary = ?, note = ?,
			updated_at = ?, attempt_count = ?
		WHERE id = ? AND client_id = ?
	`, r.Status, r.PlannedLegs, r.FilledLegs, r.FailedLegs,
		r.BeforeEquity, r.AfterEquity, r.BeforeCash, r.AfterCash,
		r.RemainingAbsDriftPct, r.MaxSleeveDriftPct, r.RemainingNotional,
		r.TotalFee, r.TotalSlippage, r.Summary, r.Note,
		updated.UTC().Format(time.RFC3339Nano), maxInt(r.AttemptCount, 1),
		r.ID, r.ClientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func upsertRebalanceAttemptTx(ctx context.Context, tx *sql.Tx, rebalanceID string, att domain.RebalanceAttempt) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO allocation_rebalance_attempts (
			rebalance_id, seq, status, planned_legs, filled_legs, failed_legs,
			before_equity, after_equity, before_cash, after_cash,
			total_fee, total_slippage, summary, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(rebalance_id, seq) DO UPDATE SET
			status = excluded.status, planned_legs = excluded.planned_legs,
			filled_legs = excluded.filled_legs, failed_legs = excluded.failed_legs,
			after_equity = excluded.after_equity, after_cash = excluded.after_cash,
			total_fee = excluded.total_fee, total_slippage = excluded.total_slippage,
			summary = excluded.summary
	`, rebalanceID, att.Seq, att.Status, att.PlannedLegs, att.FilledLegs, att.FailedLegs,
		att.BeforeEquity, att.AfterEquity, att.BeforeCash, att.AfterCash,
		att.TotalFee, att.TotalSlippage, att.Summary, att.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func replaceAttemptLinesTx(ctx context.Context, tx *sql.Tx, rebalanceID string, attempt int, lines []domain.RebalanceLineSnapshot) error {
	if attempt <= 0 {
		attempt = 1
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM allocation_rebalance_lines WHERE rebalance_id = ? AND attempt = ?`, rebalanceID, attempt); err != nil {
		return err
	}
	for _, ln := range lines {
		cash := 0
		if ln.IsCash {
			cash = 1
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO allocation_rebalance_lines (
				rebalance_id, attempt, asset, exchange, symbol, is_cash, target_pct,
				before_pct, after_pct, before_value, after_value, target_value, after_delta
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, rebalanceID, attempt, ln.Asset, string(ln.Exchange), ln.Symbol, cash, ln.TargetPct,
			ln.BeforePct, ln.AfterPct, ln.BeforeValue, ln.AfterValue, ln.TargetValue, ln.AfterDelta); err != nil {
			return err
		}
	}
	return nil
}

func maxInt(v, floor int) int {
	if v < floor {
		return floor
	}
	return v
}

func attemptOf(v int) int {
	if v <= 0 {
		return 1
	}
	return v
}

func textOrNow(t, fallback time.Time) string {
	if t.IsZero() {
		t = fallback
	}
	if t.IsZero() {
		t = time.Now().UTC()
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func (s *SQLite) loadRebalanceAttempts(ctx context.Context, r *domain.RebalanceResult) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT seq, status, planned_legs, filled_legs, failed_legs,
			before_equity, after_equity, before_cash, after_cash,
			total_fee, total_slippage, summary, created_at
		FROM allocation_rebalance_attempts WHERE rebalance_id = ? ORDER BY seq ASC
	`, r.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var att domain.RebalanceAttempt
		var created string
		if err := rows.Scan(&att.Seq, &att.Status, &att.PlannedLegs, &att.FilledLegs, &att.FailedLegs,
			&att.BeforeEquity, &att.AfterEquity, &att.BeforeCash, &att.AfterCash,
			&att.TotalFee, &att.TotalSlippage, &att.Summary, &created); err != nil {
			return err
		}
		att.TotalCost = att.TotalFee + att.TotalSlippage
		att.CreatedAt = parseTime(created)
		r.Attempts = append(r.Attempts, att)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	byAttemptLegs := map[int][]domain.RebalanceLegOutcome{}
	for _, leg := range r.Legs {
		a := leg.Attempt
		if a <= 0 {
			a = 1
		}
		byAttemptLegs[a] = append(byAttemptLegs[a], leg)
	}
	byAttemptLines := map[int][]domain.RebalanceLineSnapshot{}
	for _, ln := range r.Lines {
		a := ln.Attempt
		if a <= 0 {
			a = 1
		}
		byAttemptLines[a] = append(byAttemptLines[a], ln)
	}
	if len(r.Attempts) == 0 && (len(r.Legs) > 0 || len(r.Lines) > 0) {
		r.Attempts = []domain.RebalanceAttempt{{Seq: 1, Legs: r.Legs, Lines: r.Lines}}
		return nil
	}
	for i := range r.Attempts {
		r.Attempts[i].Legs = byAttemptLegs[r.Attempts[i].Seq]
		r.Attempts[i].Lines = byAttemptLines[r.Attempts[i].Seq]
	}
	if len(r.Attempts) > 0 {
		r.Lines = r.Attempts[len(r.Attempts)-1].Lines
	}
	r.CanContinue = domain.RebalanceCanContinue(*r)
	return nil
}
