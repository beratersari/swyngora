package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	LiquiditySweepSideHigh = "high" // poke above a prior high, then back under
	LiquiditySweepSideLow  = "low"  // poke below a prior low, then back over

	LiquiditySweepSwept = "swept"
	LiquiditySweepOpen  = "open"

	sweepSwingK          = 2
	sweepClusterPct      = 0.0015 // 0.15% — same “around 65k” shelf
	sweepMinTests        = 2
	sweepMinExcursionPct = 0.02
	sweepMaxBars         = 8 // 8×15m = 2h
	MaxLiquiditySweeps   = 20
	SweepCandleLimit     = 700
)

// SweepBar is one candle used to find levels and measure a sweep.
type SweepBar struct {
	Time         time.Time
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       float64
	BuyVolume    float64
	SellVolume   float64
	BuySellKnown bool
}

// LiquiditySweep is one poke through a prior high or low that came back
// (or is still beyond the level).
type LiquiditySweep struct {
	Side            string
	Level           float64
	LevelTime       time.Time
	Tests           int
	PiercedAt       time.Time
	ReclaimedAt     time.Time
	Extreme         float64
	Excursion       float64
	ExcursionPct    float64
	Duration        string
	DurationSeconds int
	Volume          float64
	BuyVolume       float64
	SellVolume      float64
	BuySellKnown    bool
	Bars            int
	Status          string
	Interval        string
	Title           string
	Summary         string
}

// LiquiditySweepVenue is one venue's recent sweeps.
type LiquiditySweepVenue struct {
	Exchange  Exchange
	Symbol    string
	Interval  string
	LastPrice float64
	Sweeps    []LiquiditySweep
	Current   *LiquiditySweep
	Summary   string
	Error     string
}

// LiquiditySweepReport is the API result.
type LiquiditySweepReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Venues   []LiquiditySweepVenue
	Summary  string
	Note     string
}

// SweepBarsFromCandles maps klines to sweep bars (quote volume + taker buy when present).
func SweepBarsFromCandles(candles []Candle) []SweepBar {
	out := make([]SweepBar, 0, len(candles))
	for _, c := range candles {
		hi, err1 := parseFloat(c.High)
		lo, err2 := parseFloat(c.Low)
		cl, err3 := parseFloat(c.Close)
		if err1 != nil || err2 != nil || err3 != nil || hi <= 0 || lo <= 0 || cl <= 0 {
			continue
		}
		if lo > hi {
			lo, hi = hi, lo
		}
		bar := SweepBar{Time: c.OpenTime.UTC(), High: hi, Low: lo, Close: cl, Open: cl}
		if op, err := parseFloat(c.Open); err == nil && op > 0 {
			bar.Open = op
		}
		if q, err := parseFloat(c.QuoteVolume); err == nil && q > 0 {
			bar.Volume = q
		}
		if c.TakerBuyQuote != "" {
			if buy, err := parseFloat(c.TakerBuyQuote); err == nil && buy >= 0 {
				if bar.Volume > 0 && buy > bar.Volume {
					buy = bar.Volume
				}
				bar.BuyVolume = buy
				if bar.Volume > 0 {
					bar.SellVolume = bar.Volume - buy
				}
				bar.BuySellKnown = true
			}
		}
		out = append(out, bar)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// DetectLiquiditySweeps finds high/low sweeps on chronological bars.
func DetectLiquiditySweeps(bars []SweepBar, interval time.Duration) []LiquiditySweep {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	if len(bars) < sweepSwingK*2+4 {
		return nil
	}
	ohlc := sweepOHLC(bars)
	highs := clusterSweepSwings(ohlc, true)
	lows := clusterSweepSwings(ohlc, false)
	var out []LiquiditySweep
	out = append(out, scanSweeps(bars, highs, true, interval)...)
	out = append(out, scanSweeps(bars, lows, false, interval)...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].PiercedAt.After(out[j].PiercedAt)
	})
	out = dedupeSweeps(out)
	if len(out) > MaxLiquiditySweeps {
		out = out[:MaxLiquiditySweeps]
	}
	return out
}

// BuildLiquiditySweepVenue writes summaries for one venue.
func BuildLiquiditySweepVenue(ex Exchange, symbol string, bars []SweepBar, last float64, interval time.Duration) LiquiditySweepVenue {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	out := LiquiditySweepVenue{
		Exchange: ex, Symbol: symbol, Interval: intervalLabel(interval),
		LastPrice: last, Sweeps: []LiquiditySweep{},
	}
	if last <= 0 && len(bars) > 0 {
		out.LastPrice = bars[len(bars)-1].Close
	}
	if len(bars) == 0 {
		out.Error = "not enough candles in this range"
		out.Summary = out.Error
		return out
	}
	out.Sweeps = DetectLiquiditySweeps(bars, interval)
	for i := range out.Sweeps {
		if out.Sweeps[i].Status == LiquiditySweepOpen {
			s := out.Sweeps[i]
			out.Current = &s
			break
		}
	}
	if out.Current == nil && len(out.Sweeps) > 0 {
		s := out.Sweeps[0]
		out.Current = &s
	}
	out.Summary = ExplainLiquiditySweepVenue(out)
	return out
}

// ExplainLiquiditySweepVenue writes a latest-sweep-first line.
func ExplainLiquiditySweepVenue(v LiquiditySweepVenue) string {
	if v.Error != "" {
		return v.Error
	}
	name := prettyBase(v.Symbol)
	if len(v.Sweeps) == 0 {
		return name + ": no liquidity sweep in view (price did not poke a prior high/low and come back)."
	}
	nHi, nLo := 0, 0
	for _, s := range v.Sweeps {
		if s.Side == LiquiditySweepSideHigh {
			nHi++
		} else {
			nLo++
		}
	}
	head := ""
	if v.Current != nil && v.Current.Summary != "" {
		head = v.Current.Summary
	} else {
		head = v.Sweeps[0].Summary
	}
	return fmt.Sprintf("%s: %s %d high / %d low sweep(s) in view.", name, head, nHi, nLo)
}

// ExplainLiquiditySweepReport prefers the first venue with a sweep.
func ExplainLiquiditySweepReport(r LiquiditySweepReport) string {
	for _, v := range r.Venues {
		if v.Error == "" && v.Summary != "" && len(v.Sweeps) > 0 {
			if len(r.Venues) == 1 {
				return v.Summary
			}
			return string(v.Exchange) + ": " + v.Summary
		}
	}
	for _, v := range r.Venues {
		if v.Summary != "" {
			return v.Summary
		}
	}
	return "No liquidity sweep in view."
}

type sweepTouch struct {
	idx   int
	price float64
	time  time.Time
}

type sweepLevel struct {
	price   float64
	highs   bool
	touches []sweepTouch
}

func clusterSweepSwings(bars []OHLCBar, highs bool) []sweepLevel {
	var raw []sweepTouch
	for i := range bars {
		if highs {
			if !isSwingHigh(bars, i, sweepSwingK) {
				continue
			}
			raw = append(raw, sweepTouch{i, bars[i].High, bars[i].Time})
			continue
		}
		if !isSwingLow(bars, i, sweepSwingK) {
			continue
		}
		raw = append(raw, sweepTouch{i, bars[i].Low, bars[i].Time})
	}
	if len(raw) == 0 {
		return nil
	}
	used := make([]bool, len(raw))
	var out []sweepLevel
	for i := range raw {
		if used[i] {
			continue
		}
		members := []sweepTouch{raw[i]}
		used[i] = true
		for j := i + 1; j < len(raw); j++ {
			if used[j] {
				continue
			}
			mid := (raw[i].price + raw[j].price) / 2
			if mid <= 0 {
				continue
			}
			if math.Abs(raw[j].price-raw[i].price)/mid > sweepClusterPct {
				continue
			}
			used[j] = true
			members = append(members, raw[j])
		}
		if len(members) < sweepMinTests {
			continue
		}
		sort.Slice(members, func(a, b int) bool { return members[a].idx < members[b].idx })
		lv := sweepLevel{highs: highs, touches: members}
		for _, m := range members {
			if highs {
				if m.price > lv.price {
					lv.price = m.price
				}
			} else if lv.price == 0 || m.price < lv.price {
				lv.price = m.price
			}
		}
		if lv.price > 0 {
			out = append(out, lv)
		}
	}
	return out
}

func scanSweeps(bars []SweepBar, levels []sweepLevel, high bool, step time.Duration) []LiquiditySweep {
	var out []LiquiditySweep
	for _, lv := range levels {
		if len(lv.touches) < sweepMinTests {
			continue
		}
		// Eligible after the 2nd touch is confirmed as a swing.
		start := lv.touches[sweepMinTests-1].idx + sweepSwingK + 1
		if start < 0 {
			start = 0
		}
		i := start
		for i < len(bars) {
			sw, next := takeSweep(bars, i, lv, high, step)
			if next <= i {
				i++
				continue
			}
			if sw != nil {
				out = append(out, *sw)
			}
			i = next
		}
	}
	return out
}

func takeSweep(bars []SweepBar, from int, lv sweepLevel, high bool, step time.Duration) (*LiquiditySweep, int) {
	if from >= len(bars) {
		return nil, from + 1
	}
	// Level is only the tests that already existed — later swings must not
	// rewrite a poke that already printed.
	price, tests, _ := snapshotLevel(lv, from)
	if price <= 0 || tests < sweepMinTests {
		return nil, from + 1
	}
	pierce := -1
	for i := from; i < len(bars); i++ {
		// If a new test prints before we pierce, freeze the shelf as of this bar.
		if p, n, _ := snapshotLevel(lv, i); n >= sweepMinTests && p > 0 {
			price, tests = p, n
		}
		if high && bars[i].High > price && excursionPct(bars[i].High, price) >= sweepMinExcursionPct {
			pierce = i
			break
		}
		if !high && bars[i].Low > 0 && bars[i].Low < price && excursionPct(price, bars[i].Low) >= sweepMinExcursionPct {
			pierce = i
			break
		}
	}
	if pierce < 0 {
		return nil, len(bars)
	}
	price, tests, _ = snapshotLevel(lv, pierce)
	if price <= 0 || tests < sweepMinTests {
		return nil, pierce + 1
	}
	lvAtPierce := sweepLevel{price: price, highs: lv.highs, touches: touchesBefore(lv.touches, pierce)}
	end := pierce
	extreme := bars[pierce].High
	if !high {
		extreme = bars[pierce].Low
	}
	reclaimed := false
	limit := pierce + sweepMaxBars
	if limit > len(bars)-1 {
		limit = len(bars) - 1
	}
	for j := pierce; j <= limit; j++ {
		end = j
		if high && bars[j].High > extreme {
			extreme = bars[j].High
		}
		if !high && bars[j].Low > 0 && (extreme == 0 || bars[j].Low < extreme) {
			extreme = bars[j].Low
		}
		if high && bars[j].Close < price {
			reclaimed = true
			break
		}
		if !high && bars[j].Close > price {
			reclaimed = true
			break
		}
	}
	if !reclaimed {
		// Stayed beyond the level for the full window — this shelf broke.
		if end-pierce+1 > sweepMaxBars || pierce < len(bars)-sweepMaxBars {
			return nil, len(bars)
		}
		// Recent poke, series still inside the 2h window.
	}
	sw := buildSweep(bars, lvAtPierce, pierce, end, extreme, high, reclaimed, step)
	next := end + 1
	if !reclaimed {
		next = len(bars)
	}
	return &sw, next
}

func buildSweep(bars []SweepBar, lv sweepLevel, pierce, end int, extreme float64, high, reclaimed bool, step time.Duration) LiquiditySweep {
	out := LiquiditySweep{
		Level: lv.price, LevelTime: lastTouchBefore(lv.touches, pierce).time,
		Tests:     countTouchesBefore(lv.touches, pierce),
		PiercedAt: bars[pierce].Time, Extreme: extreme,
		Bars: end - pierce + 1, Interval: intervalLabel(step),
	}
	if high {
		out.Side = LiquiditySweepSideHigh
		out.Excursion = extreme - lv.price
	} else {
		out.Side = LiquiditySweepSideLow
		out.Excursion = lv.price - extreme
	}
	if lv.price > 0 {
		out.ExcursionPct = out.Excursion / lv.price * 100
	}
	var buyKnown bool
	for i := pierce; i <= end; i++ {
		out.Volume += bars[i].Volume
		if bars[i].BuySellKnown {
			out.BuyVolume += bars[i].BuyVolume
			out.SellVolume += bars[i].SellVolume
			buyKnown = true
		}
	}
	out.BuySellKnown = buyKnown
	out.DurationSeconds = out.Bars * int(step/time.Second)
	out.Duration = formatCVDDuration(time.Duration(out.DurationSeconds) * time.Second)
	if reclaimed {
		out.Status = LiquiditySweepSwept
		out.ReclaimedAt = bars[end].Time.Add(step)
	} else {
		out.Status = LiquiditySweepOpen
	}
	out.Title = sweepTitle(out)
	out.Summary = explainSweep(out)
	return out
}

func sweepTitle(s LiquiditySweep) string {
	side := "high"
	if s.Side == LiquiditySweepSideLow {
		side = "low"
	}
	if s.Status == LiquiditySweepOpen {
		return "open " + side + " sweep"
	}
	return side + " sweep"
}

func explainSweep(s LiquiditySweep) string {
	dir := "above"
	back := "under"
	if s.Side == LiquiditySweepSideLow {
		dir = "below"
		back = "over"
	}
	vol := formatQty(s.Volume)
	if s.Status == LiquiditySweepOpen {
		return fmt.Sprintf("price is %s %s (%.2f%% through) after %d tests — not back %s yet (%s volume so far).",
			dir, formatQty(s.Level), s.ExcursionPct, s.Tests, back, vol)
	}
	return fmt.Sprintf("swept %s %s (%.2f%% through), back %s in %s on %s volume (%d tests before).",
		sideWord(s.Side), formatQty(s.Level), s.ExcursionPct, back, s.Duration, vol, s.Tests)
}

func sideWord(side string) string {
	if side == LiquiditySweepSideLow {
		return "low"
	}
	return "high"
}

// snapshotLevel is the shelf as of beforeIdx: only tests that already printed.
func snapshotLevel(lv sweepLevel, beforeIdx int) (price float64, tests int, last sweepTouch) {
	for _, t := range lv.touches {
		if t.idx >= beforeIdx {
			continue
		}
		tests++
		last = t
		if lv.highs {
			if t.price > price {
				price = t.price
			}
		} else if price == 0 || t.price < price {
			price = t.price
		}
	}
	return price, tests, last
}

func touchesBefore(touches []sweepTouch, idx int) []sweepTouch {
	out := make([]sweepTouch, 0, len(touches))
	for _, t := range touches {
		if t.idx < idx {
			out = append(out, t)
		}
	}
	return out
}

func countTouchesBefore(touches []sweepTouch, idx int) int {
	n := 0
	for _, t := range touches {
		if t.idx < idx {
			n++
		}
	}
	return n
}

func lastTouchBefore(touches []sweepTouch, idx int) sweepTouch {
	var last sweepTouch
	for _, t := range touches {
		if t.idx < idx {
			last = t
		}
	}
	return last
}

func sweepOHLC(bars []SweepBar) []OHLCBar {
	out := make([]OHLCBar, len(bars))
	for i, b := range bars {
		out[i] = OHLCBar{Time: b.Time, Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, QuoteVol: b.Volume}
	}
	return out
}

func excursionPct(far, level float64) float64 {
	if level <= 0 {
		return 0
	}
	return math.Abs(far-level) / level * 100
}

func intervalLabel(d time.Duration) string {
	switch {
	case d == time.Minute:
		return string(Interval1m)
	case d == 5*time.Minute:
		return string(Interval5m)
	case d == 15*time.Minute:
		return string(Interval15m)
	case d == time.Hour:
		return string(Interval1h)
	default:
		return formatCVDDuration(d)
	}
}

func dedupeSweeps(in []LiquiditySweep) []LiquiditySweep {
	if len(in) < 2 {
		return in
	}
	keep := make([]bool, len(in))
	for i := range keep {
		keep[i] = true
	}
	for i := range in {
		if !keep[i] {
			continue
		}
		for j := i + 1; j < len(in); j++ {
			if !keep[j] || in[i].Side != in[j].Side {
				continue
			}
			mid := (in[i].Level + in[j].Level) / 2
			if mid <= 0 || math.Abs(in[i].Level-in[j].Level)/mid > sweepClusterPct*2 {
				continue
			}
			// Same poke (overlapping or identical pierce).
			if !in[i].PiercedAt.Equal(in[j].PiercedAt) &&
				math.Abs(in[i].PiercedAt.Sub(in[j].PiercedAt).Minutes()) > 30 {
				continue
			}
			// Keep the one with more tests, then farther excursion.
			drop := j
			if in[j].Tests > in[i].Tests || (in[j].Tests == in[i].Tests && in[j].Excursion > in[i].Excursion) {
				drop = i
			}
			keep[drop] = false
			if drop == i {
				break
			}
		}
	}
	out := make([]LiquiditySweep, 0, len(in))
	for i, s := range in {
		if keep[i] {
			out = append(out, s)
		}
	}
	return out
}
