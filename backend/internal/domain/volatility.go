package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	VolWindow1h  = LiquidationWindow1h
	VolWindow4h  = LiquidationWindow4h
	VolWindow24h = LiquidationWindow24h

	VolVsHigher  = "higher"
	VolVsTypical = "typical"
	VolVsLower   = "lower"

	VolTrendExpanding = "expanding"
	VolTrendShrinking = "shrinking"
	VolTrendStable    = "stable"

	VolVsMore    = "more_volatile"
	VolVsSimilar = "similar"
	VolVsCalmer  = "calmer"

	volHigherRatio = 1.25
	volLowerRatio  = 0.80
	volExpandRatio = 1.15
	volShrinkRatio = 0.87
	volMoreRatio   = 1.30
	volCalmerRatio = 0.75
	volMinPrior    = 3
	volMinBars     = 8
)

// OHLCBar is one candle used for range and realized-vol math.
type OHLCBar struct {
	Time  time.Time
	Open  float64
	High  float64
	Low   float64
	Close float64
}

// VolMeasure is how much price moved in one slice of bars.
type VolMeasure struct {
	NetPct      float64 // last/first − 1 as percent
	RangePct    float64 // (high−low)/first close as percent
	RealizedPct float64 // stdev of bar returns × √n (path noise in % terms)
	High        float64
	Low         float64
	Bars        int
	Complete    bool
}

// VolWindow is one lookback for the coin vs its own history and vs BTC/ETH.
type VolWindow struct {
	Window     string
	Interval   string
	Coin       VolMeasure
	Previous   VolMeasure
	TypicalPct float64 // median prior range
	VsNormal   string  // higher | typical | lower
	Trend      string  // expanding | shrinking | stable
	BTC        VolMeasure
	ETH        VolMeasure
	VsBTC      string // more_volatile | similar | calmer
	VsETH      string
	VsMarket   string
	Summary    string
}

// VolatilityReport is the API result for one coin.
type VolatilityReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Windows  []VolWindow
	Summary  string
	Note     string
}

// VolWindowSpec is how we sample one lookback.
type VolWindowSpec struct {
	ID       string
	Interval CandleInterval
	Bars     int
}

// VolatilityWindows is 1m for 1h, 5m for 4h and 24h.
var VolatilityWindows = []VolWindowSpec{
	{VolWindow1h, Interval1m, 60},
	{VolWindow4h, Interval5m, 48},
	{VolWindow24h, Interval5m, 288},
}

// BarsFromCandles extracts valid OHLC, oldest first.
func BarsFromCandles(candles []Candle) []OHLCBar {
	out := make([]OHLCBar, 0, len(candles))
	for _, c := range candles {
		cl, err1 := parseClose(c.Close)
		hi, err2 := parseClose(c.High)
		lo, err3 := parseClose(c.Low)
		op, err4 := parseClose(c.Open)
		if err1 != nil || cl <= 0 || math.IsNaN(cl) || math.IsInf(cl, 0) {
			continue
		}
		if err2 != nil || hi <= 0 {
			hi = cl
		}
		if err3 != nil || lo <= 0 {
			lo = cl
		}
		if err4 != nil || op <= 0 {
			op = cl
		}
		if hi < lo {
			hi, lo = lo, hi
		}
		t := c.OpenTime.UTC()
		if t.IsZero() {
			continue
		}
		out = append(out, OHLCBar{Time: t, Open: op, High: hi, Low: lo, Close: cl})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// MeasureVolatility is net move, high-low range, and realized path noise.
func MeasureVolatility(bars []OHLCBar) VolMeasure {
	out := VolMeasure{Bars: len(bars)}
	if len(bars) < 2 {
		return out
	}
	ref := bars[0].Close
	if ref <= 0 {
		ref = bars[0].Open
	}
	if ref <= 0 {
		return out
	}
	hi, lo := bars[0].High, bars[0].Low
	closes := make([]float64, 0, len(bars))
	for _, b := range bars {
		if b.High > hi {
			hi = b.High
		}
		if b.Low > 0 && b.Low < lo {
			lo = b.Low
		}
		if b.Close > 0 {
			closes = append(closes, b.Close)
		}
	}
	out.High, out.Low = hi, lo
	out.NetPct = (bars[len(bars)-1].Close/ref - 1) * 100
	if hi > 0 && lo > 0 {
		out.RangePct = (hi - lo) / ref * 100
	}
	rets := PercentReturns(closes)
	if s := stdev(rets); s > 0 && !math.IsNaN(s) {
		out.RealizedPct = s * math.Sqrt(float64(len(rets)))
	}
	out.Complete = true
	return out
}

func stdev(xs []float64) float64 {
	n := len(xs)
	if n < 2 {
		return 0
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	m := sum / float64(n)
	var ss float64
	for _, v := range xs {
		d := v - m
		ss += d * d
	}
	return math.Sqrt(ss / float64(n-1))
}

// SplitVolWindows walks newest-first into current, previous, and earlier slices.
func SplitVolWindows(bars []OHLCBar, n int) (current, previous []OHLCBar, priors []VolMeasure) {
	if n < volMinBars || len(bars) < n {
		return bars, nil, nil
	}
	current = bars[len(bars)-n:]
	rest := bars[:len(bars)-n]
	if len(rest) >= n {
		previous = rest[len(rest)-n:]
		rest = rest[:len(rest)-n]
	}
	for len(rest) >= n {
		chunk := rest[len(rest)-n:]
		rest = rest[:len(rest)-n]
		m := MeasureVolatility(chunk)
		if m.Complete {
			priors = append(priors, m)
		}
	}
	return current, previous, priors
}

// ClassifyVolVsNormal compares current range to the median of prior windows.
func ClassifyVolVsNormal(current, typical float64, priorN int) string {
	if priorN < volMinPrior || typical <= 0 || current <= 0 {
		return VolVsTypical
	}
	r := current / typical
	switch {
	case r >= volHigherRatio:
		return VolVsHigher
	case r <= volLowerRatio:
		return VolVsLower
	default:
		return VolVsTypical
	}
}

// ClassifyVolTrend compares current range to the immediately previous window.
func ClassifyVolTrend(current, previous float64) string {
	if current <= 0 || previous <= 0 {
		return VolTrendStable
	}
	r := current / previous
	switch {
	case r >= volExpandRatio:
		return VolTrendExpanding
	case r <= volShrinkRatio:
		return VolTrendShrinking
	default:
		return VolTrendStable
	}
}

// ClassifyVsMarket compares coin range to a reference range.
func ClassifyVsMarket(coin, ref float64) string {
	if coin <= 0 || ref <= 0 {
		return VolVsSimilar
	}
	r := coin / ref
	switch {
	case r >= volMoreRatio:
		return VolVsMore
	case r <= volCalmerRatio:
		return VolVsCalmer
	default:
		return VolVsSimilar
	}
}

func marketRange(btc, eth VolMeasure, coinIsBTC, coinIsETH bool) (float64, bool) {
	switch {
	case coinIsBTC && eth.Complete:
		return eth.RangePct, true
	case coinIsETH && btc.Complete:
		return btc.RangePct, true
	case btc.Complete && eth.Complete:
		return (btc.RangePct + eth.RangePct) / 2, true
	case btc.Complete:
		return btc.RangePct, true
	case eth.Complete:
		return eth.RangePct, true
	default:
		return 0, false
	}
}

// BuildVolWindow scores one lookback for the coin vs history and vs BTC/ETH.
func BuildVolWindow(symbol string, spec VolWindowSpec, asset, btc, eth []OHLCBar) VolWindow {
	w := VolWindow{Window: spec.ID, Interval: string(spec.Interval), Trend: VolTrendStable, VsNormal: VolVsTypical, VsBTC: VolVsSimilar, VsETH: VolVsSimilar, VsMarket: VolVsSimilar}
	cur, prev, priors := SplitVolWindows(asset, spec.Bars)
	w.Coin = MeasureVolatility(cur)
	w.Previous = MeasureVolatility(prev)
	if len(priors) > 0 {
		rs := make([]float64, 0, len(priors))
		for _, p := range priors {
			if p.RangePct > 0 {
				rs = append(rs, p.RangePct)
			}
		}
		w.TypicalPct = medianFloat(rs)
	}
	if w.Coin.Complete {
		w.VsNormal = ClassifyVolVsNormal(w.Coin.RangePct, w.TypicalPct, len(priors))
		if w.Previous.Complete {
			w.Trend = ClassifyVolTrend(w.Coin.RangePct, w.Previous.RangePct)
		}
	}
	btcCur, _, _ := SplitVolWindows(btc, spec.Bars)
	ethCur, _, _ := SplitVolWindows(eth, spec.Bars)
	w.BTC = MeasureVolatility(btcCur)
	w.ETH = MeasureVolatility(ethCur)
	base := prettyBase(symbol)
	coinIsBTC := strings.EqualFold(base, "BTC")
	coinIsETH := strings.EqualFold(base, "ETH")
	if w.Coin.Complete && w.BTC.Complete && !coinIsBTC {
		w.VsBTC = ClassifyVsMarket(w.Coin.RangePct, w.BTC.RangePct)
	}
	if w.Coin.Complete && w.ETH.Complete && !coinIsETH {
		w.VsETH = ClassifyVsMarket(w.Coin.RangePct, w.ETH.RangePct)
	}
	if w.Coin.Complete {
		if mkt, ok := marketRange(w.BTC, w.ETH, coinIsBTC, coinIsETH); ok {
			w.VsMarket = ClassifyVsMarket(w.Coin.RangePct, mkt)
		}
	}
	w.Summary = ExplainVolWindow(symbol, w)
	return w
}

// ExplainVolWindow is a short read for one lookback.
func ExplainVolWindow(symbol string, w VolWindow) string {
	name := prettyBase(symbol)
	if !w.Coin.Complete {
		return fmt.Sprintf("%s: not enough %s bars to measure volatility.", name, w.Window)
	}
	s := fmt.Sprintf("Over %s, %s moved %s%% net with a %s%% high–low range",
		w.Window, name, FormatSignedPct(w.Coin.NetPct), formatFixed(w.Coin.RangePct, 2))
	if w.BTC.Complete && !strings.EqualFold(name, "BTC") {
		s += fmt.Sprintf(" (BTC range %s%%)", formatFixed(w.BTC.RangePct, 2))
	}
	s += "."
	switch w.Trend {
	case VolTrendExpanding:
		s += " The range is getting bigger."
	case VolTrendShrinking:
		s += " The range is getting smaller."
	}
	switch w.VsNormal {
	case VolVsHigher:
		s += " This is higher than normal for this coin."
	case VolVsLower:
		s += " This is quieter than normal for this coin."
	}
	switch w.VsMarket {
	case VolVsMore:
		s += " " + vsMarketClause(name, VolVsMore)
	case VolVsCalmer:
		s += " " + vsMarketClause(name, VolVsCalmer)
	case VolVsSimilar:
		if w.BTC.Complete || w.ETH.Complete {
			s += " About as jumpy as the large coins."
		}
	}
	return s
}

// ExplainVolatilityReport rolls 1h / 4h / 24h into one paragraph.
func ExplainVolatilityReport(symbol string, windows []VolWindow) string {
	name := prettyBase(symbol)
	var h1, h4, h24 *VolWindow
	for i := range windows {
		w := &windows[i]
		switch w.Window {
		case VolWindow1h:
			h1 = w
		case VolWindow4h:
			h4 = w
		case VolWindow24h:
			h24 = w
		}
	}
	parts := make([]string, 0, 3)
	add := func(w *VolWindow) {
		if w == nil || !w.Coin.Complete {
			return
		}
		parts = append(parts, fmt.Sprintf("%s range %s%%", w.Window, formatFixed(w.Coin.RangePct, 2)))
	}
	add(h1)
	add(h4)
	add(h24)
	if len(parts) == 0 {
		return name + ": not enough price history to measure volatility."
	}
	head := name + " " + joinList(parts) + "."
	src := h1
	if src == nil || !src.Coin.Complete {
		src = h4
	}
	if src == nil || !src.Coin.Complete {
		src = h24
	}
	if src == nil {
		return head
	}
	switch src.VsMarket {
	case VolVsMore, VolVsCalmer:
		return head + " " + vsMarketClause(name, src.VsMarket)
	default:
		if src.Trend == VolTrendExpanding {
			return head + " The range has been expanding recently."
		}
		if src.Trend == VolTrendShrinking {
			return head + " The range has been shrinking recently."
		}
		return head
	}
}

func vsMarketClause(name, vs string) string {
	ref := "BTC and ETH"
	if strings.EqualFold(name, "BTC") {
		ref = "ETH"
	} else if strings.EqualFold(name, "ETH") {
		ref = "BTC"
	}
	switch vs {
	case VolVsMore:
		return "More volatile than " + ref + "."
	case VolVsCalmer:
		return "Calmer than " + ref + "."
	default:
		return ""
	}
}
