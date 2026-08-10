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
	DefaultOrderBookRawLimit = 100
	MaxOrderBookRawLimit     = 500
	MinOrderBookRawLimit     = 5
	DefaultOrderBookLevels   = 20
	MaxOrderBookLevels       = 100
	MinOrderBookLevels       = 5
	// WallMultiple: a grouped level is a wall when its notional is at least this
	// times the median notional on that side.
	OrderBookWallMultiple = 4.0
	// WallMinShare: or when it holds at least this fraction of visible side notional.
	OrderBookWallMinShare = 0.12
)

// PriceLevel is one raw bid or ask from an exchange.
type PriceLevel struct {
	Price    float64
	Quantity float64
}

// OrderBookQuery fetches a raw spot book (ungrouped).
type OrderBookQuery struct {
	Symbol string
	Limit  int // raw levels per side
}

// RawOrderBook is the ungrouped venue snapshot. Bids are best-first (high→low);
// asks are best-first (low→high).
type RawOrderBook struct {
	Symbol    string
	Bids      []PriceLevel
	Asks      []PriceLevel
	UpdateID  int64
	FetchedAt time.Time
}

// OrderBookLevel is one grouped price bucket ready for UI / AI.
type OrderBookLevel struct {
	Price              string
	Quantity           string
	Notional           string
	Cumulative         string
	CumulativeNotional string
	RawCount           int
	IsWall             bool
}

// OrderBook is a grouped spot book with walls and suggested steps.
type OrderBook struct {
	Exchange            Exchange
	Symbol              string
	LastPrice           string
	BestBid             string
	BestAsk             string
	Spread              string
	SpreadPct           string
	GroupSize           string
	SuggestedGroupSizes []string
	Levels              int
	Bids                []OrderBookLevel
	Asks                []OrderBookLevel
	BidVolume           string
	AskVolume           string
	Imbalance           float64
	BidWalls            int
	AskWalls            int
	UpdatedAt           time.Time
}

// ParseGroupSize validates a client group step (e.g. 0.1, 0.01). Empty is ok (auto).
func ParseGroupSize(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%w: group must be a positive number", ErrInvalidArgument)
	}
	if v > 1e12 {
		return 0, fmt.Errorf("%w: group is too large", ErrInvalidArgument)
	}
	return v, nil
}

// ClampOrderBookLevels bounds displayed grouped rows per side.
func ClampOrderBookLevels(n int) int {
	if n <= 0 {
		return DefaultOrderBookLevels
	}
	if n < MinOrderBookLevels {
		return MinOrderBookLevels
	}
	if n > MaxOrderBookLevels {
		return MaxOrderBookLevels
	}
	return n
}

// ClampOrderBookRawLimit bounds the upstream depth request.
func ClampOrderBookRawLimit(n int) int {
	if n <= 0 {
		return DefaultOrderBookRawLimit
	}
	if n < MinOrderBookRawLimit {
		return MinOrderBookRawLimit
	}
	if n > MaxOrderBookRawLimit {
		return MaxOrderBookRawLimit
	}
	return n
}

var groupSizeLadder = []float64{
	1e-8, 2e-8, 5e-8,
	1e-7, 2e-7, 5e-7,
	1e-6, 2e-6, 5e-6,
	1e-5, 2e-5, 5e-5,
	1e-4, 2e-4, 5e-4,
	0.001, 0.002, 0.005,
	0.01, 0.02, 0.05,
	0.1, 0.2, 0.5,
	1, 2, 5,
	10, 20, 50,
	100, 200, 500,
	1000, 2000, 5000,
}

// SuggestedGroupSizes returns Binance-style grouping steps around refPrice.
func SuggestedGroupSizes(refPrice float64) []float64 {
	if refPrice <= 0 || math.IsNaN(refPrice) || math.IsInf(refPrice, 0) {
		return []float64{0.01, 0.1, 1}
	}
	lo := refPrice / 1e7
	hi := refPrice / 40
	if lo <= 0 {
		lo = groupSizeLadder[0]
	}
	var all []float64
	for _, s := range groupSizeLadder {
		if s+1e-18 >= lo && s <= hi+1e-18 {
			all = append(all, s)
		}
	}
	if len(all) == 0 {
		best := groupSizeLadder[0]
		target := refPrice / 1000
		if target <= 0 {
			return []float64{best}
		}
		bestDist := math.Abs(math.Log10(best) - math.Log10(target))
		for _, s := range groupSizeLadder[1:] {
			d := math.Abs(math.Log10(s) - math.Log10(target))
			if d < bestDist {
				best, bestDist = s, d
			}
		}
		return []float64{best}
	}
	// Prefer 1×10^n and 5×10^n (0.01, 0.1, 1, 10…) over 2× steps.
	var nice []float64
	for _, s := range all {
		exp := math.Floor(math.Log10(s))
		mant := s / math.Pow(10, exp)
		if math.Abs(mant-1) < 0.15 || math.Abs(mant-5) < 0.15 {
			nice = append(nice, s)
		}
	}
	if len(nice) >= 3 {
		all = nice
	}
	if len(all) > 8 {
		ones := make([]float64, 0, len(all))
		for _, s := range all {
			exp := math.Floor(math.Log10(s))
			mant := s / math.Pow(10, exp)
			if math.Abs(mant-1) < 0.15 {
				ones = append(ones, s)
			}
		}
		if len(ones) >= 3 {
			all = ones
		}
	}
	if len(all) > 8 {
		all = append(all[:7], all[len(all)-1])
	}
	return all
}

// DefaultGroupSize picks a readable step (usually the second-finest).
func DefaultGroupSize(sizes []float64) float64 {
	if len(sizes) == 0 {
		return 0.01
	}
	if len(sizes) >= 3 {
		return sizes[1]
	}
	if len(sizes) == 2 {
		return sizes[1]
	}
	return sizes[0]
}

// FormatGroupSize formats a step without scientific notation noise.
func FormatGroupSize(v float64) string {
	if v <= 0 {
		return ""
	}
	return formatFixed(v, decimalsForStep(v))
}

// GroupOrderBook aggregates raw levels onto groupSize and flags walls.
func GroupOrderBook(raw RawOrderBook, groupSize float64, levels int) OrderBook {
	levels = ClampOrderBookLevels(levels)
	ref := midPrice(raw)
	suggested := SuggestedGroupSizes(ref)
	if groupSize <= 0 {
		groupSize = DefaultGroupSize(suggested)
	}
	bids := groupSide(raw.Bids, groupSize, true, levels)
	asks := groupSide(raw.Asks, groupSize, false, levels)
	markWalls(bids)
	markWalls(asks)

	out := OrderBook{
		Symbol:              raw.Symbol,
		GroupSize:           FormatGroupSize(groupSize),
		SuggestedGroupSizes: formatSuggested(suggested),
		Levels:              levels,
		Bids:                toLevels(bids, groupSize),
		Asks:                toLevels(asks, groupSize),
		UpdatedAt:           raw.FetchedAt,
	}
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = time.Now().UTC()
	}
	if len(raw.Bids) > 0 {
		out.BestBid = formatFixed(raw.Bids[0].Price, decimalsForStep(groupSize))
	}
	if len(raw.Asks) > 0 {
		out.BestAsk = formatFixed(raw.Asks[0].Price, decimalsForStep(groupSize))
	}
	if len(raw.Bids) > 0 && len(raw.Asks) > 0 {
		spread := raw.Asks[0].Price - raw.Bids[0].Price
		if spread < 0 {
			spread = 0
		}
		out.Spread = formatFixed(spread, decimalsForStep(groupSize)+1)
		if ref > 0 {
			out.SpreadPct = formatFixed(spread/ref*100, 4)
		}
		out.LastPrice = formatFixed(ref, decimalsForStep(groupSize)+1)
	} else if ref > 0 {
		out.LastPrice = formatFixed(ref, decimalsForStep(groupSize)+1)
	}
	var bidVol, askVol float64
	for _, b := range bids {
		bidVol += b.qty
		if b.wall {
			out.BidWalls++
		}
	}
	for _, a := range asks {
		askVol += a.qty
		if a.wall {
			out.AskWalls++
		}
	}
	out.BidVolume = formatQty(bidVol)
	out.AskVolume = formatQty(askVol)
	if tot := bidVol + askVol; tot > 0 {
		out.Imbalance = (bidVol - askVol) / tot
	}
	return out
}

type groupedBucket struct {
	price float64
	qty   float64
	count int
	wall  bool
}

func groupSide(levels []PriceLevel, step float64, bids bool, keep int) []groupedBucket {
	if step <= 0 || len(levels) == 0 {
		return nil
	}
	acc := make(map[float64]*groupedBucket, len(levels))
	keys := make([]float64, 0, len(levels))
	for _, lv := range levels {
		if lv.Price <= 0 || lv.Quantity <= 0 || math.IsNaN(lv.Price) || math.IsNaN(lv.Quantity) {
			continue
		}
		p := groupPrice(lv.Price, step, bids)
		b, ok := acc[p]
		if !ok {
			b = &groupedBucket{price: p}
			acc[p] = b
			keys = append(keys, p)
		}
		b.qty += lv.Quantity
		b.count++
	}
	sort.Float64s(keys)
	if bids {
		// high → low
		for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
			keys[i], keys[j] = keys[j], keys[i]
		}
	}
	if len(keys) > keep {
		keys = keys[:keep]
	}
	out := make([]groupedBucket, 0, len(keys))
	for _, k := range keys {
		out = append(out, *acc[k])
	}
	return out
}

func groupPrice(price, step float64, bids bool) float64 {
	if step <= 0 {
		return price
	}
	n := price / step
	var g float64
	if bids {
		g = math.Floor(n + 1e-12)
	} else {
		g = math.Ceil(n - 1e-12)
		if g == 0 {
			g = 1
		}
	}
	return g * step
}

func markWalls(levels []groupedBucket) {
	if len(levels) < 3 {
		return
	}
	notionals := make([]float64, len(levels))
	var total float64
	for i, lv := range levels {
		n := lv.price * lv.qty
		notionals[i] = n
		total += n
	}
	med := medianFloat(notionals)
	if med <= 0 && total <= 0 {
		return
	}
	thresh := math.Max(med*OrderBookWallMultiple, total*OrderBookWallMinShare)
	minAbs := med * 2.5
	for i, n := range notionals {
		if n >= thresh && n >= minAbs {
			levels[i].wall = true
		}
	}
}

func toLevels(in []groupedBucket, step float64) []OrderBookLevel {
	out := make([]OrderBookLevel, 0, len(in))
	var cumQty, cumNotional float64
	dec := decimalsForStep(step)
	for _, b := range in {
		notional := b.price * b.qty
		cumQty += b.qty
		cumNotional += notional
		out = append(out, OrderBookLevel{
			Price:              formatFixed(b.price, dec),
			Quantity:           formatQty(b.qty),
			Notional:           formatQty(notional),
			Cumulative:         formatQty(cumQty),
			CumulativeNotional: formatQty(cumNotional),
			RawCount:           b.count,
			IsWall:             b.wall,
		})
	}
	return out
}

func midPrice(raw RawOrderBook) float64 {
	if len(raw.Bids) > 0 && len(raw.Asks) > 0 && raw.Bids[0].Price > 0 && raw.Asks[0].Price > 0 {
		return (raw.Bids[0].Price + raw.Asks[0].Price) / 2
	}
	if len(raw.Bids) > 0 && raw.Bids[0].Price > 0 {
		return raw.Bids[0].Price
	}
	if len(raw.Asks) > 0 && raw.Asks[0].Price > 0 {
		return raw.Asks[0].Price
	}
	return 0
}

func formatSuggested(sizes []float64) []string {
	out := make([]string, 0, len(sizes))
	for _, s := range sizes {
		out = append(out, FormatGroupSize(s))
	}
	return out
}

func decimalsForStep(step float64) int {
	if step <= 0 {
		return 8
	}
	if step >= 1 {
		// 1, 10, 50 still show 0–2 decimals for nicer mid prices.
		if step >= 10 {
			return 0
		}
		return 2
	}
	s := strconv.FormatFloat(step, 'f', -1, 64)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		d := len(s) - i - 1
		if d > 10 {
			return 10
		}
		return d
	}
	return 8
}

func formatFixed(v float64, dec int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	if dec < 0 {
		dec = 0
	}
	s := strconv.FormatFloat(v, 'f', dec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func formatQty(v float64) string {
	if v == 0 {
		return "0"
	}
	if v >= 1000 {
		return formatFixed(v, 2)
	}
	if v >= 1 {
		return formatFixed(v, 6)
	}
	return formatFixed(v, 8)
}

// ParsePriceQty parses exchange string tuples into a PriceLevel.
func ParsePriceQty(price, qty string) (PriceLevel, bool) {
	p, err1 := strconv.ParseFloat(strings.TrimSpace(price), 64)
	q, err2 := strconv.ParseFloat(strings.TrimSpace(qty), 64)
	if err1 != nil || err2 != nil || p <= 0 || q <= 0 || math.IsNaN(p) || math.IsNaN(q) {
		return PriceLevel{}, false
	}
	return PriceLevel{Price: p, Quantity: q}, true
}
