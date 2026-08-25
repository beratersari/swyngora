package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultAroundSimilarLimit       = 5
	MaxAroundSimilarLimit           = 10
	DefaultAroundSimilarMinCoverage = 60.0
	aroundSimilarMinScore           = 40.0
	MaxAroundSimilarHorizons        = 8
	MaxAroundSimilarHorizon         = 24 * time.Hour

	AroundSimilarFieldPrice  = "price"
	AroundSimilarFieldVolume = "volume"
	AroundSimilarFieldTakers = "takers"
	AroundSimilarFieldOI     = "oi"
	AroundSimilarFieldBook   = "book"

	AroundSimilarHorizon15m = "15m"
	AroundSimilarHorizon1h  = "1h"
	AroundSimilarHorizon4h  = "4h"
)

// AroundSimilarHorizon is one forward window after a match.
type AroundSimilarHorizon struct {
	ID  string
	Dur time.Duration
}

// DefaultAroundSimilarHorizons is 15m, 1h, 4h when the caller does not pick times.
func DefaultAroundSimilarHorizons() []AroundSimilarHorizon {
	return []AroundSimilarHorizon{
		{AroundSimilarHorizon15m, 15 * time.Minute},
		{AroundSimilarHorizon1h, time.Hour},
		{AroundSimilarHorizon4h, 4 * time.Hour},
	}
}

// AroundSimilarFields is which tape pieces to compare.
type AroundSimilarFields struct {
	Price  bool
	Volume bool
	Takers bool
	OI     bool
	Book   bool
}

// AroundSimilarWeights is how much each selected field counts.
type AroundSimilarWeights struct {
	Price  float64
	Volume float64
	Takers float64
	OI     float64
	Book   float64
}

// AroundSimilarFieldScore is one requested field on one match.
type AroundSimilarFieldScore struct {
	Name   string
	Used   bool
	Score  float64 // 0–100 closeness when used
	Weight float64
}

// AroundSimilarScoreBand is one similarity range, e.g. 40–60.
type AroundSimilarScoreBand struct {
	From float64 // inclusive
	To   float64 // exclusive (last band includes 100)
	ID   string
}

// AroundSimilarBandStat is after-move stats for one similarity range.
type AroundSimilarBandStat struct {
	From       float64
	To         float64
	Label      string
	Sample     int
	Events     int
	Up         int
	Down       int
	AveragePct float64
	MedianPct  float64
}

// AroundSimilarHorizonStat is how price usually moved after similar setups.
type AroundSimilarHorizonStat struct {
	Horizon    string
	Interval   string // candle size used to measure this horizon
	Sample     int    // unique past events with enough tape for this horizon
	Events     int
	Up         int
	Down       int
	AveragePct float64
	MedianPct  float64
	Bands      []AroundSimilarBandStat
}

// DefaultAroundSimilarBands is 40–60, 60–80, 80–100.
func DefaultAroundSimilarBands() []AroundSimilarScoreBand {
	return []AroundSimilarScoreBand{
		{From: 40, To: 60, ID: "40-60"},
		{From: 60, To: 80, ID: "60-80"},
		{From: 80, To: 100, ID: "80-100"},
	}
}

// AroundSimilarHit is a past move whose setup looks like the current tape.
type AroundSimilarHit struct {
	Move           AroundMove
	Similarity     float64
	Coverage       float64 // 0–100 share of selected weight that actually had data
	Compared       []AroundSimilarFieldScore
	Used           []string
	Missing        []string
	Matches        []string
	Before         AroundPhase
	After          AroundPhase // the move itself (during)
	DataFrom       time.Time   // start of tape used for the comparison (pre-move)
	DataTo         time.Time   // end of tape used for the comparison (at move start)
	AfterReturnPct float64
	AfterDirection string
	Summary        string
}

// AroundSimilarReport ranks past setups against the current market.
type AroundSimilarReport struct {
	Symbol         string
	Exchange       string
	Lookback       string
	Window         string
	Interval       string
	Fields         []string
	Weights        []AroundSimilarFieldScore
	MinCoverage    float64
	MinReturnPct   float64
	Horizons       []string
	AsOf           time.Time
	Current        AroundPhase
	Matches        []AroundSimilarHit
	Skipped        []AroundSimilarHit
	Events         int // unique past events after collapsing overlap / same-move
	UpAfter        int
	DownAfter      int
	MedianAfterPct float64
	AfterHorizons  []AroundSimilarHorizonStat
	Summary        string
	Note           string
}

// DefaultAroundSimilarFields compares every tape piece.
func DefaultAroundSimilarFields() AroundSimilarFields {
	return AroundSimilarFields{Price: true, Volume: true, Takers: true, OI: true, Book: true}
}

// DefaultAroundSimilarWeights gives book and OI more say than volume.
func DefaultAroundSimilarWeights() AroundSimilarWeights {
	return AroundSimilarWeights{Price: 1, Volume: 1, Takers: 1.5, OI: 3, Book: 3}
}

// ParseAroundSimilarFields accepts a CSV of price, volume, takers, oi, book.
// Empty means all. Aliases: orderbook, open_interest.
func ParseAroundSimilarFields(raw string) (AroundSimilarFields, error) {
	s := strings.TrimSpace(raw)
	if s == "" || strings.EqualFold(s, "all") {
		return DefaultAroundSimilarFields(), nil
	}
	var out AroundSimilarFields
	for _, part := range strings.Split(s, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "", "all":
			continue
		case AroundSimilarFieldPrice:
			out.Price = true
		case AroundSimilarFieldVolume:
			out.Volume = true
		case AroundSimilarFieldTakers, "taker", "flow":
			out.Takers = true
		case AroundSimilarFieldOI, "open_interest", "openinterest", "open-interest":
			out.OI = true
		case AroundSimilarFieldBook, "orderbook", "order_book", "order-book":
			out.Book = true
		default:
			return AroundSimilarFields{}, fmt.Errorf("%w: fields must be price, volume, takers, oi, book", ErrInvalidArgument)
		}
	}
	if !out.Any() {
		return AroundSimilarFields{}, fmt.Errorf("%w: select at least one field (price, volume, takers, oi, book)", ErrInvalidArgument)
	}
	return out, nil
}

// Any reports whether at least one field is on.
func (f AroundSimilarFields) Any() bool {
	return f.Price || f.Volume || f.Takers || f.OI || f.Book
}

// IDs lists selected field names in a stable order.
func (f AroundSimilarFields) IDs() []string {
	out := make([]string, 0, 5)
	if f.Price {
		out = append(out, AroundSimilarFieldPrice)
	}
	if f.Volume {
		out = append(out, AroundSimilarFieldVolume)
	}
	if f.Takers {
		out = append(out, AroundSimilarFieldTakers)
	}
	if f.OI {
		out = append(out, AroundSimilarFieldOI)
	}
	if f.Book {
		out = append(out, AroundSimilarFieldBook)
	}
	return out
}

// ParseAroundSimilarWeights overlays book:3,oi:3,volume:1 (or book=3) on defaults
// for the selected fields.
func ParseAroundSimilarWeights(raw string, fields AroundSimilarFields) (AroundSimilarWeights, error) {
	if !fields.Any() {
		fields = DefaultAroundSimilarFields()
	}
	d := DefaultAroundSimilarWeights()
	out := AroundSimilarWeights{}
	if fields.Price {
		out.Price = d.Price
	}
	if fields.Volume {
		out.Volume = d.Volume
	}
	if fields.Takers {
		out.Takers = d.Takers
	}
	if fields.OI {
		out.OI = d.OI
	}
	if fields.Book {
		out.Book = d.Book
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out, nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, val, ok := splitWeightKV(part)
		if !ok {
			return AroundSimilarWeights{}, fmt.Errorf("%w: weights must look like book:3,oi:3,volume:1", ErrInvalidArgument)
		}
		w, err := parseSimilarWeight(val)
		if err != nil {
			return AroundSimilarWeights{}, err
		}
		switch name {
		case AroundSimilarFieldPrice:
			if !fields.Price {
				return AroundSimilarWeights{}, fmt.Errorf("%w: weight for price but price is not in fields", ErrInvalidArgument)
			}
			out.Price = w
		case AroundSimilarFieldVolume:
			if !fields.Volume {
				return AroundSimilarWeights{}, fmt.Errorf("%w: weight for volume but volume is not in fields", ErrInvalidArgument)
			}
			out.Volume = w
		case AroundSimilarFieldTakers, "taker", "flow":
			if !fields.Takers {
				return AroundSimilarWeights{}, fmt.Errorf("%w: weight for takers but takers is not in fields", ErrInvalidArgument)
			}
			out.Takers = w
		case AroundSimilarFieldOI, "open_interest", "openinterest", "open-interest":
			if !fields.OI {
				return AroundSimilarWeights{}, fmt.Errorf("%w: weight for oi but oi is not in fields", ErrInvalidArgument)
			}
			out.OI = w
		case AroundSimilarFieldBook, "orderbook", "order_book", "order-book":
			if !fields.Book {
				return AroundSimilarWeights{}, fmt.Errorf("%w: weight for book but book is not in fields", ErrInvalidArgument)
			}
			out.Book = w
		default:
			return AroundSimilarWeights{}, fmt.Errorf("%w: unknown weight field %q", ErrInvalidArgument, name)
		}
	}
	return out, nil
}

func splitWeightKV(part string) (name, val string, ok bool) {
	for _, sep := range []string{":", "="} {
		if i := strings.Index(part, sep); i > 0 {
			return strings.ToLower(strings.TrimSpace(part[:i])), strings.TrimSpace(part[i+1:]), true
		}
	}
	return "", "", false
}

func parseSimilarWeight(raw string) (float64, error) {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%w: weight must be a number > 0", ErrInvalidArgument)
	}
	return v, nil
}

// ParseAroundSimilarMinCoverage is 0–100. Empty uses 60. 0 means no floor.
func ParseAroundSimilarMinCoverage(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return DefaultAroundSimilarMinCoverage, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 || v > 100 || math.IsNaN(v) {
		return 0, fmt.Errorf("%w: minCoverage must be 0–100", ErrInvalidArgument)
	}
	return v, nil
}

// ParseAroundSimilarHorizons accepts a CSV of durations (15m,30m,1h,2h,6h).
// Empty uses 15m, 1h, 4h. Sorted, unique, max 8, each 1m–24h.
func ParseAroundSimilarHorizons(raw string) ([]AroundSimilarHorizon, error) {
	s := strings.TrimSpace(raw)
	if s == "" || strings.EqualFold(s, "default") {
		return DefaultAroundSimilarHorizons(), nil
	}
	seen := map[time.Duration]bool{}
	out := make([]AroundSimilarHorizon, 0, 4)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, dur, err := parseAroundSimilarHorizon(part)
		if err != nil {
			return nil, err
		}
		if seen[dur] {
			continue
		}
		seen[dur] = true
		out = append(out, AroundSimilarHorizon{ID: id, Dur: dur})
	}
	if len(out) == 0 {
		return DefaultAroundSimilarHorizons(), nil
	}
	if len(out) > MaxAroundSimilarHorizons {
		return nil, fmt.Errorf("%w: at most %d horizons", ErrInvalidArgument, MaxAroundSimilarHorizons)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dur < out[j].Dur })
	return out, nil
}

func parseAroundSimilarHorizon(raw string) (string, time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "")
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return "", 0, fmt.Errorf("%w: horizons must look like 15m,30m,1h,2h,6h", ErrInvalidArgument)
	}
	n, err := strconv.Atoi(s[:i])
	if err != nil || n <= 0 {
		return "", 0, fmt.Errorf("%w: horizon must be a number > 0 plus m/h/d", ErrInvalidArgument)
	}
	var dur time.Duration
	var id string
	switch s[i:] {
	case "m", "min", "mins", "minute", "minutes":
		dur = time.Duration(n) * time.Minute
		id = strconv.Itoa(n) + "m"
	case "h", "hr", "hrs", "hour", "hours":
		dur = time.Duration(n) * time.Hour
		id = strconv.Itoa(n) + "h"
	case "d", "day", "days":
		dur = time.Duration(n) * 24 * time.Hour
		id = strconv.Itoa(n) + "d"
	default:
		return "", 0, fmt.Errorf("%w: horizon unit must be m, h, or d (got %q)", ErrInvalidArgument, raw)
	}
	if dur < time.Minute || dur > MaxAroundSimilarHorizon {
		return "", 0, fmt.Errorf("%w: each horizon must be between 1m and 24h", ErrInvalidArgument)
	}
	return id, dur, nil
}

// AroundSimilarHorizonIDs lists horizon ids in duration order.
func AroundSimilarHorizonIDs(hs []AroundSimilarHorizon) []string {
	out := make([]string, 0, len(hs))
	for _, h := range hs {
		out = append(out, h.ID)
	}
	return out
}

// AroundSimilarHorizonMax is the longest requested forward window.
func AroundSimilarHorizonMax(hs []AroundSimilarHorizon) time.Duration {
	var max time.Duration
	for _, h := range hs {
		if h.Dur > max {
			max = h.Dur
		}
	}
	if max <= 0 {
		return 4 * time.Hour
	}
	return max
}

// AroundSimilarBarInterval is the candle size that matches a forward window
// (largest standard bar that is not longer than the horizon).
func AroundSimilarBarInterval(d time.Duration) string {
	if d <= 0 {
		return string(Interval15m)
	}
	cands := []CandleInterval{
		Interval1d, Interval12h, Interval8h, Interval6h, Interval4h,
		Interval2h, Interval1h, Interval30m, Interval15m, Interval5m,
		Interval3m, Interval1m,
	}
	for _, iv := range cands {
		if IntervalDuration(iv) <= d {
			return string(iv)
		}
	}
	return string(Interval1m)
}

// AroundSimilarBarIntervalFor is the finest bar needed to cover every horizon.
func AroundSimilarBarIntervalFor(hs []AroundSimilarHorizon) string {
	if len(hs) == 0 {
		hs = DefaultAroundSimilarHorizons()
	}
	min := hs[0].Dur
	for _, h := range hs[1:] {
		if h.Dur > 0 && h.Dur < min {
			min = h.Dur
		}
	}
	return AroundSimilarBarInterval(min)
}

func (w AroundSimilarWeights) of(name string) float64 {
	switch name {
	case AroundSimilarFieldPrice:
		return w.Price
	case AroundSimilarFieldVolume:
		return w.Volume
	case AroundSimilarFieldTakers:
		return w.Takers
	case AroundSimilarFieldOI:
		return w.OI
	case AroundSimilarFieldBook:
		return w.Book
	default:
		return 0
	}
}

// ListAroundSimilarWeights is selected fields with their weights, for the API.
func ListAroundSimilarWeights(fields AroundSimilarFields, w AroundSimilarWeights) []AroundSimilarFieldScore {
	out := make([]AroundSimilarFieldScore, 0, 5)
	for _, id := range fields.IDs() {
		out = append(out, AroundSimilarFieldScore{Name: id, Used: true, Weight: w.of(id)})
	}
	return out
}

// ClampAroundSimilarLimit bounds how many nearest cases we keep.
func ClampAroundSimilarLimit(n int) int {
	if n <= 0 {
		return DefaultAroundSimilarLimit
	}
	if n > MaxAroundSimilarLimit {
		return MaxAroundSimilarLimit
	}
	return n
}

// MatchAroundSimilar ranks past move setups against the current before-window.
// Cases below minCoverage are returned in skipped, not as normal matches.
func MatchAroundSimilar(current AroundPhase, past []AroundMoveHit, limit int, fields AroundSimilarFields, weights AroundSimilarWeights, minCoverage float64) (matches, skipped []AroundSimilarHit) {
	if !fields.Any() {
		fields = DefaultAroundSimilarFields()
	}
	if minCoverage < 0 {
		minCoverage = DefaultAroundSimilarMinCoverage
	}
	if !current.Complete {
		return nil, nil
	}
	var keep, drop []AroundSimilarHit
	for _, hit := range past {
		if !hit.At.IsZero() && !current.From.IsZero() && !hit.At.Before(current.From) {
			continue
		}
		before, ok := AroundReportBefore(hit.Around)
		if !ok {
			continue
		}
		before = clipAroundPhaseBeforeMove(before, hit.At)
		if !before.Complete || !before.To.After(before.From) {
			continue
		}
		after, afterOK := aroundReportDuring(hit.Around)
		sc := aroundPhaseSimilarity(current, before, fields, weights)
		row := AroundSimilarHit{
			Move: hit.AroundMove, Similarity: sc.Similarity, Coverage: sc.Coverage,
			Compared: sc.Compared, Used: sc.Used, Missing: sc.Missing, Matches: sc.Matches,
			Before: before, DataFrom: before.From, DataTo: before.To,
		}
		if afterOK {
			row.After = after
			row.AfterReturnPct = after.Price.ChangePct
			row.AfterDirection = after.Price.Direction
		} else {
			row.AfterReturnPct = hit.ReturnPct
			row.AfterDirection = hit.Direction
		}
		row.Summary = explainAroundSimilarHit(row)
		if minCoverage > 0 && sc.Coverage < minCoverage {
			drop = append(drop, row)
			continue
		}
		if sc.Similarity < aroundSimilarMinScore {
			continue
		}
		keep = append(keep, row)
	}
	keep = collapseAroundSimilarHits(keep)
	if limit > 0 {
		limit = ClampAroundSimilarLimit(limit)
		if len(keep) > limit {
			keep = keep[:limit]
		}
	}
	sort.SliceStable(drop, func(i, j int) bool {
		return drop[i].Coverage > drop[j].Coverage
	})
	return keep, drop
}

// collapseAroundSimilarHits keeps one hit per past price move (highest
// similarity). Only overlapping move ranges are the same event — shared
// setup/data windows do not merge different moves.
func collapseAroundSimilarHits(hits []AroundSimilarHit) []AroundSimilarHit {
	n := len(hits)
	if n < 2 {
		return hits
	}
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if aroundSimilarSameEvent(hits[i], hits[j]) {
				ri, rj := find(i), find(j)
				if ri != rj {
					parent[rj] = ri
				}
			}
		}
	}
	best := make(map[int]int, n)
	for i := range hits {
		r := find(i)
		prev, ok := best[r]
		if !ok || hits[i].Similarity > hits[prev].Similarity ||
			(hits[i].Similarity == hits[prev].Similarity && hits[i].Move.At.After(hits[prev].Move.At)) {
			best[r] = i
		}
	}
	out := make([]AroundSimilarHit, 0, len(best))
	for _, i := range best {
		out = append(out, hits[i])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Similarity == out[j].Similarity {
			return out[i].Move.At.After(out[j].Move.At)
		}
		return out[i].Similarity > out[j].Similarity
	})
	return out
}

func aroundSimilarSameEvent(a, b AroundSimilarHit) bool {
	aFrom, aTo := aroundSimilarMoveSpan(a.Move)
	bFrom, bTo := aroundSimilarMoveSpan(b.Move)
	if aFrom.IsZero() || bFrom.IsZero() {
		return false
	}
	// [from, to) overlap of the price move itself — not the setup window.
	return aFrom.Before(bTo) && bFrom.Before(aTo)
}

// clipAroundPhaseBeforeMove drops anything after the move started.
func clipAroundPhaseBeforeMove(p AroundPhase, moveAt time.Time) AroundPhase {
	if moveAt.IsZero() {
		return p
	}
	moveAt = moveAt.UTC()
	if p.To.After(moveAt) {
		p.To = moveAt
		p.Book = nil
		p.Futures = nil
		p.Price = AroundPrice{}
		p.Flow = AroundFlow{}
		p.Profile = nil
		p.Complete = p.To.After(p.From)
	}
	if len(p.Events) > 0 {
		evs := make([]AroundEvent, 0, len(p.Events))
		for _, ev := range p.Events {
			if ev.At.IsZero() || !ev.At.After(moveAt) {
				evs = append(evs, ev)
			}
		}
		p.Events = evs
	}
	if !p.To.After(p.From) {
		p.Complete = false
	}
	return p
}

func aroundSimilarMoveSpan(m AroundMove) (time.Time, time.Time) {
	from := m.At
	to := m.Until
	if to.IsZero() || !to.After(from) {
		return from, from.Add(15 * time.Minute)
	}
	// Until is the last bar open; include that bar.
	return from, to.Add(15 * time.Minute)
}

// ExplainAroundSimilarReport writes how similar past cases resolved.
func ExplainAroundSimilarReport(r AroundSimilarReport) string {
	name := prettyBase(r.Symbol)
	if name == "" {
		name = "This coin"
	}
	if len(r.Matches) == 0 {
		if n := len(r.Skipped); n > 0 {
			return fmt.Sprintf("%s: %d past setup(s) lacked enough selected data (min coverage %s%%).",
				name, n, formatFixed(r.MinCoverage, 0))
		}
		return name + ": no past important-move setup is similar enough to the current tape."
	}
	n := r.Events
	if n == 0 {
		n = len(r.Matches)
	}
	head := fmt.Sprintf("%s: %d unique past event(s).", name, n)
	if len(r.AfterHorizons) > 0 {
		for _, h := range r.AfterHorizons {
			ev := h.Events
			if ev == 0 {
				ev = h.Sample
			}
			if ev == 0 {
				head += fmt.Sprintf(" After %s: not enough data.", h.Horizon)
				continue
			}
			head += fmt.Sprintf(" After %s: rose %d, fell %d (avg %s%%, median %s%%, %d unique event(s)).",
				h.Horizon, h.Up, h.Down, FormatSignedPct(h.AveragePct), FormatSignedPct(h.MedianPct), ev)
			for _, b := range h.Bands {
				if b.Sample == 0 {
					continue
				}
				head += fmt.Sprintf(" Similarity %s: rose %d, fell %d (avg %s%%, n=%d).",
					b.Label, b.Up, b.Down, FormatSignedPct(b.AveragePct), b.Sample)
			}
		}
	} else if r.UpAfter+r.DownAfter > 0 {
		head += fmt.Sprintf(" After those cases price rose %d time(s) and fell %d time(s)", r.UpAfter, r.DownAfter)
		if r.MedianAfterPct != 0 {
			head += fmt.Sprintf(" (median %s%%)", FormatSignedPct(r.MedianAfterPct))
		}
		head += "."
	}
	top := r.Matches[0]
	head += fmt.Sprintf(" Closest: %s%% similar", formatFixed(top.Similarity, 0))
	if top.Coverage < 99 {
		head += fmt.Sprintf(" (%s%% of selected data present)", formatFixed(top.Coverage, 0))
	}
	if len(top.Used) > 0 {
		head += " using " + joinList(top.Used)
	}
	if len(top.Missing) > 0 {
		head += "; missing " + joinList(top.Missing)
	}
	head += fmt.Sprintf("; then price %s%%.", FormatSignedPct(top.AfterReturnPct))
	return head
}

func aroundReportDuring(r *AroundReport) (AroundPhase, bool) {
	if r == nil {
		return AroundPhase{}, false
	}
	if r.Combined != nil {
		if p, ok := AroundPhaseByID(*r.Combined, AroundPhaseDuring); ok && p.Complete {
			return p, true
		}
	}
	for _, v := range r.Venues {
		if p, ok := AroundPhaseByID(v, AroundPhaseDuring); ok && p.Complete {
			return p, true
		}
	}
	return AroundPhase{}, false
}

type aroundSimilarScore struct {
	Similarity float64
	Coverage   float64
	Compared   []AroundSimilarFieldScore
	Used       []string
	Missing    []string
	Matches    []string
}

func aroundPhaseSimilarity(a, b AroundPhase, want AroundSimilarFields, w AroundSimilarWeights) aroundSimilarScore {
	type cand struct {
		name   string
		weight float64
		have   bool
		score  float64
		match  bool
	}
	var cands []cand
	if want.Volume && w.of(AroundSimilarFieldVolume) > 0 {
		c := cand{name: AroundSimilarFieldVolume, weight: w.of(AroundSimilarFieldVolume)}
		if a.Flow.TypicalKnown && b.Flow.TypicalKnown {
			c.have = true
			c.score = closenessRatio(a.Flow.VolumeRatio, b.Flow.VolumeRatio)
			c.match = c.score >= 0.7
		}
		cands = append(cands, c)
	}
	if want.Price && w.of(AroundSimilarFieldPrice) > 0 {
		c := cand{name: AroundSimilarFieldPrice, weight: w.of(AroundSimilarFieldPrice), have: true}
		c.score = closenessAbs(a.Price.ChangePct, b.Price.ChangePct, 3)
		c.match = sameDir(a.Price.Direction, b.Price.Direction) || c.score >= 0.7
		cands = append(cands, c)
	}
	if want.Takers && w.of(AroundSimilarFieldTakers) > 0 {
		c := cand{name: AroundSimilarFieldTakers, weight: w.of(AroundSimilarFieldTakers)}
		if a.Flow.BuySellKnown && b.Flow.BuySellKnown {
			c.have = true
			c.score = closenessAbs(a.Flow.BuyShare*100, b.Flow.BuyShare*100, 20)
			c.match = a.Flow.Dominant != "" && a.Flow.Dominant == b.Flow.Dominant
		}
		cands = append(cands, c)
	}
	if want.OI && w.of(AroundSimilarFieldOI) > 0 {
		c := cand{name: AroundSimilarFieldOI, weight: w.of(AroundSimilarFieldOI)}
		if a.Futures != nil && a.Futures.Complete && b.Futures != nil && b.Futures.Complete {
			c.have = true
			c.score = closenessAbs(a.Futures.OIChangePct, b.Futures.OIChangePct, 6)
			c.match = sameDir(a.Futures.OIDirection, b.Futures.OIDirection)
		}
		cands = append(cands, c)
	}
	if want.Book && w.of(AroundSimilarFieldBook) > 0 {
		c := cand{name: AroundSimilarFieldBook, weight: w.of(AroundSimilarFieldBook)}
		if a.Book != nil && a.Book.Complete && b.Book != nil && b.Book.Complete {
			c.have = true
			sb := closenessSigned(a.Book.BidNotionalDelta, b.Book.BidNotionalDelta)
			sa := closenessSigned(a.Book.AskNotionalDelta, b.Book.AskNotionalDelta)
			c.score = (sb + sa) / 2
			c.match = sb >= 0.65 && sa >= 0.65
		}
		cands = append(cands, c)
	}
	var selected, usedW, ssum float64
	out := aroundSimilarScore{}
	for _, c := range cands {
		selected += c.weight
		row := AroundSimilarFieldScore{Name: c.name, Used: c.have, Weight: c.weight}
		if c.have {
			usedW += c.weight
			ssum += c.weight * c.score
			row.Score = c.score * 100
			out.Used = append(out.Used, c.name)
			if c.match {
				out.Matches = append(out.Matches, c.name)
			}
		} else {
			out.Missing = append(out.Missing, c.name)
		}
		out.Compared = append(out.Compared, row)
	}
	if selected <= 0 {
		return out
	}
	out.Coverage = 100 * usedW / selected
	if usedW <= 0 {
		return out
	}
	// Only fields that were actually compared contribute a score.
	out.Similarity = 100 * ssum / usedW
	if out.Similarity > 100 {
		out.Similarity = 100
	}
	if out.Similarity < 0 {
		out.Similarity = 0
	}
	return out
}

func explainAroundSimilarHit(h AroundSimilarHit) string {
	when := h.Move.At.UTC().Format("15:04")
	bits := fmt.Sprintf("%s%% similar to the setup at %s", formatFixed(h.Similarity, 0), when)
	if !h.DataFrom.IsZero() && h.DataTo.After(h.DataFrom) {
		bits += fmt.Sprintf(" (data %s–%s)", h.DataFrom.UTC().Format("15:04"), h.DataTo.UTC().Format("15:04"))
	}
	if h.Coverage < 99 {
		bits += fmt.Sprintf(" (%s%% of selected data present)", formatFixed(h.Coverage, 0))
	}
	if len(h.Used) > 0 {
		bits += " using " + joinList(h.Used)
	}
	if len(h.Missing) > 0 {
		bits += "; missing " + joinList(h.Missing)
	}
	if h.AfterDirection != "" {
		bits += fmt.Sprintf("; price then %s %s%%", h.AfterDirection, FormatSignedPct(h.AfterReturnPct))
	}
	return bits + "."
}

func closenessRatio(a, b float64) float64 {
	if a <= 0 && b <= 0 {
		return 1
	}
	if a <= 0 || b <= 0 {
		return 0
	}
	d := math.Abs(math.Log(a / b))
	return 1 - math.Min(1, d/math.Log(4))
}

func closenessAbs(a, b, scale float64) float64 {
	if scale <= 0 {
		return 0
	}
	return 1 - math.Min(1, math.Abs(a-b)/scale)
}

func closenessSigned(a, b float64) float64 {
	if a == 0 && b == 0 {
		return 1
	}
	if a == 0 || b == 0 {
		if math.Abs(a+b) < 1e-9 {
			return 1
		}
		return 0.35
	}
	if a*b < 0 {
		return 0.15
	}
	mag := closenessRatio(math.Abs(a), math.Abs(b))
	return 0.55 + 0.45*mag
}

func sameDir(a, b string) bool {
	return a != "" && a == b && a != CVDDirFlat
}

// FinishAroundSimilar fills after-counts and the summary on a report.
func FinishAroundSimilar(r *AroundSimilarReport) {
	if r == nil {
		return
	}
	vals := make([]float64, 0, len(r.Matches))
	for _, m := range r.Matches {
		vals = append(vals, m.AfterReturnPct)
		switch m.AfterDirection {
		case CVDDirUp:
			r.UpAfter++
		case CVDDirDown:
			r.DownAfter++
		}
	}
	r.MedianAfterPct = medianFloat(vals)
	if r.Events == 0 {
		r.Events = len(r.Matches)
	}
	r.Summary = ExplainAroundSimilarReport(*r)
}

type aroundHorizonReturn struct {
	pct float64
	sim float64
}

// SummarizeAroundSimilarHorizons averages price after similar setups at each horizon.
// A match is left out of a horizon when the tape does not reach that far.
// bars should use AroundSimilarBarIntervalFor(horizons) so short windows are not measured on 15m candles.
func SummarizeAroundSimilarHorizons(matches []AroundSimilarHit, bars []AroundBar, asOf time.Time, horizons []AroundSimilarHorizon) []AroundSimilarHorizonStat {
	if len(horizons) == 0 {
		horizons = DefaultAroundSimilarHorizons()
	}
	matches = collapseAroundSimilarHits(matches)
	bars = sortAroundBars(bars)
	barDur := aroundBarDuration(bars)
	iv := AroundSimilarBarIntervalFor(horizons)
	if barDur > 0 {
		iv = aroundBarIntervalID(barDur)
	}
	bands := DefaultAroundSimilarBands()
	out := make([]AroundSimilarHorizonStat, 0, len(horizons))
	for _, h := range horizons {
		st := AroundSimilarHorizonStat{Horizon: h.ID, Interval: AroundSimilarBarInterval(h.Dur)}
		if iv != "" {
			st.Interval = iv
		}
		rows := make([]aroundHorizonReturn, 0, len(matches))
		for _, m := range matches {
			start := m.Move.At
			if start.IsZero() {
				continue
			}
			startPx := aroundStartPrice(bars, start, m.Move.Open)
			if startPx <= 0 {
				continue
			}
			endPx, ok := aroundCloseByHorizon(bars, start, h.Dur, barDur, asOf)
			if !ok {
				continue
			}
			pct := (endPx - startPx) / startPx * 100
			rows = append(rows, aroundHorizonReturn{pct: pct, sim: m.Similarity})
		}
		fillAroundSimilarHorizonStat(&st, rows, bands)
		out = append(out, st)
	}
	return out
}

func fillAroundSimilarHorizonStat(st *AroundSimilarHorizonStat, rows []aroundHorizonReturn, bands []AroundSimilarScoreBand) {
	if st == nil {
		return
	}
	vals := make([]float64, 0, len(rows))
	for _, r := range rows {
		vals = append(vals, r.pct)
		switch changeDir(r.pct) {
		case CVDDirUp:
			st.Up++
		case CVDDirDown:
			st.Down++
		}
	}
	st.Sample = len(vals)
	st.Events = len(vals)
	st.AveragePct = averageFloat(vals)
	st.MedianPct = medianFloat(vals)
	st.Bands = make([]AroundSimilarBandStat, 0, len(bands))
	for i, b := range bands {
		bs := AroundSimilarBandStat{From: b.From, To: b.To, Label: b.ID}
		if bs.Label == "" {
			bs.Label = formatFixed(b.From, 0) + "-" + formatFixed(b.To, 0)
		}
		var bvals []float64
		last := i == len(bands)-1
		for _, r := range rows {
			if r.sim < b.From {
				continue
			}
			if last {
				if r.sim > b.To {
					continue
				}
			} else if r.sim >= b.To {
				continue
			}
			bvals = append(bvals, r.pct)
			switch changeDir(r.pct) {
			case CVDDirUp:
				bs.Up++
			case CVDDirDown:
				bs.Down++
			}
		}
		bs.Sample = len(bvals)
		bs.Events = len(bvals)
		bs.AveragePct = averageFloat(bvals)
		bs.MedianPct = medianFloat(bvals)
		st.Bands = append(st.Bands, bs)
	}
}

func aroundBarIntervalID(d time.Duration) string {
	return AroundSimilarBarInterval(d)
}

func sortAroundBars(bars []AroundBar) []AroundBar {
	out := append([]AroundBar(nil), bars...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

func aroundBarDuration(bars []AroundBar) time.Duration {
	for i := 1; i < len(bars); i++ {
		if d := bars[i].Time.Sub(bars[i-1].Time); d > 0 {
			return d
		}
	}
	return 15 * time.Minute
}

func aroundStartPrice(bars []AroundBar, start time.Time, fallback float64) float64 {
	var before float64
	for _, b := range bars {
		if !b.Time.Before(start) {
			if b.Time.Equal(start) && b.Open > 0 {
				return b.Open
			}
			break
		}
		if b.Close > 0 {
			before = b.Close
		}
	}
	if before > 0 {
		return before
	}
	if fallback > 0 {
		return fallback
	}
	return 0
}

func aroundCloseByHorizon(bars []AroundBar, start time.Time, horizon, barDur time.Duration, asOf time.Time) (float64, bool) {
	if barDur <= 0 {
		barDur = 15 * time.Minute
	}
	target := start.Add(horizon)
	if !asOf.IsZero() && asOf.Before(target) {
		return 0, false
	}
	var endPx float64
	var endAt time.Time
	found := false
	for _, b := range bars {
		closeAt := b.Time.Add(barDur)
		if closeAt.After(target) {
			continue
		}
		if b.Close > 0 {
			endPx = b.Close
			endAt = closeAt
			found = true
		}
	}
	if !found {
		return 0, false
	}
	slack := barDur / 2
	if slack < time.Minute {
		slack = time.Minute
	}
	if endAt.Before(target.Add(-slack)) {
		return 0, false
	}
	return endPx, true
}

func averageFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	return sum / float64(len(xs))
}
