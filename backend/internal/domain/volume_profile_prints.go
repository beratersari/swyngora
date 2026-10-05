package domain

import (
	"sort"
	"time"
)

// BuildVolumeProfileFromCells bins stored minute prints onto price rows.
// Buy and sell stay on the price that printed. They are not spread across
// a candle's high and low. covered is true when the live tape already
// reaches the window open, including a quiet tape with no prints.
func BuildVolumeProfileFromCells(ex Exchange, symbol string, cells []FlowAtPricesCell, tick, last float64, from, to time.Time, covered bool) VolumeProfileVenue {
	from, to = from.UTC(), to.UTC()
	out := VolumeProfileVenue{
		Exchange:     ex,
		Symbol:       symbol,
		From:         from,
		To:           to,
		Interval:     string(Interval1m),
		TickSize:     tick,
		LastPrice:    last,
		Bins:         []VolumeProfileBin{},
		LastVsArea:   VolumeProfileVsUnknown,
		BuySellKnown: true,
		Complete:     covered,
	}
	if tick <= 0 {
		out.BuySellKnown = false
		out.Complete = false
		out.Error = "no stored prints in this range"
		out.Summary = out.Error
		return out
	}

	type acc struct {
		vol, buy, sell float64
	}
	m := map[int64]*acc{}
	minutes := map[int64]struct{}{}
	var earliest time.Time
	var rangeLow, rangeHigh float64
	for _, c := range cells {
		if c.Price <= 0 || (c.Buy <= 0 && c.Sell <= 0) {
			continue
		}
		if !c.Minute.IsZero() {
			t := c.Minute.UTC()
			if earliest.IsZero() || t.Before(earliest) {
				earliest = t
			}
			minutes[t.Unix()] = struct{}{}
		}
		if rangeLow == 0 || c.Price < rangeLow {
			rangeLow = c.Price
		}
		if c.Price > rangeHigh {
			rangeHigh = c.Price
		}
		if last <= 0 {
			last = c.Price
			out.LastPrice = last
		}
		idx := priceBinIndex(c.Price, tick)
		cur := m[idx]
		if cur == nil {
			cur = &acc{}
			m[idx] = cur
		}
		cur.buy += c.Buy
		cur.sell += c.Sell
		cur.vol += c.Buy + c.Sell
		out.BuyVolume += c.Buy
		out.SellVolume += c.Sell
	}
	out.Started = earliest
	out.BarCount = len(minutes)
	out.Low, out.High = rangeLow, rangeHigh
	if !covered && !earliest.IsZero() && !earliest.After(from) {
		out.Complete = true
	}
	if len(m) == 0 {
		out.BuySellKnown = false
		if covered {
			out.Complete = true
			out.Summary = prettyBase(symbol) + ": no aggressive prints in this window."
			return out
		}
		out.Complete = false
		out.Error = "no stored prints in this range"
		out.Summary = out.Error
		return out
	}

	idxs := make([]int64, 0, len(m))
	for idx := range m {
		idxs = append(idxs, idx)
	}
	sort.Slice(idxs, func(i, j int) bool { return idxs[i] < idxs[j] })
	bins := make([]VolumeProfileBin, 0, len(idxs))
	var total float64
	for _, idx := range idxs {
		a := m[idx]
		total += a.vol
		row := VolumeProfileBin{
			Price:      binLow(idx, tick),
			High:       binLow(idx+1, tick),
			Volume:     a.vol,
			BuyVolume:  a.buy,
			SellVolume: a.sell,
		}
		if side := a.buy + a.sell; side > 0 {
			row.BuyPct = a.buy / side * 100
		}
		bins = append(bins, row)
	}
	out.TotalVolume = total
	for i := range bins {
		if total > 0 {
			bins[i].SharePct = bins[i].Volume / total * 100
		}
	}
	annotateVolumeProfile(bins, last, total)
	out.Bins = bins
	out.POC, out.ValueArea = volumeProfileStats(bins)
	out.LastVsArea = lastVsValueArea(last, out.ValueArea)
	out.Summary = ExplainVolumeProfileVenue(out)
	return out
}

// ProfileBarsFromCells turns stored minute rows into the bars used to place
// open-interest steps on the prices that actually printed.
func ProfileBarsFromCells(cells []FlowAtPricesCell) []VolumeProfileBar {
	type g struct {
		t         time.Time
		low, high float64
		buy, sell float64
		set       bool
	}
	order := make([]int64, 0)
	m := map[int64]*g{}
	for _, c := range cells {
		if c.Price <= 0 || c.Buy+c.Sell <= 0 || c.Minute.IsZero() {
			continue
		}
		key := c.Minute.UTC().Unix()
		cur := m[key]
		if cur == nil {
			cur = &g{t: c.Minute.UTC()}
			m[key] = cur
			order = append(order, key)
		}
		if !cur.set || c.Price < cur.low {
			cur.low = c.Price
		}
		if !cur.set || c.Price > cur.high {
			cur.high = c.Price
		}
		cur.set = true
		cur.buy += c.Buy
		cur.sell += c.Sell
	}
	out := make([]VolumeProfileBar, 0, len(order))
	for _, key := range order {
		cur := m[key]
		out = append(out, VolumeProfileBar{
			Time:         cur.t,
			High:         cur.high,
			Low:          cur.low,
			Close:        cur.high,
			Volume:       cur.buy + cur.sell,
			BuyVolume:    cur.buy,
			SellVolume:   cur.sell,
			BuySellKnown: true,
		})
	}
	return out
}
