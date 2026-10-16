package domain

import (
	"math"
	"time"
)

// Price move after a paper fill. The percent is the venue price change.
// It is not profit: a sell with a positive percent means price rose after the sale.
const (
	FillMove1h  = "1h"
	FillMove4h  = "4h"
	FillMove24h = "24h"

	// FillMovePending means the one-minute bar that contains the horizon is still open.
	FillMovePending FillMoveStatus = "pending"
	// FillMoveReady means both the fill price and that minute's close are known.
	FillMoveReady FillMoveStatus = "ready"
	// FillMoveMissing means the window is due and the minute close is absent.
	// A missing window is not zero.
	FillMoveMissing FillMoveStatus = "missing"
)

// FillMoveStatus is pending, ready, or missing.
type FillMoveStatus string

// FillMove is one horizon after a paper fill.
// Pct and Price are set only when Status is ready, including a real 0%.
type FillMove struct {
	Window string         `json:"window"`
	Status FillMoveStatus `json:"status"`
	Pct    *float64       `json:"pct,omitempty"`
	Price  *float64       `json:"price,omitempty"`
}

// FillMoveWindow is a named horizon measured from the fill.
type FillMoveWindow struct {
	Name string
	Span time.Duration
}

// FillMoveWindows is 1h, then 4h, then 24h.
func FillMoveWindows() []FillMoveWindow {
	return []FillMoveWindow{
		{Name: FillMove1h, Span: time.Hour},
		{Name: FillMove4h, Span: 4 * time.Hour},
		{Name: FillMove24h, Span: 24 * time.Hour},
	}
}

// FillMoveHorizon is the instant fillAt plus the horizon, in UTC.
func FillMoveHorizon(fillAt time.Time, span time.Duration) time.Time {
	if fillAt.IsZero() {
		return time.Time{}
	}
	return fillAt.UTC().Add(span)
}

// FillMoveBarOpen is the open of the one-minute bar that contains horizon.
func FillMoveBarOpen(horizon time.Time) time.Time {
	if horizon.IsZero() {
		return time.Time{}
	}
	return horizon.UTC().Truncate(time.Minute)
}

// FillMoveDue reports whether that one-minute bar has closed.
// The bar closes at its open plus one minute, so a still-forming minute stays pending.
func FillMoveDue(horizon, now time.Time) bool {
	if horizon.IsZero() || now.IsZero() {
		return false
	}
	open := FillMoveBarOpen(horizon)
	return !now.UTC().Before(open.Add(time.Minute))
}

// FillMoves is the 1h, 4h, and 24h venue move after a fill.
// sample is called only for a due window, with the horizon instant, and returns
// the close of the one-minute bar that contains it. A zero or negative fill,
// or a zero fill time, is missing and does not call sample. A sample that is
// absent or not a positive price is missing, not 0%. Equal prices are a ready 0%.
func FillMoves(fillPrice float64, fillAt, now time.Time, sample func(horizon time.Time) (price float64, found bool)) []FillMove {
	out := make([]FillMove, 0, len(FillMoveWindows()))
	invalid := !validMovePrice(fillPrice) || fillAt.IsZero()
	if !now.IsZero() {
		now = now.UTC()
	}
	for _, w := range FillMoveWindows() {
		if invalid {
			out = append(out, FillMove{Window: w.Name, Status: FillMoveMissing})
			continue
		}
		horizon := FillMoveHorizon(fillAt, w.Span)
		if !FillMoveDue(horizon, now) {
			out = append(out, FillMove{Window: w.Name, Status: FillMovePending})
			continue
		}
		if sample == nil {
			out = append(out, FillMove{Window: w.Name, Status: FillMoveMissing})
			continue
		}
		px, found := sample(horizon)
		if !found || !validMovePrice(px) {
			out = append(out, FillMove{Window: w.Name, Status: FillMoveMissing})
			continue
		}
		pct := (px - fillPrice) / fillPrice * 100
		price := px
		out = append(out, FillMove{Window: w.Name, Status: FillMoveReady, Pct: &pct, Price: &price})
	}
	return out
}

func validMovePrice(price float64) bool {
	return price > 0 && !math.IsNaN(price) && !math.IsInf(price, 0)
}
