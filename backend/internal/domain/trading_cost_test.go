package domain

import (
	"math"
	"testing"
)

func TestTradingCostFor_PerExchange(t *testing.T) {
	b := TradingCostFor(ExchangeBinance)
	c := TradingCostFor(ExchangeCoinbase)
	y := TradingCostFor(ExchangeBybit)
	if b.FeeRate != BinanceFeeRate || b.SlippageRate != BinanceSlippageRate {
		t.Fatalf("binance %+v", b)
	}
	if c.FeeRate != CoinbaseFeeRate || c.FeeRate <= b.FeeRate {
		t.Fatalf("coinbase should be richer fee %+v", c)
	}
	if y.FeeRate != BybitFeeRate {
		t.Fatalf("bybit %+v", y)
	}
}

func TestApplySlippageAndCash(t *testing.T) {
	buy := ApplySlippage(100, TradeSideBuy, 0.001)
	sell := ApplySlippage(100, TradeSideSell, 0.001)
	if math.Abs(buy-100.1) > 1e-9 || math.Abs(sell-99.9) > 1e-9 {
		t.Fatalf("buy=%v sell=%v", buy, sell)
	}
	if math.Abs(BuyCashDebit(2, 100, 0.001)-200.2) > 1e-9 {
		t.Fatalf("buy debit=%v", BuyCashDebit(2, 100, 0.001))
	}
	if math.Abs(SellCashCredit(2, 100, 0.001)-199.8) > 1e-9 {
		t.Fatalf("sell credit=%v", SellCashCredit(2, 100, 0.001))
	}
	if math.Abs(BuyUnitCost(100, 0.001)-100.1) > 1e-12 {
		t.Fatalf("unit=%v", BuyUnitCost(100, 0.001))
	}
	cost := TradingCost{FeeRate: 0.001, SlippageRate: 0.0005}
	// reserve 1 @ 100 → 100 * 1.0005 * 1.001
	want := 100 * 1.0005 * 1.001
	if math.Abs(BuyReserveCash(1, 100, cost)-want) > 1e-9 {
		t.Fatalf("reserve=%v want=%v", BuyReserveCash(1, 100, cost), want)
	}
	if math.Abs(BuyReserveCash(1, 100, TradingCost{})-100) > 1e-12 {
		t.Fatalf("zero cost reserve should be qty*trigger")
	}
	maxQ := MaxBuyFillQty(2, 100.1, 100, 0.001)
	if math.Abs(maxQ-1) > 1e-9 {
		t.Fatalf("max fill=%v", maxQ)
	}
}

func TestZeroTradingCosts(t *testing.T) {
	z := ZeroTradingCosts(ExchangeCoinbase)
	if z.FeeRate != 0 || z.SlippageRate != 0 {
		t.Fatalf("%+v", z)
	}
}

func TestEffectiveMarginAndRawFillRoundTrip(t *testing.T) {
	fill := 100.0
	fee := 0.001
	longE := EffectiveMarginEntry(MarginLong, fill, fee)
	shortE := EffectiveMarginEntry(MarginShort, fill, fee)
	if math.Abs(longE-100.1) > 1e-12 || math.Abs(shortE-99.9) > 1e-12 {
		t.Fatalf("entry long=%v short=%v", longE, shortE)
	}
	if math.Abs(RawFillFromEffectiveEntry(MarginLong, longE, fee)-fill) > 1e-12 {
		t.Fatalf("long raw=%v", RawFillFromEffectiveEntry(MarginLong, longE, fee))
	}
	if math.Abs(RawFillFromEffectiveEntry(MarginShort, shortE, fee)-fill) > 1e-12 {
		t.Fatalf("short raw=%v", RawFillFromEffectiveEntry(MarginShort, shortE, fee))
	}
	if math.Abs(RawFillFromEffectiveEntry(MarginLong, fill, 0)-fill) > 1e-12 {
		t.Fatal("zero fee should be identity")
	}
	longX := EffectiveMarginExit(MarginLong, 110, fee)
	shortX := EffectiveMarginExit(MarginShort, 110, fee)
	if math.Abs(longX-109.89) > 1e-12 || math.Abs(shortX-110.11) > 1e-12 {
		t.Fatalf("exit long=%v short=%v", longX, shortX)
	}
}

func TestAllPaperTradingCosts(t *testing.T) {
	all := AllPaperTradingCosts()
	if len(all) != len(SupportedExchanges) {
		t.Fatalf("len=%d", len(all))
	}
	if all[0].Exchange != ExchangeBinance || all[0].FeePct != BinanceFeeRate*100 {
		t.Fatalf("%+v", all[0])
	}
	if all[1].Exchange != ExchangeCoinbase || all[1].FeeRate != CoinbaseFeeRate {
		t.Fatalf("%+v", all[1])
	}
}
