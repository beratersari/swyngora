package domain

import (
	"math"
	"testing"
	"time"
)

func TestFirstTouch_LongTP(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	bars := []OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(4 * time.Hour), Open: 100.5, High: 102, Low: 100, Close: 101},
		{OpenTime: t0.Add(8 * time.Hour), Open: 101, High: 106, Low: 100.8, Close: 105},
	}
	got := SimulateBacktestTrade(bars, 0, BacktestTradeSideLong, SwingLevels{
		Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 2, RiskPct: 3,
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP {
		t.Fatalf("%+v", got)
	}
	if got.ExitPrice != 105 || got.BarsHeld != 2 || got.Leverage != 2 {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.PnLPct-10) > 0.05 {
		t.Fatalf("pnl %+v", got)
	}
}

func TestFirstTouch_LongSLAndSameBarConservative(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	slFirst := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 100, High: 100.5, Low: 96, Close: 97},
	}, 0, "LONG", SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1})
	if slFirst == nil || slFirst.Outcome != BacktestTradeOutcomeSL || slFirst.PnLPct >= 0 {
		t.Fatalf("%+v", slFirst)
	}
	both := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 100, High: 106, Low: 96, Close: 101},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1})
	if both == nil || both.Outcome != BacktestTradeOutcomeSL {
		t.Fatalf("same-bar must take SL: %+v", both)
	}
}

func TestFirstTouch_GapThroughTP(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 107, High: 108, Low: 106, Close: 107},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP || got.ExitPrice != 105 {
		t.Fatalf("%+v", got)
	}
}

func TestFirstTouch_ShortTPAndSL(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	tp := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 99, High: 99.5, Low: 94, Close: 95},
	}, 0, BacktestTradeSideShort, SwingLevels{Entry: 100, StopLoss: 103, TakeProfit: 95, Leverage: 3})
	if tp == nil || tp.Outcome != BacktestTradeOutcomeTP {
		t.Fatalf("tp %+v", tp)
	}
	if math.Abs(tp.PnLPct-15) > 0.05 {
		t.Fatalf("short tp pnl %+v", tp)
	}
	sl := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 101, High: 104, Low: 100.5, Close: 103},
	}, 0, BacktestTradeSideShort, SwingLevels{Entry: 100, StopLoss: 103, TakeProfit: 95, Leverage: 1})
	if sl == nil || sl.Outcome != BacktestTradeOutcomeSL || sl.PnLPct >= 0 {
		t.Fatalf("sl %+v", sl)
	}
}

func TestFirstTouch_StillOpenMarksLastClose(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 100.2, High: 101, Low: 99.5, Close: 100.5},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 90, TakeProfit: 120, Leverage: 1})
	if got == nil || got.Outcome != BacktestTradeOutcomeOpen || got.ExitPrice != 100.5 {
		t.Fatalf("%+v", got)
	}
}

func TestSuggestedSwingLeverage(t *testing.T) {
	t.Parallel()
	if SuggestedSwingLeverage(2.5) != 4 {
		t.Fatalf("2.5 → %d", SuggestedSwingLeverage(2.5))
	}
	if SuggestedSwingLeverage(0.5) != BacktestMaxLeverage {
		t.Fatalf("cap %d", SuggestedSwingLeverage(0.5))
	}
	if SuggestedSwingLeverage(12) != 1 {
		t.Fatalf("wide stop %d", SuggestedSwingLeverage(12))
	}
}

func TestScannerRuleTradeSide(t *testing.T) {
	t.Parallel()
	if ScannerRuleTradeSide(ScannerRule{Type: ScannerRuleRSI, RSICondition: AlertBelow}) != BacktestTradeSideShort {
		t.Fatal("rsi below")
	}
	if ScannerRuleTradeSide(ScannerRule{Type: ScannerRuleMACrossover, MADirection: "death_cross"}) != BacktestTradeSideShort {
		t.Fatal("death")
	}
	if ScannerRuleTradeSide(ScannerRule{Type: ScannerRuleVolumeIncrease}) != BacktestTradeSideLong {
		t.Fatal("volume")
	}
}

func TestSummarizeBacktestReturns_Trades(t *testing.T) {
	t.Parallel()
	ptr := func(v float64) *float64 { return &v }
	got := SummarizeBacktestReturns([]ScannerBacktestSignal{
		{Return1d: ptr(2), Trade: &BacktestTrade{Side: "long", Outcome: BacktestTradeOutcomeTP, PnLPct: 8}},
		{Return1d: ptr(-1), Trade: &BacktestTrade{Side: "short", Outcome: BacktestTradeOutcomeSL, PnLPct: -4}},
		{Trade: &BacktestTrade{Side: "long", Outcome: BacktestTradeOutcomeOpen, PnLPct: 1}},
	})
	if got.Trades.SampleCount != 3 || got.Trades.TpHits != 1 || got.Trades.SlHits != 1 || got.Trades.OpenCount != 1 {
		t.Fatalf("%+v", got.Trades)
	}
	if got.Trades.LongCount != 2 || got.Trades.ShortCount != 1 {
		t.Fatalf("sides %+v", got.Trades)
	}
	if got.Trades.WinRate == nil || *got.Trades.WinRate != 50 {
		t.Fatalf("win %+v", got.Trades)
	}
	if got.Trades.Average == nil || *got.Trades.Average != 2 {
		t.Fatalf("avg %+v", got.Trades)
	}
}

func TestPlanSwingLevelsForSide_Short(t *testing.T) {
	t.Parallel()
	bars := synthTrend(40, 100, 0.3)
	atr, ok := ATR(bars, 14)
	if !ok {
		t.Fatal("atr")
	}
	lv, reasons := planSwingLevelsForSide(bars, atr, bars[len(bars)-1].Close, BacktestTradeSideShort)
	if lv == nil || len(reasons) > 0 {
		t.Fatalf("%+v %v", lv, reasons)
	}
	if lv.TakeProfit >= lv.Entry || lv.StopLoss <= lv.Entry {
		t.Fatalf("short levels %+v", lv)
	}
	if lv.Leverage < 1 {
		t.Fatalf("leverage %+v", lv)
	}
}
