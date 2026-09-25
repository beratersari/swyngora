package domain

import (
	"sort"
	"strconv"
	"time"
)

type heatCell struct {
	notional float64
	wall     bool
}

// BuildHeatmapFromBookHistory paints stored minute ladders into heatmap columns.
// Brief walls and other size moves that the minute ladder missed raise the
// notional at that price for the bucket they happened in.
func BuildHeatmapFromBookHistory(ex Exchange, symbol string, window time.Duration, now time.Time, snaps []BookHistorySnapshot, flashes []BookHistoryFlash, moves []BookHistoryMove) OrderBookHeatmap {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	windowSec := ClampHeatmapWindowSeconds(int(window / time.Second))
	if window <= 0 || !HeatmapUsesHistory(windowSec) {
		window = time.Duration(windowSec) * time.Second
	}
	bucket := heatmapHistoryBucket(window)
	from := now.Add(-window)
	out := OrderBookHeatmap{
		Exchange:      ex,
		Symbol:        symbol,
		WindowSeconds: windowSec,
		SampleEveryMs: int(bucket / time.Millisecond),
		Source:        "history",
	}
	type col struct {
		t    time.Time
		mid  float64
		bids map[string]*heatCell
		asks map[string]*heatCell
	}
	cols := map[int64]*col{}
	ensure := func(at time.Time) *col {
		if at.Before(from) || at.After(now.Add(time.Minute)) {
			return nil
		}
		key := at.UTC().Truncate(bucket).UnixMilli()
		c := cols[key]
		if c == nil {
			c = &col{t: time.UnixMilli(key).UTC(), bids: map[string]*heatCell{}, asks: map[string]*heatCell{}}
			cols[key] = c
		}
		return c
	}
	put := func(side map[string]*heatCell, price, notional float64, wall bool) {
		if price <= 0 || notional <= 0 {
			return
		}
		k := strconv.FormatFloat(price, 'f', 8, 64)
		cur := side[k]
		if cur == nil {
			side[k] = &heatCell{notional: notional, wall: wall}
			return
		}
		if notional > cur.notional {
			cur.notional = notional
		}
		if wall {
			cur.wall = true
		}
	}
	for _, snap := range snaps {
		c := ensure(snap.SampledAt)
		if c == nil {
			continue
		}
		if snap.Mid > 0 {
			c.mid = snap.Mid
		}
		if out.GroupSize == "" && snap.GroupSize > 0 {
			out.GroupSize = strconv.FormatFloat(snap.GroupSize, 'f', -1, 64)
		}
		for _, lv := range snap.Bids {
			put(c.bids, lv.Price, lv.Notional, lv.Wall)
		}
		for _, lv := range snap.Asks {
			put(c.asks, lv.Price, lv.Notional, lv.Wall)
		}
	}
	for _, f := range flashes {
		c := ensure(f.SeenFrom)
		if c == nil {
			continue
		}
		side := c.bids
		if f.Side == "ask" {
			side = c.asks
		}
		put(side, f.Price, f.Notional, true)
	}
	for _, mv := range moves {
		c := ensure(mv.SeenFrom)
		if c == nil {
			continue
		}
		n := mv.PeakNotional
		if mv.FromNotional > n {
			n = mv.FromNotional
		}
		if mv.ToNotional > n {
			n = mv.ToNotional
		}
		side := c.bids
		if mv.Side == "ask" {
			side = c.asks
		}
		put(side, mv.Price, n, false)
	}
	if len(cols) == 0 {
		return out
	}
	keys := make([]int64, 0, len(cols))
	for k := range cols {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out.Columns = make([]HeatmapColumn, 0, len(keys))
	for _, k := range keys {
		c := cols[k]
		col := HeatmapColumn{Time: c.t, Bids: heatCells(c.bids, true), Asks: heatCells(c.asks, false)}
		if c.mid > 0 {
			col.Mid = strconv.FormatFloat(c.mid, 'f', -1, 64)
		}
		out.Columns = append(out.Columns, col)
	}
	out.From = out.Columns[0].Time
	out.To = out.Columns[len(out.Columns)-1].Time
	return out
}

func heatmapHistoryBucket(window time.Duration) time.Duration {
	if window <= 0 {
		return time.Minute
	}
	if window/time.Minute <= HeatmapHistoryMaxColumns {
		return time.Minute
	}
	mins := int((window + time.Duration(HeatmapHistoryMaxColumns-1)*time.Minute) / (time.Duration(HeatmapHistoryMaxColumns) * time.Minute))
	if mins < 1 {
		mins = 1
	}
	return time.Duration(mins) * time.Minute
}

func heatCells(side map[string]*heatCell, bids bool) []HeatmapLevel {
	if len(side) == 0 {
		return nil
	}
	out := make([]HeatmapLevel, 0, len(side))
	for price, c := range side {
		out = append(out, HeatmapLevel{
			Price:    trimHeatPrice(price),
			Notional: strconv.FormatFloat(c.notional, 'f', -1, 64),
			IsWall:   c.wall,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		pi, _ := strconv.ParseFloat(out[i].Price, 64)
		pj, _ := strconv.ParseFloat(out[j].Price, 64)
		if bids {
			return pi > pj
		}
		return pi < pj
	})
	return out
}

func trimHeatPrice(raw string) string {
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}
