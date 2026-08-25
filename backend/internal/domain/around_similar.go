package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	DefaultAroundSimilarLimit = 5
	MaxAroundSimilarLimit     = 10
	aroundSimilarMinScore     = 40.0

	AroundSimilarFieldPrice  = "price"
	AroundSimilarFieldVolume = "volume"
	AroundSimilarFieldTakers = "takers"
	AroundSimilarFieldOI     = "oi"
	AroundSimilarFieldBook   = "book"
)

// AroundSimilarFields is which tape pieces to compare.
type AroundSimilarFields struct {
	Price  bool
	Volume bool
	Takers bool
	OI     bool
	Book   bool
}

// AroundSimilarFieldScore is one requested field on one match.
type AroundSimilarFieldScore struct {
	Name   string
	Used   bool
	Score  float64 // 0–100 closeness when used
	Weight float64
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
	MinReturnPct   float64
	AsOf           time.Time
	Current        AroundPhase
	Matches        []AroundSimilarHit
	UpAfter        int
	DownAfter      int
	MedianAfterPct float64
	Summary        string
	Note           string
}

// DefaultAroundSimilarFields compares every tape piece.
func DefaultAroundSimilarFields() AroundSimilarFields {
	return AroundSimilarFields{Price: true, Volume: true, Takers: true, OI: true, Book: true}
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
func MatchAroundSimilar(current AroundPhase, past []AroundMoveHit, limit int, fields AroundSimilarFields) []AroundSimilarHit {
	limit = ClampAroundSimilarLimit(limit)
	if !fields.Any() {
		fields = DefaultAroundSimilarFields()
	}
	if !current.Complete {
		return nil
	}
	out := make([]AroundSimilarHit, 0, len(past))
	for _, hit := range past {
		if !hit.At.IsZero() && !current.From.IsZero() && !hit.At.Before(current.From) {
			continue
		}
		before, ok := AroundReportBefore(hit.Around)
		if !ok {
			continue
		}
		after, afterOK := aroundReportDuring(hit.Around)
		sc := aroundPhaseSimilarity(current, before, fields)
		if sc.Similarity < aroundSimilarMinScore {
			continue
		}
		row := AroundSimilarHit{
			Move: hit.AroundMove, Similarity: sc.Similarity, Coverage: sc.Coverage,
			Compared: sc.Compared, Used: sc.Used, Missing: sc.Missing, Matches: sc.Matches,
			Before: before,
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
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Similarity == out[j].Similarity {
			return out[i].Move.At.After(out[j].Move.At)
		}
		return out[i].Similarity > out[j].Similarity
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ExplainAroundSimilarReport writes how similar past cases resolved.
func ExplainAroundSimilarReport(r AroundSimilarReport) string {
	name := prettyBase(r.Symbol)
	if name == "" {
		name = "This coin"
	}
	if len(r.Matches) == 0 {
		return name + ": no past important-move setup is similar enough to the current tape."
	}
	head := fmt.Sprintf("%s: %d similar past setup(s).", name, len(r.Matches))
	if r.UpAfter+r.DownAfter > 0 {
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

func aroundPhaseSimilarity(a, b AroundPhase, want AroundSimilarFields) aroundSimilarScore {
	type cand struct {
		name   string
		weight float64
		have   bool
		score  float64
		match  bool
	}
	var cands []cand
	if want.Volume {
		c := cand{name: AroundSimilarFieldVolume, weight: 0.24}
		if a.Flow.TypicalKnown && b.Flow.TypicalKnown {
			c.have = true
			c.score = closenessRatio(a.Flow.VolumeRatio, b.Flow.VolumeRatio)
			c.match = c.score >= 0.7
		}
		cands = append(cands, c)
	}
	if want.Price {
		c := cand{name: AroundSimilarFieldPrice, weight: 0.14, have: true}
		c.score = closenessAbs(a.Price.ChangePct, b.Price.ChangePct, 3)
		c.match = sameDir(a.Price.Direction, b.Price.Direction) || c.score >= 0.7
		cands = append(cands, c)
	}
	if want.Takers {
		c := cand{name: AroundSimilarFieldTakers, weight: 0.14}
		if a.Flow.BuySellKnown && b.Flow.BuySellKnown {
			c.have = true
			c.score = closenessAbs(a.Flow.BuyShare*100, b.Flow.BuyShare*100, 20)
			c.match = a.Flow.Dominant != "" && a.Flow.Dominant == b.Flow.Dominant
		}
		cands = append(cands, c)
	}
	if want.OI {
		c := cand{name: AroundSimilarFieldOI, weight: 0.24}
		if a.Futures != nil && a.Futures.Complete && b.Futures != nil && b.Futures.Complete {
			c.have = true
			c.score = closenessAbs(a.Futures.OIChangePct, b.Futures.OIChangePct, 6)
			c.match = sameDir(a.Futures.OIDirection, b.Futures.OIDirection)
		}
		cands = append(cands, c)
	}
	if want.Book {
		c := cand{name: AroundSimilarFieldBook, weight: 0.24}
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
	// Missing selected fields count as 0 so a thin overlap cannot look like 90%+.
	out.Similarity = 100 * ssum / selected
	out.Coverage = 100 * usedW / selected
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
	r.Summary = ExplainAroundSimilarReport(*r)
}
