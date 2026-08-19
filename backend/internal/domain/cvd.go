package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	CVDVsConfirms   = "confirms"
	CVDVsOpposite   = "opposite"
	CVDVsAbsorption = "absorption"
	CVDVsMixed      = "mixed"

	CVDDivPriceUpCVDDown = "price_up_cvd_down"
	CVDDivPriceDownCVDUp = "price_down_cvd_up"

	CVDWindow15m = "15m"
	CVDWindow1h  = "1h"
	CVDWindow4h  = "4h"
	CVDWindow24h = "24h"

	CVDDirUp   = "up"
	CVDDirDown = "down"
	CVDDirFlat = "flat"

	CVDMarketSpot    = "spot"
	CVDMarketFutures = "futures"

	CVDBar             = 5 * time.Minute
	DefaultCVDLookback = 24 * time.Hour
	cvdPriceFlatPct    = 0.20
	cvdBarPriceFlatPct = 0.05
	cvdMoveShare       = 0.06
	MaxCVDPoints       = 320
)

// CVDWindows are the CVD vs price lookbacks.
var CVDWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{CVDWindow15m, 15 * time.Minute},
	{CVDWindow1h, time.Hour},
	{CVDWindow4h, 4 * time.Hour},
	{CVDWindow24h, 24 * time.Hour},
}

// CVDPoint is one 5-minute bar of aggressive flow and cumulative delta.
type CVDPoint struct {
	Time           time.Time
	Price          float64
	PriceChangePct float64 // vs previous bar close
	BuyNotional    float64
	SellNotional   float64
	Delta          float64
	CVD            float64
	VsPrice        string // confirms | opposite | absorption | mixed
	Divergence     string // price_up_cvd_down | price_down_cvd_up | ""
	Shares         []CVDShare
}

// CVDShare is how much one venue added to combined CVD.
type CVDShare struct {
	Exchange Exchange
	Delta    float64 // this bar or this window's CVD change
	CVD      float64 // running contribution (overlap start = 0)
	SharePct float64 // of combined CVD (or of combined window change)
}

// CVDDivergence is price vs CVD disagreement (price up / CVD down or the reverse).
type CVDDivergence struct {
	Kind             string // price_up_cvd_down | price_down_cvd_up | ""
	VsPrice          string
	Title            string
	Summary          string
	Bars             int // opposite bars in the whole series
	LastAt           time.Time
	Since            time.Time // start of the current (or latest) run
	Duration         string    // e.g. "35m", "2h"
	DurationSeconds  int
	PriceMovePct     float64 // price change over that run
	CVDMove          float64 // CVD change over that run
	CVDMovePct       float64 // CVD move vs buy+sell in the run
}

// CVDVenueSplit is when one venue's CVD rises while the other falls.
type CVDVenueSplit struct {
	Alignment     string // same | opposite | mixed
	Binance       string // up | down | flat
	Bybit         string
	BinanceChange float64
	BybitChange   float64
	Title         string
	Summary       string
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
	Divergence     string
	Title          string
	Summary        string
	Complete       bool
	Contributions  []CVDShare
	VenueSplit     *CVDVenueSplit
}

// CVDSpotFutures is when spot CVD and futures CVD move the same or opposite ways.
type CVDSpotFutures struct {
	Alignment     string // same | opposite | mixed
	Spot          string // up | down | flat
	Futures       string
	SpotChange    float64
	FuturesChange float64
	Window        string
	Title         string
	Summary       string
	Windows       []CVDSpotFutures
}

// CVDVenueSeries is one venue's CVD path plus window reads.
type CVDVenueSeries struct {
	Exchange       Exchange
	Symbol         string
	Market         string // spot | futures
	Points         []CVDPoint
	Windows        []CVDWindowStat
	LastCVD        float64
	LastPrice      float64
	Contributions  []CVDShare
	OverlapFrom    *time.Time
	OverlapTo      *time.Time
	Divergence     CVDDivergence
	VenueSplit     *CVDVenueSplit
	Summary        string
	Error          string
	Complete       bool
}

// CVDReport is the API result.
type CVDReport struct {
	Symbol        string
	Exchange      string
	AsOf          time.Time
	Venues        []CVDVenueSeries // futures
	Combined      *CVDVenueSeries  // futures combined
	SpotVenues    []CVDVenueSeries
	SpotCombined  *CVDVenueSeries
	SpotFutures   *CVDSpotFutures
	Summary       string
	Note          string
}

// TakerBucketPort loads raw buy/sell bars for CVD.
type TakerBucketPort interface {
	GetTakerBuckets(ctx context.Context, symbol string) ([]TakerBucket, error)
}

// SpotTakerBucketPort loads spot aggressive buy/sell bars (Bybit recent trades).
type SpotTakerBucketPort interface {
	GetSpotTakerBuckets(ctx context.Context, symbol string) ([]TakerBucket, error)
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
	out := CVDVenueSeries{Exchange: ex, Symbol: symbol, Market: CVDMarketFutures, Points: []CVDPoint{}, Windows: []CVDWindowStat{}}
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
	annotateCVDPoints(out.Points)
	if n := len(out.Points); n > 0 {
		out.LastCVD = out.Points[n-1].CVD
		out.LastPrice = out.Points[n-1].Price
	}
	for _, w := range CVDWindows {
		out.Windows = append(out.Windows, summarizeCVDWindow(out.Points, w.ID, now.Add(-w.Dur), now, started))
	}
	out.Complete = cvdCoverageComplete(started, cut)
	out.Divergence = summarizeCVDDivergence(out.Points, out.Windows)
	out.Summary = ExplainCVDVenue(out)
	return out
}

// CombineCVDVenues sums per-bar delta on the overlap both venues have (never averages).
// Combined is complete only when every contributing venue is complete and the overlap
// covers the full lookback — a short Bybit history must not make a Binance-heavy series
// look finished.
func CombineCVDVenues(symbol string, venues []CVDVenueSeries, prices []CVDPrice, now time.Time) *CVDVenueSeries {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	out := &CVDVenueSeries{
		Exchange: "all", Symbol: symbol, Market: CVDMarketFutures,
		Points: []CVDPoint{}, Windows: []CVDWindowStat{},
	}
	var usable []CVDVenueSeries
	for _, v := range venues {
		if v.Error != "" || len(v.Points) == 0 {
			continue
		}
		usable = append(usable, v)
	}
	if len(usable) < 2 {
		out.Windows = emptyCVDWindows(now, time.Time{})
		out.Summary = "Combined CVD only uses the time range both Binance and Bybit have. One venue is still filling in — not shown as complete."
		return out
	}
	sort.Slice(usable, func(i, j int) bool {
		return string(usable[i].Exchange) < string(usable[j].Exchange)
	})
	if usable[0].Market != "" {
		out.Market = usable[0].Market
	}

	from, to, ok := overlapCVDRange(usable)
	if !ok {
		out.Windows = emptyCVDWindows(now, time.Time{})
		out.Summary = "Combined CVD needs overlapping 5-minute bars on both venues. Bybit history is still filling in — not shown as complete."
		return out
	}
	out.OverlapFrom = &from
	out.OverlapTo = &to

	indexes := make([]map[int64]CVDPoint, len(usable))
	for i, v := range usable {
		m := make(map[int64]CVDPoint, len(v.Points))
		for _, p := range v.Points {
			m[TruncateToBucket(p.Time, CVDBar).UnixMilli()] = p
		}
		indexes[i] = m
	}

	venueCvd := make([]float64, len(usable))
	var cvd float64
	for at := from; !at.After(to); at = at.Add(CVDBar) {
		ms := at.UnixMilli()
		pt := CVDPoint{Time: at}
		shares := make([]CVDShare, 0, len(usable))
		for i, v := range usable {
			src := indexes[i][ms] // zero value if that 5m slot is empty on one venue
			pt.BuyNotional += src.BuyNotional
			pt.SellNotional += src.SellNotional
			pt.Delta += src.Delta
			venueCvd[i] += src.Delta
			shares = append(shares, CVDShare{
				Exchange: v.Exchange, Delta: src.Delta, CVD: venueCvd[i],
			})
		}
		cvd += pt.Delta
		pt.CVD = cvd
		pt.Price = priceAt(prices, pt.Time)
		for i := range shares {
			if cvd != 0 {
				shares[i].SharePct = shares[i].CVD / cvd * 100
			}
		}
		pt.Shares = shares
		out.Points = append(out.Points, pt)
	}
	if len(out.Points) > MaxCVDPoints {
		out.Points = out.Points[len(out.Points)-MaxCVDPoints:]
	}
	annotateCVDPoints(out.Points)
	if n := len(out.Points); n > 0 {
		out.LastCVD = out.Points[n-1].CVD
		out.LastPrice = out.Points[n-1].Price
		out.Contributions = append([]CVDShare(nil), out.Points[n-1].Shares...)
	}
	started := from
	for _, w := range CVDWindows {
		stat := summarizeCVDWindow(out.Points, w.ID, now.Add(-w.Dur), now, started)
		stat.Contributions = windowShares(out.Points, now.Add(-w.Dur), usable)
		if split := venueSplitFromShares(stat.Contributions); split != nil {
			stat.VenueSplit = split
		}
		out.Windows = append(out.Windows, stat)
	}
	allComplete := true
	for _, v := range usable {
		if !v.Complete {
			allComplete = false
			break
		}
	}
	// Both venues already have a full lookback: do not fail complete just
	// because the first shared 5m slot is a bucket later than now-24h.
	out.Complete = allComplete
	out.Divergence = summarizeCVDDivergence(out.Points, out.Windows)
	out.VenueSplit = pickCVDVenueSplit(out.Windows)
	out.Summary = ExplainCVDVenue(*out)
	if !out.Complete {
		out.Summary = strings.TrimSpace(overlapIncompleteNote(from, now) + " " + out.Summary)
	}
	return out
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
	head := fmt.Sprintf("%s %s: %s", prettyBase(v.Symbol), w4.Window, w4.Summary)
	if bits := explainCVDChanges(v.Windows); bits != "" {
		head += " " + bits
	}
	if len(v.Contributions) > 0 {
		head += " " + explainCVDContributions(v.Contributions)
	}
	if v.VenueSplit != nil && v.VenueSplit.Alignment == AlignOpposite && v.VenueSplit.Summary != "" {
		head += " " + v.VenueSplit.Summary
	}
	if v.Divergence.Summary != "" && v.Divergence.Kind != "" {
		head += " " + v.Divergence.Summary
	}
	return head
}

// ExplainCVDReport picks a combined or first-venue line.
func ExplainCVDReport(r CVDReport) string {
	head := ""
	if r.Combined != nil && r.Combined.Summary != "" {
		head = "Futures " + r.Combined.Summary
	} else {
		for _, v := range r.Venues {
			if v.Summary != "" && v.Error == "" {
				head = "Futures " + string(v.Exchange) + ": " + v.Summary
				break
			}
		}
	}
	if r.SpotFutures != nil && r.SpotFutures.Alignment == AlignOpposite && r.SpotFutures.Summary != "" {
		if head != "" {
			return head + " " + r.SpotFutures.Summary
		}
		return r.SpotFutures.Summary
	}
	if r.SpotCombined != nil && r.SpotCombined.Summary != "" && head != "" {
		return head + " Spot " + r.SpotCombined.Summary
	}
	if head != "" {
		return head
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
	out.Complete = cvdCoverageComplete(started, from)
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
	out.Divergence = windowDivergence(out.PriceChangePct, out.CVDChange)
	out.Title = cvdTitle(out.VsPrice)
	if out.Divergence != "" {
		out.Title = cvdDivergenceTitle(out.Divergence)
	}
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
	nums := fmt.Sprintf("CVD %s over %s, price %s%%",
		FormatSignedQty(w.CVDChange), w.Window, FormatSignedPct(w.PriceChangePct))
	switch w.VsPrice {
	case CVDVsAbsorption:
		return nums + " — " + cvdWord + " but " + pxWord + " (absorption)."
	case CVDVsOpposite:
		return nums + " — " + pxWord + " while " + cvdWord + " (price and flow disagree)."
	case CVDVsConfirms:
		return nums + " — " + cvdWord + " and " + pxWord + "."
	default:
		return nums + " — " + cvdWord + "; " + pxWord + "."
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

func annotateCVDPoints(points []CVDPoint) {
	var prev float64
	for i := range points {
		p := &points[i]
		if prev > 0 && p.Price > 0 {
			p.PriceChangePct = (p.Price - prev) / prev * 100
		}
		vol := math.Abs(p.Delta)
		p.VsPrice = CVDVsPrice(p.PriceChangePct, p.Delta, vol)
		p.Divergence = barDivergence(p.PriceChangePct, p.Delta)
		if p.Price > 0 {
			prev = p.Price
		}
	}
}

func barDivergence(pricePct, delta float64) string {
	if math.Abs(pricePct) < cvdBarPriceFlatPct || delta == 0 {
		return ""
	}
	if pricePct > 0 && delta < 0 {
		return CVDDivPriceUpCVDDown
	}
	if pricePct < 0 && delta > 0 {
		return CVDDivPriceDownCVDUp
	}
	return ""
}

func windowDivergence(pricePct, cvdChange float64) string {
	if math.Abs(pricePct) < cvdPriceFlatPct || cvdChange == 0 {
		return ""
	}
	if signFloat(pricePct) > 0 && signFloat(cvdChange) < 0 {
		return CVDDivPriceUpCVDDown
	}
	if signFloat(pricePct) < 0 && signFloat(cvdChange) > 0 {
		return CVDDivPriceDownCVDUp
	}
	return ""
}

func summarizeCVDDivergence(points []CVDPoint, windows []CVDWindowStat) CVDDivergence {
	out := CVDDivergence{}
	for _, p := range points {
		if p.Divergence == "" {
			continue
		}
		out.Bars++
		out.LastAt = p.Time
	}
	kind, start, end := currentCVDDivergenceRun(points)
	if start >= 0 && end >= start && end < len(points) {
		out.Kind = kind
		out.Since = points[start].Time
		out.LastAt = points[end].Time
		out.DurationSeconds = int(points[end].Time.Sub(points[start].Time)/time.Second) + int(CVDBar/time.Second)
		out.Duration = formatCVDDuration(time.Duration(out.DurationSeconds) * time.Second)
		p0 := points[start].Price
		if start > 0 && points[start-1].Price > 0 {
			p0 = points[start-1].Price
		}
		p1 := points[end].Price
		if p0 > 0 && p1 > 0 {
			out.PriceMovePct = (p1 - p0) / p0 * 100
		}
		cvd0 := 0.0
		if start > 0 {
			cvd0 = points[start-1].CVD
		}
		out.CVDMove = points[end].CVD - cvd0
		var vol float64
		for i := start; i <= end; i++ {
			vol += points[i].BuyNotional + points[i].SellNotional
		}
		if vol > 0 {
			out.CVDMovePct = out.CVDMove / vol * 100
		}
	}
	var w4 CVDWindowStat
	for _, w := range windows {
		if w.Window == CVDWindow4h {
			w4 = w
			break
		}
	}
	if w4.Window == "" && len(windows) > 0 {
		w4 = windows[0]
	}
	out.VsPrice = w4.VsPrice
	if out.Kind == "" && w4.Divergence != "" {
		out.Kind = w4.Divergence
	}
	out.Title = cvdDivergenceTitle(out.Kind)
	if out.Title == "" {
		out.Title = cvdTitle(out.VsPrice)
	}
	out.Summary = explainCVDDivergence(out, w4)
	return out
}

func cvdDivergenceTitle(kind string) string {
	switch kind {
	case CVDDivPriceUpCVDDown:
		return "price up, CVD down"
	case CVDDivPriceDownCVDUp:
		return "price down, CVD up"
	default:
		return ""
	}
}

func explainCVDDivergence(d CVDDivergence, w CVDWindowStat) string {
	how := ""
	switch d.Kind {
	case CVDDivPriceUpCVDDown:
		how = "Price rose while CVD fell"
	case CVDDivPriceDownCVDUp:
		how = "Price fell while CVD rose"
	default:
		if d.Bars > 0 {
			return fmt.Sprintf("%d bars show price and CVD moving opposite ways.", d.Bars)
		}
		return ""
	}
	run := how
	if d.Duration != "" {
		run += " for " + d.Duration
	}
	run += fmt.Sprintf(" (price %s%%, CVD %s)", FormatSignedPct(d.PriceMovePct), FormatSignedQty(d.CVDMove))
	if d.Bars > 0 {
		run += fmt.Sprintf("; %d opposite bars in view", d.Bars)
	}
	if w.Window != "" && w.Divergence == d.Kind {
		run += " — also over " + w.Window
	}
	return run + "."
}

func explainCVDContributions(shares []CVDShare) string {
	if len(shares) == 0 {
		return ""
	}
	parts := make([]string, 0, len(shares))
	for _, s := range shares {
		parts = append(parts, fmt.Sprintf("%s %s (%s%%)",
			s.Exchange, FormatSignedQty(s.CVD), formatFixed(s.SharePct, 0)))
	}
	return "Added: " + strings.Join(parts, ", ") + "."
}

func windowShares(points []CVDPoint, from time.Time, venues []CVDVenueSeries) []CVDShare {
	first := map[Exchange]float64{}
	last := map[Exchange]float64{}
	var combFirst, combLast float64
	firstSet := false
	for _, p := range points {
		if p.Time.Before(from) {
			continue
		}
		if !firstSet {
			combFirst = p.CVD
			for _, s := range p.Shares {
				first[s.Exchange] = s.CVD
			}
			firstSet = true
		}
		combLast = p.CVD
		for _, s := range p.Shares {
			last[s.Exchange] = s.CVD
		}
	}
	change := combLast - combFirst
	out := make([]CVDShare, 0, len(venues))
	for _, v := range venues {
		d := last[v.Exchange] - first[v.Exchange]
		sh := CVDShare{Exchange: v.Exchange, Delta: d, CVD: last[v.Exchange]}
		if change != 0 {
			sh.SharePct = d / change * 100
		}
		out = append(out, sh)
	}
	return out
}

func emptyCVDWindows(now, started time.Time) []CVDWindowStat {
	out := make([]CVDWindowStat, 0, len(CVDWindows))
	for _, w := range CVDWindows {
		out = append(out, summarizeCVDWindow(nil, w.ID, now.Add(-w.Dur), now, started))
	}
	return out
}

func overlapCVDRange(venues []CVDVenueSeries) (from, to time.Time, ok bool) {
	for i, v := range venues {
		if len(v.Points) == 0 {
			return time.Time{}, time.Time{}, false
		}
		f := TruncateToBucket(v.Points[0].Time, CVDBar)
		t := TruncateToBucket(v.Points[len(v.Points)-1].Time, CVDBar)
		if i == 0 {
			from, to = f, t
			continue
		}
		if f.After(from) {
			from = f
		}
		if t.Before(to) {
			to = t
		}
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

func cvdCoverageComplete(started, from time.Time) bool {
	if started.IsZero() || from.IsZero() {
		return false
	}
	// One 5m bucket of slack so a bar that opens just after the cut still counts.
	return !started.After(from.Add(CVDBar))
}

func currentCVDDivergenceRun(points []CVDPoint) (kind string, start, end int) {
	end = -1
	for i := len(points) - 1; i >= 0; i-- {
		if points[i].Divergence != "" {
			end = i
			kind = points[i].Divergence
			break
		}
	}
	if end < 0 {
		return "", -1, -1
	}
	start = end
	// A quiet bar ends the run. The same kind later is a new episode.
	for i := end - 1; i >= 0; i-- {
		if points[i].Divergence != kind {
			break
		}
		start = i
	}
	return kind, start, end
}

func formatCVDDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return "1m"
	}
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

func cvdDir(delta float64) string {
	switch signFloat(delta) {
	case 1:
		return CVDDirUp
	case -1:
		return CVDDirDown
	default:
		return CVDDirFlat
	}
}

func venueSplitFromShares(shares []CVDShare) *CVDVenueSplit {
	var bin, byb *CVDShare
	for i := range shares {
		switch shares[i].Exchange {
		case ExchangeBinance:
			bin = &shares[i]
		case ExchangeBybit:
			byb = &shares[i]
		}
	}
	if bin == nil || byb == nil {
		return nil
	}
	out := CVDVenueSplit{
		Binance:       cvdDir(bin.Delta),
		Bybit:         cvdDir(byb.Delta),
		BinanceChange: bin.Delta,
		BybitChange:   byb.Delta,
	}
	switch {
	case out.Binance == CVDDirFlat || out.Bybit == CVDDirFlat:
		out.Alignment = AlignMixed
		out.Title = "venues mixed"
		out.Summary = fmt.Sprintf("Binance CVD %s (%s), Bybit %s (%s).",
			out.Binance, FormatSignedQty(out.BinanceChange), out.Bybit, FormatSignedQty(out.BybitChange))
	case out.Binance == out.Bybit:
		out.Alignment = AlignSame
		out.Title = "venues agree"
		out.Summary = fmt.Sprintf("Binance and Bybit CVD both %s.", out.Binance)
	default:
		out.Alignment = AlignOpposite
		out.Title = "venues split"
		out.Summary = fmt.Sprintf("Binance CVD %s (%s) while Bybit %s (%s).",
			out.Binance, FormatSignedQty(out.BinanceChange), out.Bybit, FormatSignedQty(out.BybitChange))
	}
	return &out
}

func pickCVDVenueSplit(windows []CVDWindowStat) *CVDVenueSplit {
	var fallback *CVDVenueSplit
	for _, w := range windows {
		if w.VenueSplit == nil {
			continue
		}
		if w.VenueSplit.Alignment == AlignOpposite {
			s := *w.VenueSplit
			s.Summary = w.Window + ": " + s.Summary
			return &s
		}
		if fallback == nil && (w.Window == CVDWindow1h || w.Window == CVDWindow15m) {
			fallback = w.VenueSplit
		}
	}
	return fallback
}

func explainCVDChanges(windows []CVDWindowStat) string {
	if len(windows) == 0 {
		return ""
	}
	parts := make([]string, 0, len(windows))
	for _, w := range windows {
		parts = append(parts, fmt.Sprintf("%s %s", w.Window, FormatSignedQty(w.CVDChange)))
	}
	return "CVD change: " + strings.Join(parts, ", ") + "."
}

func overlapIncompleteNote(from, now time.Time) string {
	if from.IsZero() {
		return "Combined CVD is not complete (need overlapping Binance and Bybit history)."
	}
	d := now.Sub(from)
	if d < 90*time.Minute {
		return fmt.Sprintf("Combined uses ~%.0f minutes where both venues have bars (Bybit still filling) — not complete.", d.Minutes())
	}
	return fmt.Sprintf("Combined uses the last %.0f hours where both venues have bars — not complete.", d.Hours())
}

func priceAt(prices []CVDPrice, t time.Time) float64 {
	if len(prices) == 0 || t.IsZero() {
		return 0
	}
	var best CVDPrice
	found := false
	end := t.Add(CVDBar)
	for _, p := range prices {
		// Bar at t owns [t, t+5m). The next candle's open must not leak in
		// or every point would share the following close and hide divergence.
		if !p.Time.Before(end) {
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

// TakerBucketsFromCandles turns spot klines with taker-buy quote volume into CVD bars.
// buy = takerBuyQuote; sell = quoteVolume − takerBuyQuote.
func TakerBucketsFromCandles(ex Exchange, symbol string, candles []Candle) []TakerBucket {
	symbol = NormalizeLiquidationSymbol(symbol)
	out := make([]TakerBucket, 0, len(candles))
	for _, c := range candles {
		if c.OpenTime.IsZero() {
			continue
		}
		quote, err1 := parseFloat(c.QuoteVolume)
		buy, err2 := parseFloat(c.TakerBuyQuote)
		if err1 != nil || err2 != nil || quote <= 0 || buy < 0 {
			continue
		}
		if buy > quote {
			buy = quote
		}
		out = append(out, TakerBucket{
			Exchange: ex, Symbol: symbol, Start: c.OpenTime.UTC(),
			BuyNotional: buy, SellNotional: quote - buy,
		})
	}
	return out
}

// SpotStoreExchange is the durable taker-bucket key for spot (avoids colliding with futures).
func SpotStoreExchange(ex Exchange) string {
	return string(ex) + "-spot"
}

// CompareSpotFutures says whether spot and futures CVD moved the same way.
func CompareSpotFutures(spot, fut *CVDVenueSeries) *CVDSpotFutures {
	if spot == nil || fut == nil || len(spot.Windows) == 0 || len(fut.Windows) == 0 {
		return nil
	}
	byWin := map[string]CVDWindowStat{}
	for _, w := range fut.Windows {
		byWin[w.Window] = w
	}
	out := &CVDSpotFutures{Windows: make([]CVDSpotFutures, 0, len(spot.Windows))}
	var primary *CVDSpotFutures
	for _, sw := range spot.Windows {
		fw, ok := byWin[sw.Window]
		if !ok {
			continue
		}
		row := spotFuturesWindow(sw, fw)
		out.Windows = append(out.Windows, row)
		if primary == nil || (primary.Alignment != AlignOpposite && row.Alignment == AlignOpposite) {
			cp := row
			primary = &cp
		}
	}
	if primary == nil {
		return nil
	}
	out.Alignment = primary.Alignment
	out.Spot = primary.Spot
	out.Futures = primary.Futures
	out.SpotChange = primary.SpotChange
	out.FuturesChange = primary.FuturesChange
	out.Window = primary.Window
	out.Title = primary.Title
	out.Summary = primary.Summary
	return out
}

func spotFuturesWindow(spot, fut CVDWindowStat) CVDSpotFutures {
	row := CVDSpotFutures{
		Spot:          cvdDir(spot.CVDChange),
		Futures:       cvdDir(fut.CVDChange),
		SpotChange:    spot.CVDChange,
		FuturesChange: fut.CVDChange,
		Window:        spot.Window,
	}
	switch {
	case row.Spot == CVDDirFlat || row.Futures == CVDDirFlat:
		row.Alignment = AlignMixed
		row.Title = "spot vs futures mixed"
		row.Summary = fmt.Sprintf("Spot CVD %s (%s) over %s, futures %s (%s).",
			row.Spot, FormatSignedQty(row.SpotChange), row.Window, row.Futures, FormatSignedQty(row.FuturesChange))
	case row.Spot == row.Futures:
		row.Alignment = AlignSame
		row.Title = "spot and futures agree"
		row.Summary = fmt.Sprintf("Spot and futures CVD both %s over %s.", row.Spot, row.Window)
	default:
		row.Alignment = AlignOpposite
		row.Title = "spot vs futures split"
		row.Summary = fmt.Sprintf("Spot CVD %s (%s) while futures %s (%s) over %s.",
			row.Spot, FormatSignedQty(row.SpotChange), row.Futures, FormatSignedQty(row.FuturesChange), row.Window)
	}
	return row
}
