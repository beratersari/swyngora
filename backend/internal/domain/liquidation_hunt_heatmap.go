package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	HuntHeatmapPriceBins = 48
	// MaxHuntHeatmapHistory is how many OI / LS / funding rows we load per metric.
	MaxHuntHeatmapHistory = 2500
)

// HuntHeatmapRange is a fixed lookback for the price×time intensity grid.
type HuntHeatmapRange string

const (
	HuntHeatmap12h HuntHeatmapRange = "12h"
	HuntHeatmap24h HuntHeatmapRange = "24h"
	HuntHeatmap3d  HuntHeatmapRange = "3d"
	HuntHeatmap7d  HuntHeatmapRange = "7d"
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
}

// ParseHuntHeatmapRange accepts 12h, 24h, 3d, 7d (default 24h).
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
	default:
		return HuntHeatmapSpec{}, fmt.Errorf("%w: range must be 12h, 24h, 3d, or 7d", ErrInvalidArgument)
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
}

// HuntHeatmapReport is a CoinGlass-style price × time liquidation intensity map.
type HuntHeatmapReport struct {
	Symbol    string            `json:"symbol"`
	Range     string            `json:"range"`
	From      time.Time         `json:"from"`
	To        time.Time         `json:"to"`
	StepSec   int               `json:"stepSec"`
	PriceMin  float64           `json:"priceMin"`
	PriceMax  float64           `json:"priceMax"`
	PriceStep float64           `json:"priceStep"`
	Prices    []float64         `json:"prices"`
	Times     []time.Time       `json:"times"`
	Binance   HuntHeatmapGrid   `json:"binance"`
	Bybit     HuntHeatmapGrid   `json:"bybit"`
	Combined  HuntHeatmapGrid   `json:"combined"`
	Review    HuntHeatmapReview `json:"review"`
	Note      string            `json:"note,omitempty"`
}

const huntHeatmapNote = "Estimated liquidation intensity at each price and time from historical open interest, that venue's own price, the hunt leverage mix, blended account long/short, and observed liquidation prints in that column. Binance and Bybit are modeled separately and never borrow each other's prices; combined is the sum. Not a live order book and not financial advice."

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
		if len(out) >= HuntHeatmapMaxCols() {
			break
		}
	}
	return out
}

// HuntHeatmapMaxCols is a hard cap on time columns.
func HuntHeatmapMaxCols() int { return 96 }

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
		Symbol:  NormalizeLiquidationSymbol(in.Symbol),
		Range:   string(spec.Range),
		From:    from,
		To:      to,
		StepSec: int(spec.Step.Seconds()),
		Times:   times,
		Note:    huntHeatmapNote,
	}
	pMin, pMax := huntHeatmapPriceExtent(allHuntVenuePrices(in.Venues), times)
	nBin := HuntHeatmapPriceBins
	if pMax <= pMin || len(times) == 0 {
		out.Binance = emptyHuntGrid("binance", times, nBin)
		out.Bybit = emptyHuntGrid("bybit", times, nBin)
		out.Combined = emptyHuntGrid("combined", times, nBin)
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
	out.Combined = sumHuntGrids("combined", out.Binance, out.Bybit)
	out.Review = ReviewHuntHeatmap(out, in.Venues, to)
	return out
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
	stale := step * 2
	if stale < 20*time.Minute {
		stale = 20 * time.Minute
	}
	withOI := 0
	for i, t := range times {
		px := nearestHuntPrice(prices, t, stale)
		oi := nearestHuntSnapshot(ser.OI, t, stale)
		if px <= 0 || oi <= 0 {
			continue
		}
		withOI++
		ls := nearestHuntSnapshotShare(ser.LongShort, t, stale*2)
		fund := nearestHuntFunding(ser.Funding, t, 8*time.Hour)
		estLong, estShort := BlendAccountShare(ls)
		longsCrowded := fund > 1e-12
		shortsCrowded := fund < -1e-12
		longMix := TiltLeverageMix(DefaultHuntLeverageMix, longsCrowded)
		shortMix := TiltLeverageMix(DefaultHuntLeverageMix, shortsCrowded)
		up := modelBands(LiquidationSideShort, "up", px, oi*estShort, shortMix, 1)
		down := modelBands(LiquidationSideLong, "down", px, oi*estLong, longMix, -1)
		addBandsToColumn(g.Shorts[i], up, pMin, pMax, pStep, nBin)
		addBandsToColumn(g.Longs[i], down, pMin, pMax, pStep, nBin)
		addLiqsToColumn(g.Longs[i], g.Shorts[i], ser.Liquidations, t, step, pMin, pMax, pStep, nBin)
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
	return g
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

func addLiqsToColumn(longs, shorts []float64, ev []LiquidationEvent, col time.Time, step time.Duration, pMin, pMax, pStep float64, nBin int) {
	if step <= 0 {
		return
	}
	half := step / 2
	lo, hi := col.Add(-half), col.Add(half)
	for _, e := range ev {
		if e.Notional <= 0 || e.Price <= 0 {
			continue
		}
		if e.Time.Before(lo) || !e.Time.Before(hi) {
			continue
		}
		idx := huntPriceBin(e.Price, pMin, pMax, pStep, nBin)
		if idx < 0 {
			continue
		}
		if e.Side == LiquidationSideShort {
			shorts[idx] += e.Notional
		} else {
			longs[idx] += e.Notional
		}
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
	var best float64 = 0.5
	var bestDt time.Duration = -1
	for _, r := range rows {
		if r.LongShare <= 0 && r.ShortShare <= 0 {
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
			ls := r.LongShare
			if ls <= 0 && r.ShortShare > 0 {
				ls = 1 - r.ShortShare
			}
			best = ls
		}
	}
	return best
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

func sumHuntGrids(ex string, a, b HuntHeatmapGrid) HuntHeatmapGrid {
	nT := len(a.Totals)
	if len(b.Totals) > nT {
		nT = len(b.Totals)
	}
	nBin := HuntHeatmapPriceBins
	if len(a.Totals) > 0 {
		nBin = len(a.Totals[0])
	} else if len(b.Totals) > 0 {
		nBin = len(b.Totals[0])
	}
	g := emptyHuntGrid(ex, make([]time.Time, nT), nBin)
	max := 0.0
	for i := 0; i < nT; i++ {
		for j := 0; j < nBin; j++ {
			var lv, sv float64
			if i < len(a.Longs) && j < len(a.Longs[i]) {
				lv += a.Longs[i][j]
				sv += a.Shorts[i][j]
			}
			if i < len(b.Longs) && j < len(b.Longs[i]) {
				lv += b.Longs[i][j]
				sv += b.Shorts[i][j]
			}
			g.Longs[i][j] = lv
			g.Shorts[i][j] = sv
			g.Totals[i][j] = lv + sv
			if g.Totals[i][j] > max {
				max = g.Totals[i][j]
			}
		}
	}
	// Combined coverage: columns where either venue had OI.
	if nT > 0 {
		covered := 0
		for i := 0; i < nT; i++ {
			has := false
			for j := 0; j < nBin; j++ {
				if g.Totals[i][j] > 0 {
					has = true
					break
				}
			}
			if has {
				covered++
			}
		}
		g.Coverage = float64(covered) / float64(nT)
		g.ColumnsWithOI = covered
	}
	g.MaxIntensity = max
	return g
}
