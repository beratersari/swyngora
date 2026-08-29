package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	FuturesMetricOpenInterest = "open_interest"
	FuturesMetricFunding      = "funding"
	FuturesMetricLongShort    = "long_short"

	DefaultFuturesHistoryLimit = 200
	MaxFuturesHistoryLimit     = 2500
)

// DefaultFuturesHistorySymbols are always sampled by the background worker.
var DefaultFuturesHistorySymbols = []string{
	"BTCUSDT", "ETHUSDT", "SOLUSDT", "XRPUSDT", "BNBUSDT",
	"DOGEUSDT", "ADAUSDT", "AVAXUSDT", "LINKUSDT", "SUIUSDT",
}

// FuturesSnapshot is one durable sample of OI, funding, or long/short.
type FuturesSnapshot struct {
	Metric        string
	Exchange      Exchange
	Symbol        string
	SampledAt     time.Time
	Predicted     bool // funding predicted-next vs settled
	Contracts     float64
	Value         float64
	FundingRate   float64
	IntervalHours int
	NextFunding   time.Time
	LongShare     float64
	ShortShare    float64
	Ratio         float64
}

// FuturesHistoryQuery filters durable samples.
type FuturesHistoryQuery struct {
	Metric   string
	Exchange string // binance | bybit | all
	Symbol   string
	From     time.Time
	To       time.Time
	Limit    int
}

// FuturesHistoryStore persists futures samples and liquidation events.
type FuturesHistoryStore interface {
	InsertSnapshot(ctx context.Context, rec FuturesSnapshot) (inserted bool, err error)
	InsertLiquidation(ctx context.Context, e LiquidationEvent) (inserted bool, err error)
	ListSnapshots(ctx context.Context, q FuturesHistoryQuery) ([]FuturesSnapshot, error)
	ListLiquidations(ctx context.Context, exchange, symbol string, from, to time.Time, limit int) ([]LiquidationEvent, error)
	ListLiquidationsSince(ctx context.Context, from time.Time, limit int) ([]LiquidationEvent, error)
	PurgeOlderThan(ctx context.Context, cutoff time.Time) (snapshots, liquidations int, err error)
}

// ParseFuturesMetric accepts open_interest, funding, long_short, or liquidations.
func ParseFuturesMetric(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case FuturesMetricOpenInterest, "oi", "open-interest", "openinterest":
		return FuturesMetricOpenInterest, nil
	case FuturesMetricFunding, "funding_rate", "funding-rate":
		return FuturesMetricFunding, nil
	case FuturesMetricLongShort, "long-short", "longshort", "ls":
		return FuturesMetricLongShort, nil
	case "liquidations", "liquidation", "liq":
		return "liquidations", nil
	case "":
		return "", fmt.Errorf("%w: metric is required", ErrInvalidArgument)
	default:
		return "", fmt.Errorf("%w: metric must be open_interest, funding, long_short, or liquidations", ErrInvalidArgument)
	}
}

// ClampFuturesHistoryLimit bounds list size.
func ClampFuturesHistoryLimit(n int) int {
	if n <= 0 {
		return DefaultFuturesHistoryLimit
	}
	if n > MaxFuturesHistoryLimit {
		return MaxFuturesHistoryLimit
	}
	return n
}

// TruncateToBucket floors t to the given UTC period (used to dedupe snapshots).
func TruncateToBucket(t time.Time, d time.Duration) time.Time {
	if t.IsZero() || d <= 0 {
		return t.UTC()
	}
	t = t.UTC()
	return time.Unix(0, t.UnixNano()-t.UnixNano()%int64(d)).UTC()
}

// NormalizeFuturesSymbols cleans and de-dupes pair ids.
func NormalizeFuturesSymbols(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		s := NormalizeLiquidationSymbol(raw)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
