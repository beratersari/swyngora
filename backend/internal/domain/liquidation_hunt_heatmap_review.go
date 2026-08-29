package domain

import (
	"sort"
	"time"
)

const (
	// HuntHeatmapHotFrac is the share of a column's peak that counts as a
	// "high" liquidation area (contiguous bins at or above this).
	HuntHeatmapHotFrac = 0.6
)

// HuntHeatmapReviewHorizons is the look-ahead set for zone validation.
var HuntHeatmapReviewHorizons = []struct {
	ID  string
	Dur time.Duration
}{
	{"1h", time.Hour},
	{"4h", 4 * time.Hour},
	{"12h", 12 * time.Hour},
}

// HuntHeatmapReviewHorizon is hit / miss stats for one look-ahead.
// HitRate and LiqIncreaseRate use only Validated signals (enough price
// path and liquidation history). Missing is everything else.
type HuntHeatmapReviewHorizon struct {
	Horizon            string  `json:"horizon"`
	Signals            int     `json:"signals"`
	Validated          int     `json:"validated"`
	Missing            int     `json:"missing"`
	PriceReady         int     `json:"priceReady"`
	PriceMissing       int     `json:"priceMissing"`
	LiqReady           int     `json:"liqReady"`
	LiqMissing         int     `json:"liqMissing"`
	Pending            int     `json:"pending"`
	Coverage           float64 `json:"coverage"`
	Hits               int     `json:"hits"`
	FalseSignals       int     `json:"falseSignals"`
	HitRate            float64 `json:"hitRate"`
	AvgTimeToHitSec    float64 `json:"avgTimeToHitSec"`
	MedianTimeToHitSec float64 `json:"medianTimeToHitSec"`
	LiqIncreased       int     `json:"liqIncreased"`
	LiqIncreaseRate    float64 `json:"liqIncreaseRate"`
	AvgLiqBefore       float64 `json:"avgLiqBefore"`
	AvgLiqAfter        float64 `json:"avgLiqAfter"`
}

// HuntHeatmapReviewVenue is validation for one venue (or combined).
type HuntHeatmapReviewVenue struct {
	Exchange string                     `json:"exchange"`
	Horizons []HuntHeatmapReviewHorizon `json:"horizons"`
}

// HuntHeatmapReview scores whether later price actually reached the hot
// zones the heatmap painted, how long that took, and whether real
// liquidations in that zone rose. Each venue uses only its own prices.
type HuntHeatmapReview struct {
	HotFrac  float64                `json:"hotFrac"`
	Binance  HuntHeatmapReviewVenue `json:"binance"`
	Bybit    HuntHeatmapReviewVenue `json:"bybit"`
	Combined HuntHeatmapReviewVenue `json:"combined"`
	Note     string                 `json:"note,omitempty"`
}

const huntHeatmapReviewNote = "Hot zones are contiguous bins at or above 60% of that column's peak, excluding the bin that already holds that venue's mark. A signal is validated only when that venue's own later candles span the 1h / 4h / 12h window and liquidation history covers the same-length windows before and after. Hit rate and liquidation-increase rate use only validated signals. Missing price or liquidation data is counted separately and is not filled from the other exchange. Combined uses the summed grid and counts a hit if either venue's own price reached the zone. Informational only — not financial advice."

func emptyHuntHeatmapReview() HuntHeatmapReview {
	return HuntHeatmapReview{
		HotFrac:  HuntHeatmapHotFrac,
		Binance:  emptyHuntReviewVenue("binance"),
		Bybit:    emptyHuntReviewVenue("bybit"),
		Combined: emptyHuntReviewVenue("combined"),
		Note:     huntHeatmapReviewNote,
	}
}

func emptyHuntReviewVenue(ex string) HuntHeatmapReviewVenue {
	hs := make([]HuntHeatmapReviewHorizon, len(HuntHeatmapReviewHorizons))
	for i, h := range HuntHeatmapReviewHorizons {
		hs[i] = HuntHeatmapReviewHorizon{Horizon: h.ID}
	}
	return HuntHeatmapReviewVenue{Exchange: ex, Horizons: hs}
}

// ReviewHuntHeatmap scores past hot zones against later venue-local prices
// and observed liquidation prints.
func ReviewHuntHeatmap(rep HuntHeatmapReport, venues []HuntHeatmapVenueSeries, now time.Time) HuntHeatmapReview {
	out := emptyHuntHeatmapReview()
	if len(rep.Times) == 0 || len(rep.Prices) == 0 || rep.PriceStep <= 0 {
		return out
	}
	if now.IsZero() {
		now = rep.To
	}
	now = now.UTC()
	byEx := map[Exchange]HuntHeatmapVenueSeries{}
	for _, v := range venues {
		byEx[v.Exchange] = v
	}
	bin, byb := byEx[ExchangeBinance], byEx[ExchangeBybit]
	nBin := len(rep.Prices)
	out.Binance = reviewHuntVenue("binance", rep.Binance, rep, now, [][]HuntHeatmapPricePoint{bin.Prices}, [][]LiquidationEvent{bin.Liquidations}, nBin)
	out.Bybit = reviewHuntVenue("bybit", rep.Bybit, rep, now, [][]HuntHeatmapPricePoint{byb.Prices}, [][]LiquidationEvent{byb.Liquidations}, nBin)
	out.Combined = reviewHuntVenue("combined", rep.Combined, rep, now,
		[][]HuntHeatmapPricePoint{bin.Prices, byb.Prices},
		[][]LiquidationEvent{bin.Liquidations, byb.Liquidations},
		nBin,
	)
	return out
}

func reviewHuntVenue(ex string, grid HuntHeatmapGrid, rep HuntHeatmapReport, now time.Time, paths [][]HuntHeatmapPricePoint, liqs [][]LiquidationEvent, nBin int) HuntHeatmapReviewVenue {
	out := emptyHuntReviewVenue(ex)
	if len(rep.Times) == 0 || nBin <= 0 {
		return out
	}
	stale := time.Duration(rep.StepSec) * time.Second * 2
	if stale < 20*time.Minute {
		stale = 20 * time.Minute
	}
	step := time.Duration(rep.StepSec) * time.Second
	if step <= 0 {
		step = 30 * time.Minute
	}
	for hi, spec := range HuntHeatmapReviewHorizons {
		acc := huntReviewAcc{horizon: spec.ID}
		var liqBefore, liqAfter float64
		var liqN int
		for i, t := range rep.Times {
			if i >= len(grid.Totals) {
				break
			}
			col := grid.Totals[i]
			marks := huntMarksAt(paths, t, stale)
			zones := huntHotZones(col, rep.PriceMin, rep.PriceMax, rep.PriceStep, nBin, marks)
			if len(zones) == 0 {
				continue
			}
			until := t.Add(spec.Dur)
			from := t.Add(-spec.Dur)
			clockOK := !now.Before(until)
			priceOK := clockOK && huntPricePathCovers(paths, t, until, step)
			liqOK := clockOK && huntLiqFeedCovers(liqs, from, until)
			fwd := huntForwardBars(paths, t, until)
			for _, z := range zones {
				acc.signals++
				if !clockOK {
					acc.pending++
				}
				if priceOK {
					acc.priceReady++
				} else {
					acc.priceMissing++
				}
				if liqOK {
					acc.liqReady++
				} else {
					acc.liqMissing++
				}
				if !priceOK || !liqOK {
					acc.missing++
					continue
				}
				acc.validated++
				hitAt, ok := huntFirstTouch(fwd, z.lo, z.hi)
				before := huntLiqInZone(liqs, from, t, z.lo, z.hi)
				after := huntLiqInZone(liqs, t, until, z.lo, z.hi)
				liqBefore += before
				liqAfter += after
				liqN++
				if !ok {
					acc.misses++
					continue
				}
				acc.hits++
				sec := hitAt.Sub(t).Seconds()
				if sec < 0 {
					sec = 0
				}
				acc.hitSecs = append(acc.hitSecs, sec)
				if after > before {
					acc.liqUp++
				}
			}
		}
		out.Horizons[hi] = acc.finish(liqBefore, liqAfter, liqN)
	}
	return out
}

type huntReviewAcc struct {
	horizon      string
	signals      int
	validated    int
	missing      int
	priceReady   int
	priceMissing int
	liqReady     int
	liqMissing   int
	pending      int
	hits         int
	misses       int
	liqUp        int
	hitSecs      []float64
}

func (a huntReviewAcc) finish(liqBefore, liqAfter float64, liqN int) HuntHeatmapReviewHorizon {
	h := HuntHeatmapReviewHorizon{
		Horizon:      a.horizon,
		Signals:      a.signals,
		Validated:    a.validated,
		Missing:      a.missing,
		PriceReady:   a.priceReady,
		PriceMissing: a.priceMissing,
		LiqReady:     a.liqReady,
		LiqMissing:   a.liqMissing,
		Pending:      a.pending,
		Hits:         a.hits,
		FalseSignals: a.misses,
	}
	if a.signals > 0 {
		h.Coverage = float64(a.validated) / float64(a.signals)
	}
	if a.validated > 0 {
		h.HitRate = float64(a.hits) / float64(a.validated)
	}
	if len(a.hitSecs) > 0 {
		sum := 0.0
		for _, s := range a.hitSecs {
			sum += s
		}
		h.AvgTimeToHitSec = sum / float64(len(a.hitSecs))
		h.MedianTimeToHitSec = medianFloat64(a.hitSecs)
	}
	if a.hits > 0 {
		h.LiqIncreased = a.liqUp
		h.LiqIncreaseRate = float64(a.liqUp) / float64(a.hits)
	}
	if liqN > 0 {
		h.AvgLiqBefore = liqBefore / float64(liqN)
		h.AvgLiqAfter = liqAfter / float64(liqN)
	}
	return h
}

type huntZone struct {
	lo, hi float64
}

func huntHotZones(col []float64, pMin, pMax, pStep float64, nBin int, marks []float64) []huntZone {
	if nBin <= 0 || pStep <= 0 || len(col) == 0 {
		return nil
	}
	peak := 0.0
	for _, v := range col {
		if v > peak {
			peak = v
		}
	}
	if peak <= 0 {
		return nil
	}
	thresh := peak * HuntHeatmapHotFrac
	hot := make([]bool, nBin)
	for j := 0; j < nBin && j < len(col); j++ {
		if col[j] >= thresh && col[j] > 0 {
			hot[j] = true
		}
	}
	var out []huntZone
	for j := 0; j < nBin; {
		if !hot[j] {
			j++
			continue
		}
		k := j + 1
		for k < nBin && hot[k] {
			k++
		}
		z := huntZone{
			hi: pMax - float64(j)*pStep,
			lo: pMax - float64(k)*pStep,
		}
		if z.lo < pMin {
			z.lo = pMin
		}
		if huntZoneHasMark(z, marks) {
			j = k
			continue
		}
		out = append(out, z)
		j = k
	}
	return out
}

func huntZoneHasMark(z huntZone, marks []float64) bool {
	for _, m := range marks {
		if m > 0 && m >= z.lo && m <= z.hi {
			return true
		}
	}
	return false
}

func huntMarksAt(paths [][]HuntHeatmapPricePoint, t time.Time, stale time.Duration) []float64 {
	out := make([]float64, 0, len(paths))
	for _, p := range paths {
		if px := nearestHuntPrice(p, t, stale); px > 0 {
			out = append(out, px)
		}
	}
	return out
}

func huntPricePathCovers(paths [][]HuntHeatmapPricePoint, after, until time.Time, step time.Duration) bool {
	fwd := huntForwardBars(paths, after, until)
	if len(fwd) == 0 {
		return false
	}
	var last time.Time
	for _, p := range fwd {
		if p.Time.After(last) {
			last = p.Time
		}
	}
	slack := step
	if slack < 15*time.Minute {
		slack = 15 * time.Minute
	}
	return !last.Before(until.Add(-slack))
}

// huntLiqFeedCovers is true when liquidation history spans [from, to], so a
// quiet window is a real zero rather than a missing feed.
func huntLiqFeedCovers(groups [][]LiquidationEvent, from, to time.Time) bool {
	var minT, maxT time.Time
	any := false
	for _, ev := range groups {
		for _, e := range ev {
			if e.Time.IsZero() {
				continue
			}
			if !any || e.Time.Before(minT) {
				minT = e.Time
			}
			if !any || e.Time.After(maxT) {
				maxT = e.Time
			}
			any = true
		}
	}
	if !any {
		return false
	}
	return !minT.After(from) && !maxT.Before(to)
}

func huntForwardBars(paths [][]HuntHeatmapPricePoint, after, until time.Time) []HuntHeatmapPricePoint {
	var out []HuntHeatmapPricePoint
	for _, path := range paths {
		for _, p := range path {
			if p.Price <= 0 && p.High <= 0 && p.Low <= 0 {
				continue
			}
			if p.Time.After(after) && !p.Time.After(until) {
				out = append(out, p)
			}
		}
	}
	return out
}

func huntFirstTouch(bars []HuntHeatmapPricePoint, lo, hi float64) (time.Time, bool) {
	var best time.Time
	found := false
	for _, p := range bars {
		if !huntBarTouches(p, lo, hi) {
			continue
		}
		if !found || p.Time.Before(best) {
			best = p.Time
			found = true
		}
	}
	return best, found
}

func huntBarTouches(p HuntHeatmapPricePoint, lo, hi float64) bool {
	h, l := p.High, p.Low
	if h <= 0 {
		h = p.Price
	}
	if l <= 0 {
		l = p.Price
	}
	if h < l {
		h, l = l, h
	}
	return l <= hi && h >= lo
}

func huntLiqInZone(groups [][]LiquidationEvent, from, to time.Time, lo, hi float64) float64 {
	sum := 0.0
	for _, ev := range groups {
		for _, e := range ev {
			if e.Notional <= 0 || e.Price <= 0 {
				continue
			}
			if !e.Time.After(from) || e.Time.After(to) {
				continue
			}
			if e.Price >= lo && e.Price <= hi {
				sum += e.Notional
			}
		}
	}
	return sum
}

func medianFloat64(in []float64) float64 {
	if len(in) == 0 {
		return 0
	}
	s := append([]float64(nil), in...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
