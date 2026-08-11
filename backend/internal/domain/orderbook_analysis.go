package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	DefaultOrderBookRangePct = 2.0
	MinOrderBookRangePct     = 0.25
	MaxOrderBookRangePct     = 10.0
	// |imbalance| below this is "balanced" (notional-weighted).
	OrderBookBalancedAbs = 0.08
	MaxAnalysisWalls     = 8
	thinWallShare        = 0.35
)

// OrderBookPressure is resting buy vs sell lean inside the price band.
const (
	OrderBookPressureBuy      = "buy"
	OrderBookPressureSell     = "sell"
	OrderBookPressureBalanced = "balanced"
)

// OrderBookWall is a large resting cluster inside the analysis band.
type OrderBookWall struct {
	Side              string  `json:"side"` // bid | ask
	Price             string  `json:"price"`
	Quantity          string  `json:"quantity"`
	Notional          string  `json:"notional"`
	DistancePct       string  `json:"distancePct"`
	Share             float64 `json:"share"`              // fraction of that side's band notional
	Behavior          string  `json:"behavior,omitempty"` // short | persistent | suspicious
	AgeSeconds        float64 `json:"ageSeconds,omitempty"`
	PresentForSeconds float64 `json:"presentForSeconds,omitempty"`
	VisibleSeconds    float64 `json:"visibleSeconds,omitempty"`
	AppearCount       int     `json:"appearCount,omitempty"`
}

// OrderBookBand is bid/ask notional inside ±RangePct of mid.
type OrderBookBand struct {
	RangePct    float64
	BidNotional string
	AskNotional string
	BidQuantity string
	AskQuantity string
	Imbalance   float64
	BidLevels   int
	AskLevels   int
}

// OrderBookAnalysis is pressure / imbalance / walls from live depth in a
// price band — not only the first few top-of-book rows.
type OrderBookAnalysis struct {
	RangePct      float64
	MidPrice      string
	BidNotional   string
	AskNotional   string
	BidQuantity   string
	AskQuantity   string
	Imbalance     float64 // (bidN-askN)/(bidN+askN); positive = more resting bids
	Pressure      string  // buy | sell | balanced
	BidLevels     int
	AskLevels     int
	CoveredBidPct string // how far the deepest included bid is from mid
	CoveredAskPct string
	Walls         []OrderBookWall
	Bands         []OrderBookBand
}

var analysisBandPcts = []float64{0.5, 1, 2, 5}

// ParseRangePct parses a ±% band. Empty means default.
func ParseRangePct(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%w: rangePct must be a positive number", ErrInvalidArgument)
	}
	if v > 100 {
		return 0, fmt.Errorf("%w: rangePct is too large", ErrInvalidArgument)
	}
	return v, nil
}

// ClampRangePct bounds the analysis band (default 2%).
func ClampRangePct(v float64) float64 {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return DefaultOrderBookRangePct
	}
	if v < MinOrderBookRangePct {
		return MinOrderBookRangePct
	}
	if v > MaxOrderBookRangePct {
		return MaxOrderBookRangePct
	}
	return v
}

// AnalyzeOrderBook scores buy/sell pressure and walls using every raw level
// whose price is within ±rangePct of this book's own mid.
func AnalyzeOrderBook(raw RawOrderBook, rangePct float64) OrderBookAnalysis {
	return AnalyzeOrderBookAt(raw, midPrice(raw), rangePct)
}

// AnalyzeOrderBookAt is AnalyzeOrderBook using an explicit mid (shared market mid).
func AnalyzeOrderBookAt(raw RawOrderBook, mid, rangePct float64) OrderBookAnalysis {
	rangePct = ClampRangePct(rangePct)
	out := OrderBookAnalysis{
		RangePct: rangePct,
		Pressure: OrderBookPressureBalanced,
		Walls:    []OrderBookWall{},
		Bands:    []OrderBookBand{},
	}
	if mid <= 0 {
		return out
	}
	out.MidPrice = formatFixed(mid, decimalsForStep(mid/10000)+1)

	primary := summarizeBand(raw, mid, rangePct)
	out.BidNotional = formatQty(primary.bidNotional)
	out.AskNotional = formatQty(primary.askNotional)
	out.BidQuantity = formatQty(primary.bidQty)
	out.AskQuantity = formatQty(primary.askQty)
	out.Imbalance = primary.imbalance
	out.Pressure = pressureFromImbalance(primary.imbalance)
	out.BidLevels = primary.bidLevels
	out.AskLevels = primary.askLevels
	out.CoveredBidPct = formatFixed(primary.coveredBid, 3)
	out.CoveredAskPct = formatFixed(primary.coveredAsk, 3)
	out.Walls = detectBandWalls(raw, mid, rangePct)

	for _, pct := range analysisBandPcts {
		b := summarizeBand(raw, mid, pct)
		out.Bands = append(out.Bands, OrderBookBand{
			RangePct:    pct,
			BidNotional: formatQty(b.bidNotional),
			AskNotional: formatQty(b.askNotional),
			BidQuantity: formatQty(b.bidQty),
			AskQuantity: formatQty(b.askQty),
			Imbalance:   b.imbalance,
			BidLevels:   b.bidLevels,
			AskLevels:   b.askLevels,
		})
	}
	return out
}

type bandStats struct {
	bidNotional, askNotional float64
	bidQty, askQty           float64
	bidLevels, askLevels     int
	imbalance                float64
	coveredBid, coveredAsk   float64
}

func summarizeBand(raw RawOrderBook, mid, rangePct float64) bandStats {
	lo, hi := bandBounds(mid, rangePct)
	return summarizeRange(raw, mid, lo, hi)
}

func summarizeRange(raw RawOrderBook, mid, lo, hi float64) bandStats {
	var s bandStats
	var minBid, maxAsk float64
	for _, lv := range raw.Bids {
		if !validLevel(lv) || lv.Price < lo {
			continue
		}
		s.bidNotional += lv.Price * lv.Quantity
		s.bidQty += lv.Quantity
		s.bidLevels++
		if minBid == 0 || lv.Price < minBid {
			minBid = lv.Price
		}
	}
	for _, lv := range raw.Asks {
		if !validLevel(lv) || lv.Price > hi {
			continue
		}
		s.askNotional += lv.Price * lv.Quantity
		s.askQty += lv.Quantity
		s.askLevels++
		if lv.Price > maxAsk {
			maxAsk = lv.Price
		}
	}
	if tot := s.bidNotional + s.askNotional; tot > 0 {
		s.imbalance = round4((s.bidNotional - s.askNotional) / tot)
	}
	if minBid > 0 {
		s.coveredBid = round4((mid - minBid) / mid * 100)
		if s.coveredBid < 0 {
			s.coveredBid = 0
		}
	}
	if maxAsk > 0 {
		s.coveredAsk = round4((maxAsk - mid) / mid * 100)
		if s.coveredAsk < 0 {
			s.coveredAsk = 0
		}
	}
	return s
}

func detectBandWalls(raw RawOrderBook, mid, rangePct float64) []OrderBookWall {
	lo, hi := bandBounds(mid, rangePct)
	return detectWallsInRange(raw, mid, lo, hi)
}

func detectWallsInRange(raw RawOrderBook, mid, lo, hi float64) []OrderBookWall {
	step := DefaultGroupSize(SuggestedGroupSizes(mid))
	if step <= 0 {
		step = mid / 1000
	}
	bids := groupBand(raw.Bids, step, true, lo, hi)
	asks := groupBand(raw.Asks, step, false, lo, hi)
	markAnalysisWalls(bids)
	markAnalysisWalls(asks)

	var bidN, askN float64
	for _, b := range bids {
		bidN += b.price * b.qty
	}
	for _, a := range asks {
		askN += a.price * a.qty
	}

	var walls []OrderBookWall
	for _, b := range bids {
		if !b.wall {
			continue
		}
		n := b.price * b.qty
		share := 0.0
		if bidN > 0 {
			share = round4(n / bidN)
		}
		walls = append(walls, OrderBookWall{
			Side:        "bid",
			Price:       formatFixed(b.price, decimalsForStep(step)),
			Quantity:    formatQty(b.qty),
			Notional:    formatQty(n),
			DistancePct: formatFixed(distPct(mid, b.price), 3),
			Share:       share,
		})
	}
	for _, a := range asks {
		if !a.wall {
			continue
		}
		n := a.price * a.qty
		share := 0.0
		if askN > 0 {
			share = round4(n / askN)
		}
		walls = append(walls, OrderBookWall{
			Side:        "ask",
			Price:       formatFixed(a.price, decimalsForStep(step)),
			Quantity:    formatQty(a.qty),
			Notional:    formatQty(n),
			DistancePct: formatFixed(distPct(mid, a.price), 3),
			Share:       share,
		})
	}
	sort.Slice(walls, func(i, j int) bool {
		ni, _ := strconv.ParseFloat(walls[i].Notional, 64)
		nj, _ := strconv.ParseFloat(walls[j].Notional, 64)
		if ni == nj {
			return walls[i].DistancePct < walls[j].DistancePct
		}
		return ni > nj
	})
	if len(walls) > MaxAnalysisWalls {
		walls = walls[:MaxAnalysisWalls]
	}
	if walls == nil {
		return []OrderBookWall{}
	}
	return walls
}

func groupBand(levels []PriceLevel, step float64, bids bool, lo, hi float64) []groupedBucket {
	filtered := make([]PriceLevel, 0, len(levels))
	for _, lv := range levels {
		if !validLevel(lv) {
			continue
		}
		if bids && lv.Price < lo {
			continue
		}
		if !bids && lv.Price > hi {
			continue
		}
		filtered = append(filtered, lv)
	}
	return groupSide(filtered, step, bids, MaxOrderBookRawLimit)
}

func markAnalysisWalls(levels []groupedBucket) {
	if len(levels) == 0 {
		return
	}
	if len(levels) >= 3 {
		markWalls(levels)
		return
	}
	var total float64
	notionals := make([]float64, len(levels))
	for i, lv := range levels {
		n := lv.price * lv.qty
		notionals[i] = n
		total += n
	}
	if total <= 0 {
		return
	}
	for i, n := range notionals {
		if n/total >= thinWallShare {
			levels[i].wall = true
		}
	}
}

func pressureFromImbalance(imbalance float64) string {
	if imbalance >= OrderBookBalancedAbs {
		return OrderBookPressureBuy
	}
	if imbalance <= -OrderBookBalancedAbs {
		return OrderBookPressureSell
	}
	return OrderBookPressureBalanced
}

func bandBounds(mid, rangePct float64) (lo, hi float64) {
	frac := rangePct / 100
	return mid * (1 - frac), mid * (1 + frac)
}

func validLevel(lv PriceLevel) bool {
	return lv.Price > 0 && lv.Quantity > 0 && !math.IsNaN(lv.Price) && !math.IsNaN(lv.Quantity)
}

func distPct(mid, price float64) float64 {
	if mid <= 0 {
		return 0
	}
	d := math.Abs(mid-price) / mid * 100
	return d
}

func round4(v float64) float64 {
	return math.Round(v*1e4) / 1e4
}
