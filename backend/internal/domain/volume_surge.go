package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	VolumeSurgeWindow5m  = "5m"
	VolumeSurgeWindow15m = "15m"
	VolumeSurgeWindow1h  = "1h"

	VolumeSurgeTypical  = "typical"
	VolumeSurgeElevated = "elevated"
	VolumeSurgeHigh     = "high"
	VolumeSurgeExtreme  = "extreme"

	VolumeSurgeLookbackBars = 300 // ~25h of 5m
	VolumeSurgeScanDefault  = 30
	VolumeSurgeScanMax      = 50
	DefaultVolumeSurgeMin   = 2.0
	MaxVolumeSurgeHits      = 40
)

// VolumeSurgeWindows are the current-vs-typical lookbacks.
var VolumeSurgeWindows = []struct {
	ID       string
	Step     time.Duration
	MinPrior int
}{
	{VolumeSurgeWindow5m, 5 * time.Minute, 12},
	{VolumeSurgeWindow15m, 15 * time.Minute, 8},
	{VolumeSurgeWindow1h, time.Hour, 6},
}

// VolumeBar is one time bucket of quote volume (and taker buy/sell when known).
type VolumeBar struct {
	Time         time.Time
	Volume       float64
	BuyVolume    float64
	SellVolume   float64
	BuySellKnown bool
}

// VolumeSurgeWindow is current volume versus that coin's own typical for one size.
type VolumeSurgeWindow struct {
	Window      string
	Current     float64
	Typical     float64 // median of prior windows
	Ratio       float64 // current / typical
	BuyCurrent  float64
	BuyTypical  float64
	BuyRatio    float64
	SellCurrent float64
	SellTypical float64
	SellRatio   float64
	Dominant    string // buy | sell | balanced | unknown
	Grade       string
	SampleBars  int
	Complete    bool
}

// VolumeSurgeVenue is one venue's surge read.
type VolumeSurgeVenue struct {
	Exchange     Exchange
	Symbol       string
	BuySellKnown bool
	Windows      []VolumeSurgeWindow
	MaxRatio     float64
	Hottest      string // window with the highest ratio
	Summary      string
	Error        string
}

// VolumeSurgeReport is the one-coin API result.
type VolumeSurgeReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Venues   []VolumeSurgeVenue
	Summary  string
	Note     string
}

// VolumeSurgeHit is one coin in a universe scan.
type VolumeSurgeHit struct {
	Symbol       string
	Exchange     Exchange
	BuySellKnown bool
	Windows      []VolumeSurgeWindow
	MaxRatio     float64
	Hottest      string
	Grade        string
	Dominant     string
	Summary      string
}

// VolumeSurgeScan is the ranked list of coins above a min ratio.
type VolumeSurgeScan struct {
	Exchange    string
	Quote       string
	MinRatio    float64
	SymbolLimit int
	AsOf        time.Time
	Hits        []VolumeSurgeHit
	Summary     string
	Note        string
}

// VolumeBarsFromCandles maps klines to 5m-or-native quote volume bars.
func VolumeBarsFromCandles(candles []Candle) []VolumeBar {
	out := make([]VolumeBar, 0, len(candles))
	for _, c := range candles {
		if c.OpenTime.IsZero() {
			continue
		}
		q, err := parseFloat(c.QuoteVolume)
		if err != nil || q < 0 {
			continue
		}
		bar := VolumeBar{Time: c.OpenTime.UTC(), Volume: q}
		if c.TakerBuyQuote != "" {
			buy, err := parseFloat(c.TakerBuyQuote)
			if err == nil && buy >= 0 {
				if q > 0 && buy > q {
					buy = q
				}
				bar.BuyVolume = buy
				bar.SellVolume = q - buy
				bar.BuySellKnown = true
			}
		}
		out = append(out, bar)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// ResampleVolumeBars folds bars into a coarser step (e.g. 5m → 15m).
func ResampleVolumeBars(in []VolumeBar, step time.Duration) []VolumeBar {
	if step <= 0 {
		return in
	}
	acc := map[int64]*VolumeBar{}
	for _, b := range in {
		if b.Time.IsZero() {
			continue
		}
		ms := TruncateToBucket(b.Time.UTC(), step).UnixMilli()
		cur := acc[ms]
		if cur == nil {
			cur = &VolumeBar{Time: time.UnixMilli(ms).UTC()}
			acc[ms] = cur
		}
		cur.Volume += b.Volume
		cur.BuyVolume += b.BuyVolume
		cur.SellVolume += b.SellVolume
		if b.BuySellKnown {
			cur.BuySellKnown = true
		}
	}
	out := make([]VolumeBar, 0, len(acc))
	for _, b := range acc {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// MeasureVolumeSurge compares the latest 5m / 15m / 1h bucket to the median of priors.
// Input should be fine bars (typically 5m). The current (last) bucket is excluded from typical.
func MeasureVolumeSurge(bars []VolumeBar) []VolumeSurgeWindow {
	out := make([]VolumeSurgeWindow, 0, len(VolumeSurgeWindows))
	for _, spec := range VolumeSurgeWindows {
		series := bars
		if spec.Step != 5*time.Minute {
			series = ResampleVolumeBars(bars, spec.Step)
		}
		out = append(out, measureSurgeWindow(series, spec.ID, spec.MinPrior))
	}
	return out
}

// BuildVolumeSurgeVenue writes per-window ratios and a short summary.
func BuildVolumeSurgeVenue(ex Exchange, symbol string, bars []VolumeBar) VolumeSurgeVenue {
	out := VolumeSurgeVenue{Exchange: ex, Symbol: symbol, Windows: []VolumeSurgeWindow{}}
	if len(bars) == 0 {
		out.Error = "not enough candles in this range"
		out.Summary = out.Error
		return out
	}
	out.Windows = MeasureVolumeSurge(bars)
	for _, w := range out.Windows {
		if w.BuyCurrent > 0 || w.SellCurrent > 0 || w.BuyTypical > 0 || w.SellTypical > 0 {
			out.BuySellKnown = true
		}
		if w.Ratio >= out.MaxRatio {
			out.MaxRatio = w.Ratio
			out.Hottest = w.Window
		}
	}
	out.Summary = ExplainVolumeSurgeVenue(out)
	return out
}

// VolumeSurgeGrade maps current/typical to a word.
func VolumeSurgeGrade(ratio float64) string {
	switch {
	case ratio >= 6:
		return VolumeSurgeExtreme
	case ratio >= 3:
		return VolumeSurgeHigh
	case ratio >= 1.5:
		return VolumeSurgeElevated
	default:
		return VolumeSurgeTypical
	}
}

// ExplainVolumeSurgeVenue writes a hottest-window line with buy/sell split.
func ExplainVolumeSurgeVenue(v VolumeSurgeVenue) string {
	if v.Error != "" {
		return v.Error
	}
	var hot VolumeSurgeWindow
	for _, w := range v.Windows {
		if w.Window == v.Hottest {
			hot = w
			break
		}
	}
	if hot.Window == "" && len(v.Windows) > 0 {
		hot = v.Windows[0]
	}
	name := prettyBase(v.Symbol)
	if hot.Typical <= 0 || !hot.Complete {
		return name + ": not enough history to judge volume versus typical."
	}
	head := fmt.Sprintf("%s %s volume is %.1fx typical (%s vs %s, %s).",
		name, hot.Window, hot.Ratio, formatQty(hot.Current), formatQty(hot.Typical), hot.Grade)
	if hot.Dominant == TakerSideBuy || hot.Dominant == TakerSideSell {
		head += fmt.Sprintf(" Increase is mostly %s (buy %.1fx, sell %.1fx).",
			hot.Dominant, hot.BuyRatio, hot.SellRatio)
	}
	return head
}

// ExplainVolumeSurgeReport prefers the hottest venue.
func ExplainVolumeSurgeReport(r VolumeSurgeReport) string {
	var best VolumeSurgeVenue
	for _, v := range r.Venues {
		if v.Error != "" {
			continue
		}
		if best.Symbol == "" || v.MaxRatio > best.MaxRatio {
			best = v
		}
	}
	if best.Summary == "" {
		return "No volume surge read yet."
	}
	if len(r.Venues) > 1 && best.Exchange != "" {
		return string(best.Exchange) + ": " + best.Summary
	}
	return best.Summary
}

// ExplainVolumeSurgeScan writes how many coins are running hot.
func ExplainVolumeSurgeScan(s VolumeSurgeScan) string {
	if len(s.Hits) == 0 {
		return fmt.Sprintf("No coins at %.1fx typical volume or more among the top %d by 24h volume.",
			s.MinRatio, s.SymbolLimit)
	}
	top := s.Hits[0]
	return fmt.Sprintf("%d coin(s) well above typical volume. Hottest: %s at %.1fx (%s %s).",
		len(s.Hits), prettyBase(top.Symbol), top.MaxRatio, top.Hottest, top.Grade)
}

func measureSurgeWindow(bars []VolumeBar, window string, minPrior int) VolumeSurgeWindow {
	out := VolumeSurgeWindow{Window: window, Dominant: "unknown", Grade: VolumeSurgeTypical}
	if len(bars) < 2 {
		return out
	}
	cur := bars[len(bars)-1]
	priors := bars[:len(bars)-1]
	var vols, buys, sells []float64
	var known int
	for _, b := range priors {
		if b.Volume > 0 {
			vols = append(vols, b.Volume)
		}
		if b.BuySellKnown {
			known++
			buys = append(buys, b.BuyVolume)
			sells = append(sells, b.SellVolume)
		}
	}
	out.Current = cur.Volume
	out.Typical = medianFloat(vols)
	out.SampleBars = len(vols)
	out.Complete = len(vols) >= minPrior && out.Typical > 0
	if out.Typical > 0 {
		out.Ratio = out.Current / out.Typical
	}
	out.Grade = VolumeSurgeGrade(out.Ratio)
	if cur.BuySellKnown && known >= minPrior/2 {
		out.BuyCurrent = cur.BuyVolume
		out.SellCurrent = cur.SellVolume
		out.BuyTypical = medianFloat(buys)
		out.SellTypical = medianFloat(sells)
		if out.BuyTypical > 0 {
			out.BuyRatio = out.BuyCurrent / out.BuyTypical
		}
		if out.SellTypical > 0 {
			out.SellRatio = out.SellCurrent / out.SellTypical
		}
		out.Dominant = surgeDominant(out.BuyRatio, out.SellRatio, out.BuyCurrent, out.SellCurrent, out.BuyTypical, out.SellTypical)
	}
	return out
}

func surgeDominant(buyRatio, sellRatio, buyNow, sellNow, buyTyp, sellTyp float64) string {
	if buyTyp <= 0 && sellTyp <= 0 {
		return "unknown"
	}
	extraBuy := buyNow - buyTyp
	if extraBuy < 0 {
		extraBuy = 0
	}
	extraSell := sellNow - sellTyp
	if extraSell < 0 {
		extraSell = 0
	}
	if extraBuy+extraSell <= 0 {
		return TakerDominant(buyNow, sellNow)
	}
	// Prefer the side that added more than its own normal.
	if buyRatio >= sellRatio*1.25 && extraBuy > extraSell*0.8 {
		return TakerSideBuy
	}
	if sellRatio >= buyRatio*1.25 && extraSell > extraBuy*0.8 {
		return TakerSideSell
	}
	return TakerDominant(extraBuy, extraSell)
}
