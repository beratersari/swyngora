package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	OptionsRealizedDays        = 30
	OptionsVolYearDays         = 365.0
	OptionsRealizedCandleLimit = 50
	optionsRealizedMinReturns  = 20
	optionsVolGapLeanPts       = 5.0

	OptionsVolLeanImplied  = "implied_richer"
	OptionsVolLeanRealized = "realized_hotter"
	OptionsVolLeanBalanced = "balanced"
)

// OptionsVolCompare is Deribit DVOL next to 30-day realized vol for the same asset.
// Gap is DVOL minus realized (vol points): 70 vs 45 → +25 (implied richer).
type OptionsVolCompare struct {
	DVOL        float64
	HasDVOL     bool
	Realized    float64
	HasRealized bool
	Days        int
	Bars        int
	Gap         float64
	HasGap      bool
	Lean        string
	Summary     string
}

// LogReturns is ln(c_i / c_{i-1}) for positive closes, oldest first.
func LogReturns(closes []float64) []float64 {
	if len(closes) < 2 {
		return nil
	}
	out := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] <= 0 || closes[i] <= 0 || math.IsNaN(closes[i]) || math.IsNaN(closes[i-1]) {
			continue
		}
		out = append(out, math.Log(closes[i]/closes[i-1]))
	}
	return out
}

// AnnualizedCloseVol is sample stdev of log returns × √periodsPerYear, in percent.
func AnnualizedCloseVol(closes []float64, periodsPerYear float64) (float64, bool) {
	rets := LogReturns(closes)
	if len(rets) < optionsRealizedMinReturns || periodsPerYear <= 0 {
		return 0, false
	}
	s := stdev(rets)
	if s <= 0 || math.IsNaN(s) || math.IsInf(s, 0) {
		return 0, false
	}
	return s * math.Sqrt(periodsPerYear) * 100, true
}

// RealizedVolAt is close-to-close 30-day realized vol as of at, annualized with 365.
// Uses daily (or coarser) bars whose open time is in (at−days, at].
func RealizedVolAt(bars []OHLCBar, at time.Time, days int) (value float64, n int, ok bool) {
	if days <= 0 {
		days = OptionsRealizedDays
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	at = at.UTC()
	from := at.AddDate(0, 0, -days)
	closes := make([]float64, 0, days+2)
	for _, b := range bars {
		t := b.Time.UTC()
		if t.After(at) || !t.After(from) || b.Close <= 0 {
			continue
		}
		closes = append(closes, b.Close)
	}
	v, ok := AnnualizedCloseVol(closes, OptionsVolYearDays)
	return v, len(closes), ok
}

// AttachOptionsVolCompare sets live DVOL vs 30-day realized vol on the snapshot.
func AttachOptionsVolCompare(snap *OptionsSnapshot, bars []OHLCBar) {
	if snap == nil {
		return
	}
	at := snap.AsOf.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	out := OptionsVolCompare{
		Days: OptionsRealizedDays,
		Lean: OptionsVolLeanBalanced,
	}
	if snap.HasDVOL && snap.DVOL > 0 {
		out.DVOL = snap.DVOL
		out.HasDVOL = true
	}
	if rv, n, ok := RealizedVolAt(bars, at, OptionsRealizedDays); ok {
		out.Realized = rv
		out.HasRealized = true
		out.Bars = n
	}
	if out.HasDVOL && out.HasRealized {
		out.Gap = out.DVOL - out.Realized
		out.HasGap = true
		out.Lean = ClassifyOptionsVolGap(out.Gap)
	}
	out.Summary = ExplainOptionsVolCompare(snap.Asset, out)
	if !out.HasDVOL && !out.HasRealized {
		return
	}
	snap.VolCompare = &out
}

// ClassifyOptionsVolGap labels a DVOL−realized spread in vol points.
func ClassifyOptionsVolGap(gap float64) string {
	switch {
	case gap >= optionsVolGapLeanPts:
		return OptionsVolLeanImplied
	case gap <= -optionsVolGapLeanPts:
		return OptionsVolLeanRealized
	default:
		return OptionsVolLeanBalanced
	}
}

// ExplainOptionsVolCompare is the desk line for DVOL vs 30-day realized.
func ExplainOptionsVolCompare(asset string, v OptionsVolCompare) string {
	name := asset
	if name == "" {
		name = OptionsAssetBTC
	}
	switch {
	case v.HasGap:
		gapAbs := math.Abs(v.Gap)
		switch v.Lean {
		case OptionsVolLeanImplied:
			return fmt.Sprintf("%s DVOL %s vs 30-day realized %s. Implied is %s vol points richer.",
				name, formatFixed(v.DVOL, 1), formatFixed(v.Realized, 1), formatFixed(gapAbs, 1))
		case OptionsVolLeanRealized:
			return fmt.Sprintf("%s DVOL %s vs 30-day realized %s. Spot has been %s vol points wilder than options are pricing.",
				name, formatFixed(v.DVOL, 1), formatFixed(v.Realized, 1), formatFixed(gapAbs, 1))
		default:
			return fmt.Sprintf("%s DVOL %s vs 30-day realized %s. Implied and realized are close.",
				name, formatFixed(v.DVOL, 1), formatFixed(v.Realized, 1))
		}
	case v.HasDVOL && !v.HasRealized:
		return fmt.Sprintf("%s DVOL %s. Not enough daily closes for 30-day realized vol.",
			name, formatFixed(v.DVOL, 1))
	case v.HasRealized:
		return fmt.Sprintf("%s 30-day realized vol %s. DVOL is missing.",
			name, formatFixed(v.Realized, 1))
	default:
		return ""
	}
}

func attachRealizedToHistory(points []OptionsHistoryPoint, bars []OHLCBar) {
	if len(points) == 0 || len(bars) == 0 {
		return
	}
	for i := range points {
		rv, _, ok := RealizedVolAt(bars, points[i].Time, OptionsRealizedDays)
		if !ok {
			continue
		}
		points[i].Realized = rv
		points[i].HasRealized = true
		if points[i].HasDVOL {
			points[i].VolGap = points[i].DVOL - rv
			points[i].HasVolGap = true
		}
	}
}
