package futuresstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Compile-time check: the archive store can feed the unusual read without
// going through the 2,500-row snapshot list.
var (
	_ domain.FuturesUnusualReader = (*SQLite)(nil)
	_ domain.CVDUnusualReader     = (*SQLite)(nil)
)

func futuresPair(exchange, symbol string) (string, string, error) {
	ex := domain.Exchange(strings.ToLower(strings.TrimSpace(exchange)))
	if !domain.IsFuturesVenue(ex) {
		return "", "", fmt.Errorf("%w: exchange must be binance, bybit, or okx", domain.ErrInvalidArgument)
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if symbol == "" {
		return "", "", fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return string(ex), symbol, nil
}

// HourlyOpenInterest averages open interest inside each clock hour.
// Five-minute worker rows and hourly archive rows share a bucket, so a long
// baseline is not crowded out by the newest page of raw snapshots.
func (s *SQLite) HourlyOpenInterest(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	ex, symbol, err := futuresPair(exchange, symbol)
	if err != nil {
		return nil, err
	}
	const q = `
SELECT (sampled_at_ms / 3600000) * 3600000, AVG(value)
FROM futures_snapshots
WHERE metric = ? AND exchange = ? AND symbol = ? AND sampled_at_ms >= ?
GROUP BY 1
ORDER BY 1`
	return s.scanUnusual(ctx, q, domain.FuturesMetricOpenInterest, ex, symbol, from.UTC().UnixMilli())
}

// SettledFunding returns stored settlement rates. Predicted rows stay out
// of the baseline.
func (s *SQLite) SettledFunding(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	ex, symbol, err := futuresPair(exchange, symbol)
	if err != nil {
		return nil, err
	}
	const q = `
SELECT (sampled_at_ms / 3600000) * 3600000, AVG(funding_rate)
FROM futures_snapshots
WHERE metric = ? AND predicted = 0 AND exchange = ? AND symbol = ? AND sampled_at_ms >= ?
GROUP BY 1
ORDER BY 1`
	return s.scanUnusual(ctx, q, domain.FuturesMetricFunding, ex, symbol, from.UTC().UnixMilli())
}

// LatestPredictedFunding is the newest predicted-rate row, if one was stored.
func (s *SQLite) LatestPredictedFunding(ctx context.Context, exchange, symbol string) (domain.UnusualSample, bool, error) {
	ex, symbol, err := futuresPair(exchange, symbol)
	if err != nil {
		return domain.UnusualSample{}, false, err
	}
	if s == nil || s.db == nil {
		return domain.UnusualSample{}, false, fmt.Errorf("futures store is nil")
	}
	const q = `
SELECT sampled_at_ms, funding_rate
FROM futures_snapshots
WHERE metric = ? AND predicted = 1 AND exchange = ? AND symbol = ?
ORDER BY sampled_at_ms DESC
LIMIT 1`
	s.mu.Lock()
	defer s.mu.Unlock()
	var ms int64
	var rate float64
	err = s.db.QueryRowContext(ctx, q, domain.FuturesMetricFunding, ex, symbol).Scan(&ms, &rate)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UnusualSample{}, false, nil
	}
	if err != nil {
		return domain.UnusualSample{}, false, err
	}
	return domain.UnusualSample{At: time.UnixMilli(ms).UTC(), Value: rate}, true, nil
}

// HourlyLiquidationNotional sums print notional inside each clock hour.
// Hours with no prints are absent; the scorer fills those inside the
// first-to-last span.
func (s *SQLite) HourlyLiquidationNotional(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	ex, symbol, err := futuresPair(exchange, symbol)
	if err != nil {
		return nil, err
	}
	const q = `
SELECT (time_ms / 3600000) * 3600000, SUM(notional)
FROM futures_liquidations
WHERE exchange = ? AND symbol = ? AND time_ms >= ?
GROUP BY 1
ORDER BY 1`
	return s.scanUnusual(ctx, q, ex, symbol, from.UTC().UnixMilli())
}

// DailyCVD sums buy minus sell notional inside each UTC day.
func (s *SQLite) DailyCVD(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	ex, symbol, err := futuresPair(exchange, symbol)
	if err != nil {
		return nil, err
	}
	const q = `
SELECT (start_ms / 86400000) * 86400000, SUM(buy_notional - sell_notional)
FROM taker_buckets
WHERE exchange = ? AND symbol = ? AND start_ms >= ?
GROUP BY 1
ORDER BY 1`
	return s.scanUnusual(ctx, q, ex, symbol, from.UTC().UnixMilli())
}

func (s *SQLite) scanUnusual(ctx context.Context, q string, args ...any) ([]domain.UnusualSample, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("futures store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, q, args...)
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
