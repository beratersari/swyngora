package futuresstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// AggregateLiquidationPrints groups stored force-orders into heatmap bins.
// Time and price indexes match domain.MeasuredTimeIndex and domain.MeasuredPriceIndex.
func (s *SQLite) AggregateLiquidationPrints(ctx context.Context, q domain.MeasuredPrintQuery) (domain.MeasuredPrintScan, error) {
	if s == nil || s.db == nil {
		return domain.MeasuredPrintScan{}, fmt.Errorf("futures store is nil")
	}
	symbol := domain.NormalizeLiquidationSymbol(q.Symbol)
	stepMS := q.Step.Milliseconds()
	nBin := q.PriceBins
	if nBin <= 0 {
		nBin = domain.HuntHeatmapPriceBins
	}
	if symbol == "" || stepMS <= 0 || q.PriceMax <= q.PriceMin || q.Columns <= 0 || q.Origin.IsZero() || !q.Until.After(q.Origin) {
		return domain.MeasuredPrintScan{}, fmt.Errorf("%w: incomplete print heatmap query", domain.ErrInvalidArgument)
	}
	pStep := (q.PriceMax - q.PriceMin) / float64(nBin)
	if pStep <= 0 {
		return domain.MeasuredPrintScan{}, fmt.Errorf("%w: incomplete print heatmap query", domain.ErrInvalidArgument)
	}
	originMS := q.Origin.UTC().UnixMilli()
	untilMS := q.Until.UTC().UnixMilli()
	exSQL, exArgs := liquidationExchangeFilter(q.Exchanges)

	// Placeholder order: clamped tbin, price CASE, symbol window, exchanges, raw tbin range.
	// A print one index past the last column is the open tail and stays on that column.
	args := []any{originMS, stepMS, q.Columns - 1, q.PriceMax, q.PriceMin, nBin - 1, q.PriceMax, pStep, nBin - 1, symbol, originMS, untilMS}
	args = append(args, exArgs...)
	args = append(args, originMS, stepMS, originMS, stepMS, q.Columns+1)
	query := `
SELECT MIN((time_ms - ?) / ?, ?) AS tbin,
	CASE
		WHEN price > ? THEN 0
		WHEN price < ? THEN ?
		ELSE MIN(MAX(CAST((? - price) / ? AS INTEGER), 0), ?)
	END AS pbin,
	CASE WHEN side = 'short' THEN 'short' ELSE 'long' END AS bin_side,
	exchange,
	SUM(notional) AS notional,
	COUNT(*) AS n
FROM futures_liquidations
WHERE symbol = ?
	AND time_ms >= ?
	AND time_ms < ?
	AND price > 0
	AND notional > 0` + exSQL + `
	AND (time_ms - ?) / ? >= 0
	AND (time_ms - ?) / ? < ?
GROUP BY exchange, bin_side, tbin, pbin`

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return domain.MeasuredPrintScan{}, err
	}
	defer rows.Close()
	scan := domain.MeasuredPrintScan{Latest: map[domain.Exchange]time.Time{}}
	for rows.Next() {
		var b domain.LiquidationPrintBin
		var ex string
		if err := rows.Scan(&b.TimeIndex, &b.PriceIndex, &b.Side, &ex, &b.Notional, &b.Count); err != nil {
			return domain.MeasuredPrintScan{}, err
		}
		b.Exchange = domain.Exchange(ex)
		scan.Bins = append(scan.Bins, b)
	}
	if err := rows.Err(); err != nil {
		return domain.MeasuredPrintScan{}, err
	}

	latestArgs := []any{symbol, originMS, untilMS}
	latestArgs = append(latestArgs, exArgs...)
	latestQuery := `
SELECT exchange, MAX(time_ms)
FROM futures_liquidations
WHERE symbol = ? AND time_ms >= ? AND time_ms < ? AND notional > 0` + exSQL + `
GROUP BY exchange`
	lrows, err := s.db.QueryContext(ctx, latestQuery, latestArgs...)
	if err != nil {
		return domain.MeasuredPrintScan{}, err
	}
	defer lrows.Close()
	for lrows.Next() {
		var ex string
		var ms int64
		if err := lrows.Scan(&ex, &ms); err != nil {
			return domain.MeasuredPrintScan{}, err
		}
		scan.Latest[domain.Exchange(ex)] = time.UnixMilli(ms).UTC()
	}
	return scan, lrows.Err()
}

// LiquidationPrintPriceSpan is the min and max positive print price in the window.
func (s *SQLite) LiquidationPrintPriceSpan(ctx context.Context, symbol string, from, to time.Time, exchanges []domain.Exchange) (float64, float64, error) {
	if s == nil || s.db == nil {
		return 0, 0, fmt.Errorf("futures store is nil")
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if symbol == "" || from.IsZero() || !to.After(from) {
		return 0, 0, fmt.Errorf("%w: incomplete print heatmap query", domain.ErrInvalidArgument)
	}
	exSQL, exArgs := liquidationExchangeFilter(exchanges)
	args := []any{symbol, from.UTC().UnixMilli(), to.UTC().UnixMilli()}
	args = append(args, exArgs...)
	query := `
SELECT MIN(price), MAX(price)
FROM futures_liquidations
WHERE symbol = ? AND time_ms >= ? AND time_ms < ? AND price > 0 AND notional > 0` + exSQL
	s.mu.Lock()
	defer s.mu.Unlock()
	var lo, hi sql.NullFloat64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&lo, &hi); err != nil {
		return 0, 0, err
	}
	if !lo.Valid || !hi.Valid || hi.Float64 <= 0 {
		return 0, 0, nil
	}
	return lo.Float64, hi.Float64, nil
}

func liquidationExchangeFilter(exchanges []domain.Exchange) (string, []any) {
	if len(exchanges) == 0 {
		return "", nil
	}
	var b strings.Builder
	args := make([]any, 0, len(exchanges))
	b.WriteString(" AND exchange IN (")
	for i, ex := range exchanges {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("?")
		args = append(args, string(ex))
	}
	b.WriteString(")")
	return b.String(), args
}
