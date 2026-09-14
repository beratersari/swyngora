package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	// LiquidationLevelsReviewMaxSamples is the newest estimates kept on the wire.
	LiquidationLevelsReviewMaxSamples = 12
	// LiquidationLevelsReviewMinPredicted skips dust bands.
	LiquidationLevelsReviewMinPredicted = 5000
	// LiquidationLevelsReviewMinValidated is how many scored bands a
	// reliability label needs before it is not "unknown".
	LiquidationLevelsReviewMinValidated = 3

	LiquidationLevelsReliabilityHigh    = "high"
	LiquidationLevelsReliabilityFair    = "fair"
	LiquidationLevelsReliabilityLow     = "low"
	LiquidationLevelsReliabilityUnknown = "unknown"
)

// LiquidationLevelsReviewHorizons is the look-ahead set (same clocks as the heatmap).
var LiquidationLevelsReviewHorizons = HuntHeatmapReviewHorizons

// LiquidationLevelsReviewBand is one predicted 10x/25x/50x/100x pocket vs later prints.
type LiquidationLevelsReviewBand struct {
	Leverage     int
	Side         string
	Price        float64
	Predicted    float64
	Realized     float64
	Hit          bool
	FillRate     float64
	Status       string
	Validated    bool
	Pending      bool
	Gap          string
	TimeToHitSec float64
}

// LiquidationLevelsReviewSample is one reconstructed estimate and how it aged.
type LiquidationLevelsReviewSample struct {
	Time      time.Time
	LastPrice float64
	Bands     []LiquidationLevelsReviewBand
}

// LiquidationLevelsReviewHorizon is hit / fill stats for one look-ahead.
type LiquidationLevelsReviewHorizon struct {
	Horizon   string
	Signals   int
	Validated int
	Missing   int
	Pending   int
	Hits      int
	HitRate   float64
	Predicted float64
	Realized  float64
	FillRate  float64
	Coverage  float64
}

// LiquidationLevelsReviewBucket is reliability for one leverage × side.
type LiquidationLevelsReviewBucket struct {
	Leverage    int
	Side        string
	Horizon     string
	Signals     int
	Validated   int
	Hits        int
	HitRate     float64
	Predicted   float64
	Realized    float64
	FillRate    float64
	Reliability string
}

// LiquidationLevelsReviewVenue is one venue's (or combined) calibration.
type LiquidationLevelsReviewVenue struct {
	Exchange   string
	Horizons   []LiquidationLevelsReviewHorizon
	ByLeverage []LiquidationLevelsReviewBucket
	Samples    []LiquidationLevelsReviewSample
}

// LiquidationLevelsReview compares past estimated levels to later liquidations.
type LiquidationLevelsReview struct {
	Binance  LiquidationLevelsReviewVenue
	Bybit    LiquidationLevelsReviewVenue
	OKX      LiquidationLevelsReviewVenue
	Combined LiquidationLevelsReviewVenue
	Summary  string
	Note     string
}

const liquidationLevelsReviewNote = "Each past estimate is rebuilt from that venue's own last price, open interest, and the same 10x/25x/50x/100x mix as the live map. Later liquidations count only when they print on that venue, on the same side, and inside the band around the predicted price. A quiet window is a miss only when liquidation history covers the look-ahead; pending / price_gap / liq_gap are not misses. Combined sums independent venues and never copies the other book. Informational only."

// LiquidationLevelsReviewInput is history used to rebuild and score past maps.
type LiquidationLevelsReviewInput struct {
	Now    time.Time
	Step   time.Duration
	Venues []HuntHeatmapVenueSeries
}

// ReviewLiquidationLevels scores reconstructed estimates against later prints.
func ReviewLiquidationLevels(in LiquidationLevelsReviewInput) LiquidationLevelsReview {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	step := in.Step
	if step <= 0 {
		step = 30 * time.Minute
	}
	stale := step * 2
	if stale < 2*time.Hour {
		stale = 2 * time.Hour
	}
	byEx := map[Exchange]HuntHeatmapVenueSeries{}
	for _, v := range in.Venues {
		byEx[v.Exchange] = v
	}
	out := LiquidationLevelsReview{
		Binance: reviewLevelsVenue(string(ExchangeBinance), byEx[ExchangeBinance], now, step, stale),
		Bybit:   reviewLevelsVenue(string(ExchangeBybit), byEx[ExchangeBybit], now, step, stale),
		OKX:     reviewLevelsVenue(string(ExchangeOKX), byEx[ExchangeOKX], now, step, stale),
		Note:    liquidationLevelsReviewNote,
	}
	out.Combined = mergeLevelsReviewVenues("combined", []LiquidationLevelsReviewVenue{out.Binance, out.Bybit, out.OKX})
	out.Summary = ExplainLiquidationLevelsReview(out)
	return out
}

func reviewLevelsVenue(ex string, ser HuntHeatmapVenueSeries, now time.Time, step, stale time.Duration) LiquidationLevelsReviewVenue {
	out := emptyLevelsReviewVenue(ex)
	built := rebuildLevelSamples(ser, now, step, stale)
	if len(built.samples) == 0 {
		return out
	}
	out.Horizons = built.horizons
	out.ByLeverage = built.buckets
	samples := built.samples
	if len(samples) > LiquidationLevelsReviewMaxSamples {
		samples = samples[len(samples)-LiquidationLevelsReviewMaxSamples:]
	}
	out.Samples = samples
	return out
}

type rebuiltLevelReview struct {
	samples  []LiquidationLevelsReviewSample
	horizons []LiquidationLevelsReviewHorizon
	buckets  []LiquidationLevelsReviewBucket
}

func emptyLevelsReviewVenue(ex string) LiquidationLevelsReviewVenue {
	hs := make([]LiquidationLevelsReviewHorizon, len(LiquidationLevelsReviewHorizons))
	for i, h := range LiquidationLevelsReviewHorizons {
		hs[i] = LiquidationLevelsReviewHorizon{Horizon: h.ID}
	}
	return LiquidationLevelsReviewVenue{Exchange: ex, Horizons: hs, ByLeverage: []LiquidationLevelsReviewBucket{}, Samples: []LiquidationLevelsReviewSample{}}
}

func rebuildLevelSamples(ser HuntHeatmapVenueSeries, now time.Time, step, stale time.Duration) rebuiltLevelReview {
	ois := downsampleSnapshots(ser.OI, step)
	if len(ois) == 0 {
		return rebuiltLevelReview{}
	}
	liqs := [][]LiquidationEvent{ser.Liquidations}
	accH := map[string]*levelReviewAcc{}
	for _, spec := range LiquidationLevelsReviewHorizons {
		accH[spec.ID] = &levelReviewAcc{horizon: spec.ID}
	}
	type bkey struct {
		lev  int
		side string
	}
	accB := map[bkey]*levelReviewAcc{}
	var samples []LiquidationLevelsReviewSample
	for _, oi := range ois {
		if oi.Value <= 0 || oi.SampledAt.IsZero() || oi.SampledAt.After(now) {
			continue
		}
		last, ok := levelPriceAtOrBefore(ser.Prices, oi.SampledAt, stale)
		if !ok || last <= 0 {
			continue
		}
		ls := snapshotAtOrBefore(ser.LongShort, oi.SampledAt, 6*time.Hour)
		longShare, shortShare := 0.5, 0.5
		if ls != nil && (ls.LongShare > 0 || ls.ShortShare > 0) {
			longShare, shortShare = BlendAccountShare(ls.LongShare)
		}
		var funding float64
		if f := snapshotAtOrBefore(ser.Funding, oi.SampledAt, 8*time.Hour); f != nil {
			funding = f.FundingRate
		}
		pred := PredictDisplayLevelBands(last, oi.Value, longShare, shortShare, funding)
		if len(pred) == 0 {
			continue
		}
		sample := LiquidationLevelsReviewSample{Time: oi.SampledAt.UTC(), LastPrice: last}
		for _, spec := range LiquidationLevelsReviewHorizons {
			for _, p := range pred {
				row := scoreLevelBandHorizon(p, spec.Dur, oi.SampledAt, now, liqs)
				accH[spec.ID].add(row)
				if spec.ID == "1h" {
					sample.Bands = append(sample.Bands, row)
					k := bkey{lev: p.Leverage, side: p.Side}
					if accB[k] == nil {
						accB[k] = &levelReviewAcc{horizon: "1h"}
					}
					accB[k].add(row)
				}
			}
		}
		samples = append(samples, sample)
	}
	hs := make([]LiquidationLevelsReviewHorizon, 0, len(LiquidationLevelsReviewHorizons))
	for _, spec := range LiquidationLevelsReviewHorizons {
		hs = append(hs, accH[spec.ID].finish())
	}
	bs := make([]LiquidationLevelsReviewBucket, 0, len(accB))
	for _, lev := range ChartLeverageBuckets {
		for _, side := range []string{LiquidationSideShort, LiquidationSideLong} {
			a := accB[bkey{lev: lev, side: side}]
			if a == nil || a.signals == 0 {
				continue
			}
			h := a.finish()
			bs = append(bs, LiquidationLevelsReviewBucket{
				Leverage: lev, Side: side, Horizon: "1h",
				Signals: h.Signals, Validated: h.Validated, Hits: h.Hits,
				HitRate: h.HitRate, Predicted: h.Predicted, Realized: h.Realized,
				FillRate: h.FillRate, Reliability: levelReliability(h.Validated, h.HitRate, h.FillRate),
			})
		}
	}
	return rebuiltLevelReview{samples: samples, horizons: hs, buckets: bs}
}

// PredictDisplayLevelBands is the live map mix at one last / OI print.
func PredictDisplayLevelBands(last, oi, longShare, shortShare, funding float64) []predictedLevelBand {
	if last <= 0 || oi <= 0 {
		return nil
	}
	if longShare <= 0 && shortShare <= 0 {
		longShare, shortShare = 0.5, 0.5
	}
	estLong := oi * longShare
	estShort := oi * shortShare
	longMix := TiltLeverageMix(DefaultHuntLeverageMix, funding > 1e-12)
	shortMix := TiltLeverageMix(DefaultHuntLeverageMix, funding < -1e-12)
	longW := displayMixWeights(longMix)
	shortW := displayMixWeights(shortMix)
	out := make([]predictedLevelBand, 0, 8)
	for _, lev := range ChartLeverageBuckets {
		dist := HuntLiqDistance(float64(lev), HuntMaintenanceMargin)
		half := levelBandHalfPct(lev)
		if w := longW[lev]; w > 0 {
			px := last * (1 - dist)
			pred := estLong * w
			if pred >= LiquidationLevelsReviewMinPredicted && px > 0 {
				out = append(out, predictedLevelBand{
					Leverage: lev, Side: LiquidationSideLong, Price: px, DistPct: -dist * 100,
					Predicted: pred, Lo: last * (1 - dist - half), Hi: last * (1 - dist + half),
				})
			}
		}
		if w := shortW[lev]; w > 0 {
			px := last * (1 + dist)
			pred := estShort * w
			if pred >= LiquidationLevelsReviewMinPredicted && px > 0 {
				out = append(out, predictedLevelBand{
					Leverage: lev, Side: LiquidationSideShort, Price: px, DistPct: dist * 100,
					Predicted: pred, Lo: last * (1 + dist - half), Hi: last * (1 + dist + half),
				})
			}
		}
	}
	return out
}

type predictedLevelBand struct {
	Leverage  int
	Side      string
	Price     float64
	DistPct   float64
	Predicted float64
	Lo, Hi    float64
}

func displayMixWeights(mix []HuntLeverageBucket) map[int]float64 {
	out := map[int]float64{ChartLeverage10: 0, ChartLeverage25: 0, ChartLeverage50: 0, ChartLeverage100: 0}
	for _, b := range mix {
		if b.Weight <= 0 {
			continue
		}
		out[ChartLeverageBucket(b.Leverage)] += b.Weight
	}
	return out
}

func levelBandHalfPct(lev int) float64 {
	d100 := HuntLiqDistance(100, HuntMaintenanceMargin)
	d50 := HuntLiqDistance(50, HuntMaintenanceMargin)
	d25 := HuntLiqDistance(25, HuntMaintenanceMargin)
	d10 := HuntLiqDistance(10, HuntMaintenanceMargin)
	switch lev {
	case ChartLeverage100:
		return (d50 - d100) / 2
	case ChartLeverage50:
		return (d25 - d50) / 2
	case ChartLeverage25:
		return (d10 - d25) / 2
	default:
		return math.Max((d10-d25)/2, liquidationLevelPad)
	}
}

func scoreLevelBandHorizon(p predictedLevelBand, dur time.Duration, t, now time.Time, liqs [][]LiquidationEvent) LiquidationLevelsReviewBand {
	until := t.Add(dur)
	clockOK := !now.Before(until)
	liqOK := clockOK && huntLiqFeedCovers(liqs, t.Add(-30*time.Minute), until)
	realized, first := realizedLiqInBand(liqs, t, until, p.Lo, p.Hi, p.Side)
	row := LiquidationLevelsReviewBand{
		Leverage: p.Leverage, Side: p.Side, Price: p.Price,
		Predicted: p.Predicted, Realized: realized,
		Pending: !clockOK,
	}
	if p.Predicted > 0 {
		row.FillRate = realized / p.Predicted
	}
	switch {
	case !clockOK:
		row.Status = HuntReviewPending
		row.Gap = "pending"
	case !liqOK:
		row.Status = HuntReviewLiqGap
		if !huntLiqHasAny(liqs) {
			row.Gap = "no_liq"
		} else {
			row.Gap = "liq_short"
		}
	default:
		row.Validated = true
		if realized > 0 {
			row.Hit = true
			row.Status = HuntReviewHit
			if !first.IsZero() {
				sec := first.Sub(t).Seconds()
				if sec < 0 {
					sec = 0
				}
				row.TimeToHitSec = sec
			}
		} else {
			row.Status = HuntReviewMiss
		}
	}
	return row
}

func realizedLiqInBand(groups [][]LiquidationEvent, from, to time.Time, lo, hi float64, side string) (float64, time.Time) {
	var sum float64
	var first time.Time
	for _, ev := range groups {
		for _, e := range ev {
			if e.Notional <= 0 || e.Price <= 0 {
				continue
			}
			if side != "" && e.Side != side {
				continue
			}
			if !e.Time.After(from) || e.Time.After(to) {
				continue
			}
			if e.Price < lo || e.Price > hi {
				continue
			}
			sum += e.Notional
			if first.IsZero() || e.Time.Before(first) {
				first = e.Time
			}
		}
	}
	return sum, first
}

type levelReviewAcc struct {
	horizon   string
	signals   int
	validated int
	missing   int
	pending   int
	hits      int
	predicted float64
	realized  float64
}

func (a *levelReviewAcc) add(b LiquidationLevelsReviewBand) {
	if a == nil {
		return
	}
	a.signals++
	if b.Pending {
		a.pending++
	}
	if !b.Validated {
		a.missing++
		return
	}
	a.validated++
	a.predicted += b.Predicted
	a.realized += b.Realized
	if b.Hit {
		a.hits++
	}
}

func (a *levelReviewAcc) finish() LiquidationLevelsReviewHorizon {
	if a == nil {
		return LiquidationLevelsReviewHorizon{}
	}
	row := LiquidationLevelsReviewHorizon{
		Horizon: a.horizon, Signals: a.signals, Validated: a.validated,
		Missing: a.missing, Pending: a.pending, Hits: a.hits,
		Predicted: a.predicted, Realized: a.realized,
	}
	if a.signals > 0 {
		row.Coverage = float64(a.validated) / float64(a.signals)
	}
	if a.validated > 0 {
		row.HitRate = float64(a.hits) / float64(a.validated)
	}
	if a.predicted > 0 {
		row.FillRate = a.realized / a.predicted
	}
	return row
}

func levelReliability(validated int, hitRate, fillRate float64) string {
	if validated < LiquidationLevelsReviewMinValidated {
		return LiquidationLevelsReliabilityUnknown
	}
	if hitRate >= 0.55 && fillRate >= 0.10 {
		return LiquidationLevelsReliabilityHigh
	}
	if hitRate >= 0.30 {
		return LiquidationLevelsReliabilityFair
	}
	return LiquidationLevelsReliabilityLow
}

func mergeLevelsReviewVenues(ex string, in []LiquidationLevelsReviewVenue) LiquidationLevelsReviewVenue {
	out := emptyLevelsReviewVenue(ex)
	accH := map[string]*levelReviewAcc{}
	for _, spec := range LiquidationLevelsReviewHorizons {
		accH[spec.ID] = &levelReviewAcc{horizon: spec.ID}
	}
	type key struct {
		lev  int
		side string
	}
	accB := map[key]*levelReviewAcc{}
	var samples []LiquidationLevelsReviewSample
	for _, v := range in {
		for _, h := range v.Horizons {
			a := accH[h.Horizon]
			if a == nil {
				continue
			}
			a.signals += h.Signals
			a.validated += h.Validated
			a.missing += h.Missing
			a.pending += h.Pending
			a.hits += h.Hits
			a.predicted += h.Predicted
			a.realized += h.Realized
		}
		for _, b := range v.ByLeverage {
			k := key{lev: b.Leverage, side: b.Side}
			if accB[k] == nil {
				accB[k] = &levelReviewAcc{horizon: b.Horizon}
			}
			a := accB[k]
			a.signals += b.Signals
			a.validated += b.Validated
			a.hits += b.Hits
			a.predicted += b.Predicted
			a.realized += b.Realized
		}
		samples = append(samples, v.Samples...)
	}
	hs := make([]LiquidationLevelsReviewHorizon, 0, len(LiquidationLevelsReviewHorizons))
	for _, spec := range LiquidationLevelsReviewHorizons {
		hs = append(hs, accH[spec.ID].finish())
	}
	out.Horizons = hs
	bs := make([]LiquidationLevelsReviewBucket, 0, len(accB))
	for _, lev := range ChartLeverageBuckets {
		for _, side := range []string{LiquidationSideShort, LiquidationSideLong} {
			a := accB[key{lev: lev, side: side}]
			if a == nil || a.signals == 0 {
				continue
			}
			h := a.finish()
			bs = append(bs, LiquidationLevelsReviewBucket{
				Leverage: lev, Side: side, Horizon: "1h",
				Signals: h.Signals, Validated: h.Validated, Hits: h.Hits,
				HitRate: h.HitRate, Predicted: h.Predicted, Realized: h.Realized,
				FillRate: h.FillRate, Reliability: levelReliability(h.Validated, h.HitRate, h.FillRate),
			})
		}
	}
	out.ByLeverage = bs
	sort.Slice(samples, func(i, j int) bool { return samples[i].Time.Before(samples[j].Time) })
	if len(samples) > LiquidationLevelsReviewMaxSamples {
		samples = samples[len(samples)-LiquidationLevelsReviewMaxSamples:]
	}
	out.Samples = samples
	return out
}

// ExplainLiquidationLevelsReview is a one-line read of combined 1h reliability.
func ExplainLiquidationLevelsReview(r LiquidationLevelsReview) string {
	var h1 LiquidationLevelsReviewHorizon
	for _, h := range r.Combined.Horizons {
		if h.Horizon == "1h" {
			h1 = h
			break
		}
	}
	if h1.Validated == 0 {
		if h1.Signals == 0 {
			return "No past estimated levels could be rebuilt from stored open interest and last price."
		}
		return fmt.Sprintf("%d estimated levels are still waiting on later liquidation history.", h1.Pending)
	}
	best := ""
	bestRate := -1.0
	for _, b := range r.Combined.ByLeverage {
		if b.Validated < LiquidationLevelsReviewMinValidated {
			continue
		}
		if b.HitRate > bestRate {
			bestRate = b.HitRate
			best = fmt.Sprintf("%dx %s", b.Leverage, b.Side)
		}
	}
	msg := fmt.Sprintf("Of %d estimated levels with later tape, %d printed nearby in 1h (%.0f%%). Realized was %.0f%% of predicted.",
		h1.Validated, h1.Hits, h1.HitRate*100, h1.FillRate*100)
	if best != "" {
		msg += " Most reliable: " + best + "."
	}
	return msg
}

func downsampleSnapshots(in []FuturesSnapshot, step time.Duration) []FuturesSnapshot {
	if step <= 0 {
		step = 30 * time.Minute
	}
	best := map[int64]FuturesSnapshot{}
	for _, r := range in {
		if r.SampledAt.IsZero() || r.Value <= 0 {
			continue
		}
		key := TruncateToBucket(r.SampledAt, step).Unix()
		prev, ok := best[key]
		if !ok || r.SampledAt.After(prev.SampledAt) {
			best[key] = r
		}
	}
	out := make([]FuturesSnapshot, 0, len(best))
	for _, r := range best {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SampledAt.Before(out[j].SampledAt) })
	return out
}

func snapshotAtOrBefore(in []FuturesSnapshot, t time.Time, stale time.Duration) *FuturesSnapshot {
	var best *FuturesSnapshot
	for i := range in {
		r := &in[i]
		if r.SampledAt.IsZero() || r.SampledAt.After(t) {
			continue
		}
		if stale > 0 && t.Sub(r.SampledAt) > stale {
			continue
		}
		if best == nil || r.SampledAt.After(best.SampledAt) {
			best = r
		}
	}
	return best
}

func levelPriceAtOrBefore(in []HuntHeatmapPricePoint, t time.Time, stale time.Duration) (float64, bool) {
	var best HuntHeatmapPricePoint
	found := false
	for _, p := range in {
		if p.Price <= 0 || p.Time.IsZero() || p.Time.After(t) {
			continue
		}
		if stale > 0 && t.Sub(p.Time) > stale {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best = p
			found = true
		}
	}
	return best.Price, found
}
