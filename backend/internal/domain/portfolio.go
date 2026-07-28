package domain

import (
	"context"
	"fmt"
	"math"
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

// Trade is a filled paper market order.
type Trade struct {
	ID          string
	ClientID    string
	Exchange    Exchange
	Symbol      string
	Side        TradeSide
	Quantity    float64
	Price       float64
	Notional    float64
	RealizedPnL float64 // non-zero on sells
	CreatedAt   time.Time
}

// PositionView is a position with mark-to-market fields.
type PositionView struct {
	Exchange       Exchange
	Symbol         string
	Quantity       float64
	AvgCost        float64
	MarkPrice      float64
	MarketValue    float64
	UnrealizedPnL  float64
	CostBasis      float64
}

// PortfolioView is the full paper-trading snapshot for a client.
type PortfolioView struct {
	ClientID         string
	Currency         string
	StartingBalance  float64
	CashBalance      float64
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

// PortfolioPort persists paper portfolios, positions, and trades.
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