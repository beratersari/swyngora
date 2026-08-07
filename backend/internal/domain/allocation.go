package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Paper allocation-basket limits (spot only; rebalance is user-triggered).
const (
	MaxAllocationBasketsPerClient = 10
	MaxAllocationTargetsPerBasket = 20
	MaxAllocationNameLen          = 80
	MinAllocationWeightPct        = 0.01
	AllocationWeightSumEpsilon    = 0.05
	MinRebalanceNotional          = 1.0
)

// AllocationTarget is one sleeve of a basket (coin or portfolio cash currency).
type AllocationTarget struct {
	Asset     string
	Exchange  Exchange // ignored for cash
	WeightPct float64
}

// AllocationBasket is a named target mix the user may rebalance toward on demand.
type AllocationBasket struct {
	ID        string
	ClientID  string
	Name      string
	Targets   []AllocationTarget
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AllocationHolding is a marked spot sleeve used to compute actual weights.
type AllocationHolding struct {
	Asset        string
	Exchange     Exchange
	Symbol       string
	IsCash       bool
	MarkPrice    float64
	Quantity     float64
	AvailableQty float64
	MarketValue  float64
}

// AllocationLine is target vs actual for one sleeve.
type AllocationLine struct {
	Asset        string
	Exchange     Exchange
	Symbol       string
	IsCash       bool
	TargetPct    float64
	ActualPct    float64
	CurrentValue float64
	TargetValue  float64
	DeltaValue   float64 // target − current; +buy / −sell
	MarkPrice    float64
}

// RebalanceLeg is one market buy or sell suggested or executed by rebalance.
type RebalanceLeg struct {
	Side     TradeSide
	Asset    string
	Exchange Exchange
	Symbol   string
	Quantity float64
	Price    float64
	Notional float64
	Reason   string // overweight | underweight | not_in_basket
}

// RebalancePlan is a preview of bringing spot holdings toward the basket.
// Never applied automatically — ExecuteRebalance is a separate user action.
type RebalancePlan struct {
	Currency      string
	Equity        float64
	Cash          float64
	AvailableCash float64
	Lines         []AllocationLine
	Legs          []RebalanceLeg
}

// NormalizeAllocationAsset uppercases a coin or cash ticker.
func NormalizeAllocationAsset(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// NormalizeAllocationName trims a basket label.
func NormalizeAllocationName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: basket name is required", ErrInvalidArgument)
	}
	if strings.ContainsAny(name, "\x00\r\n") {
		return "", fmt.Errorf("%w: name must be a single line", ErrInvalidArgument)
	}
	if len([]rune(name)) > MaxAllocationNameLen {
		return "", fmt.Errorf("%w: name must be at most %d characters", ErrInvalidArgument, MaxAllocationNameLen)
	}
	return name, nil
}

// ValidateAllocationTargets checks count, uniqueness, range, and ~100% sum.
func ValidateAllocationTargets(targets []AllocationTarget, currency string) error {
	if len(targets) == 0 {
		return fmt.Errorf("%w: at least one target is required", ErrInvalidArgument)
	}
	if len(targets) > MaxAllocationTargetsPerBasket {
		return fmt.Errorf("%w: at most %d targets per basket", ErrInvalidArgument, MaxAllocationTargetsPerBasket)
	}
	currency = NormalizeAllocationAsset(currency)
	seen := map[string]struct{}{}
	var sum float64
	for i := range targets {
		a := NormalizeAllocationAsset(targets[i].Asset)
		if a == "" {
			return fmt.Errorf("%w: target asset is required", ErrInvalidArgument)
		}
		if _, ok := seen[a]; ok {
			return fmt.Errorf("%w: duplicate target asset %s", ErrInvalidArgument, a)
		}
		seen[a] = struct{}{}
		w := targets[i].WeightPct
		if w < MinAllocationWeightPct || w > 100 || math.IsNaN(w) || math.IsInf(w, 0) {
			return fmt.Errorf("%w: weightPct for %s must be between %g and 100", ErrInvalidArgument, a, MinAllocationWeightPct)
		}
		if a != currency && targets[i].Exchange != "" && !IsValidExchange(string(targets[i].Exchange)) {
			return fmt.Errorf("%w: invalid exchange for %s", ErrInvalidArgument, a)
		}
		sum += w
	}
	if math.Abs(sum-100) > AllocationWeightSumEpsilon {
		return fmt.Errorf("%w: target weights must sum to 100 (got %g)", ErrInvalidArgument, sum)
	}
	return nil
}

// IsCashAllocation reports whether asset is the portfolio quote currency.
func IsCashAllocation(asset, currency string) bool {
	return NormalizeAllocationAsset(asset) == NormalizeAllocationAsset(currency)
}

// PlanRebalance computes drift lines and market legs (sells first, then buys).
// Holdings not listed in the basket have target 0% and are sold on rebalance.
// Cash is never traded as a pair; it absorbs sell proceeds and funds buys.
func PlanRebalance(currency string, equity, cash, availCash float64, holdings []AllocationHolding, targets []AllocationTarget) (RebalancePlan, error) {
	currency = NormalizeAllocationAsset(currency)
	if err := ValidateAllocationTargets(targets, currency); err != nil {
		return RebalancePlan{}, err
	}
	if equity <= PositionEpsilon || math.IsNaN(equity) || math.IsInf(equity, 0) {
		return RebalancePlan{}, fmt.Errorf("%w: equity must be positive to rebalance", ErrInvalidArgument)
	}
	if availCash < 0 {
		availCash = 0
	}

	targetBy := map[string]AllocationTarget{}
	for _, t := range targets {
		a := NormalizeAllocationAsset(t.Asset)
		t.Asset = a
		if t.Exchange == "" && a != currency {
			t.Exchange = DefaultExchange
		}
		targetBy[a] = t
	}

	holdBy := map[string]AllocationHolding{}
	for _, h := range holdings {
		a := NormalizeAllocationAsset(h.Asset)
		h.Asset = a
		h.IsCash = a == currency
		holdBy[a] = h
	}
	if _, ok := holdBy[currency]; !ok {
		holdBy[currency] = AllocationHolding{Asset: currency, IsCash: true, MarketValue: cash, Quantity: cash, AvailableQty: availCash, MarkPrice: 1}
	} else {
		c := holdBy[currency]
		c.IsCash = true
		c.MarketValue = cash
		c.MarkPrice = 1
		c.AvailableQty = availCash
		holdBy[currency] = c
	}

	universe := map[string]struct{}{}
	for a := range targetBy {
		universe[a] = struct{}{}
	}
	for a := range holdBy {
		universe[a] = struct{}{}
	}

	var lines []AllocationLine
	for a := range universe {
		h := holdBy[a]
		t, inBasket := targetBy[a]
		isCash := a == currency
		ex := h.Exchange
		sym := h.Symbol
		px := h.MarkPrice
		if !isCash {
			if h.Quantity <= PositionEpsilon || ex == "" {
				if inBasket && t.Exchange != "" {
					ex = t.Exchange
				}
				if ex == "" {
					ex = DefaultExchange
				}
			}
			if sym == "" {
				sym = PairSymbol(ex, a, currency)
			}
		}
		var tgtPct float64
		if inBasket {
			tgtPct = t.WeightPct
		}
		curVal := h.MarketValue
		if isCash {
			curVal = cash
		}
		tgtVal := equity * (tgtPct / 100)
		actual := 0.0
		if equity > 0 {
			actual = 100 * curVal / equity
		}
		lines = append(lines, AllocationLine{
			Asset: a, Exchange: ex, Symbol: sym, IsCash: isCash,
			TargetPct: tgtPct, ActualPct: actual,
			CurrentValue: curVal, TargetValue: tgtVal, DeltaValue: tgtVal - curVal,
			MarkPrice: px,
		})
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].Asset < lines[j].Asset })

	var sells, buys []RebalanceLeg
	for _, ln := range lines {
		if ln.IsCash {
			continue
		}
		if ln.MarkPrice <= 0 || math.IsNaN(ln.MarkPrice) {
			continue
		}
		if ln.DeltaValue <= -MinRebalanceNotional {
			h := holdBy[ln.Asset]
			qty := math.Abs(ln.DeltaValue) / ln.MarkPrice
			if h.AvailableQty > 0 && qty > h.AvailableQty {
				qty = h.AvailableQty
			}
			if qty < MinTradeQuantity {
				continue
			}
			notional := qty * ln.MarkPrice
			if notional < MinRebalanceNotional {
				continue
			}
			reason := "overweight"
			if _, in := targetBy[ln.Asset]; !in {
				reason = "not_in_basket"
			}
			sells = append(sells, RebalanceLeg{
				Side: TradeSideSell, Asset: ln.Asset, Exchange: ln.Exchange, Symbol: ln.Symbol,
				Quantity: qty, Price: ln.MarkPrice, Notional: notional, Reason: reason,
			})
		} else if ln.DeltaValue >= MinRebalanceNotional {
			if _, in := targetBy[ln.Asset]; !in {
				continue
			}
			buys = append(buys, RebalanceLeg{
				Side: TradeSideBuy, Asset: ln.Asset, Exchange: ln.Exchange, Symbol: ln.Symbol,
				Price: ln.MarkPrice, Notional: ln.DeltaValue, Reason: "underweight",
			})
		}
	}
	sort.Slice(sells, func(i, j int) bool { return sells[i].Notional > sells[j].Notional })
	sort.Slice(buys, func(i, j int) bool { return buys[i].Notional > buys[j].Notional })

	free := availCash
	for _, s := range sells {
		free += s.Notional
	}
	var funded []RebalanceLeg
	for _, b := range buys {
		if free < MinRebalanceNotional {
			break
		}
		spend := b.Notional
		if spend > free {
			spend = free
		}
		if spend < MinRebalanceNotional || b.Price <= 0 {
			continue
		}
		qty := spend / b.Price
		if qty < MinTradeQuantity {
			continue
		}
		b.Quantity = qty
		b.Notional = spend
		funded = append(funded, b)
		free -= spend
	}

	legs := append(append([]RebalanceLeg{}, sells...), funded...)
	return RebalancePlan{
		Currency: currency, Equity: equity, Cash: cash, AvailableCash: availCash,
		Lines: lines, Legs: legs,
	}, nil
}
