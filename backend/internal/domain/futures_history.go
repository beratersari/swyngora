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

	// FuturesHistoryMaxLookback is how long stored rows stay useful to the
	// 90-day heatmap, and the longest public backfill requested when that
	// endpoint documents at least this much history.
	FuturesHistoryMaxLookback = 90 * 24 * time.Hour

	// BinanceOpenInterestHistoryLookback is GET /futures/data/openInterestHist.
	// The legacy USD-M page says only the latest 1 month is available. The
	// official USD-M connector says only the latest 30 days is available.
	BinanceOpenInterestHistoryLookback = 30 * 24 * time.Hour

	// BinanceLongShortHistoryLookback is
	// GET /futures/data/globalLongShortAccountRatio,
	// GET /futures/data/topLongShortAccountRatio, and
	// GET /futures/data/topLongShortPositionRatio.
	// Each page says only the latest 30 days is available.
	BinanceLongShortHistoryLookback = 30 * 24 * time.Hour

	// OKXOpenInterestHistoryEntries is
	// GET /api/v5/rubik/stat/contracts/open-interest-history:
	// the endpoint returns at most the latest 1,440 entries.
	// The archive requests period 1H, so the cap is 1,440 hours.
	OKXOpenInterestHistoryEntries = 1440

	// OKXLongShortHistoryEntries is
	// GET /api/v5/rubik/stat/contracts/long-short-account-ratio-contract
	// and the top-trader account and position ratios. Each returns at most
	// the latest 1,440 entries. The archive requests period 1H.
	OKXLongShortHistoryEntries = 1440
)

// PublicFuturesHistoryLookback is how far startup may ask that venue for
// this metric. Zero means this archive does not request it.
//
// Bybit GET /v5/market/open-interest can be queried back to the symbol's
// launch. Bybit GET /v5/market/account-ratio can start at 20 July 2020.
// Bybit GET /v5/market/funding/history and Binance GET /fapi/v1/fundingRate
// do not document a shorter day cap. Those series use FuturesHistoryMaxLookback.
// OKX GET /api/v5/public/funding-rate-history returns at most three months.
func PublicFuturesHistoryLookback(ex Exchange, metric string, to time.Time) time.Duration {
	if to.IsZero() {
		to = time.Now().UTC()
	}
	to = to.UTC()
	// Hyperliquid publishes settled funding. It does not publish open-interest
	// or long/short history, so those series only grow while this process samples them.
	if ex == ExchangeHyperliquid {
		if metric == FuturesMetricFunding {
			return FuturesHistoryMaxLookback
		}
		return 0
	}
	switch metric {
	case FuturesMetricOpenInterest:
		switch ex {
		case ExchangeBinance:
			return BinanceOpenInterestHistoryLookback
		case ExchangeOKX:
			return OKXOpenInterestHistoryEntries * time.Hour
		default:
			return FuturesHistoryMaxLookback
		}
	case FuturesMetricLongShort:
		switch ex {
		case ExchangeBinance:
			return BinanceLongShortHistoryLookback
		case ExchangeOKX:
			return OKXLongShortHistoryEntries * time.Hour
		default:
			return FuturesHistoryMaxLookback
		}
	case FuturesMetricFunding:
		if ex == ExchangeOKX {
			return okxFundingHistoryLookback(to)
		}
		return FuturesHistoryMaxLookback
	default:
		return FuturesHistoryMaxLookback
	}
}

// okxFundingHistoryLookback is the shorter of three calendar months and
// FuturesHistoryMaxLookback. Three months is what
// GET /api/v5/public/funding-rate-history documents.
func okxFundingHistoryLookback(to time.Time) time.Duration {
	span := to.Sub(to.AddDate(0, -3, 0))
	if span <= 0 || span > FuturesHistoryMaxLookback {
		return FuturesHistoryMaxLookback
	}
	return span
}

// ClampPublicHistoryFrom raises from so [from, to] stays inside
// PublicFuturesHistoryLookback for that series. to is the end of the request.
func ClampPublicHistoryFrom(ex Exchange, metric string, from, to time.Time) time.Time {
	if !to.After(from) {
		return from.UTC()
	}
	from, to = from.UTC(), to.UTC()
	max := PublicFuturesHistoryLookback(ex, metric, to)
	if max <= 0 {
		return from
	}
	earliest := to.Add(-max)
	if from.Before(earliest) {
		return earliest
	}
	return from
}

// FuturesHistoryPageBudget is how many hourly pages of pageSize cover
// FuturesHistoryMaxLookback, plus one page so the walk can pass the boundary.
func FuturesHistoryPageBudget(pageSize int) int {
	if pageSize <= 0 {
		return 1
	}
	hours := int(FuturesHistoryMaxLookback / time.Hour)
	pages := (hours + pageSize - 1) / pageSize
	return pages + 1
}

// DefaultFuturesHistorySymbols are always sampled by the background worker.
var DefaultFuturesHistorySymbols = []string{
	"BTCUSDT", "ETHUSDT", "SOLUSDT", "XRPUSDT", "BNBUSDT",
	"DOGEUSDT", "ADAUSDT", "AVAXUSDT", "LINKUSDT", "SUIUSDT",
}

// FuturesSnapshot is one durable sample of OI, funding, or long/short.
type FuturesSnapshot struct {
	Metric          string
	Exchange        Exchange
	Symbol          string
	SampledAt       time.Time
	Predicted       bool // funding predicted-next vs settled
	Contracts       float64
	Value           float64
	FundingRate     float64
	IntervalHours   int
	NextFunding     time.Time
	LongShare       float64
	ShortShare      float64
	Ratio           float64
	SizeLongShare   float64 // Binance top-trader position long share (0 if unknown)
	TopAccLongShare float64 // Binance top-trader account-count long share (0 if unknown)
}

// FuturesHistoryQuery filters durable samples.
// Bucket, when set, asks for one sample per venue per step across the window
// instead of the newest raw rows.
type FuturesHistoryQuery struct {
	Metric   string
	Exchange string // binance | bybit | all
	Symbol   string
	From     time.Time
	To       time.Time
	Limit    int
	Bucket   time.Duration
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
