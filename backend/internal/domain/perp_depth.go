package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	DefaultPerpDepthRangePct = 1.0
	MinPerpDepthRangePct     = 0.05
	MaxPerpDepthRangePct     = 5.0

	PerpDepthLeanBid  = TakerSideBuy
	PerpDepthLeanAsk  = TakerSideSell
	PerpDepthLeanEven = TakerSideEven
)

// PerpDepthBandPcts are the ±% windows always returned (plus the requested range).
var PerpDepthBandPcts = []float64{0.1, 0.25, 0.5, 1.0, 2.0}

// PerpDepthSide is resting notional on one side of mid inside a % band.
type PerpDepthSide struct {
	Notional  float64
	Quantity  float64
	Levels    int
	EdgePrice float64 // farthest bid (below) or ask (above) in the band
}

// PerpDepthBand is bid below vs ask above inside ±RangePct of mid.
type PerpDepthBand struct {
	RangePct  float64
	Below     PerpDepthSide // bids
	Above     PerpDepthSide // asks
	Imbalance float64
	Ratio     float64 // bid / ask; 0 if ask is 0
	Lean      string  // bid | ask | balanced
	Complete  bool    // snapshot reaches this % on both sides
}

// PerpDepthVenue is one exchange's perp book (or combined).
type PerpDepthVenue struct {
	Exchange    Exchange
	Symbol      string
	Mid         float64
	CoverBidPct float64
	CoverAskPct float64
	Headline    PerpDepthBand
	Bands       []PerpDepthBand
	Summary     string
	Error       string
	AsOf        time.Time
}

// PerpDepthReport is the API result.
type PerpDepthReport struct {
	Symbol   string
	Exchange string
	RangePct float64
	AsOf     time.Time
	Venues   []PerpDepthVenue
	Combined *PerpDepthVenue
	Summary  string
	Note     string
}

// ParsePerpDepthExchange accepts binance, bybit, okx, hyperliquid, or all.
// all stays Binance, Bybit, and OKX. Hyperliquid is its own book.
func ParsePerpDepthExchange(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || s == "all" || s == "combined" {
		return "all", nil
	}
	ex := Exchange(s)
	if IsFuturesVenue(ex) || IsMarkOnlyVenue(ex) {
		return s, nil
	}
	return "", fmt.Errorf("%w: exchange must be binance, bybit, okx, hyperliquid, or all", ErrInvalidArgument)
}

// ParsePerpDepthRangePct clamps the headline ±% band (empty = 1).
func ParsePerpDepthRangePct(v float64) (float64, error) {
	if v == 0 {
		return DefaultPerpDepthRangePct, nil
	}
	if v != v || v < MinPerpDepthRangePct || v > MaxPerpDepthRangePct {
		return 0, fmt.Errorf("%w: rangePct must be between %g and %g", ErrInvalidArgument, MinPerpDepthRangePct, MaxPerpDepthRangePct)
	}
	return v, nil
}

// BuildPerpDepthVenue sums bids below mid and asks above mid in each ±% band.
func BuildPerpDepthVenue(ex Exchange, symbol string, book *RawOrderBook, rangePct float64) PerpDepthVenue {
	symbol = NormalizeLiquidationSymbol(symbol)
	if rangePct <= 0 {
		rangePct = DefaultPerpDepthRangePct
	}
	out := PerpDepthVenue{
		Exchange: ex,
		Symbol:   symbol,
		Bands:    []PerpDepthBand{},
	}
	if book == nil || (len(book.Bids) == 0 && len(book.Asks) == 0) {
		out.Error = "no perpetual depth"
		out.Summary = out.Error
		return out
	}
	mid := midPrice(*book)
	if mid <= 0 {
		out.Error = "no perpetual mid"
		out.Summary = out.Error
		return out
	}
	out.Mid = mid
	out.CoverBidPct, out.CoverAskPct = perpCoverPcts(*book, mid)
	pcts := perpBandList(rangePct)
	for _, pct := range pcts {
		band := buildPerpDepthBand(*book, mid, pct, out.CoverBidPct, out.CoverAskPct)
		out.Bands = append(out.Bands, band)
		if math.Abs(pct-rangePct) < 1e-9 {
			out.Headline = band
		}
	}
	if out.Headline.RangePct == 0 && len(out.Bands) > 0 {
		out.Headline = out.Bands[0]
		for _, b := range out.Bands {
			if math.Abs(b.RangePct-rangePct) < math.Abs(out.Headline.RangePct-rangePct) {
				out.Headline = b
			}
		}
	}
	out.Summary = ExplainPerpDepthVenue(out, rangePct)
	return out
}

// CombinePerpDepthVenues adds bid/ask notionals at the same ±% (each venue uses its own mid).
func CombinePerpDepthVenues(symbol string, venues []PerpDepthVenue, rangePct float64) *PerpDepthVenue {
	out := &PerpDepthVenue{
		Exchange: "all",
		Symbol:   NormalizeLiquidationSymbol(symbol),
		Bands:    []PerpDepthBand{},
	}
	var usable []PerpDepthVenue
	var midSum float64
	for _, v := range venues {
		if v.Error != "" || v.Mid <= 0 {
			continue
		}
		usable = append(usable, v)
		midSum += v.Mid
	}
	if len(usable) == 0 {
		out.Error = "no perpetual depth"
		out.Summary = out.Error
		return out
	}
	out.Mid = midSum / float64(len(usable))
	type acc struct {
		below, above       PerpDepthSide
		coverBid, coverAsk float64
		n                  int
	}
	m := map[float64]*acc{}
	for _, v := range usable {
		if v.CoverBidPct > out.CoverBidPct {
			out.CoverBidPct = v.CoverBidPct
		}
		if v.CoverAskPct > out.CoverAskPct {
			out.CoverAskPct = v.CoverAskPct
		}
		for _, b := range v.Bands {
			cur := m[b.RangePct]
			if cur == nil {
				cur = &acc{}
				m[b.RangePct] = cur
			}
			cur.below.Notional += b.Below.Notional
			cur.below.Quantity += b.Below.Quantity
			cur.below.Levels += b.Below.Levels
			if b.Below.EdgePrice > 0 && (cur.below.EdgePrice == 0 || b.Below.EdgePrice < cur.below.EdgePrice) {
				cur.below.EdgePrice = b.Below.EdgePrice
			}
			cur.above.Notional += b.Above.Notional
			cur.above.Quantity += b.Above.Quantity
			cur.above.Levels += b.Above.Levels
			if b.Above.EdgePrice > cur.above.EdgePrice {
				cur.above.EdgePrice = b.Above.EdgePrice
			}
			cur.n++
			if b.Complete {
				cur.coverBid++
				cur.coverAsk++
			}
		}
	}
	pcts := make([]float64, 0, len(m))
	for pct := range m {
		pcts = append(pcts, pct)
	}
	sort.Float64s(pcts)
	for _, pct := range pcts {
		a := m[pct]
		band := finishPerpDepthBand(pct, a.below, a.above, a.n > 0 && a.coverBid == float64(a.n) && a.coverAsk == float64(a.n))
		out.Bands = append(out.Bands, band)
		if math.Abs(pct-rangePct) < 1e-9 || (out.Headline.RangePct == 0 && math.Abs(pct-rangePct) < 1e-6) {
			out.Headline = band
		}
	}
	if out.Headline.RangePct == 0 && len(out.Bands) > 0 {
		out.Headline = out.Bands[0]
		for _, b := range out.Bands {
			if math.Abs(b.RangePct-rangePct) < math.Abs(out.Headline.RangePct-rangePct) {
				out.Headline = b
			}
		}
	}
	out.Summary = ExplainPerpDepthVenue(*out, rangePct)
	return out
}

// ExplainPerpDepthVenue writes a short bid-below vs ask-above line.
func ExplainPerpDepthVenue(v PerpDepthVenue, rangePct float64) string {
	if v.Error != "" {
		return v.Error
	}
	h := v.Headline
	if h.RangePct == 0 {
		h.RangePct = rangePct
	}
	name := prettyBase(v.Symbol)
	head := fmt.Sprintf("%s perp: %s bid within %g%% below vs %s ask above",
		name, formatQty(h.Below.Notional), h.RangePct, formatQty(h.Above.Notional))
	switch h.Lean {
	case PerpDepthLeanBid:
		head += " (bid-heavy)."
	case PerpDepthLeanAsk:
		head += " (ask-heavy)."
	default:
		head += "."
	}
	if !h.Complete {
		head += " Snapshot does not fully cover that band."
	}
	return head
}

// ExplainPerpDepthReport prefers combined, then the first venue with a headline.
func ExplainPerpDepthReport(r PerpDepthReport) string {
	if r.Combined != nil && r.Combined.Headline.Below.Notional+r.Combined.Headline.Above.Notional > 0 {
		return r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Error == "" && v.Summary != "" {
			return v.Summary
		}
	}
	for _, v := range r.Venues {
		if v.Summary != "" {
			return v.Summary
		}
	}
	return prettyBase(r.Symbol) + ": no perpetual depth."
}

func buildPerpDepthBand(book RawOrderBook, mid, pct, coverBid, coverAsk float64) PerpDepthBand {
	st := summarizeBand(book, mid, pct)
	below := PerpDepthSide{Notional: st.bidNotional, Quantity: st.bidQty, Levels: st.bidLevels}
	above := PerpDepthSide{Notional: st.askNotional, Quantity: st.askQty, Levels: st.askLevels}
	lo, hi := bandBounds(mid, pct)
	for _, lv := range book.Bids {
		if validLevel(lv) && lv.Price >= lo && (below.EdgePrice == 0 || lv.Price < below.EdgePrice) {
			below.EdgePrice = lv.Price
		}
	}
	for _, lv := range book.Asks {
		if validLevel(lv) && lv.Price <= hi && lv.Price > above.EdgePrice {
			above.EdgePrice = lv.Price
		}
	}
	complete := coverBid+1e-9 >= pct && coverAsk+1e-9 >= pct
	return finishPerpDepthBand(pct, below, above, complete)
}

func finishPerpDepthBand(pct float64, below, above PerpDepthSide, complete bool) PerpDepthBand {
	out := PerpDepthBand{RangePct: pct, Below: below, Above: above, Complete: complete, Lean: PerpDepthLeanEven}
	tot := below.Notional + above.Notional
	if tot > 0 {
		out.Imbalance = round4((below.Notional - above.Notional) / tot)
	}
	if above.Notional > 0 {
		out.Ratio = round4(below.Notional / above.Notional)
	}
	out.Lean = TakerDominant(below.Notional, above.Notional)
	if out.Lean == TakerSideBuy {
		out.Lean = PerpDepthLeanBid
	} else if out.Lean == TakerSideSell {
		out.Lean = PerpDepthLeanAsk
	}
	return out
}

func perpCoverPcts(book RawOrderBook, mid float64) (bidPct, askPct float64) {
	minBid, maxAsk, ok := bookExtent(book)
	if !ok || mid <= 0 {
		return 0, 0
	}
	if minBid > 0 && minBid < mid {
		bidPct = (mid - minBid) / mid * 100
	}
	if maxAsk > mid {
		askPct = (maxAsk - mid) / mid * 100
	}
	return round4(bidPct), round4(askPct)
}

func perpBandList(rangePct float64) []float64 {
	seen := map[float64]struct{}{}
	out := make([]float64, 0, len(PerpDepthBandPcts)+1)
	add := func(p float64) {
		key := math.Round(p*10000) / 10000
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	for _, p := range PerpDepthBandPcts {
		add(p)
	}
	add(rangePct)
	sort.Float64s(out)
	return out
}
