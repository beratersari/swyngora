package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

const (
	CVDVsConfirms   = "confirms"
	CVDVsOpposite   = "opposite"
	CVDVsAbsorption = "absorption"
	CVDVsMixed      = "mixed"

	CVDWindow1h  = "1h"
	CVDWindow4h  = "4h"
	CVDWindow24h = "24h"

	CVDBar             = 5 * time.Minute
	DefaultCVDLookback = 24 * time.Hour
	cvdPriceFlatPct    = 0.20
	cvdMoveShare       = 0.06
	MaxCVDPoints       = 320
)

// CVDWindows are the CVD vs price lookbacks.
var CVDWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{CVDWindow1h, time.Hour},
	{CVDWindow4h, 4 * time.Hour},
	{CVDWindow24h, 24 * time.Hour},
}

// CVDPoint is one 5-minute bar of aggressive flow and cumulative delta.
type CVDPoint struct {
	Time         time.Time
	Price        float64
	BuyNotional  float64
	SellNotional float64
	Delta        float64
	CVD          float64
}

// CVDWindowStat is how CVD and price moved over one lookback.
type CVDWindowStat struct {
	Window         string
	CVDChange      float64
	CVDChangePct   float64 // vs buy+sell in the window
	PriceChangePct float64
	BuyNotional    float64
	SellNotional   float64
	VsPrice        string
	Title          string
	Summary        string
	Complete       bool
}

// CVDVenueSeries is one venue's CVD path plus window reads.
type CVDVenueSeries struct {
	Exchange  Exchange
	Symbol    string
	Points    []CVDPoint
	Windows   []CVDWindowStat
	LastCVD   float64
	LastPrice float64
	Summary   string
	Error     string
	Complete  bool
}

// CVDReport is the API result.
type CVDReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Venues   []CVDVenueSeries
	Combined *CVDVenueSeries
	Summary  string
	Note     string
}

// TakerBucketPort loads raw buy/sell bars for CVD.
type TakerBucketPort interface {
	GetTakerBuckets(ctx context.Context, symbol string) ([]TakerBucket, error)
}

// TakerBucketStore persists taker minutes/bars so Bybit CVD can grow past the live book.
type TakerBucketStore interface {
	UpsertTakerBuckets(ctx context.Context, recs []TakerBucket) (int, error)
	ListTakerBuckets(ctx context.Context, exchange, symbol string, from, to time.Time) ([]TakerBucket, error)
	PurgeTakerBuckets(ctx context.Context, cutoff time.Time) (int, error)
}

// ResampleTakerBuckets folds bars into a coarser step (e.g. 1m → 5m).
func ResampleTakerBuckets(in []TakerBucket, step time.Duration) []TakerBucket {
	if step <= 0 {
		step = CVDBar
	}
	acc := map[int64]*TakerBucket{}
	for _, b := range in {
		if b.Start.IsZero() || (b.BuyNotional <= 0 && b.SellNotional <= 0) {
			continue
		}
		ms := TruncateToBucket(b.Start.UTC(), step).UnixMilli()
		cur := acc[ms]
		if cur == nil {
			cur = &TakerBucket{
				Exchange: b.Exchange, Symbol: NormalizeLiquidationSymbol(b.Symbol),
				Start: time.UnixMilli(ms).UTC(),
			}
			acc[ms] = cur
		}
		cur.BuyNotional += b.BuyNotional
		cur.SellNotional += b.SellNotional
		if cur.Exchange == "" {
			cur.Exchange = b.Exchange
		}
	}
	out := make([]TakerBucket, 0, len(acc))
	for _, b := range acc {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// MergeTakerBuckets unions stored and live bars; live wins on the same start.
func MergeTakerBuckets(stored, live []TakerBucket) []TakerBucket {
	acc := map[int64]TakerBucket{}
	add := func(list []TakerBucket, overwrite bool) {
		for _, b := range list {
			if b.Start.IsZero() {
				continue
			}
			ms := b.Start.UTC().UnixMilli()
			if _, ok := acc[ms]; ok && !overwrite {
				continue
			}
			b.Start = time.UnixMilli(ms).UTC()
			b.Symbol = NormalizeLiquidationSymbol(b.Symbol)
			acc[ms] = b
		}
	}
	add(stored, true)
	add(live, true)
	out := make([]TakerBucket, 0, len(acc))
	for _, b := range acc {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// BuildCVDSeries accumulates buy−sell and joins price.
func BuildCVDSeries(ex Exchange, symbol string, buckets []TakerBucket, prices []CVDPrice, now time.Time, started time.Time) CVDVenueSeries {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	cut := now.Add(-DefaultCVDLookback)
	bars := ResampleTakerBuckets(buckets, CVDBar)
	out := CVDVenueSeries{Exchange: ex, Symbol: symbol, Points: []CVDPoint{}, Windows: []CVDWindowStat{}}
	var cvd float64
	for _, b := range bars {
		if b.Start.Before(cut) {
			continue
		}
		delta := b.BuyNotional - b.SellNotional
		cvd += delta
		pt := CVDPoint{
			Time: b.Start, BuyNotional: b.BuyNotional, SellNotional: b.SellNotional,
			Delta: delta, CVD: cvd, Price: priceAt(prices, b.Start),
		}
		out.Points = append(out.Points, pt)
	}
	if len(out.Points) > MaxCVDPoints {
		out.Points = out.Points[len(out.Points)-MaxCVDPoints:]
	}
	if n := len(out.Points); n > 0 {
		out.LastCVD = out.Points[n-1].CVD
		out.LastPrice = out.Points[n-1].Price
	}
	for _, w := range CVDWindows {
		out.Windows = append(out.Windows, summarizeCVDWindow(out.Points, w.ID, now.Add(-w.Dur), now, started))
	}
	out.Complete = !started.IsZero() && !started.After(cut)
	out.Summary = ExplainCVDVenue(out)
	return out
}

// CombineCVDVenues sums per-bar delta then re-accumulates (never averages).
func CombineCVDVenues(symbol string, venues []CVDVenueSeries, prices []CVDPrice, now time.Time) *CVDVenueSeries {
	if len(venues) == 0 {
		return nil
	}
	type sum struct{ buy, sell float64 }
	acc := map[int64]*sum{}
	anyComplete := false
	started := time.Time{}
	for _, v := range venues {
		if v.Error != "" || len(v.Points) == 0 {
			continue
		}
		if v.Complete {
			anyComplete = true
		}
		for _, p := range v.Points {
			ms := TruncateToBucket(p.Time, CVDBar).UnixMilli()
			cur := acc[ms]
			if cur == nil {
				cur = &sum{}
				acc[ms] = cur
			}
			cur.buy += p.BuyNotional
			cur.sell += p.SellNotional
		}
	}
	if len(acc) == 0 {
		return nil
	}
	buckets := make([]TakerBucket, 0, len(acc))
	for ms, s := range acc {
		buckets = append(buckets, TakerBucket{
			Exchange: "all", Symbol: symbol, Start: time.UnixMilli(ms).UTC(),
			BuyNotional: s.buy, SellNotional: s.sell,
		})
	}
	if anyComplete {
		started = now.Add(-DefaultCVDLookback)
	}
	out := BuildCVDSeries("all", symbol, buckets, prices, now, started)
	out.Complete = anyComplete
	out.Summary = ExplainCVDVenue(out)
	return &out
}

// CVDPrice is a close used to plot CVD against price.
type CVDPrice struct {
	Time  time.Time
	Close float64
}

// CVDPricesFromCandles maps candle closes.
func CVDPricesFromCandles(c []Candle) []CVDPrice {
	out := make([]CVDPrice, 0, len(c))
	for _, bar := range c {
		px, err := parseFloat(bar.Close)
		if err != nil || px <= 0 || bar.OpenTime.IsZero() {
			continue
		}
		out = append(out, CVDPrice{Time: bar.OpenTime.UTC(), Close: px})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// CVDVsPrice labels how CVD moved versus price.
func CVDVsPrice(pricePct, cvdChange, volume float64) string {
	share := 0.0
	if volume > 0 {
		share = cvdChange / volume
	}
	priceSig := math.Abs(pricePct) >= cvdPriceFlatPct
	cvdSig := math.Abs(share) >= cvdMoveShare
	switch {
	case !priceSig && cvdSig:
		return CVDVsAbsorption
	case priceSig && cvdSig && signFloat(pricePct) != signFloat(cvdChange):
		return CVDVsOpposite
	case priceSig && cvdSig && signFloat(pricePct) == signFloat(cvdChange):
		return CVDVsConfirms
	default:
		return CVDVsMixed
	}
}

// ExplainCVDVenue writes a short 4h-first read.
func ExplainCVDVenue(v CVDVenueSeries) string {
	if v.Error != "" {
		return v.Error
	}
	var w4 CVDWindowStat
	for _, w := range v.Windows {
		if w.Window == CVDWindow4h {
			w4 = w
			break
		}
	}
	if w4.Window == "" && len(v.Windows) > 0 {
		w4 = v.Windows[0]
	}
	if w4.Window == "" {
		return prettyBase(v.Symbol) + ": not enough aggressive volume yet for CVD."
	}
	return fmt.Sprintf("%s %s: %s", prettyBase(v.Symbol), w4.Window, w4.Summary)
}

// ExplainCVDReport picks a combined or first-venue line.
func ExplainCVDReport(r CVDReport) string {
	if r.Combined != nil && r.Combined.Summary != "" {
		return "Combined " + r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Summary != "" && v.Error == "" {
			return string(v.Exchange) + ": " + v.Summary
		}
	}
	return "No CVD yet."
}

func summarizeCVDWindow(points []CVDPoint, window string, from, now, started time.Time) CVDWindowStat {
	out := CVDWindowStat{Window: window}
	var first, last *CVDPoint
	var buy, sell float64
	for i := range points {
		p := &points[i]
		if p.Time.Before(from) {
			continue
		}
		if first == nil {
			first = p
		}
		last = p
		buy += p.BuyNotional
		sell += p.SellNotional
	}
	out.BuyNotional, out.SellNotional = buy, sell
	out.Complete = !started.IsZero() && !started.After(from)
	if first == nil || last == nil {
		out.VsPrice = CVDVsMixed
		out.Title = "no data"
		out.Summary = "Not enough bars in this window."
		return out
	}
	out.CVDChange = last.CVD - first.CVD
	if tot := buy + sell; tot > 0 {
		out.CVDChangePct = out.CVDChange / tot * 100
	}
	if first.Price > 0 && last.Price > 0 {
		out.PriceChangePct = (last.Price - first.Price) / first.Price * 100
	}
	out.VsPrice = CVDVsPrice(out.PriceChangePct, out.CVDChange, buy+sell)
	out.Title = cvdTitle(out.VsPrice)
	out.Summary = explainCVDWindow(out)
	return out
}

func explainCVDWindow(w CVDWindowStat) string {
	cvdWord := "CVD was flat"
	if w.CVDChange > 0 {
		cvdWord = "CVD rose (more market buys)"
	} else if w.CVDChange < 0 {
		cvdWord = "CVD fell (more market sells)"
	}
	pxWord := "price was little changed"
	if w.PriceChangePct >= cvdPriceFlatPct {
		pxWord = "price rose"
	} else if w.PriceChangePct <= -cvdPriceFlatPct {
		pxWord = "price fell"
	}
	switch w.VsPrice {
	case CVDVsAbsorption:
		return cvdWord + " but " + pxWord + " — aggressive flow is not moving price (absorption)."
	case CVDVsOpposite:
		return pxWord + " while " + cvdWord + " — price and aggressive flow disagree."
	case CVDVsConfirms:
		return cvdWord + " and " + pxWord + " — flow and price agree."
	default:
		return cvdWord + "; " + pxWord + "."
	}
}

func cvdTitle(vs string) string {
	switch vs {
	case CVDVsAbsorption:
		return "absorption"
	case CVDVsOpposite:
		return "flow vs price split"
	case CVDVsConfirms:
		return "flow confirms price"
	default:
		return "mixed"
	}
}

func priceAt(prices []CVDPrice, t time.Time) float64 {
	if len(prices) == 0 || t.IsZero() {
		return 0
	}
	var best CVDPrice
	found := false
	for _, p := range prices {
		if p.Time.After(t.Add(CVDBar)) {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best, found = p, true
		}
	}
	if !found {
		return 0
	}
	return best.Close
}

func signFloat(v float64) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	default:
		return 0
	}
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
