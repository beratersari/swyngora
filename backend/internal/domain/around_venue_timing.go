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

	AroundVenueMetricOI     = "oi"
	AroundVenueMetricTakers = "takers"

	aroundVenueLookback   = 30 * time.Minute
	aroundVenueSameWindow = 2 * time.Minute
	aroundVenueMinMove    = 0.20
	aroundVenueOIMin      = 0.30
)

// AroundVenueTiming compares when Binance and Bybit started the price move.
type AroundVenueTiming struct {
	Agreement    string
	Follow       string
	Leader       string
	BinanceStart time.Time
	BybitStart   time.Time
	Lead         time.Duration
	LeadLabel    string
	Before       []AroundVenueBeforeBit
	Summary      string
}

// AroundVenueBeforeBit is OI or taker flow that printed before the laggard.
type AroundVenueBeforeBit struct {
	Metric    string
	Venue     string
	Direction string
	Phase     string
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

// CompareAroundVenueTiming says whether Binance and Bybit started together,
// one printed first and the other followed, or the move stayed on one venue.
// The leader is whoever started first — not the larger move.
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
	}
	if byImp.known && byImp.dir == "" {
		by = time.Time{}
	}

	if bn.IsZero() && by.IsZero() {
		return nil
	}
	out := &AroundVenueTiming{}
	if !bn.IsZero() {
		out.BinanceStart = bn.UTC()
	}
	if !by.IsZero() {
		out.BybitStart = by.UTC()
	}

	switch {
	case !bn.IsZero() && by.IsZero():
		out.Agreement = AroundVenueBinanceOnly
		out.Follow = AroundVenueFollowStayed
		out.Leader = string(ExchangeBinance)
		out.Before = aroundVenueBeforeBits(bnVen, byVen, time.Time{})
		out.Summary = "The price move printed on Binance. Bybit did not follow on price."
		appendAroundVenueBeforeSummary(out, "Bybit")
		return out
	case bn.IsZero() && !by.IsZero():
		out.Agreement = AroundVenueBybitOnly
		out.Follow = AroundVenueFollowStayed
		out.Leader = string(ExchangeBybit)
		out.Before = aroundVenueBeforeBits(byVen, bnVen, time.Time{})
		out.Summary = "The price move printed on Bybit. Binance did not follow on price."
		appendAroundVenueBeforeSummary(out, "Binance")
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
		appendAroundVenueBeforeSummary(out, "")
		return out
	}
	out.LeadLabel = formatOptionsLead(delta)
	sameWay := true
	if bnImp.known && byImp.known && bnImp.dir != "" && byImp.dir != "" {
		sameWay = bnImp.dir == byImp.dir
	}
	if bn.Before(by) {
		out.Leader = string(ExchangeBinance)
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
		appendAroundVenueBeforeSummary(out, "Bybit")
		return out
	}
	out.Leader = string(ExchangeBybit)
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
	appendAroundVenueBeforeSummary(out, "Binance")
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
	if v.Exchange == "" && len(v.Phases) == 0 {
		return nil
	}
	order := []string{AroundPhaseBefore, AroundPhaseDuring}
	if !until.IsZero() {
		order = append(order, AroundPhaseAfter)
	}
	var oi, tk AroundVenueBeforeBit
	for _, id := range order {
		p, ok := AroundPhaseByID(v, id)
		if !ok || !p.Complete {
			continue
		}
		if id == AroundPhaseAfter && (until.IsZero() || !p.From.Before(until)) {
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

func appendAroundVenueBeforeSummary(out *AroundVenueTiming, laggard string) {
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
	when := "In that window, "
	if laggard != "" {
		when = fmt.Sprintf("Before %s printed, ", laggard)
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
