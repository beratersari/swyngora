package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	FlowAtPricesWindow5m  = "5m"
	FlowAtPricesWindow15m = "15m"
	FlowAtPricesWindow1h  = "1h"
	FlowAtPricesWindow4h  = "4h"
	FlowAtPricesWindow1d  = "1d"
	FlowAtPricesWindow3d  = "3d"
	FlowAtPricesWindow7d  = "7d"

	DefaultFlowAtPricesWindow = FlowAtPricesWindow15m
	// MaxFlowAtPricesRange is how far a custom range may look back. Longer
	// windows come from the stored minute ladder, which only grows while
	// the process is running.
	MaxFlowAtPricesRange   = 7 * 24 * time.Hour
	MaxFlowAtPricesBins    = 200
	MinFlowAtPricesBins    = 8
	TargetFlowAtPricesBins = 64
)

// FlowAtPricesWindows are the named lookbacks the API accepts.
var FlowAtPricesWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{FlowAtPricesWindow5m, 5 * time.Minute},
	{FlowAtPricesWindow15m, 15 * time.Minute},
	{FlowAtPricesWindow1h, time.Hour},
	{FlowAtPricesWindow4h, 4 * time.Hour},
	{FlowAtPricesWindow1d, 24 * time.Hour},
	{FlowAtPricesWindow3d, 3 * 24 * time.Hour},
	{FlowAtPricesWindow7d, 7 * 24 * time.Hour},
}

// FlowAtPricesLevel is aggressive buy vs sell at one price row.
type FlowAtPricesLevel struct {
	Price      float64 // inclusive low of the row
	High       float64 // exclusive top
	BuyVolume  float64
	SellVolume float64
	Volume     float64
	BuyCount   int
	SellCount  int
	Trades     int
	Delta      float64 // buy − sell (USDT)
	BuyPct     float64
	SharePct   float64
	Dominant   string // buy | sell | balanced
	IsPoc      bool
}

// FlowAtPricesPOC is the price row with the most aggressive volume.
type FlowAtPricesPOC struct {
	Price      float64
	High       float64
	BuyVolume  float64
	SellVolume float64
	Volume     float64
	Delta      float64
	Trades     int
	SharePct   float64
}

// FlowAtPricesVenue is one exchange's print ladder (or combined).
type FlowAtPricesVenue struct {
	Exchange   Exchange
	Symbol     string
	From       time.Time
	To         time.Time
	TickSize   float64
	LastPrice  float64
	High       float64
	Low        float64
	BuyVolume  float64
	SellVolume float64
	Volume     float64
	Delta      float64
	BuyCount   int
	SellCount  int
	Trades     int
	Dominant   string
	POC        FlowAtPricesPOC
	Levels     []FlowAtPricesLevel
	Complete   bool
	Truncated  bool
	KnownFrom  time.Time
	KnownTo    time.Time
	Summary    string
	Error      string
}

// FlowAtPricesReport is the API result.
type FlowAtPricesReport struct {
	Symbol   string
	Exchange string
	Window   string
	From     time.Time
	To       time.Time
	AsOf     time.Time
	MinPrice float64
	MaxPrice float64
	TickSize float64
	Venues   []FlowAtPricesVenue
	Combined *FlowAtPricesVenue
	Summary  string
	Note     string
}

// ParseFlowAtPricesWindow accepts 5m / 15m / 1h / 4h / 1d / 3d / 7d (empty = 15m).
func ParseFlowAtPricesWindow(raw string) (string, time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = DefaultFlowAtPricesWindow
	}
	for _, w := range FlowAtPricesWindows {
		if w.ID == s {
			return w.ID, w.Dur, nil
		}
	}
	return "", 0, fmt.Errorf("%w: window must be 5m, 15m, 1h, 4h, 1d, 3d, or 7d", ErrInvalidArgument)
}

// ResolveFlowAtPricesRange picks [from, to] from an explicit range or a named window.
// start/end win when either is set. Max span is 7 days (stored minute ladder).
func ResolveFlowAtPricesRange(window string, start, end *time.Time, now time.Time) (from, to time.Time, windowID string, err error) {
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	hasStart := start != nil && !start.IsZero()
	hasEnd := end != nil && !end.IsZero()
	if hasStart || hasEnd {
		to = now
		if hasEnd {
			to = end.UTC()
		}
		from = to.Add(-15 * time.Minute)
		if hasStart {
			from = start.UTC()
		}
		if !to.After(from) {
			return time.Time{}, time.Time{}, "", fmt.Errorf("%w: endTime must be after startTime", ErrInvalidArgument)
		}
		if to.Sub(from) > MaxFlowAtPricesRange {
			return time.Time{}, time.Time{}, "", fmt.Errorf("%w: range must be <= 7d", ErrInvalidArgument)
		}
		return from, to, "custom", nil
	}
	id, dur, err := ParseFlowAtPricesWindow(window)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	return now.Add(-dur), now, id, nil
}

// ClampFlowAtPricesTick widens tick until the range is at most MaxFlowAtPricesBins.
func ClampFlowAtPricesTick(low, high, tick float64) float64 {
	return ClampVolumeProfileTick(low, high, tick)
}

// AutoFlowAtPricesTick chooses a row width so the range is about TargetFlowAtPricesBins rows.
func AutoFlowAtPricesTick(low, high float64) float64 {
	return AutoTickSize(low, high)
}

// BuildFlowAtPricesVenue bins aggressive prints onto price rows (not candle smear).
func BuildFlowAtPricesVenue(ex Exchange, symbol string, prints []TakerPrint, tick, last, minPrice, maxPrice float64, from, to time.Time) FlowAtPricesVenue {
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
	clean := filterFlowPrints(prints, from, to, minPrice, maxPrice)
	if len(clean) == 0 {
		out.Error = "no aggressive prints in this range"
		out.Summary = out.Error
		return out
	}
	var oldest, newest time.Time
	var rangeLow, rangeHigh float64
	for _, p := range clean {
		if oldest.IsZero() || p.Time.Before(oldest) {
			oldest = p.Time
		}
		if newest.IsZero() || p.Time.After(newest) {
			newest = p.Time
		}
		if rangeLow == 0 || p.Price < rangeLow {
			rangeLow = p.Price
		}
		if p.Price > rangeHigh {
			rangeHigh = p.Price
		}
	}
	out.KnownFrom, out.KnownTo = oldest, newest
	out.Low, out.High = rangeLow, rangeHigh
	if last <= 0 {
		out.LastPrice = newestPrice(clean)
	}
	if tick <= 0 {
		tick = AutoFlowAtPricesTick(rangeLow, rangeHigh)
	} else {
		tick = ClampFlowAtPricesTick(rangeLow, rangeHigh, tick)
	}
	out.TickSize = tick
	if tick <= 0 {
		out.Error = "tickSize must be > 0"
		out.Summary = out.Error
		return out
	}

	type acc struct {
		buy, sell   float64
		buyN, sellN int
	}
	m := map[int64]*acc{}
	for _, p := range clean {
		idx := priceBinIndex(p.Price, tick)
		cur := m[idx]
		if cur == nil {
			cur = &acc{}
			m[idx] = cur
		}
		switch p.Side {
		case TakerSideSell:
			cur.sell += p.Notional
			cur.sellN++
		default:
			cur.buy += p.Notional
			cur.buyN++
		}
	}
	if len(m) == 0 {
		out.Error = "no aggressive prints in this range"
		out.Summary = out.Error
		return out
	}

	idxs := make([]int64, 0, len(m))
	for idx := range m {
		idxs = append(idxs, idx)
	}
	// Highest price first — same way a footprint / book ladder is read.
	sort.Slice(idxs, func(i, j int) bool { return idxs[i] > idxs[j] })
	levels := make([]FlowAtPricesLevel, 0, len(idxs))
	var total float64
	for _, idx := range idxs {
		a := m[idx]
		vol := a.buy + a.sell
		total += vol
		row := FlowAtPricesLevel{
			Price:      binLow(idx, tick),
			High:       binLow(idx+1, tick),
			BuyVolume:  a.buy,
			SellVolume: a.sell,
			Volume:     vol,
			BuyCount:   a.buyN,
			SellCount:  a.sellN,
			Trades:     a.buyN + a.sellN,
			Delta:      a.buy - a.sell,
			Dominant:   TakerDominant(a.buy, a.sell),
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

// CombineFlowAtPricesVenues adds Binance and Bybit rows at the same tick.
func CombineFlowAtPricesVenues(symbol string, venues []FlowAtPricesVenue, tick, last float64, from, to time.Time) *FlowAtPricesVenue {
	out := &FlowAtPricesVenue{
		Exchange:  "all",
		Symbol:    NormalizeLiquidationSymbol(symbol),
		From:      from.UTC(),
		To:        to.UTC(),
		TickSize:  tick,
		LastPrice: last,
		Levels:    []FlowAtPricesLevel{},
		Dominant:  TakerSideEven,
	}
	var usable []FlowAtPricesVenue
	for _, v := range venues {
		if v.Error != "" || len(v.Levels) == 0 {
			continue
		}
		usable = append(usable, v)
	}
	if len(usable) == 0 {
		out.Error = "no aggressive prints in this range"
		out.Summary = out.Error
		return out
	}
	if last <= 0 {
		for _, v := range usable {
			if v.LastPrice > 0 {
				last = v.LastPrice
				out.LastPrice = last
				break
			}
		}
	}
	if tick <= 0 {
		var lo, hi float64
		for _, v := range usable {
			if v.Low > 0 && (lo == 0 || v.Low < lo) {
				lo = v.Low
			}
			if v.High > hi {
				hi = v.High
			}
			if v.TickSize > 0 && (tick == 0 || v.TickSize < tick) {
				tick = v.TickSize
			}
		}
		if tick <= 0 {
			tick = AutoFlowAtPricesTick(lo, hi)
		}
	}
	out.TickSize = tick
	if tick <= 0 {
		out.Error = "tickSize must be > 0"
		out.Summary = out.Error
		return out
	}

	type acc struct {
		buy, sell   float64
		buyN, sellN int
	}
	m := map[int64]*acc{}
	complete := true
	var oldest, newest time.Time
	var rangeLow, rangeHigh float64
	for _, v := range usable {
		if !v.Complete {
			complete = false
		}
		if v.Truncated {
			out.Truncated = true
		}
		if !v.KnownFrom.IsZero() && (oldest.IsZero() || v.KnownFrom.Before(oldest)) {
			oldest = v.KnownFrom
		}
		if !v.KnownTo.IsZero() && (newest.IsZero() || v.KnownTo.After(newest)) {
			newest = v.KnownTo
		}
		if v.Low > 0 && (rangeLow == 0 || v.Low < rangeLow) {
			rangeLow = v.Low
		}
		if v.High > rangeHigh {
			rangeHigh = v.High
		}
		for _, lv := range v.Levels {
			idx := priceBinIndex(lv.Price, tick)
			cur := m[idx]
			if cur == nil {
				cur = &acc{}
				m[idx] = cur
			}
			cur.buy += lv.BuyVolume
			cur.sell += lv.SellVolume
			cur.buyN += lv.BuyCount
			cur.sellN += lv.SellCount
		}
	}
	out.KnownFrom, out.KnownTo = oldest, newest
	out.Low, out.High = rangeLow, rangeHigh
	out.Complete = complete && !out.Truncated

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
			Price:      binLow(idx, tick),
			High:       binLow(idx+1, tick),
			BuyVolume:  a.buy,
			SellVolume: a.sell,
			Volume:     vol,
			BuyCount:   a.buyN,
			SellCount:  a.sellN,
			Trades:     a.buyN + a.sellN,
			Delta:      a.buy - a.sell,
			Dominant:   TakerDominant(a.buy, a.sell),
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
	out.Summary = ExplainFlowAtPricesVenue(*out)
	return out
}

// ExplainFlowAtPricesVenue writes a short POC + delta line.
func ExplainFlowAtPricesVenue(v FlowAtPricesVenue) string {
	if v.Error != "" {
		return v.Error
	}
	if v.POC.Volume <= 0 {
		return prettyBase(v.Symbol) + ": no aggressive prints in this range."
	}
	name := prettyBase(v.Symbol)
	head := fmt.Sprintf("%s most volume at %s — %s buy vs %s sell (delta %s, %d prints).",
		name, formatQty(v.POC.Price), formatQty(v.POC.BuyVolume), formatQty(v.POC.SellVolume),
		FormatSignedQty(v.POC.Delta), v.POC.Trades)
	head += fmt.Sprintf(" Tape delta %s across %d prints.", FormatSignedQty(v.Delta), v.Trades)
	if v.Truncated && !v.KnownFrom.IsZero() {
		head += fmt.Sprintf(" Tape starts at %s — earlier prints in this window are missing.", v.KnownFrom.UTC().Format(time.RFC3339))
	}
	return head
}

// ExplainFlowAtPricesReport prefers combined, then the first venue with a POC.
func ExplainFlowAtPricesReport(r FlowAtPricesReport) string {
	if r.Combined != nil && r.Combined.POC.Volume > 0 {
		return r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.POC.Volume > 0 {
			return v.Summary
		}
	}
	for _, v := range r.Venues {
		if v.Summary != "" {
			return v.Summary
		}
	}
	return prettyBase(r.Symbol) + ": no aggressive prints in this range."
}

func filterFlowPrints(prints []TakerPrint, from, to time.Time, minPrice, maxPrice float64) []TakerPrint {
	out := make([]TakerPrint, 0, len(prints))
	for _, p := range prints {
		if p.Notional <= 0 || p.Price <= 0 || p.Time.IsZero() {
			continue
		}
		t := p.Time.UTC()
		if t.Before(from) || t.After(to) {
			continue
		}
		if minPrice > 0 && p.Price < minPrice {
			continue
		}
		if maxPrice > 0 && p.Price > maxPrice {
			continue
		}
		p.Time = t
		out = append(out, p)
	}
	return out
}

func newestPrice(prints []TakerPrint) float64 {
	var latest time.Time
	var px float64
	for _, p := range prints {
		if latest.IsZero() || p.Time.After(latest) {
			latest = p.Time
			px = p.Price
		}
	}
	return px
}

func pickFlowPOC(levels []FlowAtPricesLevel, last float64) int {
	best := -1
	var bestVol float64
	var bestDist float64
	for i, lv := range levels {
		if lv.Volume <= 0 {
			continue
		}
		mid := (lv.Price + lv.High) / 2
		dist := math.Abs(mid - last)
		if best < 0 || lv.Volume > bestVol || (lv.Volume == bestVol && last > 0 && dist < bestDist) {
			best = i
			bestVol = lv.Volume
			bestDist = dist
		}
	}
	return best
}

// FlowRangeCovered reports whether the oldest print reaches the start of the
// requested window, allowing a short slack so a one-minute bucket still counts.
func FlowRangeCovered(oldest, from time.Time, dur time.Duration) bool {
	return flowComplete(oldest, from, dur)
}

func flowComplete(oldest, from time.Time, dur time.Duration) bool {
	if oldest.IsZero() || from.IsZero() {
		return false
	}
	slack := dur / 5
	if slack < 30*time.Second {
		slack = 30 * time.Second
	}
	return !oldest.After(from.Add(slack))
}
