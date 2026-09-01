package domain

import (
	"fmt"
	"math"
	"sort"
)

const (
	maxHuntCascadeSteps = 8
	huntCascadeMergePct = 0.08

	HuntCascadeRoleStart        = "start"
	HuntCascadeRoleSelf         = "self"
	HuntCascadeRoleHelped       = "helped"
	HuntCascadeRoleStall        = "stall"
	HuntCascadeRoleUnreachable  = "unreachable"
	HuntCascadeRoleMissing      = "missing"
	HuntCascadeRoleObservedOnly = "observed"

	HuntCascadeStrengthSelf   = "self"
	HuntCascadeStrengthStrong = "strong"
	HuntCascadeStrengthMixed  = "mixed"
	HuntCascadeStrengthWeak   = "weak"
)

// HuntCascadeStep is one zone on a price-ordered hunt path.
type HuntCascadeStep struct {
	Index                int       `json:"index"`
	Band                 HuntBand  `json:"band"`
	FromPrice            float64   `json:"fromPrice"`
	MovePct              float64   `json:"movePct"`
	HopPct               float64   `json:"hopPct"`
	ZoneNotional         float64   `json:"zoneNotional"`
	CumulativeNotional   float64   `json:"cumulativeNotional"`
	FuelAdds             float64   `json:"fuelAdds"`
	Standalone           BookReach `json:"standalone"`
	Incremental          BookReach `json:"incremental"`
	Remaining            BookReach `json:"remaining"`
	PriorCascadeNotional float64   `json:"priorCascadeNotional"`
	FuelSpent            float64   `json:"fuelSpent"`
	FuelLeft             float64   `json:"fuelLeft"`
	AssistancePct        *float64  `json:"assistancePct,omitempty"`
	Strength             *float64  `json:"strength,omitempty"`
	StrengthLevel        string    `json:"strengthLevel,omitempty"`
	Role                 string    `json:"role"`
	Easier               bool      `json:"easier"`
	SelfFueling          bool      `json:"selfFueling"`
	Reachable            bool      `json:"reachable"`
	ZoneEst              string    `json:"zoneEst"`
	Note                 string    `json:"note"`
}

// HuntCascadePath is the sequence of zones in one direction.
// Triggering earlier estimated liquidations can cheapen later hops.
// It does not change hunt P&L or direction scores.
type HuntCascadePath struct {
	Direction        string            `json:"direction"`
	Steps            []HuntCascadeStep `json:"steps"`
	ReachableCount   int               `json:"reachableCount"`
	EasierCount      int               `json:"easierCount"`
	SelfFuelingCount int               `json:"selfFuelingCount"`
	FeedsUntilIndex  int               `json:"feedsUntilIndex,omitempty"`
	FeedsUntilPrice  float64           `json:"feedsUntilPrice,omitempty"`
	StallsAtIndex    int               `json:"stallsAtIndex,omitempty"`
	StallsAtPrice    float64           `json:"stallsAtPrice,omitempty"`
	StallRole        string            `json:"stallRole,omitempty"`
	StallNote        string            `json:"stallNote,omitempty"`
	ChainEasier      bool              `json:"chainEasier"`
	Summary          string            `json:"summary"`
}

// BuildHuntCascadePath walks bands nearest-to-last first.
// Fuel is a running pool: a hop spends from the pool, leftover stays,
// and only a reached modeled zone adds new assumed exit flow. Spent
// fuel is not reapplied on the next hop.
func BuildHuntCascadePath(dir string, bands []HuntBand, mid float64, push []ImpactSourceLevel, side string) HuntCascadePath {
	out := HuntCascadePath{Direction: dir, Steps: []HuntCascadeStep{}}
	if mid <= 0 {
		out.Summary = "No last price for a cascade path."
		return out
	}
	ordered := cascadePathBands(bands, mid, dir)
	if len(ordered) == 0 {
		out.Summary = "No liquidation zones on this side."
		return out
	}
	var fuelPool float64
	prevPrice := mid
	var cumEst float64
	for i, b := range ordered {
		step := huntCascadeStep(i+1, b, mid, prevPrice, fuelPool, push, side)
		left := fuelPool - step.FuelSpent
		if left < 0 {
			left = 0
		}
		step.FuelLeft = left
		if step.Reachable && step.ZoneEst == "model" {
			fuelPool = left + step.FuelAdds
		} else {
			fuelPool = left
		}
		cumEst += b.EstNotional
		step.CumulativeNotional = cumEst
		out.Steps = append(out.Steps, step)
		if step.Reachable {
			out.ReachableCount++
		}
		if step.Easier {
			out.EasierCount++
		}
		if step.SelfFueling {
			out.SelfFuelingCount++
		}
		prevPrice = b.Price
	}
	markHuntCascadeStall(&out)
	out.ChainEasier = out.EasierCount > 0 || out.SelfFuelingCount > 0
	out.Summary = huntCascadeSummary(out)
	return out
}

func huntCascadeStep(index int, b HuntBand, mid, fromPrice, fuelIn float64, push []ImpactSourceLevel, side string) HuntCascadeStep {
	step := HuntCascadeStep{
		Index:                index,
		Band:                 b,
		FromPrice:            fromPrice,
		MovePct:              (b.Price - mid) / mid * 100,
		HopPct:               (b.Price - fromPrice) / mid * 100,
		ZoneNotional:         b.EstNotional,
		FuelAdds:             b.EstNotional * HuntCascadeFillRate,
		PriorCascadeNotional: fuelIn,
	}
	if b.EstNotional > 0 {
		step.ZoneEst = "model"
	} else if b.ObservedNotional > 0 {
		step.ZoneEst = "observed"
		step.FuelAdds = 0
	} else {
		step.ZoneEst = "missing"
		step.FuelAdds = 0
	}

	step.Standalone = WalkBookToPrice(side, mid, push, b.Price)
	hopBook := push
	if index > 1 {
		hopBook = LevelsBeyond(side, push, fromPrice)
	}
	noHopBook := len(hopBook) == 0
	step.Incremental = WalkBookToPrice(side, fromPrice, hopBook, b.Price)
	step.Remaining = step.Incremental

	switch {
	case noHopBook:
		step.Role = HuntCascadeRoleMissing
		step.Note = huntCascadeStartOrHop(index) + "No visible hop book."
	case !step.Incremental.Reachable:
		if step.PriorCascadeNotional > 0 {
			_, _, spent := ConsumeBookNotional(hopBook, step.PriorCascadeNotional)
			step.FuelSpent = spent
		}
		step.Role = HuntCascadeRoleUnreachable
		step.Reachable = false
		step.Note = huntCascadeStartOrHop(index) + "Visible book does not reach this zone."
	case index == 1:
		step.Reachable = true
		step.Role = HuntCascadeRoleStart
		if step.ZoneEst == "observed" {
			step.Note = "First zone from last price. Observed cluster only — not used as cascade fuel."
		} else if step.ZoneEst == "missing" {
			step.Note = "First zone from last price. No estimated liquidation in this band."
		} else {
			step.Note = fmt.Sprintf("First zone from last price. Hitting it adds %s assumed exit flow (%.0f%% of estimated liq).",
				compactUSD(step.FuelAdds), HuntCascadeFillRate*100)
		}
	case step.PriorCascadeNotional >= step.Incremental.Notional && step.Incremental.Notional > 0:
		step.SelfFueling = true
		step.Reachable = true
		step.Easier = true
		step.FuelSpent = step.Incremental.Notional
		step.Remaining = BookReach{
			Side:              side,
			TargetPrice:       b.Price,
			MidPrice:          fromPrice,
			EndPrice:          b.Price,
			MaxReachablePrice: b.Price,
			Reachable:         true,
		}
		step.AssistancePct = floatPtr(100)
		step.Strength = floatPtr(100)
		step.StrengthLevel = HuntCascadeStrengthSelf
		step.Role = HuntCascadeRoleSelf
		step.Note = fmt.Sprintf("Prior assumed exit flow (%s) covers this hop (%s).",
			compactUSD(step.PriorCascadeNotional), compactUSD(step.Incremental.Notional))
	case step.PriorCascadeNotional > 0:
		endPx, leftover, spent := ConsumeBookNotional(hopBook, step.PriorCascadeNotional)
		step.FuelSpent = spent
		start := endPx
		if start <= 0 {
			start = fromPrice
		}
		step.Remaining = WalkBookToPrice(side, start, leftover, b.Price)
		step.Reachable = step.Remaining.Reachable
		if step.Incremental.Notional > 0 && step.Remaining.Reachable {
			saved := step.Incremental.Notional - step.Remaining.Notional
			if saved < 0 {
				saved = 0
			}
			assist := clampScore(100 * saved / step.Incremental.Notional)
			step.AssistancePct = floatPtr(assist)
			step.Strength = floatPtr(assist)
			step.StrengthLevel = huntCascadeStrengthLevel(assist)
			if saved > 0 {
				step.Easier = true
				step.Role = HuntCascadeRoleHelped
				step.Note = fmt.Sprintf("Prior assumed exit flow (%s) covers %.0f%% of this hop; desk still needs %s.",
					compactUSD(step.PriorCascadeNotional), assist, compactUSD(step.Remaining.Notional))
			} else {
				step.Role = HuntCascadeRoleStall
				step.Note = fmt.Sprintf("Prior assumed exit flow (%s) does not reduce this hop (%s).",
					compactUSD(step.PriorCascadeNotional), compactUSD(step.Incremental.Notional))
			}
		} else {
			step.Role = HuntCascadeRoleUnreachable
			step.Note = fmt.Sprintf("After spending %s of prior flow, leftover book does not reach this zone.",
				compactUSD(spent))
		}
	default:
		step.Reachable = step.Incremental.Reachable
		step.Role = HuntCascadeRoleStall
		step.Note = "No prior estimated liquidation to use as exit flow. This hop needs the full visible spot walk."
	}

	if index == 1 && step.Role != HuntCascadeRoleMissing && step.Role != HuntCascadeRoleUnreachable {
		step.Role = HuntCascadeRoleStart
	}
	if step.ZoneEst == "observed" && index > 1 && step.Role != HuntCascadeRoleMissing && step.Role != HuntCascadeRoleUnreachable && step.Role != HuntCascadeRoleSelf && step.Role != HuntCascadeRoleHelped {
		step.Role = HuntCascadeRoleObservedOnly
		step.Note = "Observed cluster only — not used as cascade fuel. " + step.Note
	}
	return step
}

func huntCascadeStartOrHop(index int) string {
	if index == 1 {
		return "First zone. "
	}
	return ""
}

func huntCascadeStrengthLevel(score float64) string {
	switch {
	case score >= 100:
		return HuntCascadeStrengthSelf
	case score >= 70:
		return HuntCascadeStrengthStrong
	case score >= 40:
		return HuntCascadeStrengthMixed
	default:
		return HuntCascadeStrengthWeak
	}
}

func floatPtr(v float64) *float64 {
	x := v
	return &x
}

func markHuntCascadeStall(p *HuntCascadePath) {
	if p == nil {
		return
	}
	var lastSelf int
	var lastSelfPx float64
	for _, s := range p.Steps {
		if s.SelfFueling {
			lastSelf = s.Index
			lastSelfPx = s.Band.Price
		}
	}
	if lastSelf > 0 {
		p.FeedsUntilIndex = lastSelf
		p.FeedsUntilPrice = lastSelfPx
	}
	for _, s := range p.Steps {
		if s.Index == 1 {
			if s.Role == HuntCascadeRoleUnreachable || s.Role == HuntCascadeRoleMissing {
				p.StallsAtIndex = s.Index
				p.StallsAtPrice = s.Band.Price
				p.StallRole = s.Role
				p.StallNote = s.Note
				return
			}
			continue
		}
		if s.Role == HuntCascadeRoleSelf {
			continue
		}
		p.StallsAtIndex = s.Index
		p.StallsAtPrice = s.Band.Price
		p.StallRole = s.Role
		p.StallNote = s.Note
		return
	}
}

func cascadePathBands(bands []HuntBand, mid float64, dir string) []HuntBand {
	filtered := make([]HuntBand, 0, len(bands))
	for _, b := range bands {
		if b.Price <= 0 {
			continue
		}
		if dir == "up" && b.Price <= mid {
			continue
		}
		if dir == "down" && b.Price >= mid {
			continue
		}
		filtered = append(filtered, b)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if dir == "up" {
			if filtered[i].Price == filtered[j].Price {
				return filtered[i].EstNotional > filtered[j].EstNotional
			}
			return filtered[i].Price < filtered[j].Price
		}
		if filtered[i].Price == filtered[j].Price {
			return filtered[i].EstNotional > filtered[j].EstNotional
		}
		return filtered[i].Price > filtered[j].Price
	})
	merged := make([]HuntBand, 0, len(filtered))
	for _, b := range filtered {
		if len(merged) == 0 {
			merged = append(merged, b)
			continue
		}
		prev := &merged[len(merged)-1]
		if mid > 0 && math.Abs(prev.Price-b.Price)/mid*100 <= huntCascadeMergePct {
			prev.EstNotional += b.EstNotional
			prev.ObservedNotional += b.ObservedNotional
			if prev.Source != b.Source {
				prev.Source = "both"
			}
			if b.Leverage > 0 && (prev.Leverage == 0 || (dir == "up" && b.Price < prev.Price) || (dir == "down" && b.Price > prev.Price)) {
				prev.Leverage = b.Leverage
			}
			continue
		}
		merged = append(merged, b)
	}
	if len(merged) > maxHuntCascadeSteps {
		merged = merged[:maxHuntCascadeSteps]
	}
	return merged
}

func huntCascadeSummary(p HuntCascadePath) string {
	n := len(p.Steps)
	if n == 0 {
		return "No liquidation zones on this side."
	}
	side := "short liquidations above"
	if p.Direction == "down" {
		side = "long liquidations below"
	}
	if p.StallsAtIndex == 1 && (p.StallRole == HuntCascadeRoleUnreachable || p.StallRole == HuntCascadeRoleMissing) {
		return fmt.Sprintf("%d %s zones. Cannot start: %s", n, side, trimHuntCascadeNote(p.StallNote))
	}
	if p.FeedsUntilIndex > 0 && p.StallsAtIndex > p.FeedsUntilIndex {
		return fmt.Sprintf("%d %s zones. Cascade feeds itself through zone %d (%s). Stalls at zone %d (%s) — extra spot is needed.",
			n, side, p.FeedsUntilIndex, signedPct(p.stepMove(p.FeedsUntilIndex)),
			p.StallsAtIndex, signedPct(p.stepMove(p.StallsAtIndex)))
	}
	if p.FeedsUntilIndex > 0 && p.StallsAtIndex == 0 {
		return fmt.Sprintf("%d %s zones. Cascade stays self-fueling through zone %d (%s).",
			n, side, p.FeedsUntilIndex, signedPct(p.stepMove(p.FeedsUntilIndex)))
	}
	if p.StallsAtIndex > 1 {
		return fmt.Sprintf("%d %s zones. Cascade does not feed itself after the first zone. Extra spot is needed at zone %d (%s).",
			n, side, p.StallsAtIndex, signedPct(p.stepMove(p.StallsAtIndex)))
	}
	if p.ReachableCount == 0 {
		return fmt.Sprintf("%d %s zones, none reachable on the visible book.", n, side)
	}
	return fmt.Sprintf("%d %s zones (%d reachable on the visible book).", n, side, p.ReachableCount)
}

func (p HuntCascadePath) stepMove(index int) float64 {
	for _, s := range p.Steps {
		if s.Index == index {
			return s.MovePct
		}
	}
	return math.NaN()
}

func trimHuntCascadeNote(s string) string {
	if s == "" {
		return "visible book does not reach the first zone."
	}
	return s
}
