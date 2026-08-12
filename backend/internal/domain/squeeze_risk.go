package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	SqueezeSideLong  = "long"
	SqueezeSideShort = "short"
	SqueezeSideNone  = "balanced"

	SqueezeLevelLow      = "low"
	SqueezeLevelModerate = "moderate"
	SqueezeLevelElevated = "elevated"
	SqueezeLevelHigh     = "high"
	SqueezeLevelExtreme  = "extreme"
)

// SqueezeFactor is one scored driver of squeeze risk (0–100).
type SqueezeFactor struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
	Detail string  `json:"detail"`
}

// SqueezeSideRisk is long-squeeze or short-squeeze risk on one venue.
type SqueezeSideRisk struct {
	Side    string          `json:"side"` // long | short — who would be squeezed
	Score   float64         `json:"score"`
	Level   string          `json:"level"`
	Factors []SqueezeFactor `json:"factors"`
	Reasons []string        `json:"reasons"`
}

// SqueezeVenueReport is per-venue squeeze risk.
type SqueezeVenueReport struct {
	Exchange          Exchange        `json:"exchange"`
	Symbol            string          `json:"symbol"`
	Price             float64         `json:"price"`
	OpenInterest      float64         `json:"openInterestValue"`
	LongShare         float64         `json:"longShare"`
	ShortShare        float64         `json:"shortShare"`
	FundingRate       float64         `json:"fundingRate"`
	FundingPayer      string          `json:"fundingPayer"`
	OIChange1hPct     float64         `json:"oiChange1hPct"`
	OIChange4hPct     float64         `json:"oiChange4hPct"`
	PriceChange24hPct float64         `json:"priceChange24hPct"`
	LongSqueeze       SqueezeSideRisk `json:"longSqueeze"`
	ShortSqueeze      SqueezeSideRisk `json:"shortSqueeze"`
	CrowdedSide       string          `json:"crowdedSide"` // long | short | balanced
	HigherRisk        string          `json:"higherRisk"`  // long | short | balanced
	Summary           string          `json:"summary"`
	Error             string          `json:"error,omitempty"`
}

// SqueezeCombined is OI-weighted blend of venues plus worst-case flags.
type SqueezeCombined struct {
	LongSqueeze   SqueezeSideRisk `json:"longSqueeze"`
	ShortSqueeze  SqueezeSideRisk `json:"shortSqueeze"`
	CrowdedSide   string          `json:"crowdedSide"`
	HigherRisk    string          `json:"higherRisk"`
	DominantVenue string          `json:"dominantVenue"` // larger OI venue
	Summary       string          `json:"summary"`
}

// SqueezeReport is the API result for one symbol.
type SqueezeReport struct {
	Symbol   string               `json:"symbol"`
	Exchange string               `json:"exchange"`
	AsOf     time.Time            `json:"asOf"`
	Venues   []SqueezeVenueReport `json:"venues"`
	Combined *SqueezeCombined     `json:"combined,omitempty"`
	Note     string               `json:"note,omitempty"`
}

// SqueezeInputs feeds BuildSqueezeVenue.
type SqueezeInputs struct {
	Exchange          Exchange
	Symbol            string
	Price             float64
	OIValue           float64
	OIChange1hPct     float64 // contracts or value %; NaN if unknown
	OIChange4hPct     float64
	PriceChange24hPct float64
	LongShare         float64
	ShortShare        float64
	LongShare1hAgo    float64 // 0 if unknown
	HasLSHistory      bool
	FundingRate       float64
	FundingAvg3       float64 // avg last 3 settlements; 0 if unknown
	HasFundingAvg     bool
	// Liquidation notional in last 1h / 24h by side (the side liquidated).
	LongLiq1h   float64
	ShortLiq1h  float64
	LongLiq24h  float64
	ShortLiq24h float64
	// Nearby estimated liquidation pressure as share of side OI within ±2%.
	LongPressureNear  float64 // 0–1 of est long notional within 2% below
	ShortPressureNear float64
}

// SqueezeLevelFromScore maps 0–100 to a label.
func SqueezeLevelFromScore(score float64) string {
	switch {
	case score >= 85:
		return SqueezeLevelExtreme
	case score >= 70:
		return SqueezeLevelHigh
	case score >= 55:
		return SqueezeLevelElevated
	case score >= 35:
		return SqueezeLevelModerate
	default:
		return SqueezeLevelLow
	}
}

// CrowdedSideFromShares labels account crowding with a 5% band around 50/50.
func CrowdedSideFromShares(longShare float64) string {
	if longShare >= 0.55 {
		return SqueezeSideLong
	}
	if longShare <= 0.45 {
		return SqueezeSideShort
	}
	return SqueezeSideNone
}

func clampScore(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return round1(v)
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

// BuildSqueezeVenue scores long and short squeeze risk for one exchange.
func BuildSqueezeVenue(in SqueezeInputs) SqueezeVenueReport {
	in.Symbol = NormalizeLiquidationSymbol(in.Symbol)
	out := SqueezeVenueReport{
		Exchange:          in.Exchange,
		Symbol:            in.Symbol,
		Price:             in.Price,
		OpenInterest:      in.OIValue,
		LongShare:         in.LongShare,
		ShortShare:        in.ShortShare,
		FundingRate:       in.FundingRate,
		FundingPayer:      FundingPayer(in.FundingRate),
		OIChange1hPct:     in.OIChange1hPct,
		OIChange4hPct:     in.OIChange4hPct,
		PriceChange24hPct: in.PriceChange24hPct,
	}
	if in.LongShare == 0 && in.ShortShare == 0 {
		in.LongShare, in.ShortShare = 0.5, 0.5
		out.LongShare, out.ShortShare = 0.5, 0.5
	}
	// Normalize if needed
	if s := in.LongShare + in.ShortShare; s > 0 && math.Abs(s-1) > 0.02 {
		in.LongShare /= s
		in.ShortShare /= s
		out.LongShare, out.ShortShare = in.LongShare, in.ShortShare
	}
	out.CrowdedSide = CrowdedSideFromShares(in.LongShare)
	out.LongSqueeze = scoreLongSqueeze(in)
	out.ShortSqueeze = scoreShortSqueeze(in)
	out.HigherRisk = higherRiskSide(out.LongSqueeze.Score, out.ShortSqueeze.Score)
	out.Summary = venueSqueezeSummary(out)
	return out
}

func higherRiskSide(longScore, shortScore float64) string {
	d := longScore - shortScore
	if d >= 8 {
		return SqueezeSideLong
	}
	if d <= -8 {
		return SqueezeSideShort
	}
	return SqueezeSideNone
}

func scoreLongSqueeze(in SqueezeInputs) SqueezeSideRisk {
	// Longs get squeezed when price falls and forced longs sell.
	factors := []SqueezeFactor{
		crowdingFactor(true, in.LongShare, in.LongShare1hAgo, in.HasLSHistory),
		fundingFactor(true, in.FundingRate, in.FundingAvg3, in.HasFundingAvg),
		oiBuildFactor(true, in.OIChange1hPct, in.OIChange4hPct, in.PriceChange24hPct),
		liqHeatFactor(true, in.LongLiq1h, in.LongLiq24h, in.OIValue),
		nearPressureFactor(true, in.LongPressureNear),
	}
	return assembleSide(SqueezeSideLong, factors)
}

func scoreShortSqueeze(in SqueezeInputs) SqueezeSideRisk {
	factors := []SqueezeFactor{
		crowdingFactor(false, in.ShortShare, 1-in.LongShare1hAgo, in.HasLSHistory),
		fundingFactor(false, in.FundingRate, in.FundingAvg3, in.HasFundingAvg),
		oiBuildFactor(false, in.OIChange1hPct, in.OIChange4hPct, in.PriceChange24hPct),
		liqHeatFactor(false, in.ShortLiq1h, in.ShortLiq24h, in.OIValue),
		nearPressureFactor(false, in.ShortPressureNear),
	}
	return assembleSide(SqueezeSideShort, factors)
}

func assembleSide(side string, factors []SqueezeFactor) SqueezeSideRisk {
	var num, den float64
	for _, f := range factors {
		if f.Weight <= 0 {
			continue
		}
		num += f.Score * f.Weight
		den += f.Weight
	}
	score := 0.0
	if den > 0 {
		score = clampScore(num / den)
	}
	reasons := pickSqueezeReasons(side, score, factors)
	return SqueezeSideRisk{
		Side:    side,
		Score:   score,
		Level:   SqueezeLevelFromScore(score),
		Factors: factors,
		Reasons: reasons,
	}
}

// pickSqueezeReasons lists the strongest drivers first so the user always
// sees why risk is high or low (not only factors above an arbitrary cutoff).
func pickSqueezeReasons(side string, score float64, factors []SqueezeFactor) []string {
	type ranked struct {
		f     SqueezeFactor
		contrib float64
	}
	rows := make([]ranked, 0, len(factors))
	for _, f := range factors {
		if f.Weight <= 0 || f.Detail == "" {
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
		// Always keep the top driver; also keep anything that is at least moderate.
		if i == 0 || r.f.Score >= 40 {
			reasons = append(reasons, r.f.Detail)
		}
	}
	if len(reasons) == 0 {
		if score < 35 {
			reasons = append(reasons, fmt.Sprintf("%s squeeze risk is low: crowding, funding, and OI build are not aligned for a cascade", side))
		} else {
			reasons = append(reasons, fmt.Sprintf("%s squeeze risk is moderate without a single dominant driver", side))
		}
	}
	return reasons
}

// crowdingFactor: share of accounts on the vulnerable side.
func crowdingFactor(longSide bool, share, share1hAgo float64, hasHist bool) SqueezeFactor {
	// 50% → ~20, 55% → ~40, 60% → ~60, 70% → ~85, 80% → ~100
	base := (share - 0.45) / 0.35 * 100
	score := clampScore(base)
	label := "short account share"
	id := "crowding_short"
	if longSide {
		label = "long account share"
		id = "crowding_long"
	}
	detail := fmt.Sprintf("%s is %.1f%% of accounts", label, share*100)
	if hasHist && share1hAgo > 0 {
		d := (share - share1hAgo) * 100
		if d >= 1.5 {
			score = clampScore(score + math.Min(15, d*3))
			detail += fmt.Sprintf(" (rising %+0.1fpp vs ~1h ago)", d)
		} else if d <= -1.5 {
			score = clampScore(score - math.Min(10, -d*2))
			detail += fmt.Sprintf(" (easing %+0.1fpp vs ~1h ago)", d)
		}
	}
	if share < 0.48 {
		detail = fmt.Sprintf("%s is only %.1f%% — not crowded on this side", label, share*100)
	}
	return SqueezeFactor{ID: id, Label: label, Score: score, Weight: 0.30, Detail: detail}
}

// fundingFactor: positive rate → longs pay → long squeeze; negative → short squeeze.
func fundingFactor(longSide bool, rate, avg3 float64, hasAvg bool) SqueezeFactor {
	// Scale: 0.01% (0.0001) per 8h is mild; 0.1% (0.001) is extreme.
	r := rate
	if hasAvg {
		// Blend current predicted with recent average (more stable).
		r = 0.6*rate + 0.4*avg3
	}
	signed := r
	if !longSide {
		signed = -r
	}
	// 0 → 15 baseline (some always pay), 0.0001 → ~45, 0.0003 → ~70, 0.001 → 100
	score := clampScore(15 + signed/0.0003*55)
	if signed < 0 {
		score = clampScore(15 + signed/0.0003*40) // discount opposite
		if score > 40 {
			score = 40
		}
	}
	payer := FundingPayer(rate)
	detail := fmt.Sprintf("funding %.4f%% (payer=%s)", rate*100, payer)
	if longSide && rate > 1e-12 {
		detail = fmt.Sprintf("longs pay funding (%.4f%%) — long side is crowded on leverage", rate*100)
	} else if !longSide && rate < -1e-12 {
		detail = fmt.Sprintf("shorts pay funding (%.4f%%) — short side is crowded on leverage", rate*100)
	} else if longSide && rate < -1e-12 {
		detail = fmt.Sprintf("shorts pay funding (%.4f%%) — less long-squeeze fuel from funding", rate*100)
	} else if !longSide && rate > 1e-12 {
		detail = fmt.Sprintf("longs pay funding (%.4f%%) — less short-squeeze fuel from funding", rate*100)
	}
	return SqueezeFactor{ID: "funding", Label: "funding pressure", Score: score, Weight: 0.25, Detail: detail}
}

// oiBuildFactor: rising OI adds fuel; rising OI against the crowded side's price move is worse.
func oiBuildFactor(longSide bool, oi1h, oi4h, price24h float64) SqueezeFactor {
	// Prefer 1h, fall back to 4h.
	oi := oi1h
	window := "1h"
	if math.IsNaN(oi) || (oi == 0 && !math.IsNaN(oi4h) && oi4h != 0) {
		if !math.IsNaN(oi4h) {
			oi = oi4h
			window = "4h"
		}
	}
	if math.IsNaN(oi) {
		oi = 0
	}
	// +1% OI → ~40, +3% → ~70, +8% → 100; falling OI cools risk
	score := clampScore(20 + oi*10)
	if oi < 0 {
		score = clampScore(20 + oi*8)
	}
	detail := fmt.Sprintf("open interest %s change %+0.2f%%", window, oi)
	// Divergence: OI up + price up → long build; OI up + price down → short build
	if oi >= 1.0 && !math.IsNaN(price24h) {
		if longSide && price24h >= 0.5 {
			score = clampScore(score + 12)
			detail += "; OI rising with price — fresh long leverage likely"
		} else if !longSide && price24h <= -0.5 {
			score = clampScore(score + 12)
			detail += "; OI rising while price falls — fresh short leverage likely"
		} else if longSide && price24h <= -1 {
			// longs already under water while OI still high/up
			score = clampScore(score + 6)
			detail += "; price already soft while OI elevated — longs stressed"
		} else if !longSide && price24h >= 1 {
			score = clampScore(score + 6)
			detail += "; price already firm while OI elevated — shorts stressed"
		}
	}
	if oi < -1 {
		detail = fmt.Sprintf("open interest %s change %+0.2f%% — leverage leaving, squeeze fuel cooling", window, oi)
	}
	return SqueezeFactor{ID: "oi_build", Label: "open interest build", Score: score, Weight: 0.20, Detail: detail}
}

// liqHeatFactor: recent liquidations of that side mean a cascade may already be live.
func liqHeatFactor(longSide bool, liq1h, liq24h, oi float64) SqueezeFactor {
	// As % of OI
	var p1, p24 float64
	if oi > 0 {
		p1 = liq1h / oi * 100
		p24 = liq24h / oi * 100
	}
	// 0.01% of OI in 1h is notable; 0.1% is hot
	score := clampScore(10 + p1*400 + p24*40)
	side := "short"
	if longSide {
		side = "long"
	}
	detail := fmt.Sprintf("recent %s liquidations: 1h=%s USDT (%.3f%% of OI), 24h=%s USDT (%.3f%% of OI)",
		side, formatQty(liq1h), p1, formatQty(liq24h), p24)
	if liq1h <= 0 && liq24h <= 0 {
		detail = fmt.Sprintf("no recent %s liquidations seen in the rolling window (stream may still be warming up)", side)
		score = 15
	} else if p1 >= 0.05 {
		detail = fmt.Sprintf("%s liquidations already elevated in the last hour (%.3f%% of OI) — cascade risk live", side, p1)
	}
	return SqueezeFactor{ID: "liq_heat", Label: "recent liquidation heat", Score: score, Weight: 0.15, Detail: detail}
}

// nearPressureFactor: share of that side's OI estimated within ~2% liquidation distance.
func nearPressureFactor(longSide bool, nearShare float64) SqueezeFactor {
	if nearShare < 0 {
		nearShare = 0
	}
	if nearShare > 1 {
		nearShare = 1
	}
	score := clampScore(nearShare * 100)
	side := "short"
	dir := "above"
	if longSide {
		side = "long"
		dir = "below"
	}
	detail := fmt.Sprintf("~%.0f%% of estimated %s OI sits within ~2%% %s (high-leverage pocket)", nearShare*100, side, dir)
	if nearShare < 0.08 {
		detail = fmt.Sprintf("little estimated %s OI in the nearest ~2%% %s — cascade needs a larger move", side, dir)
	}
	return SqueezeFactor{ID: "near_pressure", Label: "nearby liquidation pocket", Score: score, Weight: 0.10, Detail: detail}
}

func venueSqueezeSummary(v SqueezeVenueReport) string {
	hi := v.HigherRisk
	if hi == SqueezeSideNone {
		return fmt.Sprintf("%s: long squeeze %s (%.0f) vs short squeeze %s (%.0f); neither side dominates. Crowding is %s.",
			v.Exchange, v.LongSqueeze.Level, v.LongSqueeze.Score, v.ShortSqueeze.Level, v.ShortSqueeze.Score, v.CrowdedSide)
	}
	var sc SqueezeSideRisk
	if hi == SqueezeSideLong {
		sc = v.LongSqueeze
	} else {
		sc = v.ShortSqueeze
	}
	why := ""
	if len(sc.Reasons) > 0 {
		why = " " + sc.Reasons[0]
	}
	return fmt.Sprintf("%s: higher risk is a %s squeeze (%s, %.0f). Crowded side: %s.%s",
		v.Exchange, hi, sc.Level, sc.Score, v.CrowdedSide, why)
}

// CombineSqueezeReports blends venues by open-interest weight.
func CombineSqueezeReports(venues []SqueezeVenueReport) *SqueezeCombined {
	if len(venues) == 0 {
		return nil
	}
	if len(venues) == 1 {
		v := venues[0]
		return &SqueezeCombined{
			LongSqueeze:   v.LongSqueeze,
			ShortSqueeze:  v.ShortSqueeze,
			CrowdedSide:   v.CrowdedSide,
			HigherRisk:    v.HigherRisk,
			DominantVenue: string(v.Exchange),
			Summary:       "Single venue only. " + v.Summary,
		}
	}
	var oiSum, longNum, shortNum float64
	var dom Exchange
	var domOI float64
	// Weighted factor averages for combined reasons
	longFactors := map[string]*SqueezeFactor{}
	shortFactors := map[string]*SqueezeFactor{}
	var longW, shortW float64

	for _, v := range venues {
		w := v.OpenInterest
		if w <= 0 {
			w = 1
		}
		oiSum += w
		longNum += v.LongSqueeze.Score * w
		shortNum += v.ShortSqueeze.Score * w
		if v.OpenInterest >= domOI {
			domOI = v.OpenInterest
			dom = v.Exchange
		}
		for _, f := range v.LongSqueeze.Factors {
			acc := longFactors[f.ID]
			if acc == nil {
				cp := f
				cp.Score = f.Score * w
				longFactors[f.ID] = &cp
			} else {
				acc.Score += f.Score * w
			}
		}
		longW += w
		for _, f := range v.ShortSqueeze.Factors {
			acc := shortFactors[f.ID]
			if acc == nil {
				cp := f
				cp.Score = f.Score * w
				shortFactors[f.ID] = &cp
			} else {
				acc.Score += f.Score * w
			}
		}
		shortW += w
	}
	if oiSum <= 0 {
		oiSum = 1
	}
	longScore := clampScore(longNum / oiSum)
	shortScore := clampScore(shortNum / oiSum)

	// Crowding: OI-weighted long share
	var longShareSum float64
	for _, v := range venues {
		w := v.OpenInterest
		if w <= 0 {
			w = 1
		}
		longShareSum += v.LongShare * w
	}
	avgLong := longShareSum / oiSum

	longSide := SqueezeSideRisk{
		Side:    SqueezeSideLong,
		Score:   longScore,
		Level:   SqueezeLevelFromScore(longScore),
		Factors: weightedFactors(longFactors, longW),
		Reasons: combinedReasons(venues, true),
	}
	shortSide := SqueezeSideRisk{
		Side:    SqueezeSideShort,
		Score:   shortScore,
		Level:   SqueezeLevelFromScore(shortScore),
		Factors: weightedFactors(shortFactors, shortW),
		Reasons: combinedReasons(venues, false),
	}

	out := &SqueezeCombined{
		LongSqueeze:   longSide,
		ShortSqueeze:  shortSide,
		CrowdedSide:   CrowdedSideFromShares(avgLong),
		HigherRisk:    higherRiskSide(longScore, shortScore),
		DominantVenue: string(dom),
	}
	out.Summary = combinedSummary(out, venues)
	return out
}

func weightedFactors(m map[string]*SqueezeFactor, w float64) []SqueezeFactor {
	if w <= 0 {
		w = 1
	}
	out := make([]SqueezeFactor, 0, len(m))
	for _, f := range m {
		out = append(out, SqueezeFactor{
			ID:     f.ID,
			Label:  f.Label,
			Score:  clampScore(f.Score / w),
			Weight: f.Weight,
			Detail: f.Detail,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weight == out[j].Weight {
			return out[i].Score > out[j].Score
		}
		return out[i].Weight > out[j].Weight
	})
	return out
}

func combinedReasons(venues []SqueezeVenueReport, longSide bool) []string {
	// Pick top reasons from the higher-scoring venue for that side.
	var best *SqueezeVenueReport
	bestScore := -1.0
	for i := range venues {
		v := &venues[i]
		sc := v.ShortSqueeze.Score
		if longSide {
			sc = v.LongSqueeze.Score
		}
		if sc > bestScore {
			bestScore = sc
			best = v
		}
	}
	if best == nil {
		return nil
	}
	if longSide {
		return append([]string{fmt.Sprintf("OI-weighted blend; strongest long-squeeze venue is %s (%.0f)", best.Exchange, best.LongSqueeze.Score)}, best.LongSqueeze.Reasons...)
	}
	return append([]string{fmt.Sprintf("OI-weighted blend; strongest short-squeeze venue is %s (%.0f)", best.Exchange, best.ShortSqueeze.Score)}, best.ShortSqueeze.Reasons...)
}

func combinedSummary(c *SqueezeCombined, venues []SqueezeVenueReport) string {
	parts := make([]string, 0, len(venues))
	for _, v := range venues {
		parts = append(parts, fmt.Sprintf("%s L%.0f/S%.0f", v.Exchange, v.LongSqueeze.Score, v.ShortSqueeze.Score))
	}
	if c.HigherRisk == SqueezeSideNone {
		return fmt.Sprintf("Combined (OI-weighted): long squeeze %s (%.0f) vs short %s (%.0f). Venues: %v. Crowding: %s.",
			c.LongSqueeze.Level, c.LongSqueeze.Score, c.ShortSqueeze.Level, c.ShortSqueeze.Score, parts, c.CrowdedSide)
	}
	sc := c.ShortSqueeze
	if c.HigherRisk == SqueezeSideLong {
		sc = c.LongSqueeze
	}
	return fmt.Sprintf("Combined (OI-weighted): higher risk is a %s squeeze (%s, %.0f). Dominant OI venue: %s. Per venue: %v.",
		c.HigherRisk, sc.Level, sc.Score, c.DominantVenue, parts)
}

// NearLiquidationPressureShare estimates the fraction of side OI within maxMovePct
// using the same stylized leverage mix as the hunt model.
func NearLiquidationPressureShare(sideIsLong bool, maxMovePct float64, fundingRate float64) float64 {
	if maxMovePct <= 0 {
		return 0
	}
	crowded := fundingRate > 1e-12
	if !sideIsLong {
		crowded = fundingRate < -1e-12
	}
	mix := TiltLeverageMix(DefaultHuntLeverageMix, crowded)
	var share float64
	for _, b := range mix {
		dist := HuntLiqDistance(b.Leverage, HuntMaintenanceMargin) * 100
		if dist <= maxMovePct {
			share += b.Weight
		}
	}
	if share > 1 {
		return 1
	}
	return share
}

// OIChangePctFromSeries returns contract % change vs window, or NaN if missing.
func OIChangePctFromSeries(ser *OpenInterestSeries, window time.Duration, now time.Time) float64 {
	if ser == nil {
		return math.NaN()
	}
	past, ok := FindOpenInterestSample(ser.History, now.Add(-window), OpenInterestSampleSlack(window))
	if !ok && past.Time.IsZero() {
		return math.NaN()
	}
	if past.Contracts <= 0 {
		if past.Value > 0 && ser.Current.Value > 0 {
			p, good := oiPctChange(ser.Current.Value, past.Value)
			if !good {
				return math.NaN()
			}
			return p
		}
		return math.NaN()
	}
	p, good := oiPctChange(ser.Current.Contracts, past.Contracts)
	if !good {
		return math.NaN()
	}
	return p
}

// SumLiquidationNotional totals events for one side since cutoff.
func SumLiquidationNotional(events []LiquidationEvent, side string, since time.Time) float64 {
	var n float64
	for _, e := range events {
		if side != "" && e.Side != side {
			continue
		}
		if !since.IsZero() && e.Time.Before(since) {
			continue
		}
		n += e.Notional
	}
	return n
}

// LongShareAbout finds long share nearest to target time from newest-first history.
func LongShareAbout(current LongShortPoint, hist []LongShortPoint, target time.Time) (float64, bool) {
	bestShare := 0.0
	bestDT := time.Duration(math.MaxInt64)
	found := false
	all := append([]LongShortPoint{current}, hist...)
	for _, p := range all {
		if p.Time.IsZero() {
			continue
		}
		d := p.Time.Sub(target)
		if d < 0 {
			d = -d
		}
		if d < bestDT {
			bestDT = d
			bestShare = p.LongShare
			found = true
		}
	}
	// Only accept if within 45 minutes of target.
	if !found || bestDT > 45*time.Minute {
		return 0, false
	}
	return bestShare, true
}
