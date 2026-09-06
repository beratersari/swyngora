package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	aroundOptionsSlack     = 30 * time.Minute
	aroundOptionsOIFloor   = 5.0
	aroundOptionsPCRFloor  = 0.10
	aroundOptionsDVOLFloor = 1.5

	AroundOptionsStartedBefore = "before"
	AroundOptionsStartedDuring = "during"

	// AroundOptionsLookbackPad is extra history so unusual-move typicals can form.
	AroundOptionsLookbackPad = 48 * time.Hour
)

// AroundOptions is listed BTC/ETH options across one around-the-move phase.
// Missing when the coin is not BTC/ETH or the history archive has no samples.
type AroundOptions struct {
	Asset        string
	OIFrom       float64
	OITo         float64
	OIChange     float64
	OIChangePct  float64
	OIDirection  string
	HasOI        bool
	PutCallFrom  float64
	PutCallTo    float64
	PutCallDelta float64
	HasPutCall   bool
	LeanFrom     string
	LeanTo       string
	DVOLFrom     float64
	DVOLTo       float64
	DVOLDelta    float64
	HasDVOL      bool
	AtmIVFrom    float64
	AtmIVTo      float64
	AtmIVDelta   float64
	HasAtmIV     bool
	Moves        []OptionsMove
	Unusual      bool
	Started      string // before | during — when the unusual print began vs the price event
	Summary      string
	Complete     bool
}

// OptionsHistoryUntil keeps samples at or before at (inclusive).
func OptionsHistoryUntil(points []OptionsHistoryPoint, at time.Time) []OptionsHistoryPoint {
	if len(points) == 0 {
		return nil
	}
	at = at.UTC()
	out := make([]OptionsHistoryPoint, 0, len(points))
	for _, p := range points {
		if p.Time.After(at) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// MergeOptionsHistoryPoints folds stored desk prints and DVOL candles.
func MergeOptionsHistoryPoints(samples []OptionsSample, dvol []OptionsDVOLPoint, from, to time.Time) []OptionsHistoryPoint {
	return mergeOptionsHistory(nil, samples, dvol, from, to)
}

// OptionsAcrossWindow is listed OI / put-call / IV across a phase.
// A series is only filled when two distinct samples both have that field —
// a missing start is not treated as zero.
func OptionsAcrossWindow(asset string, points []OptionsHistoryPoint, from, to time.Time) *AroundOptions {
	if len(points) == 0 || !to.After(from) {
		return nil
	}
	start, end, ok := optionsHistoryPair(points, from, to)
	if !ok {
		return nil
	}
	out := AroundOptions{Asset: asset}
	if start.HasOI && end.HasOI {
		out.HasOI = true
		out.OIFrom = start.OpenInterest
		out.OITo = end.OpenInterest
		out.OIChange = out.OITo - out.OIFrom
		if out.OIFrom > 0 {
			out.OIChangePct = out.OIChange / out.OIFrom * 100
		}
		out.OIDirection = changeDir(out.OIChangePct)
		out.Complete = true
	}
	if start.HasPutCallOI && end.HasPutCallOI {
		out.HasPutCall = true
		out.PutCallFrom = start.PutCallOI
		out.PutCallTo = end.PutCallOI
		out.PutCallDelta = out.PutCallTo - out.PutCallFrom
		out.LeanFrom = optionsLean(start.PutCallOI)
		out.LeanTo = optionsLean(end.PutCallOI)
		out.Complete = true
	}
	if start.HasDVOL && end.HasDVOL {
		out.HasDVOL = true
		out.DVOLFrom = start.DVOL
		out.DVOLTo = end.DVOL
		out.DVOLDelta = out.DVOLTo - out.DVOLFrom
		out.Complete = true
	}
	if start.HasAtmIV && end.HasAtmIV {
		out.HasAtmIV = true
		out.AtmIVFrom = start.AtmIV
		out.AtmIVTo = end.AtmIV
		out.AtmIVDelta = out.AtmIVTo - out.AtmIVFrom
		out.Complete = true
	}
	if !out.Complete {
		return nil
	}
	out.Unusual = optionsWindowLooksUnusual(out)
	out.Summary = explainAroundOptions(out)
	return &out
}

func optionsHistoryPair(points []OptionsHistoryPoint, from, to time.Time) (start, end OptionsHistoryPoint, ok bool) {
	from, to = from.UTC(), to.UTC()
	start = optionsHistoryAtOrBefore(points, from, aroundOptionsSlack)
	if start.Time.IsZero() {
		start = firstOptionsPointIn(points, from, to)
	}
	end = optionsHistoryAtOrBefore(points, to, aroundOptionsSlack)
	if start.Time.IsZero() || end.Time.IsZero() || !end.Time.After(start.Time) {
		return OptionsHistoryPoint{}, OptionsHistoryPoint{}, false
	}
	return start, end, true
}

func firstOptionsPointIn(points []OptionsHistoryPoint, from, to time.Time) OptionsHistoryPoint {
	for _, p := range points {
		if p.Time.Before(from) || !p.Time.Before(to) {
			continue
		}
		return p
	}
	return OptionsHistoryPoint{}
}

// FinalizeAroundOptions marks whether an unusual print started before the
// price event or only once price was already moving, and attaches lookback
// findings to the before-window only.
func FinalizeAroundOptions(phases []AroundPhase, points []OptionsHistoryPoint) {
	var before, during *AroundPhase
	for i := range phases {
		switch phases[i].Phase {
		case AroundPhaseBefore:
			before = &phases[i]
		case AroundPhaseDuring:
			during = &phases[i]
		}
	}
	if before != nil && before.To.After(before.From) {
		attachBeforeOptionsLookback(before, points)
	}
	var bOpt, dOpt *AroundOptions
	if before != nil {
		bOpt = before.Options
	}
	if during != nil {
		dOpt = during.Options
	}
	MarkAroundOptionsTiming(bOpt, dOpt)
	for i := range phases {
		if phases[i].Options == nil {
			continue
		}
		phases[i].Options.Summary = explainAroundOptions(*phases[i].Options)
		phases[i].Summary = ExplainAroundPhase(phases[i])
	}
}

func attachBeforeOptionsLookback(before *AroundPhase, points []OptionsHistoryPoint) {
	if before == nil {
		return
	}
	moves := DetectOptionsMoves(OptionsHistoryUntil(points, before.To), before.To)
	if moves == nil {
		return
	}
	if before.Options == nil {
		before.Options = &AroundOptions{Unusual: true, Started: AroundOptionsStartedBefore}
	}
	before.Options.Moves = append([]OptionsMove(nil), moves.Findings...)
	before.Options.Unusual = true
	if before.Options.Summary == "" {
		before.Options.Summary = moves.Summary
	}
}

// MarkAroundOptionsTiming sets Started on before/during unusual prints.
func MarkAroundOptionsTiming(before, during *AroundOptions) {
	beforeU := before != nil && before.Unusual
	duringU := during != nil && during.Unusual
	if beforeU {
		before.Started = AroundOptionsStartedBefore
	}
	if !duringU {
		return
	}
	if !beforeU {
		during.Started = AroundOptionsStartedDuring
		return
	}
	if optionsPrintContinued(before, during) {
		during.Started = AroundOptionsStartedBefore
		return
	}
	during.Started = AroundOptionsStartedDuring
}

func optionsPrintContinued(before, during *AroundOptions) bool {
	if before == nil || during == nil {
		return false
	}
	if before.HasOI && during.HasOI && before.OIDirection != "" && before.OIDirection == during.OIDirection && before.OIDirection != CVDDirFlat {
		return true
	}
	if before.HasDVOL && during.HasDVOL && changeDir(before.DVOLDelta) == changeDir(during.DVOLDelta) && changeDir(before.DVOLDelta) != CVDDirFlat {
		return true
	}
	if before.HasAtmIV && during.HasAtmIV && changeDir(before.AtmIVDelta) == changeDir(during.AtmIVDelta) && changeDir(before.AtmIVDelta) != CVDDirFlat {
		return true
	}
	if before.HasPutCall && during.HasPutCall && before.LeanTo != "" && before.LeanTo == during.LeanTo && before.LeanTo != OptionsLeanBalanced {
		return true
	}
	return false
}

// optionsComparable is a two-point options read that similar-setup can score.
func optionsComparable(o *AroundOptions) bool {
	if o == nil || !o.Complete {
		return false
	}
	return o.HasOI || o.HasPutCall || o.HasDVOL || o.HasAtmIV
}

func optionsSimilarity(a, b *AroundOptions) (score float64, match bool) {
	if !optionsComparable(a) || !optionsComparable(b) {
		return 0, false
	}
	var parts []float64
	if a.HasOI && b.HasOI {
		parts = append(parts, closenessSigned(a.OIChangePct, b.OIChangePct))
	}
	if a.HasPutCall && b.HasPutCall {
		parts = append(parts, closenessSigned(a.PutCallDelta, b.PutCallDelta))
	}
	if a.HasDVOL && b.HasDVOL {
		parts = append(parts, closenessSigned(a.DVOLDelta, b.DVOLDelta))
	} else if a.HasAtmIV && b.HasAtmIV {
		parts = append(parts, closenessSigned(a.AtmIVDelta, b.AtmIVDelta))
	}
	if len(parts) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, p := range parts {
		sum += p
	}
	score = sum / float64(len(parts))
	if a.Started != "" && b.Started != "" && a.Started != b.Started {
		score *= 0.5
	}
	if a.Unusual != b.Unusual {
		score *= 0.6
	}
	oiSame := !a.HasOI || !b.HasOI || sameDir(a.OIDirection, b.OIDirection) || (a.OIDirection == CVDDirFlat && b.OIDirection == CVDDirFlat)
	whenSame := a.Started == "" || b.Started == "" || a.Started == b.Started
	match = score >= 0.65 && oiSame && whenSame && a.Unusual == b.Unusual
	return score, match
}

func optionsWindowLooksUnusual(o AroundOptions) bool {
	if o.HasOI && absFloat(o.OIChangePct) >= aroundOptionsOIFloor {
		return true
	}
	if o.HasPutCall && absFloat(o.PutCallDelta) >= aroundOptionsPCRFloor {
		return true
	}
	if o.HasDVOL && absFloat(o.DVOLDelta) >= aroundOptionsDVOLFloor {
		return true
	}
	if o.HasAtmIV && absFloat(o.AtmIVDelta) >= aroundOptionsDVOLFloor {
		return true
	}
	return false
}

func explainAroundOptions(o AroundOptions) string {
	var bits []string
	if o.HasOI && o.OIDirection != "" && o.OIDirection != CVDDirFlat {
		bits = append(bits, fmt.Sprintf("listed open interest %s %s%%", o.OIDirection, FormatSignedPct(o.OIChangePct)))
	}
	if o.HasPutCall && o.LeanFrom != "" && o.LeanTo != "" && o.LeanFrom != o.LeanTo {
		bits = append(bits, fmt.Sprintf("put/call lean shifted from %s to %s", o.LeanFrom, o.LeanTo))
	} else if o.HasPutCall && absFloat(o.PutCallDelta) >= aroundOptionsPCRFloor {
		bits = append(bits, fmt.Sprintf("put/call moved %s", FormatSignedQty(o.PutCallDelta)))
	}
	if o.HasDVOL && absFloat(o.DVOLDelta) >= aroundOptionsDVOLFloor {
		bits = append(bits, fmt.Sprintf("implied vol %s %.1f pts", changeDir(o.DVOLDelta), o.DVOLDelta))
	} else if o.HasAtmIV && absFloat(o.AtmIVDelta) >= aroundOptionsDVOLFloor {
		bits = append(bits, fmt.Sprintf("ATM IV %s %.1f pts", changeDir(o.AtmIVDelta), o.AtmIVDelta))
	}
	if len(bits) == 0 {
		if o.Unusual && o.Started == AroundOptionsStartedBefore {
			return "Listed options were already unusual before the price move."
		}
		if o.Unusual && o.Started == AroundOptionsStartedDuring {
			return "Listed options became unusual during the price move."
		}
		return "Listed options were mostly steady in this window."
	}
	head := "Listed options: " + strings.Join(bits, "; ") + "."
	if o.Unusual && o.Started == AroundOptionsStartedBefore {
		return "Unusual options before the price move — " + strings.TrimPrefix(head, "Listed options: ")
	}
	if o.Unusual && o.Started == AroundOptionsStartedDuring {
		return "Unusual options during the price move — " + strings.TrimPrefix(head, "Listed options: ")
	}
	if o.Unusual {
		return "Unusual options print — " + strings.TrimPrefix(head, "Listed options: ")
	}
	return head
}

// AroundOptionsToEvents turns unusual options prints into tape events.
func AroundOptionsToEvents(o AroundOptions, phase string, at time.Time) []AroundEvent {
	if !o.Complete || !o.Unusual {
		return nil
	}
	title := unusualOptionsTitle(o)
	if len(o.Moves) > 0 {
		out := make([]AroundEvent, 0, len(o.Moves))
		for _, m := range o.Moves {
			out = append(out, AroundEvent{
				Kind:    AroundEventOptions,
				Phase:   phase,
				Side:    firstNonEmpty(o.Started, m.Topic),
				Title:   title,
				Summary: firstNonEmpty(m.Detail, m.Title),
				At:      at.UTC(),
			})
		}
		return out
	}
	return []AroundEvent{{
		Kind:    AroundEventOptions,
		Phase:   phase,
		Side:    o.Started,
		Title:   title,
		Summary: firstNonEmpty(o.Summary, "Listed options moved unusually."),
		At:      at.UTC(),
	}}
}

func unusualOptionsTitle(o AroundOptions) string {
	switch o.Started {
	case AroundOptionsStartedBefore:
		return "Unusual options before the move"
	case AroundOptionsStartedDuring:
		return "Unusual options during the move"
	default:
		return "Unusual options"
	}
}

func phaseOptions(p AroundPhase) bool {
	return p.Options != nil && p.Options.Complete
}

func optionsOIChange(p AroundPhase) float64 {
	if p.Options == nil {
		return 0
	}
	return p.Options.OIChangePct
}

func optionsOIDir(p AroundPhase) string {
	if p.Options == nil {
		return ""
	}
	return p.Options.OIDirection
}

func aroundOptionsOIPath(before, during, after AroundPhase) string {
	d := optionsOIDir(during)
	a := optionsOIDir(after)
	if d == CVDDirUp && a == CVDDirUp {
		return AroundPathContinued
	}
	if (d == CVDDirUp && a == CVDDirDown) || (d == CVDDirDown && a == CVDDirUp) {
		return AroundPathReversed
	}
	if d == CVDDirUp || d == CVDDirDown {
		return AroundPathFaded
	}
	_ = before
	return AroundPathChopped
}

func aroundOptionsOIChangeSummary(before, during, after AroundPhase, path string) string {
	switch path {
	case AroundPathContinued:
		return fmt.Sprintf("Listed options open interest kept %s after the move.", optionsOIDir(after))
	case AroundPathReversed:
		return "Listed options open interest reversed after the move."
	case AroundPathFaded:
		return fmt.Sprintf("Listed options open interest %s during the move, then flattened.", optionsOIDir(during))
	default:
		return fmt.Sprintf("Listed options OI: before %s%%, during %s%%, after %s%%.",
			FormatSignedPct(optionsOIChange(before)), FormatSignedPct(optionsOIChange(during)), FormatSignedPct(optionsOIChange(after)))
	}
}
