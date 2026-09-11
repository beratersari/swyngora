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
}

// BacktestTrade is the signal's plan plus first-touch result.
type BacktestTrade struct {
	Side       string     `json:"side"`
	Leverage   int        `json:"leverage"`
	Entry      float64    `json:"entry"`
	StopLoss   float64    `json:"stopLoss"`
	TakeProfit float64    `json:"takeProfit"`
	Outcome    string     `json:"outcome"` // tp | sl | open
	ExitPrice  float64    `json:"exitPrice,omitempty"`
	PnLPct     float64    `json:"pnlPct"` // leveraged percent
	BarsHeld   int        `json:"barsHeld,omitempty"`
	HitAt      *time.Time `json:"hitAt,omitempty"`
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

// OHLCFromCandles keeps index alignment. Missing open/high/low fall back to close.
func OHLCFromCandles(candles []Candle) []OHLC {
	out := make([]OHLC, len(candles))
	for i, c := range candles {
		cl, err := parseClose(c.Close)
		if err != nil || cl <= 0 || math.IsNaN(cl) {
			continue
		}
		o, oerr := parseClose(c.Open)
		if oerr != nil || o <= 0 || math.IsNaN(o) {
			o = cl
		}
		h, herr := parseClose(c.High)
		if herr != nil || h <= 0 || math.IsNaN(h) {
			h = math.Max(o, cl)
		}
		l, lerr := parseClose(c.Low)
		if lerr != nil || l <= 0 || math.IsNaN(l) {
			l = math.Min(o, cl)
		}
		if h < math.Max(o, cl) {
			h = math.Max(o, cl)
		}
		if l > math.Min(o, cl) {
			l = math.Min(o, cl)
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
	sl, tp := levels.StopLoss, levels.TakeProfit
	if !validTradeLevels(side, entry, sl, tp) {
		return nil
	}
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
	out := &BacktestTrade{
		Side: side, Leverage: lev, Entry: round8(entry),
		StopLoss: round8(sl), TakeProfit: round8(tp),
		Outcome: BacktestTradeOutcomeOpen,
	}
	for j := signalIdx + 1; j < len(bars); j++ {
		bar := bars[j]
		if bar.Close <= 0 && bar.High <= 0 {
			continue
		}
		hit, px, at := resolveFirstTouch(bar, side, sl, tp, opts.Interval, opts.Finer, 0)
		if hit == "" {
			continue
		}
		out.Outcome = hit
		out.ExitPrice = round8(px)
		out.BarsHeld = j - signalIdx
		if !at.IsZero() {
			t := at.UTC()
			out.HitAt = &t
		} else if !bar.OpenTime.IsZero() {
			t := bar.OpenTime.UTC()
			out.HitAt = &t
		}
		out.PnLPct = LeveragedPnLPct(side, entry, px, lev)
		return out
	}
	last := bars[len(bars)-1]
	exit := last.Close
	if exit <= 0 {
		exit = entry
	}
	out.ExitPrice = round8(exit)
	out.BarsHeld = len(bars) - 1 - signalIdx
	if out.BarsHeld < 0 {
		out.BarsHeld = 0
	}
	out.PnLPct = LeveragedPnLPct(side, entry, exit, lev)
	return out
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
	if open <= 0 {
		open = bar.Close
	}
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

func resolveFirstTouch(bar OHLC, side string, sl, tp float64, iv CandleInterval, finer FineBarLookup, depth int) (hit string, price float64, at time.Time) {
	hit, px, both := inspectBarTouch(bar, side, sl, tp)
	if !both {
		return hit, px, bar.OpenTime
	}
	if finer == nil || depth >= backtestFinerMaxDepth {
		return BacktestTradeOutcomeSL, sl, bar.OpenTime
	}
	next, ok := FinerBacktestInterval(iv)
	if !ok {
		if span := barSpan(bar, iv); span > 0 {
			next, ok = FinerBacktestIntervalForDuration(span)
		}
	}
	if !ok {
		return BacktestTradeOutcomeSL, sl, bar.OpenTime
	}
	parent := bar
	if parent.CloseTime.IsZero() || !parent.CloseTime.After(parent.OpenTime) {
		if _, end := ParentBarWindow(bar, iv); end.After(bar.OpenTime) {
			parent.CloseTime = end
		}
	}
	sub := OHLCInside(finer(parent, next), parent, iv)
	if !finerCoversParent(sub, parent, iv, next) {
		return BacktestTradeOutcomeSL, sl, bar.OpenTime
	}
	for _, sb := range sub {
		h, p, t := resolveFirstTouch(sb, side, sl, tp, next, finer, depth+1)
		if h != "" {
			return h, p, t
		}
	}
	return BacktestTradeOutcomeSL, sl, bar.OpenTime
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
		case BacktestTradeOutcomeTP:
			out.TpHits++
			closed = append(closed, s.Trade.PnLPct)
		case BacktestTradeOutcomeSL:
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
