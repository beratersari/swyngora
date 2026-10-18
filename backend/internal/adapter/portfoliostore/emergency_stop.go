package portfoliostore

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *SQLite) migrateEmergencyStops() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS emergency_stops (
	client_id  TEXT PRIMARY KEY NOT NULL,
	enabled    INTEGER NOT NULL DEFAULT 0,
	loss_pct   REAL NOT NULL DEFAULT 0,
	tripped    INTEGER NOT NULL DEFAULT 0,
	run_id     TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_emergency_stops_enabled ON emergency_stops(enabled);
CREATE TABLE IF NOT EXISTS emergency_stop_runs (
	id            TEXT PRIMARY KEY NOT NULL,
	client_id     TEXT NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL,
	daily_pnl_pct REAL NOT NULL DEFAULT 0,
	loss_pct      REAL NOT NULL DEFAULT 0,
	actions_json  TEXT NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_emergency_stop_runs_book ON emergency_stop_runs(client_id, created_at DESC);
`)
	return err
}

func (s *SQLite) GetEmergencyStop(ctx context.Context, bookID string) (*domain.EmergencyStop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row domain.EmergencyStop
	var enabled, tripped int
	err := s.db.QueryRowContext(ctx, `
SELECT client_id, enabled, loss_pct, tripped, run_id, updated_at
FROM emergency_stops WHERE client_id = ?`, bookID).Scan(
		&row.ClientID, &enabled, &row.LossPct, &tripped, &row.RunID, &row.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	row.Enabled = enabled != 0
	row.Tripped = tripped != 0
	return &row, nil
}

func (s *SQLite) UpsertEmergencyStop(ctx context.Context, row domain.EmergencyStop) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if row.UpdatedAt == "" {
		row.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	enabled, tripped := 0, 0
	if row.Enabled {
		enabled = 1
	}
	if row.Tripped {
		tripped = 1
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO emergency_stops (client_id, enabled, loss_pct, tripped, run_id, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(client_id) DO UPDATE SET
	enabled = excluded.enabled,
	loss_pct = excluded.loss_pct,
	tripped = excluded.tripped,
	run_id = excluded.run_id,
	updated_at = excluded.updated_at`,
		row.ClientID, enabled, row.LossPct, tripped, row.RunID, row.UpdatedAt)
	return err
}

func (s *SQLite) ListEnabledEmergencyStops(ctx context.Context) ([]domain.EmergencyStop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `
SELECT client_id, enabled, loss_pct, tripped, run_id, updated_at
FROM emergency_stops WHERE enabled <> 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EmergencyStop
	for rows.Next() {
		var row domain.EmergencyStop
		var enabled, tripped int
		if err := rows.Scan(&row.ClientID, &enabled, &row.LossPct, &tripped, &row.RunID, &row.UpdatedAt); err != nil {
			return nil, err
		}
		row.Enabled = enabled != 0
		row.Tripped = tripped != 0
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *SQLite) SaveEmergencyStopRun(ctx context.Context, run domain.EmergencyStopRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(run.Actions)
	if err != nil {
		return err
	}
	if run.Actions == nil {
		raw = []byte("[]")
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO emergency_stop_runs (id, client_id, created_at, updated_at, daily_pnl_pct, loss_pct, actions_json)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	updated_at = excluded.updated_at,
	daily_pnl_pct = excluded.daily_pnl_pct,
	loss_pct = excluded.loss_pct,
	actions_json = excluded.actions_json`,
		run.ID, run.ClientID, run.CreatedAt, run.UpdatedAt, run.DailyPnLPct, run.LossPct, string(raw))
	return err
}

func (s *SQLite) GetEmergencyStopRun(ctx context.Context, bookID, runID string) (*domain.EmergencyStopRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var run domain.EmergencyStopRun
	var raw string
	err := s.db.QueryRowContext(ctx, `
SELECT id, client_id, created_at, updated_at, daily_pnl_pct, loss_pct, actions_json
FROM emergency_stop_runs WHERE id = ? AND client_id = ?`, runID, bookID).Scan(
		&run.ID, &run.ClientID, &run.CreatedAt, &run.UpdatedAt, &run.DailyPnLPct, &run.LossPct, &raw)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &run.Actions); err != nil {
			return nil, err
		}
	}
	return &run, nil
}
