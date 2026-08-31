package domain

import (
	"fmt"
	"math"
	"sort"
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

// HuntFactor is one scored driver of how easy / likely a hunt direction is.
type HuntFactor struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
	Detail string  `json:"detail"`
}

// HuntDirectionScore is the 0–100 read for up (hunt shorts) or down (hunt longs).
type HuntDirectionScore struct {
	Direction string       `json:"direction"`
	Score     float64      `json:"score"`
	Level     string       `json:"level"`
	Factors   []HuntFactor `json:"factors"`
	Reasons   []string     `json:"reasons"`
}

// HuntBias compares the two directions.
type HuntBias struct {
	Lean      string  `json:"lean"`
	Margin    float64 `json:"margin"`
	UpScore   float64 `json:"upScore"`
	DownScore float64 `json:"downScore"`
	Summary   string  `json:"summary"`
}

// HuntSignals are extra tape inputs. NaN / zero-weight means “unknown”.
// Zone and desk math in BuildHuntVenue must not depend on these.
type HuntSignals struct {
	Price1hPct, Price4hPct, Price24hPct float64
	OI1hPct, OI4hPct                    float64
	HasPrice                            bool
	HasOI                               bool
	TakerBuy1h, TakerSell1h             float64
	HasTaker                            bool
	LongLiq1h, ShortLiq1h               float64
	LongLiq4h, ShortLiq4h               float64
	HasLiqWindows                       bool
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

// AttachHuntDirectionScores fills UpScore / DownScore / Bias without changing
// zone bands or hunt P&L fields.
func AttachHuntDirectionScores(v *HuntVenueReport, sig HuntSignals) {
	if v == nil {
		return
	}
	v.UpScore = scoreHuntDirection("up", *v, sig)
	v.DownScore = scoreHuntDirection("down", *v, sig)
	v.Bias = huntBiasFromScores(v.UpScore, v.DownScore)
}

// CombineHuntBias is an OI-weighted lean across venues. Venues with no price
// and no score are skipped — never filled from the other exchange.
func CombineHuntBias(venues []HuntVenueReport) *HuntBias {
	var upW, downW, oiSum float64
	n := 0
	for _, v := range venues {
		if v.Price <= 0 && v.UpScore.Score == 0 && v.DownScore.Score == 0 {
			continue
		}
		w := v.OpenInterestValue
		if w <= 0 {
			w = 1
		}
		upW += v.UpScore.Score * w
		downW += v.DownScore.Score * w
		oiSum += w
		n++
	}
	if n == 0 || oiSum <= 0 {
		return nil
	}
	up := clampScore(upW / oiSum)
	down := clampScore(downW / oiSum)
	bias := huntBiasFromScores(
		HuntDirectionScore{Direction: "up", Score: up, Level: HuntEaseFromScore(up)},
		HuntDirectionScore{Direction: "down", Score: down, Level: HuntEaseFromScore(down)},
	)
	return &bias
}

func scoreHuntDirection(dir string, v HuntVenueReport, sig HuntSignals) HuntDirectionScore {
	self, other := v.UpHunt, v.DownHunt
	if dir == "down" {
		self, other = v.DownHunt, v.UpHunt
	}
	factors := []HuntFactor{
		huntProximityFactor(dir, self),
		huntBookFactor(dir, self, other, v),
		huntEfficiencyFactor(dir, self),
		huntTrendFactor(dir, sig),
		huntCrowdingFactor(dir, v),
		huntFlowFactor(dir, sig),
	}
	return assembleHuntScore(dir, factors)
}

func assembleHuntScore(dir string, factors []HuntFactor) HuntDirectionScore {
	var num, den float64
	kept := make([]HuntFactor, 0, len(factors))
	for _, f := range factors {
		if f.Weight <= 0 {
			continue
		}
		kept = append(kept, f)
		num += f.Score * f.Weight
		den += f.Weight
	}
	score := 0.0
	if den > 0 {
		score = clampScore(num / den)
	}
	return HuntDirectionScore{
		Direction: dir,
		Score:     score,
		Level:     HuntEaseFromScore(score),
		Factors:   kept,
		Reasons:   pickHuntReasons(dir, score, kept),
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
		if sc.HouseEdge == "unreachable" {
			f.Weight = 0.08
		}
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
	if !sig.HasPrice {
		f.Weight = 0
		return f
	}
	p1, p4, p24 := sig.Price1hPct, sig.Price4hPct, sig.Price24hPct
	o1, o4 := sig.OI1hPct, sig.OI4hPct
	if !sig.HasOI {
		o1, o4 = math.NaN(), math.NaN()
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
	// raw 4h/1h price as a light extra tilt when regime is neutral
	px := firstFinite(p4, p1, p24)
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
		f.Detail = fmt.Sprintf("price %s over the last trend window (%s)", signedPct(px), firstWindowLabel(p4, p1, p24))
	} else {
		f.Detail = "price trend is flat / mixed"
	}
	return f
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
	if !sig.HasTaker && !sig.HasLiqWindows {
		f.Weight = 0
		return f
	}
	score := 50.0
	var bits []string
	if sig.HasTaker {
		tot := sig.TakerBuy1h + sig.TakerSell1h
		if tot > 0 {
			buyShare := sig.TakerBuy1h / tot
			tilt := (buyShare - 0.5) * 80
			if dir == "down" {
				tilt = -tilt
			}
			score += tilt
			if buyShare >= 0.525 {
				bits = append(bits, fmt.Sprintf("1h takers are %.0f%% buy", buyShare*100))
			} else if buyShare <= 0.475 {
				bits = append(bits, fmt.Sprintf("1h takers are %.0f%% sell", (1-buyShare)*100))
			} else {
				bits = append(bits, "1h taker flow is balanced")
			}
		}
	}
	if sig.HasLiqWindows {
		longN, shortN := sig.LongLiq1h, sig.ShortLiq1h
		if longN+shortN <= 0 {
			longN, shortN = sig.LongLiq4h, sig.ShortLiq4h
		}
		tot := longN + shortN
		if tot > 0 {
			// shorts already liquidating → tape already walking up
			shortShare := shortN / tot
			tilt := (shortShare - 0.5) * 50
			if dir == "down" {
				tilt = -tilt
			}
			score += tilt
			if shortShare >= 0.58 {
				bits = append(bits, "recent liquidations are mostly shorts")
			} else if shortShare <= 0.42 {
				bits = append(bits, "recent liquidations are mostly longs")
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
