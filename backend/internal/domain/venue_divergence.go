package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	AlignSame     = "same"
	AlignOpposite = "opposite"
	AlignMixed    = "mixed"
	AlignUnknown  = "unknown"

	LeanBullish = "bullish" // long buildup or short covering
	LeanBearish = "bearish" // short buildup or long unwinding
	LeanMixed   = "mixed"
)

// VenueSignalDiff is one compared metric.
type VenueSignalDiff struct {
	Metric       string `json:"metric"` // oi_change | funding | crowding | positioning
	Label        string `json:"label"`
	Binance      string `json:"binance"`
	Bybit        string `json:"bybit"`
	Alignment    string `json:"alignment"` // same | opposite | mixed
	Important    bool   `json:"important"`
	WhyItMatters string `json:"whyItMatters"`
}

// VenueDivergenceReport is Binance vs Bybit for one coin.
type VenueDivergenceReport struct {
	Symbol          string            `json:"symbol"`
	AsOf            time.Time         `json:"asOf"`
	Alignment       string            `json:"alignment"` // same | opposite | mixed
	BinanceLean     string            `json:"binanceLean"`
	BybitLean       string            `json:"bybitLean"`
	Important       bool              `json:"important"`
	Title           string            `json:"title"`
	Summary         string            `json:"summary"`
	Diffs           []VenueSignalDiff `json:"diffs"`
	BinanceRegime   string            `json:"binanceRegime"`
	BybitRegime     string            `json:"bybitRegime"`
	BinanceOIChange string            `json:"binanceOiChange,omitempty"`
	BybitOIChange   string            `json:"bybitOiChange,omitempty"`
	Note            string            `json:"note,omitempty"`
}

// VenueLeanFromRegime maps positioning to a market lean.
func VenueLeanFromRegime(regime string) string {
	switch regime {
	case RegimeLongBuildup, RegimeShortCovering:
		return LeanBullish
	case RegimeShortBuildup, RegimeLongUnwinding:
		return LeanBearish
	default:
		return LeanMixed
	}
}

func leanFromCrowdingAndOI(crowded, oiDir string) string {
	if oiDir == "up" && crowded == SqueezeSideLong {
		return LeanBullish
	}
	if oiDir == "up" && crowded == SqueezeSideShort {
		return LeanBearish
	}
	if oiDir == "down" && crowded == SqueezeSideLong {
		return LeanBearish // longs leaving
	}
	if oiDir == "down" && crowded == SqueezeSideShort {
		return LeanBullish // shorts leaving
	}
	return LeanMixed
}

// CompareVenuePositioning contrasts Binance and Bybit positioning reports.
func CompareVenuePositioning(symbol string, bin, byb PositioningVenueReport, now time.Time) VenueDivergenceReport {
	symbol = NormalizeLiquidationSymbol(symbol)
	out := VenueDivergenceReport{
		Symbol:        symbol,
		AsOf:          now.UTC(),
		BinanceRegime: bin.Regime,
		BybitRegime:   byb.Regime,
		Diffs:         []VenueSignalDiff{},
		Note:          "Compares Binance USD-M vs Bybit linear using the same positioning, funding, and account long/short we already publish. Informational only — not financial advice.",
	}

	binWin := pickCompareWindow(bin)
	bybWin := pickCompareWindow(byb)
	out.BinanceOIChange = FormatSignedPct(binWin.OIChangePct)
	out.BybitOIChange = FormatSignedPct(bybWin.OIChangePct)

	out.Diffs = append(out.Diffs, compareOI(binWin, bybWin))
	out.Diffs = append(out.Diffs, compareFunding(bin.FundingRate, byb.FundingRate))
	out.Diffs = append(out.Diffs, compareCrowding(bin.LongShare, byb.LongShare))
	out.Diffs = append(out.Diffs, compareRegime(bin, byb))

	binLean := VenueLeanFromRegime(bin.Regime)
	bybLean := VenueLeanFromRegime(byb.Regime)
	if binLean == LeanMixed {
		binLean = leanFromCrowdingAndOI(CrowdedSideFromShares(bin.LongShare), binWin.OIDir)
	}
	if bybLean == LeanMixed {
		bybLean = leanFromCrowdingAndOI(CrowdedSideFromShares(byb.LongShare), bybWin.OIDir)
	}
	out.BinanceLean = binLean
	out.BybitLean = bybLean

	opp, mix, same := 0, 0, 0
	importantOpp := false
	for _, d := range out.Diffs {
		switch d.Alignment {
		case AlignOpposite:
			opp++
			if d.Important {
				importantOpp = true
			}
		case AlignMixed:
			mix++
		default:
			same++
		}
	}
	switch {
	case binLean != LeanMixed && bybLean != LeanMixed && binLean != bybLean:
		out.Alignment = AlignOpposite
		out.Important = true
	case opp > 0 || mix > 0:
		out.Alignment = AlignMixed
		out.Important = importantOpp || opp > 0
	default:
		out.Alignment = AlignSame
	}

	out.Title, out.Summary = divergenceCopy(out, bin, byb)
	return out
}

func pickCompareWindow(v PositioningVenueReport) PositioningWindow {
	if v.Primary.Window != "" {
		return v.Primary
	}
	for _, w := range v.Windows {
		if w.Window == "4h" {
			return w
		}
	}
	if len(v.Windows) > 0 {
		return v.Windows[0]
	}
	return PositioningWindow{}
}

func compareOI(bin, byb PositioningWindow) VenueSignalDiff {
	d := VenueSignalDiff{
		Metric:  "oi_change",
		Label:   "Open interest (" + nzWindow(bin.Window, byb.Window) + ")",
		Binance: fmt.Sprintf("%s (%s)", bin.OIDir, FormatSignedPct(bin.OIChangePct)),
		Bybit:   fmt.Sprintf("%s (%s)", byb.OIDir, FormatSignedPct(byb.OIChangePct)),
	}
	d.Alignment = dirAlign(bin.OIDir, byb.OIDir)
	d.Important = d.Alignment == AlignOpposite
	d.WhyItMatters = "OI rising on one venue and falling on the other means leverage is being added in one book and taken off in the other — flow is not global."
	if d.Alignment == AlignSame {
		d.WhyItMatters = "Both venues are changing open interest in the same direction."
	} else if d.Alignment == AlignMixed {
		d.WhyItMatters = "One venue’s OI is flat, so there is no clear split yet."
	}
	return d
}

func compareFunding(bin, byb float64) VenueSignalDiff {
	bDec, _ := FormatFundingRate(bin)
	yDec, _ := FormatFundingRate(byb)
	d := VenueSignalDiff{
		Metric:  "funding",
		Label:   "Funding payer",
		Binance: fmt.Sprintf("%s (%s)", FundingPayer(bin), bDec),
		Bybit:   fmt.Sprintf("%s (%s)", FundingPayer(byb), yDec),
	}
	bp, yp := FundingPayer(bin), FundingPayer(byb)
	switch {
	case bp == "none" || yp == "none":
		d.Alignment = AlignMixed
		d.WhyItMatters = "Funding is near zero on at least one venue, so payer split is weak."
	case bp == yp:
		d.Alignment = AlignSame
		d.WhyItMatters = "The same side pays funding on both venues."
	default:
		d.Alignment = AlignOpposite
		d.Important = true
		d.WhyItMatters = "Opposite funding means one book is long-crowded (longs pay) while the other is short-crowded (shorts pay). That is a real positioning split, not just noise."
	}
	return d
}

func compareCrowding(binLong, bybLong float64) VenueSignalDiff {
	bc := CrowdedSideFromShares(binLong)
	yc := CrowdedSideFromShares(bybLong)
	d := VenueSignalDiff{
		Metric:  "crowding",
		Label:   "Account long/short",
		Binance: fmt.Sprintf("%s (%.1f%% long)", bc, binLong*100),
		Bybit:   fmt.Sprintf("%s (%.1f%% long)", yc, bybLong*100),
	}
	switch {
	case bc == SqueezeSideNone || yc == SqueezeSideNone:
		d.Alignment = AlignMixed
		d.WhyItMatters = "At least one venue is balanced, so crowding is not a strong split."
	case bc == yc:
		d.Alignment = AlignSame
		d.WhyItMatters = "Account crowds lean the same way on both venues."
	default:
		d.Alignment = AlignOpposite
		d.Important = true
		d.WhyItMatters = "More accounts are long on one venue and short on the other. A move can liquidate different crowds on each exchange."
	}
	return d
}

func compareRegime(bin, byb PositioningVenueReport) VenueSignalDiff {
	d := VenueSignalDiff{
		Metric:  "positioning",
		Label:   "Price + OI regime",
		Binance: bin.Label,
		Bybit:   byb.Label,
	}
	if d.Binance == "" {
		d.Binance = RegimeLabel(bin.Regime)
	}
	if d.Bybit == "" {
		d.Bybit = RegimeLabel(byb.Regime)
	}
	bl := VenueLeanFromRegime(bin.Regime)
	yl := VenueLeanFromRegime(byb.Regime)
	switch {
	case bin.Regime == RegimeNeutral || byb.Regime == RegimeNeutral:
		d.Alignment = AlignMixed
		d.WhyItMatters = "One venue is neutral, so positioning is not a clean opposite."
	case bin.Regime == byb.Regime:
		d.Alignment = AlignSame
		d.WhyItMatters = "Both books show the same price+OI regime."
	case bl != LeanMixed && yl != LeanMixed && bl != yl:
		d.Alignment = AlignOpposite
		d.Important = true
		d.WhyItMatters = "One venue is adding/covering in a bullish way while the other is adding/unwinding in a bearish way. Combined OI can hide that split."
	default:
		d.Alignment = AlignMixed
		d.WhyItMatters = "Regimes differ but they are not a clean bullish-vs-bearish opposite."
	}
	return d
}

func nzWindow(a, b string) string {
	if a != "" {
		return a
	}
	if b != "" {
		return b
	}
	return "4h"
}

func dirAlign(a, b string) string {
	if a == "" || b == "" || a == "flat" || b == "flat" {
		return AlignMixed
	}
	if a == b {
		return AlignSame
	}
	return AlignOpposite
}

func divergenceCopy(out VenueDivergenceReport, bin, byb PositioningVenueReport) (title, summary string) {
	switch out.Alignment {
	case AlignSame:
		title = "Binance and Bybit are moving in the same direction"
		summary = fmt.Sprintf("Both lean %s. Positioning: Binance %s, Bybit %s. No important opposite split in OI, funding, crowding, or regime.",
			out.BinanceLean, nzLabel(bin), nzLabel(byb))
	case AlignOpposite:
		title = "Binance and Bybit are moving in opposite directions"
		parts := importantDiffPhrases(out.Diffs)
		summary = fmt.Sprintf("Binance leans %s (%s), Bybit leans %s (%s). %s That can matter because a single combined number (or watching only one exchange) hides who is adding risk.",
			out.BinanceLean, nzLabel(bin), out.BybitLean, nzLabel(byb), parts)
	default:
		title = "Binance and Bybit are mixed — not a full opposite"
		parts := importantDiffPhrases(out.Diffs)
		if parts != "" && parts != "Lean is opposite even if some individual prints are soft." {
			summary = fmt.Sprintf("Overall lean is not a clean opposite (Binance %s / Bybit %s). %s Combined OI can still hide that split.",
				out.BinanceLean, out.BybitLean, parts)
		} else {
			summary = fmt.Sprintf("Binance %s (%s), Bybit %s (%s). Some signals differ, but not a clean opposite on every driver.",
				out.BinanceLean, nzLabel(bin), out.BybitLean, nzLabel(byb))
		}
	}
	return title, strings.TrimSpace(summary)
}

func nzLabel(v PositioningVenueReport) string {
	if v.Label != "" {
		return v.Label
	}
	return RegimeLabel(v.Regime)
}

func importantDiffPhrases(diffs []VenueSignalDiff) string {
	var bits []string
	for _, d := range diffs {
		if !d.Important {
			continue
		}
		bits = append(bits, fmt.Sprintf("%s differs (%s vs %s).", d.Label, d.Binance, d.Bybit))
	}
	if len(bits) == 0 {
		return "Lean is opposite even if some individual prints are soft."
	}
	return strings.Join(bits, " ")
}

// HasMeaningfulDivergence is true when the user should be alerted.
func HasMeaningfulDivergence(r VenueDivergenceReport) bool {
	return r.Important && r.Alignment == AlignOpposite
}
