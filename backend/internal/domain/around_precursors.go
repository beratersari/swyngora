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

// AroundPrecursorCombo is several conditions that fired in the same before-window.
type AroundPrecursorCombo struct {
	Metrics      []string
	Labels       []string
	Title        string
	UpHits       int
	DownHits     int
	UpSample     int
	DownSample   int
	Hits         int
	Sample       int
	UpSharePct   float64
	DownSharePct float64
	SharePct     float64 // overall hits / sample — used for common
	Lean         string  // up | down | both | mixed
	Common       bool
	Summary      string
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
	Combos       []AroundPrecursorCombo
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

type precursorFlags struct {
	side  string
	avail map[string]bool
	hit   map[string]bool
	vals  map[string]float64
}

const maxAroundPrecursorCombos = 10

var precursorMetricLabels = map[string]string{
	"price_quiet":     "price was quiet",
	"price_up":        "price already rising",
	"price_down":      "price already falling",
	"volume_elevated": "volume was elevated",
	"takers_buy":      "takers were buying",
	"takers_sell":     "takers were selling",
	"oi_up":           "open interest was rising",
	"oi_down":         "open interest was falling",
	"bid_pulled":      "bid liquidity was pulled",
	"ask_pulled":      "ask liquidity was pulled",
	"sweep":           "a liquidity sweep printed",
	"absorption":      "absorption printed",
}

var precursorExclusive = [][]string{
	{"price_quiet", "price_up", "price_down"},
	{"takers_buy", "takers_sell"},
	{"oi_up", "oi_down"},
}

// SummarizeAroundPrecursors compares the before-windows of scanned moves
// and keeps the conditions that show up often before ups, downs, or both.
func SummarizeAroundPrecursors(moves []AroundMoveHit) AroundPrecursorReport {
	out := AroundPrecursorReport{
		Patterns: []AroundPrecursorPattern{},
		Combos:   []AroundPrecursorCombo{},
		Moves:    make([]AroundMove, 0, len(moves)),
	}
	// side -> metric -> bucket
	acc := map[string]map[string]*precursorBucket{
		CVDDirUp:   {},
		CVDDirDown: {},
	}
	var rows []precursorFlags
	for _, hit := range moves {
		out.Moves = append(out.Moves, hit.AroundMove)
		if hit.Direction == CVDDirUp {
			out.UpMoves++
		} else if hit.Direction == CVDDirDown {
			out.DownMoves++
		}
		before, ok := AroundReportBefore(hit.Around)
		if !ok {
			continue
		}
		out.Sampled++
		side := hit.Direction
		if side != CVDDirUp && side != CVDDirDown {
			continue
		}
		fl := flagsFromBefore(side, before)
		rows = append(rows, fl)
		for metric, label := range precursorMetricLabels {
			if !fl.avail[metric] {
				continue
			}
			recordPrecursor(acc[side], metric, label, fl.hit[metric], fl.vals[metric])
		}
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
	out.Combos = findAroundPrecursorCombos(rows)
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
	for _, c := range r.Combos {
		if !c.Common {
			continue
		}
		commons = append(commons, c.Summary)
		if len(commons) == 2 {
			break
		}
	}
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
		if len(r.Combos) > 0 {
			return head + " " + r.Combos[0].Summary
		}
		if len(r.Patterns) > 0 {
			return head + " " + r.Patterns[0].Summary
		}
		return head + " No condition repeated often enough to call common."
	}
	return head + " " + joinList(commons)
}

// AroundReportBefore returns the complete before-window from combined or a venue.
func AroundReportBefore(r *AroundReport) (AroundPhase, bool) {
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

func flagsFromBefore(side string, before AroundPhase) precursorFlags {
	fl := precursorFlags{
		side:  side,
		avail: map[string]bool{},
		hit:   map[string]bool{},
		vals:  map[string]float64{},
	}
	set := func(metric string, avail, hit bool, val float64) {
		if !avail {
			return
		}
		fl.avail[metric] = true
		fl.hit[metric] = hit
		fl.vals[metric] = val
	}
	set("price_quiet", true, before.Price.Direction == CVDDirFlat, before.Price.ChangePct)
	set("price_up", true, before.Price.Direction == CVDDirUp, before.Price.ChangePct)
	set("price_down", true, before.Price.Direction == CVDDirDown, before.Price.ChangePct)
	if before.Flow.TypicalKnown {
		elev := before.Flow.VolumeRatio >= 1.5 || before.Flow.VolumeGrade == VolumeSurgeElevated || before.Flow.VolumeGrade == VolumeSurgeHigh || before.Flow.VolumeGrade == VolumeSurgeExtreme
		set("volume_elevated", true, elev, before.Flow.VolumeRatio)
	}
	if before.Flow.BuySellKnown {
		set("takers_buy", true, before.Flow.Dominant == TakerSideBuy, before.Flow.BuyShare*100)
		set("takers_sell", true, before.Flow.Dominant == TakerSideSell, before.Flow.BuyShare*100)
	}
	if before.Futures != nil && before.Futures.Complete {
		set("oi_up", true, before.Futures.OIDirection == CVDDirUp, before.Futures.OIChangePct)
		set("oi_down", true, before.Futures.OIDirection == CVDDirDown, before.Futures.OIChangePct)
	}
	if before.Book != nil && before.Book.Complete {
		set("bid_pulled", true, before.Book.BidNotionalDelta < 0, before.Book.BidNotionalDelta)
		set("ask_pulled", true, before.Book.AskNotionalDelta < 0, before.Book.AskNotionalDelta)
	}
	sw := countAroundKind(before.Events, AroundEventSweep)
	ab := countAroundKind(before.Events, AroundEventAbsorption)
	set("sweep", true, sw > 0, float64(sw))
	set("absorption", true, ab > 0, float64(ab))
	return fl
}

func findAroundPrecursorCombos(rows []precursorFlags) []AroundPrecursorCombo {
	hitCount := map[string]int{}
	for _, r := range rows {
		for m, ok := range r.hit {
			if ok {
				hitCount[m]++
			}
		}
	}
	cands := make([]string, 0, len(precursorMetricLabels))
	for m := range precursorMetricLabels {
		if hitCount[m] >= 2 {
			cands = append(cands, m)
		}
	}
	sort.Strings(cands)
	var sets [][]string
	for i := 0; i < len(cands); i++ {
		for j := i + 1; j < len(cands); j++ {
			pair := []string{cands[i], cands[j]}
			if precursorComboExclusive(pair) {
				continue
			}
			sets = append(sets, pair)
			for k := j + 1; k < len(cands); k++ {
				trip := []string{cands[i], cands[j], cands[k]}
				if precursorComboExclusive(trip) {
					continue
				}
				sets = append(sets, trip)
			}
		}
	}
	out := make([]AroundPrecursorCombo, 0, 16)
	for _, metrics := range sets {
		c := scorePrecursorCombo(rows, metrics)
		if c.Sample < AroundPrecursorMinSample {
			continue
		}
		if c.SharePct < 50 && c.UpSharePct < AroundPrecursorMinShare && c.DownSharePct < AroundPrecursorMinShare {
			continue
		}
		// Common is overall frequency, not “often on one side only”.
		c.Common = c.Sample >= AroundPrecursorMinSample && c.SharePct >= AroundPrecursorMinShare
		if !c.Common && c.SharePct < 40 && c.UpSharePct < AroundPrecursorMinShare && c.DownSharePct < AroundPrecursorMinShare {
			continue
		}
		c.Summary = explainPrecursorCombo(c)
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Common != out[j].Common {
			return out[i].Common
		}
		if len(out[i].Metrics) != len(out[j].Metrics) {
			return len(out[i].Metrics) > len(out[j].Metrics)
		}
		si := maxFloat(out[i].UpSharePct, out[i].DownSharePct) * float64(out[i].Sample)
		sj := maxFloat(out[j].UpSharePct, out[j].DownSharePct) * float64(out[j].Sample)
		if si == sj {
			return out[i].Title < out[j].Title
		}
		return si > sj
	})
	if len(out) > maxAroundPrecursorCombos {
		out = out[:maxAroundPrecursorCombos]
	}
	return out
}

func scorePrecursorCombo(rows []precursorFlags, metrics []string) AroundPrecursorCombo {
	labels := make([]string, 0, len(metrics))
	for _, m := range metrics {
		labels = append(labels, precursorMetricLabels[m])
	}
	c := AroundPrecursorCombo{Metrics: metrics, Labels: labels, Title: joinList(labels)}
	for _, r := range rows {
		avail := true
		hit := true
		for _, m := range metrics {
			if !r.avail[m] {
				avail = false
				break
			}
			if !r.hit[m] {
				hit = false
			}
		}
		if !avail {
			continue
		}
		c.Sample++
		if r.side == CVDDirUp {
			c.UpSample++
		} else {
			c.DownSample++
		}
		if !hit {
			continue
		}
		c.Hits++
		if r.side == CVDDirUp {
			c.UpHits++
		} else {
			c.DownHits++
		}
	}
	if c.UpSample > 0 {
		c.UpSharePct = float64(c.UpHits) / float64(c.UpSample) * 100
	}
	if c.DownSample > 0 {
		c.DownSharePct = float64(c.DownHits) / float64(c.DownSample) * 100
	}
	if c.Sample > 0 {
		c.SharePct = float64(c.Hits) / float64(c.Sample) * 100
	}
	c.Lean = precursorComboLean(c.UpSharePct, c.DownSharePct, c.UpSample, c.DownSample)
	return c
}

func precursorComboLean(upShare, downShare float64, upN, downN int) string {
	const gap = 15.0
	upOK, downOK := upN >= 2, downN >= 2
	if upOK && downOK {
		if upShare >= AroundPrecursorMinShare && downShare >= AroundPrecursorMinShare && absFloat(upShare-downShare) < gap {
			return "both"
		}
		if upShare >= downShare+gap {
			return CVDDirUp
		}
		if downShare >= upShare+gap {
			return CVDDirDown
		}
	}
	if upOK && upShare >= AroundPrecursorMinShare && (!downOK || upShare >= downShare) {
		return CVDDirUp
	}
	if downOK && downShare >= AroundPrecursorMinShare && (!upOK || downShare >= upShare) {
		return CVDDirDown
	}
	return "mixed"
}

func explainPrecursorCombo(c AroundPrecursorCombo) string {
	head := c.Title + " together"
	up := fmt.Sprintf("%d of %d up-moves (%s%%)", c.UpHits, c.UpSample, formatFixed(c.UpSharePct, 0))
	down := fmt.Sprintf("%d of %d down-moves (%s%%)", c.DownHits, c.DownSample, formatFixed(c.DownSharePct, 0))
	switch {
	case c.UpSample > 0 && c.DownSample > 0 && c.Lean == CVDDirUp:
		return fmt.Sprintf("%s before %s vs %s — more before increases.", head, up, down)
	case c.UpSample > 0 && c.DownSample > 0 && c.Lean == CVDDirDown:
		return fmt.Sprintf("%s before %s vs %s — more before drops.", head, up, down)
	case c.Lean == "both":
		return fmt.Sprintf("%s before %s and %s.", head, up, down)
	case c.UpSample > 0 && c.DownSample == 0:
		return fmt.Sprintf("%s before %s.", head, up)
	case c.DownSample > 0 && c.UpSample == 0:
		return fmt.Sprintf("%s before %s.", head, down)
	default:
		return fmt.Sprintf("%s before %s vs %s.", head, up, down)
	}
}

func precursorComboExclusive(metrics []string) bool {
	for _, group := range precursorExclusive {
		n := 0
		for _, m := range metrics {
			for _, g := range group {
				if m == g {
					n++
				}
			}
		}
		if n > 1 {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
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
