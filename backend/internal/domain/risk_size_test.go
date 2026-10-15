package domain

import (
	"math"
	"testing"
)

func TestSizeByStopRisk_LongUsesEquityAndStopDistance(t *testing.T) {
	got, err := SizeByStopRisk(RiskSizeInput{
		Equity: 10000, RiskPercent: 1, Entry: 100, Stop: 90, Long: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.RiskBudget-100) > 1e-9 || math.Abs(got.Quantity-10) > 1e-9 {
		t.Fatalf("budget=%v qty=%v", got.RiskBudget, got.Quantity)
	}
	if math.Abs(got.EstimatedLoss-100) > 1e-6 || math.Abs(got.LossPerUnit-10) > 1e-9 {
		t.Fatalf("loss=%v per unit=%v", got.EstimatedLoss, got.LossPerUnit)
	}
}

func TestSizeByStopRisk_LongLossMatchesMarginFill(t *testing.T) {
	cost := TradingCostFor(ExchangeBinance)
	got, err := SizeByStopRisk(RiskSizeInput{
		Equity: 10000, RiskPercent: 1, Entry: 100, Stop: 90, Long: true, Cost: cost,
	})
	if err != nil {
		t.Fatal(err)
	}
	entryFill := ApplySlippage(100, TradeSideBuy, cost.SlippageRate)
	stopFill := ApplySlippage(90, TradeSideSell, cost.SlippageRate)
	entry := EffectiveMarginEntry(MarginLong, entryFill, cost.FeeRate)
	exit := EffectiveMarginExit(MarginLong, stopFill, cost.FeeRate)
	loss := -MarginRealizedPnL(MarginLong, got.Quantity, entry, exit)
	if math.Abs(loss-got.EstimatedLoss) > 1e-6 {
		t.Fatalf("margin loss=%v sizing=%v", loss, got.EstimatedLoss)
	}
	if math.Abs(got.EstimatedLoss-got.RiskBudget) > got.LossPerUnit*1e-8+1e-6 {
		t.Fatalf("loss=%v budget=%v", got.EstimatedLoss, got.RiskBudget)
	}
	if got.Quantity >= 10 {
		t.Fatalf("fees should shrink the size, qty=%v", got.Quantity)
	}
}

func TestSizeByStopRisk_ShortLossMatchesMarginFill(t *testing.T) {
	cost := TradingCostFor(ExchangeBinance)
	got, err := SizeByStopRisk(RiskSizeInput{
		Equity: 10000, RiskPercent: 2, Entry: 100, Stop: 110, Long: false, Cost: cost,
	})
	if err != nil {
		t.Fatal(err)
	}
	entryFill := ApplySlippage(100, TradeSideSell, cost.SlippageRate)
	stopFill := ApplySlippage(110, TradeSideBuy, cost.SlippageRate)
	entry := EffectiveMarginEntry(MarginShort, entryFill, cost.FeeRate)
	exit := EffectiveMarginExit(MarginShort, stopFill, cost.FeeRate)
	loss := -MarginRealizedPnL(MarginShort, got.Quantity, entry, exit)
	if math.Abs(loss-got.EstimatedLoss) > 1e-6 {
		t.Fatalf("margin loss=%v sizing=%v", loss, got.EstimatedLoss)
	}
	if math.Abs(got.RiskBudget-200) > 1e-9 {
		t.Fatalf("budget=%v", got.RiskBudget)
	}
}

func TestSizeByStopRisk_RejectsWrongStopAndBadInputs(t *testing.T) {
	cases := []RiskSizeInput{
		{Equity: 10000, RiskPercent: 1, Entry: 100, Stop: 110, Long: true},
		{Equity: 10000, RiskPercent: 1, Entry: 100, Stop: 90, Long: false},
		{Equity: 0, RiskPercent: 1, Entry: 100, Stop: 90, Long: true},
		{Equity: 10000, RiskPercent: 0, Entry: 100, Stop: 90, Long: true},
		{Equity: 10000, RiskPercent: 101, Entry: 100, Stop: 90, Long: true},
		{Equity: 10000, RiskPercent: math.NaN(), Entry: 100, Stop: 90, Long: true},
	}
	for _, in := range cases {
		if _, err := SizeByStopRisk(in); err == nil {
			t.Fatalf("accepted %+v", in)
		}
	}
}

func TestSizeByStopRisk_RejectsQuantityOutsideRange(t *testing.T) {
	_, err := SizeByStopRisk(RiskSizeInput{
		Equity: 10000, RiskPercent: 100, Entry: 100, Stop: 100 - 1e-6, Long: true,
	})
	if err == nil {
		t.Fatal("expected the size to exceed the maximum")
	}
	_, err = SizeByStopRisk(RiskSizeInput{
		Equity: 1, RiskPercent: 0.00001, Entry: 100, Stop: 50, Long: true,
	})
	if err == nil {
		t.Fatal("expected the size to fall under the minimum")
	}
}

func TestMarkStopVsLiquidation(t *testing.T) {
	z := &OrderPreviewSizing{StopPrice: 80}
	z.MarkStopVsLiquidation(true, 90)
	if !z.StopBeyondLiquidation {
		t.Fatal("long stop below liquidation should be flagged")
	}
	z.MarkStopVsLiquidation(true, 70)
	if z.StopBeyondLiquidation {
		t.Fatal("long stop above liquidation is the exit")
	}
	z.StopPrice = 120
	z.MarkStopVsLiquidation(false, 110)
	if !z.StopBeyondLiquidation {
		t.Fatal("short stop above liquidation should be flagged")
	}
}
