package domain

import (
	"math"
	"strings"
	"time"
)

const (
	// BacktestTradeSideLong / Short are paper-style directions.
	BacktestTradeSideLong  = "long"
	BacktestTradeSideShort = "short"
	// First-touch outcomes. Open means neither TP nor SL traded before the last bar.
	BacktestTradeOutcomeTP   = "tp"
	BacktestTradeOutcomeSL   = "sl"
	BacktestTradeOutcomeOpen = "open"
	// BacktestMaxLeverage matches paper margin (1–10x).
	BacktestMaxLeverage = 10
	// BacktestTargetStopLossPct sizes leverage so a full stop is about this % of notional.
	BacktestTargetStopLossPct = 10.0
	// backtestFinerMaxDepth is 1d→1h→15m→5m→1m.
	backtestFinerMaxDepth = 5
)

// FineBarLookup returns smaller candles inside parent for the given interval.
// Used when one parent bar touches both TP and SL. Empty means "cannot resolve".
type FineBarLookup func(parent OHLC, interval CandleInterval) []OHLC

// BacktestTouchOpts is optional first-touch resolution (smaller bars).
type BacktestTouchOpts struct {
	Interval CandleInterval
	Finer    FineBarLookup
	// RangeEnd clips the Lab window. Hits after this are still recorded, but
	// RangePnLPct marks the last in-range close.
	RangeEnd time.Time
	Scale    BacktestScale
}

// BacktestTrade is the signal's plan plus first-touch result.
type BacktestTrade struct {
	Side             string              `json:"side"`
	Leverage         int                 `json:"leverage"`
	Entry            float64             `json:"entry"`
	StopLoss         float64             `json:"stopLoss"`
	TakeProfit       float64             `json:"takeProfit"`
	TakeProfit1      float64             `json:"takeProfit1,omitempty"`
	TakeProfit2      float64             `json:"takeProfit2,omitempty"`
	UseTP2           bool                `json:"useTp2,omitempty"`
	TP1ClosePct      float64             `json:"tp1ClosePct,omitempty"`
	StopAfterTP1     string              `json:"stopAfterTp1,omitempty"`
	StopMovedToEntry bool                `json:"stopMovedToEntry,omitempty"`
	Outcome          string              `json:"outcome"` // tp | sl | open | tp1 | tp1_tp2 | tp1_sl | tp1_open
	ExitPrice        float64             `json:"exitPrice,omitempty"`
	PnLPct           float64             `json:"pnlPct"` // leveraged percent of the whole fill
	BarsHeld         int                 `json:"barsHeld,omitempty"`
	HitAt            *time.Time          `json:"hitAt,omitempty"`
	HitTimeUnknown   bool                `json:"hitTimeUnknown,omitempty"`
	ClosedKnownAt    *time.Time          `json:"closedKnownAt,omitempty"`
	EnteredAt        *time.Time          `json:"enteredAt,omitempty"`
	ClosedAfterRange bool                `json:"closedAfterRange,omitempty"`
	RangeExitPrice   float64             `json:"rangeExitPrice,omitempty"`
	RangePnLPct      float64             `json:"rangePnlPct,omitempty"`
	TP1ExitPrice     float64             `json:"tp1ExitPrice,omitempty"`
	TP1PnLPct        float64             `json:"tp1PnlPct,omitempty"`
	TP1HitAt         *time.Time          `json:"tp1HitAt,omitempty"`
	TP1HitUnknown    bool                `json:"tp1HitTimeUnknown,omitempty"`
	TP1KnownAt       *time.Time          `json:"tp1KnownAt,omitempty"`
	RemainderPct     float64             `json:"remainderPct,omitempty"`
	Marks            []BacktestPriceMark `json:"marks,omitempty"`
}

// BacktestPriceMark is one print on a fill while it is open.
type BacktestPriceMark struct {
	At    time.Time `json:"at"`
	Price float64   `json:"price"`
}

// BacktestTradeStats summarizes first-touch trades across a job.
type BacktestTradeStats struct {
	SampleCount int      `json:"sampleCount"`
	TpHits      int      `json:"tpHits"`
	SlHits      int      `json:"slHits"`
	OpenCount   int      `json:"openCount"`
	LongCount   int      `json:"longCount"`
	ShortCount  int      `json:"shortCount"`
	WinRate     *float64 `json:"winRate,omitempty"` // 0–100, TP / (TP+SL)
	Average     *float64 `json:"average,omitempty"` // leveraged PnL of closed trades
	Median      *float64 `json:"median,omitempty"`
}

// NormalizeTradeSide maps unknown values to long.
func NormalizeTradeSide(side string) string {
	if strings.EqualFold(strings.TrimSpace(side), BacktestTradeSideShort) {
		return BacktestTradeSideShort
	}
	return BacktestTradeSideLong
}

// SuggestedSwingLeverage keeps a full stop near BacktestTargetStopLossPct (capped at 10x).
func SuggestedSwingLeverage(riskPct float64) int {
	if riskPct <= 0 || math.IsNaN(riskPct) {
		return 1
	}
	lev := int(math.Floor(BacktestTargetStopLossPct/riskPct + 1e-9))
	if lev < 1 {
		return 1
	}
	if lev > BacktestMaxLeverage {
		return BacktestMaxLeverage
	}
	return lev
}

// ScannerRuleTradeSide is long unless every selected condition is short-leaning.
func ScannerRuleTradeSide(r ScannerRule) string {
	conds := r.SelectedConditions()
	if len(conds) == 0 {
		return BacktestTradeSideLong
	}
	short, long := 0, 0
	for _, c := range conds {
		switch c {
		case ScannerRuleRSI:
			if r.RSICondition == AlertBelow {
				short++
			} else {
				long++
			}
		case ScannerRuleMACrossover:
			if strings.EqualFold(strings.TrimSpace(r.MADirection), "death_cross") {
				short++
			} else {
				long++
			}
		default:
			long++
		}
	}
	if short > 0 && long == 0 {
		return BacktestTradeSideShort
	}
	return BacktestTradeSideLong
}

// OHLCFromCandles keeps index alignment. Missing open stays 0 (Lab must not
// invent a fill from the close). Missing high/low fall back to close, or to
// open+close when the open is known — never to a zero open.
func OHLCFromCandles(candles []Candle) []OHLC {
	out := make([]OHLC, len(candles))
	for i, c := range candles {
		cl, err := parseClose(c.Close)
		if err != nil || cl <= 0 || math.IsNaN(cl) {
			continue
		}
		o, oerr := parseClose(c.Open)
		if oerr != nil || o <= 0 || math.IsNaN(o) {
			o = 0
		}
		h, herr := parseClose(c.High)
		if herr != nil || h <= 0 || math.IsNaN(h) {
			if o > 0 {
				h = math.Max(o, cl)
			} else {
				h = cl
			}
		}
		l, lerr := parseClose(c.Low)
		if lerr != nil || l <= 0 || math.IsNaN(l) {
			if o > 0 {
				l = math.Min(o, cl)
			} else {
				l = cl
			}
		}
		if o > 0 {
			if h < math.Max(o, cl) {
				h = math.Max(o, cl)
			}
			if l > math.Min(o, cl) {
				l = math.Min(o, cl)
			}
		} else {
			if h < cl {
				h = cl
			}
			if l <= 0 || l > cl {
				l = cl
			}
		}
		out[i] = OHLC{
			OpenTime: c.OpenTime, CloseTime: c.CloseTime,
			Open: o, High: h, Low: l, Close: cl,
		}
	}
	return out
}

// SimulateBacktestTrade walks bars after signalIdx and records which of TP or SL trades first.
func SimulateBacktestTrade(bars []OHLC, signalIdx int, side string, levels SwingLevels) *BacktestTrade {
	return SimulateBacktestTradeOpts(bars, signalIdx, side, levels, BacktestTouchOpts{})
}

// SimulateBacktestTradeOpts is SimulateBacktestTrade with optional finer bars.
// When one parent bar wicks both TP and SL, Finer is used to see which printed first.
func SimulateBacktestTradeOpts(bars []OHLC, signalIdx int, side string, levels SwingLevels, opts BacktestTouchOpts) *BacktestTrade {
	if signalIdx < 0 || signalIdx >= len(bars) {
		return nil
	}
	entry := levels.Entry
	if entry <= 0 {
		entry = bars[signalIdx].Close
	}
	if entry <= 0 {
		return nil
	}
	side = NormalizeTradeSide(side)
	sl := levels.StopLoss
	entryIdx := signalIdx + 1
	if entryIdx >= len(bars) {
		return nil
	}
	nextOpen := bars[entryIdx].Open
	if nextOpen <= 0 {
		return nil
	}
	entry = nextOpen
	plan := resolveScalePlan(side, entry, sl, levels, opts.Scale)
	lev := clampBacktestLeverage(levels)
	out := &BacktestTrade{
		Side: side, Leverage: lev, Entry: round8(entry),
		StopLoss: round8(sl), TakeProfit: round8(plan.finalTP),
		Outcome: BacktestTradeOutcomeOpen,
	}
	if plan.tp1 > 0 {
		out.TakeProfit1 = round8(plan.tp1)
	}
	if plan.use2 {
		out.TakeProfit2 = round8(plan.tp2)
		out.UseTP2 = true
		out.TP1ClosePct = plan.closePct * 100
		if plan.stopToEntry {
			out.StopAfterTP1 = BacktestStopEntry
		} else {
			out.StopAfterTP1 = BacktestStopKeep
		}
	}
	if !bars[entryIdx].OpenTime.IsZero() {
		t := bars[entryIdx].OpenTime.UTC()
		out.EnteredAt = &t
	}
	out.Marks = appendBacktestMark(out.Marks, bars[entryIdx].OpenTime, entry)

	if hit := throughAtOpenPrice(side, entry, sl, plan.tp1); hit == BacktestTradeOutcomeSL {
		return finishThroughOpen(out, hit, entry)
	}
	if plan.use2 {
		if throughAtOpenPrice(side, entry, sl, plan.tp2) == BacktestTradeOutcomeTP {
			return finishThroughOpen(out, BacktestTradeOutcomeTP, entry)
		}
		if throughAtOpenPrice(side, entry, sl, plan.tp1) == BacktestTradeOutcomeTP {
			applyTP1Scale(out, entry, entry, out.EnteredAt, false, out.EnteredAt, plan)
			if plan.stopToEntry {
				out.StopMovedToEntry = true
			}
		}
	} else if hit := throughAtOpenPrice(side, entry, sl, plan.tp1); hit != "" {
		return finishThroughOpen(out, hit, entry)
	}

	rangeEnd := opts.RangeEnd.UTC()
	markRange := func(px float64) {
		if px <= 0 {
			return
		}
		out.RangeExitPrice = round8(px)
		if out.TP1ExitPrice > 0 && plan.use2 && plan.closePct < 1 {
			out.RangePnLPct = scaledPnL(side, entry, out.TP1ExitPrice, px, plan.closePct, lev)
		} else {
			out.RangePnLPct = LeveragedPnLPct(side, entry, px, lev)
		}
	}
	phase := 1
	activeSL, activeTP := sl, plan.tp1
	if out.TP1ExitPrice > 0 {
		phase = 2
		activeSL = plan.slAfter
		activeTP = plan.tp2
	}
	lastInRange := -1
	for j := entryIdx; j < len(bars); j++ {
		bar := bars[j]
		if j > entryIdx {
			out.Marks = appendBacktestMark(out.Marks, bar.OpenTime, bar.Open)
		}
		if bar.Close <= 0 && bar.High <= 0 {
			continue
		}
		closedByEnd := rangeEnd.IsZero() || barClosedAtOrBefore(bar, rangeEnd, opts.Interval)
		spansEnd := !rangeEnd.IsZero() && !bar.OpenTime.UTC().After(rangeEnd) && !closedByEnd
		if closedByEnd {
			lastInRange = j
			if bar.Close > 0 {
				markRange(bar.Close)
			} else {
				markRange(entry)
			}
		} else if spansEnd {
			if px := priceAtOrBefore(bar, rangeEnd, opts.Interval, opts.Finer, 0); px > 0 {
				markRange(px)
				lastInRange = j
			}
		}

		if phase == 1 && plan.use2 {
			touch := resolveFirstTouch(bar, side, activeSL, plan.tp1, opts.Interval, opts.Finer, 0)
			if touch.hit == "" {
				out.Marks = appendBacktestMark(out.Marks, barMarkAt(bar, opts.Interval), bar.Close)
				continue
			}
			if touch.hit == BacktestTradeOutcomeSL {
				applyTouchExit(out, touch, bar, bars, j, entryIdx, side, entry, lev, rangeEnd, spansEnd, opts.Interval, markRange)
				return out
			}
			applyTP1Scale(out, touch.price, entry, touchTime(touch), touch.at.IsZero(), knownByOrBar(touch, bar, bars, j, opts.Interval), plan)
			if plan.stopToEntry {
				out.StopMovedToEntry = true
			}
			phase = 2
			activeSL = plan.slAfter
			activeTP = plan.tp2
			rem := remainderFirstTouch(bar, side, activeSL, activeTP, opts, touch)
			if rem.hit == "" {
				continue
			}
			applyRemainderExit(out, rem, bar, bars, j, entryIdx, side, entry, lev, rangeEnd, spansEnd, opts.Interval, plan, markRange)
			return out
		}

		touch := resolveFirstTouch(bar, side, activeSL, activeTP, opts.Interval, opts.Finer, 0)
		if touch.hit == "" {
			out.Marks = appendBacktestMark(out.Marks, barMarkAt(bar, opts.Interval), bar.Close)
			continue
		}
		if phase == 2 {
			applyRemainderExit(out, touch, bar, bars, j, entryIdx, side, entry, lev, rangeEnd, spansEnd, opts.Interval, plan, markRange)
		} else {
			applyTouchExit(out, touch, bar, bars, j, entryIdx, side, entry, lev, rangeEnd, spansEnd, opts.Interval, markRange)
		}
		return out
	}
	if lastInRange >= 0 && out.RangeExitPrice > 0 {
		out.ExitPrice = out.RangeExitPrice
		out.PnLPct = out.RangePnLPct
		out.BarsHeld = lastInRange - entryIdx
		if phase == 2 {
			out.Outcome = BacktestTradeOutcomeTP1Open
		}
		return out
	}
	last := bars[len(bars)-1]
	exit := last.Close
	if exit <= 0 {
		exit = entry
	}
	out.ExitPrice = round8(exit)
	out.BarsHeld = len(bars) - 1 - entryIdx
	if out.BarsHeld < 0 {
		out.BarsHeld = 0
	}
	if phase == 2 {
		out.Outcome = BacktestTradeOutcomeTP1Open
		out.PnLPct = scaledPnL(side, entry, out.TP1ExitPrice, exit, plan.closePct, lev)
	} else {
		out.PnLPct = LeveragedPnLPct(side, entry, exit, lev)
	}
	if out.RangeExitPrice <= 0 {
		markRange(exit)
	}
	return out
}

func finishThroughOpen(out *BacktestTrade, hit string, px float64) *BacktestTrade {
	out.Outcome = hit
	out.ExitPrice = round8(px)
	out.PnLPct = 0
	out.BarsHeld = 0
	out.HitAt = out.EnteredAt
	out.ClosedKnownAt = out.EnteredAt
	out.RangeExitPrice = out.ExitPrice
	out.RangePnLPct = 0
	return out
}

func applyTP1Scale(out *BacktestTrade, px, entry float64, at *time.Time, unknown bool, knownBy *time.Time, plan scalePlan) {
	out.TP1ExitPrice = round8(px)
	out.TP1PnLPct = LeveragedPnLPct(out.Side, entry, px, out.Leverage)
	out.TP1HitAt = at
	out.TP1HitUnknown = unknown
	out.TP1KnownAt = knownBy
	out.RemainderPct = round1((1 - plan.closePct) * 100)
	out.Outcome = BacktestTradeOutcomeTP1Open
}

func applyTouchExit(out *BacktestTrade, touch firstTouch, bar OHLC, bars []OHLC, j, entryIdx int, side string, entry float64, lev int, rangeEnd time.Time, spansEnd bool, iv CandleInterval, markRange func(float64)) {
	stampTouch(out, touch, bar, bars, j, iv)
	hitInRange := touchInRange(out, touch, rangeEnd, spansEnd)
	out.Outcome = touch.hit
	out.ExitPrice = round8(touch.price)
	out.BarsHeld = j - entryIdx
	out.PnLPct = LeveragedPnLPct(side, entry, touch.price, lev)
	if !hitInRange {
		out.ClosedAfterRange = true
	} else {
		markRange(touch.price)
	}
}

func applyRemainderExit(out *BacktestTrade, touch firstTouch, bar OHLC, bars []OHLC, j, entryIdx int, side string, entry float64, lev int, rangeEnd time.Time, spansEnd bool, iv CandleInterval, plan scalePlan, markRange func(float64)) {
	stampTouch(out, touch, bar, bars, j, iv)
	hitInRange := touchInRange(out, touch, rangeEnd, spansEnd)
	if touch.hit == BacktestTradeOutcomeTP {
		out.Outcome = BacktestTradeOutcomeTP1TP2
	} else {
		out.Outcome = BacktestTradeOutcomeTP1SL
	}
	out.ExitPrice = round8(touch.price)
	out.BarsHeld = j - entryIdx
	out.PnLPct = scaledPnL(side, entry, out.TP1ExitPrice, touch.price, plan.closePct, lev)
	if !hitInRange {
		out.ClosedAfterRange = true
	} else {
		markRange(touch.price)
	}
}

func stampTouch(out *BacktestTrade, touch firstTouch, bar OHLC, bars []OHLC, j int, iv CandleInterval) {
	barEnd := knownByOrEnd(touch, bar, bars, j, iv)
	if !touch.at.IsZero() {
		t := touch.at.UTC()
		out.HitAt = &t
		out.ClosedKnownAt = &t
		return
	}
	out.HitTimeUnknown = true
	if !barEnd.IsZero() {
		t := barEnd.UTC()
		out.ClosedKnownAt = &t
	}
}

func touchInRange(out *BacktestTrade, touch firstTouch, rangeEnd time.Time, spansEnd bool) bool {
	freeAt := TradeRiskFreeAt(out)
	hitInRange := rangeEnd.IsZero() || (freeAt != nil && !freeAt.After(rangeEnd))
	if !touch.at.IsZero() {
		return rangeEnd.IsZero() || !touch.at.UTC().After(rangeEnd)
	}
	if spansEnd {
		return false
	}
	return hitInRange
}

func knownByOrEnd(touch firstTouch, bar OHLC, bars []OHLC, j int, iv CandleInterval) time.Time {
	barEnd := touch.knownBy
	if barEnd.IsZero() || !barEnd.After(bar.OpenTime) {
		if _, end := ParentBarWindow(bar, iv); end.After(bar.OpenTime) {
			barEnd = end
		} else if j+1 < len(bars) && bars[j+1].OpenTime.After(bar.OpenTime) {
			barEnd = bars[j+1].OpenTime.UTC()
		}
	}
	return barEnd
}

func knownByOrBar(touch firstTouch, bar OHLC, bars []OHLC, j int, iv CandleInterval) *time.Time {
	end := knownByOrEnd(touch, bar, bars, j, iv)
	if end.IsZero() {
		return nil
	}
	t := end.UTC()
	return &t
}

func touchTime(touch firstTouch) *time.Time {
	if touch.at.IsZero() {
		return nil
	}
	t := touch.at.UTC()
	return &t
}

func remainderFirstTouch(bar OHLC, side string, sl, tp float64, opts BacktestTouchOpts, tp1 firstTouch) firstTouch {
	// Scale at this bar's open: the rest of the bar is after TP1.
	if !tp1.at.IsZero() && !bar.OpenTime.IsZero() && tp1.at.UTC().Equal(bar.OpenTime.UTC()) {
		return resolveFirstTouch(bar, side, sl, tp, opts.Interval, opts.Finer, 0)
	}
	if rem, ok := remainderAfterTP1(bar, side, sl, tp, opts, tp1); ok {
		return rem
	}
	// No known order. A same-bar pullback to the new entry stop may have printed
	// before TP1. Do not invent a runner result. TP2-only is safe: price had to
	// pass TP1 to reach TP2, and the new stop was not tagged.
	hit, px, both := inspectBarTouch(bar, side, sl, tp)
	if both || hit != BacktestTradeOutcomeTP {
		return firstTouch{}
	}
	_, end := ParentBarWindow(bar, opts.Interval)
	out := firstTouch{hit: hit, price: px, knownBy: end}
	if openGapsLevel(bar, side, sl, tp) {
		out.at = bar.OpenTime.UTC()
	}
	return out
}

// remainderAfterTP1 walks smaller bars after the TP1 print. ok is true when the
// tape after TP1 was actually walked (so an empty hit means "no remainder yet").
func remainderAfterTP1(bar OHLC, side string, sl, tp float64, opts BacktestTouchOpts, tp1 firstTouch) (firstTouch, bool) {
	if opts.Finer == nil {
		return firstTouch{}, false
	}
	next, ok := FinerBacktestInterval(opts.Interval)
	if !ok {
		if span := barSpan(bar, opts.Interval); span > 0 {
			next, ok = FinerBacktestIntervalForDuration(span)
		}
	}
	if !ok {
		return firstTouch{}, false
	}
	parent := bar
	if parent.CloseTime.IsZero() || !parent.CloseTime.After(parent.OpenTime) {
		if _, end := ParentBarWindow(bar, opts.Interval); end.After(bar.OpenTime) {
			parent.CloseTime = end
		}
	}
	sub := OHLCInside(opts.Finer(parent, next), parent, opts.Interval)
	start, _ := ParentBarWindow(parent, opts.Interval)
	canWalk := finerCoversParent(sub, parent, opts.Interval, next)
	if !canWalk && len(sub) > 0 && !start.IsZero() && sub[0].OpenTime.UTC().Equal(start) {
		canWalk = true
	}
	if !canWalk {
		return firstTouch{}, false
	}
	tp1Px := tp1.price
	if tp1Px <= 0 {
		tp1Px = tp
	}
	seen := !tp1.at.IsZero() && !bar.OpenTime.IsZero() && !tp1.at.Before(bar.OpenTime)
	for _, sb := range sub {
		if !seen {
			if !tagsTP(sb, side, tp1Px) {
				continue
			}
			seen = true
			// This sub-bar printed TP1. Remainder on it only if TP1 was the open.
			if throughAtOpenPrice(side, sb.Open, sl, tp1Px) != BacktestTradeOutcomeTP {
				continue
			}
		}
		got := resolveFirstTouch(sb, side, sl, tp, next, opts.Finer, 1)
		if got.hit != "" {
			return got, true
		}
	}
	return firstTouch{}, seen
}

func tagsTP(bar OHLC, side string, tp float64) bool {
	if tp <= 0 {
		return false
	}
	if side == BacktestTradeSideShort {
		if bar.Open > 0 && bar.Open <= tp {
			return true
		}
		return bar.Low > 0 && bar.Low <= tp
	}
	if bar.Open > 0 && bar.Open >= tp {
		return true
	}
	return bar.High >= tp
}

func barClosedAtOrBefore(bar OHLC, at time.Time, iv CandleInterval) bool {
	_, end := ParentBarWindow(bar, iv)
	if end.IsZero() || !end.After(bar.OpenTime) {
		return !bar.OpenTime.UTC().After(at.UTC())
	}
	return !end.After(at.UTC())
}

// priceAtOrBefore is the last print at or before at. An unfinished parent bar
// walks finer tape; if that is missing, the parent open is used (not its later close).
func priceAtOrBefore(bar OHLC, at time.Time, iv CandleInterval, finer FineBarLookup, depth int) float64 {
	at = at.UTC()
	if barClosedAtOrBefore(bar, at, iv) {
		if bar.Close > 0 {
			return bar.Close
		}
		return bar.Open
	}
	if bar.OpenTime.UTC().After(at) {
		return 0
	}
	if finer != nil && depth < backtestFinerMaxDepth {
		next, ok := FinerBacktestInterval(iv)
		if !ok {
			if span := barSpan(bar, iv); span > 0 {
				next, ok = FinerBacktestIntervalForDuration(span)
			}
		}
		if ok {
			parent := bar
			if parent.CloseTime.IsZero() || !parent.CloseTime.After(parent.OpenTime) {
				if _, end := ParentBarWindow(bar, iv); end.After(bar.OpenTime) {
					parent.CloseTime = end
				}
			}
			sub := OHLCInside(finer(parent, next), parent, iv)
			var last float64
			for _, sb := range sub {
				if sb.OpenTime.UTC().After(at) {
					break
				}
				if p := priceAtOrBefore(sb, at, next, finer, depth+1); p > 0 {
					last = p
				}
			}
			if last > 0 {
				return last
			}
		}
	}
	if bar.Open > 0 {
		return bar.Open
	}
	return 0
}

func clampBacktestLeverage(levels SwingLevels) int {
	lev := levels.Leverage
	if lev < 1 {
		lev = SuggestedSwingLeverage(levels.RiskPct)
	}
	if lev < 1 {
		lev = 1
	}
	if lev > BacktestMaxLeverage {
		lev = BacktestMaxLeverage
	}
	return lev
}

func throughAtOpenPrice(side string, open, sl, tp float64) string {
	if open <= 0 || sl <= 0 || tp <= 0 {
		return ""
	}
	if side == BacktestTradeSideShort {
		if open >= sl {
			return BacktestTradeOutcomeSL
		}
		if open <= tp {
			return BacktestTradeOutcomeTP
		}
		return ""
	}
	if open <= sl {
		return BacktestTradeOutcomeSL
	}
	if open >= tp {
		return BacktestTradeOutcomeTP
	}
	return ""
}

func validTradeLevels(side string, entry, sl, tp float64) bool {
	if entry <= 0 || sl <= 0 || tp <= 0 {
		return false
	}
	if side == BacktestTradeSideShort {
		return tp < entry && sl > entry
	}
	return sl < entry && tp > entry
}

// firstTouchOnBar returns tp|sl and the fill price. Empty hit means neither level traded.
// A gap through the stop at the open fills at the open. A gap through TP fills at TP.
// Both wicks with no finer bars count as SL.
func firstTouchOnBar(bar OHLC, side string, sl, tp float64) (hit string, price float64) {
	hit, px, both := inspectBarTouch(bar, side, sl, tp)
	if both {
		return BacktestTradeOutcomeSL, sl
	}
	return hit, px
}

// inspectBarTouch reports a clear hit, or both=true when high and low tag both levels
// and the open did not already gap through one of them. A stop gap fills at the open.
func inspectBarTouch(bar OHLC, side string, sl, tp float64) (hit string, price float64, both bool) {
	open := bar.Open
	if side == BacktestTradeSideShort {
		if open > 0 && open >= sl {
			return BacktestTradeOutcomeSL, open, false
		}
		if open > 0 && open <= tp {
			return BacktestTradeOutcomeTP, tp, false
		}
		hitSL := bar.High >= sl
		hitTP := bar.Low <= tp
		if hitSL && hitTP {
			return "", 0, true
		}
		if hitSL {
			return BacktestTradeOutcomeSL, sl, false
		}
		if hitTP {
			return BacktestTradeOutcomeTP, tp, false
		}
		return "", 0, false
	}
	if open > 0 && open <= sl {
		return BacktestTradeOutcomeSL, open, false
	}
	if open > 0 && open >= tp {
		return BacktestTradeOutcomeTP, tp, false
	}
	hitSL := bar.Low <= sl
	hitTP := bar.High >= tp
	if hitSL && hitTP {
		return "", 0, true
	}
	if hitSL {
		return BacktestTradeOutcomeSL, sl, false
	}
	if hitTP {
		return BacktestTradeOutcomeTP, tp, false
	}
	return "", 0, false
}

// FinerBacktestInterval is the next smaller bar used when a parent bar tags both levels.
// Steps stay on intervals every venue publishes (1d / 1h / 15m / 5m / 1m).
func FinerBacktestInterval(iv CandleInterval) (CandleInterval, bool) {
	d := IntervalDuration(iv)
	if d <= 0 {
		return "", false
	}
	return FinerBacktestIntervalForDuration(d)
}

// FinerBacktestIntervalForDuration picks the next smaller step from a bar's length.
func FinerBacktestIntervalForDuration(d time.Duration) (CandleInterval, bool) {
	if d <= time.Minute {
		return "", false
	}
	var next CandleInterval
	switch {
	case d > 24*time.Hour:
		next = Interval1d
	case d > time.Hour:
		next = Interval1h
	case d > 15*time.Minute:
		next = Interval15m
	case d > 5*time.Minute:
		next = Interval5m
	default:
		next = Interval1m
	}
	if step := IntervalDuration(next); step <= 0 || step >= d {
		return "", false
	}
	return next, true
}

// ParentBarWindow is [open, close) for a candle. Close is inferred from iv when missing.
func ParentBarWindow(bar OHLC, iv CandleInterval) (start, end time.Time) {
	start = bar.OpenTime.UTC()
	end = bar.CloseTime.UTC()
	if end.After(start) {
		return start, end
	}
	if d := IntervalDuration(iv); d > 0 && !start.IsZero() {
		return start, start.Add(d)
	}
	return start, end
}

// OHLCInside keeps finer bars whose open is in the parent window.
func OHLCInside(bars []OHLC, parent OHLC, parentIV CandleInterval) []OHLC {
	start, end := ParentBarWindow(parent, parentIV)
	if start.IsZero() || !end.After(start) {
		return nil
	}
	out := make([]OHLC, 0, len(bars))
	for _, b := range bars {
		ot := b.OpenTime.UTC()
		if ot.Before(start) || !ot.Before(end) {
			continue
		}
		out = append(out, b)
	}
	return out
}

type firstTouch struct {
	hit     string
	price   float64
	at      time.Time // set only when the open gapped through the level
	knownBy time.Time // bar close: earliest we know the trade is done
}

func resolveFirstTouch(bar OHLC, side string, sl, tp float64, iv CandleInterval, finer FineBarLookup, depth int) firstTouch {
	hit, px, both := inspectBarTouch(bar, side, sl, tp)
	if hit == "" && !both {
		return firstTouch{}
	}
	if finer != nil && depth < backtestFinerMaxDepth {
		next, ok := FinerBacktestInterval(iv)
		if !ok {
			if span := barSpan(bar, iv); span > 0 {
				next, ok = FinerBacktestIntervalForDuration(span)
			}
		}
		if ok {
			parent := bar
			if parent.CloseTime.IsZero() || !parent.CloseTime.After(parent.OpenTime) {
				if _, end := ParentBarWindow(bar, iv); end.After(bar.OpenTime) {
					parent.CloseTime = end
				}
			}
			sub := OHLCInside(finer(parent, next), parent, iv)
			start, _ := ParentBarWindow(parent, iv)
			canWalk := finerCoversParent(sub, parent, iv, next)
			if !canWalk && len(sub) > 0 && !start.IsZero() && sub[0].OpenTime.UTC().Equal(start) {
				canWalk = true
			}
			if canWalk {
				for _, sb := range sub {
					got := resolveFirstTouch(sb, side, sl, tp, next, finer, depth+1)
					if got.hit != "" {
						return got
					}
				}
			}
		}
	}
	if both {
		hit, px = BacktestTradeOutcomeSL, sl
	}
	if hit == "" {
		return firstTouch{}
	}
	_, end := ParentBarWindow(bar, iv)
	if end.IsZero() || !end.After(bar.OpenTime) {
		if d := IntervalDuration(iv); d > 0 && !bar.OpenTime.IsZero() {
			end = bar.OpenTime.UTC().Add(d)
		}
	}
	out := firstTouch{hit: hit, price: px, knownBy: end}
	// Only a gap through the level at the open is an exact print time.
	if !both && openGapsLevel(bar, side, sl, tp) {
		out.at = bar.OpenTime.UTC()
	}
	return out
}

func openGapsLevel(bar OHLC, side string, sl, tp float64) bool {
	open := bar.Open
	if open <= 0 {
		return false
	}
	if side == BacktestTradeSideShort {
		return open >= sl || open <= tp
	}
	return open <= sl || open >= tp
}

// TradeRiskFreeAt is when risk can be released. Unknown hit times wait until the parent bar close.
func TradeRiskFreeAt(tr *BacktestTrade) *time.Time {
	if tr == nil {
		return nil
	}
	if tr.HitTimeUnknown && tr.ClosedKnownAt != nil && !tr.ClosedKnownAt.IsZero() {
		t := tr.ClosedKnownAt.UTC()
		return &t
	}
	if tr.HitAt != nil && !tr.HitAt.IsZero() {
		t := tr.HitAt.UTC()
		return &t
	}
	if tr.ClosedKnownAt != nil && !tr.ClosedKnownAt.IsZero() {
		t := tr.ClosedKnownAt.UTC()
		return &t
	}
	return nil
}

// TradeOpenAt is true when tr has not released risk before at.
func TradeOpenAt(tr *BacktestTrade, at time.Time) bool {
	if tr == nil {
		return false
	}
	free := TradeRiskFreeAt(tr)
	if free == nil {
		return true
	}
	if at.IsZero() {
		return true
	}
	return free.After(at.UTC())
}

func appendBacktestMark(marks []BacktestPriceMark, at time.Time, px float64) []BacktestPriceMark {
	if at.IsZero() || px <= 0 || math.IsNaN(px) {
		return marks
	}
	at = at.UTC()
	if n := len(marks); n > 0 && !marks[n-1].At.Before(at) {
		marks[n-1] = BacktestPriceMark{At: at, Price: px}
		return marks
	}
	return append(marks, BacktestPriceMark{At: at, Price: px})
}

func barMarkAt(bar OHLC, iv CandleInterval) time.Time {
	if !bar.CloseTime.IsZero() && bar.CloseTime.After(bar.OpenTime) {
		return bar.CloseTime.UTC()
	}
	if _, end := ParentBarWindow(bar, iv); end.After(bar.OpenTime) {
		return end
	}
	return bar.OpenTime.UTC()
}

func markPriceAt(marks []BacktestPriceMark, at time.Time, fallback float64) float64 {
	px := fallback
	if at.IsZero() {
		if n := len(marks); n > 0 && marks[n-1].Price > 0 {
			return marks[n-1].Price
		}
		return px
	}
	at = at.UTC()
	for _, m := range marks {
		if m.At.After(at) {
			break
		}
		if m.Price > 0 {
			px = m.Price
		}
	}
	return px
}

func barSpan(bar OHLC, iv CandleInterval) time.Duration {
	if !bar.CloseTime.IsZero() && bar.CloseTime.After(bar.OpenTime) {
		return bar.CloseTime.Sub(bar.OpenTime)
	}
	return IntervalDuration(iv)
}

func finerCoversParent(sub []OHLC, parent OHLC, parentIV, finerIV CandleInterval) bool {
	start, end := ParentBarWindow(parent, parentIV)
	step := IntervalDuration(finerIV)
	if step <= 0 || !end.After(start) || len(sub) == 0 {
		return false
	}
	have := make(map[int64]struct{}, len(sub))
	for _, b := range sub {
		if !b.OpenTime.IsZero() {
			have[b.OpenTime.UTC().Unix()] = struct{}{}
		}
	}
	for t := start; t.Before(end); t = t.Add(step) {
		if _, ok := have[t.Unix()]; !ok {
			return false
		}
	}
	return true
}

// LeveragedPnLPct is (price move / entry) * leverage * 100, signed for the side.
func LeveragedPnLPct(side string, entry, exit float64, leverage int) float64 {
	if entry <= 0 || math.IsNaN(entry) || math.IsNaN(exit) {
		return 0
	}
	if leverage < 1 {
		leverage = 1
	}
	raw := (exit - entry) / entry
	if NormalizeTradeSide(side) == BacktestTradeSideShort {
		raw = (entry - exit) / entry
	}
	return round1(raw * float64(leverage) * 100)
}

func summarizeBacktestTrades(sigs []ScannerBacktestSignal) BacktestTradeStats {
	var out BacktestTradeStats
	closed := make([]float64, 0, len(sigs))
	for _, s := range sigs {
		if s.Trade == nil {
			continue
		}
		out.SampleCount++
		if s.Trade.Side == BacktestTradeSideShort {
			out.ShortCount++
		} else {
			out.LongCount++
		}
		switch s.Trade.Outcome {
		case BacktestTradeOutcomeTP, BacktestTradeOutcomeTP1, BacktestTradeOutcomeTP1TP2:
			out.TpHits++
			closed = append(closed, s.Trade.PnLPct)
		case BacktestTradeOutcomeSL, BacktestTradeOutcomeTP1SL:
			out.SlHits++
			closed = append(closed, s.Trade.PnLPct)
		default:
			out.OpenCount++
		}
	}
	nClosed := out.TpHits + out.SlHits
	if nClosed > 0 {
		wr := round1(float64(out.TpHits) / float64(nClosed) * 100)
		out.WinRate = &wr
		sum := 0.0
		for _, v := range closed {
			sum += v
		}
		avg := round1(sum / float64(len(closed)))
		med := round1(medianFloats(closed))
		out.Average = &avg
		out.Median = &med
	}
	return out
}

// PlanBacktestTrade builds ATR/structure levels for the bar and simulates first-touch.
func PlanBacktestTrade(bars []OHLC, signalIdx int, side string) *BacktestTrade {
	return PlanBacktestTradeOpts(bars, signalIdx, side, BacktestTouchOpts{})
}

// PlanBacktestTradeOpts is PlanBacktestTrade with optional finer first-touch bars.
func PlanBacktestTradeOpts(bars []OHLC, signalIdx int, side string, opts BacktestTouchOpts) *BacktestTrade {
	if signalIdx < 0 || signalIdx >= len(bars) {
		return nil
	}
	side = NormalizeTradeSide(side)
	entry := bars[signalIdx].Close
	if entry <= 0 {
		return nil
	}
	atr, _ := ATR(bars[:signalIdx+1], SwingATRPeriod)
	levels, _ := planSwingLevelsForSide(bars[:signalIdx+1], atr, entry, side)
	if levels == nil {
		return nil
	}
	return SimulateBacktestTradeOpts(bars, signalIdx, side, *levels, opts)
}
