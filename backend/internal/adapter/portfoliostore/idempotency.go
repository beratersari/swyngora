package portfoliostore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func (s *SQLite) migrateIdempotency() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS idempotency_keys (
	client_id     TEXT NOT NULL,
	key           TEXT NOT NULL,
	request_hash  TEXT NOT NULL,
	kind          TEXT NOT NULL,
	result_json   TEXT NOT NULL DEFAULT '',
	created_at    TEXT NOT NULL,
	expires_at    TEXT NOT NULL,
	PRIMARY KEY (client_id, key)
);
CREATE INDEX IF NOT EXISTS idx_idempotency_exp ON idempotency_keys(expires_at);
`)
	return err
}

func (s *SQLite) GetIdempotency(ctx context.Context, clientID, key string) (*domain.IdempotencyRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT client_id, key, request_hash, kind, result_json, created_at, expires_at
		FROM idempotency_keys WHERE client_id = ? AND key = ?
	`, clientID, key)
	rec, err := scanIdempotency(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !rec.ExpiresAt.IsZero() && time.Now().UTC().After(rec.ExpiresAt) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE client_id = ? AND key = ?`, clientID, key)
		return nil, domain.ErrNotFound
	}
	return rec, nil
}

func scanIdempotency(row scannable) (*domain.IdempotencyRecord, error) {
	var rec domain.IdempotencyRecord
	var cAt, eAt string
	if err := row.Scan(&rec.ClientID, &rec.Key, &rec.RequestHash, &rec.Kind, &rec.ResultJSON, &cAt, &eAt); err != nil {
		return nil, err
	}
	rec.CreatedAt = parseTime(cAt)
	rec.ExpiresAt = parseTime(eAt)
	return &rec, nil
}

func (s *SQLite) txInsertIdempotency(ctx context.Context, tx *sql.Tx) error {
	rec := domain.IdempotencyFromContext(ctx)
	if rec == nil || rec.Key == "" {
		return nil
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	if rec.ExpiresAt.IsZero() {
		rec.ExpiresAt = rec.CreatedAt.Add(domain.DefaultIdempotencyTTL)
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO idempotency_keys (client_id, key, request_hash, kind, result_json, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, rec.ClientID, rec.Key, rec.RequestHash, rec.Kind, rec.ResultJSON,
		fmtTime(rec.CreatedAt), fmtTime(rec.ExpiresAt))
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w", domain.ErrIdempotencyHit)
		}
		return err
	}
	return nil
}

func (s *SQLite) GetTrade(ctx context.Context, clientID, id string) (*domain.Trade, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl, pending_order_id,
		       COALESCE(lot_method, ''), COALESCE(fee, 0), COALESCE(last_price, 0), created_at
		FROM trades WHERE client_id = ? AND id = ?
	`, clientID, id)
	var t domain.Trade
	var ex, side, cAt, lotMethod string
	if err := row.Scan(&t.ID, &t.ClientID, &ex, &t.Symbol, &side, &t.Quantity, &t.Price, &t.Notional, &t.RealizedPnL, &t.PendingOrderID, &lotMethod, &t.Fee, &t.LastPrice, &cAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	t.LotMethod, _ = domain.NormalizeLotMethod(lotMethod)
	t.Exchange = domain.Exchange(ex)
	t.Side = domain.TradeSide(side)
	t.CreatedAt = parseTime(cAt)
	return &t, nil
}
