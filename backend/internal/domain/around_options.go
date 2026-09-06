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

// OptionsAcrossWindow is listed OI / put-call / IV at the start and end of a phase.
func OptionsAcrossWindow(asset string, points []OptionsHistoryPoint, from, to time.Time) *AroundOptions {
	if len(points) == 0 || !to.After(from) {
		return nil
	}
	start := optionsHistoryAtOrBefore(points, from.UTC(), aroundOptionsSlack)
	end := optionsHistoryAtOrBefore(points, to.UTC(), aroundOptionsSlack)
	if start.Time.IsZero() && end.Time.IsZero() {
		return nil
	}
	out := AroundOptions{Asset: asset}
	if start.HasOI || end.HasOI {
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
	if start.HasPutCallOI || end.HasPutCallOI {
		out.HasPutCall = true
		out.PutCallFrom = start.PutCallOI
		out.PutCallTo = end.PutCallOI
		out.PutCallDelta = out.PutCallTo - out.PutCallFrom
		if start.HasPutCallOI {
			out.LeanFrom = optionsLean(start.PutCallOI)
		}
		if end.HasPutCallOI {
			out.LeanTo = optionsLean(end.PutCallOI)
		}
		out.Complete = true
	}
	if start.HasDVOL || end.HasDVOL {
		out.HasDVOL = true
		out.DVOLFrom = start.DVOL
		out.DVOLTo = end.DVOL
		out.DVOLDelta = out.DVOLTo - out.DVOLFrom
		out.Complete = true
	}
	if start.HasAtmIV || end.HasAtmIV {
		out.HasAtmIV = true
		out.AtmIVFrom = start.AtmIV
		out.AtmIVTo = end.AtmIV
		out.AtmIVDelta = out.AtmIVTo - out.AtmIVFrom
		out.Complete = true
	}
	if !out.Complete {
		return nil
	}
	if moves := DetectOptionsMoves(OptionsHistoryUntil(points, to), to); moves != nil {
		out.Moves = append([]OptionsMove(nil), moves.Findings...)
		out.Unusual = true
		out.Summary = moves.Summary
	}
	if !out.Unusual && optionsWindowLooksUnusual(out) {
		out.Unusual = true
	}
	if out.Summary == "" {
		out.Summary = explainAroundOptions(out)
	}
	return &out
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
		return "Listed options were mostly steady in this window."
	}
	head := "Listed options: " + strings.Join(bits, "; ") + "."
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
	if len(o.Moves) > 0 {
		out := make([]AroundEvent, 0, len(o.Moves))
		for _, m := range o.Moves {
			out = append(out, AroundEvent{
				Kind:    AroundEventOptions,
				Phase:   phase,
				Side:    m.Topic,
				Title:   m.Title,
				Summary: firstNonEmpty(m.Detail, m.Title),
				At:      at.UTC(),
			})
		}
		return out
	}
	return []AroundEvent{{
		Kind:    AroundEventOptions,
		Phase:   phase,
		Title:   "Unusual options",
		Summary: firstNonEmpty(o.Summary, "Listed options moved unusually before or during this window."),
		At:      at.UTC(),
	}}
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
