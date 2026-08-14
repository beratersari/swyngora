package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// Paper margin trading limits (simulated only).
const (
	MinMarginLeverage      = 1
	MaxMarginLeverage      = 10
	MaxOpenMarginPositions = 20
	MaxOpenMarginOrders    = 50
	// DefaultMaintenanceMarginRate is fraction of notional retained as maintenance (0.5%).
	DefaultMaintenanceMarginRate = 0.005
	// DefaultMarginHourlyInterestRate is simple interest per full hour on principal
	// (~0.02%/day ≈ 0.000833%/hour; we use 0.001%/hour for paper clarity).
	DefaultMarginHourlyInterestRate = 0.00001
	// Margin close reasons.
	MarginCloseUser        = "user"
	MarginCloseLiquidation = "liquidation"
	MarginCloseStopLoss    = "stop_loss"
	MarginCloseTakeProfit  = "take_profit"
	MarginClosePartialUser = "partial_close"
	// Margin trade actions (also used for adjust/repay).
	MarginActionAddMargin    = "add_margin"
	MarginActionRemoveMargin = "remove_margin"
	MarginActionRepay        = "repay"
	MarginActionInterest     = "interest"
)

// DebtAsset is the unit of borrowed principal/interest.
type DebtAsset string

const (
	// DebtAssetQuote: long positions borrow cash (portfolio currency).
	DebtAssetQuote DebtAsset = "quote"
	// DebtAssetBase: short positions borrow the base coin.
	DebtAssetBase DebtAsset = "base"
)

// MarginMode is account-wide margin style (locked while any open pos/order).
type MarginMode string

const (
	// MarginModeIsolated: only margin assigned to a position backs that position.
	MarginModeIsolated MarginMode = "isolated"
	// MarginModeCross: wallet equity is shared across open margin positions.
	MarginModeCross MarginMode = "cross"
)

// MarginSide is long or short.
type MarginSide string

const (
	MarginLong  MarginSide = "long"
	MarginShort MarginSide = "short"
)

// MarginOrderType is market or limit open order.
type MarginOrderType string

const (
	MarginOrderMarket MarginOrderType = "market"
	MarginOrderLimit  MarginOrderType = "limit"
)

// MarginOrderStatus is the lifecycle of a resting margin open order.
type MarginOrderStatus string

const (
	MarginOrderOpen     MarginOrderStatus = "open"
	MarginOrderFilled   MarginOrderStatus = "filled"
	MarginOrderCanceled MarginOrderStatus = "canceled"
	MarginOrderRejected MarginOrderStatus = "rejected"
)

// MarginPositionStatus is open or closed.
type MarginPositionStatus string

const (
	MarginPositionOpen   MarginPositionStatus = "open"
	MarginPositionClosed MarginPositionStatus = "closed"
)

// MarginPosition is a leveraged long/short paper position (isolated or cross).
type MarginPosition struct {
	ID               string
	ClientID         string
	Exchange         Exchange
	Symbol           string
	Side             MarginSide
	Mode             MarginMode // snapshot of account mode at open
	Quantity         float64    // remaining open size
	EntryPrice       float64
	Leverage         int
	Margin           float64 // margin assigned (isolated: only this backs the pos; cross: IM share)
	// DebtPrincipal: long = borrowed cash; short = borrowed base coins.
	DebtPrincipal float64
	// DebtInterest: accrued interest in the same unit as DebtPrincipal.
	DebtInterest float64
	// DebtAsset is quote (cash) for long, base (coin) for short.
	DebtAsset DebtAsset
	// LastInterestAt is the last hour boundary when interest was applied.
	LastInterestAt   time.Time
	LiquidationPrice float64
	StopLoss         *float64
	TakeProfit       *float64
	Status           MarginPositionStatus
	UnrealizedPnL    float64 // view-only / last mark
	MarkPrice        float64 // view-only
	// DebtNotional is view-only: principal+interest in quote terms (short uses mark).
	DebtNotional  float64
	RealizedPnL   float64 // cumulative realized on this position (partials + final)
	CloseReason   string
	OpenedAt      time.Time
	UpdatedAt     time.Time
	ClosedAt      *time.Time
}

// MarginOrder is a pending limit open for margin (market fills immediately).
type MarginOrder struct {
	ID            string
	ClientID      string
	Exchange      Exchange
	Symbol        string
	Side          MarginSide
	Type          MarginOrderType
	Quantity      float64
	Leverage      int
	LimitPrice    float64 // required for limit
	ReservedMargin float64
	StopLoss      *float64
	TakeProfit    *float64
	Status        MarginOrderStatus
	PositionID    string // set when filled
	RejectReason  string
	CancelReason  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	FilledAt      *time.Time
	CanceledAt    *time.Time
}

// MarginTrade is a margin open/close/repay fill record.
type MarginTrade struct {
	ID           string
	ClientID     string
	PositionID   string
	Exchange     Exchange
	Symbol       string
	Side         MarginSide
	Action       string // open | close | liquidation | stop_loss | take_profit | repay | interest
	Quantity     float64
	Price        float64
	Notional     float64
	RealizedPnL  float64
	MarginDelta  float64 // cash change from margin lock/release (negative open, positive release)
	// PrincipalPaid / InterestPaid track debt reduction on repay or partial close.
	PrincipalPaid float64
	InterestPaid  float64
	Leverage      int
	Fee           float64
	CreatedAt     time.Time
}

// SystemCloseTradeID returns a deterministic trade id for full forced closes
// (liquidation / SL / TP). Empty when the action is not a forced system close.
// Using a stable id + unique index makes a second run after restart a no-op
// (cannot insert a second close record or re-apply cash).
func SystemCloseTradeID(action, positionID string) string {
	positionID = strings.TrimSpace(positionID)
	if positionID == "" {
		return ""
	}
	switch action {
	case MarginCloseLiquidation:
		return "margin-liq:" + positionID
	case MarginCloseStopLoss:
		return "margin-sl:" + positionID
	case MarginCloseTakeProfit:
		return "margin-tp:" + positionID
	default:
		return ""
	}
}

// IsSystemForcedCloseAction reports liquidation / stop_loss / take_profit trade actions.
func IsSystemForcedCloseAction(action string) bool {
	switch action {
	case MarginCloseLiquidation, MarginCloseStopLoss, MarginCloseTakeProfit:
		return true
	default:
		return false
	}
}

// IsValidMarginMode reports isolated|cross.
func IsValidMarginMode(s string) bool {
	switch MarginMode(strings.ToLower(strings.TrimSpace(s))) {
	case MarginModeIsolated, MarginModeCross:
		return true
	default:
		return false
	}
}

// NormalizeMarginMode parses isolated|cross; empty defaults to isolated.
func NormalizeMarginMode(s string) (MarginMode, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return MarginModeIsolated, nil
	}
	if !IsValidMarginMode(s) {
		return "", fmt.Errorf("%w: marginMode must be isolated or cross", ErrInvalidArgument)
	}
	return MarginMode(s), nil
}

// IsValidMarginSide reports long|short.
func IsValidMarginSide(s string) bool {
	switch MarginSide(strings.ToLower(strings.TrimSpace(s))) {
	case MarginLong, MarginShort:
		return true
	default:
		return false
	}
}

// NormalizeMarginSide parses long|short.
func NormalizeMarginSide(s string) (MarginSide, error) {
	side := MarginSide(strings.ToLower(strings.TrimSpace(s)))
	if !IsValidMarginSide(string(side)) {
		return "", fmt.Errorf("%w: side must be long or short", ErrInvalidArgument)
	}
	return side, nil
}

// IsValidMarginOrderType reports market|limit.
func IsValidMarginOrderType(s string) bool {
	switch MarginOrderType(strings.ToLower(strings.TrimSpace(s))) {
	case MarginOrderMarket, MarginOrderLimit:
		return true
	default:
		return false
	}
}

// NormalizeMarginOrderType parses market|limit; empty defaults to market.
func NormalizeMarginOrderType(s string) (MarginOrderType, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return MarginOrderMarket, nil
	}
	if !IsValidMarginOrderType(s) {
		return "", fmt.Errorf("%w: type must be market or limit", ErrInvalidArgument)
	}
	return MarginOrderType(s), nil
}

// IsValidMarginLeverage reports 1..10 inclusive.
func IsValidMarginLeverage(lev int) bool {
	return lev >= MinMarginLeverage && lev <= MaxMarginLeverage
}

// InitialMargin is notional / leverage = qty * price / leverage.
func InitialMargin(qty, price float64, leverage int) (float64, error) {
	if qty <= 0 || price <= 0 || math.IsNaN(qty) || math.IsNaN(price) || math.IsInf(qty, 0) || math.IsInf(price, 0) {
		return 0, fmt.Errorf("%w: quantity and price must be positive", ErrInvalidArgument)
	}
	if !IsValidMarginLeverage(leverage) {
		return 0, fmt.Errorf("%w: leverage must be between %d and %d", ErrInvalidArgument, MinMarginLeverage, MaxMarginLeverage)
	}
	return qty * price / float64(leverage), nil
}

// BorrowedPrincipalOnOpen returns debt principal and asset at open.
// Long: borrowed cash = notional - margin. Short: borrowed coins = qty.
func BorrowedPrincipalOnOpen(side MarginSide, qty, price float64, leverage int) (principal float64, asset DebtAsset, err error) {
	im, err := InitialMargin(qty, price, leverage)
	if err != nil {
		return 0, "", err
	}
	notional := qty * price
	switch side {
	case MarginLong:
		// Cash borrowed to fund the long beyond own margin.
		b := notional - im
		if b < 0 {
			b = 0
		}
		return b, DebtAssetQuote, nil
	case MarginShort:
		return qty, DebtAssetBase, nil
	default:
		return 0, "", fmt.Errorf("%w: side must be long or short", ErrInvalidArgument)
	}
}

// DebtNotionalQuote values outstanding debt in quote currency (short uses mark for coins).
func DebtNotionalQuote(side MarginSide, principal, interest, mark float64) float64 {
	total := principal + interest
	if total <= 0 {
		return 0
	}
	switch side {
	case MarginLong:
		return total // already quote
	case MarginShort:
		if mark <= 0 {
			return 0
		}
		return total * mark
	default:
		return 0
	}
}

// AccrueInterestHours applies simple interest on principal for full elapsed hours in O(1).
// It never loops hour-by-hour: hours = floor((now-lastAt)/1h), add = principal*rate*hours.
//
// Monotonic / clock-safe:
//   - if now is not strictly after lastAt (clock skew or already accrued), returns zero hours
//     and leaves interest/lastAt unchanged (never reduces interest or rewinds lastAt)
//   - newLast is always lastAt + hours (forward only)
// Fully paid principal (principal <= 0) accrues nothing.
//
// Returns new interest total, new lastInterestAt, full hours applied.
func AccrueInterestHours(principal, interest float64, lastAt, now time.Time, hourlyRate float64) (newInterest float64, newLast time.Time, hours int) {
	if principal <= PositionEpsilon || hourlyRate <= 0 {
		return interest, lastAt, 0
	}
	now = now.UTC()
	// No baseline yet (caller should set last_interest_at on open) — do not invent past hours.
	if lastAt.IsZero() {
		return interest, lastAt, 0
	}
	lastAt = lastAt.UTC()
	// Clock moved backward or same instant: do not remove or re-add interest.
	if !now.After(lastAt) {
		return interest, lastAt, 0
	}
	// O(1) catch-up for offline periods (no per-hour loop).
	hours = int(now.Sub(lastAt) / time.Hour)
	if hours <= 0 {
		return interest, lastAt, 0
	}
	add := principal * hourlyRate * float64(hours)
	newLast = lastAt.Add(time.Duration(hours) * time.Hour)
	// Guard: never move lastAt backward (should be impossible given hours > 0).
	if newLast.Before(lastAt) {
		return interest, lastAt, 0
	}
	return interest + add, newLast, hours
}

// AllocateRepayment pays interest first, then principal. amount is in debt units.
func AllocateRepayment(principal, interest, amount float64) (principalPaid, interestPaid, newPrincipal, newInterest float64) {
	if amount <= 0 {
		return 0, 0, principal, interest
	}
	if amount >= interest {
		interestPaid = interest
		rest := amount - interest
		if rest > principal {
			rest = principal
		}
		principalPaid = rest
	} else {
		interestPaid = amount
	}
	newInterest = interest - interestPaid
	newPrincipal = principal - principalPaid
	if newInterest < 0 {
		newInterest = 0
	}
	if newPrincipal < 0 {
		newPrincipal = 0
	}
	return principalPaid, interestPaid, newPrincipal, newInterest
}

// MaxBorrowNotional is the account debt ceiling in quote terms (startingBalance * (maxLev-1)).
func MaxBorrowNotional(startingBalance float64) float64 {
	if startingBalance <= 0 {
		return 0
	}
	return startingBalance * float64(MaxMarginLeverage-1)
}

// MaintenanceMargin is mmr * notional for open size.
func MaintenanceMargin(qty, entry, mmr float64) float64 {
	if qty <= 0 || entry <= 0 || mmr <= 0 {
		return 0
	}
	return qty * entry * mmr
}

// LiquidationPriceFromMargin computes liq from assigned margin without open interest.
// Prefer LiquidationPriceWithDebt when debt/interest are tracked.
func LiquidationPriceFromMargin(side MarginSide, entry, qty, margin, mmr float64) (float64, error) {
	return LiquidationPriceWithDebt(side, entry, qty, margin, 0, 0, mmr)
}

// LiquidationPriceWithDebt includes growing interest and assigned margin.
//
// Long (isolated equity = margin + (mark-entry)*qty - interest):
//
//	mark = entry - (margin - maint - interest) / qty
//	Extra margin lowers liq; interest raises liq (worse). Principal is already in entry/margin split.
//
// Short (owe principal+interest coins C):
//
//	mark = (margin + entry*qty - maint) / C
func LiquidationPriceWithDebt(side MarginSide, entry, qty, margin, debtPrincipal, debtInterest, mmr float64) (float64, error) {
	if entry <= 0 || qty <= 0 || math.IsNaN(entry) || math.IsNaN(qty) || math.IsInf(entry, 0) || math.IsInf(qty, 0) {
		return 0, fmt.Errorf("%w: entry and quantity must be positive", ErrInvalidArgument)
	}
	if margin < 0 || math.IsNaN(margin) || math.IsInf(margin, 0) {
		return 0, fmt.Errorf("%w: margin must be non-negative", ErrInvalidArgument)
	}
	if mmr < 0 || mmr >= 1 || math.IsNaN(mmr) {
		return 0, fmt.Errorf("%w: invalid maintenance margin rate", ErrInvalidArgument)
	}
	if debtPrincipal < 0 {
		debtPrincipal = 0
	}
	if debtInterest < 0 {
		debtInterest = 0
	}
	maint := MaintenanceMargin(qty, entry, mmr)
	switch side {
	case MarginLong:
		// Interest is cash liability on top of posted margin.
		// buffer may be negative when interest > margin-maint → liq rises above entry
		// so ShouldLiquidate still fires while the book is under maintenance.
		// Do not clamp buffer to 0.
		buffer := margin - maint - debtInterest
		p := entry - buffer/qty
		if p < 0 {
			p = 0
		}
		return p, nil
	case MarginShort:
		c := debtPrincipal + debtInterest
		if c <= PositionEpsilon {
			c = qty
		}
		num := margin + entry*qty - maint
		if num < 0 {
			num = 0
		}
		return num / c, nil
	default:
		return 0, fmt.Errorf("%w: side must be long or short", ErrInvalidArgument)
	}
}

// LiquidationPriceIsolated computes liq assuming initial margin = notional/leverage.
// Equivalent to LiquidationPriceFromMargin with that IM.
func LiquidationPriceIsolated(side MarginSide, entry float64, leverage int, mmr float64) (float64, error) {
	if entry <= 0 || math.IsNaN(entry) || math.IsInf(entry, 0) {
		return 0, fmt.Errorf("%w: entry price must be positive", ErrInvalidArgument)
	}
	if !IsValidMarginLeverage(leverage) {
		return 0, fmt.Errorf("%w: leverage must be between %d and %d", ErrInvalidArgument, MinMarginLeverage, MaxMarginLeverage)
	}
	// unit qty — ratio-only formula matches FromMargin(qty=1, margin=entry/lev)
	margin := entry / float64(leverage)
	return LiquidationPriceFromMargin(side, entry, 1, margin, mmr)
}

// CrossPartialLiquidationMinFraction avoids dust loops: never close less than this
// fraction of open size unless that is already a full close (or min trade size).
const CrossPartialLiquidationMinFraction = 0.01

// CrossPartialLiquidationQty is the smallest quantity to close on pos so that, after a
// proportional margin/debt reduction at mark, account equity >= totalMaint.
//
// Paper equity model: equity = cash + Σ margin + Σ unrealized.
// Closing cq of this position (proportional debt/margin) changes:
//
//	Δequity ≈ −(debt paid in quote terms)
//	Δmaint  = −mmr × entry × cq
//
// so gap = equity − maint improves by cq × (mmr×entry − debtQuote/qty) for long quote debt
// (short uses interest×mark/qty). When the coefficient is ≤ 0, partial size cannot restore
// health — returns full qty. Applies min-size / dust rules so we do not thrash on dust.
//
// Returns 0 if already healthy (equity >= totalMaint) or inputs invalid.
func CrossPartialLiquidationQty(
	side MarginSide,
	qty, entry, mark, debtPrincipal, debtInterest float64,
	debtAsset DebtAsset,
	equity, totalMaint, mmr float64,
) float64 {
	if qty <= PositionEpsilon || entry <= 0 || mark <= 0 || mmr < 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
		return 0
	}
	if equity+1e-9 >= totalMaint {
		return 0
	}
	deficit := totalMaint - equity
	if deficit <= 0 {
		return 0
	}
	// Quote-equivalent debt paid per unit closed (proportional).
	var debtPerUnit float64
	switch {
	case debtAsset == DebtAssetBase || side == MarginShort:
		// Short: principal is coins (not quote cash on close); interest paid in quote.
		debtPerUnit = debtInterest * mark / qty
	default:
		debtPerUnit = (debtPrincipal + debtInterest) / qty
	}
	// Gap improvement per unit closed: maint drop minus equity drop from debt payback.
	coeff := mmr*entry - debtPerUnit
	if coeff <= 1e-12 {
		// Reducing size does not improve equity−maint; full-close this position.
		return qty
	}
	cq := deficit / coeff
	if cq <= 0 {
		return 0
	}
	// Do not overshoot open size.
	if cq > qty {
		cq = qty
	}
	// Minimum meaningful size: max(min trade, fraction of position) to avoid dust loops.
	minLot := MinTradeQuantity
	minFrac := qty * CrossPartialLiquidationMinFraction
	if minFrac > minLot {
		minLot = minFrac
	}
	if cq+1e-15 < minLot {
		cq = minLot
	}
	if cq > qty {
		cq = qty
	}
	// If remainder would be dust / untradeable, take the full position once.
	remain := qty - cq
	if remain > PositionEpsilon && remain < MinTradeQuantity {
		return qty
	}
	if remain > PositionEpsilon && remain < qty*CrossPartialLiquidationMinFraction {
		return qty
	}
	// Near-full: close all (single record, no tiny leftover).
	if cq+MinTradeQuantity >= qty {
		return qty
	}
	return cq
}

// CrossLiquidationPrice is the mark of this position that would drive total equity
// down to total maintenance, holding other positions' unrealized PnL fixed.
//
//	equityExclThisU + U_this(mark) = totalMaint
//	long:  U = (mark-entry)*qty  → mark = entry + (totalMaint - equityExclThisU)/qty
//	short: U = (entry-mark)*qty  → mark = entry - (totalMaint - equityExclThisU)/qty
func CrossLiquidationPrice(side MarginSide, entry, qty, equityExclThisU, totalMaint float64) (float64, error) {
	if entry <= 0 || qty <= 0 {
		return 0, fmt.Errorf("%w: entry and quantity must be positive", ErrInvalidArgument)
	}
	uNeed := totalMaint - equityExclThisU
	switch side {
	case MarginLong:
		p := entry + uNeed/qty
		if p < 0 {
			p = 0
		}
		return p, nil
	case MarginShort:
		p := entry - uNeed/qty
		if p < 0 {
			p = 0
		}
		return p, nil
	default:
		return 0, fmt.Errorf("%w: side must be long or short", ErrInvalidArgument)
	}
}

// MinIsolatedMargin is the floor when removing margin: remaining IM at open leverage.
func MinIsolatedMargin(qty, entry float64, leverage int) (float64, error) {
	return InitialMargin(qty, entry, leverage)
}

// MarginUnrealizedPnL is mark-to-market PnL for an open size.
func MarginUnrealizedPnL(side MarginSide, qty, entry, mark float64) float64 {
	if qty <= PositionEpsilon {
		return 0
	}
	switch side {
	case MarginLong:
		return (mark - entry) * qty
	case MarginShort:
		return (entry - mark) * qty
	default:
		return 0
	}
}

// MarginRealizedPnL is realized PnL when closing qty at exit vs entry.
func MarginRealizedPnL(side MarginSide, qty, entry, exit float64) float64 {
	return MarginUnrealizedPnL(side, qty, entry, exit)
}

// ShouldLiquidate reports whether mark crossed liquidation for the side.
func ShouldLiquidate(side MarginSide, mark, liq float64) bool {
	if mark <= 0 || liq < 0 || math.IsNaN(mark) || math.IsNaN(liq) {
		return false
	}
	switch side {
	case MarginLong:
		return mark <= liq+1e-12
	case MarginShort:
		return mark >= liq-1e-12
	default:
		return false
	}
}

// ShouldTriggerStopLoss reports SL hit.
func ShouldTriggerStopLoss(side MarginSide, mark float64, sl *float64) bool {
	if sl == nil || *sl <= 0 || mark <= 0 {
		return false
	}
	switch side {
	case MarginLong:
		return mark <= *sl+1e-12
	case MarginShort:
		return mark >= *sl-1e-12
	default:
		return false
	}
}

// ShouldTriggerTakeProfit reports TP hit.
func ShouldTriggerTakeProfit(side MarginSide, mark float64, tp *float64) bool {
	if tp == nil || *tp <= 0 || mark <= 0 {
		return false
	}
	switch side {
	case MarginLong:
		return mark >= *tp-1e-12
	case MarginShort:
		return mark <= *tp+1e-12
	default:
		return false
	}
}

// MarginLimitTriggered reports whether a limit open should fill at last price.
// long limit: last <= limit (buy cheaper); short limit: last >= limit (sell higher).
func MarginLimitTriggered(side MarginSide, limitPrice, last float64) bool {
	if limitPrice <= 0 || last <= 0 {
		return false
	}
	switch side {
	case MarginLong:
		return last <= limitPrice+1e-12
	case MarginShort:
		return last >= limitPrice-1e-12
	default:
		return false
	}
}

// ValidateMarginBrackets checks optional SL/TP relative to side and reference price (entry or mark).
func ValidateMarginBrackets(side MarginSide, ref float64, sl, tp *float64) error {
	if ref <= 0 {
		return fmt.Errorf("%w: reference price must be positive", ErrInvalidArgument)
	}
	if sl != nil {
		if *sl <= 0 || math.IsNaN(*sl) || math.IsInf(*sl, 0) {
			return fmt.Errorf("%w: stopLoss must be a positive number", ErrInvalidArgument)
		}
		switch side {
		case MarginLong:
			if *sl >= ref {
				return fmt.Errorf("%w: long stopLoss must be below entry/mark", ErrInvalidArgument)
			}
		case MarginShort:
			if *sl <= ref {
				return fmt.Errorf("%w: short stopLoss must be above entry/mark", ErrInvalidArgument)
			}
		}
	}
	if tp != nil {
		if *tp <= 0 || math.IsNaN(*tp) || math.IsInf(*tp, 0) {
			return fmt.Errorf("%w: takeProfit must be a positive number", ErrInvalidArgument)
		}
		switch side {
		case MarginLong:
			if *tp <= ref {
				return fmt.Errorf("%w: long takeProfit must be above entry/mark", ErrInvalidArgument)
			}
		case MarginShort:
			if *tp >= ref {
				return fmt.Errorf("%w: short takeProfit must be below entry/mark", ErrInvalidArgument)
			}
		}
	}
	return nil
}

// MarginPort extends portfolio persistence for margin positions/orders/trades.
// Implemented on the same SQLite portfolio store.
type MarginPort interface {
	CreateMarginPosition(ctx context.Context, pos MarginPosition) (*MarginPosition, error)
	GetMarginPosition(ctx context.Context, clientID, id string) (*MarginPosition, error)
	ListOpenMarginPositions(ctx context.Context, clientID string) ([]MarginPosition, error)
	ListAllOpenMarginPositions(ctx context.Context) ([]MarginPosition, error)
	CountOpenMarginPositions(ctx context.Context, clientID string) (int, error)
	// UpdateMarginPosition writes fields for an open position (qty, margin, sl/tp, realized, etc.).
	UpdateMarginPosition(ctx context.Context, pos MarginPosition) error
	// CloseMarginPosition marks closed with reason.
	CloseMarginPosition(ctx context.Context, pos MarginPosition) error

	CreateMarginOrder(ctx context.Context, o MarginOrder) (*MarginOrder, error)
	GetMarginOrder(ctx context.Context, clientID, id string) (*MarginOrder, error)
	ListMarginOrders(ctx context.Context, clientID string, status MarginOrderStatus, limit, offset int) ([]MarginOrder, error)
	ListAllOpenMarginOrders(ctx context.Context) ([]MarginOrder, error)
	CountOpenMarginOrders(ctx context.Context, clientID string) (int, error)
	// SumReservedMargin returns reserved margin for open margin limit orders.
	SumReservedMargin(ctx context.Context, clientID string) (float64, error)
	CancelMarginOrder(ctx context.Context, clientID, id string, at time.Time, reason string) (*MarginOrder, error)
	// FillMarginOrder marks filled and links position id (must still be open).
	FillMarginOrder(ctx context.Context, id, positionID string, at time.Time) (*MarginOrder, error)
	RejectMarginOrder(ctx context.Context, id, reason string, at time.Time) error

	InsertMarginTrade(ctx context.Context, t MarginTrade) (*MarginTrade, error)
	GetMarginTrade(ctx context.Context, clientID, id string) (*MarginTrade, error)
	ListMarginTrades(ctx context.Context, clientID string, limit, offset int) ([]MarginTrade, error)

	// ApplyMarginOpen debits cash for margin, inserts position + trade atomically.
	ApplyMarginOpen(ctx context.Context, p *Portfolio, pos MarginPosition, t MarginTrade) error
	// ApplyMarginClose credits cash (margin release + realized), updates/closes position + trade atomically.
	// expected must match current debt + quantity; returns ErrConflict if interest/repay/close raced,
	// or ErrNotFound if the position is already closed (idempotent concurrent/restart liquidation).
	// Forced closes (liquidation/SL/TP) should use SystemCloseTradeID so a restart cannot insert a second trade.
	ApplyMarginClose(ctx context.Context, p *Portfolio, pos MarginPosition, t MarginTrade, fullClose bool, expected PositionCloseSnapshot) error
	// HasMarginTradeAction reports whether a trade with the given action exists for the position.
	HasMarginTradeAction(ctx context.Context, positionID, action string) (bool, error)
	// HasMarginTradeID reports whether a trade with the given id exists (deterministic full-close ids).
	HasMarginTradeID(ctx context.Context, tradeID string) (bool, error)
	// ApplyMarginOpenFromOrder fills a limit order into a new position in one transaction.
	ApplyMarginOpenFromOrder(ctx context.Context, p *Portfolio, orderID string, pos MarginPosition, t MarginTrade, at time.Time) error
	// ApplyMarginAdjust moves cash ↔ position.Margin and updates liquidation (isolated).
	ApplyMarginAdjust(ctx context.Context, p *Portfolio, pos MarginPosition, t MarginTrade) error
	// ApplyMarginRepay pays interest then principal. expected must match current debt or ErrConflict.
	// Returns ErrNotFound if the position was closed concurrently (e.g. liquidation).
	ApplyMarginRepay(ctx context.Context, p *Portfolio, pos MarginPosition, t MarginTrade, expected DebtSnapshot) error
	// UpdatePortfolioMarginMode sets account margin mode.
	UpdatePortfolioMarginMode(ctx context.Context, clientID string, mode MarginMode, at time.Time) error
	// AccrueInterestCAS applies interest only if debt snapshot still matches expected
	// (principal, interest, last_interest_at). Returns false if another worker/repay/close
	// changed debt, principal is zero, or position not open. Never rewinds last_interest_at.
	AccrueInterestCAS(ctx context.Context, id string, expected DebtSnapshot, newInterest float64, newLast time.Time, liq float64, at time.Time) (claimed bool, err error)
	// ListOpenMarginPositionsWithDebt returns open positions with debt_principal > 0.
	ListOpenMarginPositionsWithDebt(ctx context.Context) ([]MarginPosition, error)
}

// DebtSnapshot is the optimistic-concurrency token for debt mutations.
// Accrue, repay, and partial/full close must match this snapshot or return ErrConflict / claimed=false.
type DebtSnapshot struct {
	Principal      float64
	Interest       float64
	LastInterestAt time.Time
}

// PositionCloseSnapshot is the CAS token for close/liquidation: debt + open quantity.
// Two concurrent full closes cannot both succeed; a partial close invalidates a stale full close.
type PositionCloseSnapshot struct {
	DebtSnapshot
	Quantity float64
}

// DebtSnapshotFromPos captures debt fields for CAS.
func DebtSnapshotFromPos(p *MarginPosition) DebtSnapshot {
	if p == nil {
		return DebtSnapshot{}
	}
	return DebtSnapshot{
		Principal: p.DebtPrincipal, Interest: p.DebtInterest, LastInterestAt: p.LastInterestAt,
	}
}

// PositionCloseSnapshotFromPos captures debt + quantity for close CAS.
func PositionCloseSnapshotFromPos(p *MarginPosition) PositionCloseSnapshot {
	if p == nil {
		return PositionCloseSnapshot{}
	}
	return PositionCloseSnapshot{
		DebtSnapshot: DebtSnapshotFromPos(p),
		Quantity:     p.Quantity,
	}
}
