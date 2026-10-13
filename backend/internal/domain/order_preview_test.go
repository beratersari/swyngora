package domain

import "testing"

func TestQuoteFill_BuyIncludesSlipAndFee(t *testing.T) {
	cost := TradingCost{FeeRate: 0.001, SlippageRate: 0.0005}
	fill, notional, fee, debit, credit := QuoteFill(TradeSideBuy, 100, 2, cost)
	if fill < 100.04 || fill > 100.06 {
		t.Fatalf("fill=%v", fill)
	}
	if notional < 200.09 || notional > 200.11 {
		t.Fatalf("notional=%v", notional)
	}
	if fee < 0.20 || fee > 0.201 {
		t.Fatalf("fee=%v", fee)
	}
	if debit < notional+fee-0.001 || debit > notional+fee+0.001 || credit != 0 {
		t.Fatalf("debit=%v credit=%v fee=%v", debit, credit, fee)
	}
}

func TestQuoteFill_SellReceivesLess(t *testing.T) {
	cost := TradingCost{FeeRate: 0.001, SlippageRate: 0.0005}
	fill, _, fee, debit, credit := QuoteFill(TradeSideSell, 100, 1, cost)
	if fill > 99.96 || fill < 99.94 {
		t.Fatalf("fill=%v", fill)
	}
	if debit != 0 || credit <= 0 || fee <= 0 {
		t.Fatalf("fee=%v debit=%v credit=%v", fee, debit, credit)
	}
	if credit > fill {
		t.Fatalf("credit %v should be under fill %v", credit, fill)
	}
}

func TestBuildOrderPreviewRisk_BuyOverWeightIsBlocked(t *testing.T) {
	weight := 30.0
	limits := RiskLimits{MaxAssetWeightPct: &weight}
	got := BuildOrderPreviewRisk(limits, 10000, 10000, 10000, "BTC", 2000, 2000, 2000, true)
	if !got.Blocked || !got.WeightLimitHit {
		t.Fatalf("want blocked weight: %+v", got)
	}
	if got.ProjectedWeightPct == nil || *got.ProjectedWeightPct < 39 || *got.ProjectedWeightPct > 41 {
		t.Fatalf("weight=%v", got.ProjectedWeightPct)
	}
}

func TestBuildOrderPreviewRisk_SellDoesNotBlock(t *testing.T) {
	weight := 30.0
	limits := RiskLimits{MaxAssetWeightPct: &weight}
	got := BuildOrderPreviewRisk(limits, 10000, 10000, 10000, "BTC", 5000, 5000, 0, false)
	if got.Blocked || got.Applies {
		t.Fatalf("sell must not block: %+v", got)
	}
}
