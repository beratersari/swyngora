package domain

import (
	"sort"
	"time"
)

// VolumeProfileOiPoint is one open-interest reading in USDT.
type VolumeProfileOiPoint struct {
	Time  time.Time
	Value float64
}

// AttachVolumeProfileActivity adds open-interest change and liquidation
// notional onto the same price rows as traded volume.
//
// OI is not published per price. Each step between two readings is spread
// across the high–low of the candles that traded while that step happened,
// the same way candle volume is spread. A rise is opened; a fall is closed.
// Liquidations use the price on the print.
//
// Prints and OI steps that fall outside the volume rows are counted on the
// venue totals only, so the ladder stays the volume price areas.
func AttachVolumeProfileActivity(v *VolumeProfileVenue, bars []VolumeProfileBar, oi []VolumeProfileOiPoint, liqs []LiquidationEvent) {
	if v == nil || v.Error != "" || v.TickSize <= 0 || len(v.Bins) == 0 {
		return
	}
	idx := make(map[int64]int, len(v.Bins))
	for i, b := range v.Bins {
		idx[priceBinIndex(b.Price, v.TickSize)] = i
	}
	step := IntervalDuration(CandleInterval(v.Interval))
	if step <= 0 {
		step = time.Minute
	}
	if n := attachOiSteps(v, idx, bars, step, oi); n > 0 {
		v.OiKnown = true
	}
	if n := attachProfileLiqs(v, idx, liqs); n > 0 {
		v.LiqKnown = true
	}
	for i := range v.Bins {
		v.OiOpened += v.Bins[i].OiOpened
		v.OiClosed += v.Bins[i].OiClosed
		v.LongLiq += v.Bins[i].LongLiq
		v.ShortLiq += v.Bins[i].ShortLiq
	}
	if v.OiKnown || v.LiqKnown {
		v.Summary = ExplainVolumeProfileVenue(*v)
	}
}

func attachOiSteps(v *VolumeProfileVenue, idx map[int64]int, bars []VolumeProfileBar, step time.Duration, oi []VolumeProfileOiPoint) int {
	pts := make([]VolumeProfileOiPoint, 0, len(oi))
	for _, p := range oi {
		if p.Time.IsZero() || p.Value <= 0 {
			continue
		}
		t := p.Time.UTC()
		if !v.From.IsZero() && t.Before(v.From) {
			continue
		}
		if !v.To.IsZero() && t.After(v.To) {
			continue
		}
		pts = append(pts, VolumeProfileOiPoint{Time: t, Value: p.Value})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].Time.Before(pts[j].Time) })
	placed := 0
	for i := 1; i < len(pts); i++ {
		delta := pts[i].Value - pts[i-1].Value
		if delta == 0 {
			continue
		}
		if spreadOiDelta(v, idx, bars, step, pts[i-1].Time, pts[i].Time, delta) {
			placed++
		}
	}
	return placed
}

func spreadOiDelta(v *VolumeProfileVenue, idx map[int64]int, bars []VolumeProfileBar, step time.Duration, t0, t1 time.Time, delta float64) bool {
	type piece struct {
		low, high, w float64
	}
	var parts []piece
	var weight float64
	for _, b := range bars {
		if b.High <= 0 || b.Low <= 0 || b.Time.IsZero() {
			continue
		}
		open := b.Time.UTC()
		end := open.Add(step)
		// Bar covers [open, end). Step is (t0, t1]. Touching only at the boundary does not overlap.
		if !end.After(t0) || !open.Before(t1) {
			continue
		}
		w := b.Volume
		if w <= 0 {
			w = 1
		}
		parts = append(parts, piece{low: b.Low, high: b.High, w: w})
		weight += w
	}
	if len(parts) == 0 {
		return false
	}
	hit := false
	for _, p := range parts {
		share := delta * (p.w / weight)
		if addSignedToBins(v, idx, p.low, p.high, share) {
			hit = true
		}
	}
	return hit
}

func addSignedToBins(v *VolumeProfileVenue, idx map[int64]int, low, high float64, delta float64) bool {
	first, last := barBinRange(low, high, v.TickSize)
	var hits []int
	for b := first; b <= last; b++ {
		if i, ok := idx[b]; ok {
			hits = append(hits, i)
		}
	}
	if len(hits) == 0 {
		return false
	}
	each := delta / float64(len(hits))
	for _, i := range hits {
		if each >= 0 {
			v.Bins[i].OiOpened += each
		} else {
			v.Bins[i].OiClosed += -each
		}
	}
	return true
}

func attachProfileLiqs(v *VolumeProfileVenue, idx map[int64]int, liqs []LiquidationEvent) int {
	n := 0
	for _, e := range liqs {
		if e.Notional <= 0 || e.Price <= 0 || e.Time.IsZero() {
			continue
		}
		t := e.Time.UTC()
		if !v.From.IsZero() && t.Before(v.From) {
			continue
		}
		if !v.To.IsZero() && t.After(v.To) {
			continue
		}
		i, ok := idx[priceBinIndex(e.Price, v.TickSize)]
		if !ok {
			continue
		}
		switch e.Side {
		case LiquidationSideShort:
			v.Bins[i].ShortLiq += e.Notional
		default:
			v.Bins[i].LongLiq += e.Notional
		}
		n++
	}
	return n
}
