package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// Paper trading limits and defaults (informational / simulation only).
const (
	MaxPortfoliosPerClient = 1
	DefaultPaperCurrency   = "USDT"
	MinStartingBalance     = 1.0
	MaxStartingBalance     = 10_000_000.0
	MinTradeQuantity       = 1e-8
	MaxTradeQuantity       = 1e9
	// Position qty below this is treated as flat (closed).
	PositionEpsilon = 1e-12
	// Max open pending orders per client.
	MaxOpenPendingOrders = 50
	MinTriggerPrice      = 1e-12
	MaxTriggerPrice      = 1e15
)

// TradeSide is buy or sell.
type TradeSide string

const (
	TradeSideBuy  TradeSide = "buy"
	TradeSideSell TradeSide = "sell"
)

// Portfolio is a paper-trading account keyed by client id.
type Portfolio struct {
	ClientID         string
	Currency         string
	StartingBalance  float64
	CashBalance      float64
	RealizedPnLTotal float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Position is an open (or zero) holding of a symbol on an exchange.
type Position struct {
	ClientID  string
	Exchange  Exchange
	Symbol    string
	Quantity  float64
	AvgCost   float64 // average entry price in portfolio currency
	UpdatedAt time.Time
}

// Trade is a filled paper order leg (market or one partial/complete pending fill).
type Trade struct {
	ID             string
	ClientID       string
	Exchange       Exchange
	Symbol         string
	Side           TradeSide
	Quantity       float64
	Price          float64
	Notional       float64
	RealizedPnL    float64 // non-zero on sells
	PendingOrderID string  // set when fill belongs to a pending order
	CreatedAt      time.Time
}

// PendingOrderType is a resting paper order kind.
type PendingOrderType string

const (
	PendingLimitBuy  PendingOrderType = "limit_buy"
	PendingLimitSell PendingOrderType = "limit_sell"
	PendingStopLoss  PendingOrderType = "stop_loss"
)

// PendingOrderStatus is the lifecycle of a resting order.
type PendingOrderStatus string

const (
	PendingStatusOpen     PendingOrderStatus = "open"
	PendingStatusFilled   PendingOrderStatus = "filled"
	PendingStatusCanceled PendingOrderStatus = "canceled"
	PendingStatusRejected PendingOrderStatus = "rejected" // condition met but insufficient cash/position
)

// TimeInForce is the fill policy for a pending paper order.
type TimeInForce string

const (
	// TimeInForceGTC (good-til-canceled) rests until filled, canceled, or expiresAt.
	TimeInForceGTC TimeInForce = "gtc"
	// TimeInForceIOC (immediate-or-cancel) fills available qty on first try, cancels remainder.
	TimeInForceIOC TimeInForce = "ioc"
	// TimeInForceFOK (fill-or-kill) fills fully on first try or cancels with no fill.
	TimeInForceFOK TimeInForce = "fok"
)

// Cancel reason codes (stored on canceled orders).
const (
	CancelReasonUser         = "user"
	CancelReasonExpired      = "expired"
	CancelReasonIOCRemainder = "ioc_remainder"
	CancelReasonIOCNoFill    = "ioc_no_fill"
	CancelReasonFOKUnfilled  = "fok_unfilled"
)

// PendingOrder is a limit or stop order waiting for a price condition.
// Buy orders reserve cash (quantity * triggerPrice for remaining size).
// Sell orders reserve position quantity so it cannot be spent by other orders.
type PendingOrder struct {
	ID                string
	ClientID          string
	Exchange          Exchange
	Symbol            string
	Type              PendingOrderType
	Side              TradeSide // buy for limit_buy; sell for limit_sell and stop_loss
	Quantity          float64   // original size
	FilledQuantity    float64
	RemainingQuantity float64
	TriggerPrice      float64 // limit price or stop trigger
	ReservedCash      float64 // open buy notional lock (remaining * trigger)
	ReservedQuantity  float64 // open sell size lock
	TimeInForce       TimeInForce
	ExpiresAt         *time.Time // optional; GTC only — cancel when now >= expiresAt
	Status            PendingOrderStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
	FilledAt          *time.Time
	CanceledAt        *time.Time
	FillTradeID       string  // latest fill trade id
	FillPrice         float64 // latest fill price
	RejectReason      string
	CancelReason      string
}

// PositionView is a position with mark-to-market fields.
type PositionView struct {
	Exchange          Exchange
	Symbol            string
	Quantity          float64
	ReservedQuantity  float64
	AvailableQuantity float64
	AvgCost           float64
	MarkPrice         float64
	MarketValue       float64
	UnrealizedPnL     float64
	CostBasis         float64
}

// PortfolioView is the full paper-trading snapshot for a client.
type PortfolioView struct {
	ClientID         string
	Currency         string
	StartingBalance  float64
	CashBalance      float64
	ReservedCash     float64
	AvailableCash    float64
	PositionsValue   float64
	Equity           float64
	UnrealizedPnL    float64
	RealizedPnLTotal float64
	TotalPnL         float64 // equity - starting = realized + unrealized (approx)
	Positions        []PositionView
	Note             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PortfolioPort persists paper portfolios, positions, trades, and pending orders.
type PortfolioPort interface {
	// GetPortfolio returns the portfolio or ErrNotFound.
	GetPortfolio(ctx context.Context, clientID string) (*Portfolio, error)
	// CreatePortfolio creates a new portfolio; fails if one already exists for client.
	CreatePortfolio(ctx context.Context, p Portfolio) (*Portfolio, error)
	// UpdateCashAndRealized updates cash and realized total atomically with optional timestamp.
	UpdateCashAndRealized(ctx context.Context, clientID string, cash, realizedTotal float64, at time.Time) error

	GetPosition(ctx context.Context, clientID string, exchange Exchange, symbol string) (*Position, error)
	ListPositions(ctx context.Context, clientID string) ([]Position, error)
	// UpsertPosition writes position; deletes row if quantity is ~0.
	UpsertPosition(ctx context.Context, pos Position) error

	InsertTrade(ctx context.Context, t Trade) (*Trade, error)
	ListTrades(ctx context.Context, clientID string, limit, offset int) ([]Trade, error)
	CountTrades(ctx context.Context, clientID string) (int, error)

	// ExecuteTrade applies portfolio cash/realized, position upsert/delete, and trade insert atomically.
	ExecuteTrade(ctx context.Context, p *Portfolio, pos *Position, t Trade) error

	// CreatePendingOrder inserts a new open resting order with reservations.
	CreatePendingOrder(ctx context.Context, o PendingOrder) (*PendingOrder, error)
	// GetPendingOrder returns one order for the client or ErrNotFound.
	GetPendingOrder(ctx context.Context, clientID, id string) (*PendingOrder, error)
	// ListPendingOrders lists orders for a client, optionally filtered by status (empty = all).
	ListPendingOrders(ctx context.Context, clientID string, status PendingOrderStatus, limit, offset int) ([]PendingOrder, error)
	// CountOpenPendingOrders returns the number of open orders for a client.
	CountOpenPendingOrders(ctx context.Context, clientID string) (int, error)
	// ListAllOpenPendingOrders returns every open order (background filler).
	ListAllOpenPendingOrders(ctx context.Context) ([]PendingOrder, error)
	// SumReservedCash returns total reserved cash for open buy orders of a client.
	SumReservedCash(ctx context.Context, clientID string) (float64, error)
	// SumReservedQuantity returns reserved sell quantity for a client/exchange/symbol.
	SumReservedQuantity(ctx context.Context, clientID string, exchange Exchange, symbol string) (float64, error)
	// CancelPendingOrder sets status canceled only if still open and releases remaining reservation.
	// reason is stored as cancel_reason (e.g. user, expired, ioc_remainder).
	CancelPendingOrder(ctx context.Context, clientID, id string, at time.Time, reason string) (*PendingOrder, error)
	// ExecutePendingFill applies a (possibly partial) fill for an open order.
	// Updates remaining/reserved, inserts trade, marks filled only when remaining is zero.
	// Returns ErrNotFound if not open.
	ExecutePendingFill(ctx context.Context, order *PendingOrder, p *Portfolio, pos *Position, t Trade, at time.Time) error
	// RejectPendingOrder marks an open order rejected and releases remaining reservation.
	// Returns ErrNotFound if not open.
	RejectPendingOrder(ctx context.Context, orderID, reason string, at time.Time) error
}

// ApplyBuy updates cash and position for a market buy. Pure helper.
func ApplyBuy(cash, qty, price, posQty, avgCost float64) (newCash, newQty, newAvg float64, err error) {
	if qty <= 0 || price <= 0 || math.IsNaN(qty) || math.IsNaN(price) || math.IsInf(qty, 0) || math.IsInf(price, 0) {
		return 0, 0, 0, fmt.Errorf("%w: quantity and price must be positive", ErrInvalidArgument)
	}
	cost := qty * price
	if cash+1e-9 < cost {
		return 0, 0, 0, fmt.Errorf("%w: insufficient cash balance", ErrInvalidArgument)
	}
	newCash = cash - cost
	if posQty <= PositionEpsilon {
		return newCash, qty, price, nil
	}
	newQty = posQty + qty
	newAvg = (avgCost*posQty + price*qty) / newQty
	return newCash, newQty, newAvg, nil
}

// ApplySell updates cash, position, and realized P&L for a market sell. Pure helper.
func ApplySell(cash, qty, price, posQty, avgCost float64) (newCash, newQty, realized float64, err error) {
	if qty <= 0 || price <= 0 || math.IsNaN(qty) || math.IsNaN(price) || math.IsInf(qty, 0) || math.IsInf(price, 0) {
		return 0, 0, 0, fmt.Errorf("%w: quantity and price must be positive", ErrInvalidArgument)
	}
	if posQty+PositionEpsilon < qty {
		return 0, 0, 0, fmt.Errorf("%w: insufficient position quantity", ErrInvalidArgument)
	}
	proceeds := qty * price
	realized = (price - avgCost) * qty
	newCash = cash + proceeds
	newQty = posQty - qty
	if newQty < PositionEpsilon {
		newQty = 0
	}
	return newCash, newQty, realized, nil
}

// UnrealizedPnL is (mark - avgCost) * quantity.
func UnrealizedPnL(qty, avgCost, mark float64) float64 {
	if qty <= PositionEpsilon {
		return 0
	}
	return (mark - avgCost) * qty
}

// IsValidTradeSide reports buy/sell.
func IsValidTradeSide(s string) bool {
	switch TradeSide(s) {
	case TradeSideBuy, TradeSideSell:
		return true
	default:
		return false
	}
}

// IsValidPendingOrderType reports limit_buy | limit_sell | stop_loss.
func IsValidPendingOrderType(s string) bool {
	switch PendingOrderType(s) {
	case PendingLimitBuy, PendingLimitSell, PendingStopLoss:
		return true
	default:
		return false
	}
}

// IsValidTimeInForce reports gtc | ioc | fok.
func IsValidTimeInForce(s string) bool {
	switch TimeInForce(s) {
	case TimeInForceGTC, TimeInForceIOC, TimeInForceFOK:
		return true
	default:
		return false
	}
}

// NormalizeTimeInForce returns a valid TIF; empty defaults to gtc.
func NormalizeTimeInForce(s string) (TimeInForce, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return TimeInForceGTC, nil
	}
	if !IsValidTimeInForce(s) {
		return "", fmt.Errorf("%w: timeInForce must be gtc, ioc, or fok", ErrInvalidArgument)
	}
	return TimeInForce(s), nil
}

// PendingOrderExpired reports whether a GTC order with expiresAt is past expiry.
func PendingOrderExpired(o PendingOrder, now time.Time) bool {
	if o.ExpiresAt == nil || o.ExpiresAt.IsZero() {
		return false
	}
	return !now.Before(o.ExpiresAt.UTC())
}

// SideForPendingType returns the trade side for a pending order type.
func SideForPendingType(t PendingOrderType) TradeSide {
	switch t {
	case PendingLimitBuy:
		return TradeSideBuy
	case PendingLimitSell, PendingStopLoss:
		return TradeSideSell
	default:
		return ""
	}
}

// PendingOrderTriggered reports whether last price meets the resting order condition.
//
//	limit_buy:  last <= trigger (buy at or below limit)
//	limit_sell: last >= trigger (sell at or above limit)
//	stop_loss:  last <= trigger (sell when price falls to stop)
func PendingOrderTriggered(orderType PendingOrderType, trigger, last float64) bool {
	if trigger <= 0 || last <= 0 || math.IsNaN(trigger) || math.IsNaN(last) {
		return false
	}
	switch orderType {
	case PendingLimitBuy, PendingStopLoss:
		return last <= trigger+1e-12
	case PendingLimitSell:
		return last >= trigger-1e-12
	default:
		return false
	}
}

// BuyReserveCash is quantity * triggerPrice (max cash locked for a limit buy).
func BuyReserveCash(quantity, triggerPrice float64) float64 {
	if quantity <= 0 || triggerPrice <= 0 {
		return 0
	}
	return quantity * triggerPrice
}

// AvailableCash is total cash minus reserved cash for open buy orders.
func AvailableCash(cashBalance, reservedCash float64) float64 {
	a := cashBalance - reservedCash
	if a < 0 {
		return 0
	}
	return a
}

// AvailablePosition is held quantity minus reserved sell quantity.
func AvailablePosition(held, reservedQty float64) float64 {
	a := held - reservedQty
	if a < PositionEpsilon {
		return 0
	}
	return a
}

// MaxBuyFillQty returns how much base qty can be filled from remaining reserved cash at fillPrice.
func MaxBuyFillQty(remainingQty, reservedCash, fillPrice float64) float64 {
	if remainingQty <= PositionEpsilon || reservedCash <= 0 || fillPrice <= 0 {
		return 0
	}
	byCash := reservedCash / fillPrice
	if byCash < remainingQty {
		return byCash
	}
	return remainingQty
}

// ClampFillQty bounds a requested fill to remaining size (and optional maxFill).
// maxFill <= 0 means no extra cap.
func ClampFillQty(remaining, requested, maxFill float64) float64 {
	if remaining <= PositionEpsilon {
		return 0
	}
	q := remaining
	if requested > 0 && requested < q {
		q = requested
	}
	if maxFill > 0 && maxFill < q {
		q = maxFill
	}
	if q < MinTradeQuantity {
		return 0
	}
	return q
}

// AfterBuyFillReservation updates reserved cash after a buy fill of fillQty at fillPrice.
// Reservation for remaining size is remainingAfter * triggerPrice.
func AfterBuyFillReservation(remainingAfter, triggerPrice float64) float64 {
	if remainingAfter <= PositionEpsilon {
		return 0
	}
	return remainingAfter * triggerPrice
}

// AfterSellFillReservation updates reserved quantity after a sell fill.
func AfterSellFillReservation(remainingAfter float64) float64 {
	if remainingAfter <= PositionEpsilon {
		return 0
	}
	return remainingAfter
}