package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// Price-diff watch limits.
const (
	MaxPriceDiffWatchesPerClient = 20
	MinPriceDiffNetPct           = 0.01  // 0.01%
	MaxPriceDiffNetPct           = 50.0  // 50%
	MaxPriceDiffFeePct           = 10.0  // 10% per exchange
	// DefaultPriceDiffMaxAge is how fresh a ticker CloseTime must be to count.
	DefaultPriceDiffMaxAge = 2 * time.Minute
)

// PriceDiffWatchStatus is active (evaluated) or paused.
type PriceDiffWatchStatus string

const (
	PriceDiffWatchActive PriceDiffWatchStatus = "active"
	PriceDiffWatchPaused PriceDiffWatchStatus = "paused"
)

// PriceDiffOppStatus is open (still above threshold) or closed (went below).
type PriceDiffOppStatus string

const (
	PriceDiffOppOpen   PriceDiffOppStatus = "open"
	PriceDiffOppClosed PriceDiffOppStatus = "closed"
)

// PriceDiffWatch is a user-configured cross-exchange price difference tracker.
// MinNetDiffPct is the minimum net % edge after fees required to open an opportunity.
// Fee*Pct are taker-style fees in percent (0.1 = 0.1%) per exchange.
type PriceDiffWatch struct {
	ID             string
	ClientID       string
	Symbol         string // canonical pair e.g. BTCUSDT
	MinNetDiffPct  float64
	FeeBinancePct  float64
	FeeCoinbasePct float64
	FeeBybitPct    float64
	Status         PriceDiffWatchStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PriceDiffOpportunity is a detected buy/sell route across two exchanges.
// While status=open, the same (watch_id, buy_ex, sell_ex) must not be recreated.
// When net edge falls below the watch threshold the opportunity is closed;
// a later re-cross creates a new opportunity row.
type PriceDiffOpportunity struct {
	ID            string
	WatchID       string
	ClientID      string
	Symbol        string
	BuyExchange   Exchange
	SellExchange  Exchange
	BuyPrice      float64
	SellPrice     float64
	GrossDiffPct  float64
	NetDiffPct    float64
	MinNetDiffPct float64 // threshold snapshot at open
	Status        PriceDiffOppStatus
	OpenedAt      time.Time
	LastSeenAt    time.Time
	ClosedAt      *time.Time
}

// PriceDiffRoute is one directional arb evaluation (buy on A, sell on B).
type PriceDiffRoute struct {
	BuyExchange  Exchange
	SellExchange Exchange
	BuyPrice     float64
	SellPrice    float64
	GrossDiffPct float64
	NetDiffPct   float64
}

// FeePctFor returns the watch fee percent for an exchange.
func (w PriceDiffWatch) FeePctFor(ex Exchange) float64 {
	switch ex {
	case ExchangeBinance:
		return w.FeeBinancePct
	case ExchangeCoinbase:
		return w.FeeCoinbasePct
	case ExchangeBybit:
		return w.FeeBybitPct
	default:
		return 0
	}
}

// NetDiffPctAfterFees computes net edge % when buying at buyPrice (fee buyFeePct%)
// and selling at sellPrice (fee sellFeePct%). Fees are percent points (0.1 = 0.1%).
// net = (sell*(1-sellFee) / (buy*(1+buyFee)) - 1) * 100
func NetDiffPctAfterFees(buyPrice, sellPrice, buyFeePct, sellFeePct float64) (grossPct, netPct float64, err error) {
	if buyPrice <= 0 || sellPrice <= 0 || math.IsNaN(buyPrice) || math.IsNaN(sellPrice) ||
		math.IsInf(buyPrice, 0) || math.IsInf(sellPrice, 0) {
		return 0, 0, fmt.Errorf("%w: prices must be positive", ErrInvalidArgument)
	}
	if buyFeePct < 0 || sellFeePct < 0 || buyFeePct > MaxPriceDiffFeePct || sellFeePct > MaxPriceDiffFeePct ||
		math.IsNaN(buyFeePct) || math.IsNaN(sellFeePct) {
		return 0, 0, fmt.Errorf("%w: fee percent out of range", ErrInvalidArgument)
	}
	grossPct = (sellPrice/buyPrice - 1) * 100
	buyCost := buyPrice * (1 + buyFeePct/100)
	sellProceeds := sellPrice * (1 - sellFeePct/100)
	if buyCost <= 0 {
		return 0, 0, fmt.Errorf("%w: invalid buy cost after fees", ErrInvalidArgument)
	}
	netPct = (sellProceeds/buyCost - 1) * 100
	return grossPct, netPct, nil
}

// BestPriceDiffRoutes evaluates all ordered exchange pairs with fresh prices.
// prices maps exchange → last price (only include fresh, valid prices).
// Returns routes where netPct >= minNetDiffPct, sorted by netPct descending.
func BestPriceDiffRoutes(prices map[Exchange]float64, fees map[Exchange]float64, minNetDiffPct float64) []PriceDiffRoute {
	exs := make([]Exchange, 0, len(prices))
	for ex, p := range prices {
		if p > 0 {
			exs = append(exs, ex)
		}
	}
	var out []PriceDiffRoute
	for i := range exs {
		for j := range exs {
			if i == j {
				continue
			}
			buyEx, sellEx := exs[i], exs[j]
			buyP, sellP := prices[buyEx], prices[sellEx]
			gross, net, err := NetDiffPctAfterFees(buyP, sellP, fees[buyEx], fees[sellEx])
			if err != nil {
				continue
			}
			if net+1e-12 < minNetDiffPct {
				continue
			}
			out = append(out, PriceDiffRoute{
				BuyExchange: buyEx, SellExchange: sellEx,
				BuyPrice: buyP, SellPrice: sellP,
				GrossDiffPct: gross, NetDiffPct: net,
			})
		}
	}
	// Sort by net desc (simple insertion for tiny n).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].NetDiffPct > out[j-1].NetDiffPct; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// PriceDiffSymbolForExchange maps a canonical symbol (e.g. BTCUSDT) to the
// venue product id. Coinbase USDT pairs map to USD (BTCUSDT → BTC-USD).
func PriceDiffSymbolForExchange(ex Exchange, canonical string) string {
	canonical = strings.TrimSpace(canonical)
	if canonical == "" {
		return ""
	}
	if ex == ExchangeCoinbase {
		s := NormalizeSymbol(ex, canonical)
		if strings.HasSuffix(s, "-USDT") {
			return strings.TrimSuffix(s, "-USDT") + "-USD"
		}
		return s
	}
	return NormalizeSymbol(ex, canonical)
}

// IsFreshTicker reports whether the ticker's CloseTime is within maxAge of now.
// Zero CloseTime is treated as not fresh (caller must not invent signals).
func IsFreshTicker(t *Ticker24h, now time.Time, maxAge time.Duration) bool {
	if t == nil || maxAge <= 0 {
		return false
	}
	if t.CloseTime.IsZero() {
		return false
	}
	age := now.UTC().Sub(t.CloseTime.UTC())
	return age >= 0 && age <= maxAge
}

// PriceDiffPort persists watches and opportunities.
type PriceDiffPort interface {
	CreateWatch(ctx context.Context, w PriceDiffWatch) (*PriceDiffWatch, error)
	GetWatch(ctx context.Context, clientID, id string) (*PriceDiffWatch, error)
	ListWatches(ctx context.Context, clientID string) ([]PriceDiffWatch, error)
	ListActiveWatches(ctx context.Context) ([]PriceDiffWatch, error)
	DeleteWatch(ctx context.Context, clientID, id string) error
	CountWatches(ctx context.Context, clientID string) (int, error)

	// GetOpenOpportunity returns open opp for (watch, buy, sell) or ErrNotFound.
	GetOpenOpportunity(ctx context.Context, watchID string, buy, sell Exchange) (*PriceDiffOpportunity, error)
	// CreateOpportunity inserts a new open opportunity.
	CreateOpportunity(ctx context.Context, o PriceDiffOpportunity) (*PriceDiffOpportunity, error)
	// TouchOpportunity updates prices/net/last_seen for an open opportunity.
	TouchOpportunity(ctx context.Context, id string, buyPrice, sellPrice, grossPct, netPct float64, at time.Time) (*PriceDiffOpportunity, error)
	// CloseOpportunity sets status closed if still open.
	CloseOpportunity(ctx context.Context, id string, at time.Time) (*PriceDiffOpportunity, error)
	// ListOpenOpportunitiesForWatch returns all open opps for a watch.
	ListOpenOpportunitiesForWatch(ctx context.Context, watchID string) ([]PriceDiffOpportunity, error)
	ListOpportunities(ctx context.Context, clientID string, status PriceDiffOppStatus, limit, offset int) ([]PriceDiffOpportunity, error)
	GetOpportunity(ctx context.Context, clientID, id string) (*PriceDiffOpportunity, error)
}
