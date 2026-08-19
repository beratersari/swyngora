package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
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
	Summary  string
	Error    string
}

// BasisAgreement is whether Binance and Bybit tell the same story.
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
	ex := string(v.Exchange)
	if ex == string(ExchangeBinance) {
		ex = "Binance"
	} else if ex == string(ExchangeBybit) {
		ex = "Bybit"
	}
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
	if len(bits) == 0 {
		return head + " " + meaning
	}
	return head + " Together with " + joinAnd(bits) + ": " + meaning
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

// CompareBasisVenues says whether Binance and Bybit agree.
func CompareBasisVenues(venues []BasisVenueReport) *BasisAgreement {
	var bin, byb *BasisVenueReport
	for i := range venues {
		v := &venues[i]
		switch v.Exchange {
		case ExchangeBinance:
			bin = v
		case ExchangeBybit:
			byb = v
		}
	}
	if bin == nil || byb == nil {
		return &BasisAgreement{
			Alignment: AlignUnknown,
			Title:     "Need both venues to compare",
			Summary:   "One of Binance or Bybit is missing, so we cannot say if they agree.",
		}
	}
	bk, yk := bin.Last.Kind, byb.Last.Kind
	align := AlignSame
	switch {
	case bk == BasisKindFlat || yk == BasisKindFlat:
		if bk != yk {
			align = AlignMixed
		}
	case bk != yk:
		align = AlignOpposite
	}
	spread := math.Abs(bin.Last.DeltaPct - byb.Last.DeltaPct)
	veryDiff := spread >= 0.05 && align == AlignSame
	title := "Binance and Bybit show the same kind of gap"
	summary := fmt.Sprintf("Both are a %s (Binance %s%%, Bybit %s%%).",
		bk, FormatSignedPct(bin.Last.DeltaPct), FormatSignedPct(byb.Last.DeltaPct))
	if align == AlignOpposite {
		title = "Binance and Bybit disagree on premium vs discount"
		summary = fmt.Sprintf("Binance is a %s (%s%%), Bybit is a %s (%s%%). Combined 'the market is rich/cheap' would hide that split.",
			bk, FormatSignedPct(bin.Last.DeltaPct), yk, FormatSignedPct(byb.Last.DeltaPct))
	} else if align == AlignMixed {
		title = "One venue is flat, the other is not"
		summary = fmt.Sprintf("Binance %s (%s%%), Bybit %s (%s%%).",
			bk, FormatSignedPct(bin.Last.DeltaPct), yk, FormatSignedPct(byb.Last.DeltaPct))
	} else if veryDiff {
		title = "Same side, but the size is very different"
		summary = fmt.Sprintf("Both are a %s, but Binance is %s%% and Bybit is %s%% (gap %s pp). That is a real split in how rich/cheap each book is.",
			bk, FormatSignedPct(bin.Last.DeltaPct), FormatSignedPct(byb.Last.DeltaPct), FormatSignedPct(spread))
		align = AlignMixed
	}
	return &BasisAgreement{Alignment: align, Title: title, Summary: summary}
}
