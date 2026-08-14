package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

// Positioning regimes from price + open interest (classic futures matrix).
const (
	RegimeLongBuildup   = "long_buildup"   // price ↑ OI ↑ — new longs
	RegimeShortBuildup  = "short_buildup"  // price ↓ OI ↑ — new shorts
	RegimeLongUnwinding = "long_unwinding" // price ↓ OI ↓ — longs close
	RegimeShortCovering = "short_covering" // price ↑ OI ↓ — shorts close
	RegimeNeutral       = "neutral"        // one or both legs flat / mixed
)

const (
	// Flat deadbands so noise does not flip the regime.
	positioningPriceFlatPct = 0.15
	positioningOIFlatPct    = 0.25
)

// PositioningWindow is one lookback of price vs OI.
type PositioningWindow struct {
	Window         string  `json:"window"` // 1h | 4h | 24h
	PriceChangePct float64 `json:"priceChangePct"`
	OIChangePct    float64 `json:"oiChangePct"`
	PriceDir       string  `json:"priceDir"` // up | down | flat
	OIDir          string  `json:"oiDir"`
	Regime         string  `json:"regime"`
	Label          string  `json:"label"` // human title
	Confidence     float64 `json:"confidence"`
}

// PositioningVenueReport is per-exchange positioning.
type PositioningVenueReport struct {
	Exchange     Exchange            `json:"exchange"`
	Symbol       string              `json:"symbol"`
	Price        float64             `json:"price"`
	OpenInterest float64             `json:"openInterestValue"`
	LongShare    float64             `json:"longShare"`
	ShortShare   float64             `json:"shortShare"`
	FundingRate  float64             `json:"fundingRate"`
	FundingPayer string              `json:"fundingPayer"`
	Primary      PositioningWindow   `json:"primary"` // preferred 4h when available
	Windows      []PositioningWindow `json:"windows"`
	Regime       string              `json:"regime"`
	Label        string              `json:"label"`
	Confidence   float64             `json:"confidence"`
	Reasons      []string            `json:"reasons"`
	Summary      string              `json:"summary"`
	Error        string              `json:"error,omitempty"`
}

// PositioningCombined is the general market read across venues.
type PositioningCombined struct {
	Regime        string   `json:"regime"`
	Label         string   `json:"label"`
	Confidence    float64  `json:"confidence"`
	Agreement     string   `json:"agreement"` // agree | mixed | single
	DominantVenue string   `json:"dominantVenue"`
	Summary       string   `json:"summary"`
	Reasons       []string `json:"reasons"`
}

// PositioningReport is the API payload.
type PositioningReport struct {
	Symbol   string                   `json:"symbol"`
	Exchange string                   `json:"exchange"`
	AsOf     time.Time                `json:"asOf"`
	Venues   []PositioningVenueReport `json:"venues"`
	Combined *PositioningCombined     `json:"combined,omitempty"`
	Note     string                   `json:"note,omitempty"`
}

// PositioningInputs feeds BuildPositioningVenue.
type PositioningInputs struct {
	Exchange Exchange
	Symbol   string
	Price    float64
	OIValue  float64
	// Same-window changes; NaN if unknown.
	Price1hPct, Price4hPct, Price24hPct float64
	OI1hPct, OI4hPct, OI24hPct          float64
	LongShare                           float64
	ShortShare                          float64
	FundingRate                         float64
}

// RegimeLabel is a short English title for the regime id.
func RegimeLabel(regime string) string {
	switch regime {
	case RegimeLongBuildup:
		return "Long buildup"
	case RegimeShortBuildup:
		return "Short buildup"
	case RegimeLongUnwinding:
		return "Long unwinding"
	case RegimeShortCovering:
		return "Short covering"
	default:
		return "Neutral / mixed"
	}
}

// ClassifyPositioningRegime maps price+OI directions to a regime.
func ClassifyPositioningRegime(pricePct, oiPct float64) (regime, priceDir, oiDir string) {
	priceDir = priceDirFromPct(pricePct)
	oiDir = oiDirFromPct(oiPct)
	switch {
	case priceDir == "up" && oiDir == "up":
		return RegimeLongBuildup, priceDir, oiDir
	case priceDir == "down" && oiDir == "up":
		return RegimeShortBuildup, priceDir, oiDir
	case priceDir == "down" && oiDir == "down":
		return RegimeLongUnwinding, priceDir, oiDir
	case priceDir == "up" && oiDir == "down":
		return RegimeShortCovering, priceDir, oiDir
	default:
		return RegimeNeutral, priceDir, oiDir
	}
}

func priceDirFromPct(pct float64) string {
	if math.IsNaN(pct) {
		return "flat"
	}
	if pct > positioningPriceFlatPct {
		return "up"
	}
	if pct < -positioningPriceFlatPct {
		return "down"
	}
	return "flat"
}

func oiDirFromPct(pct float64) string {
	if math.IsNaN(pct) {
		return "flat"
	}
	if pct > positioningOIFlatPct {
		return "up"
	}
	if pct < -positioningOIFlatPct {
		return "down"
	}
	return "flat"
}

// PositioningConfidence is 0–100 from move size (larger |price| and |OI| → higher).
func PositioningConfidence(pricePct, oiPct float64) float64 {
	if math.IsNaN(pricePct) || math.IsNaN(oiPct) {
		return 0
	}
	// ~0.5% each → mid; ~2%+ → high
	p := math.Abs(pricePct)
	o := math.Abs(oiPct)
	if p < positioningPriceFlatPct || o < positioningOIFlatPct {
		// one leg flat → soft confidence
		return clampScore(15 + (p+o)*8)
	}
	return clampScore(25 + math.Min(p, 5)*10 + math.Min(o, 8)*6)
}

// BuildPositioningWindow classifies one lookback.
func BuildPositioningWindow(window string, pricePct, oiPct float64) PositioningWindow {
	reg, pd, od := ClassifyPositioningRegime(pricePct, oiPct)
	return PositioningWindow{
		Window:         window,
		PriceChangePct: nan0(pricePct),
		OIChangePct:    nan0(oiPct),
		PriceDir:       pd,
		OIDir:          od,
		Regime:         reg,
		Label:          RegimeLabel(reg),
		Confidence:     PositioningConfidence(pricePct, oiPct),
	}
}

func nan0(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return v
}

// BuildPositioningVenue builds multi-window + primary regime for one venue.
func BuildPositioningVenue(in PositioningInputs) PositioningVenueReport {
	in.Symbol = NormalizeLiquidationSymbol(in.Symbol)
	out := PositioningVenueReport{
		Exchange:     in.Exchange,
		Symbol:       in.Symbol,
		Price:        in.Price,
		OpenInterest: in.OIValue,
		LongShare:    in.LongShare,
		ShortShare:   in.ShortShare,
		FundingRate:  in.FundingRate,
		FundingPayer: FundingPayer(in.FundingRate),
		Windows:      []PositioningWindow{},
	}
	if in.LongShare == 0 && in.ShortShare == 0 {
		in.LongShare, in.ShortShare = 0.5, 0.5
		out.LongShare, out.ShortShare = 0.5, 0.5
	}

	w1 := BuildPositioningWindow("1h", in.Price1hPct, in.OI1hPct)
	w4 := BuildPositioningWindow("4h", in.Price4hPct, in.OI4hPct)
	w24 := BuildPositioningWindow("24h", in.Price24hPct, in.OI24hPct)
	// Prefer 4h when both price and OI known; else best available.
	primary := pickPrimaryWindow(w1, w4, w24)
	out.Primary = primary
	out.Windows = []PositioningWindow{w1, w4, w24}
	out.Regime = primary.Regime
	out.Label = primary.Label
	out.Confidence = primary.Confidence
	// Funding / LS can nudge confidence and reasons.
	out.Confidence = clampScore(out.Confidence + fundingLSBoost(primary.Regime, in.FundingRate, in.LongShare))
	out.Reasons = positioningReasons(primary, in, w1, w24)
	out.Summary = positioningVenueSummary(out)
	return out
}

func pickPrimaryWindow(w1, w4, w24 PositioningWindow) PositioningWindow {
	// Prefer 4h if it has a real signal or data.
	if !math.IsNaN(w4.PriceChangePct) || w4.Confidence > 0 {
		if w4.Regime != RegimeNeutral || w4.Confidence >= w1.Confidence {
			// If 4h is neutral but 1h is strong and same family, still prefer 4h for stability
			// unless 4h has no data (all zero with flat).
			if w4.PriceDir != "flat" || w4.OIDir != "flat" || w1.Regime == RegimeNeutral {
				return w4
			}
		}
	}
	if w1.Regime != RegimeNeutral && w1.Confidence >= w24.Confidence {
		return w1
	}
	if w24.Regime != RegimeNeutral {
		return w24
	}
	if w4.Window != "" {
		return w4
	}
	return w1
}

func fundingLSBoost(regime string, funding, longShare float64) float64 {
	switch regime {
	case RegimeLongBuildup:
		if funding > 1e-12 || longShare >= 0.55 {
			return 8
		}
	case RegimeShortBuildup:
		if funding < -1e-12 || longShare <= 0.45 {
			return 8
		}
	case RegimeLongUnwinding:
		if funding > 1e-12 { // still crowded longs while unwinding
			return 5
		}
	case RegimeShortCovering:
		if funding < -1e-12 {
			return 5
		}
	}
	return 0
}

func positioningReasons(primary PositioningWindow, in PositioningInputs, w1, w24 PositioningWindow) []string {
	reasons := make([]string, 0, 5)
	reasons = append(reasons, fmt.Sprintf("%s: price %s (%s), open interest %s (%s) over %s",
		primary.Label,
		primary.PriceDir, FormatSignedPct(primary.PriceChangePct),
		primary.OIDir, FormatSignedPct(primary.OIChangePct),
		primary.Window,
	))
	// Matrix explanation
	switch primary.Regime {
	case RegimeLongBuildup:
		reasons = append(reasons, "Price rising while OI rises usually means new long positions are opening (long buildup).")
	case RegimeShortBuildup:
		reasons = append(reasons, "Price falling while OI rises usually means new short positions are opening (short buildup).")
	case RegimeLongUnwinding:
		reasons = append(reasons, "Price falling while OI falls usually means longs are closing (long unwinding).")
	case RegimeShortCovering:
		reasons = append(reasons, "Price rising while OI falls usually means shorts are closing (short covering).")
	default:
		reasons = append(reasons, "Price and/or OI are too flat or mixed for a clear buildup/unwinding label on the primary window.")
	}
	// Funding corroboration
	if in.FundingRate > 1e-12 {
		if primary.Regime == RegimeLongBuildup || primary.Regime == RegimeLongUnwinding {
			reasons = append(reasons, fmt.Sprintf("Funding is positive (%.4f%%, longs pay) — fits a long-heavy book.", in.FundingRate*100))
		} else if primary.Regime == RegimeShortBuildup {
			reasons = append(reasons, fmt.Sprintf("Funding is still positive (%.4f%%) while price+OI say short buildup — shorts may be newer or less crowded in funding yet.", in.FundingRate*100))
		}
	} else if in.FundingRate < -1e-12 {
		if primary.Regime == RegimeShortBuildup || primary.Regime == RegimeShortCovering {
			reasons = append(reasons, fmt.Sprintf("Funding is negative (%.4f%%, shorts pay) — fits a short-heavy book.", in.FundingRate*100))
		} else if primary.Regime == RegimeLongBuildup {
			reasons = append(reasons, fmt.Sprintf("Funding is negative (%.4f%%) while price+OI say long buildup — longs may be building against recent short bias.", in.FundingRate*100))
		}
	}
	// Account LS
	if in.LongShare >= 0.55 {
		reasons = append(reasons, fmt.Sprintf("Account long/short is long-crowded (%.1f%% long).", in.LongShare*100))
	} else if in.LongShare <= 0.45 && in.LongShare > 0 {
		reasons = append(reasons, fmt.Sprintf("Account long/short is short-crowded (%.1f%% short).", in.ShortShare*100))
	}
	// Cross-window note
	if w1.Regime != RegimeNeutral && w1.Regime != primary.Regime && primary.Regime != RegimeNeutral {
		reasons = append(reasons, fmt.Sprintf("Note: 1h shows %s while primary is %s — short-term may be shifting.", RegimeLabel(w1.Regime), primary.Label))
	} else if w24.Regime != RegimeNeutral && w24.Regime != primary.Regime && primary.Regime != RegimeNeutral {
		reasons = append(reasons, fmt.Sprintf("Note: 24h shows %s while primary is %s.", RegimeLabel(w24.Regime), primary.Label))
	}
	return reasons
}

func positioningVenueSummary(v PositioningVenueReport) string {
	return fmt.Sprintf("%s: %s (confidence %.0f) — price %s / OI %s on %s. %s",
		v.Exchange, v.Label, v.Confidence,
		FormatSignedPct(v.Primary.PriceChangePct), FormatSignedPct(v.Primary.OIChangePct),
		v.Primary.Window, firstReasonTail(v.Reasons))
}

func firstReasonTail(reasons []string) string {
	if len(reasons) < 2 {
		if len(reasons) == 1 {
			return reasons[0]
		}
		return ""
	}
	return reasons[1]
}

// CombinePositioningReports builds the general market direction.
func CombinePositioningReports(venues []PositioningVenueReport) *PositioningCombined {
	if len(venues) == 0 {
		return nil
	}
	if len(venues) == 1 {
		v := venues[0]
		return &PositioningCombined{
			Regime:        v.Regime,
			Label:         v.Label,
			Confidence:    v.Confidence,
			Agreement:     "single",
			DominantVenue: string(v.Exchange),
			Summary:       "Single venue. " + v.Summary,
			Reasons:       append([]string{"Only one venue available."}, v.Reasons...),
		}
	}
	// OI-weighted vote by regime; dominant venue by OI.
	type vote struct {
		w, conf float64
	}
	votes := map[string]*vote{}
	var dom Exchange
	var domOI float64
	for _, v := range venues {
		w := v.OpenInterest
		if w <= 0 {
			w = 1
		}
		if v.OpenInterest >= domOI {
			domOI = v.OpenInterest
			dom = v.Exchange
		}
		r := v.Regime
		if r == "" {
			r = RegimeNeutral
		}
		if votes[r] == nil {
			votes[r] = &vote{}
		}
		votes[r].w += w
		votes[r].conf += v.Confidence * w
	}
	type ranked struct {
		reg string
		w   float64
		c   float64
	}
	list := make([]ranked, 0, len(votes))
	var totalW float64
	for reg, v := range votes {
		list = append(list, ranked{reg: reg, w: v.w, c: v.conf})
		totalW += v.w
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].w == list[j].w {
			return list[i].c > list[j].c
		}
		return list[i].w > list[j].w
	})
	best := list[0]
	conf := 0.0
	if best.w > 0 {
		conf = clampScore(best.c / best.w)
	}
	agreement := "mixed"
	if best.w/totalW >= 0.85 || (len(list) == 1) {
		agreement = "agree"
	} else if list[0].reg == RegimeNeutral {
		agreement = "mixed"
	} else if len(list) >= 2 && list[1].reg != RegimeNeutral && list[1].w > totalW*0.25 {
		agreement = "mixed"
	} else {
		agreement = "agree"
	}
	// If venues disagree on non-neutral, mark mixed and soften confidence.
	nonNeutral := map[string]float64{}
	for _, v := range venues {
		if v.Regime != RegimeNeutral {
			nonNeutral[v.Regime] += v.OpenInterest
		}
	}
	if len(nonNeutral) > 1 {
		agreement = "mixed"
		conf = clampScore(conf * 0.75)
	}

	reasons := []string{
		fmt.Sprintf("General market: %s (OI-weighted across Binance and Bybit).", RegimeLabel(best.reg)),
	}
	for _, v := range venues {
		reasons = append(reasons, fmt.Sprintf("%s: %s (%.0f conf, OI %s)",
			v.Exchange, v.Label, v.Confidence, formatQty(v.OpenInterest)))
	}
	if agreement == "mixed" {
		reasons = append(reasons, "Venues do not fully agree — treat the combined label as a soft read.")
	} else {
		reasons = append(reasons, "Venues broadly agree on the primary regime.")
	}

	out := &PositioningCombined{
		Regime:        best.reg,
		Label:         RegimeLabel(best.reg),
		Confidence:    conf,
		Agreement:     agreement,
		DominantVenue: string(dom),
		Reasons:       reasons,
	}
	out.Summary = fmt.Sprintf("Market: %s (confidence %.0f, %s). Dominant OI: %s.",
		out.Label, out.Confidence, out.Agreement, out.DominantVenue)
	return out
}

// PriceChangePctFromCloses is (last-first)/first*100 for chronological closes.
func PriceChangePctFromCloses(closes []float64) float64 {
	if len(closes) < 2 {
		return math.NaN()
	}
	first := closes[0]
	last := closes[len(closes)-1]
	if first <= 0 || last <= 0 {
		return math.NaN()
	}
	return (last - first) / first * 100
}

// ClosesFromCandles returns oldest→newest closes from candle list (any order).
func ClosesFromCandles(candles []Candle) []float64 {
	if len(candles) == 0 {
		return nil
	}
	cp := append([]Candle(nil), candles...)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].OpenTime.Before(cp[j].OpenTime)
	})
	out := make([]float64, 0, len(cp))
	for _, c := range cp {
		px, err := strconv.ParseFloat(c.Close, 64)
		if err != nil || px <= 0 {
			continue
		}
		out = append(out, px)
	}
	return out
}

// PriceChangeOverBars uses last close vs close N bars earlier (1h candles → N hours).
func PriceChangeOverBars(closes []float64, barsBack int) float64 {
	if barsBack < 1 || len(closes) < barsBack+1 {
		return math.NaN()
	}
	// last vs barsBack ago
	first := closes[len(closes)-1-barsBack]
	last := closes[len(closes)-1]
	if first <= 0 {
		return math.NaN()
	}
	return (last - first) / first * 100
}

// ParseTickerPriceChangePct parses ticker 24h percent string.
func ParseTickerPriceChangePct(raw string) float64 {
	if raw == "" {
		return math.NaN()
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return math.NaN()
	}
	return v
}
