package domain

import (
	"math"
	"sort"
	"time"
)

type pathSample struct {
	at time.Time
	px float64
}

// pathHold is one open row and, when the venue priced, the closes after the
// window's starting close.
type pathHold struct {
	spot        bool
	priced      bool
	startPx     float64
	startAt     time.Time
	quantity    float64
	mark        float64
	marketValue float64
	unrealized  float64
	side        MarginSide
	margin      float64
	liqPrice    float64
	samples     []pathSample
}

// ReplayPath reprices today's open book at each close in the window.
// The first point is the book now. A margin position that liquidates stays
// liquidated on later closes. A venue the end-of-window stress left out stays
// at its current value for the whole path.
func ReplayPath(v PortfolioView, series []ReplaySeries, plan ReplayPlan, to time.Time, interval CandleInterval, now time.Time) []StressPathPoint {
	holds := replayHolds(v, series, plan, to, interval, now)
	if !pathHasPrice(holds) {
		return nil
	}
	times := replayTimes(holds)
	if len(times) == 0 {
		return nil
	}
	cursors := make([]int, len(holds))
	for i := range cursors {
		cursors[i] = -1
	}
	liquidated := make([]bool, len(holds))
	points := make([]StressPathPoint, 0, len(times))
	for _, at := range times {
		prices := make([]float64, len(holds))
		for i, h := range holds {
			if !h.priced {
				continue
			}
			for cursors[i]+1 < len(h.samples) && !h.samples[cursors[i]+1].at.After(at) {
				cursors[i]++
			}
			px := h.startPx
			if cursors[i] >= 0 {
				px = h.samples[cursors[i]].px
			}
			prices[i] = px
		}
		equity, nLiq := pathEquity(v, holds, prices, liquidated)
		points = append(points, StressPathPoint{
			At: at, Equity: equity, Delta: equity - v.Equity, DeltaPct: pathDeltaPct(v.Equity, equity), Liquidated: nLiq,
		})
	}
	return points
}

// PathLow is the lowest equity on the path. Equal lows keep the earlier close.
func PathLow(points []StressPathPoint) (StressPathPoint, bool) {
	if len(points) == 0 {
		return StressPathPoint{}, false
	}
	low := points[0]
	for _, p := range points[1:] {
		if p.Equity < low.Equity {
			low = p
		}
	}
	return low, true
}

func pathHasPrice(holds []pathHold) bool {
	for _, h := range holds {
		if h.priced {
			return true
		}
	}
	return false
}

func replayHolds(v PortfolioView, series []ReplaySeries, plan ReplayPlan, to time.Time, interval CandleInterval, now time.Time) []pathHold {
	legs := replayLegs(series, plan, to, interval, now)
	var holds []pathHold
	for _, p := range v.Positions {
		if p.Quantity <= 0 {
			continue
		}
		mark := p.MarkPrice
		if mark <= 0 {
			mark = p.AvgCost
		}
		h := pathHold{
			spot: true, quantity: p.Quantity, mark: mark,
			marketValue: p.MarketValue, unrealized: p.UnrealizedPnL,
		}
		if leg, ok := legs[stressPosKey(string(p.Exchange), p.Symbol)]; ok {
			h.priced = true
			h.startPx = leg.startPx
			h.startAt = leg.startAt
			h.samples = leg.samples
		}
		holds = append(holds, h)
	}
	for _, p := range v.MarginPositions {
		if (p.Status != "" && p.Status != MarginPositionOpen) || p.Quantity <= 0 {
			continue
		}
		mark := p.MarkPrice
		if mark <= 0 {
			mark = p.EntryPrice
		}
		h := pathHold{
			quantity: p.Quantity, mark: mark, unrealized: p.UnrealizedPnL,
			side: p.Side, margin: p.Margin, liqPrice: p.LiquidationPrice,
		}
		if leg, ok := legs[stressPosKey(string(p.Exchange), p.Symbol)]; ok {
			h.priced = true
			h.startPx = leg.startPx
			h.startAt = leg.startAt
			h.samples = leg.samples
		}
		holds = append(holds, h)
	}
	return holds
}

type replayLeg struct {
	startPx float64
	startAt time.Time
	samples []pathSample
}

func replayLegs(series []ReplaySeries, plan ReplayPlan, to time.Time, interval CandleInterval, now time.Time) map[string]replayLeg {
	included := map[string]StressReplayVenue{}
	for _, move := range plan.Replay.Moves {
		for _, venue := range move.Venues {
			included[stressPosKey(venue.Exchange, venue.Symbol)] = venue
		}
	}
	if len(included) == 0 {
		return nil
	}
	step := IntervalDuration(interval)
	legs := map[string]replayLeg{}
	for _, row := range series {
		if row.Failed {
			continue
		}
		key := stressPosKey(row.Exchange, row.Symbol)
		venue, ok := included[key]
		if !ok || venue.Start <= 0 || venue.StartAt.IsZero() {
			continue
		}
		if _, exists := legs[key]; exists {
			continue
		}
		legs[key] = replayLeg{
			startPx: venue.Start,
			startAt: venue.StartAt.UTC(),
			samples: legSamples(row.Bars, step, venue.Start, venue.StartAt.UTC(), to, now),
		}
	}
	return legs
}

func legSamples(bars []Candle, step time.Duration, startPx float64, startAt, to, now time.Time) []pathSample {
	var out []pathSample
	for _, bar := range closedReplayBars(bars, step, now) {
		closeAt := ReplayBarClose(bar, step)
		if !closeAt.After(startAt) || closeAt.After(to) {
			continue
		}
		px, ok := parseReplayClose(bar.Close)
		if !ok {
			continue
		}
		if !validStressPct((px/startPx - 1) * 100) {
			continue
		}
		out = append(out, pathSample{at: closeAt.UTC(), px: px})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].at.Before(out[j].at) })
	return out
}

func replayTimes(holds []pathHold) []time.Time {
	seen := map[int64]struct{}{}
	var times []time.Time
	add := func(at time.Time) {
		if at.IsZero() {
			return
		}
		at = at.UTC()
		key := at.UnixNano()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		times = append(times, at)
	}
	for _, h := range holds {
		if !h.priced {
			continue
		}
		add(h.startAt)
		for _, sample := range h.samples {
			add(sample.at)
		}
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	return times
}

func pathEquity(v PortfolioView, holds []pathHold, prices []float64, liquidated []bool) (float64, int) {
	var spotValue, marginU float64
	nLiq := 0
	for i, h := range holds {
		if h.spot {
			if !h.priced || h.mark <= 0 || h.startPx <= 0 || prices[i] <= 0 {
				spotValue += h.marketValue
				continue
			}
			spotValue += h.quantity * (h.mark * (prices[i] / h.startPx))
			continue
		}
		if !h.priced || h.mark <= 0 || h.startPx <= 0 || prices[i] <= 0 {
			marginU += h.unrealized
			continue
		}
		if liquidated[i] {
			marginU += -h.margin
			nLiq++
			continue
		}
		next := h.mark * (prices[i] / h.startPx)
		u := h.unrealized + MarginUnrealizedPnL(h.side, h.quantity, h.mark, next)
		if ShouldLiquidate(h.side, next, h.liqPrice) {
			liquidated[i] = true
			u = -h.margin
			nLiq++
		}
		marginU += u
	}
	_, equity := shockEquity(v, spotValue, marginU)
	return equity, nLiq
}

func pathDeltaPct(equity, next float64) *float64 {
	if !(equity > 0) {
		return nil
	}
	pct := (next - equity) / equity * 100
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		return nil
	}
	return &pct
}
