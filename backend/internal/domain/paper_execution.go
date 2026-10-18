package domain

import (
	"math"
	"strings"
)

const (
	// ExecutionMarket is one venue, taken now.
	ExecutionMarket ExecutionStyle = "market"
	// ExecutionBestCost splits the size across the spot books now.
	ExecutionBestCost ExecutionStyle = "best_cost"
	// ExecutionSpread repeats a best-cost slice, as if each slice finds this book again.
	ExecutionSpread ExecutionStyle = "spread"

	// ExecutionFilled means this path can take the whole size.
	ExecutionFilled ExecutionStatus = "filled"
	// ExecutionExhausted means the visible book cannot hold the size. Nothing fills.
	ExecutionExhausted ExecutionStatus = "exhausted"
	// ExecutionSlippageLimited means the slippage cap cannot hold the size. Nothing fills.
	ExecutionSlippageLimited ExecutionStatus = "slippage_limited"
	// ExecutionInventoryShort means a sell asks for more coins than that path can use.
	ExecutionInventoryShort ExecutionStatus = "inventory_short"
	// ExecutionMissing means the book this path needs did not load.
	ExecutionMissing ExecutionStatus = "missing"
	// ExecutionSchedule means the time split could not be named. The other paths still price.
	ExecutionSchedule ExecutionStatus = "schedule"

	// ExecutionScheduleInterval means the slices would be closer than 15 seconds.
	ExecutionScheduleInterval = "interval"
	// ExecutionScheduleRange means the minutes or the slice count is outside the allowed window.
	ExecutionScheduleRange = "range"
	// ExecutionScheduleSize means one slice would be below the smallest paper fill.
	ExecutionScheduleSize = "size"
)

// ExecutionCompareNote is the sentence stored with the three prices.
const ExecutionCompareNote = "Three prices for the same size. Market walks one venue now. Best cost walks Binance, Coinbase, and Bybit now and takes the cheapest fee-adjusted price. Over time repeats a slice of that walk as if each slice finds this same book. If the book does not refill, over time costs the same as best cost. Later prices are not known. Fees are the taker fee. Slippage is the average fill versus the best price across those books, before the fee. A path the book cannot hold fills nothing. A slippage cap applies to best cost and over time. Nothing is placed. Not real money."

// ExecutionStyle names one way to work the same size.
type ExecutionStyle string

// ExecutionStatus says whether that way can take the size.
type ExecutionStatus string

// ExecutionQuote is the fee and slippage of one way to work the size.
// SlippagePct is a percent: 0.5 means the average fill is 0.5% worse than the best price.
// FeePct is the taker fee as a percent of notional. Cash includes the fee.
// SavedVsMarket is set when this path and the single venue can both fill.
// Positive means a buy spends less, or a sell receives more, than the single venue.
type ExecutionQuote struct {
	Style          ExecutionStyle
	Status         ExecutionStatus
	Exchange       Exchange
	Symbol         string
	Quantity       float64
	FilledQuantity float64
	AveragePrice   float64
	TouchPrice     float64
	SlippagePct    float64
	Notional       float64
	Fee            float64
	FeePct         float64
	CashDebit      float64
	CashCredit     float64
	UnitCash       float64
	SavedVsMarket  *float64
	LevelsUsed     int
	Legs           []PaperRouteLeg
	Missing        []string
	Detail         string
	Schedule       string
	SpreadMinutes  int
	SliceCount     int
	SliceQuantity  float64
	// Assumption is "refill" when the time split priced every slice on this book.
	Assumption string
}

// ExecutionCompare is the three ways to work one size. Nothing is placed.
type ExecutionCompare struct {
	Side           TradeSide
	Symbol         string
	Quantity       float64
	Venue          Exchange
	TouchPrice     float64
	TouchExchange  Exchange
	MaxSlippagePct float64
	Rows           []ExecutionQuote
	Note           string
}

// CompareExecutions prices one venue now, best cost now, and a time split.
// maxSlippage is a fraction. Zero means best cost and the time split have no price cap.
// The single venue does not use that cap. plan nil means the time split could not be named;
// schedule is interval, range, or size, and the other two paths are still priced.
// A sell on the time split locks coins the way a placed spread order does, then refills the book between slices.
func CompareExecutions(side TradeSide, qty float64, currency, symbol string, venue Exchange, maxSlippage float64, plan *SpreadSchedule, schedule, rawDetail string, rawMinutes, rawSlices int, books []PaperRouteBook) ExecutionCompare {
	if side != TradeSideSell {
		side = TradeSideBuy
	}
	if maxSlippage < 0 || math.IsNaN(maxSlippage) || math.IsInf(maxSlippage, 0) {
		maxSlippage = 0
	}
	touch, touchEx := executionTouch(side, books)
	out := ExecutionCompare{
		Side: side, Symbol: symbol, Quantity: qty, Venue: venue,
		TouchPrice: touch, TouchExchange: touchEx,
		MaxSlippagePct: maxSlippage * 100, Note: ExecutionCompareNote,
	}
	marketBook, marketKnown := executionBook(venue, books)
	marketSymbol := symbol
	if marketBook != nil && marketBook.Symbol != "" {
		marketSymbol = marketBook.Symbol
	}
	var market ExecutionQuote
	if !marketKnown {
		market = ExecutionQuote{
			Style: ExecutionMarket, Status: ExecutionMissing, Exchange: venue, Symbol: marketSymbol,
			Quantity: qty, TouchPrice: touch, Legs: []PaperRouteLeg{},
			Detail: executionMissingDetail(marketBook),
		}
		if marketBook != nil && marketBook.Missing != "" {
			market.Missing = []string{string(venue)}
		}
	} else {
		// A normal market order walks the whole visible book. The cap belongs to best cost.
		routed := RoutePaperBooks(side, qty, currency, 0, []PaperRouteBook{*marketBook})
		market = quoteFromRoute(side, ExecutionMarket, venue, marketSymbol, qty, touch, routed)
	}
	best := quoteFromRoute(side, ExecutionBestCost, Exchange(PaperRouteAll), symbol, qty, touch,
		RoutePaperBooks(side, qty, currency, maxSlippage, books))
	spread := quoteSpreadExecution(side, qty, currency, symbol, touch, maxSlippage, plan, schedule, rawDetail, rawMinutes, rawSlices, books)
	out.Rows = []ExecutionQuote{market, best, spread}
	stampSavedVsMarket(side, out.Rows)
	return out
}

func quoteSpreadExecution(side TradeSide, qty float64, currency, symbol string, touch, maxSlippage float64, plan *SpreadSchedule, schedule, rawDetail string, rawMinutes, rawSlices int, books []PaperRouteBook) ExecutionQuote {
	row := ExecutionQuote{
		Style: ExecutionSpread, Exchange: Exchange(PaperRouteAll), Symbol: symbol,
		Quantity: qty, TouchPrice: touch, Legs: []PaperRouteLeg{},
		SpreadMinutes: rawMinutes, SliceCount: rawSlices,
	}
	if plan == nil {
		row.Status = ExecutionSchedule
		row.Schedule = schedule
		if row.Schedule == "" {
			row.Schedule = ExecutionScheduleRange
		}
		row.Detail = rawDetail
		return row
	}
	target := SpreadTarget(qty, 0, plan.SliceCount)
	row.SpreadMinutes = plan.Minutes
	row.SliceCount = plan.SliceCount
	row.SliceQuantity = target
	if target < MinTradeQuantity {
		row.Status = ExecutionSchedule
		row.Schedule = ExecutionScheduleSize
		row.Detail = "Each slice would be below the smallest paper fill."
		return row
	}
	routed, ok := quoteSpreadRefill(side, qty, currency, maxSlippage, *plan, books)
	if !ok {
		row.Status = ExecutionInventoryShort
		row.Detail = "The coins already held cannot cover every slice. Nothing is filled."
		return row
	}
	quoted := quoteFromRoute(side, ExecutionSpread, Exchange(PaperRouteAll), symbol, qty, touch, routed)
	quoted.SpreadMinutes = plan.Minutes
	quoted.SliceCount = plan.SliceCount
	quoted.SliceQuantity = target
	if quoted.Status == ExecutionFilled {
		quoted.Assumption = "refill"
		quoted.Detail = "Each slice is priced as if it finds this book again. If the book does not refill, the cost matches best cost."
	}
	return quoted
}

// quoteSpreadRefill prices every slice on the current book.
// A sell's coins do not come back: each slice can only use what the lock still holds.
// False means the venues cannot lock the whole sell.
func quoteSpreadRefill(side TradeSide, qty float64, currency string, maxSlippage float64, plan SpreadSchedule, books []PaperRouteBook) (PaperRoute, bool) {
	origin := books
	working := cloneRouteBooks(books)
	if side == TradeSideSell {
		held, ok := spreadLocks(qty, books)
		if !ok {
			return PaperRoute{Side: side, RequestedQuantity: qty, InventoryShort: true}, false
		}
		working = held
	}
	merged := PaperRoute{Side: side, RequestedQuantity: qty, Legs: []PaperRouteLeg{}}
	remaining := qty
	for i := 0; i < plan.SliceCount; i++ {
		left := plan.SliceCount - i
		target := SpreadTarget(remaining, 0, left)
		one := RoutePaperBooks(side, target, currency, maxSlippage, refillBooks(origin, working))
		if !executionComplete(one, target) {
			one.RequestedQuantity = qty
			return one, true
		}
		if i == 0 {
			merged.TouchPrice = one.TouchPrice
			merged.Missing = append([]string(nil), one.Missing...)
			merged.Skipped = append([]PaperRouteSkip(nil), one.Skipped...)
		}
		mergeExecution(&merged, one)
		reduceExecutionCaps(working, one)
		remaining -= one.FilledQuantity
	}
	if merged.FilledQuantity > 0 {
		merged.AveragePrice = merged.Notional / merged.FilledQuantity
	}
	if !executionComplete(merged, qty) {
		merged.Exhausted = true
		merged.Legs = nil
		merged.FilledQuantity = 0
		merged.Notional = 0
		merged.Fee = 0
		merged.CashDebit = 0
		merged.CashCredit = 0
	}
	return merged, true
}

func quoteFromRoute(side TradeSide, style ExecutionStyle, ex Exchange, symbol string, qty, touch float64, route PaperRoute) ExecutionQuote {
	q := ExecutionQuote{
		Style: style, Exchange: ex, Symbol: symbol, Quantity: qty, TouchPrice: touch,
		Missing: append([]string(nil), route.Missing...), Legs: []PaperRouteLeg{},
	}
	switch {
	case route.SlippageLimited:
		q.Status = ExecutionSlippageLimited
		q.Detail = "Not enough size within the slippage cap. Nothing is filled."
	case route.InventoryShort:
		q.Status = ExecutionInventoryShort
		q.Detail = "Not enough coins on this path. Nothing is filled."
	case !executionComplete(route, qty):
		if len(route.Legs) == 0 && len(route.Missing) > 0 && route.VisibleQuantity <= 1e-12 {
			q.Status = ExecutionMissing
			q.Detail = "No book could take this size."
		} else {
			q.Status = ExecutionExhausted
			q.Detail = "The visible book cannot hold this size. Nothing is filled."
		}
	default:
		q.Status = ExecutionFilled
		q.FilledQuantity = route.FilledQuantity
		q.AveragePrice = route.AveragePrice
		q.Notional = route.Notional
		q.Fee = route.Fee
		q.CashDebit = route.CashDebit
		q.CashCredit = route.CashCredit
		q.LevelsUsed = route.LevelsUsed
		q.Legs = append([]PaperRouteLeg(nil), route.Legs...)
		if route.Notional > 0 {
			q.FeePct = route.Fee / route.Notional * 100
		}
		if route.FilledQuantity > 0 {
			if side == TradeSideSell {
				q.UnitCash = route.CashCredit / route.FilledQuantity
			} else {
				q.UnitCash = route.CashDebit / route.FilledQuantity
			}
		}
		if touch > 0 && route.AveragePrice > 0 {
			if side == TradeSideSell {
				q.SlippagePct = (touch - route.AveragePrice) / touch * 100
			} else {
				q.SlippagePct = (route.AveragePrice - touch) / touch * 100
			}
		}
	}
	return q
}

func stampSavedVsMarket(side TradeSide, rows []ExecutionQuote) {
	var market *ExecutionQuote
	for i := range rows {
		if rows[i].Style == ExecutionMarket {
			market = &rows[i]
			break
		}
	}
	if market == nil || market.Status != ExecutionFilled {
		return
	}
	for i := range rows {
		if rows[i].Style == ExecutionMarket || rows[i].Status != ExecutionFilled {
			continue
		}
		var saved float64
		if side == TradeSideSell {
			saved = rows[i].CashCredit - market.CashCredit
		} else {
			saved = market.CashDebit - rows[i].CashDebit
		}
		rows[i].SavedVsMarket = &saved
	}
}

func executionComplete(route PaperRoute, qty float64) bool {
	return qty > 0 && !route.Exhausted && !route.InventoryShort && !route.SlippageLimited &&
		route.FilledQuantity+1e-12 >= qty && len(route.Legs) > 0
}

func executionBook(venue Exchange, books []PaperRouteBook) (*PaperRouteBook, bool) {
	for i := range books {
		if books[i].Exchange != venue {
			continue
		}
		if strings.TrimSpace(books[i].Missing) != "" || strings.TrimSpace(books[i].Skipped) != "" {
			return &books[i], false
		}
		return &books[i], true
	}
	return nil, false
}

func executionMissingDetail(book *PaperRouteBook) string {
	if book == nil {
		return "That venue has no book in this comparison."
	}
	if strings.TrimSpace(book.Missing) != "" {
		return "That venue's book did not load. Nothing is filled."
	}
	if book.Skipped != "" {
		return "That venue's pair is outside this book's currency. Nothing is filled."
	}
	return "That venue's book did not load. Nothing is filled."
}

func executionTouch(side TradeSide, books []PaperRouteBook) (float64, Exchange) {
	probed := RoutePaperBooks(side, 1e-8, "", 0, books)
	touch := probed.TouchPrice
	if touch <= 0 {
		return 0, ""
	}
	var best Exchange
	for _, b := range books {
		if strings.TrimSpace(b.Missing) != "" || strings.TrimSpace(b.Skipped) != "" {
			continue
		}
		levels := b.Asks
		if side == TradeSideSell {
			levels = b.Bids
		}
		for _, lv := range levels {
			if !validRouteLevel(lv) || math.Abs(lv.Price-touch) > touch*1e-9 {
				continue
			}
			if best == "" || b.Exchange < best {
				best = b.Exchange
			}
		}
	}
	return touch, best
}

func spreadLocks(qty float64, books []PaperRouteBook) ([]PaperRouteBook, bool) {
	free := make([]SpreadFree, 0, len(books))
	for _, b := range books {
		if strings.TrimSpace(b.Missing) != "" || strings.TrimSpace(b.Skipped) != "" || b.CapQty == nil {
			continue
		}
		free = append(free, SpreadFree{Exchange: b.Exchange, Symbol: b.Symbol, Free: *b.CapQty})
	}
	holds, ok := AllocateSpreadHolds(qty, free)
	if !ok {
		return nil, false
	}
	out := cloneRouteBooks(books)
	qtyBy := map[string]float64{}
	for _, h := range holds {
		qtyBy[string(h.Exchange)+"|"+h.Symbol] = h.Quantity
	}
	for i := range out {
		if strings.TrimSpace(out[i].Missing) != "" || strings.TrimSpace(out[i].Skipped) != "" {
			continue
		}
		v := qtyBy[string(out[i].Exchange)+"|"+out[i].Symbol]
		out[i].CapQty = &v
	}
	return out, true
}

func cloneRouteBooks(books []PaperRouteBook) []PaperRouteBook {
	out := make([]PaperRouteBook, len(books))
	for i := range books {
		out[i] = books[i]
		out[i].Bids = append([]PriceLevel(nil), books[i].Bids...)
		out[i].Asks = append([]PriceLevel(nil), books[i].Asks...)
		if books[i].CapQty != nil {
			v := *books[i].CapQty
			out[i].CapQty = &v
		}
	}
	return out
}

func refillBooks(origin, caps []PaperRouteBook) []PaperRouteBook {
	out := cloneRouteBooks(origin)
	for i := range out {
		if i >= len(caps) || caps[i].CapQty == nil {
			continue
		}
		v := *caps[i].CapQty
		out[i].CapQty = &v
	}
	return out
}

func reduceExecutionCaps(books []PaperRouteBook, route PaperRoute) {
	for _, leg := range route.Legs {
		for i := range books {
			if books[i].Exchange != leg.Exchange || books[i].CapQty == nil {
				continue
			}
			left := *books[i].CapQty - leg.Quantity
			if left < 0 {
				left = 0
			}
			*books[i].CapQty = left
		}
	}
}

func mergeExecution(into *PaperRoute, one PaperRoute) {
	into.Notional += one.Notional
	into.Fee += one.Fee
	into.CashDebit += one.CashDebit
	into.CashCredit += one.CashCredit
	into.FilledQuantity += one.FilledQuantity
	into.LevelsUsed += one.LevelsUsed
	into.VisibleQuantity = one.VisibleQuantity
	for _, leg := range one.Legs {
		found := false
		for i := range into.Legs {
			if into.Legs[i].Exchange != leg.Exchange {
				continue
			}
			dst := &into.Legs[i]
			dst.Quantity += leg.Quantity
			dst.Notional += leg.Notional
			dst.Fee += leg.Fee
			dst.CashDebit += leg.CashDebit
			dst.CashCredit += leg.CashCredit
			dst.LevelsUsed += leg.LevelsUsed
			dst.EndPrice = leg.EndPrice
			if dst.Quantity > 0 {
				dst.FillPrice = dst.Notional / dst.Quantity
			}
			found = true
			break
		}
		if !found {
			into.Legs = append(into.Legs, leg)
		}
	}
}
