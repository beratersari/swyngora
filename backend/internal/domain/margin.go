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
	// Margin close reasons.
	MarginCloseUser        = "user"
	MarginCloseLiquidation = "liquidation"
	MarginCloseStopLoss    = "stop_loss"
	MarginCloseTakeProfit  = "take_profit"
	MarginClosePartialUser = "partial_close"
	// Margin trade actions (also used for adjust).
	MarginActionAddMargin    = "add_margin"
	MarginActionRemoveMargin = "remove_margin"
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
	LiquidationPrice float64
	StopLoss         *float64
	TakeProfit       *float64
	Status           MarginPositionStatus
	UnrealizedPnL    float64 // view-only / last mark
	MarkPrice        float64 // view-only
	RealizedPnL      float64 // cumulative realized on this position (partials + final)
	CloseReason      string
	OpenedAt         time.Time
	UpdatedAt        time.Time
	ClosedAt         *time.Time
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

// MarginTrade is a margin open/close fill record.
type MarginTrade struct {
	ID           string
	ClientID     string
	PositionID   string
	Exchange     Exchange
	Symbol       string
	Side         MarginSide
	Action       string // open | close | liquidation | stop_loss | take_profit
	Quantity     float64
	Price        float64
	Notional     float64
	RealizedPnL  float64
	MarginDelta  float64 // cash change from margin lock/release (negative open, positive release)
	Leverage     int
	CreatedAt    time.Time
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

// MaintenanceMargin is mmr * notional for open size.
func MaintenanceMargin(qty, entry, mmr float64) float64 {
	if qty <= 0 || entry <= 0 || mmr <= 0 {
		return 0
	}
	return qty * entry * mmr
}

// LiquidationPriceFromMargin computes liq from assigned margin (isolated, or IM share).
// long:  entry - (margin - maint) / qty
// short: entry + (margin - maint) / qty
func LiquidationPriceFromMargin(side MarginSide, entry, qty, margin, mmr float64) (float64, error) {
	if entry <= 0 || qty <= 0 || math.IsNaN(entry) || math.IsNaN(qty) || math.IsInf(entry, 0) || math.IsInf(qty, 0) {
		return 0, fmt.Errorf("%w: entry and quantity must be positive", ErrInvalidArgument)
	}
	if margin < 0 || math.IsNaN(margin) || math.IsInf(margin, 0) {
		return 0, fmt.Errorf("%w: margin must be non-negative", ErrInvalidArgument)
	}
	if mmr < 0 || mmr >= 1 || math.IsNaN(mmr) {
		return 0, fmt.Errorf("%w: invalid maintenance margin rate", ErrInvalidArgument)
	}
	maint := MaintenanceMargin(qty, entry, mmr)
	buffer := margin - maint
	if buffer < 0 {
		buffer = 0
	}
	switch side {
	case MarginLong:
		p := entry - buffer/qty
		if p < 0 {
			p = 0
		}
		return p, nil
	case MarginShort:
		return entry + buffer/qty, nil
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
	ListMarginTrades(ctx context.Context, clientID string, limit, offset int) ([]MarginTrade, error)

	// ApplyMarginOpen debits cash for margin, inserts position + trade atomically.
	ApplyMarginOpen(ctx context.Context, p *Portfolio, pos MarginPosition, t MarginTrade) error
	// ApplyMarginClose credits cash (margin release + realized), updates/closes position + trade atomically.
	ApplyMarginClose(ctx context.Context, p *Portfolio, pos MarginPosition, t MarginTrade, fullClose bool) error
	// ApplyMarginOpenFromOrder fills a limit order into a new position in one transaction.
	ApplyMarginOpenFromOrder(ctx context.Context, p *Portfolio, orderID string, pos MarginPosition, t MarginTrade, at time.Time) error
	// ApplyMarginAdjust moves cash ↔ position.Margin and updates liquidation (isolated).
	ApplyMarginAdjust(ctx context.Context, p *Portfolio, pos MarginPosition, t MarginTrade) error
	// UpdatePortfolioMarginMode sets account margin mode.
	UpdatePortfolioMarginMode(ctx context.Context, clientID string, mode MarginMode, at time.Time) error
}
