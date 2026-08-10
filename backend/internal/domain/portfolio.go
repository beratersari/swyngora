package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// Paper trading limits and defaults (informational / simulation only).
const (
	MaxPortfoliosPerClient     = 20
	MaxPortfolioSharesPerBook  = 50
	DefaultPortfolioName       = "Main"
	MaxPortfolioNameLen        = 64
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

// Portfolio is one named paper book owned by a clientId.
type Portfolio struct {
	ID               string
	ClientID         string // owner
	Name             string
	Currency         string
	StartingBalance  float64
	CashBalance      float64
	RealizedPnLTotal float64
	// NetDeposits is later deposits minus withdrawals (excludes opening startingBalance).
	NetDeposits float64
	// MarginMode is isolated (default) or cross; locked while open margin pos/orders exist.
	MarginMode MarginMode
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// BookID is the id used on child rows (positions, orders, …).
func (p *Portfolio) BookID() string {
	if p == nil {
		return ""
	}
	if strings.TrimSpace(p.ID) != "" {
		return p.ID
	}
	return p.ClientID
}

// ValidatePortfolioName trims and checks length.
func ValidatePortfolioName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = DefaultPortfolioName
	}
	if utf8.RuneCountInString(name) > MaxPortfolioNameLen {
		return "", fmt.Errorf("%w: name must be at most %d characters", ErrInvalidArgument, MaxPortfolioNameLen)
	}
	return name, nil
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
	RealizedPnL    float64 // non-zero on sells (from tax lots when present)
	PendingOrderID string  // set when fill belongs to a pending order
	LotMethod      LotMethod
	LotFills       []TaxLotFill // sell allocations; not required on list
	Fee            float64 // quote-currency taker fee
	LastPrice      float64 // last/mark before slippage (0 = not stored)
	CreatedAt      time.Time
}

// PendingOrderType is a resting paper order kind.
type PendingOrderType string

const (
	PendingLimitBuy      PendingOrderType = "limit_buy"
	PendingLimitSell     PendingOrderType = "limit_sell"
	PendingStopLoss      PendingOrderType = "stop_loss"
	// PendingTrailingStop sells when price falls from a ratcheting peak by a trail distance.
	PendingTrailingStop PendingOrderType = "trailing_stop"
)

// Trailing stop distance mode.
const (
	TrailTypePercent = "percent" // trailValue is fraction e.g. 0.05 = 5% below peak
	TrailTypeOffset  = "offset"  // trailValue is fixed price difference below peak
)

// PendingOrderStatus is the lifecycle of a resting order.
type PendingOrderStatus string

const (
	PendingStatusOpen     PendingOrderStatus = "open"
	PendingStatusFilled   PendingOrderStatus = "filled"
	PendingStatusCanceled PendingOrderStatus = "canceled"
	PendingStatusRejected PendingOrderStatus = "rejected" // condition met but insufficient cash/position
	// PendingStatusPending: bracket exit leg waiting for entry fill (not marketable yet).
	PendingStatusPending PendingOrderStatus = "pending"
)

// Bracket role on a pending order.
const (
	BracketRoleEntry      = "entry"
	BracketRoleTakeProfit = "take_profit"
	BracketRoleStopLoss   = "stop_loss"
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
	// CancelReasonOCOPeerFilled: this leg was canceled because the OCO peer fully filled.
	CancelReasonOCOPeerFilled = "oco_peer_filled"
	// CancelReasonOCOGroup: user canceled one OCO leg (or group); peer is canceled too.
	CancelReasonOCOGroup = "oco_group_canceled"
	// CancelReasonBracketEntry: exit legs canceled because entry was canceled/rejected.
	CancelReasonBracketEntry = "bracket_entry_canceled"
)

// PendingOrder is a limit or stop order waiting for a price condition.
// Buy orders reserve cash (quantity * triggerPrice for remaining size).
// Sell orders reserve position quantity so it cannot be spent by other orders.
// OCO pairs (take-profit limit_sell + stop_loss) share one reserved size via OCOGroupID.
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
	TriggerPrice      float64 // limit price or current stop trigger (trailing: derived from peak)
	ReservedCash      float64 // open buy notional lock (remaining * trigger)
	ReservedQuantity  float64 // open sell size lock
	TimeInForce       TimeInForce
	ExpiresAt         *time.Time // optional; GTC only — cancel when now >= expiresAt
	Status            PendingOrderStatus
	// OCOGroupID links take-profit + stop-loss legs (empty = standalone order).
	OCOGroupID string
	// OCOPeerID is the other leg's id when this order is part of an OCO pair.
	OCOPeerID string
	// TrailType is percent|offset for trailing_stop (empty otherwise).
	TrailType string
	// TrailValue is the trail distance (fraction for percent, price units for offset).
	TrailValue float64
	// TrailPeak is the high-water mark for a sell trailing stop (only ratchets up).
	TrailPeak float64
	// BracketID links entry + take-profit + stop-loss of a bracket order.
	BracketID string
	// BracketRole is entry | take_profit | stop_loss when BracketID is set.
	BracketRole string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FilledAt    *time.Time
	CanceledAt  *time.Time
	FillTradeID string  // latest fill trade id
	FillPrice   float64 // latest fill price
	RejectReason string
	CancelReason string
	// LotMethod is fifo|lifo for sell fills (ignored on buys). Empty = fifo.
	LotMethod LotMethod
}

// IsOCO reports whether this order is one leg of an OCO pair.
func (o PendingOrder) IsOCO() bool {
	return strings.TrimSpace(o.OCOGroupID) != ""
}

// IsBracketExit reports take-profit/stop-loss legs of a bracket (may be pending until entry fills).
func (o PendingOrder) IsBracketExit() bool {
	return strings.TrimSpace(o.BracketID) != "" &&
		(o.BracketRole == BracketRoleTakeProfit || o.BracketRole == BracketRoleStopLoss)
}

// IsBracketEntry reports the entry leg of a bracket order.
func (o PendingOrder) IsBracketEntry() bool {
	return strings.TrimSpace(o.BracketID) != "" && o.BracketRole == BracketRoleEntry
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
	Lots              []TaxLot
}

// PortfolioView is the full paper-trading snapshot for a client.
type PortfolioView struct {
	ID               string
	ClientID         string
	Name             string
	Currency         string
	StartingBalance  float64
	CashBalance      float64
	// NetDeposits is subsequent deposits minus withdrawals (not the opening balance).
	NetDeposits float64
	// ContributedCapital is StartingBalance + NetDeposits (money the user put in).
	ContributedCapital float64
	ReservedCash       float64 // spot pending buy reservations
	ReservedMargin     float64 // margin limit-order reservations
	AvailableCash      float64
	PositionsValue   float64
	// MarginMode is isolated or cross for this account.
	MarginMode MarginMode
	// MarginLocked is margin held in open margin positions (already deducted from cash).
	MarginLocked float64
	// MarginUnrealizedPnL is mark-to-market PnL of open margin positions.
	MarginUnrealizedPnL float64
	// MarginEquity is MarginLocked + MarginUnrealizedPnL.
	MarginEquity     float64
	Equity           float64
	UnrealizedPnL    float64 // spot + margin unrealized
	RealizedPnLTotal float64
	TotalPnL         float64 // equity - contributed capital (deposits are not profit)
	Positions        []PositionView
	MarginPositions  []MarginPosition // open margin positions (with marks when listed via View)
	Note             string
	// Role is owner | trader | viewer for the calling client.
	Role             PortfolioShareRole
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PortfolioShareRole is viewer (read), trader (trade), or owner.
type PortfolioShareRole string

const (
	PortfolioRoleViewer PortfolioShareRole = "viewer"
	PortfolioRoleTrader PortfolioShareRole = "trader"
	PortfolioRoleOwner  PortfolioShareRole = "owner"
)

// PortfolioShare grants another client access to one paper book.
type PortfolioShare struct {
	PortfolioID     string
	OwnerClientID   string
	GranteeClientID string
	Role            PortfolioShareRole
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PortfolioRank returns 0 viewer, 1 trader, 2 owner.
func PortfolioRoleRank(r PortfolioShareRole) int {
	switch r {
	case PortfolioRoleOwner:
		return 2
	case PortfolioRoleTrader:
		return 1
	default:
		return 0
	}
}

// RoleAtLeast reports whether have meets the minimum role.
func RoleAtLeast(have, min PortfolioShareRole) bool {
	return PortfolioRoleRank(have) >= PortfolioRoleRank(min)
}

// IsValidPortfolioShareRole reports viewer|trader.
func IsValidPortfolioShareRole(s string) bool {
	switch PortfolioShareRole(strings.ToLower(strings.TrimSpace(s))) {
	case PortfolioRoleViewer, PortfolioRoleTrader:
		return true
	default:
		return false
	}
}

// NormalizePortfolioShareRole parses viewer|trader.
func NormalizePortfolioShareRole(s string) (PortfolioShareRole, error) {
	r := PortfolioShareRole(strings.ToLower(strings.TrimSpace(s)))
	if !IsValidPortfolioShareRole(string(r)) {
		return "", fmt.Errorf("%w: role must be viewer or trader", ErrInvalidArgument)
	}
	return r, nil
}

// PortfolioPort persists paper portfolios, positions, trades, and pending orders.
type PortfolioPort interface {
	// GetPortfolio returns a book by id (or legacy owner id).
	GetPortfolio(ctx context.Context, id string) (*Portfolio, error)
	// ListPortfolios lists books owned by clientID.
	ListPortfolios(ctx context.Context, clientID string) ([]Portfolio, error)
	CountPortfolios(ctx context.Context, clientID string) (int, error)
	// CreatePortfolio inserts a book. ID must be set (or defaults to ClientID).
	CreatePortfolio(ctx context.Context, p Portfolio) (*Portfolio, error)
	UpdatePortfolioName(ctx context.Context, clientID, id, name string, at time.Time) (*Portfolio, error)
	DeletePortfolio(ctx context.Context, clientID, id string) error

	CreatePortfolioShare(ctx context.Context, share PortfolioShare) (*PortfolioShare, error)
	UpdatePortfolioShareRole(ctx context.Context, portfolioID, granteeClientID string, role PortfolioShareRole, at time.Time) (*PortfolioShare, error)
	GetPortfolioShare(ctx context.Context, portfolioID, granteeClientID string) (*PortfolioShare, error)
	ListPortfolioSharesByBook(ctx context.Context, portfolioID string) ([]PortfolioShare, error)
	ListPortfolioSharesByOwner(ctx context.Context, ownerClientID string) ([]PortfolioShare, error)
	ListPortfolioSharesForGrantee(ctx context.Context, granteeClientID string) ([]PortfolioShare, error)
	DeletePortfolioShare(ctx context.Context, portfolioID, granteeClientID string) error
	CountPortfolioShares(ctx context.Context, portfolioID string) (int, error)

	// UpdateCashAndRealized updates cash and realized total atomically with optional timestamp.
	UpdateCashAndRealized(ctx context.Context, bookID string, cash, realizedTotal float64, at time.Time) error

	GetPosition(ctx context.Context, clientID string, exchange Exchange, symbol string) (*Position, error)
	ListPositions(ctx context.Context, clientID string) ([]Position, error)
	// UpsertPosition writes position; deletes row if quantity is ~0.
	UpsertPosition(ctx context.Context, pos Position) error

	InsertTrade(ctx context.Context, t Trade) (*Trade, error)
	GetTrade(ctx context.Context, clientID, id string) (*Trade, error)
	ListTrades(ctx context.Context, clientID string, limit, offset int) ([]Trade, error)
	CountTrades(ctx context.Context, clientID string) (int, error)
	GetIdempotency(ctx context.Context, clientID, key string) (*IdempotencyRecord, error)

	ListOpenTaxLots(ctx context.Context, clientID string, exchange Exchange, symbol string) ([]TaxLot, error)
	ListTaxLots(ctx context.Context, clientID string, exchange Exchange, symbol string, openOnly bool) ([]TaxLot, error)
	ListTaxLotFillsForTrades(ctx context.Context, tradeIDs []string) ([]TaxLotFill, error)
	InsertTaxLot(ctx context.Context, lot TaxLot) error

	// ExecuteTrade applies portfolio cash/realized, position upsert/delete, and trade insert atomically.
	ExecuteTrade(ctx context.Context, p *Portfolio, pos *Position, t Trade, lots *LotOps) error

	// CreatePendingOrder inserts a new open resting order with reservations.
	CreatePendingOrder(ctx context.Context, o PendingOrder) (*PendingOrder, error)
	// CreateOCOPair inserts take-profit + stop-loss legs in one transaction (shared size reservation).
	CreateOCOPair(ctx context.Context, takeProfit, stopLoss PendingOrder) (tp, sl *PendingOrder, err error)
	// CreateBracket inserts entry (open) + take-profit/stop-loss (pending) in one transaction.
	// Exit legs activate and size as entry fills.
	CreateBracket(ctx context.Context, entry, takeProfit, stopLoss PendingOrder) (ent, tp, sl *PendingOrder, err error)
	// SyncBracketExitsToFilled sets exit legs' size to entryFilled (activate from pending if needed).
	// Does not increase size above entryFilled; preserves exit fills already taken.
	// Returns ErrNotFound if entry missing.
	SyncBracketExitsToFilled(ctx context.Context, clientID, bracketID string, entryFilled float64, at time.Time) error
	// CancelBracket cancels all open/pending legs of a bracket.
	CancelBracket(ctx context.Context, clientID, bracketID string, at time.Time, reason string) error
	// UpdatePendingTrail ratchets trail peak and stop trigger for an open trailing_stop
	// only when the new peak is strictly higher (never moves stop down). Returns false if not open / no move.
	UpdatePendingTrail(ctx context.Context, id string, newPeak, newTrigger float64, at time.Time) (updated bool, err error)
	// GetPendingOrder returns one order for the client or ErrNotFound.
	GetPendingOrder(ctx context.Context, clientID, id string) (*PendingOrder, error)
	// AmendPendingOrder updates remaining size, trigger, original quantity, and reservations
	// of an open order. ExpectedRemaining/ExpectedTrigger are compare-and-set guards.
	// Returns ErrNotFound if missing; ErrConflict if not open or the CAS snapshot is stale.
	AmendPendingOrder(ctx context.Context, clientID, id string, a PendingOrderAmend) (*PendingOrder, error)
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
	// CancelOpenPendingOrders cancels open and pending (inactive bracket-exit) orders for the client.
	// Empty exchange and symbol cancel every market. Non-empty symbol filters that venue+pair.
	// Non-empty exchange with empty symbol cancels every pair on that venue.
	// Returns the rows that were canceled (empty if none were open). Never ErrNotFound for an empty set.
	CancelOpenPendingOrders(ctx context.Context, clientID string, exchange Exchange, symbol string, at time.Time, reason string) ([]PendingOrder, error)
	// ExecutePendingFill applies a (possibly partial) fill for an open order.
	// Updates remaining/reserved, inserts trade, marks filled only when remaining is zero.
	// Returns ErrNotFound if not open.
	ExecutePendingFill(ctx context.Context, order *PendingOrder, p *Portfolio, pos *Position, t Trade, at time.Time, lots *LotOps) error
	// ExecuteOCOFill fills one OCO leg and, in the same transaction, either reduces the peer's
	// remaining size to match or cancels the peer when this leg is fully filled.
	// Guarantees a single cash/position update for the fill (no double apply).
	ExecuteOCOFill(ctx context.Context, filled *PendingOrder, peer *PendingOrder, p *Portfolio, pos *Position, t Trade, at time.Time, lots *LotOps) error
	// CancelOCOGroup cancels both open legs of an OCO (or the remaining open leg) and releases reservation.
	CancelOCOGroup(ctx context.Context, clientID, groupID string, at time.Time, reason string) error
	// RejectPendingOrder marks an open order rejected and releases remaining reservation.
	// Returns ErrNotFound if not open.
	RejectPendingOrder(ctx context.Context, orderID, reason string, at time.Time) error

	// Recurring buy plans (paper DCA)
	CreateRecurringBuyPlan(ctx context.Context, p RecurringBuyPlan) (*RecurringBuyPlan, error)
	GetRecurringBuyPlan(ctx context.Context, clientID, id string) (*RecurringBuyPlan, error)
	ListRecurringBuyPlans(ctx context.Context, clientID string) ([]RecurringBuyPlan, error)
	CountRecurringBuyPlans(ctx context.Context, clientID string) (int, error)
	// UpdateRecurringBuyPlanStatus sets active/paused.
	UpdateRecurringBuyPlanStatus(ctx context.Context, clientID, id string, status RecurringBuyPlanStatus, nextRunAt time.Time, at time.Time) (*RecurringBuyPlan, error)
	// UpdateRecurringBuyPlan writes name, amount, and schedule fields (not status).
	UpdateRecurringBuyPlan(ctx context.Context, clientID, id string, p RecurringBuyPlan) (*RecurringBuyPlan, error)
	DeleteRecurringBuyPlan(ctx context.Context, clientID, id string) error
	// ListDueRecurringBuyPlans returns active plans with next_run_at <= now.
	ListDueRecurringBuyPlans(ctx context.Context, now time.Time, limit int) ([]RecurringBuyPlan, error)
	// ClaimRecurringBuyRun inserts a run row; returns false if period already claimed (unique).
	ClaimRecurringBuyRun(ctx context.Context, run RecurringBuyRun) (claimed bool, out *RecurringBuyRun, err error)
	// FinishRecurringBuyRun updates run outcome and advances the plan schedule.
	FinishRecurringBuyRun(ctx context.Context, planID string, run RecurringBuyRun, nextRunAt time.Time, lastPeriodKey string, at time.Time) error
	// SkipRecurringBuyPeriod advances schedule without a new buy when period already claimed.
	AdvanceRecurringBuyPlan(ctx context.Context, planID string, nextRunAt time.Time, lastPeriodKey string, at time.Time) error
	ListRecurringBuyRuns(ctx context.Context, clientID, planID string, limit, offset int) ([]RecurringBuyRun, error)

	// Allocation baskets (spot target mix; rebalance is user-triggered only).
	CreateAllocationBasket(ctx context.Context, b AllocationBasket) (*AllocationBasket, error)
	GetAllocationBasket(ctx context.Context, clientID, id string) (*AllocationBasket, error)
	ListAllocationBaskets(ctx context.Context, clientID string) ([]AllocationBasket, error)
	CountAllocationBaskets(ctx context.Context, clientID string) (int, error)
	UpdateAllocationBasket(ctx context.Context, clientID, id string, b AllocationBasket) (*AllocationBasket, error)
	DeleteAllocationBasket(ctx context.Context, clientID, id string) error

	// Risk limits (optional user brakes; no auto-close).
	GetRiskLimits(ctx context.Context, clientID string) (*RiskLimits, error)
	UpsertRiskLimits(ctx context.Context, lim RiskLimits) (*RiskLimits, error)
	DeleteRiskLimits(ctx context.Context, clientID string) error

	// Cash movements (user deposits / withdrawals / internal transfers; not trades).
	ApplyCashMovement(ctx context.Context, p *Portfolio, m CashMovement) (*CashMovement, error)
	// ApplyInternalTransfer moves cash between two books the same owner controls, atomically.
	ApplyInternalTransfer(ctx context.Context, from, to *Portfolio, out, in CashMovement) (fromMov, toMov *CashMovement, err error)
	ListCashMovements(ctx context.Context, clientID string, limit, offset int) ([]CashMovement, error)
	CountCashMovements(ctx context.Context, clientID string) (int, error)

	// Equity history for performance charts (one row per client + time bucket).
	UpsertEquitySnapshot(ctx context.Context, snap EquitySnapshot) error
	ListEquitySnapshots(ctx context.Context, clientID string, from, to time.Time) ([]EquitySnapshot, error)
	LatestEquitySnapshotBefore(ctx context.Context, clientID string, before time.Time) (*EquitySnapshot, error)
	DeleteEquitySnapshotsBefore(ctx context.Context, before time.Time) (int64, error)
	ListPortfolioIDs(ctx context.Context) ([]string, error)

	// Margin (isolated leverage) — see MarginPort methods embedded below.
	MarginPort

	// ExportOwnedPortfolios dumps every book owned by ownerClientID (full backup snapshot).
	ExportOwnedPortfolios(ctx context.Context, ownerClientID string) ([]PortfolioSnapshot, error)
	// ImportOwnedPortfolios restores snapshots. replace deletes all owned books first.
	// Returns how many books were inserted. Caller remaps ownership/ids before calling.
	ImportOwnedPortfolios(ctx context.Context, ownerClientID string, snaps []PortfolioSnapshot, replace bool) (int, error)
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

// IsValidPendingOrderType reports limit_buy | limit_sell | stop_loss | trailing_stop.
func IsValidPendingOrderType(s string) bool {
	switch PendingOrderType(s) {
	case PendingLimitBuy, PendingLimitSell, PendingStopLoss, PendingTrailingStop:
		return true
	default:
		return false
	}
}

// IsValidTrailType reports percent | offset.
func IsValidTrailType(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case TrailTypePercent, TrailTypeOffset:
		return true
	default:
		return false
	}
}

// NormalizeTrailType returns percent|offset.
func NormalizeTrailType(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !IsValidTrailType(s) {
		return "", fmt.Errorf("%w: trailType must be percent or offset", ErrInvalidArgument)
	}
	return s, nil
}

// ValidateTrailValue checks trail distance for the given mode.
// percent: (0, 1) exclusive; offset: positive price units.
func ValidateTrailValue(trailType string, trailValue float64) error {
	if trailValue <= 0 || math.IsNaN(trailValue) || math.IsInf(trailValue, 0) {
		return fmt.Errorf("%w: trailValue must be positive", ErrInvalidArgument)
	}
	switch trailType {
	case TrailTypePercent:
		if trailValue >= 1 {
			return fmt.Errorf("%w: trailValue percent must be less than 1 (e.g. 0.05 for 5%%)", ErrInvalidArgument)
		}
	case TrailTypeOffset:
		// any positive offset ok
	default:
		return fmt.Errorf("%w: trailType must be percent or offset", ErrInvalidArgument)
	}
	return nil
}

// TrailStopPrice is the sell stop level below peak for a trailing stop.
// percent: peak * (1 - trailValue); offset: peak - trailValue (floored at 0).
func TrailStopPrice(peak, trailValue float64, trailType string) float64 {
	if peak <= 0 || trailValue <= 0 {
		return 0
	}
	switch trailType {
	case TrailTypePercent:
		p := peak * (1 - trailValue)
		if p < 0 {
			return 0
		}
		return p
	case TrailTypeOffset:
		p := peak - trailValue
		if p < 0 {
			return 0
		}
		return p
	default:
		return 0
	}
}

// RatchetTrailPeak returns the new high-water mark for a sell trailing stop (never decreases).
func RatchetTrailPeak(peak, last float64) float64 {
	if last > peak {
		return last
	}
	return peak
}

// AdvanceTrailingStop applies last price to peak and stop for a sell trailing stop.
// Peak only moves up; stop only moves up (or stays). Returns (newPeak, newStop, peakMoved).
func AdvanceTrailingStop(peak, last, trailValue float64, trailType string) (newPeak, newStop float64, peakMoved bool) {
	if peak <= 0 && last > 0 {
		peak = last
		peakMoved = true
	}
	newPeak = RatchetTrailPeak(peak, last)
	if newPeak > peak+1e-15 {
		peakMoved = true
	}
	newStop = TrailStopPrice(newPeak, trailValue, trailType)
	return newPeak, newStop, peakMoved
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
	case PendingLimitSell, PendingStopLoss, PendingTrailingStop:
		return TradeSideSell
	default:
		return ""
	}
}

// ValidateOCOPrices checks take-profit limit_sell and stop-loss prices for a long exit OCO.
// takeProfit must be above stopLoss; both positive and distinct.
func ValidateOCOPrices(takeProfit, stopLoss float64) error {
	if takeProfit <= 0 || stopLoss <= 0 || math.IsNaN(takeProfit) || math.IsNaN(stopLoss) ||
		math.IsInf(takeProfit, 0) || math.IsInf(stopLoss, 0) {
		return fmt.Errorf("%w: takeProfitPrice and stopLossPrice must be positive", ErrInvalidArgument)
	}
	if takeProfit <= stopLoss {
		return fmt.Errorf("%w: takeProfitPrice must be greater than stopLossPrice", ErrInvalidArgument)
	}
	return nil
}

// ValidateBracketPrices checks long entry limit with take-profit above and stop-loss below entry.
func ValidateBracketPrices(entry, takeProfit, stopLoss float64) error {
	if entry <= 0 || math.IsNaN(entry) || math.IsInf(entry, 0) {
		return fmt.Errorf("%w: entry triggerPrice must be positive", ErrInvalidArgument)
	}
	if err := ValidateOCOPrices(takeProfit, stopLoss); err != nil {
		return err
	}
	if takeProfit <= entry {
		return fmt.Errorf("%w: takeProfitPrice must be above entry price", ErrInvalidArgument)
	}
	if stopLoss >= entry {
		return fmt.Errorf("%w: stopLossPrice must be below entry price", ErrInvalidArgument)
	}
	return nil
}

// OCOWinnerForTick picks which OCO leg may fill on a single price update.
// If only one is triggered, that leg wins. If both are triggered (gap through both levels),
// stop_loss wins so only one fill applies (balance/position change once).
func OCOWinnerForTick(takeProfit, stopLoss *PendingOrder, last float64) *PendingOrder {
	if takeProfit == nil && stopLoss == nil {
		return nil
	}
	tpTrig := takeProfit != nil && takeProfit.Status == PendingStatusOpen &&
		PendingOrderTriggered(takeProfit.Type, takeProfit.TriggerPrice, last)
	slTrig := stopLoss != nil && stopLoss.Status == PendingStatusOpen &&
		PendingOrderTriggered(stopLoss.Type, stopLoss.TriggerPrice, last)
	switch {
	case slTrig && tpTrig:
		return stopLoss
	case slTrig:
		return stopLoss
	case tpTrig:
		return takeProfit
	default:
		return nil
	}
}

// PendingOrderTriggered reports whether last price meets the resting order condition.
//
//	limit_buy:  last <= trigger (buy at or below limit)
//	limit_sell: last >= trigger (sell at or above limit)
//	stop_loss / trailing_stop: last <= trigger (sell when price falls to stop; gaps still trigger)
func PendingOrderTriggered(orderType PendingOrderType, trigger, last float64) bool {
	if trigger <= 0 || last <= 0 || math.IsNaN(trigger) || math.IsNaN(last) {
		return false
	}
	switch orderType {
	case PendingLimitBuy, PendingStopLoss, PendingTrailingStop:
		return last <= trigger+1e-12
	case PendingLimitSell:
		return last >= trigger-1e-12
	default:
		return false
	}
}

// BuyReserveCash locks worst-case cash for a limit buy: remaining * slipped trigger * (1+fee).
func BuyReserveCash(quantity, triggerPrice float64, cost TradingCost) float64 {
	if quantity <= 0 || triggerPrice <= 0 {
		return 0
	}
	px := ApplySlippage(triggerPrice, TradeSideBuy, cost.SlippageRate)
	return quantity * px * (1 + cost.FeeRate)
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

// MaxBuyFillQty returns how much base qty can be filled from remaining reserved cash
// at fillPrice including the taker fee (unit cost = fillPrice * (1+feeRate)).
func MaxBuyFillQty(remainingQty, reservedCash, fillPrice, feeRate float64) float64 {
	if remainingQty <= PositionEpsilon || reservedCash <= 0 || fillPrice <= 0 {
		return 0
	}
	if feeRate < 0 {
		feeRate = 0
	}
	unit := fillPrice * (1 + feeRate)
	if unit <= 0 {
		return 0
	}
	byCash := reservedCash / unit
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

// AfterBuyFillReservation is remaining reserved cash after a partial buy fill.
func AfterBuyFillReservation(remainingAfter, triggerPrice float64, cost TradingCost) float64 {
	return BuyReserveCash(remainingAfter, triggerPrice, cost)
}

// AfterSellFillReservation updates reserved quantity after a sell fill.
func AfterSellFillReservation(remainingAfter float64) float64 {
	if remainingAfter <= PositionEpsilon {
		return 0
	}
	return remainingAfter
}

// PendingOrderAmend is the store write for an in-place open-order edit (same id).
type PendingOrderAmend struct {
	RemainingQuantity float64
	TriggerPrice      float64
	Quantity          float64 // filled + remaining
	ReservedCash      float64
	ReservedQuantity  float64
	ExpectedRemaining float64 // CAS: remaining at read time
	ExpectedTrigger   float64 // CAS: trigger at read time
	At                time.Time
}

// IsAmendablePendingType reports standalone types that support in-place price/size edit.
func IsAmendablePendingType(t PendingOrderType) bool {
	switch t {
	case PendingLimitBuy, PendingLimitSell, PendingStopLoss:
		return true
	default:
		return false
	}
}

// CanAmendPendingOrder reports whether o may be edited in place (price / remaining).
// Only open GTC standalone limit_buy, limit_sell, and stop_loss.
func CanAmendPendingOrder(o PendingOrder) error {
	if o.Status != PendingStatusOpen {
		return fmt.Errorf("%w: only open orders can be amended", ErrConflict)
	}
	if !IsAmendablePendingType(o.Type) {
		return fmt.Errorf("%w: only limit_buy, limit_sell, and stop_loss can be amended", ErrInvalidArgument)
	}
	tif := o.TimeInForce
	if tif == "" {
		tif = TimeInForceGTC
	}
	if tif != TimeInForceGTC {
		return fmt.Errorf("%w: only gtc orders can be amended", ErrInvalidArgument)
	}
	if o.IsOCO() {
		return fmt.Errorf("%w: oco legs cannot be amended", ErrInvalidArgument)
	}
	if strings.TrimSpace(o.BracketID) != "" {
		return fmt.Errorf("%w: bracket legs cannot be amended", ErrInvalidArgument)
	}
	return nil
}

// ValidateAmendTriggerPrice checks a new limit/stop price.
func ValidateAmendTriggerPrice(p float64) error {
	if p < MinTriggerPrice || p > MaxTriggerPrice || math.IsNaN(p) || math.IsInf(p, 0) {
		return fmt.Errorf("%w: triggerPrice out of range", ErrInvalidArgument)
	}
	return nil
}

// ValidateAmendRemaining checks a new remaining size (must stay open — not zero).
func ValidateAmendRemaining(remaining float64) error {
	if remaining < MinTradeQuantity || remaining > MaxTradeQuantity || math.IsNaN(remaining) || math.IsInf(remaining, 0) {
		return fmt.Errorf("%w: remainingQuantity out of range", ErrInvalidArgument)
	}
	return nil
}

// AmendOriginalQuantity is filled + remaining after an amend.
func AmendOriginalQuantity(filled, remaining float64) float64 {
	if filled < 0 {
		filled = 0
	}
	if remaining <= PositionEpsilon {
		return filled
	}
	return filled + remaining
}

// MaxAmendRemaining is the largest remaining size the account can back at triggerPrice.
// availableCashForOrder / availableQtyForOrder must already include this order's current reservation.
func MaxAmendRemaining(side TradeSide, triggerPrice, availableCashForOrder, availableQtyForOrder float64) float64 {
	switch side {
	case TradeSideBuy:
		if triggerPrice <= 0 || availableCashForOrder <= 0 {
			return 0
		}
		m := availableCashForOrder / triggerPrice
		if m > MaxTradeQuantity {
			return MaxTradeQuantity
		}
		return m
	case TradeSideSell:
		if availableQtyForOrder <= 0 {
			return 0
		}
		if availableQtyForOrder > MaxTradeQuantity {
			return MaxTradeQuantity
		}
		return availableQtyForOrder
	default:
		return 0
	}
}