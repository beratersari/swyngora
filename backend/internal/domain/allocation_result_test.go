package domain

import (
	"strings"
	"testing"
	"time"
)

func TestBuildRebalanceResult_PartialAndLeftover(t *testing.T) {
	basket := AllocationBasket{ID: "b1", ClientID: "c1", Name: "Core"}
	before := RebalancePlan{
		Currency: "USDT", Equity: 10000, Cash: 1500,
		Lines: []AllocationLine{
			{Asset: "BTC", TargetPct: 50, ActualPct: 65, CurrentValue: 6500, TargetValue: 5000, DeltaValue: -1500},
			{Asset: "ETH", TargetPct: 30, ActualPct: 20, CurrentValue: 2000, TargetValue: 3000, DeltaValue: 1000},
			{Asset: "USDT", IsCash: true, TargetPct: 20, ActualPct: 15, CurrentValue: 1500, TargetValue: 2000, DeltaValue: 500},
		},
	}
	after := RebalancePlan{
		Currency: "USDT", Equity: 10000, Cash: 2200,
		Lines: []AllocationLine{
			{Asset: "BTC", TargetPct: 50, ActualPct: 52, CurrentValue: 5200, TargetValue: 5000, DeltaValue: -200},
			{Asset: "ETH", TargetPct: 30, ActualPct: 26, CurrentValue: 2600, TargetValue: 3000, DeltaValue: 400},
			{Asset: "USDT", IsCash: true, TargetPct: 20, ActualPct: 22, CurrentValue: 2200, TargetValue: 2000, DeltaValue: -200},
		},
	}
	legs := []RebalanceLegOutcome{
		{Side: TradeSideSell, Asset: "BTC", Status: RebalanceLegFilled, FilledQty: 13, FillPrice: 99.8, FillNotional: 1297.4, LastPrice: 100, Fee: 1.30},
		{Side: TradeSideBuy, Asset: "ETH", Status: RebalanceLegFailed, Error: "insufficient cash", PlannedNotional: 1000},
	}
	got := BuildRebalanceResult("r1", basket, before, after, legs, time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC))
	if got.Status != RebalanceResultPartial || got.FilledLegs != 1 || got.FailedLegs != 1 {
		t.Fatalf("%+v", got)
	}
	if got.RemainingAbsDriftPct < 7.9 || got.RemainingAbsDriftPct > 8.1 {
		t.Fatalf("abs drift %v", got.RemainingAbsDriftPct)
	}
	if got.MaxSleeveDriftPct < 3.9 || got.MaxSleeveDriftPct > 4.1 {
		t.Fatalf("max sleeve %v", got.MaxSleeveDriftPct)
	}
	if got.TotalFee < 1.29 || got.TotalFee > 1.31 || got.TotalSlippage < 2.59 || got.TotalSlippage > 2.61 {
		t.Fatalf("costs fee=%v slip=%v", got.TotalFee, got.TotalSlippage)
	}
	if !strings.Contains(got.Summary, "Filled 1 of 2") || !strings.Contains(got.Summary, "fees") {
		t.Fatalf("summary %q", got.Summary)
	}
	if len(got.Lines) != 3 {
		t.Fatalf("lines %d", len(got.Lines))
	}
}

func TestBuildRebalanceResult_NoTrades(t *testing.T) {
	got := BuildRebalanceResult("r2", AllocationBasket{ID: "b", ClientID: "c", Name: "Cash"},
		RebalancePlan{Currency: "USDT", Equity: 1000, Cash: 1000},
		RebalancePlan{Currency: "USDT", Equity: 1000, Cash: 1000},
		nil, time.Now().UTC())
	if got.Status != RebalanceResultNoTrades {
		t.Fatalf("%s", got.Status)
	}
}

func TestAdverseSlippageNotional(t *testing.T) {
	if got := AdverseSlippageNotional(TradeSideSell, 100, 99.8, 10); got < 1.99 || got > 2.01 {
		t.Fatalf("sell slip %v", got)
	}
	if got := AdverseSlippageNotional(TradeSideBuy, 50, 50.1, 4); got < 0.39 || got > 0.41 {
		t.Fatalf("buy slip %v", got)
	}
	if got := AdverseSlippageNotional(TradeSideSell, 100, 100.1, 1); got != 0 {
		t.Fatalf("favorable sell is not a cost: %v", got)
	}
}
