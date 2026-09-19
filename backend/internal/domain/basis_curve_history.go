package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	BasisCurveHistoryWindow7d = "7d"
	BasisCurveHistoryLookback = 7 * 24 * time.Hour
	BasisCurveHistoryMaxMoves = 8

	BasisCurveChangeFlipped = "flipped"
	BasisCurveChangeSteeper = "steeper"
	BasisCurveChangeFlatter = "flatter"
	BasisCurveChangeStable  = "stable"

	BasisCurveMoveFlipped   = "flipped"
	BasisCurveMoveSteepened = "steepened"
	BasisCurveMoveFlattened = "flattened"

	// 2 bp of near mark to call a window steeper/flatter.
	basisCurveWindowMovePP = 0.02
	// 8 bp vs ~4h earlier to call out a steepen/flatten event.
	basisCurveEventPP   = 0.08
	basisCurvePairSlack = 2 * time.Minute
)

// BasisCurveHistoryWindows is 1h / 4h / 24h / 7d of near-vs-far spread.
var BasisCurveHistoryWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{SnapshotWindow1h, time.Hour},
	{SnapshotWindow4h, 4 * time.Hour},
	{SnapshotWindow24h, 24 * time.Hour},
	{BasisCurveHistoryWindow7d, BasisCurveHistoryLookback},
}

// BasisCurvePoint is near vs far at one time (from listed-contract mark klines).
type BasisCurvePoint struct {
	Time      time.Time
	NearMark  float64
	FarMark   float64
	Spread    float64
	SpreadPct float64
	Shape     string
	Slope     string
}

// BasisCurveWindow is live spread vs a lookback.
type BasisCurveWindow struct {
	Window        string
	PastSpreadPct float64
	ChangePct     float64 // now − past, percentage points
	PastShape     string
	Shape         string
	Change        string // flipped | steeper | flatter | stable
	Complete      bool
	SampleTime    time.Time
}

// BasisCurveMove is when the curve flipped side or got much steeper/flatter.
type BasisCurveMove struct {
	Time      time.Time
	Kind      string
	FromShape string
	ToShape   string
	SpreadPct float64
	ChangePct float64
	Summary   string
}

type curveMark struct {
	t  time.Time
	px float64
}

// AttachBasisCurveHistory fills curve history, windows, and moves from contract mark klines.
func AttachBasisCurveHistory(c BasisCurve, q BasisQuote) BasisCurve {
	if !c.Complete || c.Near == nil || c.Far == nil {
		return c
	}
	now := q.Time
	if now.IsZero() {
		now = time.Now().UTC()
	}
	pts := pairCurveMarks(contractMarks(q, c.Near.Symbol), contractMarks(q, c.Far.Symbol))
	if c.Near.Mark > 0 && c.Far.Mark > 0 {
		live := curvePointAt(now, c.Near.Mark, c.Far.Mark)
		if len(pts) == 0 || pts[len(pts)-1].Time.Before(now.Add(-30*time.Minute)) {
			pts = append(pts, live)
		} else {
			pts[len(pts)-1] = live
		}
	}
	c.History = pts
	c.Windows = basisCurveWindows(pts, now, c.SpreadPct, c.Shape)
	c.Moves = detectBasisCurveMoves(pts)
	c.Summary = appendCurveHistorySummary(c.Summary, c.Windows, c.Moves)
	return c
}

func contractMarks(q BasisQuote, symbol string) []PriceSample {
	want := strings.ToUpper(strings.TrimSpace(symbol))
	for _, c := range q.Contracts {
		if strings.ToUpper(strings.TrimSpace(c.Symbol)) == want {
			return c.History
		}
	}
	return nil
}

func pairCurveMarks(near, far []PriceSample) []BasisCurvePoint {
	n := make([]curveMark, 0, len(near))
	for _, p := range near {
		if p.Price > 0 && !p.Time.IsZero() {
			n = append(n, curveMark{t: p.Time.UTC(), px: p.Price})
		}
	}
	sort.SliceStable(n, func(i, j int) bool { return n[i].t.Before(n[j].t) })
	out := make([]BasisCurvePoint, 0, len(far))
	for _, f := range far {
		if f.Price <= 0 || f.Time.IsZero() {
			continue
		}
		np, ok := nearestMark(n, f.Time.UTC(), basisCurvePairSlack)
		if !ok {
			continue
		}
		out = append(out, curvePointAt(f.Time.UTC(), np, f.Price))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

func nearestMark(n []curveMark, at time.Time, slack time.Duration) (float64, bool) {
	bestPx := 0.0
	bestDt := time.Duration(0)
	found := false
	for _, p := range n {
		dt := p.t.Sub(at)
		if dt < 0 {
			dt = -dt
		}
		if dt > slack {
			continue
		}
		if !found || dt < bestDt {
			bestPx, bestDt, found = p.px, dt, true
		}
	}
	return bestPx, found
}

func curvePointAt(at time.Time, near, far float64) BasisCurvePoint {
	spread := far - near
	pct := 0.0
	if near > 0 {
		pct = spread / near * 100
	}
	shape, slope, _ := basisCurveShape(pct)
	return BasisCurvePoint{
		Time: at.UTC(), NearMark: near, FarMark: far,
		Spread: spread, SpreadPct: pct, Shape: shape, Slope: slope,
	}
}

func basisCurveShape(spreadPct float64) (shape, slope, richer string) {
	switch {
	case spreadPct > basisFlatPct:
		return BasisCurveContango, BasisSlopeUp, BasisRicherFar
	case spreadPct < -basisFlatPct:
		return BasisCurveBackwardation, BasisSlopeDown, BasisRicherNear
	default:
		return BasisKindFlat, BasisSlopeFlat, BasisRicherNone
	}
}

func basisCurveWindows(hist []BasisCurvePoint, now time.Time, nowPct float64, nowShape string) []BasisCurveWindow {
	out := make([]BasisCurveWindow, 0, len(BasisCurveHistoryWindows))
	for _, w := range BasisCurveHistoryWindows {
		row := BasisCurveWindow{Window: w.ID, Shape: nowShape, Change: BasisCurveChangeStable}
		past, ok := findCurveSample(hist, now.Add(-w.Dur), curveHistorySlack(w.Dur))
		if ok && past.NearMark > 0 {
			row.PastSpreadPct = past.SpreadPct
			row.ChangePct = nowPct - past.SpreadPct
			row.PastShape = past.Shape
			row.Change = basisCurveChange(nowPct, past.SpreadPct, nowShape, past.Shape)
			row.Complete = true
			row.SampleTime = past.Time
		}
		out = append(out, row)
	}
	return out
}

func curveHistorySlack(window time.Duration) time.Duration {
	// History is 1h mark klines, so 1h windows need ~one bar of slack.
	if window >= 24*time.Hour {
		return 3 * time.Hour
	}
	return 90 * time.Minute
}

func findCurveSample(hist []BasisCurvePoint, target time.Time, slack time.Duration) (BasisCurvePoint, bool) {
	var best BasisCurvePoint
	found := false
	for _, p := range hist {
		if p.Time.IsZero() || p.Time.After(target) {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best = p
			found = true
		}
	}
	if !found {
		return BasisCurvePoint{}, false
	}
	return best, !best.Time.Before(target.Add(-slack))
}

func basisCurveChange(nowPct, pastPct float64, nowShape, pastShape string) string {
	if curveSide(nowShape) != "" && curveSide(pastShape) != "" && curveSide(nowShape) != curveSide(pastShape) {
		return BasisCurveChangeFlipped
	}
	dAbs := math.Abs(nowPct) - math.Abs(pastPct)
	switch {
	case dAbs > basisCurveWindowMovePP:
		return BasisCurveChangeSteeper
	case dAbs < -basisCurveWindowMovePP:
		return BasisCurveChangeFlatter
	default:
		return BasisCurveChangeStable
	}
}

func curveSide(shape string) string {
	switch shape {
	case BasisCurveContango, BasisCurveBackwardation:
		return shape
	default:
		return ""
	}
}

func detectBasisCurveMoves(pts []BasisCurvePoint) []BasisCurveMove {
	if len(pts) < 2 {
		return nil
	}
	var out []BasisCurveMove
	lastSide := ""
	for _, p := range pts {
		side := curveSide(p.Shape)
		if side == "" {
			continue
		}
		if lastSide != "" && lastSide != side {
			out = append(out, BasisCurveMove{
				Time: p.Time, Kind: BasisCurveMoveFlipped,
				FromShape: lastSide, ToShape: side, SpreadPct: p.SpreadPct,
				Summary: fmt.Sprintf("Curve flipped from %s to %s (%s%% far vs near).",
					lastSide, side, FormatSignedPct(p.SpreadPct)),
			})
		}
		lastSide = side
	}
	if steep, ok := steepestCurveMove(pts, true); ok {
		out = append(out, steep)
	}
	if flat, ok := steepestCurveMove(pts, false); ok {
		out = append(out, flat)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	if len(out) > BasisCurveHistoryMaxMoves {
		out = out[:BasisCurveHistoryMaxMoves]
	}
	return out
}

func steepestCurveMove(pts []BasisCurvePoint, steeper bool) (BasisCurveMove, bool) {
	bestI := -1
	bestD := 0.0
	for i := range pts {
		past, ok := findCurveSample(pts, pts[i].Time.Add(-4*time.Hour), 90*time.Minute)
		if !ok {
			continue
		}
		d := math.Abs(pts[i].SpreadPct) - math.Abs(past.SpreadPct)
		if steeper {
			if d > bestD {
				bestD, bestI = d, i
			}
		} else if d < bestD {
			bestD, bestI = d, i
		}
	}
	need := basisCurveEventPP
	if !steeper {
		need = -basisCurveEventPP
	}
	if bestI < 0 {
		return BasisCurveMove{}, false
	}
	if steeper && bestD < need {
		return BasisCurveMove{}, false
	}
	if !steeper && bestD > need {
		return BasisCurveMove{}, false
	}
	p := pts[bestI]
	kind := BasisCurveMoveSteepened
	verb := "steeper"
	if !steeper {
		kind = BasisCurveMoveFlattened
		verb = "flatter"
	}
	return BasisCurveMove{
		Time: p.Time, Kind: kind, ToShape: p.Shape, SpreadPct: p.SpreadPct, ChangePct: bestD,
		Summary: fmt.Sprintf("Curve became %s over ~4h (%s pp, now %s at %s%%).",
			verb, FormatSignedPct(bestD), p.Shape, FormatSignedPct(p.SpreadPct)),
	}, true
}

func appendCurveHistorySummary(base string, windows []BasisCurveWindow, moves []BasisCurveMove) string {
	var bits []string
	for _, id := range []string{SnapshotWindow24h, SnapshotWindow4h, BasisCurveHistoryWindow7d} {
		for _, w := range windows {
			if w.Window != id || !w.Complete {
				continue
			}
			switch w.Change {
			case BasisCurveChangeFlipped:
				bits = append(bits, fmt.Sprintf("Over %s it flipped from %s to %s.", w.Window, w.PastShape, w.Shape))
			case BasisCurveChangeSteeper:
				bits = append(bits, fmt.Sprintf("Over %s the curve got steeper (%s pp).", w.Window, FormatSignedPct(w.ChangePct)))
			case BasisCurveChangeFlatter:
				bits = append(bits, fmt.Sprintf("Over %s the curve flattened (%s pp).", w.Window, FormatSignedPct(w.ChangePct)))
			}
		}
	}
	if len(bits) == 0 && len(moves) > 0 && moves[0].Kind == BasisCurveMoveFlipped {
		bits = append(bits, moves[0].Summary)
	}
	if len(bits) == 0 {
		return base
	}
	if strings.TrimSpace(base) == "" {
		return strings.Join(bits, " ")
	}
	return strings.TrimSpace(base) + " " + strings.Join(bits, " ")
}
