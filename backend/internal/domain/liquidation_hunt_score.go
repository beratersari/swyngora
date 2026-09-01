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
	HuntLeanUp   = "up"
	HuntLeanDown = "down"
	HuntLeanEven = "even"

	HuntEaseEasier = "easier"
	HuntEaseLikely = "likely"
	HuntEaseMixed  = "mixed"
	HuntEaseHard   = "hard"

	huntLeanMargin = 8.0
)

const (
	HuntFactorProximity  = "proximity"
	HuntFactorBook       = "book"
	HuntFactorEfficiency = "efficiency"
	HuntFactorTrend      = "trend"
	HuntFactorCrowding   = "crowding"
	HuntFactorFlow       = "flow"

	HuntFactorUsed     = "used"
	HuntFactorMissing  = "missing"
	HuntFactorDisabled = "disabled"
)

// HuntScoreFactorIDs is the score mix in display order.
var HuntScoreFactorIDs = []string{
	HuntFactorProximity, HuntFactorBook, HuntFactorEfficiency,
	HuntFactorTrend, HuntFactorCrowding, HuntFactorFlow,
}

// HuntScoreQueryKeys maps factor id → query/MCP parameter name.
var HuntScoreQueryKeys = map[string]string{
	HuntFactorProximity:  "weightProximity",
	HuntFactorBook:       "weightBook",
	HuntFactorEfficiency: "weightEfficiency",
	HuntFactorTrend:      "weightTrend",
	HuntFactorCrowding:   "weightCrowding",
	HuntFactorFlow:       "weightFlow",
}

var huntScoreFactorLabel = map[string]string{
	HuntFactorProximity:  "Distance to zone",
	HuntFactorBook:       "Spot walk cost",
	HuntFactorEfficiency: "Liq per spot",
	HuntFactorTrend:      "Price + OI trend",
	HuntFactorCrowding:   "Crowding + funding",
	HuntFactorFlow:       "Taker + recent liqs",
}

var defaultHuntScorePct = map[string]float64{
	HuntFactorProximity:  20,
	HuntFactorBook:       16,
	HuntFactorEfficiency: 12,
	HuntFactorTrend:      20,
	HuntFactorCrowding:   18,
	HuntFactorFlow:       14,
}

// HuntScoreWeights is the requested mix. Pct values are 0–100 and are not
// silently rescaled. Source is "default" or "custom".
type HuntScoreWeights struct {
	Source string
	Pct    map[string]float64
}

// HuntWeightRow is one factor in the requested or used mix.
type HuntWeightRow struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	WeightPct float64 `json:"weightPct"`
	Status    string  `json:"status"`
	Detail    string  `json:"detail,omitempty"`
}

// HuntScoreMix is how the direction score was built. For AI and the desk.
type HuntScoreMix struct {
	Source         string          `json:"source"`
	Requested      []HuntWeightRow `json:"requested"`
	Used           []HuntWeightRow `json:"used"`
	RequestedTotal float64         `json:"requestedTotal"`
	UsedTotal      float64         `json:"usedTotal"`
	Missing        []string        `json:"missing"`
	Disabled       []string        `json:"disabled"`
	Note           string          `json:"note"`
}

// DefaultHuntScoreWeights is the built-in mix (sums to 100).
func DefaultHuntScoreWeights() HuntScoreWeights {
	pct := make(map[string]float64, len(defaultHuntScorePct))
	for id, v := range defaultHuntScorePct {
		pct[id] = v
	}
	return HuntScoreWeights{Source: "default", Pct: pct}
}

// ParseHuntScoreWeights reads weightProximity… query/MCP fields.
// No weight params → defaults. Any present → custom; omitted keys are 0
// (disabled). Custom mix must sum to 100; it is never renormalized.
func ParseHuntScoreWeights(get func(string) string) (HuntScoreWeights, error) {
	any := false
	raw := make(map[string]string, len(HuntScoreFactorIDs))
	for _, id := range HuntScoreFactorIDs {
		v := strings.TrimSpace(get(HuntScoreQueryKeys[id]))
		raw[id] = v
		if v != "" {
			any = true
		}
	}
	if !any {
		return DefaultHuntScoreWeights(), nil
	}
	out := HuntScoreWeights{Source: "custom", Pct: make(map[string]float64, len(HuntScoreFactorIDs))}
	var sum float64
	for _, id := range HuntScoreFactorIDs {
		if raw[id] == "" {
			out.Pct[id] = 0
			continue
		}
		n, err := strconv.ParseFloat(raw[id], 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return HuntScoreWeights{}, fmt.Errorf("%w: %s is not a number", ErrInvalidArgument, HuntScoreQueryKeys[id])
		}
		if n < 0 {
			return HuntScoreWeights{}, fmt.Errorf("%w: %s cannot be negative", ErrInvalidArgument, HuntScoreQueryKeys[id])
		}
		out.Pct[id] = n
		sum += n
	}
	if math.Abs(sum-100) > 0.05 {
		return HuntScoreWeights{}, fmt.Errorf("%w: custom score weights sum to %.2f, not 100 — not applied", ErrInvalidArgument, sum)
	}
	return out, nil
}

func (w HuntScoreWeights) pctOf(id string) float64 {
	if w.Pct == nil {
		return defaultHuntScorePct[id]
	}
	return w.Pct[id]
}

// HuntFactor is one scored driver of how easy / likely a hunt direction is.
type HuntFactor struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	Score        float64 `json:"score"`
	Weight       float64 `json:"weight"`
	RequestedPct float64 `json:"requestedPct"`
	SharePct     float64 `json:"sharePct,omitempty"`
	Effect       float64 `json:"effect"`
	Status       string  `json:"status"`
	Detail       string  `json:"detail"`
}

// HuntDirectionScore is the 0–100 read for up (hunt shorts) or down (hunt longs).
type HuntDirectionScore struct {
	Direction string       `json:"direction"`
	Score     float64      `json:"score"`
	Level     string       `json:"level"`
	Coverage  float64      `json:"coverage"`
	Factors   []HuntFactor `json:"factors"`
	Reasons   []string     `json:"reasons"`
}

// HuntBias compares the two directions.
type HuntBias struct {
	Lean      string       `json:"lean"`
	Margin    float64      `json:"margin"`
	UpScore   float64      `json:"upScore"`
	DownScore float64      `json:"downScore"`
	Summary   string       `json:"summary"`
	Coverage  HuntCoverage `json:"coverage"`
	Included  []string     `json:"included,omitempty"`
	Excluded  []string     `json:"excluded,omitempty"`
}

// HuntSignals are extra tape inputs. NaN / zero-weight means “unknown”.
// Zone and desk math in BuildHuntVenue must not depend on these.
type HuntSignals struct {
	Price1hPct, Price4hPct, Price24hPct float64
	OI1hPct, OI4hPct                    float64
	HasPrice                            bool
	HasOI                               bool
	Has1hPrice, Has4hPrice              bool
	PriceFromTicker                     bool
	TakerBuy1h, TakerSell1h             float64
	HasTaker                            bool
	TakerSpan                           HuntSpan
	LongLiq1h, ShortLiq1h               float64
	LongLiq4h, ShortLiq4h               float64
	HasLiqWindows                       bool
	LiqFeedPresent                      bool
	LiqSpan1h, LiqSpan4h                HuntSpan
	OISpan1h, OISpan4h                  HuntSpan
	PriceSpan1h, PriceSpan4h            HuntSpan
	HasBook                             bool
	HasLongShort                        bool
	HasFunding                          bool
	BookError                           string
	OIError                             string
	LSError                             string
	FundError                           string
}

// HuntEaseFromScore maps 0–100 to a short label.
func HuntEaseFromScore(score float64) string {
	switch {
	case score >= 70:
		return HuntEaseEasier
	case score >= 55:
		return HuntEaseLikely
	case score >= 40:
		return HuntEaseMixed
	default:
		return HuntEaseHard
	}
}

// HuntLeanFromScores picks up / down / even from the two direction scores.
func HuntLeanFromScores(up, down float64) (lean string, margin float64) {
	margin = round1(math.Abs(up - down))
	switch {
	case up-down >= huntLeanMargin:
		return HuntLeanUp, margin
	case down-up >= huntLeanMargin:
		return HuntLeanDown, margin
	default:
		return HuntLeanEven, margin
	}
}

// HuntVenueIncluded is true when this venue may enter the combined lean.
// An erroring or unusable venue is shown on its own card but never mixed in.
func HuntVenueIncluded(v HuntVenueReport) bool {
	if v.Error != "" {
		return false
	}
	if v.Price <= 0 {
		return false
	}
	if v.Coverage.Level != "" && !v.Coverage.Usable {
		return false
	}
	return true
}

// HuntCoverageLevel maps 0–100 completeness to a short label.
func HuntCoverageLevel(score float64) string {
	switch {
	case score >= 85:
		return HuntCoverageComplete
	case score >= 65:
		return HuntCoverageUsable
	case score >= 40:
		return HuntCoverageThin
	default:
		return HuntCoverageInsufficient
	}
}

// AttachHuntDirectionScores fills UpScore / DownScore / Bias / Coverage
// without changing zone bands or hunt P&L fields.
func AttachHuntDirectionScores(v *HuntVenueReport, sig HuntSignals) {
	AttachHuntDirectionScoresWeighted(v, sig, DefaultHuntScoreWeights())
}

// AttachHuntDirectionScoresWeighted is AttachHuntDirectionScores with an
// explicit mix. Custom weights are used as given and never renormalized.
func AttachHuntDirectionScoresWeighted(v *HuntVenueReport, sig HuntSignals, w HuntScoreWeights) {
	if v == nil {
		return
	}
	if w.Pct == nil {
		w = DefaultHuntScoreWeights()
	}
	cov := buildHuntCoverage(*v, sig)
	v.Coverage = cov
	up := scoreHuntDirection("up", *v, sig, w)
	down := scoreHuntDirection("down", *v, sig, w)
	keep := huntScoreKeep(cov.Score)
	up.Score = shrinkHuntScore(up.Score, cov.Score)
	down.Score = shrinkHuntScore(down.Score, cov.Score)
	applyHuntFactorEffects(up.Factors, keep)
	applyHuntFactorEffects(down.Factors, keep)
	up.Level = HuntEaseFromScore(up.Score)
	down.Level = HuntEaseFromScore(down.Score)
	up.Coverage = cov.Score
	down.Coverage = cov.Score
	if !cov.Usable {
		up.Reasons = append([]string{"This venue is incomplete and is not used in the combined lean."}, up.Reasons...)
		down.Reasons = append([]string{"This venue is incomplete and is not used in the combined lean."}, down.Reasons...)
	} else if cov.Score < 85 && cov.Summary != "" {
		note := cov.Summary
		up.Reasons = append(up.Reasons, note)
		down.Reasons = append(down.Reasons, note)
	}
	v.UpScore = up
	v.DownScore = down
	v.Bias = huntBiasFromScores(up, down)
	v.Bias.Coverage = cov
	v.ScoreMix = BuildHuntScoreMix(w, up.Factors)
}

// shrinkHuntScore pulls a raw factor mix toward 50 when inputs are thin,
// so a one-factor read cannot look as decisive as a full tape.
func shrinkHuntScore(raw, coverage float64) float64 {
	return clampScore(50 + (raw-50)*huntScoreKeep(coverage))
}

func huntScoreKeep(coverage float64) float64 {
	c := coverage / 100
	if c < 0 {
		c = 0
	}
	if c > 1 {
		c = 1
	}
	return 0.30 + 0.70*c
}

// applyHuntFactorEffects fills sharePct (mix weight) and effect (signed
// points this factor adds to the direction score versus 50).
func applyHuntFactorEffects(factors []HuntFactor, keep float64) {
	if keep < 0 {
		keep = 0
	}
	if keep > 1 {
		keep = 1
	}
	for i := range factors {
		// Stated weight, not a silent rescale of whatever remains.
		factors[i].SharePct = factors[i].RequestedPct
		if factors[i].Status != HuntFactorUsed {
			factors[i].Effect = 0
			continue
		}
		factors[i].Effect = round1((factors[i].Score - 50) * (factors[i].RequestedPct / 100) * keep)
	}
}

func BuildHuntScoreMix(w HuntScoreWeights, factors []HuntFactor) HuntScoreMix {
	out := HuntScoreMix{
		Source:    w.Source,
		Requested: make([]HuntWeightRow, 0, len(HuntScoreFactorIDs)),
		Used:      make([]HuntWeightRow, 0, len(HuntScoreFactorIDs)),
		Missing:   []string{},
		Disabled:  []string{},
	}
	if out.Source == "" {
		out.Source = "default"
	}
	byID := make(map[string]HuntFactor, len(factors))
	for _, f := range factors {
		byID[f.ID] = f
	}
	for _, id := range HuntScoreFactorIDs {
		req := w.pctOf(id)
		row := HuntWeightRow{ID: id, Label: huntScoreFactorLabel[id], WeightPct: req}
		f, ok := byID[id]
		if ok {
			row.Status = f.Status
			row.Detail = f.Detail
		} else if req <= 0 {
			row.Status = HuntFactorDisabled
		} else {
			row.Status = HuntFactorMissing
		}
		out.Requested = append(out.Requested, HuntWeightRow{
			ID: id, Label: row.Label, WeightPct: req, Status: row.Status, Detail: row.Detail,
		})
		out.RequestedTotal += req
		switch row.Status {
		case HuntFactorUsed:
			out.Used = append(out.Used, row)
			out.UsedTotal += req
		case HuntFactorMissing:
			out.Missing = append(out.Missing, id)
		case HuntFactorDisabled:
			out.Disabled = append(out.Disabled, id)
		}
	}
	switch {
	case len(out.Missing) > 0 && out.UsedTotal > 0:
		out.Note = fmt.Sprintf("Score uses %.0f of the requested %.0f mix. Missing factors were not replaced and weights were not rescaled.", out.UsedTotal, out.RequestedTotal)
	case len(out.Missing) > 0:
		out.Note = "No requested factor had data. Score was not invented from a substitute mix."
	case out.Source == "custom":
		out.Note = "Custom weights as given (sum 100)."
	default:
		out.Note = "Default weights."
	}
	return out
}

// CombineHuntBias is an OI-weighted lean across usable venues only.
// A venue with an error or unusable coverage is listed in Excluded and
// never filled from the other exchange.
func CombineHuntBias(venues []HuntVenueReport) *HuntBias {
	var upW, downW, covW, oiSum float64
	included := make([]string, 0, len(venues))
	excluded := make([]string, 0, len(venues))
	for _, v := range venues {
		name := string(v.Exchange)
		if name == "" {
			name = v.Symbol
		}
		if !HuntVenueIncluded(v) {
			if name != "" {
				excluded = append(excluded, name)
			}
			continue
		}
		w := v.OpenInterestValue
		if w <= 0 {
			w = 1
		}
		upW += v.UpScore.Score * w
		downW += v.DownScore.Score * w
		covW += v.Coverage.Score * w
		oiSum += w
		included = append(included, name)
	}
	if len(included) == 0 || oiSum <= 0 {
		if len(excluded) == 0 {
			return nil
		}
		cov := HuntCoverage{
			Usable:  false,
			Level:   HuntCoverageInsufficient,
			Summary: "No venue had enough data for a combined lean.",
		}
		return &HuntBias{
			Lean:     HuntLeanEven,
			Summary:  cov.Summary,
			Coverage: cov,
			Excluded: excluded,
		}
	}
	up := clampScore(upW / oiSum)
	down := clampScore(downW / oiSum)
	bias := huntBiasFromScores(
		HuntDirectionScore{Direction: "up", Score: up, Level: HuntEaseFromScore(up)},
		HuntDirectionScore{Direction: "down", Score: down, Level: HuntEaseFromScore(down)},
	)
	bias.Included = included
	bias.Excluded = excluded
	bias.Coverage = HuntCoverage{
		Score:  clampScore(covW / oiSum),
		Usable: true,
	}
	bias.Coverage.Level = HuntCoverageLevel(bias.Coverage.Score)
	bias.Coverage.Summary = combinedCoverageSummary(bias)
	if len(excluded) > 0 {
		bias.Summary = bias.Summary + " Excluded from combined: " + joinHuntNames(excluded) + "."
	}
	return &bias
}

func scoreHuntDirection(dir string, v HuntVenueReport, sig HuntSignals, w HuntScoreWeights) HuntDirectionScore {
	self, other := v.UpHunt, v.DownHunt
	if dir == "down" {
		self, other = v.DownHunt, v.UpHunt
	}
	factors := []HuntFactor{
		applyHuntWeight(huntProximityFactor(dir, self), w),
		applyHuntWeight(huntBookFactor(dir, self, other, v), w),
		applyHuntWeight(huntEfficiencyFactor(dir, self), w),
		applyHuntWeight(huntTrendFactor(dir, sig), w),
		applyHuntWeight(huntCrowdingFactor(dir, v), w),
		applyHuntWeight(huntFlowFactor(dir, sig), w),
	}
	return assembleHuntScore(dir, factors)
}

func applyHuntWeight(f HuntFactor, w HuntScoreWeights) HuntFactor {
	f.RequestedPct = w.pctOf(f.ID)
	if f.RequestedPct <= 0 {
		f.Status = HuntFactorDisabled
		f.Weight = 0
		if f.Detail == "" {
			f.Detail = "Disabled in this mix."
		}
		return f
	}
	if f.Weight <= 0 {
		f.Status = HuntFactorMissing
		f.Weight = 0
		if f.Detail == "" {
			f.Detail = "No data for this factor; nothing was substituted."
		}
		return f
	}
	f.Status = HuntFactorUsed
	f.Weight = f.RequestedPct / 100
	return f
}

func assembleHuntScore(dir string, factors []HuntFactor) HuntDirectionScore {
	var num, den float64
	used := make([]HuntFactor, 0, len(factors))
	for _, f := range factors {
		if f.Status == HuntFactorUsed && f.Weight > 0 {
			num += f.Score * f.Weight
			den += f.Weight
			used = append(used, f)
		}
	}
	score := 0.0
	if den > 0 {
		score = clampScore(num / den)
	}
	return HuntDirectionScore{
		Direction: dir,
		Score:     score,
		Level:     HuntEaseFromScore(score),
		Factors:   factors,
		Reasons:   pickHuntReasons(dir, score, used),
	}
}

func pickHuntReasons(dir string, score float64, factors []HuntFactor) []string {
	type ranked struct {
		f       HuntFactor
		contrib float64
	}
	rows := make([]ranked, 0, len(factors))
	for _, f := range factors {
		if f.Detail == "" {
			continue
		}
		rows = append(rows, ranked{f: f, contrib: f.Score * f.Weight})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].contrib == rows[j].contrib {
			return rows[i].f.Score > rows[j].f.Score
		}
		return rows[i].contrib > rows[j].contrib
	})
	reasons := make([]string, 0, 4)
	for i, r := range rows {
		if i >= 3 {
			break
		}
		if i == 0 || r.f.Score >= 40 {
			reasons = append(reasons, r.f.Detail)
		}
	}
	if len(reasons) == 0 {
		side := "short liquidations above"
		if dir == "down" {
			side = "long liquidations below"
		}
		if score < 40 {
			reasons = append(reasons, fmt.Sprintf("%s looks hard: the tape and visible book do not line up toward %s", dir, side))
		} else {
			reasons = append(reasons, fmt.Sprintf("%s is mixed without a single dominant driver", dir))
		}
	}
	return reasons
}

func huntBiasFromScores(up, down HuntDirectionScore) HuntBias {
	lean, margin := HuntLeanFromScores(up.Score, down.Score)
	var summary string
	switch lean {
	case HuntLeanUp:
		summary = fmt.Sprintf("Up looks easier (%.0f vs %.0f): closer or more aligned with short liquidations.", up.Score, down.Score)
	case HuntLeanDown:
		summary = fmt.Sprintf("Down looks easier (%.0f vs %.0f): closer or more aligned with long liquidations.", down.Score, up.Score)
	default:
		summary = fmt.Sprintf("Up and down are close (%.0f vs %.0f): neither side is clearly easier.", up.Score, down.Score)
	}
	if len(up.Reasons) > 0 && lean == HuntLeanUp {
		summary = fmt.Sprintf("Up looks easier (%.0f vs %.0f). %s", up.Score, down.Score, up.Reasons[0])
	}
	if len(down.Reasons) > 0 && lean == HuntLeanDown {
		summary = fmt.Sprintf("Down looks easier (%.0f vs %.0f). %s", down.Score, up.Score, down.Reasons[0])
	}
	return HuntBias{
		Lean:      lean,
		Margin:    margin,
		UpScore:   up.Score,
		DownScore: down.Score,
		Summary:   summary,
	}
}

func huntProximityFactor(dir string, sc HuntScenario) HuntFactor {
	f := HuntFactor{ID: "proximity", Label: "Distance to zone", Weight: 0.20}
	if sc.HouseEdge == "unreachable" && sc.Target.Price <= 0 {
		f.Score = 8
		f.Detail = fmt.Sprintf("%s has no reachable liquidation cluster on the visible book", dir)
		return f
	}
	move := math.Abs(sc.Target.MovePct)
	// 0.5% → ~94, 2% → 76, 5% → 40, 10% → 12
	f.Score = clampScore(100 - move*12)
	if sc.HouseEdge == "unreachable" {
		f.Score = clampScore(f.Score * 0.45)
	}
	side := "short liqs"
	if dir == "down" {
		side = "long liqs"
	}
	f.Detail = fmt.Sprintf("%s target is %s (%.2f%%) for %s", dir, signedPct(sc.Target.MovePct), math.Abs(sc.Target.MovePct), side)
	return f
}

func huntBookFactor(dir string, self, other HuntScenario, v HuntVenueReport) HuntFactor {
	f := HuntFactor{ID: "book", Label: "Spot walk cost", Weight: 0.16}
	thisN := self.Spot.Notional
	otherN := other.Spot.Notional
	if !self.Spot.Reachable && thisN <= 0 {
		f.Score = 10
		f.Detail = fmt.Sprintf("%s cannot walk the visible book to a cluster", dir)
		return f
	}
	score := 50.0
	if otherN > 0 && thisN > 0 {
		ratio := thisN / otherN
		// cheaper than the other side → above 50
		score = 50 + (1-ratio)*40
	} else if thisN > 0 && otherN <= 0 {
		score = 62
	}
	// thinner push side vs opposite also helps
	asks, bids := v.VisibleAskNotional, v.VisibleBidNotional
	if asks+bids > 0 {
		push := asks
		if dir == "down" {
			push = bids
		}
		thin := 0.5 - push/(asks+bids)
		score += thin * 35
	}
	if self.Spot.Exhausted {
		score -= 12
	}
	f.Score = clampScore(score)
	if thisN > 0 {
		f.Detail = fmt.Sprintf("%s needs about %s visible spot to reach the zone", dir, compactUSD(thisN))
	} else {
		f.Detail = fmt.Sprintf("%s spot walk is not sized on the visible book", dir)
	}
	return f
}

func huntEfficiencyFactor(dir string, sc HuntScenario) HuntFactor {
	f := HuntFactor{ID: "efficiency", Label: "Liq per spot", Weight: 0.12}
	if sc.Spot.Notional <= 0 || sc.EstLiquidated <= 0 {
		f.Score = 14
		f.Detail = fmt.Sprintf("%s has little estimated liquidation versus the spot walk", dir)
		return f
	}
	eff := sc.Efficiency
	// 0.5x → ~38, 1x → 50, 3x → 72, 8x → 90
	f.Score = clampScore(22 + 38*math.Log10(1+eff*4))
	f.Detail = fmt.Sprintf("%s efficiency is %.1fx estimated liq per unit spot", dir, eff)
	if sc.HouseEdge == "profit" {
		f.Score = clampScore(f.Score + 8)
		f.Detail += "; desk model is profit if cascade flow appears"
	} else if sc.HouseEdge == "loss" {
		f.Detail += "; desk model is a loss after fees"
	}
	return f
}

func huntTrendFactor(dir string, sig HuntSignals) HuntFactor {
	f := HuntFactor{ID: "trend", Label: "Price + OI trend", Weight: 0.20}
	if !sig.HasPrice || (sig.PriceFromTicker && !sig.Has1hPrice && !sig.Has4hPrice) {
		f.Weight = 0
		f.Detail = "No 1h/4h price path. 24h ticker was not used."
		return f
	}
	p1, p4 := sig.Price1hPct, sig.Price4hPct
	o1, o4 := sig.OI1hPct, sig.OI4hPct
	if !sig.HasOI {
		o1, o4 = math.NaN(), math.NaN()
	}
	o1 = huntWindowOI(o1, sig.OISpan1h)
	o4 = huntWindowOI(o4, sig.OISpan4h)
	if sig.PriceSpan4h.NeedSec > 0 && (sig.PriceSpan4h.Stale || HuntSpanFraction(sig.PriceSpan4h) < 0.85) {
		p4 = math.NaN()
	}
	if sig.PriceSpan1h.NeedSec > 0 && (sig.PriceSpan1h.Stale || HuntSpanFraction(sig.PriceSpan1h) < 0.85) && !sig.Has1hPrice {
		p1 = math.NaN()
	}
	w4 := BuildPositioningWindow("4h", p4, o4)
	w1 := BuildPositioningWindow("1h", p1, o1)
	primary := w4
	if (math.IsNaN(p4) && math.IsNaN(o4)) || (w4.Regime == RegimeNeutral && w1.Regime != RegimeNeutral && w1.Confidence > w4.Confidence) {
		primary = w1
	}
	upish := primary.Regime == RegimeLongBuildup || primary.Regime == RegimeShortCovering || primary.PriceDir == "up"
	downish := primary.Regime == RegimeShortBuildup || primary.Regime == RegimeLongUnwinding || primary.PriceDir == "down"
	score := 50.0
	switch {
	case dir == "up" && upish:
		score = 58 + primary.Confidence*0.35
	case dir == "down" && downish:
		score = 58 + primary.Confidence*0.35
	case dir == "up" && downish:
		score = 42 - primary.Confidence*0.28
	case dir == "down" && upish:
		score = 42 - primary.Confidence*0.28
	}
	// 1h/4h price only — do not fall back to a 24h ticker as if it were 1h.
	px := firstFinite(p4, p1)
	if primary.Regime == RegimeNeutral && !math.IsNaN(px) {
		if dir == "up" {
			score += clampAbs(px, 6) * 3
		} else {
			score -= clampAbs(px, 6) * 3
		}
	}
	f.Score = clampScore(score)
	if primary.Regime != RegimeNeutral && primary.Label != "" {
		f.Detail = fmt.Sprintf("%s %s (price %s, OI %s)", primary.Window, primary.Label, primary.PriceDir, primary.OIDir)
	} else if !math.IsNaN(px) {
		f.Detail = fmt.Sprintf("price %s over the last trend window (%s)", signedPct(px), firstWindowLabel(p4, p1, math.NaN()))
	} else {
		f.Detail = "price trend is flat / mixed"
	}
	if note := huntStaleOINote(sig); note != "" {
		keep := huntStaleOIKeep(sig)
		f.Score = clampScore(50 + (f.Score-50)*keep)
		f.Detail += "; " + note
	}
	return f
}

// huntWindowOI is the % change for a named window only when the sample
// actually matches that window. A 2h-old print is not a 1h change.
func huntWindowOI(pct float64, span HuntSpan) float64 {
	if span.Stale || (span.NeedSec > 0 && HuntSpanFraction(span) < 0.85) {
		return math.NaN()
	}
	return pct
}

func huntHasWindowOI(span HuntSpan) bool {
	return !span.Stale && span.NeedSec > 0 && HuntSpanFraction(span) >= 0.85
}

func huntStaleOINote(sig HuntSignals) string {
	if huntHasWindowOI(sig.OISpan4h) || huntHasWindowOI(sig.OISpan1h) {
		return ""
	}
	span := sig.OISpan4h
	need := "4h"
	if !span.Stale || span.SampleAgeSec <= 0 {
		span = sig.OISpan1h
		need = "1h"
	}
	if !span.Stale || span.SampleAgeSec <= 0 {
		return ""
	}
	return fmt.Sprintf("OI sample is %s old, not used as %s", formatHuntDur(span.SampleAgeSec), need)
}

func huntStaleOIKeep(sig HuntSignals) float64 {
	if huntHasWindowOI(sig.OISpan4h) || huntHasWindowOI(sig.OISpan1h) {
		return 1
	}
	keep := 1.0
	if sig.OISpan4h.Stale {
		keep = math.Min(keep, 0.35+HuntSpanFraction(sig.OISpan4h)*0.4)
	}
	if sig.OISpan1h.Stale {
		keep = math.Min(keep, 0.35+HuntSpanFraction(sig.OISpan1h)*0.4)
	}
	return keep
}

func huntCrowdingFactor(dir string, v HuntVenueReport) HuntFactor {
	f := HuntFactor{ID: "crowding", Label: "Crowding + funding", Weight: 0.18}
	share := v.EstShortShare
	label := "shorts"
	if dir == "down" {
		share = v.EstLongShare
		label = "longs"
	}
	if share <= 0 && v.LongShare <= 0 && v.ShortShare <= 0 && v.FundingRate == 0 && v.FundingPayer == "" {
		f.Weight = 0
		return f
	}
	// 50% → 45, 60% → 68, 70% → 90
	score := 20 + (share-0.35)/0.40*80
	payer := v.FundingPayer
	if payer == "" {
		payer = FundingPayer(v.FundingRate)
	}
	// shorts paying (neg funding) favors hunting shorts (up)
	if payer == "short" && dir == "up" {
		score += 12
	}
	if payer == "long" && dir == "down" {
		score += 12
	}
	if payer == "short" && dir == "down" {
		score -= 10
	}
	if payer == "long" && dir == "up" {
		score -= 10
	}
	f.Score = clampScore(score)
	f.Detail = fmt.Sprintf("estimated %s are %.0f%% of OI", label, share*100)
	if payer == "short" || payer == "long" {
		f.Detail += fmt.Sprintf("; %ss pay funding", payer)
	}
	return f
}

func huntFlowFactor(dir string, sig HuntSignals) HuntFactor {
	f := HuntFactor{ID: "flow", Label: "Taker + recent liqs", Weight: 0.14}
	has1hTaker := sig.HasTaker && sig.TakerBuy1h+sig.TakerSell1h > 0
	has1hLiq := sig.HasLiqWindows && sig.LongLiq1h+sig.ShortLiq1h > 0
	if !has1hTaker && !has1hLiq {
		f.Weight = 0
		f.Detail = "No 1h taker or 1h liquidation window. A longer window was not substituted."
		return f
	}
	score := 50.0
	var bits []string
	if has1hTaker {
		tot := sig.TakerBuy1h + sig.TakerSell1h
		if tot > 0 {
			buyShare := sig.TakerBuy1h / tot
			tilt := (buyShare - 0.5) * 80
			frac := HuntSpanFraction(sig.TakerSpan)
			if sig.TakerSpan.NeedSec <= 0 {
				frac = 0.25
			}
			tilt *= frac
			if dir == "down" {
				tilt = -tilt
			}
			score += tilt
			spanNote := ""
			if sig.TakerSpan.NeedSec > 0 {
				spanNote = fmt.Sprintf(" (%s of %s)", formatHuntDur(sig.TakerSpan.HaveSec), formatHuntDur(sig.TakerSpan.NeedSec))
			}
			if buyShare >= 0.525 {
				bits = append(bits, fmt.Sprintf("1h takers are %.0f%% buy%s", buyShare*100, spanNote))
			} else if buyShare <= 0.475 {
				bits = append(bits, fmt.Sprintf("1h takers are %.0f%% sell%s", (1-buyShare)*100, spanNote))
			} else {
				bits = append(bits, "1h taker flow is balanced"+spanNote)
			}
		}
	}
	if has1hLiq {
		longN, shortN := sig.LongLiq1h, sig.ShortLiq1h
		liqFrac := HuntSpanFraction(sig.LiqSpan1h)
		liqWin := "1h"
		tot := longN + shortN
		if tot > 0 {
			// shorts already liquidating → tape already walking up
			shortShare := shortN / tot
			tilt := (shortShare - 0.5) * 50
			if liqFrac <= 0 {
				liqFrac = 0.25
			}
			tilt *= liqFrac
			if dir == "down" {
				tilt = -tilt
			}
			score += tilt
			if shortShare >= 0.58 {
				bits = append(bits, fmt.Sprintf("%s liquidations are mostly shorts", liqWin))
			} else if shortShare <= 0.42 {
				bits = append(bits, fmt.Sprintf("%s liquidations are mostly longs", liqWin))
			}
		}
	}
	f.Score = clampScore(score)
	if len(bits) > 0 {
		f.Detail = bits[0]
		if len(bits) > 1 {
			f.Detail += "; " + bits[1]
		}
	} else {
		f.Detail = "recent aggressive flow is mixed"
	}
	return f
}

func signedPct(v float64) string {
	if math.IsNaN(v) {
		return "n/a"
	}
	if v > 0 {
		return fmt.Sprintf("+%.2f%%", v)
	}
	return fmt.Sprintf("%.2f%%", v)
}

func compactUSD(v float64) string {
	if v >= 1e9 {
		return fmt.Sprintf("%.2fB", v/1e9)
	}
	if v >= 1e6 {
		return fmt.Sprintf("%.2fM", v/1e6)
	}
	if v >= 1e3 {
		return fmt.Sprintf("%.1fK", v/1e3)
	}
	return fmt.Sprintf("%.0f", v)
}

func firstFinite(vs ...float64) float64 {
	for _, v := range vs {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			return v
		}
	}
	return math.NaN()
}

func firstWindowLabel(p4, p1, p24 float64) string {
	switch {
	case !math.IsNaN(p4):
		return "4h"
	case !math.IsNaN(p1):
		return "1h"
	case !math.IsNaN(p24):
		return "24h"
	default:
		return "recent"
	}
}

func clampAbs(v, cap float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v > cap {
		return cap
	}
	if v < -cap {
		return -cap
	}
	return v
}

func buildHuntCoverage(v HuntVenueReport, sig HuntSignals) HuntCoverage {
	book := huntInputBook(v, sig)
	oi := huntInputOI(v, sig)
	trend := huntInputTrend(sig)
	crowd := huntInputCrowding(sig)
	flow := huntInputFlow(sig)
	inputs := []HuntInputStatus{book, oi, trend, crowd, flow}

	var num, den float64
	missing := make([]string, 0, 4)
	weak := make([]string, 0, 4)
	for _, in := range inputs {
		den += in.Weight
		switch in.Status {
		case HuntInputOK:
			num += in.Weight
		case HuntInputWeak:
			credit := in.CoverPct / 100
			if credit < 0.25 {
				credit = 0.25
			}
			if credit > 0.85 {
				credit = 0.85
			}
			num += in.Weight * credit
			weak = append(weak, in.Label)
		case HuntInputMissing:
			missing = append(missing, in.Label)
		case HuntInputError:
			missing = append(missing, in.Label)
		}
	}
	score := 0.0
	if den > 0 {
		score = clampScore(100 * num / den)
	}
	usable := v.Error == "" && v.Price > 0 && book.Status != HuntInputError && book.Status != HuntInputMissing
	if score < 40 {
		usable = false
	}
	out := HuntCoverage{
		Score:   score,
		Level:   HuntCoverageLevel(score),
		Usable:  usable,
		Inputs:  inputs,
		Missing: missing,
		Weak:    weak,
	}
	out.Summary = huntCoverageSummary(out)
	return out
}

func huntInputBook(v HuntVenueReport, sig HuntSignals) HuntInputStatus {
	in := HuntInputStatus{ID: "book", Label: "Spot book", Weight: 0.28}
	depth := v.VisibleAskNotional + v.VisibleBidNotional
	hasBook := sig.HasBook || depth > 0
	if sig.BookError != "" {
		in.Status = HuntInputError
		in.Detail = sig.BookError
		return in
	}
	if !hasBook {
		in.Status = HuntInputMissing
		in.Detail = "no visible spot book"
		return in
	}
	if v.UpHunt.Spot.Exhausted && v.DownHunt.Spot.Exhausted {
		in.Status = HuntInputWeak
		in.Detail = "visible book is exhausted on both sides"
		return in
	}
	in.Status = HuntInputOK
	in.Detail = "live spot book"
	return in
}

// HuntSpanFromDurations is have/need for one lookback. have is capped at need.
func HuntSpanFromDurations(have, need time.Duration) HuntSpan {
	if need < 0 {
		need = 0
	}
	if have < 0 {
		have = 0
	}
	if need > 0 && have > need {
		have = need
	}
	out := HuntSpan{HaveSec: have.Seconds(), NeedSec: need.Seconds()}
	if need > 0 {
		out.CoverPct = clampScore(100 * have.Seconds() / need.Seconds())
	}
	return out
}

// HuntSpanFraction is 0–1 of the requested window actually collected.
// A stale (too-old) sample uses CoverPct, never Have/Need, so a 2h-old
// print cannot look like a full 1h window.
func HuntSpanFraction(s HuntSpan) float64 {
	if s.Stale {
		f := s.CoverPct / 100
		if f > 0.7 {
			return 0.7
		}
		if f < 0 {
			return 0
		}
		return f
	}
	if s.NeedSec <= 0 {
		return 0
	}
	f := s.HaveSec / s.NeedSec
	if f > 1 {
		return 1
	}
	if f < 0 {
		return 0
	}
	return f
}

func applyHuntSpan(in *HuntInputStatus, s HuntSpan) {
	if in == nil || s.NeedSec <= 0 {
		return
	}
	in.Need = formatHuntDur(s.NeedSec)
	in.CoverPct = s.CoverPct
	if s.Stale && s.SampleAgeSec > 0 {
		in.Stale = true
		in.Age = formatHuntDur(s.SampleAgeSec)
		in.Have = in.Age
		return
	}
	in.Have = formatHuntDur(s.HaveSec)
}

// HuntOILookback is the OI % change and how well the sample matches `window`.
// A point older than the window+slack is marked Stale: ChangePct is still
// filled so the UI can show age, but it is not a 1h/4h change.
func HuntOILookback(ser *OpenInterestSeries, window time.Duration, now time.Time) (float64, HuntSpan) {
	span := HuntSpan{NeedSec: window.Seconds()}
	if ser == nil {
		return math.NaN(), span
	}
	target := now.Add(-window)
	past, complete := FindOpenInterestSample(ser.History, target, OpenInterestSampleSlack(window))
	if past.Time.IsZero() {
		return math.NaN(), span
	}
	age := now.Sub(past.Time)
	if age < 0 {
		age = 0
	}
	span.SampleAgeSec = age.Seconds()
	pct := oiPctFromPoints(ser.Current, past)
	span.ChangePct = pct
	if complete {
		span.HaveSec = window.Seconds()
		span.CoverPct = 100
		return pct, span
	}
	span.Stale = true
	span.HaveSec = 0
	cover := 0.0
	if age > 0 {
		cover = 100 * window.Seconds() / age.Seconds()
	}
	if cover > 70 {
		cover = 70
	}
	span.CoverPct = clampScore(cover)
	return math.NaN(), span
}

// HuntOILookbackSpan is how far back OI history actually reaches vs the window.
func HuntOILookbackSpan(ser *OpenInterestSeries, window time.Duration, now time.Time) HuntSpan {
	_, span := HuntOILookback(ser, window, now)
	return span
}

func oiPctFromPoints(cur, past OpenInterestPoint) float64 {
	if past.Contracts <= 0 {
		if past.Value > 0 && cur.Value > 0 {
			p, good := oiPctChange(cur.Value, past.Value)
			if !good {
				return math.NaN()
			}
			return p
		}
		return math.NaN()
	}
	p, good := oiPctChange(cur.Contracts, past.Contracts)
	if !good {
		return math.NaN()
	}
	return p
}

// HuntPriceLookbackSpan uses hourly bar count (need 2 bars for 1h, 5 for 4h).
func HuntPriceLookbackSpan(hourlyCloses int, needBars int, need time.Duration) HuntSpan {
	haveBars := hourlyCloses - 1
	if haveBars < 0 {
		haveBars = 0
	}
	if haveBars > needBars {
		haveBars = needBars
	}
	return HuntSpanFromDurations(time.Duration(haveBars)*time.Hour, need)
}

// HuntSpanFromTakerWindow uses collected vs requested seconds on a taker window.
func HuntSpanFromTakerWindow(w TakerWindowFlow) HuntSpan {
	if w.NeedSec > 0 {
		return HuntSpanFromDurations(time.Duration(w.HaveSec)*time.Second, time.Duration(w.NeedSec)*time.Second)
	}
	if w.Complete {
		return HuntSpanFromDurations(time.Hour, time.Hour)
	}
	return HuntSpan{NeedSec: 3600}
}

// HuntSpanFromLiqWindow uses live coverage seconds for one liquidation lookback.
func HuntSpanFromLiqWindow(w LiquidationWindowTotals, need time.Duration) HuntSpan {
	have := time.Duration(w.CoverageSeconds) * time.Second
	if w.Complete {
		have = need
	}
	return HuntSpanFromDurations(have, need)
}

func formatHuntDur(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	if sec >= 3600 {
		h := sec / 3600
		if math.Abs(h-math.Round(h)) < 0.05 {
			return fmt.Sprintf("%.0fh", math.Round(h))
		}
		return fmt.Sprintf("%.1fh", h)
	}
	if sec >= 60 {
		return fmt.Sprintf("%.0fm", math.Round(sec/60))
	}
	return fmt.Sprintf("%.0fs", sec)
}

func huntSpanStatus(s HuntSpan) string {
	if s.NeedSec <= 0 || s.CoverPct <= 0 {
		return HuntInputMissing
	}
	if s.CoverPct >= 85 {
		return HuntInputOK
	}
	if s.CoverPct >= 20 {
		return HuntInputWeak
	}
	return HuntInputMissing
}

func huntInputOI(v HuntVenueReport, sig HuntSignals) HuntInputStatus {
	in := HuntInputStatus{ID: "oi", Label: "Open interest", Weight: 0.16}
	if sig.OIError != "" && v.OpenInterestValue <= 0 {
		in.Status = HuntInputError
		in.Detail = sig.OIError
		return in
	}
	if v.OpenInterestValue <= 0 {
		in.Status = HuntInputMissing
		in.Detail = "no open interest"
		return in
	}
	span := sig.OISpan4h
	if span.NeedSec <= 0 || (span.CoverPct <= 0 && sig.OISpan1h.Stale) {
		span = sig.OISpan1h
	}
	if span.NeedSec > 0 {
		applyHuntSpan(&in, span)
		if span.Stale {
			in.Status = HuntInputWeak
			in.Detail = fmt.Sprintf("open interest sample is %s old (need %s)", in.Age, in.Need)
			return in
		}
		st := huntSpanStatus(span)
		if st == HuntInputOK && sig.HasOI {
			in.Status = HuntInputOK
			in.Detail = fmt.Sprintf("open interest history %s of %s", in.Have, in.Need)
			return in
		}
		if st != HuntInputMissing {
			in.Status = HuntInputWeak
			in.Detail = fmt.Sprintf("open interest history %s of %s", in.Have, in.Need)
			return in
		}
	}
	if sig.HasOI {
		in.Status = HuntInputWeak
		in.Detail = "open interest change is from a short or unmatched sample"
		return in
	}
	in.Status = HuntInputWeak
	in.Detail = "open interest has no recent change history"
	return in
}

func huntInputTrend(sig HuntSignals) HuntInputStatus {
	in := HuntInputStatus{ID: "trend", Label: "Price + OI trend", Weight: 0.20}
	if sig.PriceFromTicker && !sig.Has1hPrice && !sig.Has4hPrice {
		in.Status = HuntInputWeak
		in.Detail = "trend is the 24h ticker only"
		return in
	}
	span := sig.PriceSpan4h
	label := "4h"
	if span.NeedSec <= 0 {
		span = sig.PriceSpan1h
		label = "1h"
	}
	if span.NeedSec > 0 {
		applyHuntSpan(&in, span)
		st := huntSpanStatus(span)
		if st == HuntInputOK && sig.Has4hPrice {
			in.Status = HuntInputOK
			in.Detail = fmt.Sprintf("price path %s of %s", in.Have, in.Need)
			return in
		}
		if st != HuntInputMissing || sig.HasPrice {
			in.Status = HuntInputWeak
			if in.Have != "" {
				in.Detail = fmt.Sprintf("%s price path %s of %s", label, in.Have, in.Need)
			} else {
				in.Detail = "trend is only a short or ticker window"
			}
			return in
		}
	}
	if !sig.HasPrice {
		in.Status = HuntInputMissing
		in.Detail = "no price change window"
		return in
	}
	in.Status = HuntInputWeak
	in.Detail = "trend is only a short or ticker window"
	return in
}

func huntInputCrowding(sig HuntSignals) HuntInputStatus {
	in := HuntInputStatus{ID: "crowding", Label: "Crowding + funding", Weight: 0.18}
	if sig.LSError != "" && !sig.HasLongShort {
		in.Status = HuntInputError
		in.Detail = sig.LSError
		return in
	}
	if !sig.HasLongShort {
		in.Status = HuntInputMissing
		in.Detail = "no account long/short"
		return in
	}
	if !sig.HasFunding {
		in.Status = HuntInputWeak
		in.Detail = "long/short without a funding print"
		return in
	}
	in.Status = HuntInputOK
	in.Detail = "long/short and funding loaded"
	return in
}

func huntInputFlow(sig HuntSignals) HuntInputStatus {
	in := HuntInputStatus{ID: "flow", Label: "Taker + recent liqs", Weight: 0.18}
	taker := sig.TakerSpan
	if taker.NeedSec <= 0 && sig.HasTaker {
		// Some prints but no clock — do not treat as a full hour.
		taker = HuntSpan{HaveSec: 0, NeedSec: 3600, CoverPct: 0}
	}
	liq := sig.LiqSpan1h
	if liq.NeedSec <= 0 {
		liq = sig.LiqSpan4h
	}
	if sig.HasTaker && taker.NeedSec > 0 {
		applyHuntSpan(&in, taker)
		st := huntSpanStatus(taker)
		if st == HuntInputOK {
			in.Status = HuntInputOK
			in.Detail = fmt.Sprintf("1h taker %s of %s", formatHuntDur(taker.HaveSec), formatHuntDur(taker.NeedSec))
			return in
		}
		in.Status = HuntInputWeak
		in.Detail = fmt.Sprintf("1h taker only %s of %s", formatHuntDur(taker.HaveSec), formatHuntDur(taker.NeedSec))
		if liq.NeedSec > 0 && liq.CoverPct > 0 {
			in.Detail += fmt.Sprintf("; liqs %s of %s", formatHuntDur(liq.HaveSec), formatHuntDur(liq.NeedSec))
		}
		return in
	}
	if sig.LiqFeedPresent || sig.HasLiqWindows {
		in.Status = HuntInputWeak
		if liq.NeedSec > 0 {
			applyHuntSpan(&in, liq)
			in.Detail = fmt.Sprintf("no taker tape; liqs %s of %s", formatHuntDur(liq.HaveSec), formatHuntDur(liq.NeedSec))
		} else {
			in.Detail = "no taker tape; using observed liquidations only"
		}
		return in
	}
	in.Status = HuntInputMissing
	in.Detail = "no taker flow or liquidation feed"
	return in
}

func huntCoverageSummary(c HuntCoverage) string {
	if !c.Usable {
		if len(c.Missing) > 0 {
			return fmt.Sprintf("Insufficient data (%s missing).", joinHuntNames(c.Missing))
		}
		return "Insufficient data for a combined lean."
	}
	switch c.Level {
	case HuntCoverageComplete:
		return "Inputs look complete."
	case HuntCoverageUsable:
		if len(c.Weak) > 0 {
			return fmt.Sprintf("Usable, but %s is thin.", joinHuntNames(c.Weak))
		}
		return "Usable coverage."
	case HuntCoverageThin:
		bits := append([]string{}, c.Missing...)
		bits = append(bits, c.Weak...)
		if len(bits) > 0 {
			return fmt.Sprintf("Thin coverage: %s.", joinHuntNames(bits))
		}
		return "Thin coverage."
	default:
		return "Insufficient data."
	}
}

func combinedCoverageSummary(b HuntBias) string {
	if len(b.Excluded) > 0 && len(b.Included) > 0 {
		return fmt.Sprintf("Combined uses %s only (coverage %.0f). Left out: %s.",
			joinHuntNames(b.Included), b.Coverage.Score, joinHuntNames(b.Excluded))
	}
	if b.Coverage.Level == HuntCoverageComplete {
		return "Combined coverage looks complete."
	}
	return fmt.Sprintf("Combined coverage %.0f (%s).", b.Coverage.Score, b.Coverage.Level)
}

func joinHuntNames(in []string) string {
	switch len(in) {
	case 0:
		return ""
	case 1:
		return in[0]
	case 2:
		return in[0] + " and " + in[1]
	default:
		return strings.Join(in[:len(in)-1], ", ") + ", and " + in[len(in)-1]
	}
}
