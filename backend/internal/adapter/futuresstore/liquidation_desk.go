package futuresstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SumLiquidationSymbols rolls stored prints in [from, to) by venue and coin.
func (s *SQLite) SumLiquidationSymbols(ctx context.Context, from, to time.Time, exchanges []domain.Exchange) (domain.LiquidationSymbolScan, error) {
	if s == nil || s.db == nil {
		return domain.LiquidationSymbolScan{}, fmt.Errorf("futures store is nil")
	}
	fromMS, toMS, exSQL, exArgs, err := liquidationDeskWindow(from, to, exchanges)
	if err != nil {
		return domain.LiquidationSymbolScan{}, err
	}
	args := []any{fromMS, toMS}
	args = append(args, exArgs...)
	query := `
SELECT exchange, symbol,
	SUM(CASE WHEN side = 'long' THEN notional ELSE 0 END),
	SUM(CASE WHEN side = 'short' THEN notional ELSE 0 END),
	COUNT(*)
FROM futures_liquidations
WHERE time_ms >= ? AND time_ms < ? AND notional > 0` + exSQL + `
GROUP BY exchange, symbol`

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return domain.LiquidationSymbolScan{}, err
	}
	defer rows.Close()
	var scan domain.LiquidationSymbolScan
	for rows.Next() {
		var row domain.LiquidationSymbolSum
		var ex string
		if err := rows.Scan(&ex, &row.Symbol, &row.Long, &row.Short, &row.Count); err != nil {
			return domain.LiquidationSymbolScan{}, err
		}
		row.Exchange = domain.Exchange(ex)
		scan.Rows = append(scan.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return domain.LiquidationSymbolScan{}, err
	}
	scan.Latest, err = s.liquidationLatestLocked(ctx, fromMS, toMS, exSQL, exArgs)
	if err != nil {
		return domain.LiquidationSymbolScan{}, err
	}
	scan.Biggest, err = s.liquidationBiggestLocked(ctx, fromMS, toMS, exSQL, exArgs)
	return scan, err
}

// BucketLiquidations rolls stored prints in [from, to) into step-sized bars.
// step must match domain.LiquidationBarGrid so a print lands in the chart column.
func (s *SQLite) BucketLiquidations(ctx context.Context, from, to time.Time, step time.Duration, exchanges []domain.Exchange) (domain.LiquidationBucketScan, error) {
	if s == nil || s.db == nil {
		return domain.LiquidationBucketScan{}, fmt.Errorf("futures store is nil")
	}
	stepMS := step.Milliseconds()
	if stepMS <= 0 {
		return domain.LiquidationBucketScan{}, fmt.Errorf("%w: liquidation bar step", domain.ErrInvalidArgument)
	}
	fromMS, toMS, exSQL, exArgs, err := liquidationDeskWindow(from, to, exchanges)
	if err != nil {
		return domain.LiquidationBucketScan{}, err
	}
	args := []any{fromMS, fromMS, stepMS, stepMS, fromMS, toMS}
	args = append(args, exArgs...)
	args = append(args, fromMS, stepMS)
	query := `
SELECT (? + ((time_ms - ?) / ?) * ?) AS start_ms,
	SUM(CASE WHEN side = 'long' THEN notional ELSE 0 END),
	SUM(CASE WHEN side = 'short' THEN notional ELSE 0 END),
	COUNT(*)
FROM futures_liquidations
WHERE time_ms >= ? AND time_ms < ? AND notional > 0` + exSQL + `
GROUP BY (time_ms - ?) / ?
ORDER BY start_ms`

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return domain.LiquidationBucketScan{}, err
	}
	defer rows.Close()
	var scan domain.LiquidationBucketScan
	for rows.Next() {
		var b domain.LiquidationBucket
		var startMS int64
		if err := rows.Scan(&startMS, &b.Long, &b.Short, &b.Count); err != nil {
			return domain.LiquidationBucketScan{}, err
		}
		b.Start = time.UnixMilli(startMS).UTC()
		scan.Buckets = append(scan.Buckets, b)
	}
	if err := rows.Err(); err != nil {
		return domain.LiquidationBucketScan{}, err
	}
	scan.Latest, err = s.liquidationLatestLocked(ctx, fromMS, toMS, exSQL, exArgs)
	return scan, err
}

func liquidationDeskWindow(from, to time.Time, exchanges []domain.Exchange) (fromMS, toMS int64, exSQL string, exArgs []any, err error) {
	if from.IsZero() || !to.After(from) {
		return 0, 0, "", nil, fmt.Errorf("%w: liquidation window", domain.ErrInvalidArgument)
	}
	exSQL, exArgs = liquidationExchangeFilter(exchanges)
	return from.UTC().UnixMilli(), to.UTC().UnixMilli(), exSQL, exArgs, nil
}

func (s *SQLite) liquidationLatestLocked(ctx context.Context, fromMS, toMS int64, exSQL string, exArgs []any) (map[domain.Exchange]time.Time, error) {
	args := []any{fromMS, toMS}
	args = append(args, exArgs...)
	rows, err := s.db.QueryContext(ctx, `
SELECT exchange, MAX(time_ms)
FROM futures_liquidations
WHERE time_ms >= ? AND time_ms < ? AND notional > 0`+exSQL+`
GROUP BY exchange`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[domain.Exchange]time.Time{}
	for rows.Next() {
		var ex string
		var ms int64
		if err := rows.Scan(&ex, &ms); err != nil {
			return nil, err
		}
		out[domain.Exchange(ex)] = time.UnixMilli(ms).UTC()
	}
	return out, rows.Err()
}

func (s *SQLite) liquidationBiggestLocked(ctx context.Context, fromMS, toMS int64, exSQL string, exArgs []any) (*domain.LiquidationEvent, error) {
	args := []any{fromMS, toMS}
	args = append(args, exArgs...)
	var ex, symbol, side string
	var ms int64
	var price, qty, notional float64
	err := s.db.QueryRowContext(ctx, `
SELECT exchange, symbol, side, time_ms, price, quantity, notional
FROM futures_liquidations
WHERE time_ms >= ? AND time_ms < ? AND notional > 0`+exSQL+`
ORDER BY notional DESC
LIMIT 1`, args...).Scan(&ex, &symbol, &side, &ms, &price, &qty, &notional)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ev := domain.LiquidationEvent{
		Exchange: domain.Exchange(ex), Symbol: symbol, Side: side,
		Time: time.UnixMilli(ms).UTC(), Price: price, Quantity: qty, Notional: notional,
	}
	return &ev, nil
}
