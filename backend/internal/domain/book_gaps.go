package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	BookGapEasierUp   = "up"
	BookGapEasierDown = "down"
	BookGapEasierEven = "even"

	DefaultBookGapRangePct  = 2.0
	MinBookGapRangePct      = 0.5
	MaxBookGapRangePct      = 5.0
	bookGapStepPct          = 0.10
	bookGapThinFrac         = 0.15
	bookGapMinWidthPct      = 0.20
	bookGapEaseRatio        = 0.70
	MaxBookGaps             = 8
	DefaultBookGapScanLimit = 20
	MaxBookGapScanLimit     = 30
)

// BookGap is a run of price with very little resting size.
type BookGap struct {
	Side      string
	Direction string
	FromPrice string
	ToPrice   string
	FromPct   float64
	ToPct     float64
	WidthPct  float64
	Notional  string
}

// BookGaps says which way the book is thin and lists the empty stretches.
type BookGaps struct {
	Symbol       string
	Exchange     string
	MidPrice     string
	RangePct     float64
	Easier       string
	UpNotional   string
	DownNotional string
	UpGapPct     float64
	DownGapPct   float64
	Gaps         []BookGap
	Summary      string
	Note         string
}

// BookGapScanHit is one coin on the market gap scan.
type BookGapScanHit struct {
	Symbol       string
	Scope        string
	Easier       string
	UpNotional   string
	DownNotional string
	UpGapPct     float64
	DownGapPct   float64
	WidestSide   string
	WidestPct    float64
	Summary      string
	Error        string
}

// BookGapScan ranks liquid pairs by how one-sided the empty book is.
type BookGapScan struct {
	Exchange string
	Quote    string
	RangePct float64
	Universe int
	Hits     []BookGapScanHit
	Summary  string
	Note     string
}

// ClampBookGapRangePct bounds the band around mid.
func ClampBookGapRangePct(v float64) float64 {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return DefaultBookGapRangePct
	}
	if v < MinBookGapRangePct {
		return MinBookGapRangePct
	}
	if v > MaxBookGapRangePct {
		return MaxBookGapRangePct
	}
	return v
}

// ParseBookGapScanLimit clamps how many liquid pairs are walked.
func ParseBookGapScanLimit(n int) int {
	if n <= 0 {
		return DefaultBookGapScanLimit
	}
	if n < 5 {
		return 5
	}
	if n > MaxBookGapScanLimit {
		return MaxBookGapScanLimit
	}
	return n
}

// FindBookGaps splits ±rangePct into 0.1% steps and merges steps that hold
// far less size than the typical step on that side.
func FindBookGaps(symbol, exchange string, raws []RawOrderBook, mid, rangePct float64) BookGaps {
	rangePct = ClampBookGapRangePct(rangePct)
	if mid <= 0 {
		for _, raw := range raws {
			if m := midPrice(raw); m > 0 {
				mid = m
				break
			}
		}
	}
	out := BookGaps{
		Symbol:   strings.ToUpper(strings.TrimSpace(symbol)),
		Exchange: strings.ToLower(strings.TrimSpace(exchange)),
		RangePct: rangePct,
		Easier:   BookGapEasierEven,
		Gaps:     []BookGap{},
	}
	if mid <= 0 {
		out.Summary = "No mid price, so empty areas cannot be read."
		return out
	}
	steps := int(math.Round(rangePct / bookGapStepPct))
	if steps < 1 {
		steps = 1
	}
	asks := make([]float64, steps)
	bids := make([]float64, steps)
	for _, raw := range raws {
		addGapBuckets(asks, raw.Asks, mid, true)
		addGapBuckets(bids, raw.Bids, mid, false)
	}
	dec := decimalsForStep(mid/10000) + 1
	out.MidPrice = formatFixed(mid, dec)
	upN, downN := sumBuckets(asks), sumBuckets(bids)
	out.UpNotional = formatQty(upN)
	out.DownNotional = formatQty(downN)
	out.UpGapPct = round4(thinShare(asks) * 100)
	out.DownGapPct = round4(thinShare(bids) * 100)
	out.Easier = gapEase(upN, downN)
	gaps := append(collectGaps(asks, mid, dec, true), collectGaps(bids, mid, dec, false)...)
	sort.SliceStable(gaps, func(i, j int) bool {
		if gaps[i].WidthPct != gaps[j].WidthPct {
			return gaps[i].WidthPct > gaps[j].WidthPct
		}
		return wallNotional(gaps[i].Notional) < wallNotional(gaps[j].Notional)
	})
	if len(gaps) > MaxBookGaps {
		gaps = gaps[:MaxBookGaps]
	}
	out.Gaps = gaps
	out.Summary = bookGapSummary(out)
	return out
}

func addGapBuckets(buckets []float64, levels []PriceLevel, mid float64, ask bool) {
	step := bookGapStepPct / 100
	for _, lv := range levels {
		if lv.Price <= 0 || lv.Quantity <= 0 || mid <= 0 {
			continue
		}
		var dist float64
		if ask {
			if lv.Price <= mid {
				continue
			}
			dist = (lv.Price - mid) / mid
		} else {
			if lv.Price >= mid {
				continue
			}
			dist = (mid - lv.Price) / mid
		}
		idx := int(dist / step)
		if idx < 0 || idx >= len(buckets) {
			continue
		}
		buckets[idx] += lv.Price * lv.Quantity
	}
}

func sumBuckets(xs []float64) float64 {
	var n float64
	for _, v := range xs {
		n += v
	}
	return n
}

func thinShare(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cut := gapThinCut(xs)
	n := 0
	for _, v := range xs {
		if v <= cut {
			n++
		}
	}
	return float64(n) / float64(len(xs))
}

func gapThinCut(xs []float64) float64 {
	med := medianFloat(xs)
	if med <= 0 {
		return 0
	}
	return med * bookGapThinFrac
}

func collectGaps(buckets []float64, mid float64, dec int, ask bool) []BookGap {
	if len(buckets) == 0 {
		return nil
	}
	cut := gapThinCut(buckets)
	step := bookGapStepPct
	var out []BookGap
	for i := 0; i < len(buckets); {
		if buckets[i] > cut {
			i++
			continue
		}
		j := i
		var notional float64
		for j < len(buckets) && buckets[j] <= cut {
			notional += buckets[j]
			j++
		}
		width := float64(j-i) * step
		if width+1e-9 >= bookGapMinWidthPct {
			fromPct := float64(i) * step
			toPct := float64(j) * step
			var fromPx, toPx float64
			if ask {
				fromPx = mid * (1 + fromPct/100)
				toPx = mid * (1 + toPct/100)
			} else {
				fromPx = mid * (1 - toPct/100)
				toPx = mid * (1 - fromPct/100)
			}
			side := "ask"
			dir := BookGapEasierUp
			if !ask {
				side = "bid"
				dir = BookGapEasierDown
			}
			out = append(out, BookGap{
				Side: side, Direction: dir,
				FromPrice: formatFixed(fromPx, dec), ToPrice: formatFixed(toPx, dec),
				FromPct: round4(fromPct), ToPct: round4(toPct), WidthPct: round4(width),
				Notional: formatQty(notional),
			})
		}
		i = j
	}
	return out
}

func gapEase(up, down float64) string {
	if up <= 0 && down <= 0 {
		return BookGapEasierEven
	}
	if up < down*bookGapEaseRatio {
		return BookGapEasierUp
	}
	if down < up*bookGapEaseRatio {
		return BookGapEasierDown
	}
	return BookGapEasierEven
}

func bookGapSummary(g BookGaps) string {
	way := "about the same size on both sides"
	switch g.Easier {
	case BookGapEasierUp:
		way = "less size above the price, so a move up meets a thinner book"
	case BookGapEasierDown:
		way = "less size below the price, so a move down meets a thinner book"
	}
	extra := ""
	if len(g.Gaps) > 0 {
		top := g.Gaps[0]
		dir := "up"
		if top.Direction == BookGapEasierDown {
			dir = "down"
		}
		extra = fmt.Sprintf(" Widest empty stretch is %.2f%% %s.", top.WidthPct, dir)
	}
	return fmt.Sprintf("Inside ±%.2f%%, asks %s versus bids %s: %s.%s", g.RangePct, g.UpNotional, g.DownNotional, way, extra)
}

// BuildBookGapScan ranks coins with the thinner side first.
func BuildBookGapScan(exchange, quote string, rangePct float64, hits []BookGapScanHit) BookGapScan {
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	if exchange == "" {
		exchange = "all"
	}
	quote = strings.ToUpper(strings.TrimSpace(quote))
	if quote == "" {
		quote = "USDT"
	}
	rangePct = ClampBookGapRangePct(rangePct)
	kept := make([]BookGapScanHit, 0, len(hits))
	up, down := 0, 0
	for _, h := range hits {
		if h.Error != "" && h.Easier == "" {
			continue
		}
		kept = append(kept, h)
		switch h.Easier {
		case BookGapEasierUp:
			up++
		case BookGapEasierDown:
			down++
		}
	}
	sort.SliceStable(kept, func(i, j int) bool {
		return gapScanBefore(kept[i], kept[j])
	})
	return BookGapScan{
		Exchange: exchange,
		Quote:    quote,
		RangePct: rangePct,
		Universe: len(hits),
		Hits:     kept,
		Summary:  fmt.Sprintf("%d coins easier up, %d easier down, across %d pairs", up, down, len(hits)),
	}
}

func gapScanBefore(a, b BookGapScanHit) bool {
	ar, br := gapThinRatio(a), gapThinRatio(b)
	if ar != br {
		return ar < br
	}
	if a.WidestPct != b.WidestPct {
		return a.WidestPct > b.WidestPct
	}
	return a.Symbol < b.Symbol
}

func gapThinRatio(h BookGapScanHit) float64 {
	up := wallNotional(h.UpNotional)
	down := wallNotional(h.DownNotional)
	if up <= 0 && down <= 0 {
		return 1
	}
	hi, lo := up, down
	if down > hi {
		hi = down
	}
	if up < lo {
		lo = up
	}
	if hi <= 0 {
		return 1
	}
	return lo / hi
}
