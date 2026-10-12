package domain

import "math"

const (
	// FillPocketSpot is the newest open spot buy.
	FillPocketSpot = "spot"
	// FillPocketMargin is the margin entry when the book has no spot lot.
	FillPocketMargin = "margin"
	// FillPocketAbove means the second price sits above the first.
	FillPocketAbove = "above"
	// FillPocketBelow means the second price sits below the first.
	FillPocketBelow = "below"

	// FillPocketNote explains the mark for API and assistant readers.
	FillPocketNote = "The fill is the newest open spot buy, or the margin entry when that book has no spot lot. The pocket is the nearer estimated liquidation level on the combined Binance, Bybit, and OKX map. gapPct is the percent from the fill to that pocket. A missing venue leaves the pocket unmarked and is not zero. marginLiqPrice is this book's paper liquidation price, and marginGapPct is the percent from the pocket to that price. Equities have no map. Informational only — not financial advice."
)

// FillPocketFacts is the open paper position plus the live estimated pocket.
// PocketLoaded is false when the map could not be read.
type FillPocketFacts struct {
	Symbol         string
	Exchange       string
	FillPrice      float64
	FillKind       string
	Pocket         CloserPocket
	PocketLoaded   bool
	MarginLiqPrice float64
}

// FillPocketMark is the paper fill drawn on the liquidation map.
// Open is false when this book has no spot lot and no margin position.
// A missing venue leaves the pocket fields empty and names that venue.
type FillPocketMark struct {
	Symbol           string   `json:"symbol"`
	Exchange         string   `json:"exchange"`
	Open             bool     `json:"open"`
	Equity           bool     `json:"equity,omitempty"`
	FillPrice        float64  `json:"fillPrice,omitempty"`
	FillKind         string   `json:"fillKind,omitempty"`
	PocketSide       string   `json:"pocketSide,omitempty"`
	PocketPrice      float64  `json:"pocketPrice,omitempty"`
	GapPct           *float64 `json:"gapPct,omitempty"`
	PocketFromFill   string   `json:"pocketFromFill,omitempty"`
	MarginLiqPrice   float64  `json:"marginLiqPrice,omitempty"`
	MarginGapPct     *float64 `json:"marginGapPct,omitempty"`
	MarginFromPocket string   `json:"marginFromPocket,omitempty"`
	Missing          []string `json:"missing,omitempty"`
	Note             string   `json:"note,omitempty"`
}

// BuildFillPocketMark measures the open fill against the nearer estimated pocket.
// The percent from the fill uses the fill as the base. The percent from the pocket
// to the paper liquidation price uses the pocket as the base. A zero percent is kept.
func BuildFillPocketMark(in FillPocketFacts) FillPocketMark {
	out := FillPocketMark{
		Symbol:   in.Symbol,
		Exchange: in.Exchange,
		Note:     FillPocketNote,
	}
	if in.FillPrice <= 0 && in.MarginLiqPrice <= 0 {
		return out
	}
	out.Open = true
	if in.FillPrice > 0 {
		out.FillPrice = in.FillPrice
		out.FillKind = in.FillKind
	}
	if in.MarginLiqPrice > 0 {
		out.MarginLiqPrice = in.MarginLiqPrice
	}
	if !in.PocketLoaded || !in.Pocket.Covered {
		out.Missing = in.Pocket.Missing
		if len(out.Missing) == 0 {
			out.Missing = situationVenueNames()
		}
		return out
	}
	if in.Pocket.Side == SituationLiqNone || in.Pocket.Price <= 0 {
		out.PocketSide = SituationLiqNone
		return out
	}
	out.PocketSide = in.Pocket.Side
	out.PocketPrice = in.Pocket.Price
	if in.FillPrice > 0 {
		if pct, dir, ok := priceGap(in.FillPrice, in.Pocket.Price); ok {
			out.GapPct = &pct
			out.PocketFromFill = dir
		}
	}
	if in.MarginLiqPrice > 0 {
		if pct, dir, ok := priceGap(in.Pocket.Price, in.MarginLiqPrice); ok {
			out.MarginGapPct = &pct
			out.MarginFromPocket = dir
		}
	}
	return out
}

// priceGap is the absolute percent from base to other, rounded to one decimal.
// dir is above or below. An equal pair after rounding returns 0 and an empty dir.
func priceGap(base, other float64) (pct float64, dir string, ok bool) {
	if base <= 0 || other <= 0 || math.IsNaN(base) || math.IsNaN(other) {
		return 0, "", false
	}
	pct = roundSituationPct(math.Abs(other-base) / base * 100)
	if pct == 0 {
		return 0, "", true
	}
	if other > base {
		return pct, FillPocketAbove, true
	}
	return pct, FillPocketBelow, true
}
