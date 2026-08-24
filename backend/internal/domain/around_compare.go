package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	AroundCompareMetricPrice       = "price" // price level at the start of the move
	AroundCompareMetricMove        = "move"  // during net %
	AroundCompareMetricRange       = "range" // during high–low %
	AroundCompareMetricVolume      = "volume"
	AroundCompareMetricVolumeRatio = "volumeRatio"
	AroundCompareMetricDelta       = "delta"
	AroundCompareMetricPOC         = "poc"
	AroundCompareMetricBookMid     = "bookMid" // stored mid at the two times
	AroundCompareMetricBookBid     = "bookBid" // bid liquidity change during each move
	AroundCompareMetricBookAsk     = "bookAsk"
	AroundCompareMetricOI          = "oi"       // OI at the start of each move
	AroundCompareMetricOIChange    = "oiChange" // OI % during each move
	AroundCompareMetricFunding     = "funding"
	AroundCompareMetricLongPct     = "longPct"
	AroundCompareMetricLongLiq     = "longLiq"
	AroundCompareMetricShortLiq    = "shortLiq"
	AroundCompareMetricSweeps      = "sweeps"
	AroundCompareMetricAbsorption  = "absorption"
)

// AroundCompareDelta is to − from for one number.
type AroundCompareDelta struct {
	Metric    string
	Phase     string
	From      float64
	To        float64
	Change    float64
	ChangePct float64
	Direction string
	Summary   string
}

// AroundComparePhase is how one before/during/after window differed.
type AroundComparePhase struct {
	Phase   string
	From    AroundPhase
	To      AroundPhase
	Deltas  []AroundCompareDelta
	Summary string
}

// AroundCompareVenue is one exchange's (or combined) two-move diff.
type AroundCompareVenue struct {
	Exchange Exchange
	Symbol   string
	Phases   []AroundComparePhase
	State    []AroundCompareDelta // tape at the two event times
	Book     *AroundBook          // stored book at fromAt vs toAt
	Summary  string
	Error    string
}

// AroundCompareReport is the API result.
type AroundCompareReport struct {
	Symbol   string
	Exchange string
	FromAt   time.Time
	ToAt     time.Time
	Window   string
	During   string
	AsOf     time.Time
	FromMove *AroundReport
	ToMove   *AroundReport
	Venues   []AroundCompareVenue
	Combined *AroundCompareVenue
	Summary  string
	Note     string
}

// CompareAroundReports diffs two around-the-move tapes for the same coin.
func CompareAroundReports(from, to AroundReport) AroundCompareReport {
	out := AroundCompareReport{
		Symbol:   firstNonEmpty(to.Symbol, from.Symbol),
		Exchange: firstNonEmpty(to.Exchange, from.Exchange),
		FromAt:   from.At, ToAt: to.At,
		Window:   firstNonEmpty(to.Window, from.Window),
		During:   firstNonEmpty(to.During, from.During),
		AsOf:     to.AsOf,
		FromMove: cloneAroundReport(from),
		ToMove:   cloneAroundReport(to),
		Venues:   []AroundCompareVenue{},
	}
	if out.AsOf.IsZero() {
		out.AsOf = from.AsOf
	}
	seen := map[Exchange]struct{}{}
	for _, fv := range from.Venues {
		tv, ok := findAroundVenue(to.Venues, fv.Exchange)
		if !ok {
			out.Venues = append(out.Venues, AroundCompareVenue{
				Exchange: fv.Exchange, Symbol: fv.Symbol,
				Error:   "second time has no tape on this venue",
				Summary: "second time has no tape on this venue",
			})
			continue
		}
		seen[fv.Exchange] = struct{}{}
		out.Venues = append(out.Venues, CompareAroundVenues(fv, tv))
	}
	for _, tv := range to.Venues {
		if _, ok := seen[tv.Exchange]; ok {
			continue
		}
		out.Venues = append(out.Venues, AroundCompareVenue{
			Exchange: tv.Exchange, Symbol: tv.Symbol,
			Error:   "first time has no tape on this venue",
			Summary: "first time has no tape on this venue",
		})
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if from.Combined != nil && to.Combined != nil {
		c := CompareAroundVenues(*from.Combined, *to.Combined)
		out.Combined = &c
	}
	out.Summary = ExplainAroundCompare(out)
	return out
}

// CompareAroundVenues diffs matching phases and the tape at each event time.
func CompareAroundVenues(from, to AroundVenue) AroundCompareVenue {
	out := AroundCompareVenue{
		Exchange: to.Exchange, Symbol: firstNonEmpty(to.Symbol, from.Symbol),
		Phases: []AroundComparePhase{}, State: []AroundCompareDelta{},
	}
	if from.Error != "" && to.Error != "" {
		out.Error = to.Error
		out.Summary = to.Error
		return out
	}
	if from.Error != "" {
		out.Error = "first time: " + from.Error
		out.Summary = out.Error
		return out
	}
	if to.Error != "" {
		out.Error = "second time: " + to.Error
		out.Summary = out.Error
		return out
	}
	for _, id := range []string{AroundPhaseBefore, AroundPhaseDuring, AroundPhaseAfter} {
		a, aOK := AroundPhaseByID(from, id)
		b, bOK := AroundPhaseByID(to, id)
		if !aOK && !bOK {
			continue
		}
		out.Phases = append(out.Phases, DiffAroundPhases(id, a, b))
	}
	if dFrom, ok1 := AroundPhaseByID(from, AroundPhaseDuring); ok1 {
		if dTo, ok2 := AroundPhaseByID(to, AroundPhaseDuring); ok2 {
			out.State = compareAroundState(dFrom, dTo)
		}
	}
	out.Summary = ExplainAroundCompareVenue(out)
	return out
}

// DiffAroundPhases lists how one window differed between the two moves.
func DiffAroundPhases(phase string, from, to AroundPhase) AroundComparePhase {
	out := AroundComparePhase{Phase: phase, From: from, To: to, Deltas: []AroundCompareDelta{}}
	if from.Complete || to.Complete {
		out.Deltas = append(out.Deltas,
			qtyDelta(AroundCompareMetricMove, phase, from.Price.ChangePct, to.Price.ChangePct, true),
			qtyDelta(AroundCompareMetricRange, phase, from.Price.RangePct, to.Price.RangePct, true),
			qtyDelta(AroundCompareMetricVolume, phase, from.Flow.Volume, to.Flow.Volume, false),
		)
		if from.Flow.TypicalKnown || to.Flow.TypicalKnown {
			out.Deltas = append(out.Deltas, qtyDelta(AroundCompareMetricVolumeRatio, phase, from.Flow.VolumeRatio, to.Flow.VolumeRatio, false))
		}
		if from.Flow.BuySellKnown || to.Flow.BuySellKnown {
			out.Deltas = append(out.Deltas, qtyDelta(AroundCompareMetricDelta, phase, from.Flow.Delta, to.Flow.Delta, false))
		}
		if (from.Profile != nil && from.Profile.POC > 0) || (to.Profile != nil && to.Profile.POC > 0) {
			out.Deltas = append(out.Deltas, qtyDelta(AroundCompareMetricPOC, phase, profilePOC(from), profilePOC(to), false))
		}
		if bookComplete(from.Book) || bookComplete(to.Book) {
			out.Deltas = append(out.Deltas,
				qtyDelta(AroundCompareMetricBookBid, phase, bookBid(from.Book), bookBid(to.Book), false),
				qtyDelta(AroundCompareMetricBookAsk, phase, bookAsk(from.Book), bookAsk(to.Book), false),
			)
		}
		if futuresComplete(from.Futures) || futuresComplete(to.Futures) {
			out.Deltas = append(out.Deltas,
				qtyDelta(AroundCompareMetricOIChange, phase, futuresOIPct(from.Futures), futuresOIPct(to.Futures), true),
				qtyDelta(AroundCompareMetricLongLiq, phase, futuresLongLiq(from.Futures), futuresLongLiq(to.Futures), false),
				qtyDelta(AroundCompareMetricShortLiq, phase, futuresShortLiq(from.Futures), futuresShortLiq(to.Futures), false),
			)
		}
		out.Deltas = append(out.Deltas,
			qtyDelta(AroundCompareMetricSweeps, phase, float64(countAroundKind(from.Events, AroundEventSweep)), float64(countAroundKind(to.Events, AroundEventSweep)), false),
			qtyDelta(AroundCompareMetricAbsorption, phase, float64(countAroundKind(from.Events, AroundEventAbsorption)), float64(countAroundKind(to.Events, AroundEventAbsorption)), false),
		)
	}
	out.Summary = explainAroundComparePhase(out)
	return out
}

// AroundPhaseByID finds before, during, or after on a venue.
func AroundPhaseByID(v AroundVenue, id string) (AroundPhase, bool) {
	for _, p := range v.Phases {
		if p.Phase == id {
			return p, true
		}
	}
	return AroundPhase{}, false
}

// ExplainAroundCompare prefers combined, then the first venue.
func ExplainAroundCompare(r AroundCompareReport) string {
	name := prettyBase(r.Symbol)
	head := fmt.Sprintf("%s from %s to %s.",
		name, r.FromAt.UTC().Format("15:04"), r.ToAt.UTC().Format("15:04"))
	if r.Combined != nil && r.Combined.Summary != "" && r.Combined.Error == "" {
		return head + " " + r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Summary != "" && v.Error == "" {
			return head + " " + v.Summary
		}
	}
	for _, v := range r.Venues {
		if v.Summary != "" {
			return head + " " + v.Summary
		}
	}
	return head + " Not enough tape to compare those times."
}

// ExplainAroundCompareVenue leads with the during-move contrast.
func ExplainAroundCompareVenue(v AroundCompareVenue) string {
	if v.Error != "" {
		return v.Error
	}
	var during AroundComparePhase
	for _, p := range v.Phases {
		if p.Phase == AroundPhaseDuring {
			during = p
		}
	}
	if during.Phase == "" {
		return "No during-window to compare."
	}
	bits := make([]string, 0, 6)
	if d, ok := findCompareDelta(during.Deltas, AroundCompareMetricMove); ok {
		bits = append(bits, fmt.Sprintf("move %s%% vs %s%% (%s)",
			FormatSignedPct(d.From), FormatSignedPct(d.To), d.Direction))
	}
	if d, ok := findCompareDelta(during.Deltas, AroundCompareMetricVolume); ok && (d.From > 0 || d.To > 0) {
		bits = append(bits, fmt.Sprintf("volume %s vs %s", formatQty(d.From), formatQty(d.To)))
	}
	if d, ok := findCompareDelta(during.Deltas, AroundCompareMetricVolumeRatio); ok && (d.From > 0 || d.To > 0) {
		bits = append(bits, fmt.Sprintf("%.1fx vs %.1fx typical", d.From, d.To))
	}
	if d, ok := findCompareDelta(v.State, AroundCompareMetricPrice); ok && d.From > 0 && d.To > 0 {
		bits = append(bits, fmt.Sprintf("price level %s → %s", formatQty(d.From), formatQty(d.To)))
	}
	if d, ok := findCompareDelta(v.State, AroundCompareMetricBookMid); ok && (d.From > 0 || d.To > 0) {
		bits = append(bits, fmt.Sprintf("book mid %s", FormatSignedPct(d.ChangePct)))
	}
	if d, ok := findCompareDelta(v.State, AroundCompareMetricOI); ok && (d.From > 0 || d.To > 0) {
		bits = append(bits, fmt.Sprintf("OI %s", FormatSignedPct(d.ChangePct)))
	}
	if len(bits) == 0 {
		return "No clear difference between those two moves."
	}
	return joinList(bits) + "."
}

func compareAroundState(from, to AroundPhase) []AroundCompareDelta {
	out := []AroundCompareDelta{
		qtyDelta(AroundCompareMetricPrice, "", from.Price.Open, to.Price.Open, false),
	}
	if bookComplete(from.Book) || bookComplete(to.Book) {
		out = append(out, qtyDelta(AroundCompareMetricBookMid, "", bookFromMid(from.Book), bookFromMid(to.Book), false))
	}
	if futuresComplete(from.Futures) || futuresComplete(to.Futures) {
		out = append(out,
			qtyDelta(AroundCompareMetricOI, "", futuresOIFrom(from.Futures), futuresOIFrom(to.Futures), false),
			qtyDelta(AroundCompareMetricFunding, "", futuresFundingFrom(from.Futures), futuresFundingFrom(to.Futures), true),
			qtyDelta(AroundCompareMetricLongPct, "", futuresLongFrom(from.Futures), futuresLongFrom(to.Futures), true),
		)
	}
	return out
}

func qtyDelta(metric, phase string, from, to float64, alreadyPct bool) AroundCompareDelta {
	out := AroundCompareDelta{Metric: metric, Phase: phase, From: from, To: to}
	out.Change = to - from
	if alreadyPct {
		out.ChangePct = out.Change
	} else if from != 0 {
		out.ChangePct = out.Change / absFloat(from) * 100
	} else if to != 0 {
		out.ChangePct = 0
	}
	out.Direction = changeDir(out.ChangePct)
	if alreadyPct && out.Change == 0 {
		out.Direction = CVDDirFlat
	}
	out.Summary = explainAroundDelta(out, alreadyPct)
	return out
}

func explainAroundDelta(d AroundCompareDelta, alreadyPct bool) string {
	label := aroundCompareLabel(d.Metric)
	if alreadyPct {
		return fmt.Sprintf("%s %s%% vs %s%% (%s).",
			label, FormatSignedPct(d.From), FormatSignedPct(d.To), d.Direction)
	}
	return fmt.Sprintf("%s %s vs %s (%s).",
		label, formatQty(d.From), formatQty(d.To), d.Direction)
}

func explainAroundComparePhase(p AroundComparePhase) string {
	if d, ok := findCompareDelta(p.Deltas, AroundCompareMetricMove); ok && p.Phase == AroundPhaseDuring {
		return fmt.Sprintf("During: %s%% vs %s%%.", FormatSignedPct(d.From), FormatSignedPct(d.To))
	}
	if d, ok := findCompareDelta(p.Deltas, AroundCompareMetricVolume); ok {
		return fmt.Sprintf("%s volume %s vs %s.", p.Phase, formatQty(d.From), formatQty(d.To))
	}
	return p.Phase + ": mixed."
}

func aroundCompareLabel(metric string) string {
	switch metric {
	case AroundCompareMetricPrice:
		return "Price"
	case AroundCompareMetricMove:
		return "Move"
	case AroundCompareMetricRange:
		return "Range"
	case AroundCompareMetricVolume:
		return "Volume"
	case AroundCompareMetricVolumeRatio:
		return "Volume vs typical"
	case AroundCompareMetricDelta:
		return "Taker delta"
	case AroundCompareMetricPOC:
		return "POC"
	case AroundCompareMetricBookMid:
		return "Book mid"
	case AroundCompareMetricBookBid:
		return "Bid liquidity change"
	case AroundCompareMetricBookAsk:
		return "Ask liquidity change"
	case AroundCompareMetricOI:
		return "Open interest"
	case AroundCompareMetricOIChange:
		return "OI change"
	case AroundCompareMetricFunding:
		return "Funding"
	case AroundCompareMetricLongPct:
		return "Long %"
	case AroundCompareMetricLongLiq:
		return "Long liquidations"
	case AroundCompareMetricShortLiq:
		return "Short liquidations"
	case AroundCompareMetricSweeps:
		return "Sweeps"
	case AroundCompareMetricAbsorption:
		return "Absorption"
	default:
		return metric
	}
}

func findCompareDelta(in []AroundCompareDelta, metric string) (AroundCompareDelta, bool) {
	for _, d := range in {
		if d.Metric == metric {
			return d, true
		}
	}
	return AroundCompareDelta{}, false
}

func findAroundVenue(venues []AroundVenue, ex Exchange) (AroundVenue, bool) {
	for _, v := range venues {
		if v.Exchange == ex {
			return v, true
		}
	}
	return AroundVenue{}, false
}

func cloneAroundReport(r AroundReport) *AroundReport {
	cp := r
	return &cp
}

func profilePOC(p AroundPhase) float64 {
	if p.Profile == nil {
		return 0
	}
	return p.Profile.POC
}

func bookComplete(b *AroundBook) bool {
	return b != nil && b.Complete
}

func bookBid(b *AroundBook) float64 {
	if b == nil {
		return 0
	}
	return b.BidNotionalDelta
}

func bookAsk(b *AroundBook) float64 {
	if b == nil {
		return 0
	}
	return b.AskNotionalDelta
}

func bookFromMid(b *AroundBook) float64 {
	if b == nil {
		return 0
	}
	return b.FromMid
}

func futuresComplete(f *AroundFutures) bool {
	return f != nil && f.Complete
}

func futuresOIPct(f *AroundFutures) float64 {
	if f == nil {
		return 0
	}
	return f.OIChangePct
}

func futuresOIFrom(f *AroundFutures) float64 {
	if f == nil {
		return 0
	}
	return f.OIFrom
}

func futuresFundingFrom(f *AroundFutures) float64 {
	if f == nil {
		return 0
	}
	return f.FundingFrom
}

func futuresLongFrom(f *AroundFutures) float64 {
	if f == nil {
		return 0
	}
	return f.LongPctFrom
}

func futuresLongLiq(f *AroundFutures) float64 {
	if f == nil {
		return 0
	}
	return f.LongLiq
}

func futuresShortLiq(f *AroundFutures) float64 {
	if f == nil {
		return 0
	}
	return f.ShortLiq
}
