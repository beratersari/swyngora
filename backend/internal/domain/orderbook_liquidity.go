package domain

import (
	"math"
	"strconv"
)

// Liquidity bands around mid used for the score (not clamped by analysis min 0.25%).
var liquidityBandPcts = []float64{0.1, 0.5, 1.0}

// Nearer depth counts more for the headline score.
var liquidityBandWeights = []float64{0.5, 0.3, 0.2}

const (
	LiquidityGradeVeryLow  = "very_low"
	LiquidityGradeLow      = "low"
	LiquidityGradeMedium   = "medium"
	LiquidityGradeHigh     = "high"
	LiquidityGradeVeryHigh = "very_high"

	LiquidityWeakerBuy      = "buy"
	LiquidityWeakerSell     = "sell"
	LiquidityWeakerBalanced = "balanced"

	// scoreNotional: linear 0–20 up to $1k, then 20 + 20*log10(usd/1k)
	// so $1k=20, $10k=40, $100k=60, $1M=80, $10M=100. Never drops as size grows.
	liquidityScoreFloorUSD  = 1000.0
	liquidityScoreAtFloor   = 20.0
	liquidityBalancePenalty = 0.35
)

// LiquidityBand is bid/ask size inside ±RangePct of mid.
type LiquidityBand struct {
	RangePct      float64
	BidNotional   string
	AskNotional   string
	BidQuantity   string
	AskQuantity   string
	TotalNotional string
	Imbalance     float64
	Score         float64 // 0–100 depth in this band alone
}

// LiquidityScore is a 0–100 read of how easy the book is to trade.
type LiquidityScore struct {
	MidPrice   string
	Score      float64
	Grade      string
	WeakerSide string
	Weakness   float64 // 0–1; how lopsided the 1% band is
	Bands      []LiquidityBand
}

// VenueLiquidity is one venue's liquidity score.
type VenueLiquidity struct {
	Exchange Exchange
	Symbol   string
	Live     bool
	Error    string
	LiquidityScore
}

// MarketLiquidity is per-venue scores plus a summed market-wide score.
type MarketLiquidity struct {
	Symbol     string
	VenueCount int
	Market     LiquidityScore
	Venues     []VenueLiquidity
}

// ScoreBookLiquidity measures buy/sell notional in ±0.1 / ±0.5 / ±1% of mid
// and folds them into one score. mid ≤ 0 falls back to the book's own mid.
func ScoreBookLiquidity(raw RawOrderBook, mid float64) LiquidityScore {
	if mid <= 0 {
		mid = midPrice(raw)
	}
	out := LiquidityScore{
		Grade:      LiquidityGradeVeryLow,
		WeakerSide: LiquidityWeakerBalanced,
		Bands:      []LiquidityBand{},
	}
	if mid <= 0 {
		return out
	}
	out.MidPrice = formatFixed(mid, decimalsForStep(mid/10000)+1)

	var weighted float64
	var onePct bandStats
	for i, pct := range liquidityBandPcts {
		st := summarizeBand(raw, mid, pct)
		if pct == 1 {
			onePct = st
		}
		tot := st.bidNotional + st.askNotional
		bandScore := scoreNotionalUSD(tot)
		w := 0.0
		if i < len(liquidityBandWeights) {
			w = liquidityBandWeights[i]
		}
		weighted += bandScore * w
		out.Bands = append(out.Bands, LiquidityBand{
			RangePct:      pct,
			BidNotional:   formatQty(st.bidNotional),
			AskNotional:   formatQty(st.askNotional),
			BidQuantity:   formatQty(st.bidQty),
			AskQuantity:   formatQty(st.askQty),
			TotalNotional: formatQty(tot),
			Imbalance:     st.imbalance,
			Score:         bandScore,
		})
	}
	weaker, weakness := weakerFromBand(onePct)
	out.WeakerSide = weaker
	out.Weakness = weakness
	out.Score = round4(clamp01(weighted/100.0*(1-liquidityBalancePenalty*weakness)) * 100)
	out.Grade = liquidityGrade(out.Score)
	return out
}

// MergeLiquidityScores sums band notionals (each venue vs its own mid) and
// re-scores the market-wide book. Empty input returns a zero score.
func MergeLiquidityScores(parts []LiquidityScore) LiquidityScore {
	out := LiquidityScore{
		Grade:      LiquidityGradeVeryLow,
		WeakerSide: LiquidityWeakerBalanced,
		Bands:      []LiquidityBand{},
	}
	if len(parts) == 0 {
		return out
	}
	type acc struct {
		bidN, askN, bidQ, askQ float64
	}
	byPct := map[float64]*acc{}
	for _, pct := range liquidityBandPcts {
		byPct[pct] = &acc{}
	}
	var mid float64
	for _, p := range parts {
		if m, err := parsePlainFloat(p.MidPrice); err == nil && m > mid {
			mid = m
		}
		for _, b := range p.Bands {
			a := byPct[b.RangePct]
			if a == nil {
				a = &acc{}
				byPct[b.RangePct] = a
			}
			a.bidN += parseQty(b.BidNotional)
			a.askN += parseQty(b.AskNotional)
			a.bidQ += parseQty(b.BidQuantity)
			a.askQ += parseQty(b.AskQuantity)
		}
	}
	if mid > 0 {
		out.MidPrice = formatFixed(mid, decimalsForStep(mid/10000)+1)
	}
	var weighted float64
	var onePctImb, onePctTot float64
	for i, pct := range liquidityBandPcts {
		a := byPct[pct]
		tot := a.bidN + a.askN
		imb := 0.0
		if tot > 0 {
			imb = round4((a.bidN - a.askN) / tot)
		}
		bandScore := scoreNotionalUSD(tot)
		w := 0.0
		if i < len(liquidityBandWeights) {
			w = liquidityBandWeights[i]
		}
		weighted += bandScore * w
		if pct == 1 {
			onePctImb = imb
			onePctTot = tot
		}
		out.Bands = append(out.Bands, LiquidityBand{
			RangePct:      pct,
			BidNotional:   formatQty(a.bidN),
			AskNotional:   formatQty(a.askN),
			BidQuantity:   formatQty(a.bidQ),
			AskQuantity:   formatQty(a.askQ),
			TotalNotional: formatQty(tot),
			Imbalance:     imb,
			Score:         bandScore,
		})
	}
	weaker, weakness := weakerFromImbalance(onePctImb, onePctTot)
	out.WeakerSide = weaker
	out.Weakness = weakness
	out.Score = round4(clamp01(weighted/100.0*(1-liquidityBalancePenalty*weakness)) * 100)
	out.Grade = liquidityGrade(out.Score)
	return out
}

func scoreNotionalUSD(usd float64) float64 {
	if usd <= 0 || math.IsNaN(usd) || math.IsInf(usd, 0) {
		return 0
	}
	if usd <= liquidityScoreFloorUSD {
		return round4(usd / liquidityScoreFloorUSD * liquidityScoreAtFloor)
	}
	v := liquidityScoreAtFloor + 20*math.Log10(usd/liquidityScoreFloorUSD)
	if v > 100 {
		v = 100
	}
	return round4(v)
}

func weakerFromBand(st bandStats) (string, float64) {
	return weakerFromImbalance(st.imbalance, st.bidNotional+st.askNotional)
}

func weakerFromImbalance(imb, tot float64) (string, float64) {
	if tot <= 0 {
		return LiquidityWeakerBalanced, 0
	}
	w := math.Abs(imb)
	if w < OrderBookBalancedAbs {
		return LiquidityWeakerBalanced, round4(w)
	}
	if imb > 0 {
		// More bids than asks → sell-side (asks) is thinner.
		return LiquidityWeakerSell, round4(w)
	}
	return LiquidityWeakerBuy, round4(w)
}

func liquidityGrade(score float64) string {
	switch {
	case score >= 80:
		return LiquidityGradeVeryHigh
	case score >= 60:
		return LiquidityGradeHigh
	case score >= 40:
		return LiquidityGradeMedium
	case score >= 20:
		return LiquidityGradeLow
	default:
		return LiquidityGradeVeryLow
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func parsePlainFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func parseQty(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
