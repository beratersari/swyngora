package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	MaxFundingArbWatchesPerClient = 20
	MaxFundingArbMinProfit        = 1_000_000.0
	// FundingArbWatchScanSymbol means follow every coin in the funding-arb scan.
	FundingArbWatchScanSymbol = "*"
)

// FundingArbWatchStatus is active (evaluated) or paused.
type FundingArbWatchStatus string

const (
	FundingArbWatchActive FundingArbWatchStatus = "active"
	FundingArbWatchPaused FundingArbWatchStatus = "paused"
)

// FundingArbSignalStatus is open while net stays at/above the watch minimum.
type FundingArbSignalStatus string

const (
	FundingArbSignalOpen   FundingArbSignalStatus = "open"
	FundingArbSignalClosed FundingArbSignalStatus = "closed"
)

// FundingArbWatch is a tenant follow. Symbol FundingArbWatchScanSymbol
// follows every coin in the funding-arb scan; any other symbol is one pair.
type FundingArbWatch struct {
	ID            string
	ClientID      string
	Symbol        string
	Quote         string
	SymbolLimit   int
	Notional      float64
	HoldHours     float64
	MinProfit     float64 // quote currency after round-trip fees
	FeeBinancePct float64
	FeeBybitPct   float64
	Status        FundingArbWatchStatus
	Armed         bool // re-arm after net falls below MinProfit (single-coin)
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsScan reports whether this watch follows the funding-arb scan.
func (w FundingArbWatch) IsScan() bool {
	return w.Symbol == FundingArbWatchScanSymbol || w.Symbol == ""
}

// ResolveFundingArbWatchSymbol treats empty / * / scan / all as a scan follow.
func ResolveFundingArbWatchSymbol(raw string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	switch s {
	case "", "*", "SCAN", "ALL":
		return FundingArbWatchScanSymbol, nil
	default:
		return ValidateOpenInterestSymbol(s)
	}
}

// FundingArbSignal is one crossing above MinProfit for a watch.
type FundingArbSignal struct {
	ID            string
	WatchID       string
	ClientID      string
	Symbol        string
	LongExchange  Exchange
	ShortExchange Exchange
	NetAfterFees  float64
	MinProfit     float64
	Status        FundingArbSignalStatus
	OpenedAt      time.Time
	LastSeenAt    time.Time
	ClosedAt      *time.Time
}

// ResolveFundingArbMinProfit requires a positive finite profit floor.
func ResolveFundingArbMinProfit(v float64) (float64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0, fmt.Errorf("%w: minProfit must be > 0", ErrInvalidArgument)
	}
	if v > MaxFundingArbMinProfit {
		return 0, fmt.Errorf("%w: minProfit must be <= %s", ErrInvalidArgument, formatQty(MaxFundingArbMinProfit))
	}
	return v, nil
}

// FundingArbWatchPort persists watches and open/closed signals.
type FundingArbWatchPort interface {
	CreateWatch(ctx context.Context, w FundingArbWatch) (*FundingArbWatch, error)
	GetWatch(ctx context.Context, clientID, id string) (*FundingArbWatch, error)
	ListWatches(ctx context.Context, clientID string) ([]FundingArbWatch, error)
	ListActiveWatches(ctx context.Context) ([]FundingArbWatch, error)
	DeleteWatch(ctx context.Context, clientID, id string) error
	CountWatches(ctx context.Context, clientID string) (int, error)
	UpdateWatch(ctx context.Context, w FundingArbWatch) (*FundingArbWatch, error)
	SetWatchArmed(ctx context.Context, id string, armed bool, at time.Time) error

	GetOpenSignal(ctx context.Context, watchID, symbol string) (*FundingArbSignal, error)
	ListOpenSignals(ctx context.Context, watchID string) ([]FundingArbSignal, error)
	CreateSignal(ctx context.Context, s FundingArbSignal) (*FundingArbSignal, error)
	TouchSignal(ctx context.Context, id string, net float64, at time.Time) error
	CloseSignal(ctx context.Context, id string, at time.Time) error
	ListSignals(ctx context.Context, clientID string, status FundingArbSignalStatus, limit int) ([]FundingArbSignal, error)
}
