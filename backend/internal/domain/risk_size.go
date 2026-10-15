package domain

import (
	"fmt"
	"math"
)

// RiskSizeInput is the book equity, the percent of that equity to lose at the
// stop, and the entry and stop prices before slippage.
// Long is a spot buy or a margin long. A margin short sets Long to false.
type RiskSizeInput struct {
	Equity      float64
	RiskPercent float64
	Entry       float64
	Stop        float64
	Long        bool
	Cost        TradingCost
}

// OrderPreviewSizing is the quantity that loses about RiskBudget if the stop fills.
// The loss of one unit includes the taker fee and adverse slippage on the entry
// and on the stop, which is the same loss a paper fill would book.
type OrderPreviewSizing struct {
	RiskPercent           float64
	Equity                float64
	RiskBudget            float64
	Quantity              float64
	EntryPrice            float64
	StopPrice             float64
	LossPerUnit           float64
	EstimatedLoss         float64
	StopAttached          bool
	StopBeyondLiquidation bool
}

// ValidateRiskPercent accepts a percent of equity strictly above 0 and at most 100.
func ValidateRiskPercent(p float64) error {
	if p <= 0 || p > 100 || math.IsNaN(p) || math.IsInf(p, 0) {
		return fmt.Errorf("%w: riskPercent must be greater than 0 and at most 100", ErrInvalidArgument)
	}
	return nil
}

// SizeByStopRisk returns the base quantity whose loss at the stop is the risk budget.
//
//	budget = equity * riskPercent / 100
//	quantity = budget / lossPerUnit
//
// lossPerUnit is the cash lost on one unit after fees and slippage. For a long
// that is the buy debit minus the stop-sell credit. For a short it is the
// buy-to-cover debit minus the open-sell credit. Leverage is not an input:
// it changes margin and liquidation, not the loss if the stop fills first.
// Quantity is rounded to 1e-8, the smallest paper size.
func SizeByStopRisk(in RiskSizeInput) (OrderPreviewSizing, error) {
	if err := ValidateRiskPercent(in.RiskPercent); err != nil {
		return OrderPreviewSizing{}, err
	}
	if in.Equity <= PositionEpsilon || math.IsNaN(in.Equity) || math.IsInf(in.Equity, 0) {
		return OrderPreviewSizing{}, fmt.Errorf("%w: portfolio equity must be positive to size by risk", ErrInvalidArgument)
	}
	if in.Entry <= 0 || math.IsNaN(in.Entry) || math.IsInf(in.Entry, 0) {
		return OrderPreviewSizing{}, fmt.Errorf("%w: entry price must be positive", ErrInvalidArgument)
	}
	if in.Stop <= 0 || math.IsNaN(in.Stop) || math.IsInf(in.Stop, 0) {
		return OrderPreviewSizing{}, fmt.Errorf("%w: stopLossPrice must be positive", ErrInvalidArgument)
	}
	if in.Long && in.Stop >= in.Entry {
		return OrderPreviewSizing{}, fmt.Errorf("%w: stop must be below the entry to size a buy", ErrInvalidArgument)
	}
	if !in.Long && in.Stop <= in.Entry {
		return OrderPreviewSizing{}, fmt.Errorf("%w: stop must be above the entry to size a short", ErrInvalidArgument)
	}

	entrySide, stopSide := TradeSideBuy, TradeSideSell
	if !in.Long {
		entrySide, stopSide = TradeSideSell, TradeSideBuy
	}
	entryFill := ApplySlippage(in.Entry, entrySide, in.Cost.SlippageRate)
	stopFill := ApplySlippage(in.Stop, stopSide, in.Cost.SlippageRate)
	lossPerUnit := stopLossPerUnit(in.Long, entryFill, stopFill, in.Cost.FeeRate)
	if lossPerUnit <= 1e-12 || math.IsNaN(lossPerUnit) || math.IsInf(lossPerUnit, 0) {
		return OrderPreviewSizing{}, fmt.Errorf("%w: stop is too close to the entry after fees and slippage", ErrInvalidArgument)
	}
	budget := in.Equity * in.RiskPercent / 100
	qty := math.Round((budget/lossPerUnit)*1e8) / 1e8
	if qty < MinTradeQuantity || qty > MaxTradeQuantity || math.IsNaN(qty) || math.IsInf(qty, 0) {
		return OrderPreviewSizing{}, fmt.Errorf("%w: sized quantity is out of range", ErrInvalidArgument)
	}
	return OrderPreviewSizing{
		RiskPercent:   in.RiskPercent,
		Equity:        in.Equity,
		RiskBudget:    budget,
		Quantity:      qty,
		EntryPrice:    in.Entry,
		StopPrice:     in.Stop,
		LossPerUnit:   lossPerUnit,
		EstimatedLoss: qty * lossPerUnit,
	}, nil
}

// stopLossPerUnit is the quote lost on one unit when the stop fills.
// Long matches a spot round trip and a margin long (effective entry minus effective exit).
// Short matches a margin short.
func stopLossPerUnit(long bool, entryFill, stopFill, feeRate float64) float64 {
	if long {
		return BuyCashDebit(1, entryFill, feeRate) - SellCashCredit(1, stopFill, feeRate)
	}
	return BuyCashDebit(1, stopFill, feeRate) - SellCashCredit(1, entryFill, feeRate)
}

// MarkStopVsLiquidation flags a margin stop that would not be the exit because
// liquidation is at the stop or closer to the entry.
func (z *OrderPreviewSizing) MarkStopVsLiquidation(long bool, liq float64) {
	if z == nil || liq <= 0 || math.IsNaN(liq) {
		return
	}
	if long {
		z.StopBeyondLiquidation = z.StopPrice <= liq+1e-12
		return
	}
	z.StopBeyondLiquidation = z.StopPrice >= liq-1e-12
}
