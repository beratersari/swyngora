package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	MaxAllocationRebalancesPerBasket = 50
	MaxAllocationRebalanceList       = 50
)

// Rebalance leg / run outcomes.
const (
	RebalanceLegFilled = "filled"
	RebalanceLegFailed = "failed"

	RebalanceResultCompleted = "completed"
	RebalanceResultPartial   = "partial"
	RebalanceResultFailed    = "failed"
	RebalanceResultNoTrades  = "no_trades"
)

// RebalanceLegOutcome is one planned market leg after execute.
type RebalanceLegOutcome struct {
	Side            TradeSide
	Asset           string
	Exchange        Exchange
	Symbol          string
	Reason          string
	PlannedQty      float64
	PlannedPrice    float64
	PlannedNotional float64
	Status          string
	TradeID         string
	FilledQty       float64
	FillPrice       float64
	FillNotional    float64
	LastPrice       float64 // venue last before paper slippage
	Fee             float64 // quote-currency taker fee
	Slippage        float64 // adverse quote impact vs last
	Error           string
}

// RebalanceLineSnapshot is target vs mix before and after one run.
type RebalanceLineSnapshot struct {
	Asset        string
	Exchange     Exchange
	Symbol       string
	IsCash       bool
	TargetPct    float64
	BeforePct    float64
	AfterPct     float64
	BeforeValue  float64
	AfterValue   float64
	TargetValue  float64
	AfterDelta   float64 // target − after; leftover to go
}

// RebalanceResult is a stored run the user can open later.
type RebalanceResult struct {
	ID                   string
	ClientID             string
	BasketID             string
	BasketName           string
	Currency             string
	Status               string
	PlannedLegs          int
	FilledLegs           int
	FailedLegs           int
	BeforeEquity         float64
	AfterEquity          float64
	BeforeCash           float64
	AfterCash            float64
	RemainingAbsDriftPct float64
	MaxSleeveDriftPct    float64
	RemainingNotional    float64
	TotalFee             float64
	TotalSlippage        float64
	TotalCost            float64 // fee + slippage
	Summary              string
	Note                 string
	CreatedAt            time.Time
	Legs                 []RebalanceLegOutcome
	Lines                []RebalanceLineSnapshot
}

// BuildRebalanceResult snapshots planned vs filled legs and leftover drift.
func BuildRebalanceResult(id string, basket AllocationBasket, before, after RebalancePlan, legs []RebalanceLegOutcome, now time.Time) RebalanceResult {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	filled, failed := 0, 0
	var totalFee, totalSlip float64
	for i := range legs {
		switch legs[i].Status {
		case RebalanceLegFilled:
			filled++
			FinishRebalanceLegCosts(&legs[i])
			totalFee += legs[i].Fee
			totalSlip += legs[i].Slippage
		case RebalanceLegFailed:
			failed++
		}
	}
	status := RebalanceResultNoTrades
	switch {
	case len(legs) == 0:
		status = RebalanceResultNoTrades
	case failed == 0:
		status = RebalanceResultCompleted
	case filled == 0:
		status = RebalanceResultFailed
	default:
		status = RebalanceResultPartial
	}

	beforeBy := map[string]AllocationLine{}
	for _, ln := range before.Lines {
		beforeBy[ln.Asset] = ln
	}
	var remAbs, remNotional, maxDrift float64
	lines := make([]RebalanceLineSnapshot, 0, len(after.Lines))
	for _, ln := range after.Lines {
		drift := ln.ActualPct - ln.TargetPct
		abs := math.Abs(drift)
		if abs > maxDrift {
			maxDrift = abs
		}
		remAbs += abs
		remNotional += math.Abs(ln.DeltaValue)
		b := beforeBy[ln.Asset]
		lines = append(lines, RebalanceLineSnapshot{
			Asset: ln.Asset, Exchange: ln.Exchange, Symbol: ln.Symbol, IsCash: ln.IsCash,
			TargetPct: ln.TargetPct, BeforePct: b.ActualPct, AfterPct: ln.ActualPct,
			BeforeValue: b.CurrentValue, AfterValue: ln.CurrentValue,
			TargetValue: ln.TargetValue, AfterDelta: ln.DeltaValue,
		})
	}

	r := RebalanceResult{
		ID: id, ClientID: basket.ClientID, BasketID: basket.ID, BasketName: basket.Name,
		Currency: after.Currency, Status: status,
		PlannedLegs: len(legs), FilledLegs: filled, FailedLegs: failed,
		BeforeEquity: before.Equity, AfterEquity: after.Equity,
		BeforeCash: before.Cash, AfterCash: after.Cash,
		RemainingAbsDriftPct: remAbs, MaxSleeveDriftPct: maxDrift, RemainingNotional: remNotional,
		TotalFee: totalFee, TotalSlippage: totalSlip, TotalCost: totalFee + totalSlip,
		CreatedAt: now, Legs: legs, Lines: lines,
	}
	r.Summary = rebalanceResultSummary(r)
	r.Note = rebalanceResultNote(r)
	return r
}

func rebalanceResultSummary(r RebalanceResult) string {
	var b strings.Builder
	switch r.Status {
	case RebalanceResultNoTrades:
		b.WriteString("No trades — already near targets (or deltas below the min size).")
	case RebalanceResultCompleted:
		fmt.Fprintf(&b, "Filled all %d planned legs.", r.FilledLegs)
	case RebalanceResultPartial:
		fmt.Fprintf(&b, "Filled %d of %d planned legs; %d failed.", r.FilledLegs, r.PlannedLegs, r.FailedLegs)
	case RebalanceResultFailed:
		fmt.Fprintf(&b, "None of the %d planned legs filled.", r.PlannedLegs)
	default:
		fmt.Fprintf(&b, "Ran %d planned legs.", r.PlannedLegs)
	}
	if r.Status != RebalanceResultNoTrades {
		fmt.Fprintf(&b, " Remaining mix drift %.2f%% (largest sleeve %.2f%%, leftover %.2f %s).",
			r.RemainingAbsDriftPct, r.MaxSleeveDriftPct, r.RemainingNotional, r.Currency)
		if r.TotalCost > 0 {
			fmt.Fprintf(&b, " Paper costs %.2f %s (fees %.2f, slippage %.2f).",
				r.TotalCost, r.Currency, r.TotalFee, r.TotalSlippage)
		}
	}
	return b.String()
}

// ApplyRebalanceFill copies a paper trade onto a planned leg and derives costs.
func ApplyRebalanceFill(o *RebalanceLegOutcome, tr *Trade) {
	if o == nil || tr == nil {
		return
	}
	o.TradeID = tr.ID
	o.FilledQty = tr.Quantity
	o.FillPrice = tr.Price
	o.FillNotional = tr.Notional
	o.LastPrice = tr.LastPrice
	o.Fee = tr.Fee
	FinishRebalanceLegCosts(o)
}

// FinishRebalanceLegCosts fills slippage from last vs fill when missing.
func FinishRebalanceLegCosts(o *RebalanceLegOutcome) {
	if o == nil || o.FilledQty <= 0 || o.FillPrice <= 0 {
		return
	}
	last := o.LastPrice
	if last <= 0 {
		last = o.PlannedPrice
	}
	if o.Slippage <= 0 {
		o.Slippage = AdverseSlippageNotional(o.Side, last, o.FillPrice, o.FilledQty)
	}
}

// AdverseSlippageNotional is the quote-currency cost of fill vs last (buys pay more, sells receive less).
func AdverseSlippageNotional(side TradeSide, last, fill, qty float64) float64 {
	if qty <= 0 || last <= 0 || fill <= 0 || math.IsNaN(last) || math.IsNaN(fill) {
		return 0
	}
	switch side {
	case TradeSideBuy:
		if fill > last {
			return (fill - last) * qty
		}
	case TradeSideSell:
		if last > fill {
			return (last - fill) * qty
		}
	}
	return 0
}

func rebalanceResultNote(r RebalanceResult) string {
	switch r.Status {
	case RebalanceResultNoTrades:
		return "Already near targets (or deltas below min notional). No trades. Paper trading; not real money."
	case RebalanceResultFailed:
		return "Rebalance ran but no legs filled. Check failed legs. Paper trading; not real money."
	case RebalanceResultPartial:
		return "Rebalance partly filled. Remaining drift is allowed until you run it again. Paper trading; not real money."
	default:
		return "Rebalance executed at last prices. Drift is allowed until you run it again. Paper trading; not real money."
	}
}
