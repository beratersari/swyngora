package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	ImpactSideBuy       = "buy"
	ImpactSideSell      = "sell"
	MaxImpactQuantity   = 1e12
	MaxImpactNotional   = 1e15
	MaxImpactWalkRows   = 30
	ImpactScopeCombined = "combined"
)

// ImpactSourceLevel is one resting level tagged with its venue.
type ImpactSourceLevel struct {
	Exchange string
	Price    float64
	Quantity float64
}

// ImpactFill is one consumed slice of the book during a simulated market order.
type ImpactFill struct {
	Exchange           string
	Price              string
	Quantity           string
	Notional           string
	CumulativeQuantity string
	CumulativeNotional string
}

// OrderBookImpact is the result of walking live depth for a market buy or sell.
type OrderBookImpact struct {
	Symbol            string
	Scope             string // venue id or "combined"
	Side              string
	MidPrice          string
	BestPrice         string // best ask (buy) or best bid (sell)
	AveragePrice      string
	EndPrice          string // last fill price (how far the book was walked)
	RequestedQuantity string
	RequestedNotional string
	FilledQuantity    string
	SpentNotional     string
	UnfilledQuantity  string
	UnfilledNotional  string
	VisibleQuantity   string // resting size on that side in the snapshot
	VisibleNotional   string
	SlippagePct       float64 // adverse vs mid
	SlippageVsBestPct float64 // adverse vs best bid/ask
	ImpactPct         float64 // adverse: last fill vs mid
	Exhausted         bool
	LevelsUsed        int
	VenueCount        int
	Live              bool
	Fills             []ImpactFill
}

// ParseImpactSide returns buy or sell. Empty defaults to buy.
func ParseImpactSide(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ImpactSideBuy, nil
	}
	if s != ImpactSideBuy && s != ImpactSideSell {
		return "", fmt.Errorf("%w: side must be buy or sell", ErrInvalidArgument)
	}
	return s, nil
}

// ValidateImpactSize requires exactly one of quantity or notional, both positive and finite.
func ValidateImpactSize(quantity, notional float64) error {
	q := quantity > 0 && !math.IsNaN(quantity) && !math.IsInf(quantity, 0)
	n := notional > 0 && !math.IsNaN(notional) && !math.IsInf(notional, 0)
	if q && n {
		return fmt.Errorf("%w: provide quantity or notional, not both", ErrInvalidArgument)
	}
	if !q && !n {
		return fmt.Errorf("%w: quantity or notional is required", ErrInvalidArgument)
	}
	if q && quantity > MaxImpactQuantity {
		return fmt.Errorf("%w: quantity is too large", ErrInvalidArgument)
	}
	if n && notional > MaxImpactNotional {
		return fmt.Errorf("%w: notional is too large", ErrInvalidArgument)
	}
	return nil
}

// ImpactBookMid is the midpoint of the best bid and best ask across the books
// being walked. SharedBookMid (median venue mid) can sit above the cheapest
// cross-venue ask when venues disagree, which would make a buy look like
// negative "adverse" slippage.
func ImpactBookMid(books []VenueRawBook) float64 {
	var bestBid, bestAsk float64
	for _, b := range books {
		if b.Err != "" {
			continue
		}
		for _, lv := range b.Book.Bids {
			if !validLevel(lv) {
				continue
			}
			if lv.Price > bestBid {
				bestBid = lv.Price
			}
		}
		for _, lv := range b.Book.Asks {
			if !validLevel(lv) {
				continue
			}
			if bestAsk == 0 || lv.Price < bestAsk {
				bestAsk = lv.Price
			}
		}
	}
	if bestBid > 0 && bestAsk > 0 {
		return (bestBid + bestAsk) / 2
	}
	if bestAsk > 0 {
		return bestAsk
	}
	if bestBid > 0 {
		return bestBid
	}
	return SharedBookMid(books)
}

// CollectImpactLevels merges venue books into price order for a market walk.
// Buy walks asks low→high; sell walks bids high→low.
func CollectImpactLevels(side string, books []VenueRawBook) []ImpactSourceLevel {
	buy := side != ImpactSideSell
	var out []ImpactSourceLevel
	for _, vb := range books {
		if vb.Err != "" {
			continue
		}
		levels := vb.Book.Asks
		if !buy {
			levels = vb.Book.Bids
		}
		ex := string(vb.Exchange)
		for _, lv := range levels {
			if !validLevel(lv) {
				continue
			}
			out = append(out, ImpactSourceLevel{Exchange: ex, Price: lv.Price, Quantity: lv.Quantity})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if buy {
			if out[i].Price == out[j].Price {
				return out[i].Exchange < out[j].Exchange
			}
			return out[i].Price < out[j].Price
		}
		if out[i].Price == out[j].Price {
			return out[i].Exchange < out[j].Exchange
		}
		return out[i].Price > out[j].Price
	})
	return out
}

// SimulateMarketImpact walks resting levels until quantity or notional is filled.
func SimulateMarketImpact(symbol, scope, side string, mid float64, levels []ImpactSourceLevel, quantity, notional float64) OrderBookImpact {
	if side == "" {
		side = ImpactSideBuy
	}
	out := OrderBookImpact{
		Symbol: symbol,
		Scope:  scope,
		Side:   side,
		Fills:  []ImpactFill{},
	}
	if mid > 0 {
		out.MidPrice = formatFixed(mid, decimalsForStep(mid/10000)+1)
	}
	var visQty, visN float64
	for _, lv := range levels {
		visQty += lv.Quantity
		visN += lv.Price * lv.Quantity
	}
	out.VisibleQuantity = formatQty(visQty)
	out.VisibleNotional = formatQty(visN)
	if quantity > 0 {
		out.RequestedQuantity = formatQty(quantity)
	}
	if notional > 0 {
		out.RequestedNotional = formatQty(notional)
	}
	if len(levels) == 0 {
		out.Exhausted = quantity > 0 || notional > 0
		out.UnfilledQuantity = out.RequestedQuantity
		out.UnfilledNotional = out.RequestedNotional
		return out
	}
	out.BestPrice = formatFixed(levels[0].Price, decimalsForStep(levels[0].Price/10000)+1)

	remainQty := quantity
	remainQuote := notional
	useQuote := notional > 0 && quantity <= 0
	var filled, spent float64
	var lastPx float64
	for _, lv := range levels {
		if useQuote {
			if remainQuote <= 0 {
				break
			}
		} else if remainQty <= 0 {
			break
		}
		take := lv.Quantity
		if useQuote {
			maxQty := remainQuote / lv.Price
			if maxQty < take {
				take = maxQty
			}
		} else if remainQty < take {
			take = remainQty
		}
		if take <= 0 {
			continue
		}
		cost := take * lv.Price
		filled += take
		spent += cost
		lastPx = lv.Price
		if useQuote {
			remainQuote -= cost
			if remainQuote < 0 {
				remainQuote = 0
			}
		} else {
			remainQty -= take
		}
		out.LevelsUsed++
		if len(out.Fills) < MaxImpactWalkRows {
			out.Fills = append(out.Fills, ImpactFill{
				Exchange:           lv.Exchange,
				Price:              formatFixed(lv.Price, decimalsForStep(lv.Price/10000)+1),
				Quantity:           formatQty(take),
				Notional:           formatQty(cost),
				CumulativeQuantity: formatQty(filled),
				CumulativeNotional: formatQty(spent),
			})
		}
	}
	out.FilledQuantity = formatQty(filled)
	out.SpentNotional = formatQty(spent)
	if lastPx > 0 {
		out.EndPrice = formatFixed(lastPx, decimalsForStep(lastPx/10000)+1)
	}
	if filled > 0 {
		avg := spent / filled
		out.AveragePrice = formatFixed(avg, decimalsForStep(avg/10000)+1)
		if mid > 0 {
			if side == ImpactSideSell {
				out.SlippagePct = adversePct(mid, avg)
				out.ImpactPct = adversePct(mid, lastPx)
			} else {
				out.SlippagePct = adversePct(avg, mid)
				out.ImpactPct = adversePct(lastPx, mid)
			}
		}
		best := levels[0].Price
		if best > 0 {
			if side == ImpactSideSell {
				out.SlippageVsBestPct = adversePct(best, avg)
			} else {
				out.SlippageVsBestPct = adversePct(avg, best)
			}
		}
	}
	if useQuote {
		if remainQuote > 1e-8 {
			out.Exhausted = true
			out.UnfilledNotional = formatQty(remainQuote)
		}
	} else if remainQty > 1e-12 {
		out.Exhausted = true
		out.UnfilledQuantity = formatQty(remainQty)
	}
	return out
}

// adversePct is max(0, (worse-better)/ref*100) so a fill better than the
// reference (crossed venues) is reported as 0 adverse slippage, not negative.
func adversePct(worse, better float64) float64 {
	if better <= 0 {
		return 0
	}
	v := (worse - better) / better * 100
	if v < 0 {
		return 0
	}
	return round4(v)
}
