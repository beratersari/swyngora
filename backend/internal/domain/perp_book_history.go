package domain

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// PerpBookHistoryMaxSpan is how far a coin's stored perp book can be read.
	PerpBookHistoryMaxSpan = 30 * 24 * time.Hour
	// PerpDepthChartSpan is how far the Markets total line is drawn.
	// The 6-hour minute tape used for jumps stays PerpDepthTotalChartSpan.
	PerpDepthChartSpan = 30 * 24 * time.Hour

	perpBookHistoryGapCap = 24
)

// PerpBookStoreBands are the ±% windows kept for each liquid perpetual.
var PerpBookStoreBands = []float64{0.1, 0.5, 1}

// PerpBookHistoryStep is the column width for a stored window.
// A short view stays on minutes. Longer views fold minutes so a 30-day
// read is a few hundred points, not one row per minute.
func PerpBookHistoryStep(span time.Duration) time.Duration {
	switch {
	case span <= 6*time.Hour:
		return time.Minute
	case span <= 24*time.Hour:
		return 5 * time.Minute
	case span <= 7*24*time.Hour:
		return 30 * time.Minute
	default:
		return 2 * time.Hour
	}
}

// PerpBookBandSample is one venue's bid and ask inside one ±% at one minute.
type PerpBookBandSample struct {
	Time     time.Time
	Exchange string
	Symbol   string
	RangePct float64
	Bid      float64
	Ask      float64
	Mid      float64
	// CoverBid and CoverAsk are how far the snapshot reached, in percent of mid.
	CoverBid float64
	CoverAsk float64
}

// PerpBookGap is a minute that was not stored because a venue was missing or old.
type PerpBookGap struct {
	Time    time.Time
	Symbol  string
	Missing []string
	Stale   []string
}

// PerpBookHistoryBand is bid below mid and ask above mid at one distance.
type PerpBookHistoryBand struct {
	RangePct float64
	Bid      float64
	Ask      float64
}

// PerpBookHistoryColumn is one stored step of a coin's perp book.
type PerpBookHistoryColumn struct {
	Time     time.Time
	Mid      float64
	CoverBid float64
	CoverAsk float64
	Bands    []PerpBookHistoryBand
}

// PerpBookHistory is the stored perp book for one coin.
type PerpBookHistory struct {
	Symbol   string
	Exchange string
	From     time.Time
	To       time.Time
	Step     time.Duration
	Columns  []PerpBookHistoryColumn
	Gaps     []PerpBookGap
	Note     string
}

const perpBookHistoryNote = "Resting perpetual bid and ask within 0.1%, 0.5%, and 1% of each venue's own mid. A minute is stored only when Binance, Bybit, and OKX are all fresh. The shade stops where the snapshot runs out, so a band wider than the book is not drawn past that edge. The liquid USDT list is sampled while the server runs and kept for 30 days. Informational only — not financial advice."

// BuildPerpBookHistory folds stored band rows into one column per step.
// A column keeps the newest sample of each band inside that step.
func BuildPerpBookHistory(symbol, exchange string, from, to time.Time, samples []PerpBookBandSample, gaps []PerpBookGap) *PerpBookHistory {
	symbol = NormalizeLiquidationSymbol(symbol)
	if exchange == "" {
		exchange = "all"
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	to = to.UTC()
	if from.IsZero() || !from.Before(to) {
		from = to.Add(-24 * time.Hour)
	}
	from = from.UTC()
	if to.Sub(from) > PerpBookHistoryMaxSpan {
		from = to.Add(-PerpBookHistoryMaxSpan)
	}
	step := PerpBookHistoryStep(to.Sub(from))
	type bandPick struct {
		at       time.Time
		bid      float64
		ask      float64
		mid      float64
		pct      float64
		coverBid float64
		coverAsk float64
	}
	byBucket := map[int64]map[int]bandPick{}
	for _, row := range samples {
		if row.Time.IsZero() || row.Bid+row.Ask <= 0 {
			continue
		}
		t := row.Time.UTC()
		if t.Before(from) || t.After(to.Add(time.Minute)) {
			continue
		}
		bucket := t.UnixMilli() / step.Milliseconds() * step.Milliseconds()
		pct := canonicalPerpBookBand(row.RangePct)
		if pct == 0 {
			continue
		}
		key := int(math.Round(pct * 100))
		bands := byBucket[bucket]
		if bands == nil {
			bands = map[int]bandPick{}
			byBucket[bucket] = bands
		}
		prev, ok := bands[key]
		if ok && !t.After(prev.at) {
			continue
		}
		bands[key] = bandPick{at: t, bid: row.Bid, ask: row.Ask, mid: row.Mid, pct: pct, coverBid: row.CoverBid, coverAsk: row.CoverAsk}
	}
	keys := make([]int64, 0, len(byBucket))
	for k := range byBucket {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	cols := make([]PerpBookHistoryColumn, 0, len(keys))
	for _, bucket := range keys {
		bands := byBucket[bucket]
		order := make([]int, 0, len(bands))
		for k := range bands {
			order = append(order, k)
		}
		sort.Ints(order)
		col := PerpBookHistoryColumn{Time: time.UnixMilli(bucket).UTC(), Bands: make([]PerpBookHistoryBand, 0, len(order))}
		for _, k := range order {
			p := bands[k]
			col.Bands = append(col.Bands, PerpBookHistoryBand{RangePct: p.pct, Bid: p.bid, Ask: p.ask})
			if p.mid > 0 && (col.Mid == 0 || k == 100) {
				col.Mid = p.mid
			}
			if p.coverBid > col.CoverBid {
				col.CoverBid = p.coverBid
			}
			if p.coverAsk > col.CoverAsk {
				col.CoverAsk = p.coverAsk
			}
		}
		if col.Mid <= 0 || len(col.Bands) == 0 {
			continue
		}
		cols = append(cols, col)
	}
	keptGaps := make([]PerpBookGap, 0)
	for _, g := range gaps {
		if g.Time.IsZero() {
			continue
		}
		t := g.Time.UTC()
		if t.Before(from) || t.After(to.Add(time.Minute)) {
			continue
		}
		if len(g.Missing) == 0 && len(g.Stale) == 0 {
			continue
		}
		keptGaps = append(keptGaps, PerpBookGap{
			Time: t, Symbol: symbol,
			Missing: append([]string{}, g.Missing...),
			Stale:   append([]string{}, g.Stale...),
		})
	}
	sort.Slice(keptGaps, func(i, j int) bool { return keptGaps[i].Time.Before(keptGaps[j].Time) })
	if len(keptGaps) > perpBookHistoryGapCap {
		keptGaps = keptGaps[len(keptGaps)-perpBookHistoryGapCap:]
	}
	return &PerpBookHistory{
		Symbol: symbol, Exchange: exchange, From: from, To: to, Step: step,
		Columns: cols, Gaps: keptGaps, Note: perpBookHistoryNote,
	}
}

func canonicalPerpBookBand(pct float64) float64 {
	key := int(math.Round(pct * 100))
	switch key {
	case 10:
		return 0.1
	case 50:
		return 0.5
	case 100:
		return 1
	default:
		return 0
	}
}

// AttachPerpDepthChart keeps the bucketed total-book line inside the stored window.
// It does not recompute jumps. Those stay on the 6-hour minute tape.
func AttachPerpDepthChart(live *PerpDepthOverview, points []PerpDepthTotalPoint) {
	if live == nil {
		return
	}
	asOf := live.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	from := asOf.Add(-PerpDepthChartSpan)
	out := make([]PerpDepthTotalPoint, 0, len(points))
	for _, p := range points {
		if p.Time.IsZero() || p.Total <= 0 {
			continue
		}
		t := p.Time.UTC()
		if t.Before(from) || t.After(asOf.Add(time.Minute)) {
			continue
		}
		out = append(out, PerpDepthTotalPoint{Time: t, Total: round4(p.Total), Price: p.Price})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	live.Chart = out
}

// AttachPerpDepthChartPrices fills each chart bucket with the BTC close at or before that time.
func AttachPerpDepthChartPrices(live *PerpDepthOverview, candles []Candle) {
	if live == nil || len(live.Chart) == 0 || len(candles) == 0 {
		return
	}
	type px struct {
		at    time.Time
		close float64
	}
	points := make([]px, 0, len(candles))
	for _, c := range candles {
		t := c.OpenTime
		if t.IsZero() {
			t = c.CloseTime
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(c.Close), 64)
		if err != nil || v <= 0 || t.IsZero() {
			continue
		}
		points = append(points, px{at: t.UTC(), close: v})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].at.Before(points[j].at) })
	if len(points) == 0 {
		return
	}
	j := 0
	last := 0.0
	for i := range live.Chart {
		for j < len(points) && !points[j].at.After(live.Chart[i].Time) {
			last = points[j].close
			j++
		}
		if last > 0 {
			live.Chart[i].Price = round4(last)
		}
	}
}
