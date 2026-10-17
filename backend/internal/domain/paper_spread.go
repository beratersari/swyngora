package domain

import (
	"math"
	"time"
)

const (
	// SpreadSliceCount is how many child fills a spread order attempts.
	SpreadSliceCount = 10
	// SpreadInterval is the time between those child fills.
	SpreadInterval = time.Minute
	// SpreadReserveBuffer is the cash lock above the best ask when the order has no slippage cap.
	// 0.01 means 1%. A cap, when set, replaces this buffer.
	SpreadReserveBuffer = 0.01
)

// SpreadStatus is the life of a working spread order.
type SpreadStatus string

const (
	SpreadStatusOpen     SpreadStatus = "open"
	SpreadStatusFilled   SpreadStatus = "filled"
	SpreadStatusCanceled SpreadStatus = "canceled"
)

// SpreadPreviewNote tells the assistant what a spread preview is pricing.
const SpreadPreviewNote = "Preview only — nothing was placed. This best-cost order works as 10 slices, one every minute. Each slice reads Binance, Coinbase, and Bybit again and takes the cheapest fee-adjusted size. The legs are the first slice on this book. Later slices can use a different venue. A buy locks cash from the best ask up front. A sell locks quantity on the venues that already hold the coin. Not real money."

// SpreadPlaceNote is returned when a spread order starts. No slice has filled yet.
const SpreadPlaceNote = "Spread order is working. Ten slices, one every minute. The first slice is due now. Each slice reads the books again and takes the cheapest venue for that size. Cancel releases what has not filled. Paper only."

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
	NextSliceAt       time.Time
	EndsAt            time.Time
	Status            SpreadStatus
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

// SpreadSpan is the time from the first slice to the last scheduled slice.
func SpreadSpan() time.Duration {
	if SpreadSliceCount < 2 {
		return 0
	}
	return time.Duration(SpreadSliceCount-1) * SpreadInterval
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
