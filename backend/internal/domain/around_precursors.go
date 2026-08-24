package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	AroundPrecursorMinSample        = 3
	AroundPrecursorMinShare         = 60.0
	DefaultAroundPrecursorsLookback = "7d"
)

// AroundPrecursorPattern is one condition seen often in the window before moves.
type AroundPrecursorPattern struct {
	Metric   string
	Label    string
	Side     string // up | down | both
	Hits     int
	Sample   int
	SharePct float64
	Median   float64
	Common   bool
	Summary  string
}

// AroundPrecursorReport is how the tape looked before many important moves.
type AroundPrecursorReport struct {
	Symbol       string
	Exchange     string
	Lookback     string
	Interval     string
	Direction    string
	MinReturnPct float64
	From         time.Time
	To           time.Time
	AsOf         time.Time
	UpMoves      int
	DownMoves    int
	Sampled      int
	Patterns     []AroundPrecursorPattern
	Moves        []AroundMove
	Summary      string
	Note         string
}

type precursorBucket struct {
	label  string
	hits   int
	sample int
	vals   []float64
}

// SummarizeAroundPrecursors compares the before-windows of scanned moves
// and keeps the conditions that show up often before ups, downs, or both.
func SummarizeAroundPrecursors(moves []AroundMoveHit) AroundPrecursorReport {
	out := AroundPrecursorReport{
		Patterns: []AroundPrecursorPattern{},
		Moves:    make([]AroundMove, 0, len(moves)),
	}
	// side -> metric -> bucket
	acc := map[string]map[string]*precursorBucket{
		CVDDirUp:   {},
		CVDDirDown: {},
	}
	for _, hit := range moves {
		out.Moves = append(out.Moves, hit.AroundMove)
		if hit.Direction == CVDDirUp {
			out.UpMoves++
		} else if hit.Direction == CVDDirDown {
			out.DownMoves++
		}
		before, ok := aroundReportBefore(hit.Around)
		if !ok {
			continue
		}
		out.Sampled++
		side := hit.Direction
		if side != CVDDirUp && side != CVDDirDown {
			continue
		}
		recordPrecursor(acc[side], "price_quiet", "price was quiet", before.Price.Direction == CVDDirFlat, before.Price.ChangePct)
		recordPrecursor(acc[side], "price_up", "price already rising", before.Price.Direction == CVDDirUp, before.Price.ChangePct)
		recordPrecursor(acc[side], "price_down", "price already falling", before.Price.Direction == CVDDirDown, before.Price.ChangePct)
		if before.Flow.TypicalKnown {
			elev := before.Flow.VolumeRatio >= 1.5 || before.Flow.VolumeGrade == VolumeSurgeElevated || before.Flow.VolumeGrade == VolumeSurgeHigh || before.Flow.VolumeGrade == VolumeSurgeExtreme
			recordPrecursor(acc[side], "volume_elevated", "volume was elevated", elev, before.Flow.VolumeRatio)
		}
		if before.Flow.BuySellKnown {
			recordPrecursor(acc[side], "takers_buy", "takers were buying", before.Flow.Dominant == TakerSideBuy, before.Flow.BuyShare*100)
			recordPrecursor(acc[side], "takers_sell", "takers were selling", before.Flow.Dominant == TakerSideSell, before.Flow.BuyShare*100)
		}
		if before.Futures != nil && before.Futures.Complete {
			recordPrecursor(acc[side], "oi_up", "open interest was rising", before.Futures.OIDirection == CVDDirUp, before.Futures.OIChangePct)
			recordPrecursor(acc[side], "oi_down", "open interest was falling", before.Futures.OIDirection == CVDDirDown, before.Futures.OIChangePct)
		}
		if before.Book != nil && before.Book.Complete {
			recordPrecursor(acc[side], "bid_pulled", "bid liquidity was pulled", before.Book.BidNotionalDelta < 0, before.Book.BidNotionalDelta)
			recordPrecursor(acc[side], "ask_pulled", "ask liquidity was pulled", before.Book.AskNotionalDelta < 0, before.Book.AskNotionalDelta)
		}
		recordPrecursor(acc[side], "sweep", "a liquidity sweep printed", countAroundKind(before.Events, AroundEventSweep) > 0, float64(countAroundKind(before.Events, AroundEventSweep)))
		recordPrecursor(acc[side], "absorption", "absorption printed", countAroundKind(before.Events, AroundEventAbsorption) > 0, float64(countAroundKind(before.Events, AroundEventAbsorption)))
	}

	var pats []AroundPrecursorPattern
	for _, side := range []string{CVDDirUp, CVDDirDown} {
		for metric, b := range acc[side] {
			if b == nil || b.sample < 2 {
				continue
			}
			share := float64(b.hits) / float64(b.sample) * 100
			if share < 50 {
				continue
			}
			p := AroundPrecursorPattern{
				Metric: metric, Label: b.label, Side: side,
				Hits: b.hits, Sample: b.sample, SharePct: share,
				Median: medianFloat(b.vals),
				Common: b.sample >= AroundPrecursorMinSample && share >= AroundPrecursorMinShare,
			}
			p.Summary = explainPrecursorPattern(p)
			pats = append(pats, p)
		}
	}
	pats = mergeBothSidePrecursors(pats)
	sort.SliceStable(pats, func(i, j int) bool {
		if pats[i].Common != pats[j].Common {
			return pats[i].Common
		}
		si, sj := pats[i].SharePct*float64(pats[i].Sample), pats[j].SharePct*float64(pats[j].Sample)
		if si == sj {
			return pats[i].Metric < pats[j].Metric
		}
		return si > sj
	})
	out.Patterns = pats
	out.Summary = ExplainAroundPrecursors(out)
	return out
}

// ExplainAroundPrecursors leads with the most common before-window reads.
func ExplainAroundPrecursors(r AroundPrecursorReport) string {
	name := prettyBase(r.Symbol)
	if name == "" {
		name = "This coin"
	}
	if r.Sampled == 0 {
		return name + ": not enough before-windows to compare those moves."
	}
	head := fmt.Sprintf("%s: compared the tape before %d up-move(s) and %d down-move(s).",
		name, r.UpMoves, r.DownMoves)
	commons := make([]string, 0, 3)
	for _, p := range r.Patterns {
		if !p.Common {
			continue
		}
		commons = append(commons, p.Summary)
		if len(commons) == 3 {
			break
		}
	}
	if len(commons) == 0 {
		if len(r.Patterns) > 0 {
			return head + " " + r.Patterns[0].Summary
		}
		return head + " No condition repeated often enough to call common."
	}
	return head + " " + joinList(commons)
}

func aroundReportBefore(r *AroundReport) (AroundPhase, bool) {
	if r == nil {
		return AroundPhase{}, false
	}
	if r.Combined != nil {
		if p, ok := AroundPhaseByID(*r.Combined, AroundPhaseBefore); ok && p.Complete {
			return p, true
		}
	}
	for _, v := range r.Venues {
		if p, ok := AroundPhaseByID(v, AroundPhaseBefore); ok && p.Complete {
			return p, true
		}
	}
	return AroundPhase{}, false
}

func recordPrecursor(m map[string]*precursorBucket, metric, label string, hit bool, val float64) {
	b := m[metric]
	if b == nil {
		b = &precursorBucket{label: label}
		m[metric] = b
	}
	b.sample++
	if hit {
		b.hits++
	}
	b.vals = append(b.vals, val)
}

func explainPrecursorPattern(p AroundPrecursorPattern) string {
	side := "up-moves"
	if p.Side == CVDDirDown {
		side = "down-moves"
	} else if p.Side == "both" {
		side = "up and down moves"
	}
	return fmt.Sprintf("Before %s, %s in %d of %d (%s%%).",
		side, p.Label, p.Hits, p.Sample, formatFixed(p.SharePct, 0))
}

func mergeBothSidePrecursors(in []AroundPrecursorPattern) []AroundPrecursorPattern {
	type key struct{ metric, side string }
	idx := map[key]AroundPrecursorPattern{}
	for _, p := range in {
		idx[key{p.Metric, p.Side}] = p
	}
	var out []AroundPrecursorPattern
	seen := map[string]bool{}
	for _, p := range in {
		if seen[p.Metric] {
			continue
		}
		up, uOK := idx[key{p.Metric, CVDDirUp}]
		dn, dOK := idx[key{p.Metric, CVDDirDown}]
		if uOK && dOK && up.Common && dn.Common {
			both := AroundPrecursorPattern{
				Metric: p.Metric, Label: p.Label, Side: "both",
				Hits: up.Hits + dn.Hits, Sample: up.Sample + dn.Sample,
				Common: true,
			}
			if both.Sample > 0 {
				both.SharePct = float64(both.Hits) / float64(both.Sample) * 100
			}
			both.Summary = explainPrecursorPattern(both)
			out = append(out, both)
			seen[p.Metric] = true
			continue
		}
		out = append(out, p)
	}
	return out
}
