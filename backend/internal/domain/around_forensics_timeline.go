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
	ForensicsTimelineSweep   = "sweep"
	ForensicsTimelineAbsorb  = "absorption"
)

// ForensicsTimeline is one clock of what printed around the event.
type ForensicsTimeline struct {
	Events  []ForensicsTimelineEvent
	Summary string
}

// ForensicsTimelineEvent is one beat on that clock.
type ForensicsTimelineEvent struct {
	At          time.Time
	Offset      time.Duration
	OffsetLabel string
	Phase       string
	Kind        string
	Venue       string
	Direction   string
	Title       string
	Detail      string
}

// BuildForensicsTimeline orders OI, takers, price, follow-through, liquidations,
// options, and sweeps on one clock. Missing series are omitted.
func BuildForensicsTimeline(r ForensicsReport) ForensicsTimeline {
	if r.Tape == nil {
		return ForensicsTimeline{}
	}
	plan, _ := ResolveAroundPlan(r.Window, r.During, r.At, r.AsOf)
	if plan.At.IsZero() {
		plan.At = r.At.UTC()
	}
	var evs []ForensicsTimelineEvent
	for _, v := range r.Tape.Venues {
		if v.Error != "" || v.Exchange == "" || v.Exchange == "all" {
			continue
		}
		evs = append(evs, timelineVenueBeats(v, plan)...)
	}
	if len(evs) == 0 && r.Tape.Combined != nil && r.Tape.Combined.Error == "" {
		evs = append(evs, timelineVenueBeats(*r.Tape.Combined, plan)...)
	}
	if ev := timelineOptionsBeat(r.Tape, plan); ev.Kind != "" {
		evs = append(evs, ev)
	}
	evs = append(evs, timelineTapeEvents(r.Tape, plan)...)
	evs = dedupeForensicsTimeline(evs)
	sortForensicsTimeline(evs)
	return ForensicsTimeline{
		Events:  evs,
		Summary: explainForensicsTimeline(evs),
	}
}

func timelineVenueBeats(v AroundVenue, plan AroundPlan) []ForensicsTimelineEvent {
	venue := string(v.Exchange)
	var out []ForensicsTimelineEvent
	if !v.OIStart.IsZero() && (v.OIDir == CVDDirUp || v.OIDir == CVDDirDown) {
		word := "rising"
		if v.OIDir == CVDDirDown {
			word = "falling"
		}
		out = append(out, newForensicsTimelineEvent(v.OIStart, plan, ForensicsTimelineOI, venue, v.OIDir,
			fmt.Sprintf("%s open interest started %s", prettyTimelineVenue(venue), word),
			timelineUntilDetail(v.OIUntil)))
	}
	if !v.TakerStart.IsZero() && (v.TakerDir == TakerSideBuy || v.TakerDir == TakerSideSell) {
		out = append(out, newForensicsTimelineEvent(v.TakerStart, plan, ForensicsTimelineTakers, venue, v.TakerDir,
			fmt.Sprintf("%s takers started %s", prettyTimelineVenue(venue), v.TakerDir+"ing"),
			timelineUntilDetail(v.TakerUntil)))
	}
	if !v.MoveStart.IsZero() {
		dir := timelinePriceDir(v)
		word := "moving"
		if dir == CVDDirUp {
			word = "moving up"
		} else if dir == CVDDirDown {
			word = "moving down"
		}
		out = append(out, newForensicsTimelineEvent(v.MoveStart, plan, ForensicsTimelinePrice, venue, dir,
			fmt.Sprintf("%s price started %s", prettyTimelineVenue(venue), word), ""))
	}
	if !v.LiqStart.IsZero() {
		side := v.LiqSide
		title := fmt.Sprintf("%s liquidations started", prettyTimelineVenue(venue))
		if side == LiquidationSideLong || side == LiquidationSideShort {
			title = fmt.Sprintf("%s %s liquidations started", prettyTimelineVenue(venue), side)
		}
		out = append(out, newForensicsTimelineEvent(v.LiqStart, plan, ForensicsTimelineLiq, venue, side, title, ""))
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

func timelineOptionsBeat(tape *AroundReport, plan AroundPlan) ForensicsTimelineEvent {
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
			if o == nil || !o.Unusual || o.StartedAt.IsZero() {
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
	return newForensicsTimelineEvent(best.StartedAt, plan, ForensicsTimelineOptions, string(ExchangeDeribit), best.OIDirection, title, strings.TrimSpace(best.Summary))
}

func timelineTapeEvents(tape *AroundReport, plan AroundPlan) []ForensicsTimelineEvent {
	if tape == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []ForensicsTimelineEvent
	add := func(ev AroundEvent, venue string) {
		if ev.At.IsZero() {
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
		out = append(out, newForensicsTimelineEvent(ev.At, plan, kind, venue, ev.Side, title, ev.Summary))
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

func newForensicsTimelineEvent(at time.Time, plan AroundPlan, kind, venue, dir, title, detail string) ForensicsTimelineEvent {
	at = at.UTC()
	off := at.Sub(plan.At)
	if plan.At.IsZero() {
		off = 0
	}
	return ForensicsTimelineEvent{
		At:          at,
		Offset:      off,
		OffsetLabel: timelineOffsetLabel(off),
		Phase:       timelinePhaseOf(at, plan),
		Kind:        kind,
		Venue:       venue,
		Direction:   dir,
		Title:       title,
		Detail:      detail,
	}
}

func timelineOffsetLabel(d time.Duration) string {
	if d < 0 {
		d = -d
		if d < 45*time.Second {
			return "at the event"
		}
		return formatOptionsLead(d) + " before"
	}
	if d < 45*time.Second {
		return "at the event"
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
		case ForensicsTimelineOptions:
			return 3
		case ForensicsTimelinePrice:
			return 4
		case ForensicsTimelineLiq:
			return 5
		case ForensicsTimelineSweep:
			return 6
		case ForensicsTimelineAbsorb:
			return 7
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

func explainForensicsTimeline(evs []ForensicsTimelineEvent) string {
	if len(evs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(evs))
	for _, e := range evs {
		s := e.Title
		if e.OffsetLabel != "" {
			if e.OffsetLabel == "at the event" {
				s += " at the event"
			} else {
				s += " (" + e.OffsetLabel + ")"
			}
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ". ") + "."
}
