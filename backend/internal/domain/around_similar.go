package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	DefaultAroundSimilarLimit = 5
	MaxAroundSimilarLimit     = 10
	aroundSimilarMinScore     = 40.0
)

// AroundSimilarHit is a past move whose setup looks like the current tape.
type AroundSimilarHit struct {
	Move           AroundMove
	Similarity     float64
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
func MatchAroundSimilar(current AroundPhase, past []AroundMoveHit, limit int) []AroundSimilarHit {
	limit = ClampAroundSimilarLimit(limit)
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
		score, matches := aroundPhaseSimilarity(current, before)
		if score < aroundSimilarMinScore {
			continue
		}
		row := AroundSimilarHit{
			Move: hit.AroundMove, Similarity: score, Matches: matches, Before: before,
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
	head += fmt.Sprintf(" Closest: %s%% similar, then price %s%%.",
		formatFixed(top.Similarity, 0), FormatSignedPct(top.AfterReturnPct))
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

func aroundPhaseSimilarity(a, b AroundPhase) (float64, []string) {
	type feat struct {
		name   string
		score  float64
		weight float64
		match  bool
	}
	var feats []feat
	if a.Flow.TypicalKnown && b.Flow.TypicalKnown {
		s := closenessRatio(a.Flow.VolumeRatio, b.Flow.VolumeRatio)
		feats = append(feats, feat{"volume", s, 0.24, s >= 0.7})
	}
	feats = append(feats, feat{"price", closenessAbs(a.Price.ChangePct, b.Price.ChangePct, 3), 0.14, sameDir(a.Price.Direction, b.Price.Direction)})
	if a.Flow.BuySellKnown && b.Flow.BuySellKnown {
		s := closenessAbs(a.Flow.BuyShare*100, b.Flow.BuyShare*100, 20)
		feats = append(feats, feat{"takers", s, 0.14, a.Flow.Dominant != "" && a.Flow.Dominant == b.Flow.Dominant})
	}
	if a.Futures != nil && a.Futures.Complete && b.Futures != nil && b.Futures.Complete {
		s := closenessAbs(a.Futures.OIChangePct, b.Futures.OIChangePct, 6)
		feats = append(feats, feat{"oi", s, 0.24, sameDir(a.Futures.OIDirection, b.Futures.OIDirection)})
	}
	if a.Book != nil && a.Book.Complete && b.Book != nil && b.Book.Complete {
		sb := closenessSigned(a.Book.BidNotionalDelta, b.Book.BidNotionalDelta)
		sa := closenessSigned(a.Book.AskNotionalDelta, b.Book.AskNotionalDelta)
		book := (sb + sa) / 2
		feats = append(feats, feat{"book", book, 0.24, sb >= 0.65 && sa >= 0.65})
	}
	var wsum, ssum float64
	var matches []string
	for _, f := range feats {
		wsum += f.weight
		ssum += f.weight * f.score
		if f.match {
			matches = append(matches, f.name)
		}
	}
	if wsum <= 0 {
		return 0, nil
	}
	score := 100 * ssum / wsum
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score, matches
}

func explainAroundSimilarHit(h AroundSimilarHit) string {
	when := h.Move.At.UTC().Format("15:04")
	bits := fmt.Sprintf("%s%% similar to the setup at %s", formatFixed(h.Similarity, 0), when)
	if h.AfterDirection != "" {
		bits += fmt.Sprintf("; price then %s %s%%", h.AfterDirection, FormatSignedPct(h.AfterReturnPct))
	}
	if len(h.Matches) > 0 {
		bits += " (matched " + joinList(h.Matches) + ")"
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
