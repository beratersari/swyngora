package domain

import (
	"math"
	"time"
)

const (
	WhaleIntentOpenLong   = "open_long"
	WhaleIntentCloseShort = "close_short"
	WhaleIntentOpenShort  = "open_short"
	WhaleIntentCloseLong  = "close_long"
	WhaleIntentUnclear    = "unclear"

	// WhaleFollowAfter is how long after the print we read price and OI.
	WhaleFollowAfter = 15 * time.Minute
	// whaleOIFlatPct treats a smaller open-interest move as noise.
	whaleOIFlatPct = 0.05
	whaleOISlack   = 15 * time.Minute
)

// WhaleMark is a price at a time, used to see what the market did after a print.
type WhaleMark struct {
	Time  time.Time
	Price float64
}

// ClassifyWhaleIntent reads a taker buy or sell together with the open-interest
// change across that print. Buy + OI up opens longs. Buy + OI down closes shorts.
// Sell + OI up opens shorts. Sell + OI down closes longs.
func ClassifyWhaleIntent(side string, oiBefore, oiAfter float64) (intent string, changePct float64) {
	if oiBefore <= 0 || oiAfter <= 0 {
		return WhaleIntentUnclear, 0
	}
	changePct = (oiAfter - oiBefore) / oiBefore * 100
	if math.Abs(changePct) < whaleOIFlatPct {
		return WhaleIntentUnclear, changePct
	}
	up := changePct > 0
	switch side {
	case WhaleSideBuy, WhaleSideLong:
		if up {
			return WhaleIntentOpenLong, changePct
		}
		return WhaleIntentCloseShort, changePct
	case WhaleSideSell, WhaleSideShort:
		if up {
			return WhaleIntentOpenShort, changePct
		}
		return WhaleIntentCloseLong, changePct
	default:
		return WhaleIntentUnclear, changePct
	}
}

// AnnotateWhaleFollow sets intent from OI across the print, and the price and
// OI change 15 minutes later when those samples exist.
func AnnotateWhaleFollow(ev *WhaleEvent, oi []FuturesSnapshot, marks []WhaleMark, now time.Time) {
	if ev == nil {
		return
	}
	if ev.Kind == WhaleKindLiquidation {
		if ev.Side == WhaleSideLong {
			ev.Intent = WhaleIntentCloseLong
		} else if ev.Side == WhaleSideShort {
			ev.Intent = WhaleIntentCloseShort
		}
	}
	before := whaleOIAtOrBefore(oi, ev.Exchange, ev.Symbol, ev.FirstTime)
	afterPrint := whaleOIAtOrAfter(oi, ev.Exchange, ev.Symbol, ev.LastTime)
	if before != nil && afterPrint != nil && afterPrint.SampledAt.After(before.SampledAt) && before.Value > 0 {
		intent, pct := ClassifyWhaleIntent(ev.Side, before.Value, afterPrint.Value)
		ev.OIChangePct = pct
		if ev.Kind != WhaleKindLiquidation {
			ev.Intent = intent
		}
	} else if ev.Intent == "" {
		ev.Intent = WhaleIntentUnclear
	}
	target := ev.LastTime.Add(WhaleFollowAfter)
	if now.Before(target) {
		return
	}
	if px, ok := whalePriceAtOrAfter(marks, target); ok && ev.AvgPrice > 0 {
		ev.PriceAfterPct = (px - ev.AvgPrice) / ev.AvgPrice * 100
		ev.AfterReady = true
	}
	if later := whaleOIAtOrAfter(oi, ev.Exchange, ev.Symbol, target); later != nil && afterPrint != nil && afterPrint.Value > 0 {
		ev.OIAfterPct = (later.Value - afterPrint.Value) / afterPrint.Value * 100
		ev.AfterReady = true
	}
}

func whaleOIAtOrBefore(rows []FuturesSnapshot, ex Exchange, symbol string, at time.Time) *FuturesSnapshot {
	var best *FuturesSnapshot
	for i := range rows {
		r := &rows[i]
		if !whaleOIRow(r, ex, symbol) || r.SampledAt.After(at) || at.Sub(r.SampledAt) > whaleOISlack {
			continue
		}
		if best == nil || r.SampledAt.After(best.SampledAt) {
			best = r
		}
	}
	return best
}

func whaleOIAtOrAfter(rows []FuturesSnapshot, ex Exchange, symbol string, at time.Time) *FuturesSnapshot {
	var best *FuturesSnapshot
	for i := range rows {
		r := &rows[i]
		if !whaleOIRow(r, ex, symbol) || r.SampledAt.Before(at) || r.SampledAt.Sub(at) > whaleOISlack {
			continue
		}
		if best == nil || r.SampledAt.Before(best.SampledAt) {
			best = r
		}
	}
	return best
}

func whaleOIRow(r *FuturesSnapshot, ex Exchange, symbol string) bool {
	return r != nil && r.Exchange == ex && r.Symbol == symbol && r.Value > 0 && !r.SampledAt.IsZero()
}

func whalePriceAtOrAfter(marks []WhaleMark, at time.Time) (float64, bool) {
	var best *WhaleMark
	for i := range marks {
		m := &marks[i]
		if m.Price <= 0 || m.Time.Before(at) || m.Time.Sub(at) > 5*time.Minute {
			continue
		}
		if best == nil || m.Time.Before(best.Time) {
			best = m
		}
	}
	if best == nil {
		return 0, false
	}
	return best.Price, true
}

func whaleIntentPhrase(intent string) string {
	switch intent {
	case WhaleIntentOpenLong:
		return "opened longs"
	case WhaleIntentCloseShort:
		return "closed shorts"
	case WhaleIntentOpenShort:
		return "opened shorts"
	case WhaleIntentCloseLong:
		return "closed longs"
	default:
		return ""
	}
}
