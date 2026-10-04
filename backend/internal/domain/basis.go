package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	BasisKindPremium  = "premium" // futures above spot
	BasisKindDiscount = "discount"
	BasisKindFlat     = "flat"

	BasisTrendExpanding = "expanding" // |basis| getting larger
	BasisTrendShrinking = "shrinking"
	BasisTrendStable    = "stable"
	BasisTrendFlipped   = "flipped" // premium ↔ discount

	// Flat if |basis| < 1 bp of spot (0.01%).
	basisFlatPct      = 0.01
	basisTrendMovePct = 0.005 // 0.5 bp change in basis % to call expand/shrink

	BasisContractPerp           = "perp"
	BasisContractCurrentQuarter = "current_quarter"
	BasisContractNextQuarter    = "next_quarter"
	BasisContractDated          = "dated"

	BasisCurveContango      = "contango"      // far richer than near
	BasisCurveBackwardation = "backwardation" // near richer than far
	BasisCurveHumped        = "humped"        // not monotonic
	BasisCurveUnknown       = "unknown"

	BasisSlopeUp   = "up"
	BasisSlopeDown = "down"
	BasisSlopeFlat = "flat"

	BasisRicherNear = "near"
	BasisRicherFar  = "far"
	BasisRicherNone = "none"
)

// BasisQuote is one venue's live perp vs spot/index plus 5m history of mark−index.
type BasisQuote struct {
	Exchange    Exchange
	Symbol      string
	FuturesLast float64
	FuturesMark float64
	SpotIndex   float64
	SpotLast    float64 // venue spot last; 0 if unused
	Time        time.Time
	History     []BasisHistPoint // oldest first, 5m mark vs index
	FundingRate float64
	OIChange1h  float64 // NaN if unknown
	// Contracts are listed futures for this pair (perp + dated), when the venue has them.
	Contracts []BasisContractQuote
}

// BasisContractQuote is one listed futures contract on a venue.
type BasisContractQuote struct {
	Symbol  string
	Kind    string // perp | current_quarter | next_quarter | dated
	Expiry  time.Time
	Last    float64
	Mark    float64
	Index   float64
	History []PriceSample // mark closes, oldest first (optional)
}

// BasisHistPoint is mark vs index at one sample time.
type BasisHistPoint struct {
	Time     time.Time
	Mark     float64
	Index    float64
	Basis    float64
	BasisPct float64
}

// BasisLevel is the live gap.
type BasisLevel struct {
	Futures  float64
	Spot     float64
	Delta    float64 // futures − spot
	DeltaPct float64 // (futures−spot)/spot*100
	Kind     string  // premium | discount | flat
	Source   string  // last_vs_index | mark_vs_index
}

// BasisWindowChange is how the gap moved vs a lookback.
type BasisWindowChange struct {
	Window     string
	PastPct    float64
	ChangePct  float64 // currentPct − pastPct (signed, percentage points)
	Trend      string  // expanding | shrinking | stable | flipped
	Complete   bool
	SampleTime time.Time
}

// BasisVenueReport is one exchange's basis picture.
type BasisVenueReport struct {
	Exchange Exchange
	Symbol   string
	Last     BasisLevel // perp last vs spot index (what you see)
	Mark     BasisLevel // mark vs index (what funding uses)
	Windows  []BasisWindowChange
	Trend    string // from 1h when present, else 5m
	Curve    BasisCurve
	Summary  string
	Error    string
}

// BasisContractPoint is one listed contract on the futures curve.
type BasisContractPoint struct {
	Symbol       string
	Kind         string
	Label        string
	Expiry       time.Time
	DaysToExpiry int
	Last         float64
	Mark         float64
	Index        float64
	VsIndex      float64
	VsIndexPct   float64
}

// BasisCurve is how listed futures are priced against each other.
type BasisCurve struct {
	Points    []BasisContractPoint
	Near      *BasisContractPoint
	Far       *BasisContractPoint
	Spread    float64 // far mark − near mark
	SpreadPct float64 // (far−near)/near*100
	Shape     string  // contango | backwardation | flat | humped | unknown
	Slope     string  // up | down | flat
	Richer    string  // near | far | none
	Summary   string
	Complete  bool
	History   []BasisCurvePoint
	Windows   []BasisCurveWindow
	Moves     []BasisCurveMove
}

// BasisAgreement is whether Binance, Bybit, and OKX tell the same story.
type BasisAgreement struct {
	Alignment string // same | opposite | mixed
	Title     string
	Summary   string
}

// BasisReport is the API result.
type BasisReport struct {
	Symbol    string
	Exchange  string
	AsOf      time.Time
	Venues    []BasisVenueReport
	Agreement *BasisAgreement
	Note      string
}

// BasisPort loads live + recent futures-vs-spot basis for one venue.
type BasisPort interface {
	GetBasisQuote(ctx context.Context, symbol string) (*BasisQuote, error)
}

// ParseBasisExchange is binance, bybit, or all.
func ParseBasisExchange(raw string) (string, error) {
	return ParseLiquidationExchange(raw)
}

// ComputeBasis is futures − spot and the percent of spot.
func ComputeBasis(futures, spot float64) (delta, pct float64, kind string) {
	if spot <= 0 || futures <= 0 || math.IsNaN(spot) || math.IsNaN(futures) {
		return 0, 0, BasisKindFlat
	}
	delta = futures - spot
	pct = delta / spot * 100
	switch {
	case pct > basisFlatPct:
		kind = BasisKindPremium
	case pct < -basisFlatPct:
		kind = BasisKindDiscount
	default:
		kind = BasisKindFlat
	}
	return delta, pct, kind
}

// BasisTrendFromChange classifies |gap| getting bigger or smaller.
func BasisTrendFromChange(nowPct, pastPct float64) string {
	if math.IsNaN(nowPct) || math.IsNaN(pastPct) {
		return BasisTrendStable
	}
	nowK := basisKind(nowPct)
	pastK := basisKind(pastPct)
	if nowK != BasisKindFlat && pastK != BasisKindFlat && nowK != pastK {
		return BasisTrendFlipped
	}
	dAbs := math.Abs(nowPct) - math.Abs(pastPct)
	switch {
	case dAbs > basisTrendMovePct:
		return BasisTrendExpanding
	case dAbs < -basisTrendMovePct:
		return BasisTrendShrinking
	default:
		return BasisTrendStable
	}
}

func basisKind(pct float64) string {
	if pct > basisFlatPct {
		return BasisKindPremium
	}
	if pct < -basisFlatPct {
		return BasisKindDiscount
	}
	return BasisKindFlat
}

// FindBasisSample is the latest history point at or before target (within slack).
func FindBasisSample(hist []BasisHistPoint, target time.Time, slack time.Duration) (BasisHistPoint, bool) {
	var best BasisHistPoint
	found := false
	for _, p := range hist {
		if p.Time.IsZero() || p.Time.After(target) {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best = p
			found = true
		}
	}
	if !found {
		return BasisHistPoint{}, false
	}
	return best, !best.Time.Before(target.Add(-slack))
}

// BuildBasisHistory pairs mark and index closes (same open time) into basis points.
func BuildBasisHistory(marks, indexes []PriceSample) []BasisHistPoint {
	idx := map[int64]float64{}
	for _, p := range indexes {
		if p.Time.IsZero() || p.Price <= 0 {
			continue
		}
		idx[p.Time.UTC().UnixMilli()] = p.Price
	}
	out := make([]BasisHistPoint, 0, len(marks))
	for _, m := range marks {
		if m.Time.IsZero() || m.Price <= 0 {
			continue
		}
		ix, ok := idx[m.Time.UTC().UnixMilli()]
		if !ok || ix <= 0 {
			continue
		}
		d, pct, _ := ComputeBasis(m.Price, ix)
		out = append(out, BasisHistPoint{
			Time: m.Time.UTC(), Mark: m.Price, Index: ix, Basis: d, BasisPct: pct,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// PriceSample is a timestamped price (kline close).
type PriceSample struct {
	Time  time.Time
	Price float64
}

// BuildBasisVenue computes live basis, lookback trend, and a short read.
func BuildBasisVenue(q BasisQuote) BasisVenueReport {
	q.Symbol = NormalizeLiquidationSymbol(q.Symbol)
	out := BasisVenueReport{Exchange: q.Exchange, Symbol: q.Symbol, Windows: []BasisWindowChange{}}
	spot := q.SpotIndex
	if spot <= 0 {
		spot = q.SpotLast
	}
	fut := q.FuturesLast
	if fut <= 0 {
		fut = q.FuturesMark
	}
	d, pct, kind := ComputeBasis(fut, spot)
	src := "last_vs_index"
	if q.FuturesLast <= 0 && q.FuturesMark > 0 {
		src = "mark_vs_index"
	}
	out.Last = BasisLevel{Futures: fut, Spot: spot, Delta: d, DeltaPct: pct, Kind: kind, Source: src}
	md, mp, mk := ComputeBasis(q.FuturesMark, q.SpotIndex)
	out.Mark = BasisLevel{
		Futures: q.FuturesMark, Spot: q.SpotIndex, Delta: md, DeltaPct: mp, Kind: mk, Source: "mark_vs_index",
	}

	now := q.Time
	if now.IsZero() {
		now = time.Now().UTC()
	}
	nowPct := out.Mark.DeltaPct
	if out.Mark.Spot <= 0 {
		nowPct = out.Last.DeltaPct
	}
	for _, w := range []struct {
		id  string
		dur time.Duration
	}{
		{TakerWindow5m, 5 * time.Minute},
		{TakerWindow1h, time.Hour},
		{TakerWindow4h, 4 * time.Hour},
	} {
		past, ok := FindBasisSample(q.History, now.Add(-w.dur), basisSampleSlack(w.dur))
		ch := BasisWindowChange{Window: w.id}
		if ok && past.Index > 0 {
			ch.PastPct = past.BasisPct
			ch.ChangePct = nowPct - past.BasisPct
			ch.Trend = BasisTrendFromChange(nowPct, past.BasisPct)
			ch.Complete = true
			ch.SampleTime = past.Time
		}
		out.Windows = append(out.Windows, ch)
	}
	out.Trend = pickBasisTrend(out.Windows)
	out.Curve = AttachBasisCurveHistory(BuildBasisCurve(q), q)
	out.Summary = ExplainBasis(out, q)
	return out
}

// basisSampleSlack is at least one 5m kline so open-time stamps still match.
func basisSampleSlack(window time.Duration) time.Duration {
	s := OpenInterestSampleSlack(window)
	if s < 6*time.Minute {
		return 6 * time.Minute
	}
	return s
}

func pickBasisTrend(ws []BasisWindowChange) string {
	for _, id := range []string{TakerWindow1h, TakerWindow5m, TakerWindow4h} {
		for _, w := range ws {
			if w.Window == id && w.Complete && w.Trend != "" {
				return w.Trend
			}
		}
	}
	return BasisTrendStable
}

// ExplainBasis writes a short read with funding and OI when present.
func ExplainBasis(v BasisVenueReport, q BasisQuote) string {
	ex := basisVenueTitle(v.Exchange)
	lv := v.Last
	if lv.Spot <= 0 {
		return ex + ": no spot/index price to compare."
	}
	gap := fmt.Sprintf("futures %s vs spot %s (%s %s / %s%%)",
		formatQty(lv.Futures), formatQty(lv.Spot), lv.Kind, formatQty(math.Abs(lv.Delta)), FormatSignedPct(lv.DeltaPct))
	head := fmt.Sprintf("%s: perpetual is a %s — %s.", ex, lv.Kind, gap)
	if lv.Kind == BasisKindFlat {
		head = fmt.Sprintf("%s: perpetual is about in line with spot (%s / %s%%).", ex, formatQty(lv.Delta), FormatSignedPct(lv.DeltaPct))
	}
	if v.Trend == BasisTrendExpanding {
		head += " The gap is getting bigger."
	} else if v.Trend == BasisTrendShrinking {
		head += " The gap is getting smaller."
	} else if v.Trend == BasisTrendFlipped {
		head += " The gap flipped side versus recently."
	}

	bits := make([]string, 0, 3)
	if q.FundingRate > 1e-12 {
		bits = append(bits, "longs pay funding")
	} else if q.FundingRate < -1e-12 {
		bits = append(bits, "shorts pay funding")
	}
	if !math.IsNaN(q.OIChange1h) && math.Abs(q.OIChange1h) >= 0.2 {
		bits = append(bits, "OI "+FormatSignedPct(q.OIChange1h)+" over 1h")
	}
	meaning := basisMeaning(lv.Kind, v.Trend, q)
	curve := ""
	if v.Curve.Complete && v.Curve.Summary != "" {
		curve = " " + v.Curve.Summary
	}
	if len(bits) == 0 {
		return head + " " + meaning + curve
	}
	return head + " Together with " + joinAnd(bits) + ": " + meaning + curve
}

func basisMeaning(kind, trend string, q BasisQuote) string {
	fundPos := q.FundingRate > 1e-12
	fundNeg := q.FundingRate < -1e-12
	oiUp := !math.IsNaN(q.OIChange1h) && q.OIChange1h > 0.25
	switch {
	case kind == BasisKindPremium && fundPos && oiUp:
		return "perp above spot with longs paying and OI rising usually means long pressure — traders pay a premium to stay long."
	case kind == BasisKindDiscount && fundNeg && oiUp:
		return "perp below spot with shorts paying and OI rising usually means short pressure — the book is bid for the downside."
	case kind == BasisKindPremium && fundNeg:
		return "perp is rich to spot but shorts still pay funding — the cash-and-carry / premium and the funding crowd disagree; treat both, do not force one story."
	case kind == BasisKindDiscount && fundPos:
		return "perp is cheap to spot while longs pay funding — spot/index may be leading, or perp is lagging a long crowd."
	case kind == BasisKindPremium && trend == BasisTrendExpanding:
		return "a widening premium often means more demand for the perp than for spot (longs willing to pay up)."
	case kind == BasisKindDiscount && trend == BasisTrendExpanding:
		return "a widening discount often means more selling pressure in the perp than in spot."
	case trend == BasisTrendShrinking:
		return "a shrinking gap means perp and spot are converging — the extra long/short pressure in futures is easing."
	case kind == BasisKindPremium:
		return "a premium is typical when longs are eager; check if funding agrees."
	case kind == BasisKindDiscount:
		return "a discount is typical when shorts are eager or spot is bid versus the perp."
	default:
		return "no strong premium or discount right now."
	}
}

func basisVenueTitle(ex Exchange) string {
	switch ex {
	case ExchangeBinance:
		return "Binance"
	case ExchangeBybit:
		return "Bybit"
	case ExchangeOKX:
		return "OKX"
	default:
		if ex == "" {
			return ""
		}
		return string(ex)
	}
}

func basisQuoteUsable(v *BasisVenueReport) bool {
	return v != nil && v.Error == "" && v.Last.Spot > 0 && v.Last.Futures > 0
}

// CompareBasisVenues says whether Binance, Bybit, and OKX agree.
// A missing or unusable quote leaves the comparison unknown.
func CompareBasisVenues(venues []BasisVenueReport) *BasisAgreement {
	byEx := make(map[Exchange]*BasisVenueReport, len(venues))
	for i := range venues {
		byEx[venues[i].Exchange] = &venues[i]
	}
	got := make([]*BasisVenueReport, 0, 3)
	missing := make([]string, 0, 3)
	for _, ex := range FuturesVenues() {
		v := byEx[ex]
		if !basisQuoteUsable(v) {
			missing = append(missing, basisVenueTitle(ex))
			continue
		}
		got = append(got, v)
	}
	if len(missing) > 0 {
		verb := "is"
		if len(missing) != 1 {
			verb = "are"
		}
		return &BasisAgreement{
			Alignment: AlignUnknown,
			Title:     "Need Binance, Bybit, and OKX to compare",
			Summary:   fmt.Sprintf("%s %s missing, so the venues are not compared.", joinAnd(missing), verb),
		}
	}
	kinds := map[string]int{}
	minPct, maxPct := got[0].Last.DeltaPct, got[0].Last.DeltaPct
	for _, v := range got {
		kinds[v.Last.Kind]++
		if v.Last.DeltaPct < minPct {
			minPct = v.Last.DeltaPct
		}
		if v.Last.DeltaPct > maxPct {
			maxPct = v.Last.DeltaPct
		}
	}
	flat := kinds[BasisKindFlat] > 0
	premium := kinds[BasisKindPremium] > 0
	discount := kinds[BasisKindDiscount] > 0
	align := AlignSame
	switch {
	case flat && (premium || discount):
		align = AlignMixed
	case premium && discount:
		align = AlignOpposite
	}
	spread := math.Abs(maxPct - minPct)
	veryDiff := !flat && align == AlignSame && spread >= 0.05
	parts := make([]string, len(got))
	kindParts := make([]string, len(got))
	for i, v := range got {
		title := basisVenueTitle(v.Exchange)
		parts[i] = fmt.Sprintf("%s %s%%", title, FormatSignedPct(v.Last.DeltaPct))
		kindParts[i] = fmt.Sprintf("%s is a %s (%s%%)", title, v.Last.Kind, FormatSignedPct(v.Last.DeltaPct))
	}
	kind := got[0].Last.Kind
	title := "Binance, Bybit, and OKX show the same kind of gap"
	summary := fmt.Sprintf("All three are a %s (%s).", kind, joinAnd(parts))
	if align == AlignOpposite {
		title = "The venues disagree on premium vs discount"
		summary = fmt.Sprintf("%s. Combined 'the market is rich/cheap' would hide that split.", joinAnd(kindParts))
	} else if align == AlignMixed {
		title = "One venue is flat and another is not"
		summary = joinAnd(kindParts) + "."
	} else if veryDiff {
		title = "Same side, but the size is very different"
		summary = fmt.Sprintf("All three are a %s, but the size differs (%s, gap %s pp).", kind, joinAnd(parts), FormatSignedPct(spread))
		align = AlignMixed
	}
	return &BasisAgreement{Alignment: align, Title: title, Summary: summary}
}

// BuildBasisCurve ranks listed contracts by days to expiry and says if the
// far contract is richer (contango / up) or cheaper (backwardation / down).
func BuildBasisCurve(q BasisQuote) BasisCurve {
	now := q.Time
	if now.IsZero() {
		now = time.Now().UTC()
	}
	pts := make([]BasisContractPoint, 0, len(q.Contracts)+1)
	seen := map[string]bool{}
	add := func(c BasisContractQuote) {
		sym := strings.TrimSpace(c.Symbol)
		if sym == "" || seen[sym] {
			return
		}
		px := c.Mark
		if px <= 0 {
			px = c.Last
		}
		if px <= 0 {
			return
		}
		idx := c.Index
		if idx <= 0 {
			idx = q.SpotIndex
		}
		dte := 0
		if !c.Expiry.IsZero() && c.Kind != BasisContractPerp {
			dte = int(c.Expiry.UTC().Sub(now).Hours() / 24)
			if dte < 0 {
				return
			}
		}
		vs, vsPct, _ := ComputeBasis(px, idx)
		pts = append(pts, BasisContractPoint{
			Symbol: sym, Kind: contractKind(c.Kind, c.Expiry),
			Label: contractLabel(c.Kind, c.Expiry), Expiry: c.Expiry.UTC(),
			DaysToExpiry: dte, Last: c.Last, Mark: px, Index: idx,
			VsIndex: vs, VsIndexPct: vsPct,
		})
		seen[sym] = true
	}
	for _, c := range q.Contracts {
		add(c)
	}
	if q.FuturesMark > 0 || q.FuturesLast > 0 {
		add(BasisContractQuote{
			Symbol: q.Symbol, Kind: BasisContractPerp,
			Last: q.FuturesLast, Mark: q.FuturesMark, Index: q.SpotIndex,
		})
	}
	sort.SliceStable(pts, func(i, j int) bool {
		if pts[i].DaysToExpiry != pts[j].DaysToExpiry {
			return pts[i].DaysToExpiry < pts[j].DaysToExpiry
		}
		return pts[i].Symbol < pts[j].Symbol
	})
	out := BasisCurve{Points: pts, Shape: BasisCurveUnknown, Slope: BasisSlopeFlat, Richer: BasisRicherNone}
	if len(pts) < 2 {
		out.Summary = "No dated futures listed to compare against the perpetual."
		return out
	}
	near, far := pickCurveEnds(pts)
	if near == nil || far == nil || near.Mark <= 0 {
		out.Summary = "Need two futures prices to read the curve."
		return out
	}
	n, f := *near, *far
	out.Near, out.Far = &n, &f
	out.Spread = f.Mark - n.Mark
	out.SpreadPct = out.Spread / n.Mark * 100
	out.Shape, out.Slope, out.Richer = basisCurveShape(out.SpreadPct)
	out.Complete = true
	out.Summary = explainBasisCurve(out)
	if curveIsHumped(pts) && out.Shape != BasisKindFlat {
		out.Shape = BasisCurveHumped
		out.Summary += " The path between those two is not a straight slope."
	}
	return out
}

func pickCurveEnds(pts []BasisContractPoint) (near, far *BasisContractPoint) {
	var dated []BasisContractPoint
	var perp *BasisContractPoint
	for i := range pts {
		p := &pts[i]
		if p.Kind == BasisContractPerp || p.Expiry.IsZero() {
			if perp == nil {
				perp = p
			}
			continue
		}
		dated = append(dated, *p)
	}
	if len(dated) >= 2 {
		n, f := dated[0], dated[len(dated)-1]
		return &n, &f
	}
	if len(dated) == 1 && perp != nil {
		d := dated[0]
		return perp, &d
	}
	if len(pts) >= 2 {
		a, b := pts[0], pts[len(pts)-1]
		return &a, &b
	}
	return nil, nil
}

func curveIsHumped(pts []BasisContractPoint) bool {
	if len(pts) < 3 {
		return false
	}
	up, down := false, false
	for i := 1; i < len(pts); i++ {
		d := pts[i].Mark - pts[i-1].Mark
		pct := 0.0
		if pts[i-1].Mark > 0 {
			pct = d / pts[i-1].Mark * 100
		}
		if pct > basisFlatPct {
			up = true
		} else if pct < -basisFlatPct {
			down = true
		}
	}
	return up && down
}

func contractKind(kind string, expiry time.Time) string {
	switch kind {
	case BasisContractPerp, BasisContractCurrentQuarter, BasisContractNextQuarter, BasisContractDated:
		return kind
	default:
		if expiry.IsZero() {
			return BasisContractPerp
		}
		return BasisContractDated
	}
}

func contractLabel(kind string, expiry time.Time) string {
	if kind == BasisContractPerp || expiry.IsZero() {
		return "Perp"
	}
	return expiry.UTC().Format("02 Jan 2006")
}

func explainBasisCurve(c BasisCurve) string {
	if c.Near == nil || c.Far == nil {
		return ""
	}
	near, far := c.Near.Label, c.Far.Label
	gap := fmt.Sprintf("%s / %s%%", formatQty(math.Abs(c.Spread)), FormatSignedPct(c.SpreadPct))
	switch c.Richer {
	case BasisRicherFar:
		return fmt.Sprintf("Far expiry (%s) is more expensive than near (%s) by %s — the curve is in contango (up).", far, near, gap)
	case BasisRicherNear:
		return fmt.Sprintf("Near expiry (%s) is more expensive than far (%s) by %s — the curve is in backwardation (down).", near, far, gap)
	default:
		return fmt.Sprintf("Near (%s) and far (%s) expiries are about in line (%s).", near, far, gap)
	}
}
