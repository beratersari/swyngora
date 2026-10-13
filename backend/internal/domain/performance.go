package domain

import (
	"fmt"
	"math"
	"sort"
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
// Equity is mark-to-market account value and still rises when cash is added.
// ExternalFlow is the net external cash inside that equity relative to the
// baseline embedded in start equity. The performance level is Equity − ExternalFlow.
type EquityPoint struct {
	Time           time.Time
	Equity         float64
	CashBalance    float64
	PositionsValue float64
	MarginEquity   float64
	ExternalFlow   float64
}

// ExternalFlow is contributed capital after one deposit, withdrawal, or transfer.
// The opening balance is stored with NetAfter 0 and is not external cash.
type ExternalFlow struct {
	At       time.Time
	NetAfter float64
}

// FlowDelta is external cash that entered the book at At.
// Amount is negative when cash left.
type FlowDelta struct {
	At     time.Time
	Amount float64
}

// PortfolioPerformance is period trading P&L plus the equity series for a chart.
type PortfolioPerformance struct {
	ClientID    string
	Currency    string
	Period      PerformancePeriod
	StartAt     time.Time
	EndAt       time.Time
	StartEquity float64
	EndEquity   float64
	// ChangeAmount is trading P&L over the window (portfolio currency).
	// Equity change minus net external cash. Deposits, withdrawals, and
	// transfers are not profit or loss.
	ChangeAmount float64
	// ChangePct is the Modified Dietz percent for that trading P&L.
	// Nil when the capital base is ~0 or negative.
	ChangePct *float64
	// NetFlow is external cash added between the baseline and the live mark
	// (deposits + transfers in − withdrawals − transfers out).
	// The opening balance is not included.
	NetFlow float64
	// Partial is true when the portfolio is younger than the requested period.
	Partial    bool
	PointCount int
	Points     []EquityPoint
	// MaxDrawdown is the largest peak-to-trough drop of the performance line
	// (equity with external cash removed). A withdrawal is not a drawdown.
	MaxDrawdown float64
	// MaxDrawdownPct is that drop as a percent of the performance peak; nil when the peak is ~0.
	MaxDrawdownPct *float64
	// DrawdownFrom is when the peak before that drop was printed.
	DrawdownFrom time.Time
	// DrawdownTo is when the performance line reached the trough.
	DrawdownTo time.Time
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

// NetDepositsAsOf is contributed capital at at, from flows sorted by time.
// A flow at exactly at counts. No flow yet means 0 (opening balance is not net deposits).
func NetDepositsAsOf(flows []ExternalFlow, at time.Time) float64 {
	at = at.UTC()
	net := 0.0
	for _, f := range flows {
		if f.At.UTC().After(at) {
			break
		}
		net = f.NetAfter
	}
	return net
}

// FlowDeltasAfter lists external cash strictly after `after` and at or before `until`.
// flows must be sorted by time. Each amount is the change in contributed capital.
func FlowDeltasAfter(flows []ExternalFlow, after, until time.Time) []FlowDelta {
	after = after.UTC()
	until = until.UTC()
	prev := NetDepositsAsOf(flows, after)
	var out []FlowDelta
	for _, f := range flows {
		at := f.At.UTC()
		if !at.After(after) {
			continue
		}
		if at.After(until) {
			break
		}
		delta := f.NetAfter - prev
		prev = f.NetAfter
		if math.Abs(delta) < perfPctEpsilon {
			continue
		}
		out = append(out, FlowDelta{At: at, Amount: delta})
	}
	return out
}

// ModifiedDietzPct is trading P&L divided by starting equity plus time-weighted external cash.
// Cash added near the end of the window barely enters the base. Cash added at the start does.
// Nil when the base is ~0 or negative, or when an input is not a finite number.
func ModifiedDietzPct(startEquity, pnl float64, start, end time.Time, deltas []FlowDelta) *float64 {
	if math.IsNaN(startEquity) || math.IsNaN(pnl) || math.IsInf(startEquity, 0) || math.IsInf(pnl, 0) {
		return nil
	}
	denom := startEquity
	span := end.Sub(start).Seconds()
	if span > 0 {
		for _, d := range deltas {
			if math.IsNaN(d.Amount) || math.IsInf(d.Amount, 0) {
				return nil
			}
			w := end.Sub(d.At).Seconds() / span
			if w < 0 {
				w = 0
			} else if w > 1 {
				w = 1
			}
			denom += w * d.Amount
		}
	}
	if denom < perfPctEpsilon {
		return nil
	}
	v := pnl / denom * 100
	return &v
}

func sortExternalFlows(flows []ExternalFlow) []ExternalFlow {
	if len(flows) < 2 {
		return flows
	}
	ordered := make([]ExternalFlow, len(flows))
	copy(ordered, flows)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].At.Before(ordered[j].At)
	})
	return ordered
}

func flowAsOf(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return t.UTC()
}

// AssemblePerformance builds period trading P&L and the equity series.
// requestedStart is now-minus-period. carry is the last snapshot strictly before the window start.
// flows are cash-ledger marks (opening balance included, with NetAfter 0). endNetDeposits is the
// live book's contributed capital. Start net deposits follow the equity figure used as the baseline,
// which can be older than the window when the last snapshot is stale.
func AssemblePerformance(
	period PerformancePeriod,
	requestedStart, endAt, createdAt time.Time,
	startingBalance float64,
	currency string,
	clientID string,
	carry *EquitySnapshot,
	snaps []EquitySnapshot,
	live EquityPoint,
	flows []ExternalFlow,
	endNetDeposits float64,
	note string,
) PortfolioPerformance {
	requestedStart = requestedStart.UTC()
	endAt = endAt.UTC()
	createdAt = createdAt.UTC()
	flows = sortExternalFlows(flows)
	windowStart := requestedStart
	partial := false
	if !createdAt.IsZero() && createdAt.After(requestedStart) {
		windowStart = createdAt
		partial = true
	}

	// startEquity and startNet must describe the same moment. Using net deposits
	// at the window clock against a stale carry would treat the cash added after
	// that carry as trading profit.
	startEquity := startingBalance
	baselineAt := windowStart
	startNet := 0.0
	if !partial && carry != nil {
		startEquity = carry.Equity
		baselineAt = flowAsOf(carry.TakenAt)
		if baselineAt.IsZero() {
			baselineAt = flowAsOf(carry.BucketAt)
		}
		startNet = NetDepositsAsOf(flows, baselineAt)
	} else if !partial && len(snaps) > 0 && !snaps[0].BucketAt.After(windowStart) {
		startEquity = snaps[0].Equity
		baselineAt = flowAsOf(snaps[0].TakenAt)
		if baselineAt.IsZero() {
			baselineAt = flowAsOf(snaps[0].BucketAt)
		}
		startNet = NetDepositsAsOf(flows, baselineAt)
	}

	points := make([]EquityPoint, 0, len(snaps)+2)
	// Synthetic start is needed unless an in-window snapshot sits at windowStart.
	// A snapshot *before* the window must not suppress the synthetic (it is skipped below).
	needSynthetic := true
	for i := range snaps {
		if snaps[i].BucketAt.Before(windowStart) {
			continue
		}
		if !snaps[i].BucketAt.After(windowStart) {
			needSynthetic = false
		}
		break
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
		asOf := flowAsOf(s.TakenAt)
		if asOf.IsZero() {
			asOf = flowAsOf(s.BucketAt)
		}
		points = append(points, EquityPoint{
			Time:           s.TakenAt,
			Equity:         s.Equity,
			CashBalance:    s.CashBalance,
			PositionsValue: s.PositionsValue,
			MarginEquity:   s.MarginEquity,
			ExternalFlow:   NetDepositsAsOf(flows, asOf) - startNet,
		})
	}
	endEquity := live.Equity
	if live.Time.IsZero() && len(points) > 0 {
		endEquity = points[len(points)-1].Equity
	}
	liveFlow := endNetDeposits - startNet
	live.ExternalFlow = liveFlow
	// Always close with live mark so the chart reaches "now".
	if len(points) == 0 || live.Time.After(points[len(points)-1].Time) ||
		math.Abs(live.Equity-points[len(points)-1].Equity) > perfPctEpsilon {
		if live.Time.IsZero() {
			live.Time = endAt
		}
		points = append(points, live)
	} else if len(points) > 0 {
		points[len(points)-1].ExternalFlow = liveFlow
	}

	chg := TradingPnLChange(startEquity, endEquity, startNet, endNetDeposits)
	deltas := FlowDeltasAfter(flows, baselineAt, endAt)
	adjusted := make([]EquityPoint, len(points))
	for i, p := range points {
		adjusted[i] = p
		adjusted[i].Equity = p.Equity - p.ExternalFlow
	}
	ddAmt, ddPct, ddFrom, ddTo := MaxDrawdown(adjusted)
	return PortfolioPerformance{
		ClientID:       clientID,
		Currency:       currency,
		Period:         period,
		StartAt:        windowStart,
		EndAt:          endAt,
		StartEquity:    startEquity,
		EndEquity:      endEquity,
		ChangeAmount:   chg,
		ChangePct:      ModifiedDietzPct(startEquity, chg, baselineAt, endAt, deltas),
		NetFlow:        endNetDeposits - startNet,
		Partial:        partial,
		PointCount:     len(points),
		Points:         points,
		MaxDrawdown:    ddAmt,
		MaxDrawdownPct: ddPct,
		DrawdownFrom:   ddFrom,
		DrawdownTo:     ddTo,
		Note:           note,
	}
}

// MaxDrawdown is the largest fall from a running peak to a later trough.
// Amount is peak minus trough (0 when equity never fell). Percent is that
// amount over the peak. From is the peak time; to is the trough time.
func MaxDrawdown(points []EquityPoint) (amount float64, pct *float64, from, to time.Time) {
	if len(points) == 0 {
		return 0, nil, time.Time{}, time.Time{}
	}
	peak := points[0].Equity
	peakAt := points[0].Time
	best := 0.0
	bestFrom, bestTo := time.Time{}, time.Time{}
	bestPeak := peak
	for _, p := range points {
		if p.Equity > peak+perfPctEpsilon {
			peak = p.Equity
			peakAt = p.Time
			continue
		}
		drop := peak - p.Equity
		if drop > best+perfPctEpsilon {
			best = drop
			bestFrom = peakAt
			bestTo = p.Time
			bestPeak = peak
		}
	}
	if best <= perfPctEpsilon {
		return 0, nil, time.Time{}, time.Time{}
	}
	var outPct *float64
	if math.Abs(bestPeak) >= perfPctEpsilon {
		v := best / bestPeak * 100
		outPct = &v
	}
	return best, outPct, bestFrom, bestTo
}
