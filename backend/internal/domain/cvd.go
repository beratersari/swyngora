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

	// CVDSplitAgrees means the tape is definite and there is no current run.
	// A 4-hour window that disagrees, with no Since, is this and not a run.
	CVDSplitAgrees = "agrees"

	CVDWindow15m = "15m"
	CVDWindow1h  = "1h"
	CVDWindow4h  = "4h"
	CVDWindow24h = "24h"

	// Anchors reset the sum at a clock, unlike the rolling windows above.
	CVDAnchorDay = "day"
	CVDAnchor4h  = "4h"
	// CVDAnchorClick is a candle the reader chose. Bars that start before it are left out.
	CVDAnchorClick = "click"

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

// CVDSpan is one CVD lookback. Longer ranges use coarser bars so the line
// still fits a few hundred points. The minutes underneath stay in the store.
type CVDSpan struct {
	ID             string
	Lookback       time.Duration
	Bar            time.Duration
	MaxPoints      int
	CandleInterval string
	CandleLimit    int
}

// DefaultCVDSpan is the 24-hour, 5-minute series.
func DefaultCVDSpan() CVDSpan {
	return CVDSpan{
		ID: "24h", Lookback: DefaultCVDLookback, Bar: CVDBar, MaxPoints: MaxCVDPoints,
		CandleInterval: "5m", CandleLimit: 320,
	}
}

// ParseCVDRange accepts 24h (default), 7d, 30d, and 90d.
// 7d is 30-minute bars, 30d is 2-hour bars, 90d is 4-hour bars.
func ParseCVDRange(raw string) (CVDSpan, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "24h":
		return DefaultCVDSpan(), nil
	case "7d":
		return CVDSpan{
			ID: "7d", Lookback: 7 * 24 * time.Hour, Bar: 30 * time.Minute, MaxPoints: 400,
			CandleInterval: "30m", CandleLimit: 360,
		}, nil
	case "30d":
		return CVDSpan{
			ID: "30d", Lookback: 30 * 24 * time.Hour, Bar: 2 * time.Hour, MaxPoints: 400,
			CandleInterval: "2h", CandleLimit: 380,
		}, nil
	case "90d":
		return CVDSpan{
			ID: "90d", Lookback: 90 * 24 * time.Hour, Bar: 4 * time.Hour, MaxPoints: 600,
			CandleInterval: "4h", CandleLimit: 560,
		}, nil
	default:
		return CVDSpan{}, fmt.Errorf("%w: cvd range must be 24h, 7d, 30d, or 90d", ErrInvalidArgument)
	}
}

// ParseCVDAnchor reads an optional RFC3339 candle time. Empty means no click anchor.
// The time has to fall inside the 90-day tape. A minute of clock skew still counts as now.
func ParseCVDAnchor(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: anchor must be RFC3339", ErrInvalidArgument)
	}
	at = at.UTC()
	now = now.UTC()
	if at.After(now.Add(time.Minute)) {
		return time.Time{}, fmt.Errorf("%w: anchor is in the future", ErrInvalidArgument)
	}
	if now.Sub(at) > 90*24*time.Hour {
		return time.Time{}, fmt.Errorf("%w: anchor is older than 90 days", ErrInvalidArgument)
	}
	return at, nil
}

// BarLabel is the candle-interval name for this span.
func (s CVDSpan) BarLabel() string {
	switch s.Bar {
	case 5 * time.Minute:
		return "5m"
	case 30 * time.Minute:
		return "30m"
	case 2 * time.Hour:
		return "2h"
	case 4 * time.Hour:
		return "4h"
	default:
		if s.Bar <= 0 {
			return "5m"
		}
		return s.Bar.String()
	}
}

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
	// SizeKnown is true when every minute in this bar was split into whale and retail.
	// WhaleDelta and RetailDelta are buy minus sell inside that class.
	// WhaleCVD and RetailCVD run only across sized bars. An unknown bar leaves them zero.
	SizeKnown   bool
	WhaleDelta  float64
	RetailDelta float64
	WhaleCVD    float64
	RetailCVD   float64
	VsPrice     string // confirms | opposite | absorption | mixed
	Divergence  string // price_up_cvd_down | price_down_cvd_up | ""
	Shares      []CVDShare
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
	Kind            string // price_up_cvd_down | price_down_cvd_up | ""
	VsPrice         string
	Title           string
	Summary         string
	Bars            int // opposite bars in the whole series
	LastAt          time.Time
	Since           time.Time // start of the current (or latest) run
	Duration        string    // e.g. "35m", "2h"
	DurationSeconds int
	PriceMovePct    float64 // price change over that run
	CVDMove         float64 // CVD change over that run
	CVDMovePct      float64 // CVD move vs buy+sell in the run
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

// CVDAnchor is CVD rebased to zero at a fixed clock (day open, 4h open, or a chosen candle).
// A rolling 4h window at 10:30 starts at 06:30; the 4h anchor starts at 08:00.
type CVDAnchor struct {
	Anchor         string // day | 4h | click
	From           time.Time
	CVD            float64 // sum of bar deltas with start >= From
	CVDChangePct   float64
	PriceChangePct float64
	BuyNotional    float64
	SellNotional   float64
	VsPrice        string
	Divergence     string
	Title          string
	Summary        string
	Complete       bool
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
	Exchange      Exchange
	Symbol        string
	Market        string // spot | futures
	Range         string
	Bar           time.Duration
	MaxPoints     int
	Points        []CVDPoint
	Windows       []CVDWindowStat
	Anchors       []CVDAnchor
	LastCVD       float64
	LastPrice     float64
	Contributions []CVDShare
	OverlapFrom   *time.Time
	OverlapTo     *time.Time
	Divergence    CVDDivergence
	VenueSplit    *CVDVenueSplit
	Summary       string
	Error         string
	Complete      bool
}

// CVDReport is the API result.
type CVDReport struct {
	Symbol       string
	Exchange     string
	Range        string
	Bar          string
	AsOf         time.Time
	Venues       []CVDVenueSeries // futures
	Combined     *CVDVenueSeries  // futures combined
	SpotVenues   []CVDVenueSeries
	SpotCombined *CVDVenueSeries
	SpotFutures  *CVDSpotFutures
	Summary      string
	Note         string
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

// sizedFold is one coarser bar. cleared means a constituent minute had notional
// and no print-size split, so the coarse bar must not pretend to be sized.
type sizedFold struct {
	bar     TakerBucket
	cleared bool
}

// ResampleTakerBuckets folds bars into a coarser step (e.g. 1m → 5m).
// The coarse bar is sized only when every minute that has notional is sized.
// One unknown minute clears whale and retail on that coarse bar.
func ResampleTakerBuckets(in []TakerBucket, step time.Duration) []TakerBucket {
	if step <= 0 {
		step = CVDBar
	}
	acc := map[int64]*sizedFold{}
	for _, b := range in {
		if b.Start.IsZero() || (b.BuyNotional <= 0 && b.SellNotional <= 0) {
			continue
		}
		ms := TruncateToBucket(b.Start.UTC(), step).UnixMilli()
		cur := acc[ms]
		if cur == nil {
			cur = &sizedFold{bar: TakerBucket{
				Exchange: b.Exchange, Symbol: NormalizeLiquidationSymbol(b.Symbol),
				Start: time.UnixMilli(ms).UTC(),
			}}
			acc[ms] = cur
		}
		cur.bar.BuyNotional += b.BuyNotional
		cur.bar.SellNotional += b.SellNotional
		if cur.bar.Exchange == "" {
			cur.bar.Exchange = b.Exchange
		}
		if cur.cleared || !b.SizeKnown {
			cur.cleared = true
			cur.bar.SizeKnown = false
			cur.bar.WhaleBuy = 0
			cur.bar.WhaleSell = 0
			cur.bar.RetailBuy = 0
			cur.bar.RetailSell = 0
			continue
		}
		cur.bar.SizeKnown = true
		cur.bar.WhaleBuy += b.WhaleBuy
		cur.bar.WhaleSell += b.WhaleSell
		cur.bar.RetailBuy += b.RetailBuy
		cur.bar.RetailSell += b.RetailSell
	}
	out := make([]TakerBucket, 0, len(acc))
	for _, b := range acc {
		out = append(out, b.bar)
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

// BuildCVDSeries accumulates buy−sell and joins price over the default 24h span.
func BuildCVDSeries(ex Exchange, symbol string, buckets []TakerBucket, prices []CVDPrice, now time.Time, started time.Time) CVDVenueSeries {
	return BuildCVDSeriesSpan(ex, symbol, buckets, prices, now, started, DefaultCVDSpan())
}

// BuildCVDSeriesSpan accumulates buy−sell over one lookback.
// Bars coarser than the stored minutes are summed. Points before the lookback are dropped.
func BuildCVDSeriesSpan(ex Exchange, symbol string, buckets []TakerBucket, prices []CVDPrice, now time.Time, started time.Time, span CVDSpan) CVDVenueSeries {
	if span.Bar <= 0 || span.Lookback <= 0 {
		span = DefaultCVDSpan()
	}
	if span.MaxPoints <= 0 {
		span.MaxPoints = MaxCVDPoints
	}
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	cut := now.Add(-span.Lookback)
	bars := ResampleTakerBuckets(buckets, span.Bar)
	out := CVDVenueSeries{
		Exchange: ex, Symbol: symbol, Market: CVDMarketFutures,
		Range: span.ID, Bar: span.Bar, MaxPoints: span.MaxPoints,
		Points: []CVDPoint{}, Windows: []CVDWindowStat{},
	}
	var cvd, whaleCVD, retailCVD float64
	for _, b := range bars {
		if b.Start.Before(cut) {
			continue
		}
		delta := b.BuyNotional - b.SellNotional
		cvd += delta
		pt := CVDPoint{
			Time: b.Start, BuyNotional: b.BuyNotional, SellNotional: b.SellNotional,
			Delta: delta, CVD: cvd, Price: priceAtBar(prices, b.Start, span.Bar),
		}
		if b.SizeKnown {
			pt.SizeKnown = true
			pt.WhaleDelta = b.WhaleBuy - b.WhaleSell
			pt.RetailDelta = b.RetailBuy - b.RetailSell
			whaleCVD += pt.WhaleDelta
			retailCVD += pt.RetailDelta
			pt.WhaleCVD = whaleCVD
			pt.RetailCVD = retailCVD
		}
		out.Points = append(out.Points, pt)
	}
	if len(out.Points) > span.MaxPoints {
		out.Points = out.Points[len(out.Points)-span.MaxPoints:]
	}
	annotateCVDPoints(out.Points)
	if n := len(out.Points); n > 0 {
		out.LastCVD = out.Points[n-1].CVD
		out.LastPrice = out.Points[n-1].Price
	}
	for _, w := range CVDWindows {
		out.Windows = append(out.Windows, summarizeCVDWindow(out.Points, w.ID, now.Add(-w.Dur), now, started))
	}
	out.Anchors = attachCVDAnchors(out.Points, now, started)
	out.Complete = cvdCoverageCompleteBar(started, cut, span.Bar)
	out.Divergence = summarizeCVDDivergence(out.Points, out.Windows)
	out.Summary = ExplainCVDVenue(out)
	return out
}

// CombineCVDVenues sums per-bar delta on the overlap the venues share (never averages).
// Combined is complete only when every venue passed in has points and is itself
// complete. A missing venue is not treated as zero. A short history on one venue
// must not make the others' longer tape look finished.
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
		out.Anchors = attachCVDAnchors(nil, now, time.Time{})
		out.Summary = "Combined CVD only uses the time range both Binance and Bybit have. One venue is still filling in — not shown as complete."
		return out
	}
	sort.Slice(usable, func(i, j int) bool {
		return string(usable[i].Exchange) < string(usable[j].Exchange)
	})
	if usable[0].Market != "" {
		out.Market = usable[0].Market
	}

	bar := cvdSeriesBar(usable[0])
	capN := cvdSeriesCap(usable[0])
	out.Range = usable[0].Range
	out.Bar = bar
	out.MaxPoints = capN
	from, to, ok := overlapCVDRange(usable)
	if !ok {
		out.Windows = emptyCVDWindows(now, time.Time{})
		out.Anchors = attachCVDAnchors(nil, now, time.Time{})
		out.Summary = "Combined CVD needs overlapping bars on both venues. Bybit history is still filling in — not shown as complete."
		return out
	}
	out.OverlapFrom = &from
	out.OverlapTo = &to

	indexes := make([]map[int64]CVDPoint, len(usable))
	for i, v := range usable {
		m := make(map[int64]CVDPoint, len(v.Points))
		for _, p := range v.Points {
			m[TruncateToBucket(p.Time, bar).UnixMilli()] = p
		}
		indexes[i] = m
	}

	venueCvd := make([]float64, len(usable))
	var cvd, whaleCVD, retailCVD float64
	for at := from; !at.After(to); at = at.Add(bar) {
		ms := at.UnixMilli()
		pt := CVDPoint{Time: at}
		shares := make([]CVDShare, 0, len(usable))
		sizeOK := true
		var whaleDelta, retailDelta float64
		for i, v := range usable {
			src, ok := indexes[i][ms] // missing slot is not a sized zero
			pt.BuyNotional += src.BuyNotional
			pt.SellNotional += src.SellNotional
			pt.Delta += src.Delta
			venueCvd[i] += src.Delta
			shares = append(shares, CVDShare{
				Exchange: v.Exchange, Delta: src.Delta, CVD: venueCvd[i],
			})
			if !ok || !src.SizeKnown {
				sizeOK = false
				continue
			}
			whaleDelta += src.WhaleDelta
			retailDelta += src.RetailDelta
		}
		cvd += pt.Delta
		pt.CVD = cvd
		// Combined whale and retail exist only when every venue's bar at this
		// step was sized. An empty slot or an unsized venue is not zero.
		if sizeOK {
			pt.SizeKnown = true
			pt.WhaleDelta = whaleDelta
			pt.RetailDelta = retailDelta
			whaleCVD += whaleDelta
			retailCVD += retailDelta
			pt.WhaleCVD = whaleCVD
			pt.RetailCVD = retailCVD
		}
		pt.Price = priceAtBar(prices, pt.Time, bar)
		for i := range shares {
			if cvd != 0 {
				shares[i].SharePct = shares[i].CVD / cvd * 100
			}
		}
		pt.Shares = shares
		out.Points = append(out.Points, pt)
	}
	if len(out.Points) > capN {
		out.Points = out.Points[len(out.Points)-capN:]
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
	out.Anchors = attachCVDAnchors(out.Points, now, started)
	usableIncomplete := false
	for _, v := range usable {
		if !v.Complete {
			usableIncomplete = true
			break
		}
	}
	// A venue that was requested and has no bars is not zero. Combined stays
	// incomplete until every venue passed in reaches the window open.
	// Venues that do cover the lookback stay complete when the first shared
	// bar is one bucket later than the cut.
	missing := cvdVenuesWithoutPoints(venues)
	out.Complete = !usableIncomplete && len(missing) == 0
	out.Divergence = summarizeCVDDivergence(out.Points, out.Windows)
	out.VenueSplit = pickCVDVenueSplit(out.Windows)
	out.Summary = ExplainCVDVenue(*out)
	if !out.Complete {
		notes := make([]string, 0, 2)
		if len(missing) > 0 {
			notes = append(notes, fmt.Sprintf(
				"Combined CVD stays incomplete until %s reaches the window open.",
				strings.Join(missing, ", "),
			))
		}
		if usableIncomplete {
			notes = append(notes, overlapIncompleteNote(from, now))
		}
		out.Summary = strings.TrimSpace(strings.Join(notes, " ") + " " + out.Summary)
	}
	return out
}

// cvdVenuesWithoutPoints names venues that were asked for and did not return bars.
func cvdVenuesWithoutPoints(venues []CVDVenueSeries) []string {
	missing := make([]string, 0)
	for _, v := range venues {
		if v.Error == "" && len(v.Points) > 0 {
			continue
		}
		name := string(v.Exchange)
		if name == "" {
			name = "venue"
		}
		missing = append(missing, name)
	}
	return missing
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
	if bits := explainCVDAnchors(v.Anchors); bits != "" {
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

func cvdSeriesBar(v CVDVenueSeries) time.Duration {
	if v.Bar > 0 {
		return v.Bar
	}
	return CVDBar
}

func cvdSeriesCap(v CVDVenueSeries) int {
	if v.MaxPoints > 0 {
		return v.MaxPoints
	}
	return MaxCVDPoints
}

func overlapCVDRange(venues []CVDVenueSeries) (from, to time.Time, ok bool) {
	for i, v := range venues {
		if len(v.Points) == 0 {
			return time.Time{}, time.Time{}, false
		}
		step := cvdSeriesBar(v)
		f := TruncateToBucket(v.Points[0].Time, step)
		t := TruncateToBucket(v.Points[len(v.Points)-1].Time, step)
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
	return cvdCoverageCompleteBar(started, from, CVDBar)
}

func cvdCoverageCompleteBar(started, from time.Time, bar time.Duration) bool {
	if started.IsZero() || from.IsZero() {
		return false
	}
	if bar <= 0 {
		bar = CVDBar
	}
	// One bar of slack so a bucket that opens just after the cut still counts.
	return !started.After(from.Add(bar))
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

// CVDSplit is the current price-versus-CVD run on one series.
// Kind is agrees when the series is definite and no run is open.
// Seconds is signed: price up / CVD down is positive, the reverse is negative.
type CVDSplit struct {
	OK       bool
	Kind     string
	Duration string
	Seconds  int
	Since    time.Time
	At       time.Time
}

// ReadCVDSplit reads the current run. A window fallback (kind set, Since zero)
// is agrees. OK is false when the newest bar has no price.
func ReadCVDSplit(series CVDVenueSeries) CVDSplit {
	if len(series.Points) == 0 || series.LastPrice <= 0 {
		return CVDSplit{}
	}
	last := series.Points[len(series.Points)-1]
	if last.Price <= 0 {
		return CVDSplit{}
	}
	out := CVDSplit{OK: true, Kind: CVDSplitAgrees, At: last.Time}
	runKind, start, end := currentCVDDivergenceRun(series.Points)
	// The latest episode in the series is not the current run once a later
	// bar moves price and CVD the same way. Flat bars after the episode do
	// not close it. A 4-hour window fallback has no since.
	if runKind == "" || start < 0 || end < start || series.Divergence.Since.IsZero() || series.Divergence.Kind != runKind || !cvdSplitRunOpen(series.Points, end) {
		return out
	}
	out.Kind = runKind
	out.Duration = series.Divergence.Duration
	out.Seconds = signedCVDSplitSeconds(runKind, series.Divergence.DurationSeconds)
	out.Since = series.Divergence.Since
	if !series.Divergence.LastAt.IsZero() {
		out.At = series.Divergence.LastAt
	}
	return out
}

// VenueCVDSplit builds the 24h, 5-minute series and keeps the split only when
// the tape reaches 00:00 UTC and the newest bar has a price.
// At is the newest minute bucket, not the 5-minute bar open.
func VenueCVDSplit(ex Exchange, symbol string, buckets []TakerBucket, prices []CVDPrice, now time.Time) (CVDSplit, CVDVenueSeries) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	started := now
	for _, b := range buckets {
		if !b.Start.IsZero() && b.Start.Before(started) {
			started = b.Start
		}
	}
	series := BuildCVDSeriesSpan(ex, symbol, buckets, prices, now, started, DefaultCVDSpan())
	_, newest, dayOK := DayCVDDelta(buckets, now, started)
	split := ReadCVDSplit(series)
	split.At = newest
	if !dayOK || !split.OK {
		return CVDSplit{At: newest}, series
	}
	return split, series
}

// cvdSplitRunOpen reports whether every bar after the episode is quiet.
// A quiet bar is the same bar barDivergence leaves unmarked: a flat price
// or a zero delta. An agreeing bar closes the run.
func cvdSplitRunOpen(points []CVDPoint, end int) bool {
	for i := end + 1; i < len(points); i++ {
		p := points[i]
		if math.Abs(p.PriceChangePct) >= cvdBarPriceFlatPct && p.Delta != 0 {
			return false
		}
	}
	return true
}

func signedCVDSplitSeconds(kind string, seconds int) int {
	if seconds < 0 {
		seconds = -seconds
	}
	switch kind {
	case CVDDivPriceUpCVDDown:
		return seconds
	case CVDDivPriceDownCVDUp:
		return -seconds
	default:
		return 0
	}
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

// cvdAnchorStarts are 00:00 UTC and the current 4-hour block (00/04/08/12/16/20).
func cvdAnchorStarts(now time.Time) []struct {
	ID   string
	From time.Time
} {
	now = now.UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	block := now.Truncate(4 * time.Hour)
	return []struct {
		ID   string
		From time.Time
	}{
		{CVDAnchorDay, day},
		{CVDAnchor4h, block},
	}
}

func attachCVDAnchors(points []CVDPoint, now, started time.Time) []CVDAnchor {
	starts := cvdAnchorStarts(now)
	out := make([]CVDAnchor, 0, len(starts))
	for _, a := range starts {
		out = append(out, summarizeCVDAnchor(points, a.ID, a.From, started))
	}
	return out
}

// AttachCVDClickAnchor sums each series again from a chosen candle.
// Day and 4h anchors stay. A bar that opened before the candle is not included.
func AttachCVDClickAnchor(rep *CVDReport, at time.Time) {
	if rep == nil || at.IsZero() {
		return
	}
	for i := range rep.Venues {
		stampClickAnchor(&rep.Venues[i], at)
	}
	if rep.Combined != nil {
		stampClickAnchor(rep.Combined, at)
	}
	for i := range rep.SpotVenues {
		stampClickAnchor(&rep.SpotVenues[i], at)
	}
	if rep.SpotCombined != nil {
		stampClickAnchor(rep.SpotCombined, at)
	}
	rep.Summary = ExplainCVDReport(*rep)
}

func stampClickAnchor(v *CVDVenueSeries, at time.Time) {
	if v == nil {
		return
	}
	started := time.Time{}
	if len(v.Points) > 0 {
		started = v.Points[0].Time
	}
	v.Anchors = append(v.Anchors, summarizeCVDAnchor(v.Points, CVDAnchorClick, at, started))
	if v.Error == "" {
		v.Summary = ExplainCVDVenue(*v)
	}
}

func summarizeCVDAnchor(points []CVDPoint, anchor string, from, started time.Time) CVDAnchor {
	out := CVDAnchor{Anchor: anchor, From: from.UTC()}
	var first, last *CVDPoint
	for i := range points {
		p := &points[i]
		if p.Time.Before(from) {
			continue
		}
		if first == nil {
			first = p
		}
		last = p
		out.BuyNotional += p.BuyNotional
		out.SellNotional += p.SellNotional
		out.CVD += p.Delta
	}
	out.Complete = cvdCoverageComplete(started, from) && first != nil
	label := cvdAnchorLabel(anchor)
	if first == nil || last == nil {
		out.VsPrice = CVDVsMixed
		out.Title = "no data"
		out.Summary = "No bars since the " + label + "."
		return out
	}
	if tot := out.BuyNotional + out.SellNotional; tot > 0 {
		out.CVDChangePct = out.CVD / tot * 100
	}
	if first.Price > 0 && last.Price > 0 {
		out.PriceChangePct = (last.Price - first.Price) / first.Price * 100
	}
	out.VsPrice = CVDVsPrice(out.PriceChangePct, out.CVD, out.BuyNotional+out.SellNotional)
	out.Divergence = windowDivergence(out.PriceChangePct, out.CVD)
	out.Title = cvdTitle(out.VsPrice)
	if out.Divergence != "" {
		out.Title = cvdDivergenceTitle(out.Divergence)
	}
	out.Summary = explainCVDAnchor(out)
	if !out.Complete {
		out.Summary += " History starts later, so this is only the bars on hand."
	}
	return out
}

func cvdAnchorLabel(anchor string) string {
	switch anchor {
	case CVDAnchorDay:
		return "day open"
	case CVDAnchorClick:
		return "chosen candle"
	default:
		return "4h open"
	}
}

func explainCVDAnchor(a CVDAnchor) string {
	cvdWord := "CVD was flat"
	if a.CVD > 0 {
		cvdWord = "CVD rose (more market buys)"
	} else if a.CVD < 0 {
		cvdWord = "CVD fell (more market sells)"
	}
	pxWord := "price was little changed"
	if a.PriceChangePct >= cvdPriceFlatPct {
		pxWord = "price rose"
	} else if a.PriceChangePct <= -cvdPriceFlatPct {
		pxWord = "price fell"
	}
	clock := a.From.UTC().Format("15:04")
	if a.Anchor == CVDAnchorClick {
		clock = a.From.UTC().Format("2006-01-02 15:04")
	}
	nums := fmt.Sprintf("Since the %s (%s UTC), CVD %s, price %s%%",
		cvdAnchorLabel(a.Anchor), clock, FormatSignedQty(a.CVD), FormatSignedPct(a.PriceChangePct))
	switch a.VsPrice {
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

func explainCVDAnchors(anchors []CVDAnchor) string {
	parts := make([]string, 0, len(anchors))
	for _, a := range anchors {
		if a.From.IsZero() {
			continue
		}
		parts = append(parts, fmt.Sprintf("since %s UTC %s", a.From.UTC().Format("15:04"), FormatSignedQty(a.CVD)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Anchored CVD: " + strings.Join(parts, ", ") + "."
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
	return priceAtBar(prices, t, CVDBar)
}

func priceAtBar(prices []CVDPrice, t time.Time, bar time.Duration) float64 {
	if len(prices) == 0 || t.IsZero() {
		return 0
	}
	if bar <= 0 {
		bar = CVDBar
	}
	var best CVDPrice
	found := false
	end := t.Add(bar)
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
