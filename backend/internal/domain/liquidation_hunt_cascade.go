package domain

import (
	"fmt"
	"math"
	"sort"
)

const (
	maxHuntCascadeSteps     = 8
	huntCascadeMergePct     = 0.08
	huntCascadeEasierRatio  = 0.92
	huntCascadeSelfFuelFrac = 0.02
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
	Standalone           BookReach `json:"standalone"`
	Incremental          BookReach `json:"incremental"`
	Remaining            BookReach `json:"remaining"`
	PriorCascadeNotional float64   `json:"priorCascadeNotional"`
	AssistancePct        float64   `json:"assistancePct"`
	Easier               bool      `json:"easier"`
	SelfFueling          bool      `json:"selfFueling"`
	Reachable            bool      `json:"reachable"`
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
	ChainEasier      bool              `json:"chainEasier"`
	Summary          string            `json:"summary"`
}

// BuildHuntCascadePath walks bands nearest-to-last first and asks whether
// HuntCascadeFillRate of earlier EstNotional covers part of the next hop.
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
	var priorEst float64
	prevPrice := mid
	for i, b := range ordered {
		step := huntCascadeStep(dir, i+1, b, mid, prevPrice, priorEst, push, side)
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
		priorEst += b.EstNotional
		prevPrice = b.Price
	}
	out.ChainEasier = out.EasierCount > 0 || out.SelfFuelingCount > 0
	out.Summary = huntCascadeSummary(out)
	return out
}

func huntCascadeStep(dir string, index int, b HuntBand, mid, fromPrice, priorEst float64, push []ImpactSourceLevel, side string) HuntCascadeStep {
	step := HuntCascadeStep{
		Index:                index,
		Band:                 b,
		FromPrice:            fromPrice,
		MovePct:              (b.Price - mid) / mid * 100,
		HopPct:               (b.Price - fromPrice) / mid * 100,
		ZoneNotional:         b.EstNotional,
		CumulativeNotional:   priorEst + b.EstNotional,
		PriorCascadeNotional: priorEst * HuntCascadeFillRate,
	}
	step.Standalone = WalkBookToPrice(side, mid, push, b.Price)
	hopBook := push
	if index > 1 {
		hopBook = LevelsBeyond(side, push, fromPrice)
	}
	step.Incremental = WalkBookToPrice(side, fromPrice, hopBook, b.Price)
	step.Remaining = step.Incremental
	if step.PriorCascadeNotional > 0 && len(hopBook) > 0 {
		endPx, leftover, _ := ConsumeBookNotional(hopBook, step.PriorCascadeNotional)
		if cascadeReached(side, endPx, b.Price) {
			step.SelfFueling = true
			step.Remaining = BookReach{
				Side:              side,
				TargetPrice:       b.Price,
				MidPrice:          fromPrice,
				EndPrice:          endPx,
				MaxReachablePrice: endPx,
				Reachable:         true,
			}
		} else {
			start := endPx
			if start <= 0 {
				start = fromPrice
			}
			step.Remaining = WalkBookToPrice(side, start, leftover, b.Price)
		}
	}
	step.Reachable = step.Remaining.Reachable || step.SelfFueling || step.Incremental.Reachable && step.Remaining.Notional <= 0
	if step.SelfFueling {
		step.AssistancePct = 100
	} else if step.Remaining.Reachable && step.Incremental.Notional > 0 {
		saved := step.Incremental.Notional - step.Remaining.Notional
		if saved < 0 {
			saved = 0
		}
		step.AssistancePct = clampScore(100 * saved / step.Incremental.Notional)
	}
	if index > 1 && (step.SelfFueling || (step.Remaining.Reachable && step.Incremental.Notional > 0 &&
		step.Remaining.Notional+1e-9 < step.Incremental.Notional*huntCascadeEasierRatio)) {
		step.Easier = true
	}
	if step.SelfFueling && step.Incremental.Notional > 0 &&
		step.PriorCascadeNotional < step.Incremental.Notional*huntCascadeSelfFuelFrac {
		step.SelfFueling = false
	}
	step.Note = huntCascadeNote(step, index, b)
	return step
}

func cascadeReached(side string, endPrice, target float64) bool {
	if endPrice <= 0 || target <= 0 {
		return false
	}
	if side == ImpactSideSell {
		return endPrice <= target
	}
	return endPrice >= target
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

func huntCascadeNote(step HuntCascadeStep, index int, b HuntBand) string {
	switch {
	case !step.Reachable && !step.Incremental.Reachable:
		return "Visible book does not reach this zone."
	case index == 1:
		if b.EstNotional <= 0 && b.ObservedNotional > 0 {
			return "First zone from last price (observed cluster; not used as cascade fuel)."
		}
		return "First zone from last price."
	case step.SelfFueling:
		return fmt.Sprintf("Prior estimated liquidations walk through this zone (about %s of assumed exit flow).", compactUSD(step.PriorCascadeNotional))
	case step.Easier:
		return fmt.Sprintf("Triggering the previous zone covers about %.0f%% of this hop.", step.AssistancePct)
	case b.EstNotional <= 0 && b.ObservedNotional > 0:
		return "Observed cluster only — not used as cascade fuel for the next hop."
	default:
		return "Prior cascade is small versus the remaining book to here."
	}
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
	switch {
	case p.ReachableCount == 0:
		return fmt.Sprintf("%d %s zones, none reachable on the visible book.", n, side)
	case p.SelfFuelingCount > 0 && p.EasierCount > 0:
		return fmt.Sprintf("%d %s zones (%d reachable). Hitting earlier zones cheapens %d later hop(s); %d look self-fueling.",
			n, side, p.ReachableCount, p.EasierCount, p.SelfFuelingCount)
	case p.SelfFuelingCount > 0:
		return fmt.Sprintf("%d %s zones (%d reachable). %d later hop(s) look self-fueling after earlier liquidations.",
			n, side, p.ReachableCount, p.SelfFuelingCount)
	case p.EasierCount > 0:
		return fmt.Sprintf("%d %s zones (%d reachable). Hitting earlier zones cheapens %d later hop(s).",
			n, side, p.ReachableCount, p.EasierCount)
	default:
		return fmt.Sprintf("%d %s zones (%d reachable). Prior hops do not cheapen the next one enough — each step still needs desk flow.",
			n, side, p.ReachableCount)
	}
}
