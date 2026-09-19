package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	ForensicsTimelineOI      = "oi"
	ForensicsTimelineTakers  = "takers"
	ForensicsTimelinePrice   = "price"
	ForensicsTimelineLiq     = "liquidations"
	ForensicsTimelineOptions = "options"
	ForensicsTimelineGamma   = "gamma"
	ForensicsTimelineSweep   = "sweep"
	ForensicsTimelineAbsorb  = "absorption"
)

// ForensicsTimeline is one clock of what printed around the event.
type ForensicsTimeline struct {
	Events     []ForensicsTimelineEvent
	Summary    string
	KnownUntil time.Time
	OrderKnown bool
}

// ForensicsTimelineEvent is one beat on that clock.
type ForensicsTimelineEvent struct {
	At           time.Time
	Offset       time.Duration
	OffsetLabel  string
	Phase        string
	Kind         string
	Venue        string
	Direction    string
	Title        string
	Detail       string
	Resolution   time.Duration
	OrderUnknown bool
}

// ForensicsTimelineKnownUntil is the latest print the timeline may show.
// That is the observer clock (asOf, or now). Prints after this time were
// not visible yet. Reviewing an old event with asOf=now keeps later
// prints that have already happened.
func ForensicsTimelineKnownUntil(asOf time.Time) time.Time {
	if asOf.IsZero() {
		return time.Now().UTC()
	}
	return asOf.UTC()
}

func timelineEventKnown(at, cutoff time.Time) bool {
	if at.IsZero() {
		return false
	}
	if cutoff.IsZero() {
		return true
	}
	return !at.UTC().After(cutoff.UTC())
}

// BuildForensicsTimeline orders OI, takers, price, follow-through, liquidations,
// options, dealer gamma, and sweeps on one clock. Missing series are omitted. Prints after
// KnownUntil are omitted — they were not visible yet at that observer clock.
func BuildForensicsTimeline(r ForensicsReport) ForensicsTimeline {
	if r.Tape == nil {
		return ForensicsTimeline{}
	}
	cutoff := ForensicsTimelineKnownUntil(r.AsOf)
	plan, _ := ResolveAroundPlan(r.Window, r.During, r.At, cutoff)
	if plan.At.IsZero() {
		plan.At = r.At.UTC()
	}
	var evs []ForensicsTimelineEvent
	for _, v := range r.Tape.Venues {
		if v.Error != "" || v.Exchange == "" || v.Exchange == "all" {
			continue
		}
		evs = append(evs, timelineVenueBeats(v, plan, cutoff)...)
	}
	if len(evs) == 0 && r.Tape.Combined != nil && r.Tape.Combined.Error == "" {
		evs = append(evs, timelineVenueBeats(*r.Tape.Combined, plan, cutoff)...)
	}
	if ev := timelineGammaBeat(r.Tape, plan, cutoff); ev.Kind != "" {
		evs = append(evs, ev)
	}
	if ev := timelineOptionsBeat(r.Tape, plan, cutoff); ev.Kind != "" {
		evs = append(evs, ev)
	}
	evs = append(evs, timelineTapeEvents(r.Tape, plan, cutoff)...)
	evs = dedupeForensicsTimeline(evs)
	sortForensicsTimeline(evs)
	markForensicsTimelineOrder(evs)
	return ForensicsTimeline{
		Events:     evs,
		Summary:    explainForensicsTimeline(evs),
		KnownUntil: cutoff,
		OrderKnown: forensicsTimelineOrderKnown(evs),
	}
}

func timelineKindResolution(kind, interval string) time.Duration {
	switch kind {
	case ForensicsTimelineLiq:
		return 0
	case ForensicsTimelineOI, ForensicsTimelineOptions, ForensicsTimelineGamma:
		return aroundVenueOIStep
	case ForensicsTimelineSweep:
		return 15 * time.Minute
	default:
		return aroundIntervalOrMinute(interval)
	}
}

func timelineVenueBeats(v AroundVenue, plan AroundPlan, cutoff time.Time) []ForensicsTimelineEvent {
	venue := string(v.Exchange)
	var out []ForensicsTimelineEvent
	if timelineEventKnown(v.OIStart, cutoff) && (v.OIDir == CVDDirUp || v.OIDir == CVDDirDown) {
		word := "rising"
		if v.OIDir == CVDDirDown {
			word = "falling"
		}
		out = append(out, newForensicsTimelineEvent(v.OIStart, plan, ForensicsTimelineOI, venue, v.OIDir,
			fmt.Sprintf("%s open interest started %s", prettyTimelineVenue(venue), word),
			timelineUntilDetail(v.OIUntil), timelineKindResolution(ForensicsTimelineOI, v.Interval)))
	}
	if timelineEventKnown(v.TakerStart, cutoff) && (v.TakerDir == TakerSideBuy || v.TakerDir == TakerSideSell) {
		out = append(out, newForensicsTimelineEvent(v.TakerStart, plan, ForensicsTimelineTakers, venue, v.TakerDir,
			fmt.Sprintf("%s takers started %s", prettyTimelineVenue(venue), v.TakerDir+"ing"),
			timelineUntilDetail(v.TakerUntil), timelineKindResolution(ForensicsTimelineTakers, v.Interval)))
	}
	if timelineEventKnown(v.MoveStart, cutoff) {
		dir := timelinePriceDir(v)
		word := "moving"
		if dir == CVDDirUp {
			word = "moving up"
		} else if dir == CVDDirDown {
			word = "moving down"
		}
		out = append(out, newForensicsTimelineEvent(v.MoveStart, plan, ForensicsTimelinePrice, venue, dir,
			fmt.Sprintf("%s price started %s", prettyTimelineVenue(venue), word), "", timelineKindResolution(ForensicsTimelinePrice, v.Interval)))
	}
	if timelineEventKnown(v.LiqStart, cutoff) {
		side := v.LiqSide
		title := fmt.Sprintf("%s liquidations started", prettyTimelineVenue(venue))
		if side == LiquidationSideLong || side == LiquidationSideShort {
			title = fmt.Sprintf("%s %s liquidations started", prettyTimelineVenue(venue), side)
		}
		out = append(out, newForensicsTimelineEvent(v.LiqStart, plan, ForensicsTimelineLiq, venue, side, title, "", timelineKindResolution(ForensicsTimelineLiq, v.Interval)))
	}
	return out
}

func timelinePriceDir(v AroundVenue) string {
	if p, ok := AroundPhaseByID(v, AroundPhaseDuring); ok && (p.Price.Direction == CVDDirUp || p.Price.Direction == CVDDirDown) {
		return p.Price.Direction
	}
	if p, ok := AroundPhaseByID(v, AroundPhaseAfter); ok && (p.Price.Direction == CVDDirUp || p.Price.Direction == CVDDirDown) {
		return p.Price.Direction
	}
	return ""
}

func timelineUntilDetail(until string) string {
	switch until {
	case AroundVenueUntilContinued:
		return "Kept going until price moved."
	case AroundVenueUntilFaded:
		return "Faded before price moved."
	case AroundVenueUntilReversed:
		return "Reversed before price moved."
	default:
		return ""
	}
}

func timelineGammaBeat(tape *AroundReport, plan AroundPlan, cutoff time.Time) ForensicsTimelineEvent {
	if tape == nil {
		return ForensicsTimelineEvent{}
	}
	o := beforePhaseGamma(tape)
	if o == nil || (!o.HasGamma && !o.HasGammaFlip) {
		return ForensicsTimelineEvent{}
	}
	at := o.GammaAt
	if at.IsZero() {
		at = plan.At
	}
	if !timelineEventKnown(at, cutoff) {
		return ForensicsTimelineEvent{}
	}
	title, detail := timelineGammaCopy(*o)
	if title == "" {
		return ForensicsTimelineEvent{}
	}
	return newForensicsTimelineEvent(at, plan, ForensicsTimelineGamma, string(ExchangeDeribit), o.GammaRegime, title, detail, timelineKindResolution(ForensicsTimelineGamma, ""))
}

func beforePhaseGamma(tape *AroundReport) *AroundOptions {
	look := func(v *AroundVenue) *AroundOptions {
		if v == nil {
			return nil
		}
		for i := range v.Phases {
			if v.Phases[i].Phase != AroundPhaseBefore {
				continue
			}
			o := v.Phases[i].Options
			if o != nil && (o.HasGamma || o.HasGammaFlip) {
				cp := *o
				return &cp
			}
		}
		return nil
	}
	if o := look(tape.Combined); o != nil {
		return o
	}
	for i := range tape.Venues {
		if o := look(&tape.Venues[i]); o != nil {
			return o
		}
	}
	return nil
}

func timelineGammaCopy(o AroundOptions) (title, detail string) {
	switch o.GammaRegime {
	case GammaRegimeShort:
		title = "Dealers were short gamma"
	case GammaRegimeLong:
		title = "Dealers were long gamma"
	case GammaRegimeFlat:
		title = "Dealer gamma was balanced"
	}
	var bits []string
	if o.HasGammaFlip && o.GammaFlip > 0 {
		switch o.GammaFlipSide {
		case GammaFlipAbove:
			bits = append(bits, fmt.Sprintf("Spot was above the flip (%.0f)", o.GammaFlip))
		case GammaFlipBelow:
			bits = append(bits, fmt.Sprintf("Spot was below the flip (%.0f)", o.GammaFlip))
		case GammaFlipAt:
			bits = append(bits, fmt.Sprintf("Spot was at the flip (%.0f)", o.GammaFlip))
		}
	}
	if o.HasGammaRegimeChange && o.GammaRegimeFrom != "" {
		bits = append(bits, fmt.Sprintf("Gamma flipped from %s to %s before the move", o.GammaRegimeFrom, o.GammaRegime))
	}
	if title == "" && len(bits) > 0 {
		title = bits[0]
		bits = bits[1:]
	}
	detail = strings.Join(bits, ". ")
	if detail != "" && !strings.HasSuffix(detail, ".") {
		detail += "."
	}
	return title, detail
}

func timelineOptionsBeat(tape *AroundReport, plan AroundPlan, cutoff time.Time) ForensicsTimelineEvent {
	if tape == nil {
		return ForensicsTimelineEvent{}
	}
	var best *AroundOptions
	look := func(v *AroundVenue) {
		if v == nil {
			return
		}
		for i := range v.Phases {
			o := v.Phases[i].Options
			if o == nil || !o.Unusual || !timelineEventKnown(o.StartedAt, cutoff) {
				continue
			}
			if best == nil || o.StartedAt.Before(best.StartedAt) {
				cp := *o
				best = &cp
			}
		}
	}
	look(tape.Combined)
	for i := range tape.Venues {
		look(&tape.Venues[i])
	}
	if best == nil {
		return ForensicsTimelineEvent{}
	}
	title := "Listed options turned unusual"
	if best.LeadLabel != "" && best.Started == AroundOptionsStartedBefore {
		title = "Listed options turned unusual " + best.LeadLabel + " before the move"
	}
	return newForensicsTimelineEvent(best.StartedAt, plan, ForensicsTimelineOptions, string(ExchangeDeribit), best.OIDirection, title, strings.TrimSpace(best.Summary), timelineKindResolution(ForensicsTimelineOptions, ""))
}

func timelineTapeEvents(tape *AroundReport, plan AroundPlan, cutoff time.Time) []ForensicsTimelineEvent {
	if tape == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []ForensicsTimelineEvent
	add := func(ev AroundEvent, venue string) {
		if !timelineEventKnown(ev.At, cutoff) {
			return
		}
		kind := ev.Kind
		if kind != AroundEventSweep && kind != AroundEventAbsorption {
			return
		}
		key := kind + "|" + ev.At.UTC().Truncate(time.Second).Format(time.RFC3339) + "|" + ev.Title
		if seen[key] {
			return
		}
		seen[key] = true
		title := ev.Title
		if title == "" {
			title = kind
		}
		out = append(out, newForensicsTimelineEvent(ev.At, plan, kind, venue, ev.Side, title, ev.Summary, timelineKindResolution(kind, "")))
	}
	for _, v := range tape.Venues {
		for _, ev := range v.Events {
			add(ev, string(v.Exchange))
		}
	}
	if tape.Combined != nil {
		for _, ev := range tape.Combined.Events {
			add(ev, string(tape.Combined.Exchange))
		}
	}
	return out
}

func newForensicsTimelineEvent(at time.Time, plan AroundPlan, kind, venue, dir, title, detail string, res time.Duration) ForensicsTimelineEvent {
	at = at.UTC()
	off := at.Sub(plan.At)
	if plan.At.IsZero() {
		off = 0
	}
	return ForensicsTimelineEvent{
		At:          at,
		Offset:      off,
		OffsetLabel: timelineOffsetLabelAt(at, plan.At, AroundStamp{At: at, Resolution: res, Backward: timelineKindBackward(kind)}),
		Phase:       timelinePhaseOf(at, plan),
		Kind:        kind,
		Venue:       venue,
		Direction:   dir,
		Title:       title,
		Detail:      detail,
		Resolution:  res,
	}
}

func timelineKindBackward(kind string) bool {
	return kind == ForensicsTimelineOI || kind == ForensicsTimelineOptions || kind == ForensicsTimelineGamma
}

func timelineOffsetLabelAt(at, event time.Time, stamp AroundStamp) string {
	if !event.IsZero() && AroundStampsOverlap(stamp, AroundStamp{At: event}) {
		off := at.Sub(event)
		if off < 0 {
			off = -off
		}
		if off < 45*time.Second {
			return "at the event"
		}
		return "around the event"
	}
	return timelineOffsetLabel(at.Sub(event), stamp.Resolution)
}

func timelineOffsetLabel(d, res time.Duration) string {
	abs := d
	if abs < 0 {
		abs = -abs
	}
	if abs < 45*time.Second {
		return "at the event"
	}
	if res > 0 && abs < res {
		return "around the event"
	}
	if d < 0 {
		return formatOptionsLead(-d) + " before"
	}
	return formatOptionsLead(d) + " after"
}

// FormatForensicsTimelineOffset is the signed short offset on the wire (-8m, 0, +3m).
func FormatForensicsTimelineOffset(d time.Duration) string {
	if d < 0 {
		d = -d
		if d < 45*time.Second {
			return "0"
		}
		return "-" + FormatOptionsLeadShort(d)
	}
	if d < 45*time.Second {
		return "0"
	}
	return "+" + FormatOptionsLeadShort(d)
}

func timelinePhaseOf(at time.Time, plan AroundPlan) string {
	if plan.At.IsZero() {
		return ""
	}
	if !plan.AfterFrom.IsZero() && !at.Before(plan.AfterFrom) {
		return AroundPhaseAfter
	}
	if !plan.DuringFrom.IsZero() && !at.Before(plan.DuringFrom) {
		return AroundPhaseDuring
	}
	return AroundPhaseBefore
}

func prettyTimelineVenue(id string) string {
	switch Exchange(strings.ToLower(strings.TrimSpace(id))) {
	case ExchangeBinance:
		return "Binance"
	case ExchangeBybit:
		return "Bybit"
	case ExchangeDeribit:
		return "Deribit"
	case ExchangeCoinbase:
		return "Coinbase"
	default:
		if id == "" || id == "all" {
			return "The market"
		}
		return id
	}
}

func dedupeForensicsTimeline(in []ForensicsTimelineEvent) []ForensicsTimelineEvent {
	seen := map[string]int{}
	out := make([]ForensicsTimelineEvent, 0, len(in))
	for _, e := range in {
		if e.At.IsZero() || e.Title == "" {
			continue
		}
		key := e.Kind + "|" + e.Venue + "|" + e.At.Truncate(time.Second).Format(time.RFC3339)
		if i, ok := seen[key]; ok {
			if e.Detail != "" && out[i].Detail == "" {
				out[i].Detail = e.Detail
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, e)
	}
	return out
}

func sortForensicsTimeline(evs []ForensicsTimelineEvent) {
	rank := func(kind string) int {
		switch kind {
		case ForensicsTimelineOI:
			return 1
		case ForensicsTimelineTakers:
			return 2
		case ForensicsTimelineGamma:
			return 3
		case ForensicsTimelineOptions:
			return 4
		case ForensicsTimelinePrice:
			return 5
		case ForensicsTimelineLiq:
			return 6
		case ForensicsTimelineSweep:
			return 7
		case ForensicsTimelineAbsorb:
			return 8
		default:
			return 8
		}
	}
	sort.SliceStable(evs, func(i, j int) bool {
		if !evs[i].At.Equal(evs[j].At) {
			return evs[i].At.Before(evs[j].At)
		}
		if ri, rj := rank(evs[i].Kind), rank(evs[j].Kind); ri != rj {
			return ri < rj
		}
		return evs[i].Venue < evs[j].Venue
	})
}

func timelineEventStamp(e ForensicsTimelineEvent) AroundStamp {
	return AroundStamp{At: e.At, Resolution: e.Resolution, Backward: timelineKindBackward(e.Kind)}
}

func markForensicsTimelineOrder(evs []ForensicsTimelineEvent) {
	for i := range evs {
		for j := range evs {
			if i == j {
				continue
			}
			if AroundStampsOverlap(timelineEventStamp(evs[i]), timelineEventStamp(evs[j])) {
				evs[i].OrderUnknown = true
				break
			}
		}
	}
}

func forensicsTimelineOrderKnown(evs []ForensicsTimelineEvent) bool {
	if len(evs) <= 1 {
		return true
	}
	for _, e := range evs {
		if e.OrderUnknown {
			return false
		}
	}
	return true
}

func explainForensicsTimeline(evs []ForensicsTimelineEvent) string {
	if len(evs) == 0 {
		return ""
	}
	var parts []string
	for i := 0; i < len(evs); {
		cluster := []ForensicsTimelineEvent{evs[i]}
		j := i + 1
		for j < len(evs) {
			ok := false
			for _, c := range cluster {
				if AroundStampsOverlap(timelineEventStamp(c), timelineEventStamp(evs[j])) {
					ok = true
					break
				}
			}
			if !ok {
				break
			}
			cluster = append(cluster, evs[j])
			j++
		}
		parts = append(parts, explainForensicsTimelineCluster(cluster))
		i = j
	}
	return strings.Join(parts, " ")
}

func explainForensicsTimelineCluster(cluster []ForensicsTimelineEvent) string {
	if len(cluster) == 1 {
		e := cluster[0]
		s := e.Title
		if e.OffsetLabel != "" {
			if e.OffsetLabel == "at the event" {
				s += " at the event"
			} else {
				s += " (" + e.OffsetLabel + ")"
			}
		}
		return s + "."
	}
	titles := make([]string, 0, len(cluster))
	for _, e := range cluster {
		if e.Title != "" {
			titles = append(titles, e.Title)
		}
	}
	at := cluster[0].At.UTC().Format("15:04")
	return "Around " + at + " UTC: " + strings.Join(titles, "; ") + ". Order is not certain at this tape's bar size."
}
