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

// LiquidationLevelBar is estimated long/short notional at one price.
type LiquidationLevelBar struct {
	Price         string
	LongNotional  string
	ShortNotional string
	TotalNotional string
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
	Kind      string
	Symbol    string
	Exchange  string
	Range     string
	From      time.Time
	To        time.Time
	LastPrice string
	Levels    []LiquidationLevelBar
	Bars      []LiquidationTimeBar
	Note      string
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

// LastHuntPrice is the newest venue print in the heatmap series.
func LastHuntPrice(venues []HuntHeatmapVenueSeries, exchange string) float64 {
	var best time.Time
	var px float64
	for _, v := range venues {
		if exchange != "" && exchange != "all" && exchange != "combined" && string(v.Exchange) != exchange {
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

// Err if range is not a hunt window — re-export helper for callers.
func ParseLiquidationLevelsRange(raw string) (HuntHeatmapSpec, error) {
	spec, err := ParseHuntHeatmapRange(raw)
	if err != nil {
		return spec, fmt.Errorf("%w: range must be 12h, 24h, 3d, or 7d", ErrInvalidArgument)
	}
	return spec, nil
}
