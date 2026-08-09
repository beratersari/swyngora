package portfoliostore

import (
	"fmt"
)

func (s *SQLite) migrateMultiBook() error {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('portfolios') WHERE name = 'id'`).Scan(&n)
	if err != nil {
		return fmt.Errorf("pragma portfolios.id: %w", err)
	}
	if n > 0 {
		return nil
	}
	// Legacy PK was client_id. Rebuild so each book has its own id (id = old client_id).
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
CREATE TABLE portfolios_v2 (
	id                TEXT PRIMARY KEY NOT NULL,
	client_id         TEXT NOT NULL,
	name              TEXT NOT NULL DEFAULT 'Main',
	currency          TEXT NOT NULL,
	starting_balance  REAL NOT NULL,
	cash_balance      REAL NOT NULL,
	realized_pnl_total REAL NOT NULL DEFAULT 0,
	net_deposits      REAL NOT NULL DEFAULT 0,
	margin_mode       TEXT NOT NULL DEFAULT 'isolated',
	created_at        TEXT NOT NULL,
	updated_at        TEXT NOT NULL
);`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO portfolios_v2 (id, client_id, name, currency, starting_balance, cash_balance, realized_pnl_total, net_deposits, margin_mode, created_at, updated_at)
SELECT client_id, client_id, 'Main', currency, starting_balance, cash_balance,
	COALESCE(realized_pnl_total, 0), COALESCE(net_deposits, 0), COALESCE(margin_mode, 'isolated'), created_at, updated_at
FROM portfolios;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE portfolios`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE portfolios_v2 RENAME TO portfolios`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_portfolios_client ON portfolios(client_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_portfolios_client_name ON portfolios(client_id, name)`); err != nil {
		return err
	}
	return tx.Commit()
}
