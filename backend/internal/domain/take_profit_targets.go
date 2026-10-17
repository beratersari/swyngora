package domain

import (
	"fmt"
	"math"
	"sort"
)

// MaxTakeProfitTargets is how many scale-out limits one OCO or bracket may rest.
const MaxTakeProfitTargets = 8

// TakeProfitPercentSumEpsilon is how far the shares may sit from 100.
const TakeProfitPercentSumEpsilon = 0.01

// TakeProfitTarget is one slice of an OCO or bracket take-profit.
// Percent is the share of the order quantity (the shares sum to 100).
// Price is an absolute limit. OffsetPercent is a percent above the reference
// (5 means +5%). A target carries one of those, not both.
// For a bracket the reference is the entry limit. For an OCO it is the average
// cost of the open position, or the last price when that cost is missing.
type TakeProfitTarget struct {
	Percent       float64 `json:"percent"`
	Price         float64 `json:"price,omitempty"`
	OffsetPercent float64 `json:"offsetPercent,omitempty"`
}

// ResolvedTakeProfit is a target after the limit price and the coin size are known.
type ResolvedTakeProfit struct {
	Percent       float64
	OffsetPercent float64
	Price         float64
	Quantity      float64
}

// NormalizeTakeProfitTargets checks the scale-out plan and turns offsets into prices.
// An empty list uses fallbackPrice as one target for the whole quantity.
// aboveReference is true for a bracket: every limit must sit above the entry.
// Targets are ordered from the nearest price to the furthest. The furthest
// share absorbs rounding so the sizes add up to quantity when the shares add up to 100.
func NormalizeTakeProfitTargets(raw []TakeProfitTarget, fallbackPrice, quantity, reference, stop float64, aboveReference bool) ([]ResolvedTakeProfit, error) {
	if quantity < MinTradeQuantity || quantity > MaxTradeQuantity || math.IsNaN(quantity) || math.IsInf(quantity, 0) {
		return nil, fmt.Errorf("%w: quantity out of range", ErrInvalidArgument)
	}
	if stop <= 0 || math.IsNaN(stop) || math.IsInf(stop, 0) || stop > MaxTriggerPrice {
		return nil, fmt.Errorf("%w: stopLossPrice must be positive", ErrInvalidArgument)
	}
	targets := raw
	if len(targets) == 0 {
		if fallbackPrice <= 0 || math.IsNaN(fallbackPrice) || math.IsInf(fallbackPrice, 0) {
			return nil, fmt.Errorf("%w: takeProfitPrice or takeProfitTargets is required", ErrInvalidArgument)
		}
		targets = []TakeProfitTarget{{Percent: 100, Price: fallbackPrice}}
	}
	if len(targets) > MaxTakeProfitTargets {
		return nil, fmt.Errorf("%w: at most %d take-profit targets", ErrInvalidArgument, MaxTakeProfitTargets)
	}
	resolved := make([]ResolvedTakeProfit, 0, len(targets))
	var sum float64
	for _, t := range targets {
		if t.Percent <= 0 || t.Percent > 100 || math.IsNaN(t.Percent) || math.IsInf(t.Percent, 0) {
			return nil, fmt.Errorf("%w: take-profit percent must be greater than 0 and at most 100", ErrInvalidArgument)
		}
		sum += t.Percent
		price, offset, err := takeProfitLimit(t, reference)
		if err != nil {
			return nil, err
		}
		if price <= stop {
			return nil, fmt.Errorf("%w: each take-profit price must be above the stop", ErrInvalidArgument)
		}
		if aboveReference && price <= reference {
			return nil, fmt.Errorf("%w: each take-profit price must be above the entry", ErrInvalidArgument)
		}
		resolved = append(resolved, ResolvedTakeProfit{
			Percent: t.Percent, OffsetPercent: offset, Price: price,
		})
	}
	if math.Abs(sum-100) > TakeProfitPercentSumEpsilon {
		return nil, fmt.Errorf("%w: take-profit percents must sum to 100", ErrInvalidArgument)
	}
	sort.SliceStable(resolved, func(i, j int) bool {
		return resolved[i].Price < resolved[j].Price
	})
	for i := 1; i < len(resolved); i++ {
		if resolved[i].Price <= resolved[i-1].Price {
			return nil, fmt.Errorf("%w: take-profit prices must differ", ErrInvalidArgument)
		}
	}
	percents := make([]float64, len(resolved))
	for i := range resolved {
		percents[i] = resolved[i].Percent
	}
	qty, err := SplitTargetQuantities(quantity, percents)
	if err != nil {
		return nil, err
	}
	for i := range resolved {
		if qty[i] < MinTradeQuantity {
			return nil, fmt.Errorf("%w: a take-profit share is below the minimum size", ErrInvalidArgument)
		}
		resolved[i].Quantity = qty[i]
	}
	return resolved, nil
}

func takeProfitLimit(t TakeProfitTarget, reference float64) (price, offset float64, err error) {
	hasPrice := t.Price > 0 && !math.IsNaN(t.Price) && !math.IsInf(t.Price, 0)
	hasOffset := t.OffsetPercent > 0 && !math.IsNaN(t.OffsetPercent) && !math.IsInf(t.OffsetPercent, 0)
	if hasPrice && hasOffset {
		return 0, 0, fmt.Errorf("%w: a take-profit target uses a price or a gain percent, not both", ErrInvalidArgument)
	}
	if !hasPrice && !hasOffset {
		return 0, 0, fmt.Errorf("%w: each take-profit target needs a price or a gain percent", ErrInvalidArgument)
	}
	if hasOffset {
		if t.OffsetPercent > 1000 {
			return 0, 0, fmt.Errorf("%w: take-profit gain percent is too large", ErrInvalidArgument)
		}
		if reference <= 0 || math.IsNaN(reference) || math.IsInf(reference, 0) {
			return 0, 0, fmt.Errorf("%w: a reference price is required for a gain percent", ErrInvalidArgument)
		}
		price = reference * (1 + t.OffsetPercent/100)
		offset = t.OffsetPercent
	} else {
		price = t.Price
	}
	if price < MinTriggerPrice || price > MaxTriggerPrice || math.IsNaN(price) || math.IsInf(price, 0) {
		return 0, 0, fmt.Errorf("%w: take-profit price out of range", ErrInvalidArgument)
	}
	return price, offset, nil
}

// SplitTargetQuantities divides total by percent shares.
// When the shares add up to 100, the last share absorbs rounding so the sizes
// add up to total. A smaller sum keeps the unused share out of these sizes.
func SplitTargetQuantities(total float64, percents []float64) ([]float64, error) {
	if len(percents) == 0 {
		return nil, fmt.Errorf("%w: take-profit targets are required", ErrInvalidArgument)
	}
	if total < 0 || math.IsNaN(total) || math.IsInf(total, 0) {
		return nil, fmt.Errorf("%w: quantity out of range", ErrInvalidArgument)
	}
	out := make([]float64, len(percents))
	if total == 0 {
		return out, nil
	}
	var sumP, exactSum float64
	exact := make([]float64, len(percents))
	for i, p := range percents {
		if p < 0 || math.IsNaN(p) || math.IsInf(p, 0) {
			return nil, fmt.Errorf("%w: take-profit percent must be greater than 0 and at most 100", ErrInvalidArgument)
		}
		sumP += p
		exact[i] = total * p / 100
		exactSum += exact[i]
	}
	var used float64
	for i := 0; i < len(percents)-1; i++ {
		q := math.Round(exact[i]*1e8) / 1e8
		if q < 0 {
			q = 0
		}
		out[i] = q
		used += q
	}
	if math.Abs(sumP-100) <= TakeProfitPercentSumEpsilon {
		out[len(out)-1] = total - used
	} else {
		out[len(out)-1] = exactSum - used
	}
	if out[len(out)-1] < 0 {
		out[len(out)-1] = 0
	}
	return out, nil
}

// ValidateMoveStopToEntry checks that the buy price sits strictly between the stop and the first target.
// The stop can then move up to that buy price without crossing a target or moving down.
func ValidateMoveStopToEntry(buy, stop, firstTarget float64) error {
	if buy <= 0 || math.IsNaN(buy) || math.IsInf(buy, 0) {
		return fmt.Errorf("%w: a buy price is required to move the stop", ErrInvalidArgument)
	}
	if stop <= 0 || buy <= stop {
		return fmt.Errorf("%w: the buy price must be above the stop", ErrInvalidArgument)
	}
	if firstTarget > 0 && buy >= firstTarget {
		return fmt.Errorf("%w: the buy price must be below the first take-profit", ErrInvalidArgument)
	}
	return nil
}

// RaiseStopToBuyPrice is the stop trigger after the nearest target has filled.
// Zero means the stop stays put: the option is off, the buy price is missing,
// it is not above the current stop, or it is not below the next open target.
func RaiseStopToBuyPrice(move bool, buy, stop, nextTarget float64) float64 {
	if !move || buy <= 0 || math.IsNaN(buy) || math.IsInf(buy, 0) {
		return 0
	}
	if buy <= stop+1e-9 {
		return 0
	}
	if nextTarget > 0 && buy >= nextTarget-1e-9 {
		return 0
	}
	return buy
}

// ScaleRemainings shrinks open slices so they add up to target.
// The last slice absorbs rounding. A target already at or below the sum is left as-is.
func ScaleRemainings(remainings []float64, target float64) []float64 {
	out := append([]float64(nil), remainings...)
	if len(out) == 0 {
		return out
	}
	var sum float64
	for _, r := range out {
		if r > 0 {
			sum += r
		}
	}
	if sum <= PositionEpsilon || target >= sum-1e-9 {
		return out
	}
	if target <= PositionEpsilon {
		for i := range out {
			out[i] = 0
		}
		return out
	}
	var used float64
	for i := 0; i < len(out)-1; i++ {
		if out[i] <= 0 {
			out[i] = 0
			continue
		}
		q := math.Round(out[i]*target/sum*1e8) / 1e8
		if q < 0 {
			q = 0
		}
		out[i] = q
		used += q
	}
	last := target - used
	if last < 0 {
		last = 0
	}
	out[len(out)-1] = last
	return out
}
