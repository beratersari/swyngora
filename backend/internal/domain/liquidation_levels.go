package domain

import (
	"fmt"
	"math"
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
	liquidationLevelBins = 80
	// liquidationLevelPad widens the map past the 5x band so that
	// distance is a real row, not piled onto the outer bin.
	liquidationLevelPad = 0.02
	// liquidationLevelKernelHalf is how many bins a leverage cluster
	// is spread across (triangle). CoinGlass-style maps are a density,
	// not a single spike at the exact isolated liq price.
	liquidationLevelKernelHalf = 2.5
)

// ChartLeverageBuckets is the display mix (100x nearest last, 10x farthest).
var ChartLeverageBuckets = []int{ChartLeverage10, ChartLeverage25, ChartLeverage50, ChartLeverage100}

// LiquidationLeverageSlice is notional at one display leverage on one bar.
type LiquidationLeverageSlice struct {
	Leverage        int
	LongNotional    string
	ShortNotional   string
	LongConfidence  float64 // 0–1 display weight from later-print review
	ShortConfidence float64
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
	CumByLeverage []LiquidationLeverageSlice
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
	Review     *LiquidationLevelsReview
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
	case string(ExchangeOKX):
		return rep.OKX
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
// each venue's own last price and leverage mix. Venues are aligned by
// percent from their own last onto one shared axis — a missing venue is
// never used as another venue's last. Combined lastPrice is the
// OI-weighted reference of venues that actually have a last.
func BuildLiquidationLevelsFromHunt(rep HuntReport) LiquidationLevelsReport {
	out := LiquidationLevelsReport{
		Kind:       LiquidationLevelsKindMap,
		Symbol:     rep.Symbol,
		Exchange:   rep.Exchange,
		LastPrices: map[string]string{},
		Levels:     []LiquidationLevelBar{},
		Missing:    []string{},
		Note:       "Estimated liquidation notional at each price from that venue's own last, open interest, and a 5x–125x mix drawn as 10x/25x/50x/100x. 5x keeps its own distance and uses the 10x color; 75x and 125x use the 100x color. Longs sit below last (price drop); shorts sit above (price rise). Each bar is the walk from last to that price (cumulative), plus the mass at that bin. Combined aligns venues by percent from their own last and uses an OI-weighted reference last — it does not copy a missing venue's last. Informational only.",
	}
	want := FuturesVenues()
	if rep.Exchange != "" && rep.Exchange != "all" {
		want = []Exchange{Exchange(rep.Exchange)}
	}
	var used []HuntVenueReport
	for _, ex := range want {
		v, ok := huntVenue(rep.Venues, ex)
		if !ok || v.Price <= 0 || v.OpenInterestValue <= 0 {
			out.Missing = append(out.Missing, string(ex))
			continue
		}
		out.LastPrices[string(ex)] = formatFixed(v.Price, decimalsForStep(v.Price/10000)+1)
		used = append(used, v)
	}
	ref := oiWeightedLast(used)
	if ref <= 0 {
		return out
	}
	out.LastPrice = formatFixed(ref, decimalsForStep(ref/10000)+1)
	lo, hi, step := levelGridBounds(ref)
	bins := makeLevelGrid(lo, hi, step)
	for _, v := range used {
		depositVenueOnLevelGrid(bins, v, ref, lo, hi, step)
	}
	out.Levels = finishLevelBars(bins, ref)
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

func oiWeightedLast(venues []HuntVenueReport) float64 {
	var num, den float64
	for _, v := range venues {
		if v.Price <= 0 {
			continue
		}
		w := v.OpenInterestValue
		if w <= 0 {
			w = 1
		}
		num += v.Price * w
		den += w
	}
	if den <= 0 {
		return 0
	}
	return num / den
}

func levelGridBounds(ref float64) (lo, hi, step float64) {
	// 5x is drawn in the 10x color, but it liquidates near 20% from last.
	// The old 10x span clipped that weight onto the outer bins.
	span := HuntLiqDistance(5, HuntMaintenanceMargin) + liquidationLevelPad
	lo = ref * (1 - span)
	hi = ref * (1 + span)
	if lo <= 0 {
		lo = ref * 0.8
	}
	step = (hi - lo) / float64(liquidationLevelBins)
	if step <= 0 {
		step = ref * 0.002
	}
	return lo, hi, step
}

func makeLevelGrid(lo, hi, step float64) []levelBin {
	out := make([]levelBin, liquidationLevelBins)
	for i := 0; i < liquidationLevelBins; i++ {
		// index 0 = highest price (same as heatmap)
		out[i].price = hi - (float64(i)+0.5)*step
	}
	return out
}

func venueSideNotionals(v HuntVenueReport) (longN, shortN float64) {
	if v.EstLongNotional > 0 || v.EstShortNotional > 0 {
		return v.EstLongNotional, v.EstShortNotional
	}
	for _, b := range v.DownPressure {
		longN += b.EstNotional
	}
	for _, b := range v.UpPressure {
		shortN += b.EstNotional
	}
	return longN, shortN
}

func depositVenueOnLevelGrid(bins []levelBin, v HuntVenueReport, ref, lo, hi, step float64) {
	if v.Price <= 0 || ref <= 0 || len(bins) == 0 {
		return
	}
	longN, shortN := venueSideNotionals(v)
	if longN <= 0 && shortN <= 0 {
		return
	}
	// Prefer the full leverage mix (CoinGlass density). Fall back to the
	// hunt bands when the venue has no estimated side notionals.
	if v.EstLongNotional > 0 || v.EstShortNotional > 0 {
		longsCrowded := v.FundingRate > 1e-12
		shortsCrowded := v.FundingRate < -1e-12
		depositMixOnGrid(bins, v.Price, ref, longN, TiltLeverageMix(DefaultHuntLeverageMix, longsCrowded), -1, hi, step)
		depositMixOnGrid(bins, v.Price, ref, shortN, TiltLeverageMix(DefaultHuntLeverageMix, shortsCrowded), 1, hi, step)
		return
	}
	mapBandsToSharedGrid(bins, v.DownPressure, v.Price, ref, hi, step)
	mapBandsToSharedGrid(bins, v.UpPressure, v.Price, ref, hi, step)
	_ = lo
}

func depositMixOnGrid(bins []levelBin, ownLast, ref, sideN float64, mix []HuntLeverageBucket, sign float64, hi, step float64) {
	if sideN <= 0 || ownLast <= 0 || ref <= 0 {
		return
	}
	for _, b := range mix {
		if b.Leverage < 1 || b.Weight <= 0 {
			continue
		}
		dist := HuntLiqDistance(b.Leverage, HuntMaintenanceMargin)
		shared := ref * (1 + sign*dist)
		li := leverageIndex(ChartLeverageBucket(b.Leverage))
		depositLevelKernel(bins, shared, sideN*b.Weight, li, sign > 0, hi, step)
	}
}

func mapBandsToSharedGrid(bins []levelBin, bands []HuntBand, ownLast, ref, hi, step float64) {
	if ownLast <= 0 || ref <= 0 {
		return
	}
	for _, b := range bands {
		if b.Price <= 0 || b.EstNotional <= 0 {
			continue
		}
		pct := (b.Price - ownLast) / ownLast
		shared := ref * (1 + pct)
		lev := b.Leverage
		if lev < 1 {
			if pct >= 0 {
				lev = ChartLeverage100
			} else {
				lev = ChartLeverage10
			}
		}
		li := leverageIndex(ChartLeverageBucket(lev))
		depositLevelKernel(bins, shared, b.EstNotional, li, b.Side == LiquidationSideShort || pct > 0, hi, step)
	}
}

func depositLevelKernel(bins []levelBin, px, amount float64, li int, short bool, hi, step float64) {
	n := len(bins)
	if n == 0 || amount <= 0 || step <= 0 {
		return
	}
	center := (hi - px) / step
	if center < 0 {
		center = 0
	}
	if center > float64(n) {
		center = float64(n)
	}
	half := liquidationLevelKernelHalf
	type wt struct {
		i int
		w float64
	}
	var parts []wt
	var sum float64
	loI := int(math.Floor(center - half))
	hiI := int(math.Ceil(center + half))
	for i := loI; i <= hiI; i++ {
		if i < 0 || i >= n {
			continue
		}
		d := math.Abs((float64(i) + 0.5) - center)
		w := 1 - d/half
		if w <= 0 {
			continue
		}
		parts = append(parts, wt{i: i, w: w})
		sum += w
	}
	if sum <= 0 || len(parts) == 0 {
		idx := int(center)
		if idx >= n {
			idx = n - 1
		}
		if idx < 0 {
			idx = 0
		}
		if short {
			bins[idx].short[li] += amount
		} else {
			bins[idx].long[li] += amount
		}
		return
	}
	for _, p := range parts {
		add := amount * p.w / sum
		if short {
			bins[p.i].short[li] += add
		} else {
			bins[p.i].long[li] += add
		}
	}
}

func finishLevelBars(bins []levelBin, last float64) []LiquidationLevelBar {
	out := make([]LiquidationLevelBar, 0, len(bins))
	type acc struct {
		long  [4]float64
		short [4]float64
	}
	above := make([]acc, len(bins))
	below := make([]acc, len(bins))
	var cumAbove, cumBelow acc
	// bins[0] is highest. Walk from last toward the top for the short side.
	for i := len(bins) - 1; i >= 0; i-- {
		if last > 0 && bins[i].price <= last {
			continue
		}
		for li := 0; li < 4; li++ {
			cumAbove.long[li] += bins[i].long[li]
			cumAbove.short[li] += bins[i].short[li]
		}
		above[i] = cumAbove
	}
	for i := 0; i < len(bins); i++ {
		if last > 0 && bins[i].price >= last {
			continue
		}
		for li := 0; li < 4; li++ {
			cumBelow.long[li] += bins[i].long[li]
			cumBelow.short[li] += bins[i].short[li]
		}
		below[i] = cumBelow
	}
	for i, b := range bins {
		longN := sum4(b.long)
		shortN := sum4(b.short)
		var walked acc
		if last <= 0 || b.price >= last {
			walked = above[i]
		} else {
			walked = below[i]
		}
		slices := make([]LiquidationLeverageSlice, 0, 4)
		cumSlices := make([]LiquidationLeverageSlice, 0, 4)
		for _, lev := range ChartLeverageBuckets {
			li := leverageIndex(lev)
			slices = append(slices, LiquidationLeverageSlice{
				Leverage:      lev,
				LongNotional:  formatQty(b.long[li]),
				ShortNotional: formatQty(b.short[li]),
			})
			cumSlices = append(cumSlices, LiquidationLeverageSlice{
				Leverage:      lev,
				LongNotional:  formatQty(walked.long[li]),
				ShortNotional: formatQty(walked.short[li]),
			})
		}
		out = append(out, LiquidationLevelBar{
			Price:         formatFixed(b.price, decimalsForStep(b.price/10000)+1),
			LongNotional:  formatQty(longN),
			ShortNotional: formatQty(shortN),
			TotalNotional: formatQty(longN + shortN),
			CumLong:       formatQty(sum4(walked.long)),
			CumShort:      formatQty(sum4(walked.short)),
			CumTotal:      formatQty(sum4(walked.long) + sum4(walked.short)),
			ByLeverage:    slices,
			CumByLeverage: cumSlices,
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
	if err != nil || spec.Range == HuntHeatmap90d {
		return HuntHeatmapSpec{}, fmt.Errorf("%w: range must be 12h, 24h, 3d, 7d, 14d, or 30d", ErrInvalidArgument)
	}
	return spec, nil
}
