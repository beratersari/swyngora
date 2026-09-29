package bookhiststore

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Compile-time check: spread context reads hourly averages, not ladder payloads.
var _ domain.SpreadUnusualReader = (*SQLite)(nil)

// HourlySpreadPct averages top-of-book spread percent inside each clock hour.
func (s *SQLite) HourlySpreadPct(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	ex := domain.ParseExchange(exchange)
	if ex == "" {
		return nil, fmt.Errorf("%w: exchange must be a spot book venue", domain.ErrInvalidArgument)
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("book history store is nil")
	}
	const q = `
SELECT (sampled_at_ms / 3600000) * 3600000, AVG(spread_pct)
FROM book_snapshots
WHERE exchange = ? AND symbol = ? AND sampled_at_ms >= ?
GROUP BY 1
ORDER BY 1`
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, q, string(ex), symbol, from.UTC().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.UnusualSample
	for rows.Next() {
		var ms int64
		var value float64
		if err := rows.Scan(&ms, &value); err != nil {
			return nil, err
		}
		out = append(out, domain.UnusualSample{At: time.UnixMilli(ms).UTC(), Value: value})
	}
	return out, rows.Err()
}
