package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// CandleFootprintMaxRows is how many price rows one candle keeps.
	// Wider storage bins roll up so the chart can paint them.
	CandleFootprintMaxRows = 24
	// CandleFootprintMaxCandles is the most candles one read returns.
	// A wider view keeps the newest candles and says it was shortened.
	CandleFootprintMaxCandles = 400
	// CandleFootprintDefaultCandles is the lookback when no range is set.
	CandleFootprintDefaultCandles = 120
	// CandleFootprintImbalance is the buy/sell multiple that marks a row.
	CandleFootprintImbalance = 3
	// CandleFootprintOneSidedShare is the smallest share of the candle
	// that a one-sided row needs before it counts as an imbalance.
	CandleFootprintOneSidedShare = 0.01
)

// CandleFootprintRow is buy versus sell at one price inside a candle.
type CandleFootprintRow struct {
	Price     float64
	High      float64
	Buy       float64
	Sell      float64
	BuyN      int
	SellN     int
	Imbalance string // buy | sell | empty
	IsPoc     bool
}

// CandleFootprint is one candle's price ladder.
type CandleFootprint struct {
	Open  time.Time
	Buy   float64
	Sell  float64
	BuyN  int
	SellN int
	POC   float64
	Rows  []CandleFootprintRow
}

// CandleFootprintBook is one venue, or the combined book.
type CandleFootprintBook struct {
	Exchange Exchange
	Candles  []CandleFootprint
}

// CandleFootprintSource is one venue's stored minutes.
// Present is true when that venue was recorded for the range, including
// a quiet tape whose live minute was covered and had no prints.
type CandleFootprintSource struct {
	Exchange Exchange
	Cells    []FlowAtPricesCell
	Present  bool
}

// CandleFootprintReport is buy and sell inside each candle.
type CandleFootprintReport struct {
	Symbol    string
	Exchange  string
	Interval  string
	Step      time.Duration
	From      time.Time
	To        time.Time
	AsOf      time.Time
	Venues    []CandleFootprintBook
	Combined  *CandleFootprintBook
	Definite  bool
	Missing   []Exchange
	Truncated bool
	Summary   string
	Note      string
}

// ParseCandleFootprintInterval accepts the chart intervals (1m, 15m, 1h, 4h, 1d, 1w, 1M).
func ParseCandleFootprintInterval(raw string) (string, time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, fmt.Errorf("%w: interval is required", ErrInvalidArgument)
	}
	if raw == "1M" {
		return raw, 30 * 24 * time.Hour, nil
	}
	i := 0
	for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
		i++
	}
	if i == 0 || i == len(raw) {
		return "", 0, fmt.Errorf("%w: interval must look like 1m, 1h, 4h, 1d, or 1w", ErrInvalidArgument)
	}
	n, err := strconv.Atoi(raw[:i])
	if err != nil || n <= 0 {
		return "", 0, fmt.Errorf("%w: interval must look like 1m, 1h, 4h, 1d, or 1w", ErrInvalidArgument)
	}
	var unit time.Duration
	switch raw[i:] {
	case "m":
		unit = time.Minute
	case "h":
		unit = time.Hour
	case "d":
		unit = 24 * time.Hour
	case "w":
		unit = 7 * 24 * time.Hour
	default:
		return "", 0, fmt.Errorf("%w: interval must look like 1m, 1h, 4h, 1d, or 1w", ErrInvalidArgument)
	}
	step := time.Duration(n) * unit
	if step <= 0 || step > MaxFlowAtPricesRange {
		return "", 0, fmt.Errorf("%w: interval must be from 1m through 30d", ErrInvalidArgument)
	}
	return raw, step, nil
}

// CandleFootprintOpen is the candle open that contains t.
// Steps that divide a UTC day align to midnight. A week aligns to Monday.
// 1M is a 30-day block, matching the chart interval.
func CandleFootprintOpen(t time.Time, step time.Duration) time.Time {
	t = t.UTC()
	if step <= 0 {
		return t
	}
	if step == 7*24*time.Hour {
		midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		delta := (int(midnight.Weekday()) - int(time.Monday) + 7) % 7
		return midnight.AddDate(0, 0, -delta)
	}
	day := 24 * time.Hour
	if step <= day && day%step == 0 {
		midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return midnight.Add(t.Sub(midnight).Truncate(step))
	}
	sec := int64(step.Seconds())
	if sec <= 0 {
		return t
	}
	return time.Unix((t.Unix()/sec)*sec, 0).UTC()
}

// ResolveCandleFootprintRange aligns a request to candle boundaries.
// An empty range is the newest default candles. A span past 30 days or
// past CandleFootprintMaxCandles keeps the newest part and reports truncated.
func ResolveCandleFootprintRange(step time.Duration, start, end *time.Time, now time.Time) (from, to time.Time, truncated bool, err error) {
	if step <= 0 {
		return time.Time{}, time.Time{}, false, fmt.Errorf("%w: interval is required", ErrInvalidArgument)
	}
	now = now.UTC()
	if (start == nil) != (end == nil) {
		return time.Time{}, time.Time{}, false, fmt.Errorf("%w: startTime and endTime are set together", ErrInvalidArgument)
	}
	if start == nil {
		to = now
		from = now.Add(-time.Duration(CandleFootprintDefaultCandles) * step)
	} else {
		from = start.UTC()
		to = end.UTC()
	}
	if !to.After(from) {
		return time.Time{}, time.Time{}, false, fmt.Errorf("%w: endTime must be after startTime", ErrInvalidArgument)
	}
	if to.After(now) {
		to = now
	}
	if !to.After(from) {
		return time.Time{}, time.Time{}, false, fmt.Errorf("%w: endTime must be after startTime", ErrInvalidArgument)
	}
	from = CandleFootprintOpen(from, step)
	last := CandleFootprintOpen(to.Add(-time.Nanosecond), step)
	to = last.Add(step)
	if !to.After(from) {
		to = from.Add(step)
	}
	if to.Sub(from) > MaxFlowAtPricesRange {
		from = CandleFootprintOpen(to.Add(-MaxFlowAtPricesRange), step)
		truncated = true
	}
	if to.Sub(from) > time.Duration(CandleFootprintMaxCandles)*step {
		from = to.Add(-time.Duration(CandleFootprintMaxCandles) * step)
		truncated = true
	}
	return from, to, truncated, nil
}

// BuildCandleFootprints groups stored minute cells into candle ladders.
// Combined is set only when every expected venue was present. A missing
// venue is named and is not treated as zero.
func BuildCandleFootprints(symbol, requested, interval string, step time.Duration, from, to, asOf time.Time, expected []Exchange, sources []CandleFootprintSource, truncated bool) CandleFootprintReport {
	symbol = NormalizeLiquidationSymbol(symbol)
	out := CandleFootprintReport{
		Symbol:    symbol,
		Exchange:  requested,
		Interval:  interval,
		Step:      step,
		From:      from.UTC(),
		To:        to.UTC(),
		AsOf:      asOf.UTC(),
		Venues:    []CandleFootprintBook{},
		Truncated: truncated,
		Note:      candleFootprintNote,
	}
	byEx := map[Exchange]CandleFootprintSource{}
	for _, src := range sources {
		byEx[src.Exchange] = src
	}
	var missing []Exchange
	present := make([]CandleFootprintSource, 0, len(expected))
	for _, ex := range expected {
		src, ok := byEx[ex]
		if !ok || !src.Present {
			missing = append(missing, ex)
			continue
		}
		present = append(present, src)
		book := CandleFootprintBook{Exchange: ex, Candles: footprintCandles(src.Cells, from, to, step)}
		out.Venues = append(out.Venues, book)
	}
	out.Missing = missing
	out.Definite = len(expected) > 0 && len(missing) == 0
	if requested == "all" && out.Definite {
		merged := mergeFootprintCells(present)
		combined := CandleFootprintBook{Exchange: "all", Candles: footprintCandles(merged, from, to, step)}
		out.Combined = &combined
	}
	out.Summary = explainCandleFootprint(out)
	return out
}

const candleFootprintNote = "Buy and sell inside each candle are aggressive futures prints at those prices, from stored minutes (about 30 days) plus the current minute when the live tape covers it. Binance minutes for the liquid USDT list are filled from public aggTrades, about 7 days. Bybit and OKX stay on the minutes this process recorded. Up to 24 price rows are kept per candle. A row is marked when one side is at least 3× the other. Combined is filled only when every venue was recorded for this range. A missing venue is named and is not treated as zero. Informational only — not financial advice."

func explainCandleFootprint(r CandleFootprintReport) string {
	if len(r.Missing) > 0 && r.Combined == nil {
		names := make([]string, len(r.Missing))
		for i, ex := range r.Missing {
			names[i] = string(ex)
		}
		return fmt.Sprintf("No combined candle footprint: %s has no stored minutes in this range.", strings.Join(names, ", "))
	}
	book := r.Combined
	if book == nil && len(r.Venues) == 1 {
		book = &r.Venues[0]
	}
	if book == nil || len(book.Candles) == 0 {
		return "No stored flow minutes in this range."
	}
	var buy, sell float64
	for _, c := range book.Candles {
		buy += c.Buy
		sell += c.Sell
	}
	return fmt.Sprintf("%s %s: %d candles, buy %s, sell %s.", r.Interval, r.Symbol, len(book.Candles), formatQty(buy), formatQty(sell))
}

type footprintBin struct {
	price float64
	buy   float64
	sell  float64
	buyN  int
	sellN int
}

func footprintCandles(cells []FlowAtPricesCell, from, to time.Time, step time.Duration) []CandleFootprint {
	if step <= 0 {
		return []CandleFootprint{}
	}
	grouped := map[int64][]footprintBin{}
	for _, c := range cells {
		if c.Price <= 0 || (c.Buy <= 0 && c.Sell <= 0) || c.Minute.IsZero() {
			continue
		}
		minute := c.Minute.UTC()
		if !from.IsZero() && minute.Before(from) {
			continue
		}
		if !to.IsZero() && !minute.Before(to) {
			continue
		}
		open := CandleFootprintOpen(minute, step)
		if open.Before(from) || !open.Before(to) {
			continue
		}
		key := open.Unix()
		bins := grouped[key]
		bins = addFootprintBin(bins, c)
		grouped[key] = bins
	}
	if len(grouped) == 0 {
		return []CandleFootprint{}
	}
	keys := make([]int64, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make([]CandleFootprint, 0, len(keys))
	for _, k := range keys {
		rows := rollFootprintRows(grouped[k])
		if len(rows) == 0 {
			continue
		}
		candle := CandleFootprint{Open: time.Unix(k, 0).UTC(), Rows: rows}
		var best float64
		bestAt := -1
		for i, row := range rows {
			candle.Buy += row.Buy
			candle.Sell += row.Sell
			candle.BuyN += row.BuyN
			candle.SellN += row.SellN
			vol := row.Buy + row.Sell
			if bestAt < 0 || vol > best || (vol == best && row.Price < rows[bestAt].Price) {
				best = vol
				bestAt = i
			}
		}
		if bestAt >= 0 {
			candle.Rows[bestAt].IsPoc = true
			candle.POC = candle.Rows[bestAt].Price
		}
		vol := candle.Buy + candle.Sell
		for i := range candle.Rows {
			candle.Rows[i].Imbalance = footprintImbalance(candle.Rows[i].Buy, candle.Rows[i].Sell, vol)
		}
		out = append(out, candle)
	}
	return out
}

func addFootprintBin(bins []footprintBin, c FlowAtPricesCell) []footprintBin {
	px := priceE8(c.Price)
	for i := range bins {
		if priceE8(bins[i].price) == px {
			bins[i].buy += c.Buy
			bins[i].sell += c.Sell
			bins[i].buyN += c.BuyN
			bins[i].sellN += c.SellN
			return bins
		}
	}
	return append(bins, footprintBin{price: c.Price, buy: c.Buy, sell: c.Sell, buyN: c.BuyN, sellN: c.SellN})
}

func priceE8(price float64) int64 {
	return int64(math.Round(price * 1e8))
}

func rollFootprintRows(bins []footprintBin) []CandleFootprintRow {
	if len(bins) == 0 {
		return nil
	}
	sort.Slice(bins, func(i, j int) bool { return bins[i].price < bins[j].price })
	groups := groupFootprintBins(bins, CandleFootprintMaxRows)
	rows := make([]CandleFootprintRow, 0, len(groups))
	for i, group := range groups {
		if len(group) == 0 {
			continue
		}
		row := CandleFootprintRow{Price: group[0].price}
		last := group[len(group)-1].price
		next := 0.0
		if i+1 < len(groups) && len(groups[i+1]) > 0 {
			next = groups[i+1][0].price
		}
		row.High = footprintBinHigh(last, next)
		for _, bin := range group {
			row.Buy += bin.buy
			row.Sell += bin.sell
			row.BuyN += bin.buyN
			row.SellN += bin.sellN
		}
		if row.Buy+row.Sell <= 0 || row.High <= row.Price {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func groupFootprintBins(bins []footprintBin, max int) [][]footprintBin {
	if max <= 0 || len(bins) <= max {
		out := make([][]footprintBin, len(bins))
		for i := range bins {
			out[i] = []footprintBin{bins[i]}
		}
		return out
	}
	size := (len(bins) + max - 1) / max
	out := make([][]footprintBin, 0, (len(bins)+size-1)/size)
	for i := 0; i < len(bins); i += size {
		end := i + size
		if end > len(bins) {
			end = len(bins)
		}
		out = append(out, bins[i:end])
	}
	return out
}

func footprintBinHigh(price, next float64) float64 {
	tick := FlowHistoryTick(price)
	if tick <= 0 {
		tick = price * 0.0005
		if tick <= 0 {
			tick = 1e-8
		}
	}
	high := price + tick
	if next > price && next < high {
		return next
	}
	return high
}

func footprintImbalance(buy, sell, candleVol float64) string {
	tot := buy + sell
	if tot <= 0 || candleVol <= 0 {
		return ""
	}
	side := TakerSideBuy
	hi, lo := buy, sell
	if sell > buy {
		side = TakerSideSell
		hi, lo = sell, buy
	}
	if lo <= 0 {
		if tot/candleVol >= CandleFootprintOneSidedShare {
			return side
		}
		return ""
	}
	if hi/lo >= CandleFootprintImbalance {
		return side
	}
	return ""
}

func mergeFootprintCells(sources []CandleFootprintSource) []FlowAtPricesCell {
	var out []FlowAtPricesCell
	for _, src := range sources {
		out = append(out, src.Cells...)
	}
	return out
}
