package domain

import (
	"fmt"
	"math"
	"sort"
)

const (
	LevelKindSupport    = "support"
	LevelKindResistance = "resistance"
	LevelKindAt         = "at"

	LevelBreakNone        = ""
	LevelBreakApproaching = "approaching"
	LevelBreakTesting     = "testing"
	LevelBreakBroken      = "broken"

	levelSwingK         = 2
	levelBinPct         = 0.0035 // 0.35% bins
	levelMergeBins      = 1
	levelBookPadPct     = 0.004 // ±0.4% around the zone for book size
	levelApproachPct    = 0.45
	levelMaxEachSide    = 5
	levelMinTouches     = 2
	levelMinVolumeShare = 0.02
)

// PriceLevelZone is one support or resistance area.
type PriceLevelZone struct {
	Kind          string  // support | resistance | at
	Price         float64 // zone mid
	Low           float64
	High          float64
	DistancePct   float64 // (mid-last)/last*100; negative = below
	Tests         int     // distinct visits
	Volume        float64 // quote volume of bars that touched the zone
	BidNotional   float64 // resting bids in/near the zone
	AskNotional   float64 // resting asks in/near the zone
	LiquiditySide string  // bid | ask | both | none
	Break         *LevelBreakout
}

// LevelBreakout is how hard price is pressing or has gone through a zone.
type LevelBreakout struct {
	Status  string // approaching | testing | broken
	Score   int    // 0–100
	Volume  string // quiet | normal | heavy
	Book    string // thin | mixed | thick
	Taker   string // buy | sell | balanced | unknown
	Summary string
}

// LevelsReport is the API result.
type LevelsReport struct {
	Symbol      string
	Exchange    string
	Price       float64
	Supports    []PriceLevelZone
	Resistances []PriceLevelZone
	Active      *PriceLevelZone // nearest being tested / broken, if any
	Summary     string
	Note        string
}

// BuildLevelsReport finds zones and writes the summary.
func BuildLevelsReport(symbol, exchange string, bars []OHLCBar, book *RawOrderBook, taker *TakerVenueFlow, last float64) LevelsReport {
	if last <= 0 {
		last = lastBar(bars).Close
	}
	zones := FindPriceLevels(bars, book, taker, last)
	sup, res, active := splitLevelSides(zones)
	// Include "at" zones in the nearer list so sitting-on-a-shelf is visible.
	for _, z := range zones {
		if z.Kind != LevelKindAt {
			continue
		}
		if last >= z.Price {
			sup = append([]PriceLevelZone{z}, sup...)
		} else {
			res = append([]PriceLevelZone{z}, res...)
		}
	}
	return LevelsReport{
		Symbol: symbol, Exchange: exchange, Price: last,
		Supports: sup, Resistances: res, Active: active,
		Summary: ExplainLevels(symbol, last, sup, res, active),
	}
}

// FindPriceLevels clusters swing + volume nodes and attaches book + breakout.
func FindPriceLevels(bars []OHLCBar, book *RawOrderBook, taker *TakerVenueFlow, last float64) []PriceLevelZone {
	if last <= 0 {
		last = lastBar(bars).Close
	}
	if last <= 0 || len(bars) < 8 {
		return nil
	}
	width := last * levelBinPct
	if width <= 0 {
		return nil
	}
	type bin struct {
		vol    float64
		n      int
		swings int
	}
	acc := map[int]*bin{}
	bump := func(px, vol float64, swing bool) {
		if px <= 0 {
			return
		}
		k := int(math.Round(px / width))
		b := acc[k]
		if b == nil {
			b = &bin{}
			acc[k] = b
		}
		b.n++
		if vol > 0 {
			b.vol += vol
		}
		if swing {
			b.swings++
		}
	}
	for i, b := range bars {
		bump(b.High, b.QuoteVol*0.5, isSwingHigh(bars, i, levelSwingK))
		bump(b.Low, b.QuoteVol*0.5, isSwingLow(bars, i, levelSwingK))
		bump(b.Close, 0, false)
	}
	if len(acc) == 0 {
		return nil
	}
	var totalVol float64
	for _, b := range acc {
		totalVol += b.vol
	}
	keys := make([]int, 0, len(acc))
	for k := range acc {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	used := map[int]bool{}
	var zones []PriceLevelZone
	for _, k := range keys {
		if used[k] {
			continue
		}
		b := acc[k]
		// Keep bins that were swung to or hold a meaningful slice of volume.
		if b.swings == 0 && b.n < 3 && (totalVol <= 0 || b.vol < totalVol*levelMinVolumeShare) {
			continue
		}
		lo, hi := k, k
		vol, n, swings := b.vol, b.n, b.swings
		used[k] = true
		for d := 1; d <= levelMergeBins; d++ {
			if nb := acc[k+d]; nb != nil && !used[k+d] {
				used[k+d] = true
				hi = k + d
				vol += nb.vol
				n += nb.n
				swings += nb.swings
			}
			if nb := acc[k-d]; nb != nil && !used[k-d] {
				used[k-d] = true
				if k-d < lo {
					lo = k - d
				}
				vol += nb.vol
				n += nb.n
				swings += nb.swings
			}
		}
		z := PriceLevelZone{
			Low:    (float64(lo) - 0.5) * width,
			High:   (float64(hi) + 0.5) * width,
			Volume: vol,
		}
		z.Price = (z.Low + z.High) / 2
		z.DistancePct = (z.Price - last) / last * 100
		switch {
		case z.High < last*(1-0.001):
			z.Kind = LevelKindSupport
		case z.Low > last*(1+0.001):
			z.Kind = LevelKindResistance
		default:
			z.Kind = LevelKindAt
		}
		z.Tests = countLevelVisits(bars, z)
		if z.Tests < 1 && swings < 1 {
			continue
		}
		attachBookLiquidity(&z, book, last)
		z.Break = ScoreLevelBreakout(z, bars, last, taker)
		zones = append(zones, z)
	}
	sort.SliceStable(zones, func(i, j int) bool {
		// Prefer more tests, then more volume, then closer to price.
		if zones[i].Tests != zones[j].Tests {
			return zones[i].Tests > zones[j].Tests
		}
		if zones[i].Volume != zones[j].Volume {
			return zones[i].Volume > zones[j].Volume
		}
		return math.Abs(zones[i].DistancePct) < math.Abs(zones[j].DistancePct)
	})
	return pickNearestLevels(zones, last)
}

func isSwingHigh(bars []OHLCBar, i, k int) bool {
	if i < k || i+k >= len(bars) {
		return false
	}
	h := bars[i].High
	for j := i - k; j <= i+k; j++ {
		if j != i && bars[j].High > h {
			return false
		}
	}
	return h > 0
}

func isSwingLow(bars []OHLCBar, i, k int) bool {
	if i < k || i+k >= len(bars) {
		return false
	}
	l := bars[i].Low
	if l <= 0 {
		return false
	}
	for j := i - k; j <= i+k; j++ {
		if j != i && bars[j].Low > 0 && bars[j].Low < l {
			return false
		}
	}
	return true
}

func countLevelVisits(bars []OHLCBar, z PriceLevelZone) int {
	visits := 0
	in := false
	for _, b := range bars {
		touch := b.High >= z.Low && b.Low <= z.High
		if touch && !in {
			visits++
		}
		in = touch
	}
	return visits
}

func attachBookLiquidity(z *PriceLevelZone, book *RawOrderBook, last float64) {
	if z == nil || book == nil {
		z.setLiqSide()
		return
	}
	half := math.Max((z.High-z.Low)/2, last*levelBookPadPct)
	// Also look a bit farther so a nearby wall still counts as "around" the area.
	if last*0.012 > half {
		half = last * 0.012
	}
	if half <= 0 {
		half = last * levelBookPadPct
	}
	lo, hi := z.Price-half, z.Price+half
	for _, lv := range book.Bids {
		if lv.Price >= lo && lv.Price <= hi && lv.Quantity > 0 {
			z.BidNotional += lv.Price * lv.Quantity
		}
	}
	for _, lv := range book.Asks {
		if lv.Price >= lo && lv.Price <= hi && lv.Quantity > 0 {
			z.AskNotional += lv.Price * lv.Quantity
		}
	}
	z.setLiqSide()
}

func (z *PriceLevelZone) setLiqSide() {
	const minN = 1.0
	hasB, hasA := z.BidNotional >= minN, z.AskNotional >= minN
	switch {
	case hasB && hasA:
		z.LiquiditySide = "both"
	case hasB:
		z.LiquiditySide = "bid"
	case hasA:
		z.LiquiditySide = "ask"
	default:
		z.LiquiditySide = "none"
	}
}

func keepNearAndTested(in []PriceLevelZone, n int) []PriceLevelZone {
	if len(in) <= n {
		return in
	}
	out := append([]PriceLevelZone(nil), in[:n]...)
	seen := map[float64]bool{}
	for _, z := range out {
		seen[z.Price] = true
	}
	// Keep a well-tested shelf even if it is farther than the nearest n.
	rest := append([]PriceLevelZone(nil), in[n:]...)
	sort.SliceStable(rest, func(i, j int) bool { return rest[i].Tests > rest[j].Tests })
	for _, z := range rest {
		if z.Tests < 3 || seen[z.Price] {
			continue
		}
		out = append(out, z)
		if len(out) >= n+2 {
			break
		}
	}
	return out
}

func pickNearestLevels(zones []PriceLevelZone, last float64) []PriceLevelZone {
	var sup, res, at []PriceLevelZone
	for _, z := range zones {
		switch z.Kind {
		case LevelKindSupport:
			sup = append(sup, z)
		case LevelKindResistance:
			res = append(res, z)
		default:
			at = append(at, z)
		}
	}
	byDist := func(a []PriceLevelZone) {
		sort.SliceStable(a, func(i, j int) bool {
			return math.Abs(a[i].DistancePct) < math.Abs(a[j].DistancePct)
		})
	}
	byDist(sup)
	byDist(res)
	byDist(at)
	sup = keepNearAndTested(sup, levelMaxEachSide)
	res = keepNearAndTested(res, levelMaxEachSide)
	// Keep at most two "at" zones (price sitting on a shelf).
	if len(at) > 2 {
		at = at[:2]
	}
	out := append(append(append([]PriceLevelZone{}, sup...), res...), at...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Price < out[j].Price })
	_ = last
	return out
}

// ScoreLevelBreakout rates a press or break using volume, book, and takers.
func ScoreLevelBreakout(z PriceLevelZone, bars []OHLCBar, last float64, taker *TakerVenueFlow) *LevelBreakout {
	if last <= 0 || z.Price <= 0 {
		return nil
	}
	dist := math.Abs(z.DistancePct)
	overlaps := last >= z.Low && last <= z.High
	broken := false
	if z.Kind == LevelKindResistance && last > z.High {
		broken = true
	}
	if z.Kind == LevelKindSupport && last < z.Low {
		broken = true
	}
	if z.Kind == LevelKindAt {
		overlaps = true
	}
	if !broken && !overlaps && dist > levelApproachPct {
		return nil
	}
	out := &LevelBreakout{Taker: TakerSideEven, Volume: "normal", Book: "mixed"}
	switch {
	case broken:
		out.Status = LevelBreakBroken
	case overlaps:
		out.Status = LevelBreakTesting
	default:
		out.Status = LevelBreakApproaching
	}

	// Recent vs typical quote volume (last 6 bars vs median of prior 24).
	volLabel, volScore := levelVolumePulse(bars)
	out.Volume = volLabel

	// Book: for a resistance break we want thin asks; support break thin bids.
	bookScore := 0
	switch {
	case z.Kind == LevelKindResistance || (broken && last > z.Price):
		if z.AskNotional <= 0 {
			out.Book = "thin"
			bookScore = 28
		} else if z.BidNotional > z.AskNotional*2 {
			out.Book = "thin"
			bookScore = 22
		} else if z.AskNotional > z.BidNotional*2 && z.AskNotional > 0 {
			out.Book = "thick"
			bookScore = 6
		} else {
			out.Book = "mixed"
			bookScore = 14
		}
	default:
		if z.BidNotional <= 0 {
			out.Book = "thin"
			bookScore = 28
		} else if z.AskNotional > z.BidNotional*2 {
			out.Book = "thin"
			bookScore = 22
		} else if z.BidNotional > z.AskNotional*2 {
			out.Book = "thick"
			bookScore = 6
		} else {
			out.Book = "mixed"
			bookScore = 14
		}
	}

	takerSide, takerScore := levelTakerPulse(taker, z.Kind, broken)
	out.Taker = takerSide

	score := volScore + bookScore + takerScore
	if broken {
		score += 10
	}
	if overlaps && !broken {
		score += 6
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	out.Score = score
	out.Summary = explainLevelBreak(z, *out, last)
	return out
}

func levelVolumePulse(bars []OHLCBar) (string, int) {
	if len(bars) < 10 {
		return "normal", 12
	}
	recentN := 6
	if recentN > len(bars) {
		recentN = len(bars)
	}
	var recent float64
	for _, b := range bars[len(bars)-recentN:] {
		recent += b.QuoteVol
	}
	priors := make([]float64, 0, 24)
	start := len(bars) - recentN - 24
	if start < 0 {
		start = 0
	}
	for i := start; i < len(bars)-recentN; i++ {
		priors = append(priors, bars[i].QuoteVol)
	}
	med := medianFloat(priors)
	if med <= 0 {
		return "normal", 12
	}
	avg := recent / float64(recentN)
	r := avg / med
	switch {
	case r >= 1.8:
		return "heavy", 36
	case r <= 0.6:
		return "quiet", 6
	default:
		return "normal", 16
	}
}

func levelTakerPulse(taker *TakerVenueFlow, kind string, broken bool) (string, int) {
	if taker == nil {
		return "unknown", 8
	}
	var w TakerWindowFlow
	for _, x := range taker.Windows {
		if x.Window == TakerWindow1h && x.Complete {
			w = x
			break
		}
	}
	if w.Window == "" {
		for _, x := range taker.Windows {
			if x.Complete {
				w = x
				break
			}
		}
	}
	if w.Window == "" {
		return "unknown", 8
	}
	side := w.Dominant
	if side == "" {
		side = TakerSideEven
	}
	wantBuy := kind == LevelKindResistance || (broken && kind != LevelKindSupport)
	switch {
	case wantBuy && side == TakerSideBuy:
		return side, 30
	case !wantBuy && side == TakerSideSell:
		return side, 30
	case side == TakerSideEven:
		return side, 12
	default:
		return side, 6 // flow fights the break
	}
}

func explainLevelBreak(z PriceLevelZone, b LevelBreakout, last float64) string {
	_ = last
	name := z.Kind
	if name == LevelKindAt {
		name = "level"
	}
	switch b.Status {
	case LevelBreakBroken:
		return fmt.Sprintf("Price has broken this %s. Volume is %s, the book nearby is %s, and takers are %s (score %d/100).",
			name, b.Volume, b.Book, b.Taker, b.Score)
	case LevelBreakTesting:
		return fmt.Sprintf("Price is testing this %s. Volume is %s, the book nearby is %s, and takers are %s (score %d/100).",
			name, b.Volume, b.Book, b.Taker, b.Score)
	default:
		return fmt.Sprintf("Price is approaching this %s. Volume is %s, the book nearby is %s, and takers are %s (score %d/100).",
			name, b.Volume, b.Book, b.Taker, b.Score)
	}
}

func splitLevelSides(zones []PriceLevelZone) (sup, res []PriceLevelZone, active *PriceLevelZone) {
	for i := range zones {
		z := zones[i]
		switch z.Kind {
		case LevelKindSupport:
			sup = append(sup, z)
		case LevelKindResistance:
			res = append(res, z)
		}
		if z.Break == nil {
			continue
		}
		if active == nil || breakRank(z.Break) > breakRank(active.Break) {
			cp := z
			active = &cp
		} else if breakRank(z.Break) == breakRank(active.Break) && math.Abs(z.DistancePct) < math.Abs(active.DistancePct) {
			cp := z
			active = &cp
		}
	}
	sort.SliceStable(sup, func(i, j int) bool { return sup[i].Price > sup[j].Price }) // nearest first (closest below)
	sort.SliceStable(res, func(i, j int) bool { return res[i].Price < res[j].Price })
	return sup, res, active
}

func breakRank(b *LevelBreakout) int {
	if b == nil {
		return 0
	}
	switch b.Status {
	case LevelBreakBroken:
		return 3
	case LevelBreakTesting:
		return 2
	case LevelBreakApproaching:
		return 1
	default:
		return 0
	}
}

// ExplainLevels writes a short S/R + breakout read.
func ExplainLevels(symbol string, last float64, sup, res []PriceLevelZone, active *PriceLevelZone) string {
	name := prettyBase(symbol)
	if last <= 0 {
		return name + ": no price to map support and resistance."
	}
	if len(sup) == 0 && len(res) == 0 {
		return name + ": not enough history to mark support or resistance."
	}
	parts := make([]string, 0, 3)
	if len(sup) > 0 {
		s := sup[0]
		parts = append(parts, fmt.Sprintf("nearest support %s (%s%%, tested %d×)",
			formatQty(s.Price), FormatSignedPct(s.DistancePct), s.Tests))
	}
	if len(res) > 0 {
		r := res[0]
		parts = append(parts, fmt.Sprintf("nearest resistance %s (%s%%, tested %d×)",
			formatQty(r.Price), FormatSignedPct(r.DistancePct), r.Tests))
	}
	head := name + " " + joinList(parts) + "."
	if active != nil && active.Break != nil {
		return head + " " + active.Break.Summary
	}
	return head
}
