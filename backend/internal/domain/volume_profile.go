package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	VolumeProfileWindow1h  = "1h"
	VolumeProfileWindow4h  = "4h"
	VolumeProfileWindow24h = "24h"
	VolumeProfileWindow7d  = "7d"
	VolumeProfileWindow30d = "30d"

	DefaultVolumeProfileWindow = VolumeProfileWindow24h
	DefaultValueAreaPct        = 70.0
	MaxVolumeProfileRange      = 30 * 24 * time.Hour
	MaxVolumeProfileBins       = 200
	MinVolumeProfileBins       = 8
	TargetVolumeProfileBins    = 64
	MaxVolumeProfilePages      = 6
	VolumeProfilePageLimit     = 1000

	VolumeProfileVsAbove   = "above"
	VolumeProfileVsInside  = "inside"
	VolumeProfileVsBelow   = "below"
	VolumeProfileVsUnknown = "unknown"
)

// VolumeProfileWindows are the named lookbacks the API accepts.
var VolumeProfileWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{VolumeProfileWindow1h, time.Hour},
	{VolumeProfileWindow4h, 4 * time.Hour},
	{VolumeProfileWindow24h, 24 * time.Hour},
	{VolumeProfileWindow7d, 7 * 24 * time.Hour},
	{VolumeProfileWindow30d, 30 * 24 * time.Hour},
}

// VolumeProfileBar is one candle flattened for binning.
type VolumeProfileBar struct {
	Time         time.Time
	High         float64
	Low          float64
	Close        float64
	Volume       float64 // quote notional
	BuyVolume    float64
	SellVolume   float64
	BuySellKnown bool
}

// VolumeProfileShare is one venue's contribution to a combined bin.
type VolumeProfileShare struct {
	Exchange Exchange
	Volume   float64
	SharePct float64
}

// VolumeProfileBin is traded quote volume in one price row.
type VolumeProfileBin struct {
	Price       float64 // inclusive low of the row
	High        float64 // exclusive top
	Volume      float64
	BuyVolume   float64
	SellVolume  float64
	BuyPct      float64
	SharePct    float64
	IsPoc       bool
	InValueArea bool
	IsHvn       bool
	Shares      []VolumeProfileShare
}

// VolumeProfilePOC is the price row with the most volume.
type VolumeProfilePOC struct {
	Price      float64
	High       float64
	Volume     float64
	BuyVolume  float64
	SellVolume float64
	SharePct   float64
}

// VolumeProfileValueArea is the contiguous block around the POC that holds
// about DefaultValueAreaPct of total volume.
type VolumeProfileValueArea struct {
	Low       float64
	High      float64
	Volume    float64
	VolumePct float64
	BinCount  int
}

// VolumeProfileVenue is one exchange's profile (or combined).
type VolumeProfileVenue struct {
	Exchange       Exchange
	Symbol         string
	From           time.Time
	To             time.Time
	Interval       string
	TickSize       float64
	LastPrice      float64
	High           float64
	Low            float64
	TotalVolume    float64
	BuyVolume      float64
	SellVolume     float64
	BuySellKnown   bool
	BuySellPartial bool
	LastVsArea     string
	POC            VolumeProfilePOC
	ValueArea      VolumeProfileValueArea
	Bins           []VolumeProfileBin
	BarCount       int
	Summary        string
	Error          string
}

// VolumeProfileReport is the API result.
type VolumeProfileReport struct {
	Symbol   string
	Exchange string
	Window   string
	From     time.Time
	To       time.Time
	AsOf     time.Time
	Venues   []VolumeProfileVenue
	Combined *VolumeProfileVenue
	Summary  string
	Note     string
}

// ParseVolumeProfileWindow accepts 1h / 4h / 24h / 7d / 30d (empty = 24h).
func ParseVolumeProfileWindow(raw string) (string, time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = DefaultVolumeProfileWindow
	}
	for _, w := range VolumeProfileWindows {
		if w.ID == s {
			return w.ID, w.Dur, nil
		}
	}
	return "", 0, fmt.Errorf("%w: window must be 1h, 4h, 24h, 7d, or 30d", ErrInvalidArgument)
}

// ResolveVolumeProfileRange picks [from, to] from an explicit range or a named window.
// start/end win when either is set. Max span is 30 days.
func ResolveVolumeProfileRange(window string, start, end *time.Time, now time.Time) (from, to time.Time, windowID string, err error) {
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
		from = to.Add(-24 * time.Hour)
		if hasStart {
			from = start.UTC()
		}
		if !to.After(from) {
			return time.Time{}, time.Time{}, "", fmt.Errorf("%w: endTime must be after startTime", ErrInvalidArgument)
		}
		if to.Sub(from) > MaxVolumeProfileRange {
			return time.Time{}, time.Time{}, "", fmt.Errorf("%w: range must be <= 30d", ErrInvalidArgument)
		}
		return from, to, "custom", nil
	}
	id, dur, err := ParseVolumeProfileWindow(window)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	return now.Add(-dur), now, id, nil
}

// ProfileBarInterval picks a candle size so a range stays near a few thousand bars.
func ProfileBarInterval(dur time.Duration) CandleInterval {
	if dur <= 0 {
		return Interval5m
	}
	switch {
	case dur <= 36*time.Hour:
		return Interval1m
	case dur <= 8*24*time.Hour:
		return Interval5m
	default:
		return Interval15m
	}
}

// NiceTickSize snaps a raw step to 1 / 2 / 5 × 10^n.
func NiceTickSize(raw float64) float64 {
	if raw <= 0 || math.IsNaN(raw) || math.IsInf(raw, 0) {
		return 0
	}
	exp := math.Floor(math.Log10(raw))
	base := math.Pow(10, exp)
	n := raw / base
	switch {
	case n <= 1:
		return 1 * base
	case n <= 2:
		return 2 * base
	case n <= 5:
		return 5 * base
	default:
		return 10 * base
	}
}

// AutoTickSize chooses a row width so the range is about TargetVolumeProfileBins rows.
func AutoTickSize(low, high float64) float64 {
	span := high - low
	if span <= 0 {
		if low > 0 {
			return NiceTickSize(low * 0.0005)
		}
		return 0
	}
	tick := NiceTickSize(span / float64(TargetVolumeProfileBins))
	return ClampVolumeProfileTick(low, high, tick)
}

// ClampVolumeProfileTick widens tick until the range is at most MaxVolumeProfileBins.
func ClampVolumeProfileTick(low, high, tick float64) float64 {
	span := high - low
	if span <= 0 {
		if tick > 0 {
			return tick
		}
		return 0
	}
	if tick <= 0 {
		tick = NiceTickSize(span / float64(TargetVolumeProfileBins))
	}
	for i := 0; i < 16 && tick > 0; i++ {
		n := int(math.Floor(span/tick + 1.5))
		if n <= MaxVolumeProfileBins {
			return tick
		}
		next := NiceTickSize(tick * 2)
		if next <= tick {
			return tick
		}
		tick = next
	}
	return tick
}

// VolumeProfileBarsFromCandles maps klines to profile bars (quote volume).
// Buy = taker-buy quote when present; sell = remainder.
func VolumeProfileBarsFromCandles(candles []Candle) []VolumeProfileBar {
	out := make([]VolumeProfileBar, 0, len(candles))
	for _, c := range candles {
		high, err1 := parseFloat(c.High)
		low, err2 := parseFloat(c.Low)
		if err1 != nil || err2 != nil || high <= 0 || low <= 0 {
			continue
		}
		if low > high {
			low, high = high, low
		}
		quote, err := parseFloat(c.QuoteVolume)
		if err != nil || quote <= 0 {
			continue
		}
		bar := VolumeProfileBar{
			Time: c.OpenTime.UTC(), High: high, Low: low, Volume: quote,
		}
		if closePx, err := parseFloat(c.Close); err == nil && closePx > 0 {
			bar.Close = closePx
		}
		if c.TakerBuyQuote != "" {
			buy, err := parseFloat(c.TakerBuyQuote)
			if err == nil && buy >= 0 {
				if buy > quote {
					buy = quote
				}
				bar.BuyVolume = buy
				bar.SellVolume = quote - buy
				bar.BuySellKnown = true
			}
		}
		out = append(out, bar)
	}
	return out
}

// BuildVolumeProfile distributes each bar's quote volume across price rows.
func BuildVolumeProfile(ex Exchange, symbol string, bars []VolumeProfileBar, tick, last float64, from, to time.Time, interval CandleInterval) VolumeProfileVenue {
	out := VolumeProfileVenue{
		Exchange:   ex,
		Symbol:     symbol,
		From:       from.UTC(),
		To:         to.UTC(),
		Interval:   string(interval),
		TickSize:   tick,
		LastPrice:  last,
		Bins:       []VolumeProfileBin{},
		LastVsArea: VolumeProfileVsUnknown,
	}
	if last <= 0 {
		for i := len(bars) - 1; i >= 0; i-- {
			if bars[i].Close > 0 {
				last = bars[i].Close
				out.LastPrice = last
				break
			}
		}
	}
	if tick <= 0 || len(bars) == 0 {
		out.Error = "not enough candles in this range"
		out.Summary = out.Error
		return out
	}

	type acc struct {
		vol, buy, sell float64
		known          float64
	}
	m := map[int64]*acc{}
	var rangeLow, rangeHigh float64
	var knownVol, unknownVol float64
	for _, b := range bars {
		if b.Volume <= 0 || b.High <= 0 || b.Low <= 0 {
			continue
		}
		out.BarCount++
		if rangeLow == 0 || b.Low < rangeLow {
			rangeLow = b.Low
		}
		if b.High > rangeHigh {
			rangeHigh = b.High
		}
		if b.BuySellKnown {
			knownVol += b.Volume
			out.BuyVolume += b.BuyVolume
			out.SellVolume += b.SellVolume
		} else {
			unknownVol += b.Volume
		}
		first, lastIdx := barBinRange(b.Low, b.High, tick)
		n := lastIdx - first + 1
		if n <= 0 {
			continue
		}
		each := b.Volume / float64(n)
		buyEach, sellEach := 0.0, 0.0
		if b.BuySellKnown {
			buyEach = b.BuyVolume / float64(n)
			sellEach = b.SellVolume / float64(n)
		}
		for idx := first; idx <= lastIdx; idx++ {
			cur := m[idx]
			if cur == nil {
				cur = &acc{}
				m[idx] = cur
			}
			cur.vol += each
			cur.buy += buyEach
			cur.sell += sellEach
			if b.BuySellKnown {
				cur.known += each
			}
		}
	}
	out.Low, out.High = rangeLow, rangeHigh
	out.BuySellKnown = knownVol > 0
	out.BuySellPartial = knownVol > 0 && unknownVol > 0
	if len(m) == 0 {
		out.Error = "not enough candles in this range"
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

// CombineVolumeProfiles adds Binance and Bybit rows at the same tick.
func CombineVolumeProfiles(symbol string, venues []VolumeProfileVenue, tick, last float64, from, to time.Time, interval CandleInterval) *VolumeProfileVenue {
	out := &VolumeProfileVenue{
		Exchange:   "all",
		Symbol:     symbol,
		From:       from.UTC(),
		To:         to.UTC(),
		Interval:   string(interval),
		TickSize:   tick,
		LastPrice:  last,
		Bins:       []VolumeProfileBin{},
		LastVsArea: VolumeProfileVsUnknown,
	}
	var usable []VolumeProfileVenue
	for _, v := range venues {
		if v.Error != "" || len(v.Bins) == 0 {
			continue
		}
		usable = append(usable, v)
	}
	if len(usable) == 0 {
		out.Error = "not enough candles in this range"
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
	type acc struct {
		vol, buy, sell float64
		shares         map[Exchange]float64
	}
	m := map[int64]*acc{}
	var known, unknown float64
	for _, v := range usable {
		out.BarCount += v.BarCount
		if out.Low == 0 || (v.Low > 0 && v.Low < out.Low) {
			out.Low = v.Low
		}
		if v.High > out.High {
			out.High = v.High
		}
		if v.BuySellKnown {
			known += v.BuyVolume + v.SellVolume
			out.BuyVolume += v.BuyVolume
			out.SellVolume += v.SellVolume
			if v.BuySellPartial || v.TotalVolume > v.BuyVolume+v.SellVolume+1e-9 {
				unknown += v.TotalVolume - (v.BuyVolume + v.SellVolume)
			}
		} else {
			unknown += v.TotalVolume
		}
		for _, b := range v.Bins {
			idx := priceBinIndex(b.Price, tick)
			cur := m[idx]
			if cur == nil {
				cur = &acc{shares: map[Exchange]float64{}}
				m[idx] = cur
			}
			cur.vol += b.Volume
			cur.buy += b.BuyVolume
			cur.sell += b.SellVolume
			cur.shares[v.Exchange] += b.Volume
		}
	}
	out.BuySellKnown = known > 0
	out.BuySellPartial = known > 0 && unknown > 0
	if len(m) == 0 {
		out.Error = "not enough candles in this range"
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
			Price: binLow(idx, tick), High: binLow(idx+1, tick),
			Volume: a.vol, BuyVolume: a.buy, SellVolume: a.sell,
		}
		if side := a.buy + a.sell; side > 0 {
			row.BuyPct = a.buy / side * 100
		}
		if len(usable) > 1 {
			shares := make([]VolumeProfileShare, 0, len(usable))
			for _, v := range usable {
				sh := VolumeProfileShare{Exchange: v.Exchange, Volume: a.shares[v.Exchange]}
				if a.vol > 0 {
					sh.SharePct = sh.Volume / a.vol * 100
				}
				shares = append(shares, sh)
			}
			sort.Slice(shares, func(i, j int) bool {
				return string(shares[i].Exchange) < string(shares[j].Exchange)
			})
			row.Shares = shares
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
	out.Summary = ExplainVolumeProfileVenue(*out)
	return out
}

// ExplainVolumeProfileVenue writes a short POC + value-area line.
func ExplainVolumeProfileVenue(v VolumeProfileVenue) string {
	if v.Error != "" {
		return v.Error
	}
	if v.POC.Volume <= 0 {
		return prettyBase(v.Symbol) + ": not enough volume for a profile."
	}
	name := prettyBase(v.Symbol)
	where := "inside"
	switch v.LastVsArea {
	case VolumeProfileVsAbove:
		where = "above"
	case VolumeProfileVsBelow:
		where = "below"
	case VolumeProfileVsUnknown:
		where = "relative to"
	}
	head := fmt.Sprintf("%s volume clustered at %s (POC, %s%% of volume). About %.0f%% of volume sat between %s and %s.",
		name, formatQty(v.POC.Price), formatFixed(v.POC.SharePct, 0),
		DefaultValueAreaPct, formatQty(v.ValueArea.Low), formatQty(v.ValueArea.High))
	if v.BuySellKnown && v.POC.BuyVolume+v.POC.SellVolume > 0 {
		switch TakerDominant(v.POC.BuyVolume, v.POC.SellVolume) {
		case TakerSideBuy:
			head += " More market buys than sells at the POC."
		case TakerSideSell:
			head += " More market sells than buys at the POC."
		default:
			head += " Buy and sell were balanced at the POC."
		}
	}
	if v.LastPrice > 0 && v.LastVsArea != VolumeProfileVsUnknown {
		head += fmt.Sprintf(" Last %s is %s the value area.", formatQty(v.LastPrice), where)
	}
	if len(v.Bins) > 0 && len(v.Bins[0].Shares) > 0 {
		parts := make([]string, 0, len(v.Bins[0].Shares))
		// Use POC shares when present.
		var pocShares []VolumeProfileShare
		for _, b := range v.Bins {
			if b.IsPoc {
				pocShares = b.Shares
				break
			}
		}
		for _, s := range pocShares {
			parts = append(parts, fmt.Sprintf("%s %s%%", s.Exchange, formatFixed(s.SharePct, 0)))
		}
		if len(parts) > 0 {
			head += " At the POC: " + strings.Join(parts, ", ") + "."
		}
	}
	return head
}

// ExplainVolumeProfileReport prefers combined, else the first venue.
func ExplainVolumeProfileReport(r VolumeProfileReport) string {
	if r.Combined != nil && r.Combined.Summary != "" && r.Combined.Error == "" {
		return "Combined: " + r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Summary != "" && v.Error == "" {
			return string(v.Exchange) + ": " + v.Summary
		}
	}
	for _, v := range r.Venues {
		if v.Summary != "" {
			return v.Summary
		}
	}
	return "No volume profile yet."
}

func annotateVolumeProfile(bins []VolumeProfileBin, last, total float64) {
	if len(bins) == 0 {
		return
	}
	poc := 0
	for i := 1; i < len(bins); i++ {
		if bins[i].Volume > bins[poc].Volume {
			poc = i
			continue
		}
		if bins[i].Volume == bins[poc].Volume && last > 0 {
			di := math.Abs((bins[i].Price+bins[i].High)/2 - last)
			dp := math.Abs((bins[poc].Price+bins[poc].High)/2 - last)
			if di < dp {
				poc = i
			}
		}
	}
	bins[poc].IsPoc = true
	lo, hi, _ := valueAreaBounds(bins, poc, total)
	for i := lo; i <= hi; i++ {
		bins[i].InValueArea = true
	}
	threshold := bins[poc].Volume * 0.5
	for i := range bins {
		if bins[i].IsPoc || bins[i].Volume < threshold || bins[i].Volume <= 0 {
			continue
		}
		left, right := 0.0, 0.0
		if i > 0 {
			left = bins[i-1].Volume
		}
		if i+1 < len(bins) {
			right = bins[i+1].Volume
		}
		if bins[i].Volume >= left && bins[i].Volume >= right && (bins[i].Volume > left || bins[i].Volume > right) {
			bins[i].IsHvn = true
		}
	}
}

func valueAreaBounds(bins []VolumeProfileBin, poc int, total float64) (lo, hi int, covered float64) {
	lo, hi = poc, poc
	covered = bins[poc].Volume
	target := total * (DefaultValueAreaPct / 100)
	for covered < target-1e-9 {
		canLo := lo > 0
		canHi := hi+1 < len(bins)
		if !canLo && !canHi {
			break
		}
		below, above := 0.0, 0.0
		if canLo {
			below = bins[lo-1].Volume
		}
		if canHi {
			above = bins[hi+1].Volume
		}
		switch {
		case canLo && canHi && above > below:
			hi++
			covered += bins[hi].Volume
		case canLo && canHi && below > above:
			lo--
			covered += bins[lo].Volume
		case canLo && canHi:
			hi++
			covered += bins[hi].Volume
			lo--
			covered += bins[lo].Volume
		case canHi:
			hi++
			covered += bins[hi].Volume
		default:
			lo--
			covered += bins[lo].Volume
		}
	}
	return lo, hi, covered
}

func volumeProfileStats(bins []VolumeProfileBin) (VolumeProfilePOC, VolumeProfileValueArea) {
	var poc VolumeProfilePOC
	var va VolumeProfileValueArea
	if len(bins) == 0 {
		return poc, va
	}
	lo, hi := 0, 0
	found := false
	for i, b := range bins {
		if b.IsPoc {
			poc = VolumeProfilePOC{
				Price: b.Price, High: b.High, Volume: b.Volume,
				BuyVolume: b.BuyVolume, SellVolume: b.SellVolume, SharePct: b.SharePct,
			}
		}
		if b.InValueArea {
			if !found {
				lo, hi = i, i
				found = true
			} else {
				hi = i
			}
			va.Volume += b.Volume
			va.BinCount++
		}
	}
	if found {
		va.Low = bins[lo].Price
		va.High = bins[hi].High
	}
	var total float64
	for _, b := range bins {
		total += b.Volume
	}
	if total > 0 {
		va.VolumePct = va.Volume / total * 100
	}
	return poc, va
}

func lastVsValueArea(last float64, va VolumeProfileValueArea) string {
	if last <= 0 || va.Low <= 0 || va.High <= 0 {
		return VolumeProfileVsUnknown
	}
	switch {
	case last < va.Low:
		return VolumeProfileVsBelow
	case last > va.High:
		return VolumeProfileVsAbove
	default:
		return VolumeProfileVsInside
	}
}

func priceBinIndex(price, tick float64) int64 {
	if tick <= 0 {
		return 0
	}
	return int64(math.Floor(price/tick + 1e-12))
}

func binLow(idx int64, tick float64) float64 {
	return float64(idx) * tick
}

func barBinRange(low, high, tick float64) (first, last int64) {
	first = priceBinIndex(low, tick)
	last = priceBinIndex(high, tick)
	if last < first {
		last = first
	}
	return first, last
}
