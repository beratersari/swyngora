package futuresstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ListGridSnapshots returns the latest sample of each predicted flag inside
// each bucket, for every symbol in the read.
func (s *SQLite) ListGridSnapshots(ctx context.Context, q domain.GridRead) ([]domain.GridSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	symbols := domain.NormalizeFuturesSymbols(q.Symbols)
	if len(symbols) == 0 || q.Step <= 0 || q.Metric == "" {
		return nil, nil
	}
	bucket := q.Step.Milliseconds()
	var b strings.Builder
	args := []any{bucket, bucket, bucket, q.Metric}
	b.WriteString(`SELECT symbol, exchange, bucket, sampled_at_ms, predicted, value, funding_rate
		FROM (
			SELECT symbol, exchange, (sampled_at_ms / ?) * ? AS bucket, sampled_at_ms, predicted, value, funding_rate,
				ROW_NUMBER() OVER (
					PARTITION BY symbol, exchange, sampled_at_ms / ?, predicted
					ORDER BY sampled_at_ms DESC
				) AS rn
			FROM futures_snapshots
			WHERE metric = ? AND symbol IN (`)
	for i, sym := range symbols {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('?')
		args = append(args, sym)
	}
	b.WriteString(`)`)
	if !q.From.IsZero() {
		b.WriteString(` AND sampled_at_ms >= ?`)
		args = append(args, q.From.UTC().UnixMilli())
	}
	if !q.To.IsZero() {
		b.WriteString(` AND sampled_at_ms <= ?`)
		args = append(args, q.To.UTC().UnixMilli())
	}
	b.WriteString(`) AS grid WHERE rn = 1`)

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GridSnapshot
	for rows.Next() {
		var rec domain.GridSnapshot
		var ex string
		var bucketMS, sampled int64
		var pred int
		if err := rows.Scan(&rec.Symbol, &ex, &bucketMS, &sampled, &pred, &rec.Value, &rec.FundingRate); err != nil {
			return nil, err
		}
		rec.Metric = q.Metric
		rec.Exchange = domain.Exchange(ex)
		rec.Bucket = time.UnixMilli(bucketMS).UTC()
		rec.SampledAt = time.UnixMilli(sampled).UTC()
		rec.Predicted = pred == 1
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ListGridTakerSums folds stored minutes into one buy/sell total per bucket.
func (s *SQLite) ListGridTakerSums(ctx context.Context, q domain.GridRead) ([]domain.GridTakerSum, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	symbols := domain.NormalizeFuturesSymbols(q.Symbols)
	if len(symbols) == 0 || q.Step <= 0 {
		return nil, nil
	}
	bucket := q.Step.Milliseconds()
	var b strings.Builder
	args := []any{bucket, bucket}
	b.WriteString(`SELECT symbol, exchange, (start_ms / ?) * ?,
		SUM(buy_notional), SUM(sell_notional), COUNT(*), MIN(start_ms), MAX(start_ms)
		FROM taker_buckets WHERE symbol IN (`)
	for i, sym := range symbols {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('?')
		args = append(args, sym)
	}
	b.WriteString(`)`)
	if !q.From.IsZero() {
		b.WriteString(` AND start_ms >= ?`)
		args = append(args, q.From.UTC().UnixMilli())
	}
	if !q.To.IsZero() {
		b.WriteString(` AND start_ms <= ?`)
		args = append(args, q.To.UTC().UnixMilli())
	}
	b.WriteString(` GROUP BY symbol, exchange, start_ms / ?`)
	args = append(args, bucket)

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GridTakerSum
	for rows.Next() {
		var rec domain.GridTakerSum
		var ex string
		var bucketMS, first, last int64
		if err := rows.Scan(&rec.Symbol, &ex, &bucketMS, &rec.Buy, &rec.Sell, &rec.Minutes, &first, &last); err != nil {
			return nil, err
		}
		rec.Exchange = domain.Exchange(ex)
		rec.Bucket = time.UnixMilli(bucketMS).UTC()
		rec.First = time.UnixMilli(first).UTC()
		rec.Last = time.UnixMilli(last).UTC()
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ListGridLiquidations sums stored force-order notional per bucket.
func (s *SQLite) ListGridLiquidations(ctx context.Context, q domain.GridRead) ([]domain.GridLiquidationSum, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	symbols := domain.NormalizeFuturesSymbols(q.Symbols)
	if len(symbols) == 0 || q.Step <= 0 {
		return nil, nil
	}
	bucket := q.Step.Milliseconds()
	var b strings.Builder
	args := []any{bucket, bucket}
	b.WriteString(`SELECT symbol, exchange, (time_ms / ?) * ?, SUM(notional)
		FROM futures_liquidations WHERE symbol IN (`)
	for i, sym := range symbols {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('?')
		args = append(args, sym)
	}
	b.WriteString(`)`)
	if !q.From.IsZero() {
		b.WriteString(` AND time_ms >= ?`)
		args = append(args, q.From.UTC().UnixMilli())
	}
	if !q.To.IsZero() {
		b.WriteString(` AND time_ms <= ?`)
		args = append(args, q.To.UTC().UnixMilli())
	}
	b.WriteString(` GROUP BY symbol, exchange, time_ms / ?`)
	args = append(args, bucket)

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GridLiquidationSum
	for rows.Next() {
		var rec domain.GridLiquidationSum
		var ex string
		var bucketMS int64
		if err := rows.Scan(&rec.Symbol, &ex, &bucketMS, &rec.Notional); err != nil {
			return nil, err
		}
		rec.Exchange = domain.Exchange(ex)
		rec.Bucket = time.UnixMilli(bucketMS).UTC()
		out = append(out, rec)
	}
	return out, rows.Err()
}
