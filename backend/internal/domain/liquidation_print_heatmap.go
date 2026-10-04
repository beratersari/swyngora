package domain

import (
	"context"
	"time"
)

const (
	// HuntHeatmapKindModel is the open-interest leverage estimate.
	HuntHeatmapKindModel = "model"
	// HuntHeatmapKindPrints is stored force-order notional only.
	HuntHeatmapKindPrints = "prints"
)

const measuredHeatmapNote = "Measured liquidation prints at each price and time. Longs are positions closed by a sell liquidation. Shorts are positions closed by a buy. These are stored force-order prints, not an open-interest leverage model. A column is recorded when a print landed there, or when that venue was watched and the column is outside a known feed gap. A quiet recorded column is a real zero. A column before the watch, or inside a gap with no print, stays empty. Binance, Bybit, and OKX never borrow each other's prints. Combined sums a column only when all three were recorded. Not financial advice."

// LiquidationPrintBin is one venue's notional in a time column and price row.
// TimeIndex 0 is the first heatmap column. PriceIndex 0 is the highest price.
type LiquidationPrintBin struct {
	Exchange   Exchange
	Side       string
	TimeIndex  int
	PriceIndex int
	Notional   float64
	Count      int
}

// MeasuredPrintQuery is the window and price axis shared by SQL and Go binning.
// Origin is the start of the first column (column time minus half a step).
// Until is the end of the last column (exclusive).
type MeasuredPrintQuery struct {
	Symbol    string
	Origin    time.Time
	Until     time.Time
	Step      time.Duration
	Columns   int
	PriceMin  float64
	PriceMax  float64
	PriceBins int
	Exchanges []Exchange
}

// MeasuredPrintScan is aggregated print notional plus the newest stored time per venue.
type MeasuredPrintScan struct {
	Bins     []LiquidationPrintBin
	Latest   map[Exchange]time.Time
	PriceMin float64
	PriceMax float64
}

// LiquidationPrintAggregator rolls stored prints into price × time bins.
// Implementations must use MeasuredTimeIndex and MeasuredPriceIndex.
type LiquidationPrintAggregator interface {
	AggregateLiquidationPrints(ctx context.Context, q MeasuredPrintQuery) (MeasuredPrintScan, error)
	LiquidationPrintPriceSpan(ctx context.Context, symbol string, from, to time.Time, exchanges []Exchange) (min, max float64, err error)
}

// MeasuredHeatmapInput builds a print-only heatmap on a fixed price axis.
type MeasuredHeatmapInput struct {
	Symbol   string
	Spec     HuntHeatmapSpec
	To       time.Time
	PriceMin float64
	PriceMax float64
	Bins     []LiquidationPrintBin
	Coverage []LiquidationCoverage
	Venues   []Exchange
}

// MeasuredTimeIndex is the column for t. Origin is the inclusive start of column 0.
// Integer division matches SQLite (time_ms - origin_ms) / step_ms.
func MeasuredTimeIndex(t, origin time.Time, step time.Duration) int {
	stepMS := step.Milliseconds()
	if stepMS <= 0 {
		return -1
	}
	return int((t.UTC().UnixMilli() - origin.UTC().UnixMilli()) / stepMS)
}

// MeasuredPriceIndex is the price row, 0 = highest. It matches huntPriceBin.
func MeasuredPriceIndex(px, pMin, pMax float64, nBin int) int {
	if nBin <= 0 {
		nBin = HuntHeatmapPriceBins
	}
	step := 0.0
	if pMax > pMin {
		step = (pMax - pMin) / float64(nBin)
	}
	return huntPriceBin(px, pMin, pMax, step, nBin)
}

// MeasuredPriceExtent pads candle closes the same way as the model map.
func MeasuredPriceExtent(prices []HuntHeatmapPricePoint, times []time.Time) (float64, float64) {
	return huntHeatmapPriceExtent(prices, times)
}

// MeasuredPrintPriceExtent pads a print-only high and low so one price is not a single line.
func MeasuredPrintPriceExtent(lo, hi float64) (float64, float64) {
	if hi <= 0 {
		return 0, 0
	}
	if lo <= 0 || lo > hi {
		lo = hi
	}
	pad := hi * 0.01
	span := hi - lo
	if span > 0 && pad < span*0.05 {
		pad = span * 0.05
	}
	if pad <= 0 {
		pad = 1
	}
	lo -= pad
	hi += pad
	if lo <= 0 {
		lo = hi * 0.01
	}
	if hi <= lo {
		hi = lo + 1
	}
	return lo, hi
}

// MeasuredHeatmapWindow is the half-open print window for the heatmap columns.
func MeasuredHeatmapWindow(spec HuntHeatmapSpec, to time.Time) (times []time.Time, origin, until time.Time) {
	if to.IsZero() {
		to = time.Now().UTC()
	}
	times = HuntHeatmapColumns(spec, to.UTC())
	if len(times) == 0 || spec.Step <= 0 {
		return times, time.Time{}, time.Time{}
	}
	origin = times[0].Add(-spec.Step / 2)
	until = times[len(times)-1].Add(spec.Step / 2)
	// The last column is the truncated step. Prints after its closed edge
	// and before now still belong on that column.
	if to.After(until) {
		until = to.UTC()
	}
	return times, origin, until
}

// BinMeasuredPrintEvents groups raw prints onto the query axis.
// Events at or after Until, or before Origin, are left out. Latest is the newest
// included print time per venue, including prints that miss the price axis.
func BinMeasuredPrintEvents(q MeasuredPrintQuery, events []LiquidationEvent) MeasuredPrintScan {
	scan := MeasuredPrintScan{Latest: map[Exchange]time.Time{}}
	nBin := q.PriceBins
	if nBin <= 0 {
		nBin = HuntHeatmapPriceBins
	}
	type key struct {
		ex         Exchange
		side       string
		tBin, pBin int
	}
	acc := map[key]*LiquidationPrintBin{}
	order := make([]key, 0, len(events))
	for _, e := range events {
		if e.Notional <= 0 || e.Price <= 0 || e.Time.IsZero() || e.Exchange == "" {
			continue
		}
		if len(q.Exchanges) > 0 && !measuredExchangeWanted(q.Exchanges, e.Exchange) {
			continue
		}
		t := e.Time.UTC()
		if !q.Origin.IsZero() && t.Before(q.Origin) {
			continue
		}
		if !q.Until.IsZero() && !t.Before(q.Until) {
			continue
		}
		if prev, ok := scan.Latest[e.Exchange]; !ok || t.After(prev) {
			scan.Latest[e.Exchange] = t
		}
		ti := MeasuredTimeIndex(t, q.Origin, q.Step)
		if ti < 0 || (q.Columns > 0 && ti > q.Columns) {
			continue
		}
		if q.Columns > 0 && ti == q.Columns {
			ti = q.Columns - 1
		}
		pi := MeasuredPriceIndex(e.Price, q.PriceMin, q.PriceMax, nBin)
		if pi < 0 {
			continue
		}
		side := LiquidationSideLong
		if e.Side == LiquidationSideShort {
			side = LiquidationSideShort
		}
		k := key{e.Exchange, side, ti, pi}
		b := acc[k]
		if b == nil {
			b = &LiquidationPrintBin{Exchange: e.Exchange, Side: side, TimeIndex: ti, PriceIndex: pi}
			acc[k] = b
			order = append(order, k)
		}
		b.Notional += e.Notional
		b.Count++
	}
	scan.Bins = make([]LiquidationPrintBin, 0, len(order))
	for _, k := range order {
		scan.Bins = append(scan.Bins, *acc[k])
	}
	return scan
}

func measuredExchangeWanted(want []Exchange, ex Exchange) bool {
	for _, v := range want {
		if v == ex {
			return true
		}
	}
	return false
}

// BuildMeasuredLiquidationHeatmap bins stored prints only. It does not add model bands.
// A column with a print is recorded. A column after FirstWatch and outside a known gap,
// with no print, is a real zero. Anything else stays empty and is left out of combined.
func BuildMeasuredLiquidationHeatmap(in MeasuredHeatmapInput) HuntHeatmapReport {
	spec := in.Spec
	if spec.Step <= 0 {
		spec, _ = ParseHuntHeatmapRange("24h")
	}
	to := in.To.UTC()
	if to.IsZero() {
		to = time.Now().UTC()
	}
	times, _, _ := MeasuredHeatmapWindow(spec, to)
	from := to.Add(-spec.Window)
	if len(times) > 0 {
		from = times[0].Add(-spec.Step)
	}
	want := in.Venues
	if len(want) == 0 {
		want = FuturesVenues()
	}
	out := HuntHeatmapReport{
		Symbol:  NormalizeLiquidationSymbol(in.Symbol),
		Range:   string(spec.Range),
		Kind:    HuntHeatmapKindPrints,
		From:    from,
		To:      to,
		StepSec: int(spec.Step.Seconds()),
		Times:   times,
		Note:    measuredHeatmapNote,
		Review:  emptyMeasuredReview(),
	}
	nBin := HuntHeatmapPriceBins
	pMin, pMax := in.PriceMin, in.PriceMax
	if pMax <= pMin || len(times) == 0 {
		out.Binance = emptyHuntGrid("binance", times, nBin)
		out.Bybit = emptyHuntGrid("bybit", times, nBin)
		out.OKX = emptyHuntGrid("okx", times, nBin)
		out.Combined = emptyHuntGrid("combined", times, nBin)
		out.MissingVenues = missingHuntVenues(out.Binance, out.Bybit, out.OKX)
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

	grids := map[Exchange]HuntHeatmapGrid{}
	for _, ex := range want {
		grids[ex] = buildMeasuredVenueGrid(string(ex), ex, out.Symbol, times, spec.Step, in.Bins, in.Coverage, nBin)
	}
	out.Binance = gridOrEmpty(grids, ExchangeBinance, "binance", times, nBin)
	out.Bybit = gridOrEmpty(grids, ExchangeBybit, "bybit", times, nBin)
	out.OKX = gridOrEmpty(grids, ExchangeOKX, "okx", times, nBin)
	out.Combined = sumHuntGrids("combined", out.Binance, out.Bybit, out.OKX)
	out.MissingVenues = missingHuntVenues(out.Binance, out.Bybit, out.OKX)
	return out
}

func gridOrEmpty(grids map[Exchange]HuntHeatmapGrid, ex Exchange, name string, times []time.Time, nBin int) HuntHeatmapGrid {
	if g, ok := grids[ex]; ok {
		return g
	}
	return emptyHuntGrid(name, times, nBin)
}

func buildMeasuredVenueGrid(name string, ex Exchange, symbol string, times []time.Time, step time.Duration, bins []LiquidationPrintBin, coverage []LiquidationCoverage, nBin int) HuntHeatmapGrid {
	g := emptyHuntGrid(name, times, nBin)
	if len(times) == 0 {
		return g
	}
	hasPrint := make([]bool, len(times))
	for _, b := range bins {
		if b.Exchange != ex || b.Notional <= 0 {
			continue
		}
		if b.TimeIndex < 0 || b.TimeIndex >= len(times) || b.PriceIndex < 0 || b.PriceIndex >= nBin {
			continue
		}
		if b.Side == LiquidationSideShort {
			g.Shorts[b.TimeIndex][b.PriceIndex] += b.Notional
		} else {
			g.Longs[b.TimeIndex][b.PriceIndex] += b.Notional
		}
		hasPrint[b.TimeIndex] = true
	}
	cov := printCoverage(ex, symbol, coverage)
	with := 0
	max := 0.0
	for i, col := range times {
		recorded := hasPrint[i] || printColumnRecorded(col, step, cov)
		g.HasData[i] = recorded
		if !recorded {
			zeroFloatRow(g.Longs[i])
			zeroFloatRow(g.Shorts[i])
			continue
		}
		with++
		for j := 0; j < nBin; j++ {
			g.Totals[i][j] = g.Longs[i][j] + g.Shorts[i][j]
			if g.Totals[i][j] > max {
				max = g.Totals[i][j]
			}
		}
	}
	g.ColumnsWithOI = with
	if len(times) > 0 {
		g.Coverage = float64(with) / float64(len(times))
	}
	g.MaxIntensity = max
	return g
}

func zeroFloatRow(row []float64) {
	for i := range row {
		row[i] = 0
	}
}

// printCoverage prefers the pair clock. A pair with no gaps keeps the venue-wide gaps.
// A pair that never started falls back to the venue-wide clock.
func printCoverage(ex Exchange, symbol string, rows []LiquidationCoverage) LiquidationCoverage {
	symbol = NormalizeLiquidationSymbol(symbol)
	var pair, venue LiquidationCoverage
	hasPair, hasVenue := false, false
	for _, r := range rows {
		if r.Exchange != ex {
			continue
		}
		if r.Symbol == "" {
			venue = r
			hasVenue = true
			continue
		}
		if symbol != "" && NormalizeLiquidationSymbol(r.Symbol) == symbol {
			pair = r
			hasPair = true
		}
	}
	if hasPair && !pair.FirstWatch.IsZero() {
		if len(pair.Gaps) == 0 && hasVenue {
			pair.Gaps = venue.Gaps
		}
		return pair
	}
	if hasVenue {
		return venue
	}
	return LiquidationCoverage{Exchange: ex}
}

// printColumnRecorded is true when the watch started before the column ended
// and the column center is outside every known gap. A print is handled by the caller.
func printColumnRecorded(col time.Time, step time.Duration, cov LiquidationCoverage) bool {
	if cov.FirstWatch.IsZero() || step <= 0 {
		return false
	}
	end := col.Add(step / 2)
	if !cov.FirstWatch.Before(end) {
		return false
	}
	return !printGapContains(cov.Gaps, col)
}

// printGapContains reports whether at sits in a gap. An open gap (To zero) runs from From onward.
func printGapContains(gaps []LiquidationGap, at time.Time) bool {
	for _, g := range gaps {
		if g.From.IsZero() || at.Before(g.From) {
			continue
		}
		if g.To.IsZero() || at.Before(g.To) {
			return true
		}
	}
	return false
}

func emptyMeasuredReview() HuntHeatmapReview {
	return HuntHeatmapReview{
		Binance:  HuntHeatmapReviewVenue{Exchange: "binance"},
		Bybit:    HuntHeatmapReviewVenue{Exchange: "bybit"},
		OKX:      HuntHeatmapReviewVenue{Exchange: "okx"},
		Combined: HuntHeatmapReviewVenue{Exchange: "combined"},
	}
}
