package portfoliostore

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const insertTradeSQL = `
INSERT INTO trades (
	id, client_id, exchange, symbol, side, quantity, price, notional, realized_pnl,
	pending_order_id, lot_method, fee, last_price, situation_json, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

type sqlExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func execInsertTrade(ctx context.Context, ex sqlExec, t domain.Trade, at time.Time) error {
	if at.IsZero() {
		at = t.CreatedAt
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := ex.ExecContext(ctx, insertTradeSQL,
		t.ID, t.ClientID, string(t.Exchange), t.Symbol, string(t.Side), t.Quantity, t.Price, t.Notional, t.RealizedPnL,
		t.PendingOrderID, string(t.LotMethod), t.Fee, t.LastPrice, marshalSituation(t.Situation),
		at.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func marshalSituation(s *domain.CoinSituation) string {
	if s == nil {
		return ""
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ListSituationCloses returns every sold slice in the book with the buy's stored read.
// A lot whose source trade is missing has a nil situation.
func (s *SQLite) ListSituationCloses(ctx context.Context, clientID string) ([]domain.SituationClose, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.exchange, f.realized_pnl, COALESCE(t.situation_json, '')
		FROM tax_lot_fills f
		JOIN tax_lots l ON l.id = f.lot_id
		LEFT JOIN trades t ON t.client_id = l.client_id AND t.id = l.source_trade_id
		WHERE l.client_id = ?
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.SituationClose, 0)
	for rows.Next() {
		var ex, situation string
		var pnl float64
		if err := rows.Scan(&ex, &pnl, &situation); err != nil {
			return nil, err
		}
		out = append(out, domain.SituationClose{
			Exchange:    domain.Exchange(ex),
			RealizedPnL: pnl,
			Situation:   unmarshalSituation(situation),
		})
	}
	return out, rows.Err()
}

func unmarshalSituation(raw string) *domain.CoinSituation {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var s domain.CoinSituation
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil
	}
	return &s
}

func scanTrade(row scannable) (domain.Trade, error) {
	var t domain.Trade
	var ex, side, cAt, lotMethod, situation string
	if err := row.Scan(
		&t.ID, &t.ClientID, &ex, &t.Symbol, &side, &t.Quantity, &t.Price, &t.Notional, &t.RealizedPnL, &t.PendingOrderID,
		&lotMethod, &t.Fee, &t.LastPrice, &situation, &cAt,
	); err != nil {
		return domain.Trade{}, err
	}
	t.LotMethod, _ = domain.NormalizeLotMethod(lotMethod)
	t.Exchange = domain.Exchange(ex)
	t.Side = domain.TradeSide(side)
	t.Situation = unmarshalSituation(situation)
	t.CreatedAt = parseTime(cAt)
	return t, nil
}
