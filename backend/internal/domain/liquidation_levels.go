package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	LiquidationLevelsKindMap   = "levels"
	LiquidationLevelsKindTotal = "totals"
)

// Chart leverage buckets shown on the CoinGlass-style map (10 / 25 / 50 / 100).
const (
	ChartLeverage10      = 10
	ChartLeverage25      = 25
	ChartLeverage50      = 50
	ChartLeverage100     = 100
	liquidationLevelBins = 40
)

// ChartLeverageBuckets is the display mix (100x nearest last, 10x farthest).
var ChartLeverageBuckets = []int{ChartLeverage10, ChartLeverage25, ChartLeverage50, ChartLeverage100}

// LiquidationLeverageSlice is notional at one display leverage on one bar.
type LiquidationLeverageSlice struct {
	Leverage      int
	LongNotional  string
	ShortNotional string
}

// LiquidationLevelBar is estimated long/short notional at one price.
type LiquidationLevelBar struct {
	Price         string
	LongNotional  string
	ShortNotional string
	TotalNotional string
	CumLong       string
	CumShort      string
	CumTotal      string
	ByLeverage    []LiquidationLeverageSlice
}

// LiquidationTimeBar is observed long/short notional in one time bucket.
type LiquidationTimeBar struct {
	Time          time.Time
	LongNotional  string
	ShortNotional string
	TotalNotional string
	Count         int
}

// LiquidationLevelsReport is the CoinGlass-style map (per coin) or
// market-wide total bars (all coins).
type LiquidationLevelsReport struct {
	Kind       string
	Symbol     string
	Exchange   string
	Range      string
	From       time.Time
	To         time.Time
	LastPrice  string
	LastPrices map[string]string
	Levels     []LiquidationLevelBar
	Bars       []LiquidationTimeBar
	Feed       LiquidationFeed
	Missing    []string
	Note       string
}

// ChartLeverageBucket maps a hunt leverage onto 10 / 25 / 50 / 100.
func ChartLeverageBucket(lev float64) int {
	switch {
	case lev <= 15:
		return ChartLeverage10
	case lev <= 35:
		return ChartLeverage25
	case lev <= 70:
		return ChartLeverage50
	default:
		return ChartLeverage100
	}
}

// ParseLiquidationLevelsSymbol accepts a pair, or all/* (empty = all).
func ParseLiquidationLevelsSymbol(raw string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "-", "")
	if s == "" || s == "ALL" || s == "*" {
		return "all", nil
	}
	return ValidateOpenInterestSymbol(s)
}

// CollapseHuntHeatmapLevels sums time columns into one bar per price bin.
// prices[0] is the highest bin (same order as the heatmap).
func CollapseHuntHeatmapLevels(grid HuntHeatmapGrid, prices []float64) []LiquidationLevelBar {
	n := len(prices)
	if n == 0 {
		return []LiquidationLevelBar{}
	}
	out := make([]LiquidationLevelBar, 0, n)
	for i, px := range prices {
		var longN, shortN float64
		for t := 0; t < len(grid.Longs); t++ {
			if i < len(grid.Longs[t]) {
				longN += grid.Longs[t][i]
			}
		}
		for t := 0; t < len(grid.Shorts); t++ {
			if i < len(grid.Shorts[t]) {
				shortN += grid.Shorts[t][i]
			}
		}
		out = append(out, LiquidationLevelBar{
			Price:         formatFixed(px, decimalsForStep(px/10000)+1),
			LongNotional:  formatQty(longN),
			ShortNotional: formatQty(shortN),
			TotalNotional: formatQty(longN + shortN),
		})
	}
	return out
}

// PickHuntHeatmapGrid returns the venue slice (combined when exchange is all).
func PickHuntHeatmapGrid(rep HuntHeatmapReport, exchange string) HuntHeatmapGrid {
	switch exchange {
	case string(ExchangeBinance):
		return rep.Binance
	case string(ExchangeBybit):
		return rep.Bybit
	default:
		return rep.Combined
	}
}

// LiquidationTimeBarStep is the bucket size for an observed-totals window.
func LiquidationTimeBarStep(window time.Duration) time.Duration {
	switch {
	case window <= time.Hour:
		return 5 * time.Minute
	case window <= 4*time.Hour:
		return 15 * time.Minute
	case window <= 12*time.Hour:
		return 30 * time.Minute
	default:
		return time.Hour
	}
}

// TimeBars folds stored prints into long/short buckets. symbol "" or "all"
// includes every coin. exchange is binance, bybit, or all.
func (b *LiquidationBook) TimeBars(exchange, symbol string, from, to time.Time, step time.Duration) []LiquidationTimeBar {
	if to.IsZero() {
		to = time.Now().UTC()
	}
	to = to.UTC()
	from = from.UTC()
	if from.IsZero() || !from.Before(to) {
		from = to.Add(-24 * time.Hour)
	}
	if step <= 0 {
		step = LiquidationTimeBarStep(to.Sub(from))
	}
	n := int(to.Sub(from) / step)
	if n < 1 {
		n = 1
	}
	if n > 96 {
		n = 96
		step = to.Sub(from) / time.Duration(n)
		if step <= 0 {
			step = time.Hour
		}
	}
	bars := make([]LiquidationTimeBar, n)
	longs := make([]float64, n)
	shorts := make([]float64, n)
	for i := 0; i < n; i++ {
		bars[i].Time = from.Add(time.Duration(i) * step)
	}
	if b == nil {
		for i := range bars {
			bars[i].LongNotional = "0"
			bars[i].ShortNotional = "0"
			bars[i].TotalNotional = "0"
		}
		return bars
	}
	symbol = NormalizeLiquidationSymbol(symbol)
	if symbol == "ALL" {
		symbol = ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for sym, list := range b.bySym {
		if symbol != "" && sym != symbol {
			continue
		}
		for _, e := range list {
			if exchange != "" && exchange != "all" && string(e.Exchange) != exchange {
				continue
			}
			if e.Time.Before(from) || !e.Time.Before(to) {
				continue
			}
			i := int(e.Time.Sub(from) / step)
			if i < 0 || i >= n {
				continue
			}
			switch e.Side {
			case LiquidationSideLong:
				longs[i] += e.Notional
			case LiquidationSideShort:
				shorts[i] += e.Notional
			}
			bars[i].Count++
		}
	}
	for i := range bars {
		bars[i].LongNotional = formatQty(longs[i])
		bars[i].ShortNotional = formatQty(shorts[i])
		bars[i].TotalNotional = formatQty(longs[i] + shorts[i])
	}
	return bars
}

// LastHuntPrice is the newest print on that venue only.
// exchange=all / combined returns 0 — never pick the other venue's last.
func LastHuntPrice(venues []HuntHeatmapVenueSeries, exchange string) float64 {
	if exchange == "" || exchange == "all" || exchange == "combined" {
		return 0
	}
	var best time.Time
	var px float64
	for _, v := range venues {
		if string(v.Exchange) != exchange {
			continue
		}
		for _, p := range v.Prices {
			if p.Price <= 0 {
				continue
			}
			if best.IsZero() || p.Time.After(best) {
				best = p.Time
				px = p.Price
			}
		}
	}
	return px
}

type levelBin struct {
	price float64
	long  [4]float64
	short [4]float64
}

func leverageIndex(lev int) int {
	switch lev {
	case ChartLeverage10:
		return 0
	case ChartLeverage25:
		return 1
	case ChartLeverage50:
		return 2
	default:
		return 3
	}
}

// BuildLiquidationLevelsFromHunt makes CoinGlass-style horizontal bars from
// each venue's own hunt bands. Venues are never used as each other's last price.
func BuildLiquidationLevelsFromHunt(rep HuntReport) LiquidationLevelsReport {
	out := LiquidationLevelsReport{
		Kind:       LiquidationLevelsKindMap,
		Symbol:     rep.Symbol,
		Exchange:   rep.Exchange,
		LastPrices: map[string]string{},
		Levels:     []LiquidationLevelBar{},
		Missing:    []string{},
		Note:       "Estimated liquidation notional at each price from that venue's own last price, open interest, and a 10x/25x/50x/100x leverage mix. Longs sit below last (price drop); shorts sit above (price rise). Combined sums independently modeled venues and does not borrow a missing venue's last price. Informational only.",
	}
	want := []Exchange{ExchangeBinance, ExchangeBybit}
	if rep.Exchange != "" && rep.Exchange != "all" {
		want = []Exchange{Exchange(rep.Exchange)}
	}
	var bins []levelBin
	var last float64
	lasts := map[string]float64{}
	for _, ex := range want {
		v, ok := huntVenue(rep.Venues, ex)
		if !ok || v.Price <= 0 || v.OpenInterestValue <= 0 {
			out.Missing = append(out.Missing, string(ex))
			continue
		}
		lasts[string(ex)] = v.Price
		out.LastPrices[string(ex)] = formatFixed(v.Price, decimalsForStep(v.Price/10000)+1)
		own := binsFromHuntVenue(v)
		if last == 0 {
			last = v.Price
		}
		bins = mergeLevelBins(bins, own, last)
	}
	if len(want) == 1 && last > 0 {
		out.LastPrice = formatFixed(last, decimalsForStep(last/10000)+1)
	}
	if last <= 0 && len(bins) > 0 {
		last = bins[len(bins)/2].price
	}
	out.Levels = finishLevelBars(bins, last)
	return out
}

func huntVenue(venues []HuntVenueReport, ex Exchange) (HuntVenueReport, bool) {
	for _, v := range venues {
		if v.Exchange == ex {
			return v, true
		}
	}
	return HuntVenueReport{}, false
}

func binsFromHuntVenue(v HuntVenueReport) []levelBin {
	if v.Price <= 0 {
		return nil
	}
	lo := v.Price * (1 - HuntLiqDistance(10, HuntMaintenanceMargin) - 0.02)
	hi := v.Price * (1 + HuntLiqDistance(10, HuntMaintenanceMargin) + 0.02)
	if lo <= 0 {
		lo = v.Price * 0.8
	}
	step := (hi - lo) / float64(liquidationLevelBins)
	if step <= 0 {
		step = v.Price * 0.002
	}
	out := make([]levelBin, liquidationLevelBins)
	for i := 0; i < liquidationLevelBins; i++ {
		// index 0 = highest price (same as heatmap)
		out[i].price = hi - (float64(i)+0.5)*step
	}
	addBandsToLevelBins(out, v.UpPressure, lo, hi, step)
	addBandsToLevelBins(out, v.DownPressure, lo, hi, step)
	return out
}

func addBandsToLevelBins(bins []levelBin, bands []HuntBand, lo, hi, step float64) {
	n := len(bins)
	if n == 0 || step <= 0 {
		return
	}
	for _, b := range bands {
		if b.Price <= 0 || b.EstNotional <= 0 {
			continue
		}
		idx := int((hi - b.Price) / step)
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		li := leverageIndex(ChartLeverageBucket(b.Leverage))
		if b.Side == LiquidationSideShort {
			bins[idx].short[li] += b.EstNotional
		} else {
			bins[idx].long[li] += b.EstNotional
		}
	}
}

func mergeLevelBins(dst, src []levelBin, ref float64) []levelBin {
	if len(dst) == 0 {
		return src
	}
	if len(src) == 0 {
		return dst
	}
	if ref <= 0 {
		ref = dst[0].price
	}
	// Re-bin src onto dst's price axis (nearest).
	for _, s := range src {
		best := 0
		bestDt := -1.0
		for i, d := range dst {
			dt := s.price - d.price
			if dt < 0 {
				dt = -dt
			}
			if bestDt < 0 || dt < bestDt {
				bestDt = dt
				best = i
			}
		}
		for i := 0; i < 4; i++ {
			dst[best].long[i] += s.long[i]
			dst[best].short[i] += s.short[i]
		}
	}
	return dst
}

func finishLevelBars(bins []levelBin, last float64) []LiquidationLevelBar {
	out := make([]LiquidationLevelBar, 0, len(bins))
	// Cumulative from last price toward each bar (CoinGlass hover).
	var cumAboveLong, cumAboveShort, cumBelowLong, cumBelowShort float64
	// bins[0] is highest. Walk from last toward high for shorts, toward low for longs.
	type acc struct{ long, short float64 }
	above := make([]acc, len(bins))
	below := make([]acc, len(bins))
	// above last (higher prices): walk from last toward top (index 0)
	for i := len(bins) - 1; i >= 0; i-- {
		if last > 0 && bins[i].price <= last {
			continue
		}
		cumAboveShort += sum4(bins[i].short)
		cumAboveLong += sum4(bins[i].long)
		above[i] = acc{long: cumAboveLong, short: cumAboveShort}
	}
	for i := 0; i < len(bins); i++ {
		if last > 0 && bins[i].price >= last {
			continue
		}
		cumBelowLong += sum4(bins[i].long)
		cumBelowShort += sum4(bins[i].short)
		below[i] = acc{long: cumBelowLong, short: cumBelowShort}
	}
	for i, b := range bins {
		longN := sum4(b.long)
		shortN := sum4(b.short)
		var cumL, cumS float64
		if last <= 0 || b.price >= last {
			cumL, cumS = above[i].long, above[i].short
		} else {
			cumL, cumS = below[i].long, below[i].short
		}
		slices := make([]LiquidationLeverageSlice, 0, 4)
		for _, lev := range ChartLeverageBuckets {
			li := leverageIndex(lev)
			if b.long[li] <= 0 && b.short[li] <= 0 {
				continue
			}
			slices = append(slices, LiquidationLeverageSlice{
				Leverage:      lev,
				LongNotional:  formatQty(b.long[li]),
				ShortNotional: formatQty(b.short[li]),
			})
		}
		out = append(out, LiquidationLevelBar{
			Price:         formatFixed(b.price, decimalsForStep(b.price/10000)+1),
			LongNotional:  formatQty(longN),
			ShortNotional: formatQty(shortN),
			TotalNotional: formatQty(longN + shortN),
			CumLong:       formatQty(cumL),
			CumShort:      formatQty(cumS),
			CumTotal:      formatQty(cumL + cumS),
			ByLeverage:    slices,
		})
	}
	return out
}

func sum4(v [4]float64) float64 {
	return v[0] + v[1] + v[2] + v[3]
}

// Err if range is not a hunt window — re-export helper for callers.
func ParseLiquidationLevelsRange(raw string) (HuntHeatmapSpec, error) {
	spec, err := ParseHuntHeatmapRange(raw)
	if err != nil {
		return spec, fmt.Errorf("%w: range must be 12h, 24h, 3d, or 7d", ErrInvalidArgument)
	}
	return spec, nil
}
