package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	// SpreadDefaultSlices is the slice count when a request omits one.
	SpreadDefaultSlices = 10
	// SpreadDefaultMinutes is the duration when a request omits one.
	SpreadDefaultMinutes = 10
	// SpreadMinSlices is the smallest schedule. One slice is an immediate order.
	SpreadMinSlices = 2
	// SpreadMaxSlices bounds how often the book is read.
	SpreadMaxSlices = 60
	// SpreadMinMinutes is the shortest working window.
	SpreadMinMinutes = 1
	// SpreadMaxMinutes is one day.
	SpreadMaxMinutes = 24 * 60
	// SpreadMinInterval is the closest two slices may be. The worker passes about this often.
	SpreadMinInterval = 15 * time.Second
	// SpreadReserveBuffer is the cash lock above the best ask when the order has no slippage cap.
	// 0.01 means 1%. A cap, when set, replaces this buffer.
	SpreadReserveBuffer = 0.01
)

// SpreadSchedule is how a best-cost order is cut across a clock.
// The first slice is due when the order is placed. The last slice is due when Duration ends.
// The gap is Duration / (SliceCount - 1).
type SpreadSchedule struct {
	SliceCount int
	Minutes    int
	Duration   time.Duration
	Interval   time.Duration
}

// NormalizeSpreadSchedule reads a duration in minutes and a slice count.
// Zero for both means 10 slices over 10 minutes. A gap shorter than 15 seconds is rejected.
func NormalizeSpreadSchedule(minutes, slices int) (SpreadSchedule, error) {
	if minutes == 0 && slices == 0 {
		minutes = SpreadDefaultMinutes
		slices = SpreadDefaultSlices
	}
	if minutes == 0 {
		minutes = SpreadDefaultMinutes
	}
	if slices == 0 {
		slices = SpreadDefaultSlices
	}
	if slices < SpreadMinSlices || slices > SpreadMaxSlices {
		return SpreadSchedule{}, fmt.Errorf("%w: slices must be from %d to %d", ErrInvalidArgument, SpreadMinSlices, SpreadMaxSlices)
	}
	if minutes < SpreadMinMinutes || minutes > SpreadMaxMinutes {
		return SpreadSchedule{}, fmt.Errorf("%w: duration must be from %d to %d minutes", ErrInvalidArgument, SpreadMinMinutes, SpreadMaxMinutes)
	}
	duration := time.Duration(minutes) * time.Minute
	interval := duration / time.Duration(slices-1)
	if interval < SpreadMinInterval {
		return SpreadSchedule{}, fmt.Errorf("%w: slices would be closer than %s", ErrInvalidArgument, SpreadMinInterval)
	}
	return SpreadSchedule{SliceCount: slices, Minutes: minutes, Duration: duration, Interval: interval}, nil
}

// IntervalSec is the gap rounded to the nearest second, for a preview line.
func (s SpreadSchedule) IntervalSec() int {
	if s.Interval <= 0 {
		return 0
	}
	return int((s.Interval + time.Second/2) / time.Second)
}

// ExplainSpread is the English preview sentence for this schedule.
func ExplainSpread(slices, minutes int) string {
	return fmt.Sprintf("Preview only — nothing was placed. This best-cost order works as %d slices over %d minutes. The first slice is due when you place it. The last slice is due at the end. Each slice reads Binance, Coinbase, and Bybit again and takes the cheapest fee-adjusted size. The legs are the first slice on this book. Later slices can use a different venue. A buy locks cash from the best ask up front. A sell locks quantity on the venues that already hold the coin. Not real money.", slices, minutes)
}

// ExplainSpreadPlace is returned when that schedule starts. No slice has filled yet.
func ExplainSpreadPlace(slices, minutes int) string {
	return fmt.Sprintf("Spread order is working. %d slices over %d minutes. The first slice is due now. The last slice is due at the end. Each slice reads the books again and takes the cheapest venue for that size. Cancel releases what has not filled. Paper only.", slices, minutes)
}

// SpreadStatus is the life of a working spread order.
type SpreadStatus string

const (
	SpreadStatusOpen     SpreadStatus = "open"
	SpreadStatusFilled   SpreadStatus = "filled"
	SpreadStatusCanceled SpreadStatus = "canceled"
)

// SpreadHold is quantity a spread sell has locked on one venue.
type SpreadHold struct {
	Exchange Exchange
	Symbol   string
	Quantity float64
}

// SpreadFree is the unsold quantity a venue can still offer a new spread sell.
type SpreadFree struct {
	Exchange Exchange
	Symbol   string
	Free     float64
}

// SpreadOrder is a spot market order filled in slices.
// Each slice re-reads the books and takes the cheapest venue for that size.
type SpreadOrder struct {
	ID                string
	ClientID          string
	Symbol            string
	Side              TradeSide
	Quantity          float64
	FilledQuantity    float64
	RemainingQuantity float64
	SliceCount        int
	SlicesDone        int
	// Duration is the clock from the first slice to the last.
	Duration    time.Duration
	NextSliceAt time.Time
	EndsAt      time.Time
	Status      SpreadStatus
	// MaxSlippage is a fraction. Zero means no price cap.
	MaxSlippage    float64
	ReservedCash   float64
	FilledNotional float64
	FilledFee      float64
	LotMethod      LotMethod
	ReasonSet      bool
	Reason         []string
	CancelReason   string
	Holds          []SpreadHold
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SliceInterval is the gap stored on this order.
// The last slice is due Duration after the first. A missing duration keeps a one-minute gap.
func (o SpreadOrder) SliceInterval() time.Duration {
	if o.SliceCount < 2 {
		return time.Minute
	}
	d := o.Duration
	if d <= 0 {
		d = time.Duration(o.SliceCount-1) * time.Minute
	}
	gap := d / time.Duration(o.SliceCount-1)
	if gap < time.Second {
		return time.Second
	}
	return gap
}

// DurationMinutes is the working window in whole minutes.
func (o SpreadOrder) DurationMinutes() int {
	if o.Duration <= 0 {
		return 0
	}
	return int(o.Duration / time.Minute)
}

// SpreadTarget is the size of the next slice.
// It is the quantity still open divided by the slices still to attempt.
func SpreadTarget(remaining float64, slicesDone, sliceCount int) float64 {
	if remaining <= 0 || math.IsNaN(remaining) || math.IsInf(remaining, 0) {
		return 0
	}
	left := sliceCount - slicesDone
	if left < 1 {
		left = 1
	}
	return remaining / float64(left)
}

// SpreadReservePrice is the raw price a buy lock is built from.
// A slippage cap uses that cap. Otherwise the lock sits 1% above the best ask.
func SpreadReservePrice(touch, maxSlippage float64) float64 {
	if touch <= 0 || math.IsNaN(touch) || math.IsInf(touch, 0) {
		return 0
	}
	if maxSlippage > 0 && !math.IsNaN(maxSlippage) && !math.IsInf(maxSlippage, 0) {
		return touch * (1 + maxSlippage)
	}
	return touch * (1 + SpreadReserveBuffer)
}

// SpreadBuyReserve is the cash locked for the whole buy, including the fee.
func SpreadBuyReserve(qty, touch, maxSlippage, feeRate float64) float64 {
	price := SpreadReservePrice(touch, maxSlippage)
	if qty <= 0 || price <= 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
		return 0
	}
	if feeRate < 0 || math.IsNaN(feeRate) || math.IsInf(feeRate, 0) {
		feeRate = 0
	}
	return qty * price * (1 + feeRate)
}

// AllocateSpreadHolds parks a sell across the venues that already hold the coin.
// The shares follow the free quantity, and they sum to qty. False means the venues cannot cover qty.
func AllocateSpreadHolds(qty float64, free []SpreadFree) ([]SpreadHold, bool) {
	if qty <= 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
		return nil, false
	}
	rows := make([]SpreadFree, 0, len(free))
	var sum float64
	for _, row := range free {
		if row.Free > PositionEpsilon && row.Symbol != "" && row.Exchange != "" {
			rows = append(rows, row)
			sum += row.Free
		}
	}
	if sum+1e-9 < qty {
		return nil, false
	}
	holds := make([]SpreadHold, 0, len(rows))
	left := qty
	for i, row := range rows {
		part := qty * row.Free / sum
		if i == len(rows)-1 || part > left {
			part = left
		}
		if part > row.Free {
			part = row.Free
		}
		if part <= PositionEpsilon {
			continue
		}
		holds = append(holds, SpreadHold{Exchange: row.Exchange, Symbol: row.Symbol, Quantity: part})
		left -= part
	}
	if left > 1e-8 {
		return nil, false
	}
	return holds, true
}

// ReduceSpreadHolds subtracts a fill from the locks and drops empty ones.
func ReduceSpreadHolds(holds []SpreadHold, filled []SpreadHold) []SpreadHold {
	used := map[string]float64{}
	for _, row := range filled {
		used[string(row.Exchange)+"|"+row.Symbol] += row.Quantity
	}
	out := make([]SpreadHold, 0, len(holds))
	for _, row := range holds {
		left := row.Quantity - used[string(row.Exchange)+"|"+row.Symbol]
		if left <= PositionEpsilon {
			continue
		}
		row.Quantity = left
		out = append(out, row)
	}
	return out
}

// SpreadBuyRelease is the reserved cash a filled slice gives back.
// It releases the slice's share of the lock, and at least the cash the fill actually spent.
func SpreadBuyRelease(reserved, remaining, filled, spent float64) float64 {
	if reserved <= 0 || filled <= 0 {
		return 0
	}
	share := reserved
	if remaining > PositionEpsilon {
		share = reserved * (filled / remaining)
	}
	if spent > share {
		share = spent
	}
	if share > reserved {
		share = reserved
	}
	if share < 0 {
		return 0
	}
	return share
}
