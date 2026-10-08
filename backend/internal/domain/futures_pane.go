package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// MaxFuturesPaneRows is the safety cap for one unbucketed window read
	// before it is folded into chart buckets. Ninety days of 5-minute
	// samples on three venues fits under this.
	MaxFuturesPaneRows = 120_000
)

// ParseFuturesBucket accepts the coin-chart steps: 5m, 30m, 2h, and 4h.
func ParseFuturesBucket(raw string) (time.Duration, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "5m":
		return 5 * time.Minute, nil
	case "30m":
		return 30 * time.Minute, nil
	case "2h":
		return 2 * time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	default:
		return 0, fmt.Errorf("%w: bucket must be 5m, 30m, 2h, or 4h", ErrInvalidArgument)
	}
}

// FuturesBucketLabel is the query value for a chart bucket.
func FuturesBucketLabel(d time.Duration) (string, error) {
	switch d {
	case 5 * time.Minute:
		return "5m", nil
	case 30 * time.Minute:
		return "30m", nil
	case 2 * time.Hour:
		return "2h", nil
	case 4 * time.Hour:
		return "4h", nil
	default:
		return "", fmt.Errorf("%w: bucket must be 5m, 30m, 2h, or 4h", ErrInvalidArgument)
	}
}

// FuturesPaneBucketForSpan matches the CVD steps to a candle span.
// Up to 24h is 5 minutes, up to 7 days is 30 minutes, up to 30 days is
// 2 hours, and anything longer is 4 hours.
func FuturesPaneBucketForSpan(span time.Duration) time.Duration {
	switch {
	case span <= 24*time.Hour:
		return 5 * time.Minute
	case span <= 7*24*time.Hour:
		return 30 * time.Minute
	case span <= 30*24*time.Hour:
		return 2 * time.Hour
	default:
		return 4 * time.Hour
	}
}

// ClampFuturesPaneWindow keeps a bucketed read inside the 90-day archive.
// A zero end time means now. A start before the cap is raised to the cap.
func ClampFuturesPaneWindow(from, to, now time.Time) (time.Time, time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	if to.IsZero() || to.After(now) {
		to = now
	} else {
		to = to.UTC()
	}
	earliest := to.Add(-FuturesHistoryMaxLookback)
	if from.IsZero() || from.Before(earliest) {
		from = earliest
	} else {
		from = from.UTC()
	}
	if from.After(to) {
		from = to
	}
	return from, to
}

// FuturesPaneNote describes a bucketed history payload.
func FuturesPaneNote(bucket time.Duration) string {
	label, err := FuturesBucketLabel(bucket)
	if err != nil {
		label = bucket.String()
	}
	return "One stored sample per " + label + " bucket across the window, capped at 90 days. " +
		"Open interest, funding, and long/short keep the last sample in the bucket. " +
		"Funding prefers a settled print over a predicted one. " +
		"Liquidations sum notional per venue and side. " +
		"A bucket with no sample is omitted and is not zero. Informational only."
}

// DownsampleSnapshots keeps the last sample in each bucket for each venue.
// Funding keeps the latest settled print in the bucket when one exists.
// A bucket with no row is left out.
func DownsampleSnapshots(metric string, rows []FuturesSnapshot, bucket time.Duration) []FuturesSnapshot {
	if bucket <= 0 || len(rows) == 0 {
		return nil
	}
	type key struct {
		ex   Exchange
		unix int64
	}
	best := map[key]FuturesSnapshot{}
	for _, row := range rows {
		if metric != "" && row.Metric != metric {
			continue
		}
		if row.SampledAt.IsZero() || row.Exchange == "" {
			continue
		}
		k := key{ex: row.Exchange, unix: TruncateToBucket(row.SampledAt, bucket).Unix()}
		prev, ok := best[k]
		if !ok || betterPaneSample(metric, prev, row) {
			best[k] = row
		}
	}
	if len(best) == 0 {
		return nil
	}
	out := make([]FuturesSnapshot, 0, len(best))
	for _, row := range best {
		if metric != "" {
			row.Metric = metric
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SampledAt.After(out[j].SampledAt) })
	return out
}

func betterPaneSample(metric string, cur, cand FuturesSnapshot) bool {
	if metric == FuturesMetricFunding {
		if cur.Predicted != cand.Predicted {
			return !cand.Predicted
		}
	}
	return cand.SampledAt.After(cur.SampledAt)
}

// DownsampleLiquidations sums notional and quantity into each bucket.
// The row time is the bucket open. A bucket with no prints is left out.
func DownsampleLiquidations(rows []LiquidationEvent, bucket time.Duration) []LiquidationEvent {
	if bucket <= 0 || len(rows) == 0 {
		return nil
	}
	type key struct {
		ex   Exchange
		side string
		unix int64
	}
	type acc struct {
		notional float64
		quantity float64
		symbol   string
	}
	sums := map[key]acc{}
	for _, row := range rows {
		if row.Time.IsZero() || row.Exchange == "" {
			continue
		}
		if row.Notional == 0 && row.Quantity == 0 {
			continue
		}
		k := key{
			ex:   row.Exchange,
			side: strings.ToLower(strings.TrimSpace(row.Side)),
			unix: TruncateToBucket(row.Time, bucket).Unix(),
		}
		cur := sums[k]
		cur.notional += row.Notional
		cur.quantity += row.Quantity
		if cur.symbol == "" {
			cur.symbol = row.Symbol
		}
		sums[k] = cur
	}
	if len(sums) == 0 {
		return nil
	}
	out := make([]LiquidationEvent, 0, len(sums))
	for k, sum := range sums {
		if sum.notional == 0 && sum.quantity == 0 {
			continue
		}
		out = append(out, LiquidationEvent{
			Exchange: k.ex,
			Symbol:   sum.symbol,
			Side:     k.side,
			Quantity: sum.quantity,
			Notional: sum.notional,
			Time:     time.Unix(k.unix, 0).UTC(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	return out
}
