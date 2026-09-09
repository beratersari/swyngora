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
)

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
		hit, px := firstTouchOnBar(bar, side, sl, tp)
		if hit == "" {
			continue
		}
		out.Outcome = hit
		out.ExitPrice = round8(px)
		out.BarsHeld = j - signalIdx
		if !bar.OpenTime.IsZero() {
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
// A gap through a level at the open takes that level. Both wicks in one bar count as SL (conservative).
func firstTouchOnBar(bar OHLC, side string, sl, tp float64) (hit string, price float64) {
	open := bar.Open
	if open <= 0 {
		open = bar.Close
	}
	if side == BacktestTradeSideShort {
		if open > 0 && open >= sl {
			return BacktestTradeOutcomeSL, sl
		}
		if open > 0 && open <= tp {
			return BacktestTradeOutcomeTP, tp
		}
		hitSL := bar.High >= sl
		hitTP := bar.Low <= tp
		if hitSL && hitTP {
			return BacktestTradeOutcomeSL, sl
		}
		if hitSL {
			return BacktestTradeOutcomeSL, sl
		}
		if hitTP {
			return BacktestTradeOutcomeTP, tp
		}
		return "", 0
	}
	if open > 0 && open <= sl {
		return BacktestTradeOutcomeSL, sl
	}
	if open > 0 && open >= tp {
		return BacktestTradeOutcomeTP, tp
	}
	hitSL := bar.Low <= sl
	hitTP := bar.High >= tp
	if hitSL && hitTP {
		return BacktestTradeOutcomeSL, sl
	}
	if hitSL {
		return BacktestTradeOutcomeSL, sl
	}
	if hitTP {
		return BacktestTradeOutcomeTP, tp
	}
	return "", 0
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
	return SimulateBacktestTrade(bars, signalIdx, side, *levels)
}
