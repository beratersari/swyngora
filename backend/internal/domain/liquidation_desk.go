package domain

import (
	"context"
	"sort"
	"time"
)

// LiquidationSymbolSum is stored long/short notional for one venue and coin.
type LiquidationSymbolSum struct {
	Exchange Exchange
	Symbol   string
	Long     float64
	Short    float64
	Count    int
}

// LiquidationSymbolScan is one overview window read from the print archive.
type LiquidationSymbolScan struct {
	Rows    []LiquidationSymbolSum
	Latest  map[Exchange]time.Time
	Biggest *LiquidationEvent
}

// LiquidationBucket is stored long/short notional in one time bar, all coins.
type LiquidationBucket struct {
	Start time.Time
	Long  float64
	Short float64
	Count int
}

// LiquidationBucketScan is the all-coin chart read from the print archive.
type LiquidationBucketScan struct {
	Buckets []LiquidationBucket
	Latest  map[Exchange]time.Time
}

// LiquidationDeskArchive rolls stored force-order prints across coins.
// The overview and the all-coin chart use it so a 24-hour memory cap, or a
// restart, does not hide prints SQLite already kept.
type LiquidationDeskArchive interface {
	SumLiquidationSymbols(ctx context.Context, from, to time.Time, exchanges []Exchange) (LiquidationSymbolScan, error)
	BucketLiquidations(ctx context.Context, from, to time.Time, step time.Duration, exchanges []Exchange) (LiquidationBucketScan, error)
}

// LiquidationBarGrid is the all-coin chart axis. The step is a whole number of
// milliseconds so SQLite and Go land a print in the same bar. At most
// maxLiquidationTimeBars columns; a 30-day window uses wider bars.
func LiquidationBarGrid(from, to time.Time, step time.Duration) (origin, until time.Time, stepOut time.Duration, n int) {
	from = from.UTC()
	to = to.UTC()
	if !to.After(from) {
		to = from.Add(time.Hour)
	}
	if step <= 0 {
		step = LiquidationTimeBarStep(to.Sub(from))
	}
	n = int(to.Sub(from) / step)
	if n < 1 {
		n = 1
	}
	if n > maxLiquidationTimeBars {
		n = maxLiquidationTimeBars
	}
	spanMS := to.Sub(from).Milliseconds()
	stepMS := spanMS / int64(n)
	if stepMS < 1 {
		stepMS = 1
	}
	stepOut = time.Duration(stepMS) * time.Millisecond
	return from, from.Add(time.Duration(n) * stepOut), stepOut, n
}

// MergeLiquidationSymbolTail adds prints that are newer than the newest stored
// row for their venue. A print at or before that row is already in the sum.
func MergeLiquidationSymbolTail(rows []LiquidationSymbolSum, latest map[Exchange]time.Time, events []LiquidationEvent, from, to time.Time) []LiquidationSymbolSum {
	out := append([]LiquidationSymbolSum(nil), rows...)
	idx := make(map[string]int, len(out))
	for i, row := range out {
		idx[liquidationSumKey(row.Exchange, row.Symbol)] = i
	}
	for _, e := range events {
		if !liquidationEventOpen(e, from, to) || !liquidationEventAfterLatest(e, latest) {
			continue
		}
		sym := NormalizeLiquidationSymbol(e.Symbol)
		longN, shortN := liquidationSideNotional(e)
		key := liquidationSumKey(e.Exchange, sym)
		if i, ok := idx[key]; ok {
			out[i].Long += longN
			out[i].Short += shortN
			out[i].Count++
			continue
		}
		idx[key] = len(out)
		out = append(out, LiquidationSymbolSum{
			Exchange: e.Exchange, Symbol: sym, Long: longN, Short: shortN, Count: 1,
		})
	}
	return out
}

// MergeLiquidationBiggest keeps the larger of the stored max and a newer memory print.
func MergeLiquidationBiggest(stored *LiquidationEvent, events []LiquidationEvent, latest map[Exchange]time.Time, from, to time.Time) *LiquidationEvent {
	var best *LiquidationEvent
	if stored != nil && stored.Notional > 0 {
		cp := *stored
		best = &cp
	}
	for _, e := range events {
		if e.Notional <= 0 || !liquidationEventOpen(e, from, to) || !liquidationEventAfterLatest(e, latest) {
			continue
		}
		if best == nil || e.Notional > best.Notional {
			cp := e
			best = &cp
		}
	}
	return best
}

// LiquidationWindowFromSums is one overview card. Complete still means the
// live clock covered the window. The notionals are the stored prints.
func LiquidationWindowFromSums(rows []LiquidationSymbolSum, windowID string, coverage time.Duration, biggest *LiquidationEvent) LiquidationWindowTotals {
	out := LiquidationWindowTotals{Window: windowID}
	if coverage < 0 {
		coverage = 0
	}
	dur := LiquidationWindowDuration(windowID)
	if dur > 0 {
		if coverage >= dur {
			out.Complete = true
			out.CoverageSeconds = int64(dur.Seconds())
		} else {
			out.CoverageSeconds = int64(coverage.Seconds())
		}
	}
	var longN, shortN float64
	for _, row := range rows {
		longN += row.Long
		shortN += row.Short
		out.Count += row.Count
	}
	out.LongNotional = formatQty(longN)
	out.ShortNotional = formatQty(shortN)
	out.TotalNotional = formatQty(longN + shortN)
	if biggest != nil && biggest.Notional > 0 {
		hit := LiquidationHit{
			Exchange: string(biggest.Exchange),
			Side:     biggest.Side,
			Price:    formatFixed(biggest.Price, decimalsForStep(biggest.Price/10000)+1),
			Quantity: formatQty(biggest.Quantity),
			Notional: formatQty(biggest.Notional),
			Time:     biggest.Time.UTC(),
		}
		out.Biggest = &hit
	}
	return out
}

// LiquidationCoinsFromSums ranks coins by total notional. Venues for one coin are added.
func LiquidationCoinsFromSums(rows []LiquidationSymbolSum, limit int) []LiquidationCoinTile {
	limit = ClampLiquidationOverviewLimit(limit)
	type acc struct {
		sym   string
		long  float64
		short float64
		count int
	}
	by := map[string]*acc{}
	order := make([]string, 0, len(rows))
	for _, row := range rows {
		sym := NormalizeLiquidationSymbol(row.Symbol)
		if sym == "" || row.Count == 0 && row.Long == 0 && row.Short == 0 {
			continue
		}
		a := by[sym]
		if a == nil {
			a = &acc{sym: sym}
			by[sym] = a
			order = append(order, sym)
		}
		a.long += row.Long
		a.short += row.Short
		a.count += row.Count
	}
	type ranked struct {
		tile  LiquidationCoinTile
		total float64
	}
	list := make([]ranked, 0, len(order))
	for _, sym := range order {
		a := by[sym]
		if a.count == 0 {
			continue
		}
		total := a.long + a.short
		list = append(list, ranked{
			total: total,
			tile: LiquidationCoinTile{
				Symbol:        sym,
				Base:          LiquidationBaseAsset(sym),
				LongNotional:  formatQty(a.long),
				ShortNotional: formatQty(a.short),
				TotalNotional: formatQty(total),
				Count:         a.count,
			},
		})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].total != list[j].total {
			return list[i].total > list[j].total
		}
		return list[i].tile.Symbol < list[j].tile.Symbol
	})
	if len(list) > limit {
		list = list[:limit]
	}
	out := make([]LiquidationCoinTile, 0, len(list))
	for _, row := range list {
		out = append(out, row.tile)
	}
	return out
}

// LiquidationTimeBarsFromBuckets fills the chart grid. Stored buckets and
// newer memory prints share it. Bars before the first print stay zero.
func LiquidationTimeBarsFromBuckets(scan LiquidationBucketScan, events []LiquidationEvent, origin time.Time, step time.Duration, n int) []LiquidationTimeBar {
	if n < 1 {
		n = 1
	}
	bars := make([]LiquidationTimeBar, n)
	longs := make([]float64, n)
	shorts := make([]float64, n)
	origin = origin.UTC()
	for i := 0; i < n; i++ {
		bars[i].Time = origin.Add(time.Duration(i) * step)
	}
	add := func(t time.Time, longN, shortN float64, count int) {
		idx := liquidationBarIndex(t, origin, step, n)
		if idx < 0 {
			return
		}
		longs[idx] += longN
		shorts[idx] += shortN
		bars[idx].Count += count
	}
	for _, b := range scan.Buckets {
		add(b.Start, b.Long, b.Short, b.Count)
	}
	until := origin.Add(time.Duration(n) * step)
	for _, e := range events {
		if !liquidationEventOpen(e, origin, until) || !liquidationEventAfterLatest(e, scan.Latest) {
			continue
		}
		longN, shortN := liquidationSideNotional(e)
		add(e.Time, longN, shortN, 1)
	}
	for i := range bars {
		bars[i].LongNotional = formatQty(longs[i])
		bars[i].ShortNotional = formatQty(shorts[i])
		bars[i].TotalNotional = formatQty(longs[i] + shorts[i])
	}
	return bars
}

func liquidationBarIndex(t, origin time.Time, step time.Duration, n int) int {
	stepMS := step.Milliseconds()
	if stepMS <= 0 || n < 1 {
		return -1
	}
	idx := int((t.UTC().UnixMilli() - origin.UTC().UnixMilli()) / stepMS)
	if idx < 0 || idx >= n {
		return -1
	}
	return idx
}

func liquidationEventOpen(e LiquidationEvent, from, to time.Time) bool {
	if e.Time.IsZero() || e.Notional <= 0 || e.Symbol == "" {
		return false
	}
	if !from.IsZero() && e.Time.Before(from) {
		return false
	}
	if !to.IsZero() && !e.Time.Before(to) {
		return false
	}
	return true
}

func liquidationEventAfterLatest(e LiquidationEvent, latest map[Exchange]time.Time) bool {
	if len(latest) == 0 {
		return true
	}
	t, ok := latest[e.Exchange]
	if !ok || t.IsZero() {
		return true
	}
	// SQLite stores millisecond times. A memory print from that same
	// millisecond is already in the sum; only a later millisecond is new.
	return e.Time.UnixMilli() > t.UnixMilli()
}

func liquidationSideNotional(e LiquidationEvent) (longN, shortN float64) {
	if e.Side == LiquidationSideShort {
		return 0, e.Notional
	}
	return e.Notional, 0
}

func liquidationSumKey(ex Exchange, symbol string) string {
	return string(ex) + "\x00" + NormalizeLiquidationSymbol(symbol)
}
