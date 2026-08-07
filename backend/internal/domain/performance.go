package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Paper portfolio performance lookback windows.
const (
	PerformancePeriod1D PerformancePeriod = "1d"
	PerformancePeriod1W PerformancePeriod = "1w"
	PerformancePeriod1M PerformancePeriod = "1m"
	PerformancePeriod3M PerformancePeriod = "3m"

	// DefaultSnapshotInterval is the equity-history bucket (UTC truncate).
	DefaultSnapshotInterval = 15 * time.Minute
	// DefaultSnapshotRetention keeps a bit more than 3 months of buckets.
	DefaultSnapshotRetention = 100 * 24 * time.Hour

	perfPctEpsilon = 1e-9
)

// PerformancePeriod is a named lookback for equity history.
type PerformancePeriod string

// ParsePerformancePeriod accepts 1d|1w|1m|3m (case-insensitive).
func ParsePerformancePeriod(raw string) (PerformancePeriod, time.Duration, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(PerformancePeriod1W):
		return PerformancePeriod1W, 7 * 24 * time.Hour, nil
	case string(PerformancePeriod1D):
		return PerformancePeriod1D, 24 * time.Hour, nil
	case string(PerformancePeriod1M):
		return PerformancePeriod1M, 30 * 24 * time.Hour, nil
	case string(PerformancePeriod3M):
		return PerformancePeriod3M, 90 * 24 * time.Hour, nil
	default:
		return "", 0, fmt.Errorf("%w: period must be 1d, 1w, 1m, or 3m", ErrInvalidArgument)
	}
}

// SnapshotBucket floors t to interval in UTC (empty interval → default).
func SnapshotBucket(t time.Time, interval time.Duration) time.Time {
	if interval <= 0 {
		interval = DefaultSnapshotInterval
	}
	return t.UTC().Truncate(interval)
}

// EquitySnapshot is a durable mark-to-market sample for one client.
type EquitySnapshot struct {
	ClientID       string
	BucketAt       time.Time
	TakenAt        time.Time
	Equity         float64
	CashBalance    float64
	PositionsValue float64
	MarginEquity   float64
	UnrealizedPnL  float64
	RealizedPnL    float64
}

// EquityPoint is one chart/sample point.
type EquityPoint struct {
	Time           time.Time
	Equity         float64
	CashBalance    float64
	PositionsValue float64
	MarginEquity   float64
}

// PortfolioPerformance is period P&L plus the equity series for a chart.
type PortfolioPerformance struct {
	ClientID     string
	Currency     string
	Period       PerformancePeriod
	StartAt      time.Time
	EndAt        time.Time
	StartEquity  float64
	EndEquity    float64
	ChangeAmount float64
	// ChangePct is percent vs startEquity; nil when start is ~0.
	ChangePct *float64
	// Partial is true when the portfolio is younger than the requested period.
	Partial    bool
	PointCount int
	Points     []EquityPoint
	Note       string
}

// PerformanceChangePct returns (end-start)/start*100, or nil when start≈0.
func PerformanceChangePct(start, end float64) *float64 {
	if math.IsNaN(start) || math.IsNaN(end) || math.IsInf(start, 0) || math.IsInf(end, 0) {
		return nil
	}
	if math.Abs(start) < perfPctEpsilon {
		return nil
	}
	v := (end - start) / start * 100
	return &v
}

// AssemblePerformance builds period P&L + series from stored snapshots and live equity.
// requestedStart is now-minus-period. carry is the last snapshot strictly before the window start.
func AssemblePerformance(
	period PerformancePeriod,
	requestedStart, endAt, createdAt time.Time,
	startingBalance float64,
	currency string,
	clientID string,
	carry *EquitySnapshot,
	snaps []EquitySnapshot,
	live EquityPoint,
	note string,
) PortfolioPerformance {
	requestedStart = requestedStart.UTC()
	endAt = endAt.UTC()
	createdAt = createdAt.UTC()
	windowStart := requestedStart
	partial := false
	if !createdAt.IsZero() && createdAt.After(requestedStart) {
		windowStart = createdAt
		partial = true
	}

	startEquity := startingBalance
	if !partial && carry != nil {
		startEquity = carry.Equity
	} else if !partial && len(snaps) > 0 && !snaps[0].BucketAt.After(windowStart) {
		startEquity = snaps[0].Equity
	}

	points := make([]EquityPoint, 0, len(snaps)+2)
	needSynthetic := true
	if len(snaps) > 0 && !snaps[0].BucketAt.After(windowStart) {
		needSynthetic = false
	}
	if needSynthetic {
		syn := EquityPoint{Time: windowStart, Equity: startEquity}
		if carry != nil {
			syn.CashBalance = carry.CashBalance
			syn.PositionsValue = carry.PositionsValue
			syn.MarginEquity = carry.MarginEquity
		} else {
			syn.CashBalance = startingBalance
		}
		points = append(points, syn)
	}
	for _, s := range snaps {
		if s.BucketAt.Before(windowStart) {
			continue
		}
		points = append(points, EquityPoint{
			Time:           s.TakenAt,
			Equity:         s.Equity,
			CashBalance:    s.CashBalance,
			PositionsValue: s.PositionsValue,
			MarginEquity:   s.MarginEquity,
		})
	}
	// Always close with live mark so the chart reaches "now".
	if len(points) == 0 || live.Time.After(points[len(points)-1].Time) ||
		math.Abs(live.Equity-points[len(points)-1].Equity) > perfPctEpsilon {
		if live.Time.IsZero() {
			live.Time = endAt
		}
		points = append(points, live)
	}

	endEquity := live.Equity
	if live.Time.IsZero() && len(points) > 0 {
		endEquity = points[len(points)-1].Equity
	}
	chg := endEquity - startEquity
	return PortfolioPerformance{
		ClientID:     clientID,
		Currency:     currency,
		Period:       period,
		StartAt:      windowStart,
		EndAt:        endAt,
		StartEquity:  startEquity,
		EndEquity:    endEquity,
		ChangeAmount: chg,
		ChangePct:    PerformanceChangePct(startEquity, endEquity),
		Partial:      partial,
		PointCount:   len(points),
		Points:       points,
		Note:         note,
	}
}
