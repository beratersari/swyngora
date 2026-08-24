package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	AroundPhaseBefore = "before"
	AroundPhaseDuring = "during"
	AroundPhaseAfter  = "after"

	AroundWindow15m = "15m"
	AroundWindow30m = "30m"
	AroundWindow1h  = "1h"
	AroundWindow2h  = "2h"
	AroundWindow4h  = "4h"

	AroundDuring5m  = "5m"
	AroundDuring15m = "15m"
	AroundDuring30m = "30m"
	AroundDuring1h  = "1h"

	DefaultAroundWindow = AroundWindow1h
	DefaultAroundDuring = AroundDuring15m
	MaxAroundLookback   = 30 * 24 * time.Hour
	AroundTypicalPriors = 8

	AroundPathContinued = "continued"
	AroundPathReversed  = "reversed"
	AroundPathFaded     = "faded"
	AroundPathChopped   = "chopped"

	AroundEventSweep      = "sweep"
	AroundEventAbsorption = "absorption"
)

// AroundWindows are accepted before/after lookbacks.
var AroundWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{AroundWindow15m, 15 * time.Minute},
	{AroundWindow30m, 30 * time.Minute},
	{AroundWindow1h, time.Hour},
	{AroundWindow2h, 2 * time.Hour},
	{AroundWindow4h, 4 * time.Hour},
}

// AroundDurings are accepted core-move lengths.
var AroundDurings = []struct {
	ID  string
	Dur time.Duration
}{
	{AroundDuring5m, 5 * time.Minute},
	{AroundDuring15m, 15 * time.Minute},
	{AroundDuring30m, 30 * time.Minute},
	{AroundDuring1h, time.Hour},
}

// AroundBar is one candle flattened for a before/during/after read.
type AroundBar struct {
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

// AroundPlan is the three adjacent windows around an event time.
//
//	before: [at − window, at)
//	during: [at, at + during)
//	after:  [at + during, at + during + window)
type AroundPlan struct {
	At         time.Time
	Window     string
	WindowDur  time.Duration
	During     string
	DuringDur  time.Duration
	BeforeFrom time.Time
	BeforeTo   time.Time
	DuringFrom time.Time
	DuringTo   time.Time
	AfterFrom  time.Time
	AfterTo    time.Time
	From       time.Time
	To         time.Time
	Clipped    bool
}

// AroundPrice is OHLC and net move of one phase.
type AroundPrice struct {
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Change    float64
	ChangePct float64
	Range     float64
	RangePct  float64
	Direction string
}

// AroundFlow is volume, taker split, VWAP, and vs-typical for one phase.
type AroundFlow struct {
	Volume        float64
	BuyVolume     float64
	SellVolume    float64
	Delta         float64
	BuyShare      float64
	Dominant      string
	BuySellKnown  bool
	VWAP          float64
	VsVWAP        string
	DistancePct   float64
	Typical       float64
	VolumeRatio   float64
	VolumeGrade   string
	TypicalSample int
	TypicalKnown  bool
}

// AroundProfile is the compact volume-profile read of one phase (no bins).
type AroundProfile struct {
	POC           float64
	POCVolume     float64
	ValueAreaLow  float64
	ValueAreaHigh float64
	LastVsArea    string
}

// AroundBook is how the stored spot book changed across a phase.
type AroundBook struct {
	FromMid          float64
	ToMid            float64
	MidDelta         float64
	MidDeltaPct      float64
	BidNotionalDelta float64
	AskNotionalDelta float64
	ImbalanceDelta   float64
	WallsAdded       int
	WallsRemoved     int
	Summary          string
	Complete         bool
}

// AroundFutures is OI / funding / long% / liquidations across a phase.
type AroundFutures struct {
	OIFrom      float64
	OITo        float64
	OIChange    float64
	OIChangePct float64
	OIDirection string
	FundingFrom float64
	FundingTo   float64
	LongPctFrom float64
	LongPctTo   float64
	LongLiq     float64
	ShortLiq    float64
	Complete    bool
}

// AroundEvent is a sweep or absorption that landed in a phase.
type AroundEvent struct {
	Kind    string
	Phase   string
	Side    string
	Title   string
	Summary string
	At      time.Time
	Level   float64
	Score   int
}

// AroundPhase is one of before / during / after.
type AroundPhase struct {
	Phase    string
	From     time.Time
	To       time.Time
	BarCount int
	Price    AroundPrice
	Flow     AroundFlow
	Profile  *AroundProfile
	Book     *AroundBook
	Futures  *AroundFutures
	Events   []AroundEvent
	Summary  string
	Complete bool
}

// AroundChange is how one metric moved across the three phases.
type AroundChange struct {
	Metric    string
	Before    float64
	During    float64
	After     float64
	Path      string
	Direction string
	Summary   string
}

// AroundVenue is one exchange's (or combined) around-the-move read.
type AroundVenue struct {
	Exchange Exchange
	Symbol   string
	Interval string
	Phases   []AroundPhase
	Changes  []AroundChange
	Events   []AroundEvent
	Summary  string
	Error    string
}

// AroundReport is the API result.
type AroundReport struct {
	Symbol   string
	Exchange string
	At       time.Time
	Window   string
	During   string
	From     time.Time
	To       time.Time
	AsOf     time.Time
	Clipped  bool
	Venues   []AroundVenue
	Combined *AroundVenue
	Summary  string
	Note     string
}

// ParseAroundWindow accepts 15m / 30m / 1h / 2h / 4h (empty = 1h).
func ParseAroundWindow(raw string) (string, time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = DefaultAroundWindow
	}
	for _, w := range AroundWindows {
		if w.ID == s {
			return w.ID, w.Dur, nil
		}
	}
	return "", 0, fmt.Errorf("%w: window must be 15m, 30m, 1h, 2h, or 4h", ErrInvalidArgument)
}

// ParseAroundDuring accepts 5m / 15m / 30m / 1h (empty = 15m).
func ParseAroundDuring(raw string) (string, time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = DefaultAroundDuring
	}
	for _, w := range AroundDurings {
		if w.ID == s {
			return w.ID, w.Dur, nil
		}
	}
	return "", 0, fmt.Errorf("%w: during must be 5m, 15m, 30m, or 1h", ErrInvalidArgument)
}

// ResolveAroundPlan builds the three adjacent windows around at.
// after.To is clipped to now when the look-forward is still in the future.
func ResolveAroundPlan(window, during string, at, now time.Time) (AroundPlan, error) {
	if at.IsZero() {
		return AroundPlan{}, fmt.Errorf("%w: at time is required", ErrInvalidArgument)
	}
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	at = at.UTC()
	if at.After(now) {
		return AroundPlan{}, fmt.Errorf("%w: at time is in the future", ErrInvalidArgument)
	}
	if now.Sub(at) > MaxAroundLookback {
		return AroundPlan{}, fmt.Errorf("%w: at time must be within the last 30 days", ErrInvalidArgument)
	}
	winID, winDur, err := ParseAroundWindow(window)
	if err != nil {
		return AroundPlan{}, err
	}
	durID, durDur, err := ParseAroundDuring(during)
	if err != nil {
		return AroundPlan{}, err
	}
	p := AroundPlan{
		At: at, Window: winID, WindowDur: winDur, During: durID, DuringDur: durDur,
		BeforeFrom: at.Add(-winDur), BeforeTo: at,
		DuringFrom: at, DuringTo: at.Add(durDur),
		AfterFrom: at.Add(durDur), AfterTo: at.Add(durDur + winDur),
	}
	if p.DuringTo.After(now) {
		p.DuringTo = now
		p.Clipped = true
	}
	if p.AfterFrom.After(now) {
		p.AfterFrom = now
		p.AfterTo = now
		p.Clipped = true
	} else if p.AfterTo.After(now) {
		p.AfterTo = now
		p.Clipped = true
	}
	p.From = p.BeforeFrom
	p.To = p.AfterTo
	if p.To.Before(p.DuringTo) {
		p.To = p.DuringTo
	}
	return p, nil
}

// AroundPhaseSpans returns before, during, after in that order.
func AroundPhaseSpans(p AroundPlan) []struct {
	ID       string
	From, To time.Time
} {
	return []struct {
		ID       string
		From, To time.Time
	}{
		{AroundPhaseBefore, p.BeforeFrom, p.BeforeTo},
		{AroundPhaseDuring, p.DuringFrom, p.DuringTo},
		{AroundPhaseAfter, p.AfterFrom, p.AfterTo},
	}
}

// AroundBarsFromCandles maps klines (quote volume + optional taker-buy).
func AroundBarsFromCandles(candles []Candle) []AroundBar {
	out := make([]AroundBar, 0, len(candles))
	for _, c := range candles {
		if c.OpenTime.IsZero() {
			continue
		}
		high, err1 := parseFloat(c.High)
		low, err2 := parseFloat(c.Low)
		if err1 != nil || err2 != nil || high <= 0 || low <= 0 {
			continue
		}
		if low > high {
			low, high = high, low
		}
		quote, err := parseFloat(c.QuoteVolume)
		if err != nil || quote < 0 {
			continue
		}
		bar := AroundBar{Time: c.OpenTime.UTC(), High: high, Low: low, Volume: quote}
		if o, err := parseFloat(c.Open); err == nil && o > 0 {
			bar.Open = o
		}
		if cl, err := parseFloat(c.Close); err == nil && cl > 0 {
			bar.Close = cl
		}
		if bar.Open <= 0 {
			bar.Open = bar.Close
		}
		if c.TakerBuyQuote != "" {
			buy, err := parseFloat(c.TakerBuyQuote)
			if err == nil && buy >= 0 {
				if quote > 0 && buy > quote {
					buy = quote
				}
				bar.BuyVolume = buy
				bar.SellVolume = quote - buy
				bar.BuySellKnown = true
			}
		}
		out = append(out, bar)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// FilterAroundBars keeps bars whose open is in [from, to).
func FilterAroundBars(bars []AroundBar, from, to time.Time) []AroundBar {
	if len(bars) == 0 || !to.After(from) {
		return nil
	}
	out := make([]AroundBar, 0, len(bars))
	for _, b := range bars {
		if b.Time.Before(from) || !b.Time.Before(to) {
			continue
		}
		out = append(out, b)
	}
	return out
}

// BuildAroundPhase aggregates one window of bars and compares volume to typical.
func BuildAroundPhase(phase string, bars, all []AroundBar, from, to time.Time) AroundPhase {
	out := AroundPhase{Phase: phase, From: from.UTC(), To: to.UTC(), Events: []AroundEvent{}}
	if !to.After(from) {
		out.Summary = phase + ": window has not started yet."
		return out
	}
	slice := FilterAroundBars(bars, from, to)
	if len(slice) == 0 {
		out.Summary = phase + ": no candles in this window."
		return out
	}
	out.BarCount = len(slice)
	out.Complete = true
	out.Price = aroundPriceOf(slice)
	out.Flow = aroundFlowOf(slice)
	typ, n, ok := typicalAroundVolume(all, from, to)
	if ok {
		out.Flow.Typical = typ
		out.Flow.TypicalSample = n
		out.Flow.TypicalKnown = true
		if typ > 0 {
			out.Flow.VolumeRatio = out.Flow.Volume / typ
		}
		out.Flow.VolumeGrade = VolumeSurgeGrade(out.Flow.VolumeRatio)
	}
	if prof := aroundProfileOf(slice, from, to); prof != nil {
		out.Profile = prof
	}
	if ev := aroundAbsorptionEvent(phase, out); ev != nil {
		out.Events = append(out.Events, *ev)
	}
	out.Summary = ExplainAroundPhase(out)
	return out
}

// BuildAroundVenue fills the three phases, path changes, and a short read.
func BuildAroundVenue(ex Exchange, symbol string, bars []AroundBar, plan AroundPlan, interval CandleInterval) AroundVenue {
	out := AroundVenue{
		Exchange: ex, Symbol: symbol, Interval: string(interval),
		Phases: []AroundPhase{}, Changes: []AroundChange{}, Events: []AroundEvent{},
	}
	if len(bars) == 0 {
		out.Error = "not enough candles around that time"
		out.Summary = out.Error
		return out
	}
	for _, sp := range AroundPhaseSpans(plan) {
		ph := BuildAroundPhase(sp.ID, bars, bars, sp.From, sp.To)
		out.Phases = append(out.Phases, ph)
		out.Events = append(out.Events, ph.Events...)
	}
	out.Changes = CompareAroundPhases(out.Phases)
	out.Summary = ExplainAroundVenue(out)
	return out
}

// AttachAroundEvents adds sweep (or other) events into the matching phase.
func AttachAroundEvents(v *AroundVenue, extra []AroundEvent) {
	if v == nil || len(extra) == 0 {
		return
	}
	for _, ev := range extra {
		placed := false
		for i := range v.Phases {
			if ev.At.IsZero() {
				continue
			}
			if !ev.At.Before(v.Phases[i].From) && ev.At.Before(v.Phases[i].To) {
				ev.Phase = v.Phases[i].Phase
				v.Phases[i].Events = append(v.Phases[i].Events, ev)
				v.Phases[i].Summary = ExplainAroundPhase(v.Phases[i])
				placed = true
				break
			}
		}
		if !placed {
			ev.Phase = firstNonEmpty(ev.Phase, AroundPhaseDuring)
		}
		v.Events = append(v.Events, ev)
	}
	v.Summary = ExplainAroundVenue(*v)
}

// CompareAroundPhases writes price / volume / delta paths across the three windows.
func CompareAroundPhases(phases []AroundPhase) []AroundChange {
	var before, during, after AroundPhase
	for _, p := range phases {
		switch p.Phase {
		case AroundPhaseBefore:
			before = p
		case AroundPhaseDuring:
			during = p
		case AroundPhaseAfter:
			after = p
		}
	}
	out := make([]AroundChange, 0, 3)
	if during.Complete || before.Complete || after.Complete {
		path := aroundPricePath(before.Price.Direction, during.Price.Direction, after.Price.Direction)
		out = append(out, AroundChange{
			Metric: "price", Before: before.Price.ChangePct, During: during.Price.ChangePct,
			After: after.Price.ChangePct, Path: path, Direction: during.Price.Direction,
			Summary: aroundPriceChangeSummary(before, during, after, path),
		})
	}
	if before.Flow.Volume > 0 || during.Flow.Volume > 0 || after.Flow.Volume > 0 {
		path := aroundVolumePath(before.Flow.Volume, during.Flow.Volume, after.Flow.Volume)
		out = append(out, AroundChange{
			Metric: "volume", Before: before.Flow.Volume, During: during.Flow.Volume,
			After: after.Flow.Volume, Path: path,
			Summary: aroundVolumeChangeSummary(before, during, after, path),
		})
	}
	if before.Flow.BuySellKnown || during.Flow.BuySellKnown || after.Flow.BuySellKnown {
		path := aroundDeltaPath(before.Flow.Dominant, during.Flow.Dominant, after.Flow.Dominant)
		out = append(out, AroundChange{
			Metric: "delta", Before: before.Flow.Delta, During: during.Flow.Delta,
			After: after.Flow.Delta, Path: path, Direction: during.Flow.Dominant,
			Summary: aroundDeltaChangeSummary(before, during, after, path),
		})
	}
	if bf, df, af := phaseFutures(before), phaseFutures(during), phaseFutures(after); bf || df || af {
		path := aroundOIPath(before, during, after)
		out = append(out, AroundChange{
			Metric: "oi",
			Before: futuresOIChange(before), During: futuresOIChange(during), After: futuresOIChange(after),
			Path: path, Summary: aroundOIChangeSummary(before, during, after, path),
		})
	}
	return out
}

// CombineAroundVenues adds quote volume across venues and volume-weights price/VWAP.
func CombineAroundVenues(symbol string, venues []AroundVenue, plan AroundPlan, interval CandleInterval) *AroundVenue {
	out := &AroundVenue{
		Exchange: "all", Symbol: symbol, Interval: string(interval),
		Phases: []AroundPhase{}, Changes: []AroundChange{}, Events: []AroundEvent{},
	}
	usable := 0
	for _, v := range venues {
		if v.Error == "" && len(v.Phases) > 0 {
			usable++
		}
	}
	if usable == 0 {
		out.Error = "not enough candles around that time"
		out.Summary = out.Error
		return out
	}
	for _, sp := range AroundPhaseSpans(plan) {
		var parts []AroundPhase
		for _, v := range venues {
			if v.Error != "" {
				continue
			}
			for _, p := range v.Phases {
				if p.Phase == sp.ID {
					parts = append(parts, p)
				}
			}
		}
		out.Phases = append(out.Phases, combineAroundPhases(sp.ID, parts, sp.From, sp.To))
	}
	for _, v := range venues {
		out.Events = append(out.Events, v.Events...)
	}
	out.Changes = CompareAroundPhases(out.Phases)
	out.Summary = ExplainAroundVenue(*out)
	return out
}

// AroundSweepsToEvents keeps sweeps whose poke sits inside [from, to).
func AroundSweepsToEvents(sweeps []LiquiditySweep, from, to time.Time) []AroundEvent {
	out := make([]AroundEvent, 0, len(sweeps))
	for _, s := range sweeps {
		at := s.PiercedAt
		if at.IsZero() || at.Before(from) || !at.Before(to) {
			continue
		}
		out = append(out, AroundEvent{
			Kind: AroundEventSweep, Side: s.Side, Title: s.Title, Summary: s.Summary,
			At: at, Level: s.Level,
		})
	}
	return out
}

// ExplainAroundPhase is one window in plain language.
func ExplainAroundPhase(p AroundPhase) string {
	if p.Summary != "" && !p.Complete && p.Price.Open == 0 {
		return p.Summary
	}
	if !p.Complete {
		if p.Summary != "" {
			return p.Summary
		}
		return p.Phase + ": incomplete."
	}
	head := fmt.Sprintf("%s: price %s%% (%s → %s), volume %s",
		p.Phase, FormatSignedPct(p.Price.ChangePct),
		formatQty(p.Price.Open), formatQty(p.Price.Close), formatQty(p.Flow.Volume))
	if p.Flow.TypicalKnown && p.Flow.Typical > 0 {
		head += fmt.Sprintf(" (%.1fx typical, %s)", p.Flow.VolumeRatio, p.Flow.VolumeGrade)
	}
	if p.Flow.BuySellKnown && p.Flow.Dominant != "" && p.Flow.Dominant != TakerSideEven {
		head += ", takers " + p.Flow.Dominant
	}
	if p.Flow.VWAP > 0 && p.Flow.VsVWAP != "" && p.Flow.VsVWAP != VolumeProfileVsUnknown {
		head += ", close " + p.Flow.VsVWAP + " VWAP"
	}
	if p.Profile != nil && p.Profile.POC > 0 {
		head += fmt.Sprintf(", POC %s", formatQty(p.Profile.POC))
	}
	if f := p.Futures; f != nil && f.Complete {
		head += fmt.Sprintf(", OI %s%%", FormatSignedPct(f.OIChangePct))
	}
	for _, ev := range p.Events {
		if ev.Kind == AroundEventSweep {
			head += "; sweep " + ev.Side
			break
		}
	}
	for _, ev := range p.Events {
		if ev.Kind == AroundEventAbsorption && ev.Side != "" {
			head += "; " + ev.Side + " absorbing"
			break
		}
	}
	return head + "."
}

// ExplainAroundVenue prefers the during move, then what happened after.
func ExplainAroundVenue(v AroundVenue) string {
	if v.Error != "" {
		return v.Error
	}
	name := prettyBase(v.Symbol)
	var during, after, before AroundPhase
	for i := range v.Phases {
		switch v.Phases[i].Phase {
		case AroundPhaseBefore:
			before = v.Phases[i]
		case AroundPhaseDuring:
			during = v.Phases[i]
		case AroundPhaseAfter:
			after = v.Phases[i]
		}
	}
	if !during.Complete && !before.Complete && !after.Complete {
		return name + ": not enough candles around that time."
	}
	head := name
	if during.Complete {
		head += fmt.Sprintf(" moved %s%% during the window", FormatSignedPct(during.Price.ChangePct))
		if during.Flow.TypicalKnown && during.Flow.VolumeRatio >= 1.5 {
			head += fmt.Sprintf(" on %.1fx typical volume", during.Flow.VolumeRatio)
		}
		if during.Flow.BuySellKnown && during.Flow.Dominant != "" && during.Flow.Dominant != TakerSideEven {
			head += " (" + during.Flow.Dominant + "-led)"
		}
		head += "."
	} else if before.Complete {
		head += fmt.Sprintf(" before: %s%%.", FormatSignedPct(before.Price.ChangePct))
	}
	for _, ch := range v.Changes {
		if ch.Metric == "price" && ch.Path != "" && ch.Summary != "" {
			head += " " + ch.Summary
			break
		}
	}
	if n := countAroundKind(v.Events, AroundEventSweep); n > 0 {
		head += fmt.Sprintf(" %d liquidity sweep(s) in the span.", n)
	}
	return strings.TrimSpace(head)
}

// ExplainAroundReport prefers combined, then the first complete venue.
func ExplainAroundReport(r AroundReport) string {
	if r.Combined != nil && r.Combined.Summary != "" && r.Combined.Error == "" {
		return "Combined: " + r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Summary != "" && v.Error == "" {
			return string(v.Exchange) + ": " + v.Summary
		}
	}
	for _, v := range r.Venues {
		if v.Summary != "" {
			return v.Summary
		}
	}
	return "No tape around that time."
}

func aroundPriceOf(bars []AroundBar) AroundPrice {
	out := AroundPrice{Direction: CVDDirFlat}
	if len(bars) == 0 {
		return out
	}
	out.Open = bars[0].Open
	if out.Open <= 0 {
		out.Open = bars[0].Close
	}
	out.Close = bars[len(bars)-1].Close
	if out.Close <= 0 {
		out.Close = bars[len(bars)-1].Open
	}
	for _, b := range bars {
		if b.High > out.High {
			out.High = b.High
		}
		if out.Low == 0 || (b.Low > 0 && b.Low < out.Low) {
			out.Low = b.Low
		}
	}
	out.Change = out.Close - out.Open
	if out.Open > 0 {
		out.ChangePct = out.Change / out.Open * 100
		out.Range = out.High - out.Low
		if out.Range < 0 {
			out.Range = 0
		}
		out.RangePct = out.Range / out.Open * 100
	}
	out.Direction = changeDir(out.ChangePct)
	return out
}

func aroundFlowOf(bars []AroundBar) AroundFlow {
	out := AroundFlow{Dominant: TakerSideEven, VsVWAP: VolumeProfileVsUnknown}
	var pv float64
	for _, b := range bars {
		out.Volume += b.Volume
		out.BuyVolume += b.BuyVolume
		out.SellVolume += b.SellVolume
		if b.BuySellKnown {
			out.BuySellKnown = true
		}
		tp := TypicalPrice(b.High, b.Low, b.Close)
		if tp > 0 && b.Volume > 0 {
			pv += tp * b.Volume
		}
	}
	out.Delta = out.BuyVolume - out.SellVolume
	if out.BuyVolume+out.SellVolume > 0 {
		out.BuyShare = out.BuyVolume / (out.BuyVolume + out.SellVolume)
	}
	if out.BuySellKnown {
		out.Dominant = TakerDominant(out.BuyVolume, out.SellVolume)
	}
	if out.Volume > 0 && pv > 0 {
		out.VWAP = pv / out.Volume
		last := bars[len(bars)-1].Close
		if last <= 0 {
			last = bars[len(bars)-1].Open
		}
		if last > 0 && out.VWAP > 0 {
			out.DistancePct = (last - out.VWAP) / out.VWAP * 100
			switch {
			case out.DistancePct > 0.02:
				out.VsVWAP = VolumeProfileVsAbove
			case out.DistancePct < -0.02:
				out.VsVWAP = VolumeProfileVsBelow
			default:
				out.VsVWAP = "at"
			}
		}
	}
	return out
}

func typicalAroundVolume(all []AroundBar, from, to time.Time) (typical float64, n int, ok bool) {
	dur := to.Sub(from)
	if dur <= 0 || len(all) == 0 {
		return 0, 0, false
	}
	vols := make([]float64, 0, AroundTypicalPriors)
	end := from
	for i := 0; i < AroundTypicalPriors; i++ {
		start := end.Add(-dur)
		var sum float64
		var any bool
		for _, b := range all {
			if b.Time.Before(start) || !b.Time.Before(end) {
				continue
			}
			sum += b.Volume
			any = true
		}
		if any {
			vols = append(vols, sum)
		}
		end = start
	}
	if len(vols) < 3 {
		return 0, len(vols), false
	}
	typ := medianFloat(vols)
	return typ, len(vols), typ > 0
}

func aroundProfileOf(bars []AroundBar, from, to time.Time) *AroundProfile {
	vp := make([]VolumeProfileBar, 0, len(bars))
	var last, lo, hi float64
	for _, b := range bars {
		if b.High <= 0 || b.Low <= 0 || b.Volume <= 0 {
			continue
		}
		row := VolumeProfileBar{
			Time: b.Time, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume,
			BuyVolume: b.BuyVolume, SellVolume: b.SellVolume, BuySellKnown: b.BuySellKnown,
		}
		vp = append(vp, row)
		if last == 0 && b.Close > 0 {
			last = b.Close
		}
		if lo == 0 || b.Low < lo {
			lo = b.Low
		}
		if b.High > hi {
			hi = b.High
		}
	}
	if last == 0 && len(bars) > 0 {
		last = bars[len(bars)-1].Close
	}
	tick := AutoTickSize(lo, hi)
	if tick <= 0 || len(vp) == 0 {
		return nil
	}
	got := BuildVolumeProfile("", "", vp, tick, last, from, to, Interval1m)
	if got.Error != "" || got.POC.Price <= 0 {
		return nil
	}
	return &AroundProfile{
		POC: got.POC.Price, POCVolume: got.POC.Volume,
		ValueAreaLow: got.ValueArea.Low, ValueAreaHigh: got.ValueArea.High,
		LastVsArea: got.LastVsArea,
	}
}

func aroundAbsorptionEvent(phase string, p AroundPhase) *AroundEvent {
	if !p.Flow.BuySellKnown || p.Flow.Volume <= 0 {
		return nil
	}
	flat := aroundFlatPct(p.To.Sub(p.From))
	got := ClassifyAbsorption(p.Flow.BuyVolume, p.Flow.SellVolume, p.Price.ChangePct, flat)
	if got.Kind == AbsorptionKindNone || got.Score < 40 {
		return nil
	}
	title := got.Kind + " absorption"
	if got.Grade != "" {
		title = got.Grade + " " + title
	}
	return &AroundEvent{
		Kind: AroundEventAbsorption, Phase: phase, Side: got.Kind,
		Title: title, At: p.From, Score: got.Score,
		Summary: fmt.Sprintf("%s absorption (%s): market %s vs price %s%%.",
			got.Grade, got.Kind, got.Absorber, FormatSignedPct(p.Price.ChangePct)),
	}
}

func aroundFlatPct(d time.Duration) float64 {
	h := d.Hours()
	if h <= 0 {
		return 0.15
	}
	p := 0.10 + h*0.25
	if p > 0.80 {
		return 0.80
	}
	return p
}

func aroundPricePath(before, during, after string) string {
	if during == "" || during == CVDDirFlat {
		if after != "" && after != CVDDirFlat && before != "" && before != CVDDirFlat && after != before {
			return AroundPathReversed
		}
		return AroundPathChopped
	}
	if after == during {
		return AroundPathContinued
	}
	if after != "" && after != CVDDirFlat && after != during {
		return AroundPathReversed
	}
	if after == CVDDirFlat || after == "" {
		return AroundPathFaded
	}
	return AroundPathChopped
}

func aroundVolumePath(before, during, after float64) string {
	if during > before*1.5 && after < during*0.7 {
		return AroundPathFaded
	}
	if during > before*1.5 && after >= during*0.7 {
		return AroundPathContinued
	}
	if during < before*0.8 && after > during*1.3 {
		return AroundPathContinued
	}
	return AroundPathChopped
}

func aroundDeltaPath(before, during, after string) string {
	if during == "" || during == TakerSideEven {
		return AroundPathChopped
	}
	if after == during {
		return AroundPathContinued
	}
	if after != "" && after != TakerSideEven && after != during {
		return AroundPathReversed
	}
	return AroundPathFaded
}

func aroundOIPath(before, during, after AroundPhase) string {
	d := futuresOIDir(during)
	a := futuresOIDir(after)
	if d == "up" && a == "up" {
		return AroundPathContinued
	}
	if d == "up" && a == "down" {
		return AroundPathReversed
	}
	if d == "down" && a == "up" {
		return AroundPathReversed
	}
	if d == "up" || d == "down" {
		return AroundPathFaded
	}
	_ = before
	return AroundPathChopped
}

func aroundPriceChangeSummary(before, during, after AroundPhase, path string) string {
	switch path {
	case AroundPathContinued:
		return fmt.Sprintf("Price %s %s%% during the move and kept going %s%% after.",
			during.Price.Direction, FormatSignedPct(during.Price.ChangePct), FormatSignedPct(after.Price.ChangePct))
	case AroundPathReversed:
		return fmt.Sprintf("Price %s %s%% during the move, then reversed %s%% after.",
			during.Price.Direction, FormatSignedPct(during.Price.ChangePct), FormatSignedPct(after.Price.ChangePct))
	case AroundPathFaded:
		return fmt.Sprintf("Price %s %s%% during the move, then went quiet (%s%% after).",
			during.Price.Direction, FormatSignedPct(during.Price.ChangePct), FormatSignedPct(after.Price.ChangePct))
	default:
		return fmt.Sprintf("Price path was mixed (before %s%%, during %s%%, after %s%%).",
			FormatSignedPct(before.Price.ChangePct), FormatSignedPct(during.Price.ChangePct), FormatSignedPct(after.Price.ChangePct))
	}
}

func aroundVolumeChangeSummary(before, during, after AroundPhase, path string) string {
	switch path {
	case AroundPathFaded:
		return fmt.Sprintf("Volume spiked during the move (%s) then dried up (%s).",
			formatQty(during.Flow.Volume), formatQty(after.Flow.Volume))
	case AroundPathContinued:
		return fmt.Sprintf("Volume stayed elevated after the move (%s then %s).",
			formatQty(during.Flow.Volume), formatQty(after.Flow.Volume))
	default:
		return fmt.Sprintf("Volume: before %s, during %s, after %s.",
			formatQty(before.Flow.Volume), formatQty(during.Flow.Volume), formatQty(after.Flow.Volume))
	}
}

func aroundDeltaChangeSummary(before, during, after AroundPhase, path string) string {
	_ = before
	switch path {
	case AroundPathContinued:
		return "Aggressive " + during.Flow.Dominant + " continued after the move."
	case AroundPathReversed:
		return "Takers flipped from " + during.Flow.Dominant + " to " + after.Flow.Dominant + " after the move."
	case AroundPathFaded:
		return "Aggressive " + during.Flow.Dominant + " faded after the move."
	default:
		return "Taker side was mixed across the three windows."
	}
}

func aroundOIChangeSummary(before, during, after AroundPhase, path string) string {
	_ = before
	switch path {
	case AroundPathContinued:
		return fmt.Sprintf("Open interest kept %s after the move.", futuresOIDir(after))
	case AroundPathReversed:
		return "Open interest reversed after the move (positions were likely closed)."
	case AroundPathFaded:
		return fmt.Sprintf("Open interest %s during the move, then flattened.", futuresOIDir(during))
	default:
		return "Open interest did not show a clear path."
	}
}

func combineAroundPhases(id string, parts []AroundPhase, from, to time.Time) AroundPhase {
	out := AroundPhase{Phase: id, From: from.UTC(), To: to.UTC(), Events: []AroundEvent{}}
	if len(parts) == 0 {
		out.Summary = id + ": no candles in this window."
		return out
	}
	var oW, cW, hMax, lMin, vol, buy, sell, pv, typW, typN float64
	var known, complete bool
	var sample int
	var books []AroundBook
	var futs []AroundFutures
	var profs []AroundProfile
	for _, p := range parts {
		if !p.Complete {
			continue
		}
		complete = true
		out.BarCount += p.BarCount
		w := p.Flow.Volume
		if w <= 0 {
			w = 1
		}
		if p.Price.Open > 0 {
			oW += p.Price.Open * w
			cW += p.Price.Close * w
			vol += p.Flow.Volume
		}
		if p.Price.High > hMax {
			hMax = p.Price.High
		}
		if lMin == 0 || (p.Price.Low > 0 && p.Price.Low < lMin) {
			lMin = p.Price.Low
		}
		buy += p.Flow.BuyVolume
		sell += p.Flow.SellVolume
		if p.Flow.BuySellKnown {
			known = true
		}
		if p.Flow.VWAP > 0 && p.Flow.Volume > 0 {
			pv += p.Flow.VWAP * p.Flow.Volume
		}
		if p.Flow.TypicalKnown && p.Flow.Typical > 0 {
			typW += p.Flow.Typical
			typN++
			if p.Flow.TypicalSample > sample {
				sample = p.Flow.TypicalSample
			}
		}
		if p.Profile != nil {
			profs = append(profs, *p.Profile)
		}
		if p.Book != nil && p.Book.Complete {
			books = append(books, *p.Book)
		}
		if p.Futures != nil && p.Futures.Complete {
			futs = append(futs, *p.Futures)
		}
		out.Events = append(out.Events, p.Events...)
	}
	if !complete {
		out.Summary = id + ": no candles in this window."
		return out
	}
	out.Complete = true
	open, closePx := 0.0, 0.0
	if vol > 0 {
		open = oW / vol
		closePx = cW / vol
	}
	out.Price = AroundPrice{Open: open, High: hMax, Low: lMin, Close: closePx, Direction: CVDDirFlat}
	out.Price.Change = closePx - open
	if open > 0 {
		out.Price.ChangePct = out.Price.Change / open * 100
		out.Price.Range = hMax - lMin
		out.Price.RangePct = out.Price.Range / open * 100
	}
	out.Price.Direction = changeDir(out.Price.ChangePct)
	out.Flow = AroundFlow{
		Volume: vol, BuyVolume: buy, SellVolume: sell, Delta: buy - sell,
		BuySellKnown: known, Dominant: TakerSideEven, VsVWAP: VolumeProfileVsUnknown,
	}
	if buy+sell > 0 {
		out.Flow.BuyShare = buy / (buy + sell)
	}
	if known {
		out.Flow.Dominant = TakerDominant(buy, sell)
	}
	if vol > 0 && pv > 0 {
		out.Flow.VWAP = pv / vol
		if closePx > 0 {
			out.Flow.DistancePct = (closePx - out.Flow.VWAP) / out.Flow.VWAP * 100
			switch {
			case out.Flow.DistancePct > 0.02:
				out.Flow.VsVWAP = VolumeProfileVsAbove
			case out.Flow.DistancePct < -0.02:
				out.Flow.VsVWAP = VolumeProfileVsBelow
			default:
				out.Flow.VsVWAP = "at"
			}
		}
	}
	if typN > 0 {
		out.Flow.Typical = typW / typN
		out.Flow.TypicalSample = sample
		out.Flow.TypicalKnown = true
		if out.Flow.Typical > 0 {
			out.Flow.VolumeRatio = vol / out.Flow.Typical
		}
		out.Flow.VolumeGrade = VolumeSurgeGrade(out.Flow.VolumeRatio)
	}
	if len(profs) > 0 {
		p := profs[0]
		out.Profile = &p
	}
	if len(books) > 0 {
		b := combineAroundBooks(books)
		out.Book = &b
	}
	if len(futs) > 0 {
		f := combineAroundFutures(futs)
		out.Futures = &f
	}
	out.Summary = ExplainAroundPhase(out)
	return out
}

func combineAroundBooks(in []AroundBook) AroundBook {
	out := AroundBook{Complete: true}
	for _, b := range in {
		out.FromMid += b.FromMid
		out.ToMid += b.ToMid
		out.MidDelta += b.MidDelta
		out.MidDeltaPct += b.MidDeltaPct
		out.BidNotionalDelta += b.BidNotionalDelta
		out.AskNotionalDelta += b.AskNotionalDelta
		out.ImbalanceDelta += b.ImbalanceDelta
		out.WallsAdded += b.WallsAdded
		out.WallsRemoved += b.WallsRemoved
	}
	n := float64(len(in))
	if n > 0 {
		out.FromMid /= n
		out.ToMid /= n
		out.MidDelta /= n
		out.MidDeltaPct /= n
		out.ImbalanceDelta /= n
	}
	out.Summary = fmt.Sprintf("Book: mid %s%%, bid liq %s, ask liq %s.",
		FormatSignedPct(out.MidDeltaPct), FormatSignedQty(out.BidNotionalDelta), FormatSignedQty(out.AskNotionalDelta))
	return out
}

func combineAroundFutures(in []AroundFutures) AroundFutures {
	out := AroundFutures{Complete: true}
	var oiN, fundN, lsN float64
	for _, f := range in {
		if f.OIFrom > 0 || f.OITo > 0 {
			out.OIFrom += f.OIFrom
			out.OITo += f.OITo
			oiN++
		}
		if f.FundingFrom != 0 || f.FundingTo != 0 {
			out.FundingFrom += f.FundingFrom
			out.FundingTo += f.FundingTo
			fundN++
		}
		if f.LongPctFrom != 0 || f.LongPctTo != 0 {
			out.LongPctFrom += f.LongPctFrom
			out.LongPctTo += f.LongPctTo
			lsN++
		}
		out.LongLiq += f.LongLiq
		out.ShortLiq += f.ShortLiq
	}
	if oiN > 0 {
		out.OIChange = out.OITo - out.OIFrom
		if out.OIFrom > 0 {
			out.OIChangePct = out.OIChange / out.OIFrom * 100
		}
		out.OIDirection = changeDir(out.OIChangePct)
	}
	if fundN > 0 {
		out.FundingFrom /= fundN
		out.FundingTo /= fundN
	}
	if lsN > 0 {
		out.LongPctFrom /= lsN
		out.LongPctTo /= lsN
	}
	return out
}

func phaseFutures(p AroundPhase) bool {
	return p.Futures != nil && p.Futures.Complete
}

func futuresOIChange(p AroundPhase) float64 {
	if p.Futures == nil {
		return 0
	}
	return p.Futures.OIChangePct
}

func futuresOIDir(p AroundPhase) string {
	if p.Futures == nil || !p.Futures.Complete {
		return CVDDirFlat
	}
	if p.Futures.OIDirection != "" {
		return p.Futures.OIDirection
	}
	return changeDir(p.Futures.OIChangePct)
}

func countAroundKind(evs []AroundEvent, kind string) int {
	n := 0
	for _, e := range evs {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

// BookFromDiff compactly stores a stored-book compare on a phase.
func BookFromDiff(d BookHistoryDiff) *AroundBook {
	if d.From.SampledAt.IsZero() || d.To.SampledAt.IsZero() {
		return nil
	}
	return &AroundBook{
		FromMid: d.From.Mid, ToMid: d.To.Mid,
		MidDelta: d.MidDelta, MidDeltaPct: d.MidDeltaPct,
		BidNotionalDelta: d.BidNotionalDelta, AskNotionalDelta: d.AskNotionalDelta,
		ImbalanceDelta: d.ImbalanceDelta,
		WallsAdded:     len(d.WallsAdded), WallsRemoved: len(d.WallsRemoved),
		Summary: d.Summary, Complete: true,
	}
}

// FuturesAcrossSamples is OI / funding / long% at two times plus liquidations in the window.
func FuturesAcrossSamples(fromOI, toOI, fromFund, toFund, fromLS, toLS *FuturesSnapshot, liqs []LiquidationEvent) *AroundFutures {
	out := AroundFutures{}
	if fromOI != nil {
		out.OIFrom = futuresOIValue(*fromOI)
	}
	if toOI != nil {
		out.OITo = futuresOIValue(*toOI)
	}
	if out.OIFrom > 0 || out.OITo > 0 {
		out.OIChange = out.OITo - out.OIFrom
		if out.OIFrom > 0 {
			out.OIChangePct = out.OIChange / out.OIFrom * 100
		}
		out.OIDirection = changeDir(out.OIChangePct)
		out.Complete = true
	}
	if fromFund != nil {
		out.FundingFrom = fromFund.FundingRate
		out.Complete = true
	}
	if toFund != nil {
		out.FundingTo = toFund.FundingRate
		out.Complete = true
	}
	if fromLS != nil {
		out.LongPctFrom = fromLS.LongShare * 100
		out.Complete = true
	}
	if toLS != nil {
		out.LongPctTo = toLS.LongShare * 100
		out.Complete = true
	}
	for _, e := range liqs {
		switch e.Side {
		case LiquidationSideLong:
			out.LongLiq += e.Notional
		case LiquidationSideShort:
			out.ShortLiq += e.Notional
		}
		if e.Notional > 0 {
			out.Complete = true
		}
	}
	if !out.Complete {
		return nil
	}
	return &out
}

// NearestFuturesSnapshot prefers at-or-before within slack.
func NearestFuturesSnapshot(rows []FuturesSnapshot, at time.Time, slack time.Duration) *FuturesSnapshot {
	if at.IsZero() || len(rows) == 0 {
		return nil
	}
	if slack <= 0 {
		slack = 10 * time.Minute
	}
	at = at.UTC()
	var before, after *FuturesSnapshot
	for i := range rows {
		r := &rows[i]
		if r.SampledAt.IsZero() {
			continue
		}
		t := r.SampledAt.UTC()
		if !t.After(at) {
			if before == nil || t.After(before.SampledAt) {
				before = r
			}
			continue
		}
		if after == nil || t.Before(after.SampledAt) {
			after = r
		}
	}
	if before != nil && at.Sub(before.SampledAt.UTC()) <= slack {
		return before
	}
	if after != nil && after.SampledAt.UTC().Sub(at) <= slack {
		return after
	}
	if before != nil {
		return before
	}
	return after
}

func futuresOIValue(s FuturesSnapshot) float64 {
	if s.Value > 0 {
		return s.Value
	}
	return s.Contracts
}
