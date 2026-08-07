package domain

import (
	"errors"
	"math"
	"testing"
)

func TestValidateAllocationTargets(t *testing.T) {
	err := ValidateAllocationTargets([]AllocationTarget{
		{Asset: "BTC", WeightPct: 50}, {Asset: "ETH", WeightPct: 30}, {Asset: "USDT", WeightPct: 20},
	}, "USDT")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAllocationTargets([]AllocationTarget{
		{Asset: "BTC", WeightPct: 50}, {Asset: "ETH", WeightPct: 30},
	}, "USDT"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("sum: %v", err)
	}
	if err := ValidateAllocationTargets([]AllocationTarget{
		{Asset: "BTC", WeightPct: 50}, {Asset: "btc", WeightPct: 50},
	}, "USDT"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("dup: %v", err)
	}
}

func TestPlanRebalance_SellsOverweightThenBuys(t *testing.T) {
	// Equity 10_000: BTC 6500 (65%), ETH 2000 (20%), cash 1500 (15%). Target 50/30/20.
	holdings := []AllocationHolding{
		{Asset: "BTC", Exchange: ExchangeBinance, Symbol: "BTCUSDT", MarkPrice: 100, Quantity: 65, AvailableQty: 65, MarketValue: 6500},
		{Asset: "ETH", Exchange: ExchangeBinance, Symbol: "ETHUSDT", MarkPrice: 50, Quantity: 40, AvailableQty: 40, MarketValue: 2000},
	}
	targets := []AllocationTarget{
		{Asset: "BTC", Exchange: ExchangeBinance, WeightPct: 50},
		{Asset: "ETH", Exchange: ExchangeBinance, WeightPct: 30},
		{Asset: "USDT", WeightPct: 20},
	}
	plan, err := PlanRebalance("USDT", 10000, 1500, 1500, holdings, targets)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Legs) < 2 {
		t.Fatalf("legs=%+v", plan.Legs)
	}
	if plan.Legs[0].Side != TradeSideSell || plan.Legs[0].Asset != "BTC" {
		t.Fatalf("first sell BTC: %+v", plan.Legs[0])
	}
	if math.Abs(plan.Legs[0].Notional-1500) > 1 {
		t.Fatalf("sell ~1500 got %v", plan.Legs[0].Notional)
	}
	var boughtETH bool
	for _, l := range plan.Legs {
		if l.Side == TradeSideBuy && l.Asset == "ETH" {
			boughtETH = true
			if math.Abs(l.Notional-1000) > 2 {
				t.Fatalf("buy ETH ~1000 got %v", l.Notional)
			}
		}
	}
	if !boughtETH {
		t.Fatalf("expected ETH buy: %+v", plan.Legs)
	}
	// Drift lines exist even without executing
	var btc AllocationLine
	for _, ln := range plan.Lines {
		if ln.Asset == "BTC" {
			btc = ln
		}
	}
	if math.Abs(btc.ActualPct-65) > 0.1 || math.Abs(btc.TargetPct-50) > 1e-9 {
		t.Fatalf("btc line %+v", btc)
	}
}

func TestPlanRebalance_NotInBasketSold(t *testing.T) {
	holdings := []AllocationHolding{
		{Asset: "SOL", Exchange: ExchangeBinance, Symbol: "SOLUSDT", MarkPrice: 10, Quantity: 100, AvailableQty: 100, MarketValue: 1000},
	}
	targets := []AllocationTarget{{Asset: "USDT", WeightPct: 100}}
	plan, err := PlanRebalance("USDT", 2000, 1000, 1000, holdings, targets)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Legs) != 1 || plan.Legs[0].Reason != "not_in_basket" || plan.Legs[0].Asset != "SOL" {
		t.Fatalf("%+v", plan.Legs)
	}
}

func TestSplitBaseQuoteAndPair(t *testing.T) {
	b, q := SplitBaseQuote(ExchangeBinance, "btcusdt")
	if b != "BTC" || q != "USDT" {
		t.Fatalf("%s %s", b, q)
	}
	if p := PairSymbol(ExchangeBinance, "ETH", "USDT"); p != "ETHUSDT" {
		t.Fatalf("%s", p)
	}
	if p := PairSymbol(ExchangeCoinbase, "BTC", "USD"); p != "BTC-USD" {
		t.Fatalf("%s", p)
	}
}
