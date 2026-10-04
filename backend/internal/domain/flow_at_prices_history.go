package domain

import (
	"context"
	"math"
	"sort"
	"time"
)

// FlowAtPricesCell is buy/sell notional and print counts at one stored price
// during one closed minute. Price is the low of the storage row.
type FlowAtPricesCell struct {
	Exchange Exchange
	Symbol   string
	Minute   time.Time
	Price    float64
	Buy      float64
	Sell     float64
	BuyN     int
	SellN    int
}

// FlowAtPricesStore keeps minute price rows so a later lookback is not limited
// to the live venue tape.
type FlowAtPricesStore interface {
	AddFlowCells(ctx context.Context, cells []FlowAtPricesCell) error
	ListFlowCells(ctx context.Context, exchange Exchange, symbol string, from, to time.Time) ([]FlowAtPricesCell, error)
	FlowCursor(ctx context.Context, exchange Exchange, symbol string) (time.Time, error)
	SetFlowCursor(ctx context.Context, exchange Exchange, symbol string, at time.Time) error
	PurgeFlowCells(ctx context.Context, cutoff time.Time) (int, error)
}

// FlowHistoryTick is the stored row width: about 0.05% of price, snapped to
// 1, 2, or 5 × 10^n. Display ticks that are wider roll these rows up.
func FlowHistoryTick(price float64) float64 {
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return 0
	}
	raw := price * 0.0005
	if raw <= 0 {
		return 0
	}
	exp := math.Floor(math.Log10(raw))
	pow := math.Pow(10, exp)
	n := raw / pow
	m := 10.0
	switch {
	case n < 1.5:
		m = 1
	case n < 3.5:
		m = 2
	case n < 7.5:
		m = 5
	}
	return m * pow
}

// PrintsFromFlowCells turns stored minute rows into counted fills.
// Each side is one row whose Count is the stored print count and whose
// Notional is that side's total. A side with no count is left out.
func PrintsFromFlowCells(cells []FlowAtPricesCell) []TakerPrint {
	out := make([]TakerPrint, 0, len(cells))
	for _, c := range cells {
		if c.Minute.IsZero() {
			continue
		}
		at := c.Minute.UTC()
		symbol := NormalizeLiquidationSymbol(c.Symbol)
		if c.BuyN > 0 && c.Buy > 0 {
			out = append(out, TakerPrint{
				Exchange: c.Exchange, Symbol: symbol, Side: TakerSideBuy,
				Price: c.Price, Notional: c.Buy, Time: at, Count: c.BuyN,
			})
		}
		if c.SellN > 0 && c.Sell > 0 {
			out = append(out, TakerPrint{
				Exchange: c.Exchange, Symbol: symbol, Side: TakerSideSell,
				Price: c.Price, Notional: c.Sell, Time: at, Count: c.SellN,
			})
		}
	}
	return out
}

// AccumulateFlowPrints groups prints newer than after into closed-minute rows.
// newest is the latest print time that was stored.
func AccumulateFlowPrints(prints []TakerPrint, after time.Time) (cells []FlowAtPricesCell, newest time.Time) {
	type key struct {
		ex  Exchange
		sym string
		min int64
		idx int64
	}
	acc := map[key]*FlowAtPricesCell{}
	for _, p := range prints {
		if p.Price <= 0 || p.Notional <= 0 || p.Time.IsZero() {
			continue
		}
		if !after.IsZero() && !p.Time.After(after) {
			continue
		}
		tick := FlowHistoryTick(p.Price)
		if tick <= 0 {
			continue
		}
		if p.Time.After(newest) {
			newest = p.Time
		}
		minute := p.Time.UTC().Truncate(time.Minute)
		idx := priceBinIndex(p.Price, tick)
		k := key{ex: p.Exchange, sym: NormalizeLiquidationSymbol(p.Symbol), min: minute.Unix(), idx: idx}
		cur := acc[k]
		if cur == nil {
			cur = &FlowAtPricesCell{
				Exchange: p.Exchange,
				Symbol:   k.sym,
				Minute:   minute,
				Price:    binLow(idx, tick),
			}
			acc[k] = cur
		}
		if p.Side == TakerSideSell {
			cur.Sell += p.Notional
			cur.SellN++
		} else {
			cur.Buy += p.Notional
			cur.BuyN++
		}
	}
	cells = make([]FlowAtPricesCell, 0, len(acc))
	for _, c := range acc {
		cells = append(cells, *c)
	}
	return cells, newest.UTC()
}

// LatestFlowMinute is the newest stored minute, or zero when cells is empty.
func LatestFlowMinute(cells []FlowAtPricesCell) time.Time {
	var latest time.Time
	for _, c := range cells {
		if c.Minute.After(latest) {
			latest = c.Minute
		}
	}
	return latest
}

// BuildFlowAtPricesFromCells bins stored minute rows onto the display tick.
func BuildFlowAtPricesFromCells(ex Exchange, symbol string, cells []FlowAtPricesCell, tick, last, minPrice, maxPrice float64, from, to time.Time) FlowAtPricesVenue {
	symbol = NormalizeLiquidationSymbol(symbol)
	from, to = from.UTC(), to.UTC()
	out := FlowAtPricesVenue{
		Exchange:  ex,
		Symbol:    symbol,
		From:      from,
		To:        to,
		TickSize:  tick,
		LastPrice: last,
		Levels:    []FlowAtPricesLevel{},
		Dominant:  TakerSideEven,
	}
	type kept struct {
		price       float64
		buy, sell   float64
		buyN, sellN int
		minute      time.Time
	}
	rows := make([]kept, 0, len(cells))
	var oldest, newest time.Time
	var rangeLow, rangeHigh float64
	for _, c := range cells {
		if c.Price <= 0 || (c.Buy <= 0 && c.Sell <= 0) {
			continue
		}
		minute := c.Minute.UTC()
		if !from.IsZero() && minute.Add(time.Minute).Before(from) {
			continue
		}
		if !to.IsZero() && !minute.Before(to) {
			continue
		}
		if minPrice > 0 && c.Price+1e-12 < minPrice {
			continue
		}
		if maxPrice > 0 && c.Price > maxPrice {
			continue
		}
		if oldest.IsZero() || minute.Before(oldest) {
			oldest = minute
		}
		end := minute.Add(time.Minute)
		if newest.IsZero() || end.After(newest) {
			newest = end
		}
		if rangeLow == 0 || c.Price < rangeLow {
			rangeLow = c.Price
		}
		if c.Price > rangeHigh {
			rangeHigh = c.Price
		}
		rows = append(rows, kept{price: c.Price, buy: c.Buy, sell: c.Sell, buyN: c.BuyN, sellN: c.SellN, minute: minute})
	}
	if len(rows) == 0 {
		out.Error = "no aggressive prints in this range"
		out.Summary = out.Error
		return out
	}
	if tick <= 0 {
		tick = AutoFlowAtPricesTick(rangeLow, rangeHigh)
	} else {
		tick = ClampFlowAtPricesTick(rangeLow, rangeHigh, tick)
	}
	if tick <= 0 {
		tick = FlowHistoryTick(rangeLow)
	}
	type acc struct {
		buy, sell   float64
		buyN, sellN int
	}
	m := map[int64]*acc{}
	for _, row := range rows {
		idx := priceBinIndex(row.price, tick)
		cur := m[idx]
		if cur == nil {
			cur = &acc{}
			m[idx] = cur
		}
		cur.buy += row.buy
		cur.sell += row.sell
		cur.buyN += row.buyN
		cur.sellN += row.sellN
	}
	out.TickSize = tick
	out.KnownFrom, out.KnownTo = oldest, newest
	out.Low, out.High = rangeLow, rangeHigh
	if last <= 0 {
		out.LastPrice = rangeHigh
	}
	idxs := make([]int64, 0, len(m))
	for idx := range m {
		idxs = append(idxs, idx)
	}
	sort.Slice(idxs, func(i, j int) bool { return idxs[i] > idxs[j] })
	levels := make([]FlowAtPricesLevel, 0, len(idxs))
	var total float64
	for _, idx := range idxs {
		a := m[idx]
		vol := a.buy + a.sell
		total += vol
		row := FlowAtPricesLevel{
			Price: binLow(idx, tick), High: binLow(idx+1, tick),
			BuyVolume: a.buy, SellVolume: a.sell, Volume: vol,
			BuyCount: a.buyN, SellCount: a.sellN, Trades: a.buyN + a.sellN,
			Delta: a.buy - a.sell, Dominant: TakerDominant(a.buy, a.sell),
		}
		if vol > 0 {
			row.BuyPct = a.buy / vol * 100
		}
		levels = append(levels, row)
		out.BuyVolume += a.buy
		out.SellVolume += a.sell
		out.BuyCount += a.buyN
		out.SellCount += a.sellN
	}
	out.Volume = total
	out.Delta = out.BuyVolume - out.SellVolume
	out.Trades = out.BuyCount + out.SellCount
	out.Dominant = TakerDominant(out.BuyVolume, out.SellVolume)
	for i := range levels {
		if total > 0 {
			levels[i].SharePct = levels[i].Volume / total * 100
		}
	}
	pocIdx := pickFlowPOC(levels, out.LastPrice)
	if pocIdx >= 0 {
		levels[pocIdx].IsPoc = true
		p := levels[pocIdx]
		out.POC = FlowAtPricesPOC{
			Price: p.Price, High: p.High, BuyVolume: p.BuyVolume, SellVolume: p.SellVolume,
			Volume: p.Volume, Delta: p.Delta, Trades: p.Trades, SharePct: p.SharePct,
		}
	}
	out.Levels = levels
	out.Complete = flowComplete(oldest, from, to.Sub(from))
	out.Truncated = !out.Complete
	out.Summary = ExplainFlowAtPricesVenue(out)
	return out
}
