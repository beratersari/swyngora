package domain

import (
	"math"
	"sort"
)

const portfolioCorrNote = "How the open coins moved together over the last day, from 5-minute percent changes on the venue with the largest line in each coin. Together is the size-weighted average correlation, from -1 to +1. Swing is the size of a day built from those 5-minute moves. Apart is that same mix if the coins had moved independently. A short notional is the opposite of a long. A coin without enough shared minutes is named and left out, and is not treated as zero. This is not a forecast. Nothing is traded. Not financial advice."

const (
	// PortfolioCorrWindow is the same day window the coin correlation read uses.
	PortfolioCorrWindow = CorrWindow24h
	// PortfolioCorrInterval is the bar size for that day window.
	PortfolioCorrInterval = "5m"
)

// PortfolioCorrInput is one coin's signed notional and its percent returns.
// Returns must already share one clock with the other coins. Positive notional
// is long. Negative notional is short.
type PortfolioCorrInput struct {
	Asset    string
	Notional float64
	Returns  []float64
}

// PortfolioCorrHolding is one coin inside the read.
type PortfolioCorrHolding struct {
	Asset    string  `json:"asset"`
	Notional float64 `json:"notional"`
	Side     string  `json:"side"`
}

// PortfolioCorrPair is how two coins moved together.
type PortfolioCorrPair struct {
	Left     string   `json:"left"`
	Right    string   `json:"right"`
	Corr     *float64 `json:"corr,omitempty"`
	Relation string   `json:"relation,omitempty"`
	Samples  int      `json:"samples"`
	Complete bool     `json:"complete"`
}

// PortfolioCorrelation is the open book's correlation and the day swing.
// Add is set when the caller asked what one extra signed notional would do.
type PortfolioCorrelation struct {
	Window   string                 `json:"window"`
	Interval string                 `json:"interval"`
	Gross    float64                `json:"gross"`
	Holdings []PortfolioCorrHolding `json:"holdings"`
	Pairs    []PortfolioCorrPair    `json:"pairs"`
	Together *float64               `json:"together,omitempty"`
	SwingPct *float64               `json:"swingPct,omitempty"`
	ApartPct *float64               `json:"apartPct,omitempty"`
	Samples  int                    `json:"samples"`
	Missing  []string               `json:"missing"`
	Note     string                 `json:"note"`
	Add      *PortfolioCorrAdd      `json:"add,omitempty"`
}

// PortfolioCorrAdd is the book after one extra signed notional.
// Complete is false when that coin cannot join the shared minutes.
// The before numbers stay on PortfolioCorrelation.
type PortfolioCorrAdd struct {
	Asset    string   `json:"asset"`
	Notional float64  `json:"notional"`
	Side     string   `json:"side"`
	Together *float64 `json:"together,omitempty"`
	SwingPct *float64 `json:"swingPct,omitempty"`
	ApartPct *float64 `json:"apartPct,omitempty"`
	Samples  int      `json:"samples"`
	Complete bool     `json:"complete"`
}

// PortfolioCorrReady reports whether a shared return list is long enough.
func PortfolioCorrReady(samples int) bool {
	return samples >= corrMinSamples
}

// PortfolioCorrSpec is the 24h 5-minute window shared with price correlation.
func PortfolioCorrSpec() CorrWindowSpec {
	for _, spec := range CorrelationWindows {
		if spec.ID == CorrWindow24h {
			return spec
		}
	}
	return CorrWindowSpec{ID: CorrWindow24h, Interval: Interval5m, Bars: 289}
}

// BuildPortfolioCorrelation folds signed notionals and aligned returns into
// one book. A short return list leaves the swing blank. It does not invent a
// correlation of zero.
func BuildPortfolioCorrelation(in []PortfolioCorrInput) PortfolioCorrelation {
	out := PortfolioCorrelation{
		Window:   PortfolioCorrWindow,
		Interval: PortfolioCorrInterval,
		Holdings: []PortfolioCorrHolding{},
		Pairs:    []PortfolioCorrPair{},
		Missing:  []string{},
		Note:     portfolioCorrNote,
	}
	used := make([]PortfolioCorrInput, 0, len(in))
	for _, row := range in {
		if row.Asset == "" || math.Abs(row.Notional) <= PositionEpsilon || math.IsNaN(row.Notional) {
			continue
		}
		used = append(used, row)
	}
	if len(used) == 0 {
		return out
	}
	n := len(used[0].Returns)
	for _, row := range used {
		if len(row.Returns) != n {
			for _, h := range used {
				out.Missing = append(out.Missing, h.Asset)
			}
			return out
		}
	}
	out.Samples = n
	var gross float64
	for _, row := range used {
		gross += math.Abs(row.Notional)
		out.Holdings = append(out.Holdings, PortfolioCorrHolding{
			Asset: row.Asset, Notional: row.Notional, Side: corrSide(row.Notional),
		})
	}
	out.Gross = gross
	if n < corrMinSamples || gross <= PositionEpsilon {
		for _, row := range used {
			out.Missing = append(out.Missing, row.Asset)
		}
		out.Holdings = []PortfolioCorrHolding{}
		out.Gross = 0
		out.Samples = n
		return out
	}
	for i := 0; i < len(used); i++ {
		for j := i + 1; j < len(used); j++ {
			out.Pairs = append(out.Pairs, corrPair(used[i], used[j]))
		}
	}
	sort.SliceStable(out.Pairs, func(i, j int) bool {
		return pairBefore(out.Pairs[i], out.Pairs[j])
	})
	if together, ok := weightedTogether(used, out.Pairs); ok {
		out.Together = &together
	}
	swing, apart, ok := bookSwing(used, gross)
	if ok {
		out.SwingPct = &swing
		out.ApartPct = &apart
	}
	return out
}

func corrSide(notional float64) string {
	if notional < 0 {
		return "short"
	}
	return "long"
}

func corrPair(a, b PortfolioCorrInput) PortfolioCorrPair {
	left, right := a, b
	if right.Asset < left.Asset {
		left, right = right, left
	}
	c := PearsonCorr(left.Returns, right.Returns)
	pair := PortfolioCorrPair{
		Left: left.Asset, Right: right.Asset, Samples: len(left.Returns),
	}
	if math.IsNaN(c) {
		return pair
	}
	pair.Complete = true
	pair.Corr = &c
	pair.Relation = ClassifyCorrRelation(c)
	return pair
}

// pairBefore lists the strongest absolute correlation first.
// A pair with no correlation stays at the bottom. Equal strength keeps left, then right.
func pairBefore(a, b PortfolioCorrPair) bool {
	ac, aok := pairAbs(a)
	bc, bok := pairAbs(b)
	if aok != bok {
		return aok
	}
	if aok && ac != bc {
		return ac > bc
	}
	if a.Left != b.Left {
		return a.Left < b.Left
	}
	return a.Right < b.Right
}

func pairAbs(p PortfolioCorrPair) (float64, bool) {
	if !p.Complete || p.Corr == nil || math.IsNaN(*p.Corr) {
		return 0, false
	}
	return math.Abs(*p.Corr), true
}

func weightedTogether(rows []PortfolioCorrInput, pairs []PortfolioCorrPair) (float64, bool) {
	by := map[string]float64{}
	for _, row := range rows {
		by[row.Asset] = math.Abs(row.Notional)
	}
	var acc, wsum float64
	for _, pair := range pairs {
		if !pair.Complete || pair.Corr == nil {
			continue
		}
		w := by[pair.Left] * by[pair.Right]
		if w <= 0 {
			continue
		}
		acc += w * *pair.Corr
		wsum += w
	}
	if wsum <= 0 {
		return 0, false
	}
	return acc / wsum, true
}

// bookSwing is the day-scaled standard deviation of the signed mix, and the
// same scale with correlations set to zero.
func bookSwing(rows []PortfolioCorrInput, gross float64) (swing, apart float64, ok bool) {
	if gross <= PositionEpsilon || len(rows) == 0 {
		return 0, 0, false
	}
	n := len(rows[0].Returns)
	if n < corrMinSamples {
		return 0, 0, false
	}
	book := make([]float64, n)
	var indep float64
	for _, row := range rows {
		w := row.Notional / gross
		for i, r := range row.Returns {
			if math.IsNaN(r) {
				return 0, 0, false
			}
			book[i] += w * r
		}
		v := popVariance(row.Returns)
		if math.IsNaN(v) {
			return 0, 0, false
		}
		indep += w * w * v
	}
	scale := math.Sqrt(float64(n))
	swing = popStdev(book) * scale
	apart = math.Sqrt(indep) * scale
	if math.IsNaN(swing) || math.IsNaN(apart) {
		return 0, 0, false
	}
	return swing, apart, true
}

func popStdev(xs []float64) float64 {
	v := popVariance(xs)
	if math.IsNaN(v) {
		return math.NaN()
	}
	return math.Sqrt(v)
}

func popVariance(xs []float64) float64 {
	n := float64(len(xs))
	if n < 2 {
		return math.NaN()
	}
	var sum float64
	for _, v := range xs {
		if math.IsNaN(v) {
			return math.NaN()
		}
		sum += v
	}
	mean := sum / n
	var ss float64
	for _, v := range xs {
		d := v - mean
		ss += d * d
	}
	return ss / n
}
