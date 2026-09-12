package domain

import (
	"math"
	"strings"
	"time"
)

const (
	// BacktestTradeOutcomeTP1 scaled at TP1 and fully closed there (no runner).
	BacktestTradeOutcomeTP1 = "tp1"
	// BacktestTradeOutcomeTP1TP2 scaled at TP1 then closed the rest at TP2.
	BacktestTradeOutcomeTP1TP2 = "tp1_tp2"
	// BacktestTradeOutcomeTP1SL scaled at TP1 then the rest hit the stop.
	BacktestTradeOutcomeTP1SL = "tp1_sl"
	// BacktestTradeOutcomeTP1Open scaled at TP1 and the rest is still open.
	BacktestTradeOutcomeTP1Open = "tp1_open"

	// BacktestStopKeep leaves the original stop on the runner after TP1.
	BacktestStopKeep = "keep"
	// BacktestStopEntry moves the runner stop to the fill price after TP1.
	BacktestStopEntry = "entry"
	// BacktestStopTrail ratchets the runner stop with price and never gives it back.
	BacktestStopTrail = "trail"

	// BacktestDefaultTP1ClosePct is the share closed at TP1 when TP2 is on.
	BacktestDefaultTP1ClosePct = 50.0
	backtestScaleMinClose      = 1.0
	backtestScaleMaxClose      = 99.0

	// BacktestDefaultFeePct is the Lab UI default taker fee (0.10%).
	BacktestDefaultFeePct = 0.10
	// BacktestDefaultSlipPct is the Lab UI default adverse slippage (0.05%).
	BacktestDefaultSlipPct = 0.05
	backtestCostPctMax     = 2.0

	// SwingTP1RR is the first scale (1R). TP2 stays the farther swing target.
	SwingTP1RR = 1.0
)

// BacktestScale is Lab config for TP1/TP2 and the runner stop.
type BacktestScale struct {
	UseTP2       bool    `json:"useTp2,omitempty"`
	TP1ClosePct  float64 `json:"tp1ClosePct,omitempty"`
	StopAfterTP1 string  `json:"stopAfterTp1,omitempty"` // keep | entry | trail
	FeePct       float64 `json:"feePct,omitempty"`       // 0.10 = 0.10%
	SlipPct      float64 `json:"slipPct,omitempty"`      // 0.05 = 0.05%
}

// NormalizeBacktestScale applies TP2 defaults when enabled and clamps costs.
func NormalizeBacktestScale(s BacktestScale) BacktestScale {
	out := BacktestScale{FeePct: clampLabCostPct(s.FeePct), SlipPct: clampLabCostPct(s.SlipPct)}
	if !s.UseTP2 {
		return out
	}
	pct := s.TP1ClosePct
	if pct <= 0 || math.IsNaN(pct) {
		pct = BacktestDefaultTP1ClosePct
	}
	if pct < backtestScaleMinClose {
		pct = backtestScaleMinClose
	}
	if pct > backtestScaleMaxClose {
		pct = backtestScaleMaxClose
	}
	stop := strings.ToLower(strings.TrimSpace(s.StopAfterTP1))
	switch stop {
	case BacktestStopKeep, BacktestStopTrail:
	default:
		stop = BacktestStopEntry
	}
	out.UseTP2 = true
	out.TP1ClosePct = pct
	out.StopAfterTP1 = stop
	return out
}

func clampLabCostPct(v float64) float64 {
	if v <= 0 || math.IsNaN(v) {
		return 0
	}
	if v > backtestCostPctMax {
		return backtestCostPctMax
	}
	return v
}

func (s BacktestScale) costRates() (feeRate, slipRate float64) {
	n := NormalizeBacktestScale(s)
	return n.FeePct / 100, n.SlipPct / 100
}

type labFill struct {
	entry    float64
	sideName string
	lev      int
	fee      float64
	slip     float64
}

func newLabFill(side string, slippedEntry float64, lev int, scale BacktestScale) labFill {
	fee, slip := scale.costRates()
	if lev < 1 {
		lev = 1
	}
	return labFill{sideName: side, entry: slippedEntry, lev: lev, fee: fee, slip: slip}
}

func labSlip(side string, isEntry bool, px, slipRate float64) float64 {
	buy := (side == BacktestTradeSideLong && isEntry) || (side == BacktestTradeSideShort && !isEntry)
	if buy {
		return ApplySlippage(px, TradeSideBuy, slipRate)
	}
	return ApplySlippage(px, TradeSideSell, slipRate)
}

func (f labFill) slipOut(raw float64) float64 {
	return labSlip(f.sideName, false, raw, f.slip)
}

func labFeePnLPct(entry, fillPx, frac, feeRate float64, lev int) float64 {
	if entry <= 0 || fillPx <= 0 || frac <= 0 || feeRate <= 0 {
		return 0
	}
	return -round1(feeRate * frac * (fillPx / entry) * float64(lev) * 100)
}

func (f labFill) slice(exitSlipped, frac float64) float64 {
	if frac <= 0 {
		return 0
	}
	gross := frac * LeveragedPnLPct(f.sideName, f.entry, exitSlipped, f.lev)
	entryFee := labFeePnLPct(f.entry, f.entry, frac, f.fee, f.lev)
	exitFee := labFeePnLPct(f.entry, exitSlipped, frac, f.fee, f.lev)
	return round1(gross + entryFee + exitFee)
}

func (f labFill) mark(exitSlipped, closedFrac, realizedSlice float64) float64 {
	rem := 1 - closedFrac
	if rem < 0 {
		rem = 0
	}
	mtm := rem * LeveragedPnLPct(f.sideName, f.entry, exitSlipped, f.lev)
	entryFee := labFeePnLPct(f.entry, f.entry, rem, f.fee, f.lev)
	return round1(realizedSlice + mtm + entryFee)
}

type scalePlan struct {
	tp1, tp2    float64
	finalTP     float64
	use2        bool
	closePct    float64
	stopToEntry bool
	trail       bool
	slAfter     float64
	trailDist   float64
}

func resolveScalePlan(side string, entry, sl float64, levels SwingLevels, scale BacktestScale) scalePlan {
	scale = NormalizeBacktestScale(scale)
	tp1 := levels.TakeProfit1
	tp2 := levels.TakeProfit2
	if tp1 <= 0 {
		tp1 = levels.TakeProfit
	}
	if tp2 <= 0 {
		tp2 = levels.TakeProfit
	}
	p := scalePlan{tp1: tp1, tp2: tp2, finalTP: tp1, closePct: 1, slAfter: sl}
	if !validTP(side, entry, tp1) && validTP(side, entry, tp2) {
		p.tp1, p.finalTP = tp2, tp2
		return p
	}
	if !validTP(side, entry, p.tp1) {
		return p
	}
	use2 := scale.UseTP2 && validTP(side, entry, tp2) && !samePrice(tp1, tp2) && fartherTP(side, tp2, tp1)
	if !use2 {
		return p
	}
	p.use2 = true
	p.tp2 = tp2
	p.finalTP = tp2
	p.closePct = scale.TP1ClosePct / 100
	p.stopToEntry = scale.StopAfterTP1 == BacktestStopEntry
	p.trail = scale.StopAfterTP1 == BacktestStopTrail
	p.trailDist = math.Abs(entry - sl)
	if p.stopToEntry {
		p.slAfter = entry
	}
	return p
}

func ratchetStop(side string, cur, extreme, dist float64) float64 {
	if dist <= 0 || extreme <= 0 || math.IsNaN(extreme) {
		return cur
	}
	if side == BacktestTradeSideShort {
		cand := extreme + dist
		if cur <= 0 || cand < cur {
			return cand
		}
		return cur
	}
	cand := extreme - dist
	if cand > cur {
		return cand
	}
	return cur
}

func favorableExtreme(bar OHLC, side string) float64 {
	if side == BacktestTradeSideShort {
		if bar.Low > 0 {
			return bar.Low
		}
		return bar.Close
	}
	if bar.High > 0 {
		return bar.High
	}
	return bar.Close
}

func remainingStopRisk(side string, entry, stop, notional float64) float64 {
	if entry <= 0 || notional <= 0 || stop <= 0 {
		return 0
	}
	if side == BacktestTradeSideShort {
		if stop <= entry {
			return 0
		}
		return notional * (stop - entry) / entry
	}
	if stop >= entry {
		return 0
	}
	return notional * (entry - stop) / entry
}

func stopAtOrThroughEntry(side string, entry, stop float64) bool {
	if entry <= 0 || stop <= 0 {
		return false
	}
	if side == BacktestTradeSideShort {
		return stop <= entry
	}
	return stop >= entry
}

func validTP(side string, entry, tp float64) bool {
	if entry <= 0 || tp <= 0 {
		return false
	}
	if side == BacktestTradeSideShort {
		return tp < entry
	}
	return tp > entry
}

func samePrice(a, b float64) bool {
	if a <= 0 || b <= 0 {
		return a == b
	}
	d := math.Abs(a - b)
	return d <= 1e-8 || d/math.Max(math.Abs(a), math.Abs(b)) < 1e-9
}

func fartherTP(side string, a, b float64) bool {
	if side == BacktestTradeSideShort {
		return a < b
	}
	return a > b
}

func tradeFullyClosed(tr *BacktestTrade) bool {
	if tr == nil {
		return false
	}
	switch tr.Outcome {
	case BacktestTradeOutcomeTP, BacktestTradeOutcomeSL, BacktestTradeOutcomeTP1, BacktestTradeOutcomeTP1TP2, BacktestTradeOutcomeTP1SL:
		return true
	default:
		return false
	}
}

type tradeScale struct {
	at          time.Time
	pct         float64
	price       float64
	stopToEntry bool
}

func tradeScaleEvent(tr *BacktestTrade) *tradeScale {
	if tr == nil || tr.TP1ExitPrice <= 0 || tr.TP1ClosePct <= 0 || tr.TP1ClosePct >= 100 {
		return nil
	}
	at := tradeTP1FreeAt(tr)
	if at == nil {
		return nil
	}
	return &tradeScale{
		at: at.UTC(), pct: tr.TP1ClosePct / 100, price: tr.TP1ExitPrice,
		stopToEntry: tr.StopMovedToEntry || tr.StopAfterTP1 == BacktestStopEntry,
	}
}

func tradeTP1FreeAt(tr *BacktestTrade) *time.Time {
	if tr == nil {
		return nil
	}
	if tr.TP1HitUnknown && tr.TP1KnownAt != nil && !tr.TP1KnownAt.IsZero() {
		t := tr.TP1KnownAt.UTC()
		return &t
	}
	if tr.TP1HitAt != nil && !tr.TP1HitAt.IsZero() {
		t := tr.TP1HitAt.UTC()
		return &t
	}
	if tr.TP1KnownAt != nil && !tr.TP1KnownAt.IsZero() {
		t := tr.TP1KnownAt.UTC()
		return &t
	}
	return nil
}

func scaledPnL(side string, entry, tp1Px, remPx, closePct float64, lev int) float64 {
	if closePct < 0 {
		closePct = 0
	}
	if closePct > 1 {
		closePct = 1
	}
	a := closePct * LeveragedPnLPct(side, entry, tp1Px, lev)
	b := (1 - closePct) * LeveragedPnLPct(side, entry, remPx, lev)
	return round1(a + b)
}
