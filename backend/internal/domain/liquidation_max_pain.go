package domain

import (
	"fmt"
	"math"
	"time"
)

const maxPainLevels = 6

// MaxPainPocket is one liquidation cluster: biggest size at a price, not a hunt target.
type MaxPainPocket struct {
	Exchange         string  `json:"exchange,omitempty"`
	Side             string  `json:"side"`
	Direction        string  `json:"direction"`
	Price            float64 `json:"price"`
	MovePct          float64 `json:"movePct"`
	Notional         float64 `json:"notional"`
	EstNotional      float64 `json:"estNotional"`
	ObservedNotional float64 `json:"observedNotional,omitempty"`
	Leverage         float64 `json:"leverage,omitempty"`
	Source           string  `json:"source"`
}

// MaxPainVenue is the largest pockets above (shorts) and below (longs) last price.
type MaxPainVenue struct {
	Exchange          Exchange        `json:"exchange"`
	Symbol            string          `json:"symbol"`
	Price             float64         `json:"price"`
	OpenInterestValue float64         `json:"openInterestValue"`
	Above             *MaxPainPocket  `json:"above,omitempty"`
	Below             *MaxPainPocket  `json:"below,omitempty"`
	AboveLevels       []MaxPainPocket `json:"aboveLevels"`
	BelowLevels       []MaxPainPocket `json:"belowLevels"`
	Error             string          `json:"error,omitempty"`
}

// MaxPainReport is per-venue max-pain plus the single largest pocket each side.
type MaxPainReport struct {
	Symbol      string          `json:"symbol"`
	Exchange    string          `json:"exchange"`
	AsOf        time.Time       `json:"asOf"`
	Above       *MaxPainPocket  `json:"above,omitempty"`
	Below       *MaxPainPocket  `json:"below,omitempty"`
	Venues      []MaxPainVenue  `json:"venues"`
	Summary     string          `json:"summary"`
	Assumptions HuntAssumptions `json:"assumptions"`
	Note        string          `json:"note"`
}

// HuntBandPressure is estimated plus observed notional at a band.
func HuntBandPressure(b HuntBand) float64 {
	n := b.EstNotional + b.ObservedNotional
	if n < 0 {
		return 0
	}
	return n
}

// PickMaxPainPocket is the largest-pressure band. Not the hunt target.
func PickMaxPainPocket(bands []HuntBand, exchange string) *MaxPainPocket {
	var best *HuntBand
	bestN := 0.0
	for i := range bands {
		n := HuntBandPressure(bands[i])
		if bands[i].Price <= 0 || n <= 0 {
			continue
		}
		if best == nil || n > bestN || (n == bestN && math.Abs(bands[i].MovePct) < math.Abs(best.MovePct)) {
			b := bands[i]
			best = &b
			bestN = n
		}
	}
	if best == nil {
		return nil
	}
	return maxPainPocketFromBand(*best, exchange)
}

func maxPainPocketFromBand(b HuntBand, exchange string) *MaxPainPocket {
	return &MaxPainPocket{
		Exchange:         exchange,
		Side:             b.Side,
		Direction:        b.Direction,
		Price:            b.Price,
		MovePct:          b.MovePct,
		Notional:         HuntBandPressure(b),
		EstNotional:      b.EstNotional,
		ObservedNotional: b.ObservedNotional,
		Leverage:         b.Leverage,
		Source:           b.Source,
	}
}

func maxPainLevelsFromBands(bands []HuntBand, exchange string, limit int) []MaxPainPocket {
	if limit <= 0 {
		limit = maxPainLevels
	}
	out := make([]MaxPainPocket, 0, limit)
	for _, b := range bands {
		p := PickMaxPainPocket([]HuntBand{b}, exchange)
		if p == nil {
			continue
		}
		out = append(out, *p)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// MaxPainFromVenue maps hunt pressure bands to max-pain pockets.
// Hunt P&L / scores are ignored.
func MaxPainFromVenue(v HuntVenueReport) MaxPainVenue {
	name := string(v.Exchange)
	out := MaxPainVenue{
		Exchange:          v.Exchange,
		Symbol:            v.Symbol,
		Price:             v.Price,
		OpenInterestValue: v.OpenInterestValue,
		Above:             PickMaxPainPocket(v.UpPressure, name),
		Below:             PickMaxPainPocket(v.DownPressure, name),
		AboveLevels:       maxPainLevelsFromBands(v.UpPressure, name, maxPainLevels),
		BelowLevels:       maxPainLevelsFromBands(v.DownPressure, name, maxPainLevels),
		Error:             v.Error,
	}
	if out.AboveLevels == nil {
		out.AboveLevels = []MaxPainPocket{}
	}
	if out.BelowLevels == nil {
		out.BelowLevels = []MaxPainPocket{}
	}
	return out
}

// CombineMaxPainPockets picks the single largest above/below pocket across venues.
// Prices are never averaged.
func CombineMaxPainPockets(venues []MaxPainVenue) (above, below *MaxPainPocket) {
	for _, v := range venues {
		if v.Error != "" || v.Price <= 0 {
			continue
		}
		if v.Above != nil && (above == nil || v.Above.Notional > above.Notional) {
			cp := *v.Above
			above = &cp
		}
		if v.Below != nil && (below == nil || v.Below.Notional > below.Notional) {
			cp := *v.Below
			below = &cp
		}
	}
	return above, below
}

// MaxPainSummary is a one-line read of the two largest pockets.
func MaxPainSummary(above, below *MaxPainPocket) string {
	switch {
	case above == nil && below == nil:
		return "No estimated liquidation pockets on the visible model."
	case above != nil && below == nil:
		return fmt.Sprintf("Largest pocket is shorts above at %s (%s, %s).",
			compactUSD(above.Price), signedPct(above.MovePct), compactUSD(above.Notional))
	case above == nil && below != nil:
		return fmt.Sprintf("Largest pocket is longs below at %s (%s, %s).",
			compactUSD(below.Price), signedPct(below.MovePct), compactUSD(below.Notional))
	default:
		return fmt.Sprintf("Largest short pocket is %s above (%s). Largest long pocket is %s below (%s).",
			compactUSD(above.Notional), signedPct(above.MovePct),
			compactUSD(below.Notional), signedPct(below.MovePct))
	}
}
