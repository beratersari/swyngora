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
// found is true when that minute has a ladder, even if filled is only part
// of qty (or zero). A missing minute returns found false so the caller keeps
// the fixed slip percent.
func priceFromLadder(ladder StoredLadder, f BacktestBookFill, exit bool, at time.Time, slipped, qty float64) (px, slip, filled float64, found bool) {
	if ladder == nil || at.IsZero() || qty <= 0 || slipped <= 0 {
		return 0, 0, 0, false
	}
	bids, asks, ok := ladder(f.Exchange, f.Symbol, at)
	if !ok {
		return 0, 0, 0, false
	}
	buy := bookBuys(f.Side, exit)
	levels := asks
	if !buy {
		levels = bids
	}
	ref := undoFixedSlip(buy, slipped, f.SlipRate)
	avg, slipFrac, got, walked := WalkStoredLadder(levels, qty, ref, buy)
	if !walked {
		return 0, 0, 0, true
	}
	return avg, slipFrac, got, true
}

// closeNotionalFromBook is the quote size the ladder actually filled.
// A full print returns notional. A thin book returns only the filled share.
func closeNotionalFromBook(notional, qty, filled float64) float64 {
	if notional <= 0 || qty <= 0 || filled <= 0 {
		return 0
	}
	if filled+1e-9 >= qty {
		return notional
	}
	return notional * (filled / qty)
}
