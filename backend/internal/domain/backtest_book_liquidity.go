package domain

import "time"

// StoredLadder returns the grouped bid and ask ladder nearest a fill time.
// found is false when that minute was not stored.
type StoredLadder func(ex Exchange, symbol string, at time.Time) (bids, asks []BookHistoryLevel, found bool)

// WalkStoredLadder fills quantity against levels that are already best-first.
// avg is the volume-weighted price. slipFrac is the adverse distance from ref
// (0 when the book is at or inside ref). filled can be less than quantity.
func WalkStoredLadder(levels []BookHistoryLevel, qty, ref float64, buy bool) (avg, slipFrac, filled float64, ok bool) {
	if qty <= 0 || ref <= 0 {
		return 0, 0, 0, false
	}
	src := make([]ImpactSourceLevel, 0, len(levels))
	for _, lv := range levels {
		q := lv.Quantity
		if q <= 0 && lv.Price > 0 && lv.Notional > 0 {
			q = lv.Notional / lv.Price
		}
		if lv.Price <= 0 || q <= 0 {
			continue
		}
		src = append(src, ImpactSourceLevel{Price: lv.Price, Quantity: q})
	}
	got, _, avg, _, _ := WalkBookQuantity(src, qty)
	if got <= 0 || avg <= 0 {
		return 0, 0, 0, false
	}
	return avg, adverseBookSlip(buy, ref, avg), got, true
}

func adverseBookSlip(buy bool, ref, avg float64) float64 {
	if ref <= 0 || avg <= 0 {
		return 0
	}
	if buy && avg > ref {
		return (avg - ref) / ref
	}
	if !buy && avg < ref {
		return (ref - avg) / ref
	}
	return 0
}

func bookBuys(side string, exit bool) bool {
	long := NormalizeTradeSide(side) != BacktestTradeSideShort
	if exit {
		return !long
	}
	return long
}

func undoFixedSlip(buy bool, slipped, rate float64) float64 {
	if slipped <= 0 {
		return slipped
	}
	if rate <= 0 {
		return slipped
	}
	if buy {
		return slipped / (1 + rate)
	}
	if rate < 1 {
		return slipped / (1 - rate)
	}
	return slipped
}

// priceFromLadder walks the stored book for this leg. The reference is the
// price before the fixed slip percent, so the book is not applied twice.
func priceFromLadder(ladder StoredLadder, f BacktestBookFill, exit bool, at time.Time, slipped, qty float64) (px, slip float64, filled float64, ok bool) {
	if ladder == nil || at.IsZero() || qty <= 0 || slipped <= 0 {
		return 0, 0, 0, false
	}
	bids, asks, found := ladder(f.Exchange, f.Symbol, at)
	if !found {
		return 0, 0, 0, false
	}
	buy := bookBuys(f.Side, exit)
	levels := asks
	if !buy {
		levels = bids
	}
	ref := undoFixedSlip(buy, slipped, f.SlipRate)
	return WalkStoredLadder(levels, qty, ref, buy)
}
