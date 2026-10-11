package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	HuntHeatmapPriceBins = 48
	// MaxHuntHeatmapHistory is the raw page of newest OI, long/short, and
	// funding rows. Windows past 24 hours do not use this cap. They read one
	// stored sample per bucket so older hourly rows stay in the grid.
	// See HuntSnapshotBucket.
	MaxHuntHeatmapHistory = 2500
	// HuntHeatmapMaxColumns caps the time axis (30d at 6h is 120; 90d at 1d is 90).
	HuntHeatmapMaxColumns = 128
)

// HuntHeatmapRange is a fixed lookback for the price×time intensity grid.
type HuntHeatmapRange string

const (
	HuntHeatmap12h HuntHeatmapRange = "12h"
	HuntHeatmap24h HuntHeatmapRange = "24h"
	HuntHeatmap3d  HuntHeatmapRange = "3d"
	HuntHeatmap7d  HuntHeatmapRange = "7d"
	HuntHeatmap14d HuntHeatmapRange = "14d"
	HuntHeatmap30d HuntHeatmapRange = "30d"
	HuntHeatmap90d HuntHeatmapRange = "90d"
)

// HuntHeatmapSpec is window + column step + candle interval for a range.
type HuntHeatmapSpec struct {
	Range     HuntHeatmapRange
	Window    time.Duration
	Step      time.Duration
	CandleIV  string
	CandleLim int
}

// HuntHeatmapSpecs is the supported CoinGlass-style windows.
var HuntHeatmapSpecs = []HuntHeatmapSpec{
	{Range: HuntHeatmap12h, Window: 12 * time.Hour, Step: 15 * time.Minute, CandleIV: "15m", CandleLim: 60},
	{Range: HuntHeatmap24h, Window: 24 * time.Hour, Step: 30 * time.Minute, CandleIV: "15m", CandleLim: 110},
	{Range: HuntHeatmap3d, Window: 72 * time.Hour, Step: time.Hour, CandleIV: "1h", CandleLim: 80},
	{Range: HuntHeatmap7d, Window: 168 * time.Hour, Step: 2 * time.Hour, CandleIV: "1h", CandleLim: 180},
	{Range: HuntHeatmap14d, Window: 14 * 24 * time.Hour, Step: 4 * time.Hour, CandleIV: "4h", CandleLim: 90},
	{Range: HuntHeatmap30d, Window: 30 * 24 * time.Hour, Step: 6 * time.Hour, CandleIV: "4h", CandleLim: 190},
	{Range: HuntHeatmap90d, Window: 90 * 24 * time.Hour, Step: 24 * time.Hour, CandleIV: "1d", CandleLim: 100},
}

// HuntSnapshotBucket is the stored-sample step for a heatmap or open-level
// window. Up to 24 hours stays on the newest raw rows. A longer window asks
// for one sample per chart bucket so newer five-minute rows do not crowd
// out older hourly archive rows. A zero result means the raw page.
func HuntSnapshotBucket(span time.Duration) time.Duration {
	if span <= 24*time.Hour {
		return 0
	}
	return FuturesPaneBucketForSpan(span)
}

// ParseHuntHeatmapRange accepts 12h, 24h, 3d, 7d, 14d, 30d, 90d (default 24h).
func ParseHuntHeatmapRange(raw string) (HuntHeatmapSpec, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = string(HuntHeatmap24h)
	}
	switch s {
	case "12h", "12":
		return HuntHeatmapSpecs[0], nil
	case "24h", "24", "1d":
		return HuntHeatmapSpecs[1], nil
	case "3d", "72h":
		return HuntHeatmapSpecs[2], nil
	case "7d", "168h", "1w":
		return HuntHeatmapSpecs[3], nil
	case "14d", "336h", "2w":
		return HuntHeatmapSpecs[4], nil
	case "30d", "720h":
		return HuntHeatmapSpecs[5], nil
	case "90d", "2160h":
		return HuntHeatmapSpecs[6], nil
	default:
		return HuntHeatmapSpec{}, fmt.Errorf("%w: range must be 12h, 24h, 3d, 7d, 14d, 30d, or 90d", ErrInvalidArgument)
	}
}

// HuntHeatmapPricePoint is one venue's own print used to place bands
// and to score later zone hits. Never mix venues on one series.
type HuntHeatmapPricePoint struct {
	Time  time.Time
	Price float64
	High  float64
	Low   float64
}

// HuntHeatmapVenueSeries is one venue's history for the heatmap.
type HuntHeatmapVenueSeries struct {
	Exchange     Exchange
	Prices       []HuntHeatmapPricePoint
	OI           []FuturesSnapshot
	LongShort    []FuturesSnapshot
	Funding      []FuturesSnapshot
	Liquidations []LiquidationEvent
}

// HuntHeatmapInput feeds BuildHuntHeatmap.
type HuntHeatmapInput struct {
	Symbol string
	Spec   HuntHeatmapSpec
	To     time.Time
	Venues []HuntHeatmapVenueSeries
}

// HuntHeatmapGrid is intensity for one venue (or combined).
// Matrices are [time][price] with price index 0 = highest bin.
type HuntHeatmapGrid struct {
	Exchange      string      `json:"exchange"`
	Longs         [][]float64 `json:"longs"`
	Shorts        [][]float64 `json:"shorts"`
	Totals        [][]float64 `json:"totals"`
	MaxIntensity  float64     `json:"maxIntensity"`
	Coverage      float64     `json:"coverage"`
	ColumnsWithOI int         `json:"columnsWithOi"`
	LastPrice     float64     `json:"lastPrice,omitempty"`
	HasData       []bool      `json:"hasData,omitempty"`
	// AccountBlend is stored when a column used that venue's account ratio,
	// even when the column fell back to 50/50, and mixed when both happened.
	// OKX uses its stored account ratio. A column with no ratio is even.
	AccountBlend string `json:"accountBlend,omitempty"`
	// PrintAnchors is measured long and short notional since 00:00 UTC and
	// the current 4-hour open. Empty on the leverage model.
	PrintAnchors []PrintAnchor `json:"printAnchors,omitempty"`
}

// HuntHeatmapReport is a CoinGlass-style price × time liquidation intensity map.
type HuntHeatmapReport struct {
	Symbol        string               `json:"symbol"`
	Range         string               `json:"range"`
	From          time.Time            `json:"from"`
	To            time.Time            `json:"to"`
	StepSec       int                  `json:"stepSec"`
	PriceMin      float64              `json:"priceMin"`
	PriceMax      float64              `json:"priceMax"`
	PriceStep     float64              `json:"priceStep"`
	Prices        []float64            `json:"prices"`
	Times         []time.Time          `json:"times"`
	Binance       HuntHeatmapGrid      `json:"binance"`
	Bybit         HuntHeatmapGrid      `json:"bybit"`
	OKX           HuntHeatmapGrid      `json:"okx"`
	Combined      HuntHeatmapGrid      `json:"combined"`
	Review        HuntHeatmapReview    `json:"review"`
	LeverageMix   []HuntLeverageBucket `json:"leverageMix,omitempty"`
	MissingVenues []string             `json:"missingVenues,omitempty"`
	// Kind is model (open-interest estimate) or prints (stored force orders only).
	Kind string `json:"kind,omitempty"`
	Note string `json:"note,omitempty"`
}

const huntHeatmapNote = "Estimated liquidation intensity at each price and time from historical open interest, that venue's own price, the hunt leverage mix, and blended account long/short. This grid does not add force-order prints; those are the separate measured map. Binance, Bybit, and OKX are modeled separately and never borrow each other's prices. Each venue uses its stored account ratio. A column with no stored ratio is an even split. Combined sums a column only when all three venues have their own price and open interest. Not a live order book and not financial advice."

// HuntHeatmapColumns is the UTC-aligned time axis ending at or before to.
func HuntHeatmapColumns(spec HuntHeatmapSpec, to time.Time) []time.Time {
	if spec.Step <= 0 || spec.Window <= 0 {
		return nil
	}
	to = TruncateToBucket(to.UTC(), spec.Step)
	from := to.Add(-spec.Window)
	out := make([]time.Time, 0, int(spec.Window/spec.Step)+1)
	for t := from.Add(spec.Step); !t.After(to); t = t.Add(spec.Step) {
		out = append(out, t)
		if len(out) >= HuntHeatmapMaxColumns {
			break
		}
	}
	return out
}

// HuntHeatmapMaxCols is a hard cap on time columns.
func HuntHeatmapMaxCols() int { return HuntHeatmapMaxColumns }

// BuildHuntHeatmap projects the hunt liquidation model onto a price × time grid.
func BuildHuntHeatmap(in HuntHeatmapInput) HuntHeatmapReport {
	spec := in.Spec
	if spec.Step <= 0 {
		spec, _ = ParseHuntHeatmapRange("24h")
	}
	to := in.To.UTC()
	if to.IsZero() {
		to = time.Now().UTC()
	}
	times := HuntHeatmapColumns(spec, to)
	from := to.Add(-spec.Window)
	if len(times) > 0 {
		from = times[0].Add(-spec.Step)
	}
	out := HuntHeatmapReport{
		Symbol:      NormalizeLiquidationSymbol(in.Symbol),
		Range:       string(spec.Range),
		Kind:        HuntHeatmapKindModel,
		From:        from,
		To:          to,
		StepSec:     int(spec.Step.Seconds()),
		Times:       times,
		LeverageMix: huntLeverageMixCopy(),
		Note:        huntHeatmapNote,
	}
	pMin, pMax := huntHeatmapPriceExtent(allHuntVenuePrices(in.Venues), times)
	nBin := HuntHeatmapPriceBins
	if pMax <= pMin || len(times) == 0 {
		out.Binance = emptyHuntGrid("binance", times, nBin)
		out.Bybit = emptyHuntGrid("bybit", times, nBin)
		out.OKX = emptyHuntGrid("okx", times, nBin)
		out.Combined = emptyHuntGrid("combined", times, nBin)
		out.MissingVenues = missingHuntVenues(out.Binance, out.Bybit, out.OKX)
		out.Review = emptyHuntHeatmapReview()
		return out
	}
	step := (pMax - pMin) / float64(nBin)
	if step <= 0 {
		step = pMin * 0.001
		if step <= 0 {
			step = 1
		}
		pMax = pMin + step*float64(nBin)
	}
	prices := make([]float64, nBin)
	for i := 0; i < nBin; i++ {
		prices[i] = pMax - (float64(i)+0.5)*step
	}
	out.PriceMin, out.PriceMax, out.PriceStep = pMin, pMax, step
	out.Prices = prices

	byEx := map[Exchange]HuntHeatmapVenueSeries{}
	for _, v := range in.Venues {
		byEx[v.Exchange] = v
	}
	out.Binance = buildHuntVenueGrid("binance", byEx[ExchangeBinance], times, spec.Step, byEx[ExchangeBinance].Prices, pMin, pMax, step, nBin)
	out.Bybit = buildHuntVenueGrid("bybit", byEx[ExchangeBybit], times, spec.Step, byEx[ExchangeBybit].Prices, pMin, pMax, step, nBin)
	out.OKX = buildHuntVenueGrid("okx", byEx[ExchangeOKX], times, spec.Step, byEx[ExchangeOKX].Prices, pMin, pMax, step, nBin)
	out.Binance.LastPrice = lastHuntSeriesPrice(byEx[ExchangeBinance].Prices)
	out.Bybit.LastPrice = lastHuntSeriesPrice(byEx[ExchangeBybit].Prices)
	out.OKX.LastPrice = lastHuntSeriesPrice(byEx[ExchangeOKX].Prices)
	out.Combined = sumHuntGrids("combined", out.Binance, out.Bybit, out.OKX)
	out.MissingVenues = missingHuntVenues(out.Binance, out.Bybit, out.OKX)
	out.Review = ReviewHuntHeatmap(out, in.Venues, to)
	return out
}

func huntLeverageMixCopy() []HuntLeverageBucket {
	return append([]HuntLeverageBucket(nil), DefaultHuntLeverageMix...)
}

func missingHuntVenues(binance, bybit, okx HuntHeatmapGrid) []string {
	var out []string
	if binance.ColumnsWithOI == 0 {
		out = append(out, "binance")
	}
	if bybit.ColumnsWithOI == 0 {
		out = append(out, "bybit")
	}
	if okx.ColumnsWithOI == 0 {
		out = append(out, "okx")
	}
	return out
}

func lastHuntSeriesPrice(pts []HuntHeatmapPricePoint) float64 {
	var best time.Time
	var px float64
	for _, p := range pts {
		if p.Price <= 0 {
			continue
		}
		if best.IsZero() || p.Time.After(best) {
			best = p.Time
			px = p.Price
		}
	}
	return px
}

func allHuntVenuePrices(venues []HuntHeatmapVenueSeries) []HuntHeatmapPricePoint {
	n := 0
	for _, v := range venues {
		n += len(v.Prices)
	}
	out := make([]HuntHeatmapPricePoint, 0, n)
	for _, v := range venues {
		out = append(out, v.Prices...)
	}
	return out
}

func huntHeatmapPriceExtent(prices []HuntHeatmapPricePoint, times []time.Time) (float64, float64) {
	lo, hi := 0.0, 0.0
	have := false
	var t0, t1 time.Time
	if len(times) > 0 {
		t0, t1 = times[0], times[len(times)-1]
	}
	for _, p := range prices {
		if p.Price <= 0 {
			continue
		}
		if !t0.IsZero() && p.Time.Before(t0.Add(-2*time.Hour)) {
			continue
		}
		if !t1.IsZero() && p.Time.After(t1.Add(2*time.Hour)) {
			continue
		}
		if !have {
			lo, hi, have = p.Price, p.Price, true
			continue
		}
		if p.Price < lo {
			lo = p.Price
		}
		if p.Price > hi {
			hi = p.Price
		}
	}
	if !have {
		// Candle CloseTime can sit outside the column window (stub/test feeds,
		// a single last print). Still need a price axis.
		for _, p := range prices {
			if p.Price <= 0 {
				continue
			}
			if !have {
				lo, hi, have = p.Price, p.Price, true
				continue
			}
			if p.Price < lo {
				lo = p.Price
			}
			if p.Price > hi {
				hi = p.Price
			}
		}
	}
	if !have {
		return 0, 0
	}
	// Include the farthest 5× isolated band (~20%).
	pad := HuntLiqDistance(5, HuntMaintenanceMargin) + 0.01
	lo -= hi * pad
	hi += hi * pad
	if lo <= 0 {
		lo = hi * 0.01
	}
	return lo, hi
}

func emptyHuntGrid(ex string, times []time.Time, nBin int) HuntHeatmapGrid {
	nT := len(times)
	g := HuntHeatmapGrid{
		Exchange: ex,
		Longs:    make([][]float64, nT),
		Shorts:   make([][]float64, nT),
		Totals:   make([][]float64, nT),
		HasData:  make([]bool, nT),
	}
	for i := 0; i < nT; i++ {
		g.Longs[i] = make([]float64, nBin)
		g.Shorts[i] = make([]float64, nBin)
		g.Totals[i] = make([]float64, nBin)
	}
	return g
}

func buildHuntVenueGrid(ex string, ser HuntHeatmapVenueSeries, times []time.Time, step time.Duration, prices []HuntHeatmapPricePoint, pMin, pMax, pStep float64, nBin int) HuntHeatmapGrid {
	g := emptyHuntGrid(ex, times, nBin)
	if len(times) == 0 {
		return g
	}
	stale := huntMatchStale(step)
	withOI := 0
	storedBlend, evenBlend := 0, 0
	for i, t := range times {
		px := nearestHuntPrice(prices, t, stale)
		oi := nearestHuntSnapshot(ser.OI, t, stale)
		if px <= 0 || oi <= 0 {
			continue
		}
		g.HasData[i] = true
		withOI++
		estLong, estShort, stored := nearestHuntBlendShares(ser.LongShort, t, stale*2)
		if stored {
			storedBlend++
		} else {
			evenBlend++
		}
		fund := nearestHuntFunding(ser.Funding, t, 8*time.Hour)
		longsCrowded := fund > 1e-12
		shortsCrowded := fund < -1e-12
		longMix := TiltLeverageMix(DefaultHuntLeverageMix, longsCrowded)
		shortMix := TiltLeverageMix(DefaultHuntLeverageMix, shortsCrowded)
		up := modelBands(LiquidationSideShort, "up", px, oi*estShort, shortMix, 1)
		down := modelBands(LiquidationSideLong, "down", px, oi*estLong, longMix, -1)
		addBandsToColumn(g.Shorts[i], up, pMin, pMax, pStep, nBin)
		addBandsToColumn(g.Longs[i], down, pMin, pMax, pStep, nBin)
		for j := 0; j < nBin; j++ {
			g.Totals[i][j] = g.Longs[i][j] + g.Shorts[i][j]
			if g.Totals[i][j] > g.MaxIntensity {
				g.MaxIntensity = g.Totals[i][j]
			}
		}
	}
	if len(times) > 0 {
		g.Coverage = float64(withOI) / float64(len(times))
	}
	g.ColumnsWithOI = withOI
	g.AccountBlend = huntAccountBlendLabel(storedBlend, evenBlend)
	return g
}

func huntAccountBlendLabel(stored, even int) string {
	switch {
	case stored > 0 && even > 0:
		return "mixed"
	case stored > 0:
		return "stored"
	case even > 0:
		return "even"
	default:
		return ""
	}
}

func addBandsToColumn(col []float64, bands []HuntBand, pMin, pMax, pStep float64, nBin int) {
	for _, b := range bands {
		if b.EstNotional <= 0 {
			continue
		}
		idx := huntPriceBin(b.Price, pMin, pMax, pStep, nBin)
		if idx < 0 {
			continue
		}
		col[idx] += b.EstNotional
	}
}

func huntPriceBin(px, pMin, pMax, pStep float64, nBin int) int {
	if px <= 0 || pStep <= 0 || nBin <= 0 {
		return -1
	}
	if px < pMin || px > pMax {
		if px < pMin {
			return nBin - 1
		}
		return 0
	}
	i := int(math.Floor((pMax - px) / pStep))
	if i < 0 {
		return 0
	}
	if i >= nBin {
		return nBin - 1
	}
	return i
}

func huntMatchStale(step time.Duration) time.Duration {
	stale := step * 2
	if stale < 20*time.Minute {
		stale = 20 * time.Minute
	}
	return stale
}

// HuntOICovers reports whether every heatmap column has an open-interest
// sample inside the same stale window the grid uses to paint a cell.
func HuntOICovers(rows []FuturesSnapshot, times []time.Time, step time.Duration) bool {
	_, _, missing := HuntOIGap(rows, times, step)
	return !missing
}

// HuntOIGap returns the earliest and latest column times that have no
// open-interest sample, widened by the match window so a fetched print can
// land on those columns. missing is false when every column is covered.
func HuntOIGap(rows []FuturesSnapshot, times []time.Time, step time.Duration) (from, to time.Time, missing bool) {
	if len(times) == 0 {
		return time.Time{}, time.Time{}, false
	}
	stale := huntMatchStale(step)
	var lo, hi time.Time
	for _, t := range times {
		if nearestHuntSnapshot(rows, t, stale) > 0 {
			continue
		}
		if !missing || t.Before(lo) {
			lo = t
		}
		if !missing || t.After(hi) {
			hi = t
		}
		missing = true
	}
	if !missing {
		return time.Time{}, time.Time{}, false
	}
	return lo.Add(-stale), hi.Add(stale), true
}

func nearestHuntPrice(pts []HuntHeatmapPricePoint, at time.Time, stale time.Duration) float64 {
	var best float64
	var bestDt time.Duration = -1
	for _, p := range pts {
		if p.Price <= 0 {
			continue
		}
		dt := at.Sub(p.Time)
		if dt < 0 {
			dt = -dt
		}
		if stale > 0 && dt > stale {
			continue
		}
		if bestDt < 0 || dt < bestDt {
			bestDt = dt
			best = p.Price
		}
	}
	return best
}

func nearestHuntSnapshot(rows []FuturesSnapshot, at time.Time, stale time.Duration) float64 {
	var best float64
	var bestDt time.Duration = -1
	for _, r := range rows {
		v := r.Value
		if v <= 0 {
			v = r.Contracts
		}
		if v <= 0 {
			continue
		}
		dt := at.Sub(r.SampledAt)
		if dt < 0 {
			dt = -dt
		}
		if stale > 0 && dt > stale {
			continue
		}
		if bestDt < 0 || dt < bestDt {
			bestDt = dt
			best = v
		}
	}
	return best
}

func nearestHuntSnapshotShare(rows []FuturesSnapshot, at time.Time, stale time.Duration) float64 {
	estLong, _, _ := nearestHuntBlendShares(rows, at, stale)
	return estLong
}

func nearestHuntBlendShares(rows []FuturesSnapshot, at time.Time, stale time.Duration) (estLong, estShort float64, stored bool) {
	var best *FuturesSnapshot
	var bestDt time.Duration = -1
	for i := range rows {
		r := &rows[i]
		if r.LongShare <= 0 && r.ShortShare <= 0 && r.SizeLongShare <= 0 {
			continue
		}
		dt := at.Sub(r.SampledAt)
		if dt < 0 {
			dt = -dt
		}
		if stale > 0 && dt > stale {
			continue
		}
		if bestDt < 0 || dt < bestDt {
			bestDt = dt
			best = r
		}
	}
	if best == nil {
		return 0.5, 0.5, false
	}
	estLong, estShort = HuntBlendFromSnapshot(best)
	return estLong, estShort, true
}

func nearestHuntFunding(rows []FuturesSnapshot, at time.Time, stale time.Duration) float64 {
	var best float64
	var bestDt time.Duration = -1
	for _, r := range rows {
		dt := at.Sub(r.SampledAt)
		if dt < 0 {
			dt = -dt
		}
		if stale > 0 && dt > stale {
			continue
		}
		if bestDt < 0 || dt < bestDt {
			bestDt = dt
			best = r.FundingRate
		}
	}
	return best
}

func sumHuntGrids(ex string, parts ...HuntHeatmapGrid) HuntHeatmapGrid {
	nT := 0
	nBin := HuntHeatmapPriceBins
	for _, p := range parts {
		if len(p.Totals) > nT {
			nT = len(p.Totals)
		}
		if len(p.Totals) > 0 && len(p.Totals[0]) > 0 {
			nBin = len(p.Totals[0])
		}
	}
	g := emptyHuntGrid(ex, make([]time.Time, nT), nBin)
	max := 0.0
	covered := 0
	for i := 0; i < nT; i++ {
		// Combined sums a column only when every venue had its own price and OI.
		ok := len(parts) > 0
		for _, p := range parts {
			if i >= len(p.HasData) || !p.HasData[i] {
				ok = false
				break
			}
		}
		g.HasData[i] = ok
		if !ok {
			continue
		}
		covered++
		for j := 0; j < nBin; j++ {
			var lv, sv float64
			for _, p := range parts {
				if i < len(p.Longs) && j < len(p.Longs[i]) {
					lv += p.Longs[i][j]
				}
				if i < len(p.Shorts) && j < len(p.Shorts[i]) {
					sv += p.Shorts[i][j]
				}
			}
			g.Longs[i][j] = lv
			g.Shorts[i][j] = sv
			g.Totals[i][j] = lv + sv
			if g.Totals[i][j] > max {
				max = g.Totals[i][j]
			}
		}
	}
	if nT > 0 {
		g.Coverage = float64(covered) / float64(nT)
		g.ColumnsWithOI = covered
	}
	g.MaxIntensity = max
	g.AccountBlend = combinedAccountBlend(parts)
	return g
}

func combinedAccountBlend(parts []HuntHeatmapGrid) string {
	stored, even := false, false
	for _, p := range parts {
		switch p.AccountBlend {
		case "stored":
			stored = true
		case "even":
			even = true
		case "mixed":
			stored, even = true, true
		}
	}
	return huntAccountBlendLabel(boolCount(stored), boolCount(even))
}

func boolCount(v bool) int {
	if v {
		return 1
	}
	return 0
}
