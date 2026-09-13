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
	MaxRebalanceAttempts             = 10
)

// Rebalance leg / run outcomes.
const (
	RebalanceLegFilled = "filled"
	RebalanceLegFailed = "failed"

	RebalanceResultCompleted    = "completed"
	RebalanceResultPartial      = "partial"
	RebalanceResultFailed       = "failed"
	RebalanceResultNoTrades     = "no_trades"
	RebalanceResultRunning      = "running"
	RebalanceResultInterrupted  = "interrupted"
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
	Attempt         int
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
	Attempt      int
}

// RebalanceAttempt is one try (first run or a later continue).
type RebalanceAttempt struct {
	Seq           int
	Status        string
	PlannedLegs   int
	FilledLegs    int
	FailedLegs    int
	BeforeEquity  float64
	AfterEquity   float64
	BeforeCash    float64
	AfterCash     float64
	TotalFee      float64
	TotalSlippage float64
	TotalCost     float64
	Summary       string
	CreatedAt     time.Time
	Legs          []RebalanceLegOutcome
	Lines         []RebalanceLineSnapshot
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
	UpdatedAt            time.Time
	AttemptCount         int
	CanContinue          bool
	Legs                 []RebalanceLegOutcome // all attempts, oldest first
	Lines                []RebalanceLineSnapshot
	Attempts             []RebalanceAttempt
}

// BuildRebalanceResult snapshots planned vs filled legs and leftover drift (attempt 1).
func BuildRebalanceResult(id string, basket AllocationBasket, before, after RebalancePlan, legs []RebalanceLegOutcome, now time.Time) RebalanceResult {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	r := RebalanceResult{
		ID: id, ClientID: basket.ClientID, BasketID: basket.ID, BasketName: basket.Name,
		Currency: after.Currency, CreatedAt: now, UpdatedAt: now,
	}
	att := BuildRebalanceAttempt(1, before, after, legs, now)
	ApplyRebalanceAttempt(&r, att)
	return r
}

// BuildRebalanceAttempt snapshots one try against a before/after plan.
func BuildRebalanceAttempt(seq int, before, after RebalancePlan, legs []RebalanceLegOutcome, now time.Time) RebalanceAttempt {
	if seq <= 0 {
		seq = 1
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	filled, failed := 0, 0
	var totalFee, totalSlip float64
	outLegs := append([]RebalanceLegOutcome(nil), legs...)
	for i := range outLegs {
		outLegs[i].Attempt = seq
		switch outLegs[i].Status {
		case RebalanceLegFilled:
			filled++
			FinishRebalanceLegCosts(&outLegs[i])
			totalFee += outLegs[i].Fee
			totalSlip += outLegs[i].Slippage
		case RebalanceLegFailed:
			failed++
		}
	}
	lines, remAbs, remNotional, maxDrift := rebalanceLineSnapshots(seq, before, after)
	att := RebalanceAttempt{
		Seq: seq, PlannedLegs: len(outLegs), FilledLegs: filled, FailedLegs: failed,
		BeforeEquity: before.Equity, AfterEquity: after.Equity,
		BeforeCash: before.Cash, AfterCash: after.Cash,
		TotalFee: totalFee, TotalSlippage: totalSlip, TotalCost: totalFee + totalSlip,
		CreatedAt: now, Legs: outLegs, Lines: lines,
	}
	att.Status = RebalanceAttemptStatus(remNotional, filled, failed, len(outLegs))
	att.Summary = rebalanceAttemptSummary(att, remAbs, remNotional, maxDrift, after.Currency)
	return att
}

func rebalanceLineSnapshots(seq int, before, after RebalancePlan) (lines []RebalanceLineSnapshot, remAbs, remNotional, maxDrift float64) {
	beforeBy := map[string]AllocationLine{}
	for _, ln := range before.Lines {
		beforeBy[ln.Asset] = ln
	}
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
			TargetValue: ln.TargetValue, AfterDelta: ln.DeltaValue, Attempt: seq,
		})
	}
	return lines, remAbs, remNotional, maxDrift
}

// ApplyRebalanceAttempt rolls an attempt onto the stored job and refreshes leftover mix.
func ApplyRebalanceAttempt(r *RebalanceResult, att RebalanceAttempt) {
	if r == nil {
		return
	}
	if att.Seq <= 0 {
		att.Seq = len(r.Attempts) + 1
	}
	r.Attempts = append(r.Attempts, att)
	r.AttemptCount = len(r.Attempts)
	r.PlannedLegs += att.PlannedLegs
	r.FilledLegs += att.FilledLegs
	r.FailedLegs += att.FailedLegs
	r.TotalFee += att.TotalFee
	r.TotalSlippage += att.TotalSlippage
	r.TotalCost = r.TotalFee + r.TotalSlippage
	if r.BeforeEquity == 0 && r.BeforeCash == 0 && att.Seq == 1 {
		r.BeforeEquity = att.BeforeEquity
		r.BeforeCash = att.BeforeCash
	}
	r.AfterEquity = att.AfterEquity
	r.AfterCash = att.AfterCash
	r.UpdatedAt = att.CreatedAt
	r.Legs = append(r.Legs, att.Legs...)
	r.Lines = att.Lines
	var remAbs, remNotional, maxDrift float64
	for _, ln := range att.Lines {
		d := math.Abs(ln.AfterPct - ln.TargetPct)
		if d > maxDrift {
			maxDrift = d
		}
		remAbs += d
		remNotional += math.Abs(ln.AfterDelta)
	}
	r.RemainingAbsDriftPct = remAbs
	r.MaxSleeveDriftPct = maxDrift
	r.RemainingNotional = remNotional
	r.Status = RebalanceAttemptStatus(remNotional, r.FilledLegs, r.FailedLegs, r.PlannedLegs)
	r.CanContinue = RebalanceCanContinue(*r)
	r.Summary = rebalanceResultSummary(*r)
	r.Note = rebalanceResultNote(*r)
}

// RebalanceAttemptStatus is leftover-aware: leftover mix can stay partial even if every planned leg filled.
func RebalanceAttemptStatus(remainingNotional float64, filled, failed, planned int) string {
	if remainingNotional+1e-9 < MinRebalanceNotional {
		if planned == 0 && filled == 0 {
			return RebalanceResultNoTrades
		}
		return RebalanceResultCompleted
	}
	if planned == 0 {
		return RebalanceResultPartial
	}
	if filled == 0 {
		return RebalanceResultFailed
	}
	if failed > 0 || remainingNotional >= MinRebalanceNotional {
		return RebalanceResultPartial
	}
	return RebalanceResultCompleted
}

// RebalanceCanContinue reports leftover mix still worth another try.
func RebalanceCanContinue(r RebalanceResult) bool {
	if r.AttemptCount >= MaxRebalanceAttempts {
		return false
	}
	switch r.Status {
	case RebalanceResultRunning, RebalanceResultInterrupted, RebalanceResultPartial, RebalanceResultFailed:
		return true
	default:
		return r.RemainingNotional+1e-9 >= MinRebalanceNotional
	}
}

func rebalanceAttemptSummary(att RebalanceAttempt, remAbs, remNotional, maxDrift float64, currency string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Attempt %d: ", att.Seq)
	switch {
	case att.PlannedLegs == 0:
		b.WriteString("no new legs.")
	case att.FailedLegs == 0:
		fmt.Fprintf(&b, "filled all %d legs.", att.FilledLegs)
	case att.FilledLegs == 0:
		fmt.Fprintf(&b, "none of %d legs filled.", att.PlannedLegs)
	default:
		fmt.Fprintf(&b, "filled %d of %d legs; %d failed.", att.FilledLegs, att.PlannedLegs, att.FailedLegs)
	}
	if remNotional >= MinRebalanceNotional {
		fmt.Fprintf(&b, " Leftover %.2f %s (drift %.2f%%).", remNotional, currency, remAbs)
	}
	if att.TotalCost > 0 {
		fmt.Fprintf(&b, " Costs %.2f %s.", att.TotalCost, currency)
	}
	return b.String()
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
	if r.AttemptCount > 1 {
		fmt.Fprintf(&b, " %d attempts.", r.AttemptCount)
	}
	if r.Status != RebalanceResultNoTrades {
		fmt.Fprintf(&b, " Remaining mix drift %.2f%% (largest sleeve %.2f%%, leftover %.2f %s).",
			r.RemainingAbsDriftPct, r.MaxSleeveDriftPct, r.RemainingNotional, r.Currency)
		if r.TotalCost > 0 {
			fmt.Fprintf(&b, " Paper costs %.2f %s (fees %.2f, slippage %.2f).",
				r.TotalCost, r.Currency, r.TotalFee, r.TotalSlippage)
		}
	}
	if r.CanContinue {
		b.WriteString(" Can continue from the current book.")
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
		return "Rebalance partly filled. Continue to finish leftover mix from the current book without replaying filled legs. Paper trading; not real money."
	case RebalanceResultInterrupted:
		return "Rebalance stopped before every planned leg finished. Continue to re-plan leftover mix from the current book. Paper trading; not real money."
	case RebalanceResultRunning:
		return "Rebalance is still running. Paper trading; not real money."
	default:
		return "Rebalance executed at last prices. Drift is allowed until you run it again. Paper trading; not real money."
	}
}
