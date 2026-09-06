package domain

import (
	"fmt"
	"math"
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

	AroundOptionsWeightSingle  = "single"
	AroundOptionsWeightPair    = "pair"
	AroundOptionsWeightCluster = "cluster"

	// AroundOptionsLookbackPad is extra history so unusual-move typicals can form.
	AroundOptionsLookbackPad = 48 * time.Hour

	optionsLeadWindow = time.Hour
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
	StartedAt    time.Time
	Lead         time.Duration // how long before the price event the print began
	LeadLabel    string        // "about 45 minutes"
	Together     int           // unusual series count (OI / put-call / IV)
	Weight       string        // single | pair | cluster
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
	at := time.Time{}
	if before != nil {
		at = before.To
	}
	if bOpt != nil {
		MeasureAroundOptionsLead(bOpt, points, at)
		fillAroundOptionsWeight(bOpt)
	}
	if dOpt != nil {
		if dOpt.Started == AroundOptionsStartedBefore && !at.IsZero() {
			MeasureAroundOptionsLead(dOpt, points, at)
		}
		fillAroundOptionsWeight(dOpt)
	}
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

// MeasureAroundOptionsLead finds how long before `at` the unusual print began.
func MeasureAroundOptionsLead(o *AroundOptions, points []OptionsHistoryPoint, at time.Time) {
	if o == nil || at.IsZero() || o.Started == AroundOptionsStartedDuring {
		return
	}
	var earliest time.Time
	for _, topic := range optionsUnusualTopics(*o) {
		start, ok := optionsTopicBurstStart(points, at, topic)
		if !ok || !start.Before(at) {
			continue
		}
		if earliest.IsZero() || start.Before(earliest) {
			earliest = start
		}
	}
	if earliest.IsZero() {
		return
	}
	lead := at.UTC().Sub(earliest.UTC())
	if lead < time.Minute {
		return
	}
	o.StartedAt = earliest.UTC()
	o.Lead = lead
	o.LeadLabel = formatOptionsLead(lead)
}

func optionsUnusualTopics(o AroundOptions) []string {
	var out []string
	seen := map[string]bool{}
	add := func(topic string) {
		if topic == "" || seen[topic] {
			return
		}
		seen[topic] = true
		out = append(out, topic)
	}
	if o.HasOI && absFloat(o.OIChangePct) >= aroundOptionsOIFloor {
		add(OptionsMoveTopicOI)
	}
	if o.HasPutCall && absFloat(o.PutCallDelta) >= aroundOptionsPCRFloor {
		add(OptionsMoveTopicLean)
	}
	if (o.HasDVOL && absFloat(o.DVOLDelta) >= aroundOptionsDVOLFloor) || (o.HasAtmIV && absFloat(o.AtmIVDelta) >= aroundOptionsDVOLFloor) {
		add(OptionsMoveTopicVol)
	}
	for _, m := range o.Moves {
		add(m.Topic)
	}
	if o.Unusual && len(out) == 0 {
		add(OptionsMoveTopicOI)
	}
	return out
}

func fillAroundOptionsWeight(o *AroundOptions) {
	if o == nil {
		return
	}
	n := len(optionsUnusualTopics(*o))
	if n == 0 && o.Unusual {
		n = 1
	}
	o.Together = n
	switch {
	case n >= 3:
		o.Weight = AroundOptionsWeightCluster
	case n == 2:
		o.Weight = AroundOptionsWeightPair
	case n == 1:
		o.Weight = AroundOptionsWeightSingle
	default:
		o.Weight = ""
	}
}

func optionsTopicBurstStart(points []OptionsHistoryPoint, at time.Time, topic string) (time.Time, bool) {
	pts := OptionsHistoryUntil(points, at)
	if len(pts) < 2 {
		return time.Time{}, false
	}
	floor := optionsTopicFloor(topic)
	if floor <= 0 {
		return time.Time{}, false
	}
	var burst time.Time
	in := false
	for i := range pts {
		past := optionsHistoryAtOrBefore(pts, pts[i].Time.Add(-optionsLeadWindow), optionsLeadWindow+aroundOptionsSlack)
		if past.Time.IsZero() {
			continue
		}
		ch, ok := optionsSeriesChange(past, pts[i], topic)
		if !ok {
			continue
		}
		if math.Abs(ch) >= floor {
			if !in {
				burst = pts[i].Time
				in = true
			}
			continue
		}
		if in && pts[i].Time.Before(at.Add(-2*time.Minute)) {
			in = false
			burst = time.Time{}
		}
	}
	if !in || burst.IsZero() || !burst.Before(at) {
		return time.Time{}, false
	}
	return burst.UTC(), true
}

func optionsTopicFloor(topic string) float64 {
	switch topic {
	case OptionsMoveTopicOI:
		return aroundOptionsOIFloor
	case OptionsMoveTopicLean:
		return aroundOptionsPCRFloor
	case OptionsMoveTopicVol:
		return aroundOptionsDVOLFloor
	default:
		return 0
	}
}

func optionsSeriesChange(from, to OptionsHistoryPoint, topic string) (float64, bool) {
	switch topic {
	case OptionsMoveTopicOI:
		if !from.HasOI || !to.HasOI || from.OpenInterest <= 0 {
			return 0, false
		}
		return (to.OpenInterest - from.OpenInterest) / from.OpenInterest * 100, true
	case OptionsMoveTopicLean:
		if !from.HasPutCallOI || !to.HasPutCallOI {
			return 0, false
		}
		return to.PutCallOI - from.PutCallOI, true
	case OptionsMoveTopicVol:
		if from.HasDVOL && to.HasDVOL {
			return to.DVOL - from.DVOL, true
		}
		if from.HasAtmIV && to.HasAtmIV {
			return to.AtmIV - from.AtmIV, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func formatOptionsLead(d time.Duration) string {
	if d < 90*time.Second {
		return "about a minute"
	}
	if d < 90*time.Minute {
		m := int(d.Round(time.Minute) / time.Minute)
		if m <= 1 {
			return "about a minute"
		}
		return fmt.Sprintf("about %d minutes", m)
	}
	h := d.Hours()
	if h < 1.75 {
		return "about an hour"
	}
	if h < 36 {
		return fmt.Sprintf("about %.0f hours", math.Round(h))
	}
	return fmt.Sprintf("about %.0f days", math.Round(h/24))
}

// FormatOptionsLeadShort is 45m / 2h / 1d for the API wire.
func FormatOptionsLeadShort(d time.Duration) string {
	if d < 90*time.Second {
		return "1m"
	}
	if d < 90*time.Minute {
		m := int(d.Round(time.Minute) / time.Minute)
		if m < 1 {
			m = 1
		}
		return fmt.Sprintf("%dm", m)
	}
	h := int(math.Round(d.Hours()))
	if h < 1 {
		h = 1
	}
	if h < 36 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dd", int(math.Round(d.Hours()/24)))
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
	if (a.Together >= 2) != (b.Together >= 2) {
		score *= 0.75
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
		if o.Unusual {
			return unusualOptionsLeadSentence(o) + "."
		}
		return "Listed options were mostly steady in this window."
	}
	head := "Listed options: " + strings.Join(bits, "; ") + "."
	if o.Unusual {
		return unusualOptionsLeadSentence(o) + " — " + strings.TrimPrefix(head, "Listed options: ")
	}
	return head
}

func unusualOptionsLeadSentence(o AroundOptions) string {
	when := "Listed options were unusual"
	switch o.Started {
	case AroundOptionsStartedBefore:
		when = "Listed options were already unusual before the price move"
		if o.LeadLabel != "" {
			when = "Listed options were already unusual " + o.LeadLabel + " before the price move"
		}
	case AroundOptionsStartedDuring:
		when = "Listed options became unusual during the price move"
	}
	switch o.Weight {
	case AroundOptionsWeightCluster:
		return when + ", and open interest, put/call, and implied vol all moved together"
	case AroundOptionsWeightPair:
		return when + ", and two options series moved together"
	default:
		return when
	}
}

// AroundOptionsToEvents turns unusual options prints into tape events.
func AroundOptionsToEvents(o AroundOptions, phase string, at time.Time) []AroundEvent {
	if !o.Unusual {
		return nil
	}
	title := unusualOptionsTitle(o)
	score := optionsEventScore(o)
	when := at.UTC()
	if !o.StartedAt.IsZero() {
		when = o.StartedAt
	}
	if len(o.Moves) > 0 {
		out := make([]AroundEvent, 0, len(o.Moves))
		for _, m := range o.Moves {
			out = append(out, AroundEvent{
				Kind:    AroundEventOptions,
				Phase:   phase,
				Side:    firstNonEmpty(o.Started, m.Topic),
				Title:   title,
				Summary: firstNonEmpty(m.Detail, m.Title),
				At:      when,
				Score:   score,
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
		At:      when,
		Score:   score,
	}}
}

func unusualOptionsTitle(o AroundOptions) string {
	lead := ""
	if o.Started == AroundOptionsStartedBefore && o.Lead >= time.Minute {
		lead = " (" + FormatOptionsLeadShort(o.Lead) + " earlier)"
	}
	switch {
	case o.Weight == AroundOptionsWeightCluster && o.Started == AroundOptionsStartedBefore:
		return "Clustered unusual options before the move" + lead
	case o.Weight == AroundOptionsWeightCluster && o.Started == AroundOptionsStartedDuring:
		return "Clustered unusual options during the move"
	case o.Weight == AroundOptionsWeightPair && o.Started == AroundOptionsStartedBefore:
		return "Two unusual options series before the move" + lead
	case o.Weight == AroundOptionsWeightPair && o.Started == AroundOptionsStartedDuring:
		return "Two unusual options series during the move"
	case o.Started == AroundOptionsStartedBefore:
		return "Unusual options before the move" + lead
	case o.Started == AroundOptionsStartedDuring:
		return "Unusual options during the move"
	default:
		return "Unusual options"
	}
}

func optionsEventScore(o AroundOptions) int {
	n := o.Together
	if n <= 0 {
		n = 1
	}
	return n * 30
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
