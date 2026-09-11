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
	if got.ExitPrice != 105 || got.Leverage != 2 {
		t.Fatalf("%+v", got)
	}
	if got.EnteredAt == nil || !got.EnteredAt.Equal(t0.Add(4*time.Hour)) {
		t.Fatalf("enter next open %+v", got.EnteredAt)
	}
	if math.Abs(got.Entry-100.5) > 1e-9 {
		t.Fatalf("fill next open %+v", got)
	}
	if got.BarsHeld != 1 {
		t.Fatalf("held from entry bar %+v", got)
	}
	wantPnL := 2 * (105/100.5 - 1) * 100
	if math.Abs(got.PnLPct-wantPnL) > 0.05 {
		t.Fatalf("pnl %+v want %v", got, wantPnL)
	}
}

func TestFirstTouch_EntersNextOpenNotSignalOpen(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 90, High: 110, Low: 89, Close: 100},
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 101, High: 102, Low: 100.5, Close: 101.5},
		{OpenTime: t0.Add(8 * time.Hour), CloseTime: t0.Add(12 * time.Hour), Open: 101.5, High: 106, Low: 101, Close: 105},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.Entry-101) > 1e-9 {
		t.Fatalf("must fill next open, not signal open/close: %+v", got)
	}
	if got.EnteredAt == nil || !got.EnteredAt.Equal(t0.Add(4*time.Hour)) {
		t.Fatalf("enteredAt %+v", got.EnteredAt)
	}
}

func TestFirstTouch_MissingNextOpenHasNoTrade(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(4 * time.Hour), Open: 0, High: 106, Low: 99, Close: 104},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1})
	if got != nil {
		t.Fatalf("missing next open must not invent Outcome=open at plan entry: %+v", got)
	}
}

func TestFirstTouch_LastBarHasNoTrade(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1})
	if got != nil {
		t.Fatalf("no next bar must not invent Outcome=open at plan entry: %+v", got)
	}
}

func TestOHLCFromCandles_DoesNotInventOpen(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := OHLCFromCandles([]Candle{
		{OpenTime: t0, Open: "", High: "101", Low: "99", Close: "100"},
		{OpenTime: t0.Add(time.Hour), Open: "0", High: "102", Low: "98", Close: "101"},
	})
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Open != 0 || got[0].Close != 100 || got[0].High != 101 || got[0].Low != 99 {
		t.Fatalf("keep close/high/low, leave open empty: %+v", got[0])
	}
	if got[1].Open != 0 || got[1].Close != 101 {
		t.Fatalf("zero open stays missing: %+v", got[1])
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
		t.Fatalf("same-bar without finer bars must take SL: %+v", both)
	}
}

func TestFirstTouch_SameBarFinerTakesTP(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	parent := OHLC{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 106, Low: 96, Close: 101}
	finer := []OHLC{
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(5 * time.Hour), Open: 100, High: 106, Low: 99.5, Close: 105},
		{OpenTime: t0.Add(5 * time.Hour), CloseTime: t0.Add(6 * time.Hour), Open: 105, High: 105.5, Low: 104, Close: 104.5},
		{OpenTime: t0.Add(6 * time.Hour), CloseTime: t0.Add(7 * time.Hour), Open: 104.5, High: 105, Low: 97, Close: 98},
		{OpenTime: t0.Add(7 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 98, High: 101, Low: 96, Close: 101},
	}
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		parent,
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 2}, BacktestTouchOpts{
		Interval: Interval4h,
		Finer:    func(OHLC, CandleInterval) []OHLC { return finer },
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP || got.ExitPrice != 105 {
		t.Fatalf("finer TP first: %+v", got)
	}
	if math.Abs(got.PnLPct-10) > 0.05 {
		t.Fatalf("tp pnl %+v", got)
	}
	if got.HitAt != nil || !got.HitTimeUnknown {
		t.Fatalf("wick is not an exact time: %+v", got)
	}
	if got.ClosedKnownAt == nil || !got.ClosedKnownAt.Equal(t0.Add(5*time.Hour)) {
		t.Fatalf("hold risk until the 1h close %+v", got.ClosedKnownAt)
	}
}

func TestFirstTouch_SameBarFinerTakesSL(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	parent := OHLC{OpenTime: t0.Add(time.Hour), CloseTime: t0.Add(2 * time.Hour), Open: 100, High: 106, Low: 96, Close: 101}
	finer := []OHLC{
		{OpenTime: t0.Add(60 * time.Minute), CloseTime: t0.Add(75 * time.Minute), Open: 100, High: 101, Low: 96, Close: 97},
		{OpenTime: t0.Add(75 * time.Minute), CloseTime: t0.Add(90 * time.Minute), Open: 97, High: 106, Low: 97, Close: 105},
		{OpenTime: t0.Add(90 * time.Minute), CloseTime: t0.Add(105 * time.Minute), Open: 105, High: 105.2, Low: 104, Close: 104.5},
		{OpenTime: t0.Add(105 * time.Minute), CloseTime: t0.Add(120 * time.Minute), Open: 104.5, High: 105, Low: 100, Close: 101},
	}
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		parent,
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1}, BacktestTouchOpts{
		Interval: Interval1h,
		Finer:    func(OHLC, CandleInterval) []OHLC { return finer },
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeSL || got.ExitPrice != 97 || got.PnLPct >= 0 {
		t.Fatalf("finer SL first: %+v", got)
	}
}

func TestFirstTouch_SameBarFinerRecursesThenTP(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	parent := OHLC{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 106, Low: 96, Close: 101}
	hour := []OHLC{
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(5 * time.Hour), Open: 100, High: 106, Low: 96, Close: 101},
		{OpenTime: t0.Add(5 * time.Hour), CloseTime: t0.Add(6 * time.Hour), Open: 101, High: 102, Low: 100, Close: 101},
		{OpenTime: t0.Add(6 * time.Hour), CloseTime: t0.Add(7 * time.Hour), Open: 101, High: 102, Low: 100, Close: 101},
		{OpenTime: t0.Add(7 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 101, High: 102, Low: 100, Close: 101},
	}
	m15 := []OHLC{
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(4*time.Hour + 15*time.Minute), Open: 100, High: 104, Low: 99.8, Close: 103},
		{OpenTime: t0.Add(4*time.Hour + 15*time.Minute), CloseTime: t0.Add(4*time.Hour + 30*time.Minute), Open: 103, High: 106, Low: 102, Close: 105},
		{OpenTime: t0.Add(4*time.Hour + 30*time.Minute), CloseTime: t0.Add(4*time.Hour + 45*time.Minute), Open: 105, High: 105.5, Low: 96, Close: 97},
		{OpenTime: t0.Add(4*time.Hour + 45*time.Minute), CloseTime: t0.Add(5 * time.Hour), Open: 97, High: 101, Low: 96.5, Close: 101},
	}
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		parent,
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1}, BacktestTouchOpts{
		Interval: Interval4h,
		Finer: func(_ OHLC, iv CandleInterval) []OHLC {
			if iv == Interval1h {
				return hour
			}
			return m15
		},
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP || got.ExitPrice != 105 {
		t.Fatalf("1h still both, 15m TP first: %+v", got)
	}
	if math.Abs(got.PnLPct-5) > 0.05 {
		t.Fatalf("pnl %+v", got)
	}
}

func TestFirstTouch_SameBarIncompleteFinerKeepsSL(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	parent := OHLC{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 106, Low: 96, Close: 101}
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		parent,
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1}, BacktestTouchOpts{
		Interval: Interval4h,
		Finer: func(OHLC, CandleInterval) []OHLC {
			return []OHLC{
				{OpenTime: t0.Add(6 * time.Hour), Open: 100, High: 106, Low: 99, Close: 105},
			}
		},
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeSL {
		t.Fatalf("incomplete finer must stay conservative: %+v", got)
	}
}

func TestFirstTouch_ShortSameBarFinerTakesTP(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	parent := OHLC{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 104, Low: 94, Close: 99}
	finer := []OHLC{
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(5 * time.Hour), Open: 100, High: 101, Low: 94, Close: 95},
		{OpenTime: t0.Add(5 * time.Hour), CloseTime: t0.Add(6 * time.Hour), Open: 95, High: 104, Low: 95, Close: 103},
		{OpenTime: t0.Add(6 * time.Hour), CloseTime: t0.Add(7 * time.Hour), Open: 103, High: 103.5, Low: 102, Close: 102.5},
		{OpenTime: t0.Add(7 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 102.5, High: 103, Low: 99, Close: 99},
	}
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		parent,
	}, 0, BacktestTradeSideShort, SwingLevels{Entry: 100, StopLoss: 103, TakeProfit: 95, Leverage: 3}, BacktestTouchOpts{
		Interval: Interval4h,
		Finer:    func(OHLC, CandleInterval) []OHLC { return finer },
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP || got.ExitPrice != 95 {
		t.Fatalf("short finer TP: %+v", got)
	}
	if math.Abs(got.PnLPct-15) > 0.05 {
		t.Fatalf("short tp pnl %+v", got)
	}
}

func TestFinerBacktestInterval(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   CandleInterval
		want CandleInterval
		ok   bool
	}{
		{Interval4h, Interval1h, true},
		{Interval1d, Interval1h, true},
		{Interval1w, Interval1d, true},
		{Interval1h, Interval15m, true},
		{Interval15m, Interval5m, true},
		{Interval5m, Interval1m, true},
		{Interval1m, "", false},
	}
	for _, tc := range cases {
		got, ok := FinerBacktestInterval(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("%s → %s %v, want %s %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestFirstTouch_RangeMarkBeforeLaterTP(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := t0.Add(6 * time.Hour)
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 101, Low: 99.5, Close: 100.5},
		{OpenTime: t0.Add(8 * time.Hour), CloseTime: t0.Add(12 * time.Hour), Open: 100.5, High: 101, Low: 100, Close: 100.8},
		{OpenTime: t0.Add(12 * time.Hour), CloseTime: t0.Add(16 * time.Hour), Open: 101, High: 106, Low: 100.8, Close: 105},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1}, BacktestTouchOpts{
		RangeEnd: rangeEnd,
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP || !got.ClosedAfterRange {
		t.Fatalf("later TP %+v", got)
	}
	if math.Abs(got.RangeExitPrice-100) > 1e-9 {
		t.Fatalf("mid-candle must not use later close 100.5: %+v", got)
	}
	if math.Abs(got.RangePnLPct) > 0.05 {
		t.Fatalf("range pnl %+v", got)
	}
}

func TestFirstTouch_MidCandleUsesFinerPrice(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := t0.Add(6 * time.Hour)
	finer := []OHLC{
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(5 * time.Hour), Open: 100, High: 100.2, Low: 99.8, Close: 100.1},
		{OpenTime: t0.Add(5 * time.Hour), CloseTime: t0.Add(6 * time.Hour), Open: 100.1, High: 100.4, Low: 100, Close: 100.3},
		{OpenTime: t0.Add(6 * time.Hour), CloseTime: t0.Add(7 * time.Hour), Open: 100.3, High: 101, Low: 100.2, Close: 100.8},
	}
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 102, Low: 99.5, Close: 101.5},
		{OpenTime: t0.Add(8 * time.Hour), CloseTime: t0.Add(12 * time.Hour), Open: 101.5, High: 102, Low: 101, Close: 101.6},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 90, TakeProfit: 130, Leverage: 1}, BacktestTouchOpts{
		Interval: Interval4h,
		RangeEnd: rangeEnd,
		Finer:    func(OHLC, CandleInterval) []OHLC { return finer },
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeOpen {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.RangeExitPrice-100.3) > 1e-9 {
		t.Fatalf("want 1h close at range end, not 4h close 101.5: %+v", got)
	}
}

func TestFirstTouch_GapThroughTP(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 107, High: 108, Low: 106, Close: 107},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.Entry-107) > 1e-9 || got.ExitPrice != 107 || math.Abs(got.PnLPct) > 0.05 {
		t.Fatalf("already through TP at next open: %+v", got)
	}
}

func TestFirstTouch_GapThroughStopFillsOpen(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	long := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 94, High: 95, Low: 93, Close: 94.5},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 2})
	if long == nil || long.Outcome != BacktestTradeOutcomeSL || long.ExitPrice != 94 {
		t.Fatalf("long gap SL %+v", long)
	}
	if math.Abs(long.Entry-94) > 1e-9 || math.Abs(long.PnLPct) > 0.05 {
		t.Fatalf("must not PnL from the old signal close: %+v", long)
	}
	short := SimulateBacktestTrade([]OHLC{
		{OpenTime: t0, Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(time.Hour), Open: 106, High: 107, Low: 105, Close: 106.5},
	}, 0, BacktestTradeSideShort, SwingLevels{Entry: 100, StopLoss: 103, TakeProfit: 95, Leverage: 1})
	if short == nil || short.Outcome != BacktestTradeOutcomeSL || short.ExitPrice != 106 {
		t.Fatalf("short gap SL %+v", short)
	}
	if math.Abs(short.Entry-106) > 1e-9 || math.Abs(short.PnLPct) > 0.05 {
		t.Fatalf("must not PnL from the old signal close: %+v", short)
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
	wantShort := 3.0 * (99.0 - 95.0) / 99.0 * 100
	if math.Abs(tp.PnLPct-wantShort) > 0.05 {
		t.Fatalf("short tp pnl %+v want %v", tp, wantShort)
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

func TestFirstTouch_StillOpenIgnoresFutureTape(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 101, Low: 99.5, Close: 100.4},
		{OpenTime: t0.Add(24 * time.Hour), CloseTime: t0.Add(28 * time.Hour), Open: 110, High: 112, Low: 109, Close: 111},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 90, TakeProfit: 130, Leverage: 1}, BacktestTouchOpts{
		RangeEnd: t0.Add(8 * time.Hour),
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeOpen || got.ClosedAfterRange {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.ExitPrice-100.4) > 1e-9 || math.Abs(got.RangeExitPrice-100.4) > 1e-9 {
		t.Fatalf("must mark last in-range close, not future 111: %+v", got)
	}
	if got.HitAt != nil {
		t.Fatalf("open trade must not invent a hit: %+v", got.HitAt)
	}
}

func TestFirstTouch_HitTimeFromFinerNotParentOpen(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	parent := OHLC{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 106, Low: 99, Close: 104}
	finer := []OHLC{
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(5 * time.Hour), Open: 100, High: 101, Low: 99.5, Close: 100.5},
		{OpenTime: t0.Add(5 * time.Hour), CloseTime: t0.Add(6 * time.Hour), Open: 100.5, High: 102, Low: 100, Close: 101},
		{OpenTime: t0.Add(6 * time.Hour), CloseTime: t0.Add(7 * time.Hour), Open: 101, High: 106, Low: 100.8, Close: 105},
		{OpenTime: t0.Add(7 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 105, High: 105.5, Low: 104, Close: 104},
	}
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		parent,
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1}, BacktestTouchOpts{
		Interval: Interval4h,
		Finer:    func(OHLC, CandleInterval) []OHLC { return finer },
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP {
		t.Fatalf("%+v", got)
	}
	if got.HitAt != nil || !got.HitTimeUnknown {
		t.Fatalf("1h wick is not exact: %+v", got)
	}
	if got.ClosedKnownAt == nil || !got.ClosedKnownAt.Equal(t0.Add(7*time.Hour)) {
		t.Fatalf("known-by is that 1h close, not 4h open: %+v", got.ClosedKnownAt)
	}
}

func TestFirstTouch_GapOpenIsExactHitTime(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 106, High: 107, Low: 105, Close: 106},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1}, BacktestTouchOpts{
		Interval: Interval4h,
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP || got.HitTimeUnknown {
		t.Fatalf("gap through TP is exact: %+v", got)
	}
	if math.Abs(got.Entry-106) > 1e-9 || got.ExitPrice != 106 || math.Abs(got.PnLPct) > 0.05 {
		t.Fatalf("fill at the real open, not the old entry: %+v", got)
	}
	if got.HitAt == nil || !got.HitAt.Equal(t0.Add(4*time.Hour)) {
		t.Fatalf("hitAt %+v", got.HitAt)
	}
}

func TestFirstTouch_UnknownTimeWithoutFiner(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateBacktestTradeOpts([]OHLC{
		{OpenTime: t0, CloseTime: t0.Add(4 * time.Hour), Open: 100, High: 101, Low: 99, Close: 100},
		{OpenTime: t0.Add(4 * time.Hour), CloseTime: t0.Add(8 * time.Hour), Open: 100, High: 106, Low: 99, Close: 105},
	}, 0, BacktestTradeSideLong, SwingLevels{Entry: 100, StopLoss: 97, TakeProfit: 105, Leverage: 1}, BacktestTouchOpts{
		Interval: Interval4h,
	})
	if got == nil || got.Outcome != BacktestTradeOutcomeTP || got.ExitPrice != 105 {
		t.Fatalf("keep TP %+v", got)
	}
	if got.HitAt != nil || !got.HitTimeUnknown {
		t.Fatalf("time must be unknown without finer: %+v", got)
	}
	if got.ClosedKnownAt == nil || !got.ClosedKnownAt.Equal(t0.Add(8*time.Hour)) {
		t.Fatalf("risk free at parent close %+v", got.ClosedKnownAt)
	}
}

func TestTradeOpenAt(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	open := &BacktestTrade{Outcome: BacktestTradeOutcomeOpen}
	if !TradeOpenAt(open, t0) {
		t.Fatal("no hit stays open")
	}
	hit := t0.Add(8 * time.Hour)
	closed := &BacktestTrade{Outcome: BacktestTradeOutcomeTP, HitAt: &hit}
	if TradeOpenAt(closed, t0.Add(8*time.Hour)) {
		t.Fatal("closed at entry instant is free")
	}
	if !TradeOpenAt(closed, t0.Add(4*time.Hour)) {
		t.Fatal("still open before hit")
	}
	known := t0.Add(8 * time.Hour)
	unk := &BacktestTrade{Outcome: BacktestTradeOutcomeSL, HitTimeUnknown: true, ClosedKnownAt: &known}
	if !TradeOpenAt(unk, t0.Add(4*time.Hour)) {
		t.Fatal("unknown hit must hold risk until bar close")
	}
	if TradeOpenAt(unk, t0.Add(8*time.Hour)) {
		t.Fatal("risk frees at known close")
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
