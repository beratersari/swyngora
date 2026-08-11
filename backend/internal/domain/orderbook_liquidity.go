package domain

import (
	"math"
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
	MidPrice     string
	UsedRangePct float64 // symmetric ±% both sides actually reach
	Score        float64
	Grade        string
	WeakerSide   string
	Weakness     float64 // 0–1; how lopsided the widest included band is
	Bands        []LiquidityBand
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

// ScoreBookLiquidity measures buy/sell notional only in ±0.1 / ±0.5 / ±1%
// bands the book actually reaches on both sides. mid ≤ 0 uses the book's mid.
func ScoreBookLiquidity(raw RawOrderBook, mid float64) LiquidityScore {
	if mid <= 0 {
		mid = midPrice(raw)
	}
	return scoreLiquidityAt(mid, bookCoverPct(raw, mid), []RawOrderBook{raw})
}

// ScoreMarketLiquidity sums venues only inside the symmetric ±% every
// contributing book can reach from the shared mid.
func ScoreMarketLiquidity(books []VenueRawBook) LiquidityScore {
	var raws []RawOrderBook
	for _, b := range books {
		if b.Err != "" {
			continue
		}
		raws = append(raws, b.Book)
	}
	mid := SharedBookMid(books)
	return scoreLiquidityAt(mid, commonCoverPct(mid, books), raws)
}

func scoreLiquidityAt(mid, coverPct float64, books []RawOrderBook) LiquidityScore {
	out := LiquidityScore{
		Grade:      LiquidityGradeVeryLow,
		WeakerSide: LiquidityWeakerBalanced,
		Bands:      []LiquidityBand{},
	}
	if mid <= 0 || len(books) == 0 {
		return out
	}
	out.MidPrice = formatFixed(mid, decimalsForStep(mid/10000)+1)
	out.UsedRangePct = round4(coverPct)

	var weighted, wsum float64
	var widest bandStats
	var haveWidest bool
	for i, pct := range liquidityBandPcts {
		if !bandFullyCovered(coverPct, pct) {
			continue
		}
		st := sumBooksInBand(books, mid, pct)
		tot := st.bidNotional + st.askNotional
		bandScore := scoreNotionalUSD(tot)
		w := 0.0
		if i < len(liquidityBandWeights) {
			w = liquidityBandWeights[i]
		}
		weighted += bandScore * w
		wsum += w
		widest = st
		haveWidest = true
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
	if !haveWidest && coverPct > 0 {
		// Book does not reach even ±0.1%; score the real window, do not invent 0.5/1%.
		widest = sumBooksInBand(books, mid, coverPct)
		haveWidest = true
		tot := widest.bidNotional + widest.askNotional
		weighted = scoreNotionalUSD(tot)
		wsum = 1
	}
	if haveWidest {
		weaker, weakness := weakerFromBand(widest)
		out.WeakerSide = weaker
		out.Weakness = weakness
		if wsum > 0 {
			out.Score = round4(clamp01((weighted/wsum)/100.0*(1-liquidityBalancePenalty*weakness)) * 100)
		}
	}
	out.Grade = liquidityGrade(out.Score)
	return out
}

func sumBooksInBand(books []RawOrderBook, mid, pct float64) bandStats {
	var tot bandStats
	for _, raw := range books {
		st := summarizeBand(raw, mid, pct)
		tot.bidNotional += st.bidNotional
		tot.askNotional += st.askNotional
		tot.bidQty += st.bidQty
		tot.askQty += st.askQty
		tot.bidLevels += st.bidLevels
		tot.askLevels += st.askLevels
	}
	if s := tot.bidNotional + tot.askNotional; s > 0 {
		tot.imbalance = round4((tot.bidNotional - tot.askNotional) / s)
	}
	return tot
}

// bookCoverPct is the symmetric ±% both sides of this book actually reach.
func bookCoverPct(raw RawOrderBook, mid float64) float64 {
	if mid <= 0 {
		return 0
	}
	minBid, maxAsk, ok := bookExtent(raw)
	if !ok {
		return 0
	}
	bidPct := (mid - minBid) / mid * 100
	askPct := (maxAsk - mid) / mid * 100
	if bidPct < 0 {
		bidPct = 0
	}
	if askPct < 0 {
		askPct = 0
	}
	if askPct < bidPct {
		return askPct
	}
	return bidPct
}

func commonCoverPct(mid float64, books []VenueRawBook) float64 {
	first := true
	var used float64
	for _, b := range books {
		if b.Err != "" {
			continue
		}
		c := bookCoverPct(b.Book, mid)
		if first || c < used {
			used = c
			first = false
		}
	}
	if first {
		return 0
	}
	return used
}

func bandFullyCovered(coverPct, bandPct float64) bool {
	return coverPct+1e-9 >= bandPct
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
