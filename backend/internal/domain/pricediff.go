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
	MinPriceDiffNetPct           = 0.20 // 0.20% — above typical USDT/USD noise
	MaxPriceDiffNetPct           = 50.0 // 50%
	MaxPriceDiffFeePct           = 10.0 // 10% per exchange
	MinPriceDiffNotional         = 1.0
	MaxPriceDiffNotional         = 1e9
	MaxPriceDiffMinProfit        = 1e9
	MaxPriceDiffMinDurationSec   = 24 * 60 * 60 // 24h
	// DefaultPriceDiffMaxAge is how fresh a ticker CloseTime or book FetchedAt must be.
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
// Notional is the quote size walked on live books. MinProfit is the minimum
// after-fee profit (quote currency) required to open an opportunity.
// MinNetDiffPct is an optional extra % floor. Fee*Pct are taker fees (0.1 = 0.1%).
type PriceDiffWatch struct {
	ID             string
	ClientID       string
	Symbol         string // canonical pair e.g. BTCUSDT
	Notional       float64
	MinProfit      float64
	MinNetDiffPct  float64
	MinDurationSec float64 // condition must stay true this long before open; 0 = immediate
	FeeBinancePct  float64
	FeeCoinbasePct float64
	FeeBybitPct    float64
	Status         PriceDiffWatchStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PriceDiffRouteArm is when a watch first saw a qualifying book fill on a route.
type PriceDiffRouteArm struct {
	WatchID      string
	BuyExchange  Exchange
	SellExchange Exchange
	ArmedAt      time.Time
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

// IsFreshOrderBook reports whether the book's FetchedAt is within maxAge of now.
// Zero FetchedAt is not fresh.
func IsFreshOrderBook(b *RawOrderBook, now time.Time, maxAge time.Duration) bool {
	if b == nil || maxAge <= 0 {
		return false
	}
	if b.FetchedAt.IsZero() {
		return false
	}
	age := now.UTC().Sub(b.FetchedAt.UTC())
	if age < 0 {
		return true
	}
	return age <= maxAge
}

// ResolvePriceDiffMinDuration accepts seconds (0 = open on first qualifying tick).
func ResolvePriceDiffMinDuration(sec float64) (time.Duration, error) {
	if sec < 0 || sec > MaxPriceDiffMinDurationSec || math.IsNaN(sec) || math.IsInf(sec, 0) {
		return 0, fmt.Errorf("%w: minDurationSec must be between 0 and %d", ErrInvalidArgument, MaxPriceDiffMinDurationSec)
	}
	return time.Duration(sec * float64(time.Second)), nil
}

// PriceDiffHeldLongEnough is true when min is 0, or since has been true for min.
func PriceDiffHeldLongEnough(since, now time.Time, min time.Duration) bool {
	if min <= 0 {
		return true
	}
	if since.IsZero() {
		return false
	}
	return !now.UTC().Before(since.UTC().Add(min))
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

	// Route arms remember when a qualifying fill first appeared so min duration can elapse.
	ListRouteArms(ctx context.Context, watchID string) ([]PriceDiffRouteArm, error)
	// SetRouteArm records first-seen time; an existing arm is left unchanged.
	SetRouteArm(ctx context.Context, arm PriceDiffRouteArm) error
	ClearRouteArm(ctx context.Context, watchID string, buy, sell Exchange) error
}
