package domain

import (
	"fmt"
	"math"
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

	aroundVenueLookback   = 30 * time.Minute
	aroundVenueSameWindow = 2 * time.Minute
	aroundVenueMinMove    = 0.20
	aroundVenueOIMin      = 0.30
	aroundVenueTakerTail  = 5 * time.Minute
	// aroundVenueUntilFresh is how close the last OI sample or known taker
	// bar must be to the price start before we will say continued / faded /
	// reversed. Wider than one futures-history tick (5m) so a current print
	// still counts; a longer hole means the path to price is unknown.
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
	if phaseFrom.Equal(plan.DuringFrom) {
		if s := aroundVenueLookbackStart(bars, plan, dir, thresh); !s.IsZero() {
			return s
		}
	}
	run := FilterAroundBars(bars, phaseFrom, searchTo)
	if t := firstAroundRunBar(run, pre.Close, dir, thresh); !t.IsZero() {
		return t
	}
	if t := firstAroundRunBar(run, pre.Close, dir, 0); !t.IsZero() {
		return t
	}
	if len(run) > 0 {
		return run[0].Time.UTC()
	}
	return time.Time{}
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
	if t := firstAroundTakerLean(FilterAroundBars(bars, searchFrom, searchTo), want); !t.IsZero() {
		return t, want
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
	return hit, dir
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
// when the last known bar is the start print, is too far before price, or
// the last minutes before price have no tape — do not guess the path.
func MarkAroundTakerUntil(bars []AroundBar, start time.Time, dir string, until time.Time) string {
	if start.IsZero() || until.IsZero() || !until.After(start) || (dir != TakerSideBuy && dir != TakerSideSell) {
		return ""
	}
	last, ok := lastAroundKnownTakerBar(bars, until)
	if !ok || !aroundVenueUntilFreshEnough(last.Time, start, until) {
		return ""
	}
	tailFrom := until.Add(-aroundVenueTakerTail)
	if tailFrom.Before(start) {
		tailFrom = start
	}
	tailBars := aroundBarsThrough(bars, tailFrom, until)
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

func lastAroundKnownTakerBar(bars []AroundBar, until time.Time) (AroundBar, bool) {
	var out AroundBar
	ok := false
	for _, b := range bars {
		if !b.BuySellKnown || b.Time.IsZero() || b.Time.After(until) {
			continue
		}
		if !ok || b.Time.After(out.Time) {
			out = b
			ok = true
		}
	}
	return out, ok
}

func aroundBarsThrough(bars []AroundBar, from, to time.Time) []AroundBar {
	if len(bars) == 0 || to.Before(from) {
		return nil
	}
	out := make([]AroundBar, 0, len(bars))
	for _, b := range bars {
		if b.Time.Before(from) || b.Time.After(to) {
			continue
		}
		out = append(out, b)
	}
	return out
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
	if delta <= aroundVenueSameWindow {
		out.Agreement = AroundVenueSame
		out.Follow = AroundVenueFollowTogether
		out.Before = aroundVenueBeforeBits(bnVen, byVen, time.Time{})
		out.Summary = "Binance and Bybit started the price move at about the same time."
		appendAroundVenueFirstSignal(out, bnVen, byVen, bn)
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
}

func aroundVenueClocks(v AroundVenue) []aroundVenueClock {
	if v.Exchange == "" {
		return nil
	}
	ex := string(v.Exchange)
	var out []aroundVenueClock
	if !v.MoveStart.IsZero() {
		out = append(out, aroundVenueClock{venue: ex, metric: AroundVenueMetricPrice, at: v.MoveStart.UTC()})
	}
	if !v.OIStart.IsZero() && aroundVenueHeldToPrice(v.OIUntil) {
		out = append(out, aroundVenueClock{venue: ex, metric: AroundVenueMetricOI, at: v.OIStart.UTC()})
	}
	if !v.TakerStart.IsZero() && aroundVenueHeldToPrice(v.TakerUntil) {
		out = append(out, aroundVenueClock{venue: ex, metric: AroundVenueMetricTakers, at: v.TakerStart.UTC()})
	}
	return out
}

func aroundTimingFirstSignal(bn, by AroundVenue) (venue, metric string, at time.Time) {
	var best aroundVenueClock
	for _, c := range append(aroundVenueClocks(bn), aroundVenueClocks(by)...) {
		if c.at.IsZero() {
			continue
		}
		if best.at.IsZero() || c.at.Before(best.at) {
			best = c
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
	if priceLead.IsZero() || !at.Before(priceLead) {
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
