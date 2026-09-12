package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	AroundVenueSame           = "same"
	AroundVenueBinanceEarlier = "binance_earlier"
	AroundVenueBybitEarlier   = "bybit_earlier"
	AroundVenueBinanceOnly    = "binance_only"
	AroundVenueBybitOnly      = "bybit_only"
	AroundVenueUnknown        = "unknown"

	AroundVenueFollowTogether = "together"
	AroundVenueFollowed       = "followed"
	AroundVenueFollowStayed   = "stayed"

	AroundVenueMetricPrice  = "price"
	AroundVenueMetricOI     = "oi"
	AroundVenueMetricTakers = "takers"

	AroundVenueUntilContinued = "continued"
	AroundVenueUntilFaded     = "faded"
	AroundVenueUntilReversed  = "reversed"

	aroundVenueLookback  = 30 * time.Minute
	aroundVenueMinMove   = 0.20
	aroundVenueOIStep    = 5 * time.Minute
	aroundVenueOIMin     = 0.30
	aroundVenueTakerTail = 5 * time.Minute
	// aroundVenueUntilFresh is how close the last OI sample must be to
	// the price start before we will say continued / faded / reversed.
	// Wider than one futures-history tick (5m) so a current print still
	// counts; a longer hole means the path to price is unknown.
	aroundVenueUntilFresh = 8 * time.Minute
)

// AroundVenueTiming compares when Binance and Bybit started the price move.
type AroundVenueTiming struct {
	Agreement         string
	Follow            string
	Leader            string
	LeaderMetric      string
	BinanceStart      time.Time
	BybitStart        time.Time
	BinanceOIStart    time.Time
	BybitOIStart      time.Time
	BinanceTakerStart time.Time
	BybitTakerStart   time.Time
	Lead              time.Duration
	LeadLabel         string
	Before            []AroundVenueBeforeBit
	Summary           string
}

// AroundVenueBeforeBit is OI or taker flow that printed before the laggard's real start.
type AroundVenueBeforeBit struct {
	Metric    string
	Venue     string
	Direction string
	Phase     string
	At        time.Time
	Until     string
	Summary   string
}

type aroundVenueImpulse struct {
	dir   string
	pct   float64
	known bool
}

// AroundStamp is a time we only know to a bar or sample. The print
// happened in [At, At+Resolution). Resolution 0 is an exact tick.
type AroundStamp struct {
	At         time.Time
	Resolution time.Duration
}

// AroundStampLatest is the last moment this print could still have been.
func AroundStampLatest(s AroundStamp) time.Time {
	if s.At.IsZero() {
		return time.Time{}
	}
	if s.Resolution <= 0 {
		return s.At
	}
	return s.At.Add(s.Resolution)
}

// AroundStampBefore is true only when a is entirely before b.
func AroundStampBefore(a, b AroundStamp) bool {
	if a.At.IsZero() || b.At.IsZero() {
		return false
	}
	return !AroundStampLatest(a).After(b.At) && a.At.Before(b.At)
}

// AroundStampsOverlap is true when the two windows could be in either order.
func AroundStampsOverlap(a, b AroundStamp) bool {
	if a.At.IsZero() || b.At.IsZero() {
		return false
	}
	return !AroundStampBefore(a, b) && !AroundStampBefore(b, a)
}

func aroundVenuePriceStamp(v AroundVenue) AroundStamp {
	return AroundStamp{At: v.MoveStart.UTC(), Resolution: aroundIntervalOrMinute(v.Interval)}
}

// FindAroundPriceRunStart is when this venue first printed its own impulse.
// A venue that did not actually move returns zero — not the during-window open.
func FindAroundPriceRunStart(bars []AroundBar, plan AroundPlan) time.Time {
	during := FilterAroundBars(bars, plan.DuringFrom, plan.DuringTo)
	after := FilterAroundBars(bars, plan.AfterFrom, plan.AfterTo)
	dPx := aroundPriceOf(during)
	aPx := aroundPriceOf(after)

	var dir string
	var mag float64
	phaseFrom := plan.DuringFrom
	searchTo := plan.DuringTo
	switch {
	case aroundPriceMoved(dPx):
		dir, mag = dPx.Direction, dPx.ChangePct
	case aroundPriceMoved(aPx):
		dir, mag = aPx.Direction, aPx.ChangePct
		phaseFrom = plan.AfterFrom
		searchTo = plan.AfterTo
	default:
		return time.Time{}
	}

	pre := aroundBarBefore(bars, phaseFrom)
	if pre.Close <= 0 && len(during) > 0 {
		pre.Close = during[0].Open
		if pre.Close <= 0 {
			pre.Close = during[0].Close
		}
	}
	if pre.Close <= 0 && len(after) > 0 {
		pre.Close = after[0].Open
		if pre.Close <= 0 {
			pre.Close = after[0].Close
		}
	}
	if pre.Close <= 0 {
		return time.Time{}
	}

	thresh := math.Max(0.15, 0.20*math.Abs(mag))
	var hit time.Time
	if phaseFrom.Equal(plan.DuringFrom) {
		hit = aroundVenueLookbackStart(bars, plan, dir, thresh)
	}
	if hit.IsZero() {
		run := FilterAroundBars(bars, phaseFrom, searchTo)
		if t := firstAroundRunBar(run, pre.Close, dir, thresh); !t.IsZero() {
			hit = t
		} else if t := firstAroundRunBar(run, pre.Close, dir, 0); !t.IsZero() {
			hit = t
		} else if len(run) > 0 {
			hit = run[0].Time.UTC()
		}
	}
	step := aroundPriceBarStep(bars)
	if aroundStartIsTapeAppearance(bars, hit, phaseFrom, step) || aroundStartAfterHole(bars, hit, step) {
		return time.Time{}
	}
	return hit
}

// aroundStartAfterHole is true when the bar before start is missing or
// more than two steps back — the move may have begun in the gap.
func aroundStartAfterHole(bars []AroundBar, start time.Time, step time.Duration) bool {
	if start.IsZero() {
		return false
	}
	if step <= 0 {
		step = time.Minute
	}
	pre := aroundBarBefore(bars, start)
	if pre.Time.IsZero() {
		return true
	}
	return start.Sub(pre.Time) > 2*step
}

// aroundPriceBarStep is the smallest positive gap between candle opens.
func aroundPriceBarStep(bars []AroundBar) time.Duration {
	var times []time.Time
	for _, b := range bars {
		if b.Time.IsZero() {
			continue
		}
		times = append(times, b.Time.UTC())
	}
	if len(times) < 2 {
		return time.Minute
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	var step time.Duration
	for i := 1; i < len(times); i++ {
		d := times[i].Sub(times[i-1])
		if d > 0 && (step == 0 || d < step) {
			step = d
		}
	}
	if step <= 0 {
		return time.Minute
	}
	return step
}

// aroundStartIsTapeAppearance is true when start is just the first bar
// after a hole. We never saw this venue in the window before that print,
// so we cannot say it started then versus earlier while data was missing.
func aroundStartIsTapeAppearance(bars []AroundBar, start, coverFrom time.Time, step time.Duration) bool {
	if start.IsZero() || coverFrom.IsZero() {
		return false
	}
	if step <= 0 {
		step = time.Minute
	}
	first := aroundFirstBar(bars, coverFrom, start.Add(step))
	if first.Time.IsZero() {
		first = aroundFirstBar(bars, coverFrom, start.Add(24*time.Hour))
	}
	if first.Time.IsZero() {
		return true
	}
	if !first.Time.After(coverFrom.Add(step)) {
		return false
	}
	// First few late bars are the tape appearing, not a dated start.
	return !start.After(first.Time.Add(2 * step))
}

// aroundVenueLookbackStart is set only when price was already away from the
// lookback baseline at `at` — a wiggle that retraced does not count.
func aroundVenueLookbackStart(bars []AroundBar, plan AroundPlan, dir string, thresh float64) time.Time {
	lookFrom := plan.DuringFrom.Add(-aroundVenueLookback)
	if lookFrom.Before(plan.BeforeFrom) {
		lookFrom = plan.BeforeFrom
	}
	base := aroundBarBefore(bars, lookFrom)
	if base.Close <= 0 {
		base = aroundFirstBar(bars, lookFrom, plan.DuringFrom)
	}
	if base.Close <= 0 {
		return time.Time{}
	}
	first := aroundFirstBar(bars, plan.DuringFrom, plan.DuringTo)
	if first.Close <= 0 {
		return time.Time{}
	}
	ch := (first.Close - base.Close) / base.Close * 100
	away := (dir == CVDDirUp && ch >= thresh) || (dir == CVDDirDown && ch <= -thresh)
	if !away {
		return time.Time{}
	}
	early := FilterAroundBars(bars, lookFrom, plan.DuringFrom.Add(time.Minute))
	return firstAroundRunBar(early, base.Close, dir, thresh)
}

func aroundPriceMoved(p AroundPrice) bool {
	if p.Direction != CVDDirUp && p.Direction != CVDDirDown {
		return false
	}
	return math.Abs(p.ChangePct) >= aroundVenueMinMove
}

func aroundBarBefore(bars []AroundBar, at time.Time) AroundBar {
	var out AroundBar
	for i := range bars {
		if !bars[i].Time.Before(at) {
			break
		}
		out = bars[i]
	}
	return out
}

func aroundFirstBar(bars []AroundBar, from, to time.Time) AroundBar {
	for i := range bars {
		if bars[i].Time.Before(from) {
			continue
		}
		if !bars[i].Time.Before(to) {
			break
		}
		return bars[i]
	}
	return AroundBar{}
}

func firstAroundRunBar(bars []AroundBar, pre float64, dir string, thresh float64) time.Time {
	if pre <= 0 {
		return time.Time{}
	}
	for _, b := range bars {
		if b.Close <= 0 {
			continue
		}
		ch := (b.Close - pre) / pre * 100
		if dir == CVDDirUp && ch >= thresh && (thresh > 0 || ch > 0) {
			return b.Time.UTC()
		}
		if dir == CVDDirDown && ch <= -thresh && (thresh > 0 || ch < 0) {
			return b.Time.UTC()
		}
	}
	return time.Time{}
}

// FindAroundTakerRunStart is when this venue's takers first leaned the way
// they leaned during the impulse. Zero if they stayed balanced.
func FindAroundTakerRunStart(bars []AroundBar, plan AroundPlan) (time.Time, string) {
	during := aroundFlowOf(FilterAroundBars(bars, plan.DuringFrom, plan.DuringTo))
	after := aroundFlowOf(FilterAroundBars(bars, plan.AfterFrom, plan.AfterTo))
	want := ""
	searchFrom, searchTo := plan.DuringFrom, plan.DuringTo
	switch {
	case aroundTakerLeaned(during):
		want = during.Dominant
	case aroundTakerLeaned(after):
		want = after.Dominant
		searchFrom, searchTo = plan.AfterFrom, plan.AfterTo
	default:
		return time.Time{}, ""
	}
	pre := aroundFlowOf(FilterAroundBars(bars, searchFrom.Add(-15*time.Minute), searchFrom))
	if aroundTakerLeaned(pre) && pre.Dominant == want {
		lookFrom := plan.DuringFrom.Add(-aroundVenueLookback)
		if lookFrom.Before(plan.BeforeFrom) {
			lookFrom = plan.BeforeFrom
		}
		if t := firstAroundTakerLean(FilterAroundBars(bars, lookFrom, searchFrom), want); !t.IsZero() {
			return t, want
		}
	}
	step := aroundPriceBarStep(bars)
	if t := firstAroundTakerLean(FilterAroundBars(bars, searchFrom, searchTo), want); !t.IsZero() {
		if aroundStartIsTapeAppearance(bars, t, searchFrom, step) || aroundStartAfterHole(bars, t, step) {
			return time.Time{}, ""
		}
		return t, want
	}
	if aroundStartIsTapeAppearance(bars, searchFrom, searchFrom, step) || aroundStartAfterHole(bars, searchFrom, step) {
		return time.Time{}, ""
	}
	return searchFrom.UTC(), want
}

func aroundTakerLeaned(f AroundFlow) bool {
	return f.BuySellKnown && (f.Dominant == TakerSideBuy || f.Dominant == TakerSideSell)
}

func firstAroundTakerLean(bars []AroundBar, want string) time.Time {
	var buy, sell float64
	for _, b := range bars {
		if !b.BuySellKnown {
			continue
		}
		buy += b.BuyVolume
		sell += b.SellVolume
		if TakerDominant(buy, sell) == want {
			return b.Time.UTC()
		}
	}
	return time.Time{}
}

// FindAroundOIRunStart is when this venue's open interest first left the
// quiet baseline in the impulse direction. Zero if OI did not move.
func FindAroundOIRunStart(snaps []FuturesSnapshot, plan AroundPlan) (time.Time, string) {
	d := aroundOIWindow(snaps, plan.DuringFrom, plan.DuringTo)
	a := aroundOIWindow(snaps, plan.AfterFrom, plan.AfterTo)
	dir := ""
	mag := 0.0
	searchTo := plan.DuringTo
	switch {
	case aroundOIMoved(d):
		dir, mag = d.dir, d.pct
	case aroundOIMoved(a):
		dir, mag = a.dir, a.pct
		searchTo = plan.AfterTo
	default:
		return time.Time{}, ""
	}
	lookFrom := plan.DuringFrom.Add(-aroundVenueLookback)
	if lookFrom.Before(plan.BeforeFrom) {
		lookFrom = plan.BeforeFrom
	}
	base := LatestFuturesSnapshotAtOrBefore(snaps, lookFrom)
	if base == nil {
		base = LatestFuturesSnapshotAtOrBefore(snaps, plan.DuringFrom)
	}
	if base == nil {
		return time.Time{}, ""
	}
	baseV := futuresOIValue(*base)
	if baseV <= 0 {
		return time.Time{}, ""
	}
	thresh := math.Max(aroundVenueOIMin, 0.25*math.Abs(mag))
	var hit time.Time
	for i := range snaps {
		s := snaps[i]
		if s.SampledAt.IsZero() || !s.SampledAt.After(base.SampledAt) || s.SampledAt.After(searchTo) {
			continue
		}
		v := futuresOIValue(s)
		if v <= 0 {
			continue
		}
		ch := (v - baseV) / baseV * 100
		ok := (dir == CVDDirUp && ch >= thresh) || (dir == CVDDirDown && ch <= -thresh)
		if !ok {
			continue
		}
		if hit.IsZero() || s.SampledAt.Before(hit) {
			hit = s.SampledAt.UTC()
		}
	}
	if hit.IsZero() {
		return time.Time{}, ""
	}
	var sampleTimes []time.Time
	for i := range snaps {
		if !snaps[i].SampledAt.IsZero() {
			sampleTimes = append(sampleTimes, snaps[i].SampledAt.UTC())
		}
	}
	if aroundSampleIsTapeAppearance(sampleTimes, hit, plan.DuringFrom, aroundVenueOIStep) ||
		aroundSampleAfterHole(sampleTimes, hit, aroundVenueOIStep) {
		return time.Time{}, ""
	}
	return hit, dir
}

func aroundSampleAfterHole(samples []time.Time, start time.Time, step time.Duration) bool {
	if start.IsZero() {
		return false
	}
	if step <= 0 {
		step = aroundVenueOIStep
	}
	var pre time.Time
	for _, t := range samples {
		if t.Before(start) && (pre.IsZero() || t.After(pre)) {
			pre = t
		}
	}
	if pre.IsZero() {
		return true
	}
	return start.Sub(pre) > 2*step
}

// aroundSampleIsTapeAppearance is the futures-sample form of
// aroundStartIsTapeAppearance (first print after a hole is not a start).
func aroundSampleIsTapeAppearance(samples []time.Time, start, coverFrom time.Time, step time.Duration) bool {
	if start.IsZero() || coverFrom.IsZero() {
		return false
	}
	if step <= 0 {
		step = 5 * time.Minute
	}
	var first time.Time
	for _, t := range samples {
		if t.Before(coverFrom) {
			continue
		}
		if first.IsZero() || t.Before(first) {
			first = t
		}
	}
	if first.IsZero() {
		return true
	}
	if !first.After(coverFrom.Add(step)) {
		return false
	}
	return !start.After(first.Add(2 * step))
}

type aroundOISpan struct {
	dir string
	pct float64
}

func aroundOIWindow(snaps []FuturesSnapshot, from, to time.Time) aroundOISpan {
	if !to.After(from) {
		return aroundOISpan{}
	}
	a := LatestFuturesSnapshotAtOrBefore(snaps, from)
	b := LatestFuturesSnapshotAtOrBefore(snaps, to)
	if a == nil || b == nil {
		return aroundOISpan{}
	}
	av, bv := futuresOIValue(*a), futuresOIValue(*b)
	if av <= 0 {
		return aroundOISpan{}
	}
	pct := (bv - av) / av * 100
	return aroundOISpan{dir: changeDir(pct), pct: pct}
}

func aroundOIMoved(s aroundOISpan) bool {
	if s.dir != CVDDirUp && s.dir != CVDDirDown {
		return false
	}
	return math.Abs(s.pct) >= aroundVenueOIMin
}

// AroundVenueUntilPrice is when the price move of interest starts: this
// venue's own print if it printed before `at`, otherwise the event time.
func AroundVenueUntilPrice(moveStart time.Time, plan AroundPlan) time.Time {
	until := plan.DuringFrom
	if !moveStart.IsZero() && moveStart.Before(until) {
		return moveStart.UTC()
	}
	if until.IsZero() {
		return moveStart.UTC()
	}
	return until.UTC()
}

// MarkAroundOIUntil says whether OI kept moving the same way from start
// until the price move, or spiked and then faded / reversed. Empty when
// the last sample is the start print or is too far before price — the
// path in between is unknown, so do not guess continued / faded / reversed.
func MarkAroundOIUntil(snaps []FuturesSnapshot, start time.Time, dir string, until time.Time) string {
	if start.IsZero() || until.IsZero() || !until.After(start) || (dir != CVDDirUp && dir != CVDDirDown) {
		return ""
	}
	pre := LatestFuturesSnapshotAtOrBefore(snaps, start.Add(-time.Millisecond))
	if pre == nil {
		pre = LatestFuturesSnapshotAtOrBefore(snaps, start)
	}
	end := LatestFuturesSnapshotAtOrBefore(snaps, until)
	if pre == nil || end == nil {
		return ""
	}
	if !aroundVenueUntilFreshEnough(end.SampledAt, start, until) {
		return ""
	}
	preV := futuresOIValue(*pre)
	endV := futuresOIValue(*end)
	if preV <= 0 {
		return ""
	}
	endCh := (endV - preV) / preV * 100
	peak := 0.0
	for i := range snaps {
		s := snaps[i]
		if s.SampledAt.IsZero() || s.SampledAt.Before(start) || s.SampledAt.After(until) {
			continue
		}
		v := futuresOIValue(s)
		if v <= 0 {
			continue
		}
		ch := (v - preV) / preV * 100
		if dir == CVDDirUp && ch > peak {
			peak = ch
		}
		if dir == CVDDirDown && -ch > peak {
			peak = -ch
		}
	}
	if peak < math.Abs(endCh) {
		peak = math.Abs(endCh)
	}
	hold := math.Max(aroundVenueOIMin, 0.4*peak)
	if dir == CVDDirUp {
		if endCh <= -aroundVenueOIMin {
			return AroundVenueUntilReversed
		}
		if endCh >= hold {
			return AroundVenueUntilContinued
		}
		return AroundVenueUntilFaded
	}
	if endCh >= aroundVenueOIMin {
		return AroundVenueUntilReversed
	}
	if endCh <= -hold {
		return AroundVenueUntilContinued
	}
	return AroundVenueUntilFaded
}

// MarkAroundTakerUntil says whether takers kept leaning the same way from
// start until the price move, or spiked and then faded / flipped. Empty
// when the pre-price tape is not continuous enough to know the path —
// a last bar near price is not enough if there is a long hole after the
// start, and one start bar is not enough to say the flow kept going.
// Continuity is in bar steps (one missing 1m or 5m bar), not a fixed
// clock. The bar at `until` is the first bar of the move, not pre-price
// tape.
func MarkAroundTakerUntil(bars []AroundBar, start time.Time, dir string, until time.Time, interval CandleInterval) string {
	if start.IsZero() || until.IsZero() || !until.After(start) || (dir != TakerSideBuy && dir != TakerSideSell) {
		return ""
	}
	step := aroundTakerBarStep(bars, interval)
	if step <= 0 || !aroundTakerPathContinuous(bars, start, until, step) {
		return ""
	}
	tailFrom := until.Add(-aroundTakerTailWindow(step))
	if tailFrom.Before(start) {
		tailFrom = start
	}
	// [tailFrom, until) — never the price-start bar.
	tailBars := FilterAroundBars(bars, tailFrom, until)
	if !aroundTakerBarsKnown(tailBars) {
		return ""
	}
	tail := aroundFlowOf(tailBars)
	if aroundTakerLeaned(tail) {
		if tail.Dominant == dir {
			return AroundVenueUntilContinued
		}
		return AroundVenueUntilReversed
	}
	return AroundVenueUntilFaded
}

// aroundTakerBarStep is the tape's bar length. Prefer the venue interval;
// if that is missing, use the smallest positive gap between known bars.
func aroundTakerBarStep(bars []AroundBar, interval CandleInterval) time.Duration {
	if d := IntervalDuration(interval); d > 0 {
		return d
	}
	var times []time.Time
	for _, b := range bars {
		if !b.BuySellKnown || b.Time.IsZero() {
			continue
		}
		times = append(times, b.Time.UTC())
	}
	if len(times) < 2 {
		return 0
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	var step time.Duration
	for i := 1; i < len(times); i++ {
		d := times[i].Sub(times[i-1])
		if d > 0 && (step == 0 || d < step) {
			step = d
		}
	}
	return step
}

func aroundTakerTailWindow(step time.Duration) time.Duration {
	if step <= 0 {
		return aroundVenueTakerTail
	}
	w := 2 * step
	if w < aroundVenueTakerTail {
		return aroundVenueTakerTail
	}
	return w
}

// aroundTakerPathContinuous is true when known taker bars cover start
// through the last bar before price with no hole longer than one missing
// bar at `step`. One bar is only the start print — there must be later
// pre-price tape so the flow can be said to have kept going. The
// price-start bar is not part of this path.
func aroundTakerPathContinuous(bars []AroundBar, start, until time.Time, step time.Duration) bool {
	if start.IsZero() || until.IsZero() || !until.After(start) || step <= 0 {
		return false
	}
	var known []time.Time
	for _, b := range bars {
		if !b.BuySellKnown || b.Time.IsZero() || b.Time.Before(start) || !b.Time.Before(until) {
			continue
		}
		known = append(known, b.Time.UTC())
	}
	if len(known) < 2 {
		return false
	}
	sort.Slice(known, func(i, j int) bool { return known[i].Before(known[j]) })
	last := known[len(known)-1]
	if !last.After(start) || last.Sub(start) < step {
		return false
	}
	maxMid := 2 * step
	if known[0].Sub(start) > step {
		return false
	}
	for i := 1; i < len(known); i++ {
		if known[i].Sub(known[i-1]) > maxMid {
			return false
		}
	}
	return until.Sub(last) <= step
}

// aroundVenueUntilFreshEnough is true when last is after the start print
// and close enough to until that the path to price is still known.
func aroundVenueUntilFreshEnough(last, start, until time.Time) bool {
	if last.IsZero() || start.IsZero() || until.IsZero() {
		return false
	}
	if !last.After(start) {
		return false
	}
	age := until.Sub(last)
	return age >= 0 && age <= aroundVenueUntilFresh
}

func aroundTakerBarsKnown(bars []AroundBar) bool {
	for _, b := range bars {
		if b.BuySellKnown {
			return true
		}
	}
	return false
}

func aroundVenueHeldToPrice(until string) bool {
	return until == AroundVenueUntilContinued
}

// CompareAroundVenueTiming says whether Binance and Bybit started together,
// one printed first and the other followed, or the move stayed on one venue.
// The first signal (price, OI, or takers) is the leader — not only price,
// and not the larger move. "Before" the laggard uses that venue's real
// start time, not the during-window close.
func CompareAroundVenueTiming(venues []AroundVenue) *AroundVenueTiming {
	var bnVen, byVen AroundVenue
	var bn, by time.Time
	for _, v := range venues {
		switch v.Exchange {
		case ExchangeBinance:
			bnVen = v
			bn = v.MoveStart
		case ExchangeBybit:
			byVen = v
			by = v.MoveStart
		}
	}
	bnImp := aroundVenueImpulseOf(bnVen)
	byImp := aroundVenueImpulseOf(byVen)
	if bnImp.known && bnImp.dir == "" {
		bn = time.Time{}
		bnVen.MoveStart = time.Time{}
	}
	if byImp.known && byImp.dir == "" {
		by = time.Time{}
		byVen.MoveStart = time.Time{}
	}
	// First candle after a hole is missing tape, not a follow.
	if aroundVenueStartFromMissingTape(byVen, bn) {
		by = time.Time{}
		byVen.MoveStart = time.Time{}
	}
	if aroundVenueStartFromMissingTape(bnVen, by) {
		bn = time.Time{}
		bnVen.MoveStart = time.Time{}
	}

	if bn.IsZero() && by.IsZero() && bnVen.OIStart.IsZero() && byVen.OIStart.IsZero() &&
		bnVen.TakerStart.IsZero() && byVen.TakerStart.IsZero() {
		return nil
	}
	out := &AroundVenueTiming{}
	if !bn.IsZero() {
		out.BinanceStart = bn.UTC()
	}
	if !by.IsZero() {
		out.BybitStart = by.UTC()
	}
	if !bnVen.OIStart.IsZero() {
		out.BinanceOIStart = bnVen.OIStart.UTC()
	}
	if !byVen.OIStart.IsZero() {
		out.BybitOIStart = byVen.OIStart.UTC()
	}
	if !bnVen.TakerStart.IsZero() {
		out.BinanceTakerStart = bnVen.TakerStart.UTC()
	}
	if !byVen.TakerStart.IsZero() {
		out.BybitTakerStart = byVen.TakerStart.UTC()
	}
	out.Leader, out.LeaderMetric, _ = aroundTimingFirstSignal(bnVen, byVen)

	switch {
	case !bn.IsZero() && by.IsZero():
		out.Agreement = AroundVenueBinanceOnly
		out.Follow = AroundVenueFollowStayed
		if out.Leader == "" {
			out.Leader = string(ExchangeBinance)
			out.LeaderMetric = AroundVenueMetricPrice
		}
		out.Before = aroundVenueBeforeBits(bnVen, byVen, time.Time{})
		out.Summary = "The price move printed on Binance. Bybit did not follow on price."
		appendAroundVenueFirstSignal(out, bnVen, byVen, bn)
		appendAroundVenueBeforeSummary(out, "Bybit", time.Time{})
		return out
	case bn.IsZero() && !by.IsZero():
		out.Agreement = AroundVenueBybitOnly
		out.Follow = AroundVenueFollowStayed
		if out.Leader == "" {
			out.Leader = string(ExchangeBybit)
			out.LeaderMetric = AroundVenueMetricPrice
		}
		out.Before = aroundVenueBeforeBits(byVen, bnVen, time.Time{})
		out.Summary = "The price move printed on Bybit. Binance did not follow on price."
		appendAroundVenueFirstSignal(out, bnVen, byVen, by)
		appendAroundVenueBeforeSummary(out, "Binance", time.Time{})
		return out
	case bn.IsZero() && by.IsZero():
		out.Agreement = AroundVenueUnknown
		out.Follow = AroundVenueFollowStayed
		out.Before = aroundVenueBeforeBits(bnVen, byVen, time.Time{})
		out.Summary = aroundVenueFirstSignalSentence(bnVen, byVen, time.Time{})
		if out.Summary == "" {
			return nil
		}
		appendAroundVenueBeforeSummary(out, "", time.Time{})
		return out
	}

	delta := bn.Sub(by)
	if delta < 0 {
		delta = -delta
	}
	out.Lead = delta
	if AroundStampsOverlap(aroundVenuePriceStamp(bnVen), aroundVenuePriceStamp(byVen)) {
		out.Agreement = AroundVenueSame
		out.Follow = AroundVenueFollowTogether
		out.Before = aroundVenueBeforeBits(bnVen, byVen, time.Time{})
		out.Summary = "Binance and Bybit started the price move around the same time. The tape is not fine enough to say who printed first."
		appendAroundVenueFirstSignal(out, bnVen, byVen, time.Time{})
		appendAroundVenueBeforeSummary(out, "", time.Time{})
		return out
	}
	out.LeadLabel = formatOptionsLead(delta)
	sameWay := true
	if bnImp.known && byImp.known && bnImp.dir != "" && byImp.dir != "" {
		sameWay = bnImp.dir == byImp.dir
	}
	if bn.Before(by) {
		if out.Leader == "" {
			out.Leader = string(ExchangeBinance)
			out.LeaderMetric = AroundVenueMetricPrice
		}
		out.Before = aroundVenueBeforeBits(bnVen, byVen, by)
		if sameWay {
			out.Agreement = AroundVenueBinanceEarlier
			out.Follow = AroundVenueFollowed
			out.Summary = fmt.Sprintf("Binance started the price move %s before Bybit. Bybit followed later.", out.LeadLabel)
		} else {
			out.Agreement = AroundVenueBinanceEarlier
			out.Follow = AroundVenueFollowStayed
			out.Summary = fmt.Sprintf("Binance started the price move %s before Bybit. Bybit printed the other way.", out.LeadLabel)
		}
		appendAroundVenueFirstSignal(out, bnVen, byVen, bn)
		appendAroundVenueBeforeSummary(out, "Bybit", by)
		return out
	}
	if out.Leader == "" {
		out.Leader = string(ExchangeBybit)
		out.LeaderMetric = AroundVenueMetricPrice
	}
	out.Before = aroundVenueBeforeBits(byVen, bnVen, bn)
	if sameWay {
		out.Agreement = AroundVenueBybitEarlier
		out.Follow = AroundVenueFollowed
		out.Summary = fmt.Sprintf("Bybit started the price move %s before Binance. Binance followed later.", out.LeadLabel)
	} else {
		out.Agreement = AroundVenueBybitEarlier
		out.Follow = AroundVenueFollowStayed
		out.Summary = fmt.Sprintf("Bybit started the price move %s before Binance. Binance printed the other way.", out.LeadLabel)
	}
	appendAroundVenueFirstSignal(out, bnVen, byVen, by)
	appendAroundVenueBeforeSummary(out, "Binance", bn)
	return out
}

// aroundVenueStartFromMissingTape is true when this venue's start is
// the first bar we have, and that bar is after the other venue already
// moved — we cannot say it followed then versus earlier while data was missing.
func aroundVenueStartFromMissingTape(v AroundVenue, peerStart time.Time) bool {
	if v.MoveStart.IsZero() || v.TapeFrom.IsZero() || peerStart.IsZero() {
		return false
	}
	step := aroundIntervalOrMinute(v.Interval)
	if !v.TapeFrom.After(peerStart) {
		return false
	}
	return !v.MoveStart.After(v.TapeFrom.Add(2 * step))
}

func aroundIntervalOrMinute(raw string) time.Duration {
	if d := IntervalDuration(CandleInterval(strings.ToLower(strings.TrimSpace(raw)))); d > 0 {
		return d
	}
	return time.Minute
}

func aroundVenueImpulseOf(v AroundVenue) aroundVenueImpulse {
	d, dOK := AroundPhaseByID(v, AroundPhaseDuring)
	a, aOK := AroundPhaseByID(v, AroundPhaseAfter)
	if !dOK && !aOK {
		return aroundVenueImpulse{}
	}
	if dOK && aroundPriceMoved(d.Price) {
		return aroundVenueImpulse{dir: d.Price.Direction, pct: math.Abs(d.Price.ChangePct), known: true}
	}
	if aOK && aroundPriceMoved(a.Price) {
		return aroundVenueImpulse{dir: a.Price.Direction, pct: math.Abs(a.Price.ChangePct), known: true}
	}
	return aroundVenueImpulse{known: true}
}

func aroundVenueBeforeBits(leader, other AroundVenue, until time.Time) []AroundVenueBeforeBit {
	var out []AroundVenueBeforeBit
	out = append(out, aroundVenueTapeBits(leader, until)...)
	out = append(out, aroundVenueTapeBits(other, until)...)
	return out
}

func aroundVenueTapeBits(v AroundVenue, until time.Time) []AroundVenueBeforeBit {
	if v.Exchange == "" && len(v.Phases) == 0 && v.OIStart.IsZero() && v.TakerStart.IsZero() {
		return nil
	}
	var oi, tk AroundVenueBeforeBit
	if aroundStartBeforeCutoff(v.OIStart, until) && (v.OIDir == CVDDirUp || v.OIDir == CVDDirDown) {
		oi = aroundStartOIBit(v)
	}
	if aroundStartBeforeCutoff(v.TakerStart, until) && (v.TakerDir == TakerSideBuy || v.TakerDir == TakerSideSell) {
		tk = aroundStartTakerBit(v)
	}
	order := []string{AroundPhaseBefore, AroundPhaseDuring, AroundPhaseAfter}
	for _, id := range order {
		p, ok := AroundPhaseByID(v, id)
		if !ok || !p.Complete || !aroundPhaseBeforeCutoff(p, until) {
			continue
		}
		if oi.Metric == "" {
			oi = aroundPhaseOIBit(v.Exchange, p)
		}
		if tk.Metric == "" {
			tk = aroundPhaseTakerBit(v.Exchange, p)
		}
	}
	var out []AroundVenueBeforeBit
	if oi.Metric != "" {
		out = append(out, oi)
	}
	if tk.Metric != "" {
		out = append(out, tk)
	}
	return out
}

// aroundPhaseBeforeCutoff is true only when the whole phase ends at or before
// the laggard's real start. A 12:00–12:15 during window is not "before 12:05".
func aroundPhaseBeforeCutoff(p AroundPhase, until time.Time) bool {
	if until.IsZero() {
		return p.Phase != AroundPhaseAfter
	}
	if p.To.IsZero() {
		return false
	}
	return !p.To.After(until)
}

func aroundStartBeforeCutoff(start, until time.Time) bool {
	if start.IsZero() {
		return false
	}
	return until.IsZero() || start.Before(until)
}

func aroundStartOIBit(v AroundVenue) AroundVenueBeforeBit {
	word := "rose"
	if v.OIDir == CVDDirDown {
		word = "fell"
	}
	head := fmt.Sprintf("%s open interest %s from %s UTC",
		aroundExchangeLabel(v.Exchange), word, v.OIStart.UTC().Format("15:04"))
	return AroundVenueBeforeBit{
		Metric: AroundVenueMetricOI, Venue: string(v.Exchange), Direction: v.OIDir,
		At: v.OIStart.UTC(), Until: v.OIUntil,
		Summary: aroundVenueUntilPhrase(head, v.OIUntil),
	}
}

func aroundStartTakerBit(v AroundVenue) AroundVenueBeforeBit {
	word := "were buying"
	if v.TakerDir == TakerSideSell {
		word = "were selling"
	}
	head := fmt.Sprintf("%s takers %s from %s UTC",
		aroundExchangeLabel(v.Exchange), word, v.TakerStart.UTC().Format("15:04"))
	return AroundVenueBeforeBit{
		Metric: AroundVenueMetricTakers, Venue: string(v.Exchange), Direction: v.TakerDir,
		At: v.TakerStart.UTC(), Until: v.TakerUntil,
		Summary: aroundVenueUntilPhrase(head, v.TakerUntil),
	}
}

func aroundVenueUntilPhrase(head, until string) string {
	switch until {
	case AroundVenueUntilContinued:
		return head + " and kept going until price"
	case AroundVenueUntilFaded:
		return head + " then faded before price"
	case AroundVenueUntilReversed:
		return head + " then turned the other way before price"
	default:
		return head
	}
}

func aroundPhaseOIBit(ex Exchange, p AroundPhase) AroundVenueBeforeBit {
	if p.Futures == nil || !p.Futures.Complete {
		return AroundVenueBeforeBit{}
	}
	dir := p.Futures.OIDirection
	if dir == "" {
		dir = changeDir(p.Futures.OIChangePct)
	}
	if dir != CVDDirUp && dir != CVDDirDown {
		return AroundVenueBeforeBit{}
	}
	if math.Abs(p.Futures.OIChangePct) < aroundVenueOIMin {
		return AroundVenueBeforeBit{}
	}
	word := "rose"
	if dir == CVDDirDown {
		word = "fell"
	}
	return AroundVenueBeforeBit{
		Metric: AroundVenueMetricOI, Venue: string(ex), Direction: dir, Phase: p.Phase,
		Summary: fmt.Sprintf("%s open interest %s", aroundExchangeLabel(ex), word),
	}
}

func aroundPhaseTakerBit(ex Exchange, p AroundPhase) AroundVenueBeforeBit {
	if !p.Flow.BuySellKnown || p.Flow.Dominant == "" || p.Flow.Dominant == TakerSideEven {
		return AroundVenueBeforeBit{}
	}
	word := "were buying"
	dir := p.Flow.Dominant
	if dir == TakerSideSell {
		word = "were selling"
	} else if dir != TakerSideBuy {
		return AroundVenueBeforeBit{}
	}
	return AroundVenueBeforeBit{
		Metric: AroundVenueMetricTakers, Venue: string(ex), Direction: dir, Phase: p.Phase,
		Summary: fmt.Sprintf("%s takers %s", aroundExchangeLabel(ex), word),
	}
}

type aroundVenueClock struct {
	venue  string
	metric string
	at     time.Time
	res    time.Duration
}

func aroundVenueClocks(v AroundVenue) []aroundVenueClock {
	if v.Exchange == "" {
		return nil
	}
	ex := string(v.Exchange)
	step := aroundIntervalOrMinute(v.Interval)
	var out []aroundVenueClock
	if !v.MoveStart.IsZero() {
		out = append(out, aroundVenueClock{venue: ex, metric: AroundVenueMetricPrice, at: v.MoveStart.UTC(), res: step})
	}
	if !v.OIStart.IsZero() && aroundVenueHeldToPrice(v.OIUntil) {
		out = append(out, aroundVenueClock{venue: ex, metric: AroundVenueMetricOI, at: v.OIStart.UTC(), res: aroundVenueOIStep})
	}
	if !v.TakerStart.IsZero() && aroundVenueHeldToPrice(v.TakerUntil) {
		out = append(out, aroundVenueClock{venue: ex, metric: AroundVenueMetricTakers, at: v.TakerStart.UTC(), res: step})
	}
	return out
}

func aroundClockStamp(c aroundVenueClock) AroundStamp {
	return AroundStamp{At: c.at, Resolution: c.res}
}

func aroundTimingFirstSignal(bn, by AroundVenue) (venue, metric string, at time.Time) {
	clocks := append(aroundVenueClocks(bn), aroundVenueClocks(by)...)
	var best aroundVenueClock
	for _, c := range clocks {
		if c.at.IsZero() {
			continue
		}
		if best.at.IsZero() || c.at.Before(best.at) {
			best = c
		}
	}
	if best.at.IsZero() {
		return "", "", time.Time{}
	}
	for _, c := range clocks {
		if c.at.IsZero() || (c.venue == best.venue && c.metric == best.metric && c.at.Equal(best.at)) {
			continue
		}
		if AroundStampsOverlap(aroundClockStamp(best), aroundClockStamp(c)) {
			return "", "", time.Time{}
		}
	}
	return best.venue, best.metric, best.at
}

func aroundVenueFirstSignalSentence(bn, by AroundVenue, priceLead time.Time) string {
	venue, metric, at := aroundTimingFirstSignal(bn, by)
	if at.IsZero() || metric == "" || metric == AroundVenueMetricPrice {
		return ""
	}
	label := aroundExchangeLabel(Exchange(venue))
	what := "open interest started rising"
	if metric == AroundVenueMetricOI {
		dir := bn.OIDir
		if venue == string(ExchangeBybit) {
			dir = by.OIDir
		}
		if dir == CVDDirDown {
			what = "open interest started falling"
		}
	} else if metric == AroundVenueMetricTakers {
		dir := bn.TakerDir
		if venue == string(ExchangeBybit) {
			dir = by.TakerDir
		}
		what = "takers were already buying"
		if dir == TakerSideSell {
			what = "takers were already selling"
		}
	}
	sig := AroundStamp{At: at.UTC(), Resolution: aroundVenueSignalResolution(venue, metric, bn, by)}
	px := AroundStamp{At: priceLead.UTC(), Resolution: aroundVenuePriceResolution(priceLead, bn, by)}
	if priceLead.IsZero() || !AroundStampBefore(sig, px) {
		if !priceLead.IsZero() && AroundStampsOverlap(sig, px) {
			return ""
		}
		return fmt.Sprintf("%s %s from %s UTC and kept going.", label, what, at.UTC().Format("15:04"))
	}
	lead := formatOptionsLead(priceLead.Sub(at))
	priceVenue := aroundExchangeLabel(ExchangeBinance)
	if !by.MoveStart.IsZero() && by.MoveStart.Equal(priceLead) && (bn.MoveStart.IsZero() || !bn.MoveStart.Equal(priceLead)) {
		priceVenue = aroundExchangeLabel(ExchangeBybit)
	}
	return fmt.Sprintf("%s %s %s before %s price and kept going.", label, what, lead, priceVenue)
}

func appendAroundVenueFirstSignal(out *AroundVenueTiming, bn, by AroundVenue, priceLead time.Time) {
	if out == nil {
		return
	}
	s := aroundVenueFirstSignalSentence(bn, by, priceLead)
	if s == "" {
		return
	}
	if out.Summary == "" {
		out.Summary = s
		return
	}
	out.Summary = strings.TrimSpace(out.Summary) + " " + s
}

func aroundVenueSignalResolution(venue, metric string, bn, by AroundVenue) time.Duration {
	if metric == AroundVenueMetricOI {
		return aroundVenueOIStep
	}
	v := bn
	if venue == string(ExchangeBybit) {
		v = by
	}
	return aroundIntervalOrMinute(v.Interval)
}

func aroundVenuePriceResolution(priceLead time.Time, bn, by AroundVenue) time.Duration {
	if !by.MoveStart.IsZero() && by.MoveStart.Equal(priceLead) && (bn.MoveStart.IsZero() || !bn.MoveStart.Equal(priceLead)) {
		return aroundIntervalOrMinute(by.Interval)
	}
	return aroundIntervalOrMinute(bn.Interval)
}

func aroundExchangeLabel(ex Exchange) string {
	switch ex {
	case ExchangeBinance:
		return "Binance"
	case ExchangeBybit:
		return "Bybit"
	default:
		s := string(ex)
		if s == "" {
			return "listed"
		}
		return s
	}
}

func appendAroundVenueBeforeSummary(out *AroundVenueTiming, laggard string, until time.Time) {
	if out == nil || len(out.Before) == 0 {
		return
	}
	parts := make([]string, 0, len(out.Before))
	for _, b := range out.Before {
		if b.Summary != "" {
			parts = append(parts, b.Summary)
		}
	}
	if len(parts) == 0 {
		return
	}
	when := ""
	if laggard != "" && !until.IsZero() {
		when = fmt.Sprintf("Before %s printed at %s UTC, ", laggard, until.UTC().Format("15:04"))
	} else if laggard == "" && until.IsZero() {
		when = "In that window, "
	}
	out.Summary = strings.TrimSpace(out.Summary) + " " + when + joinAroundVenueBits(parts) + "."
}

func joinAroundVenueBits(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}
