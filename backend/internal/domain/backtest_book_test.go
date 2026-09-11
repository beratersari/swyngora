package domain

import (
	"math"
	"testing"
	"time"
)

func TestMergeBacktestMarkets_KeepsZeroAndErrors(t *testing.T) {
	t.Parallel()
	planned := []BacktestMarket{
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT"},
		{Exchange: ExchangeBinance, Symbol: "ADAUSDT"},
		{Exchange: ExchangeBinance, Symbol: "XRPUSDT"},
	}
	results := []BacktestMarketResult{
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Status: BacktestMarketOK, SignalCount: 2},
		{Exchange: ExchangeBinance, Symbol: "ADAUSDT", Status: BacktestMarketOK, SignalCount: 0, Bars: 40},
		{Exchange: ExchangeBinance, Symbol: "XRPUSDT", Status: BacktestMarketError, Error: "upstream 429"},
	}
	ptr := func(v float64) *float64 { return &v }
	got := MergeBacktestMarkets(planned, results, []ScannerBacktestSignal{
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Return1d: ptr(2)},
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Return1d: ptr(1)},
	})
	if len(got) != 3 {
		t.Fatalf("%+v", got)
	}
	if got[0].Symbol != "ETHUSDT" || got[0].SignalCount != 2 {
		t.Fatalf("eth %+v", got[0])
	}
	if got[1].Symbol != "ADAUSDT" || got[1].SignalCount != 0 || got[1].Status != BacktestMarketOK {
		t.Fatalf("ada %+v", got[1])
	}
	if got[2].Status != BacktestMarketError || got[2].Error != "upstream 429" {
		t.Fatalf("xrp %+v", got[2])
	}
}

func TestSimulateSharedBacktestBook_RiskStopAndRange(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	// 1% risk, 1% stop: full stop loses 1% of 10000 = 100.
	stop := SimulateSharedBacktestBook(10000, 1, []BacktestBookFill{
		{Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 99, RangeExitPrice: 99, Leverage: 1, ClosedInRange: true},
	}, t0.Add(24*time.Hour))
	if stop.TradesTaken != 1 {
		t.Fatalf("taken %+v", stop)
	}
	if math.Abs(stop.EndingBalance-9900) > 0.05 {
		t.Fatalf("stop should lose ~1%%: %+v", stop)
	}
	// Next trade sizes on 9900: another 1% stop loses 99.
	two := SimulateSharedBacktestBook(10000, 1, []BacktestBookFill{
		{Symbol: "A", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 99, RangeExitPrice: 99, Leverage: 1, ClosedInRange: true},
		{Symbol: "B", Side: BacktestTradeSideLong, EntryAt: t0.Add(2 * time.Hour), ExitAt: t0.Add(3 * time.Hour),
			Entry: 50, StopLoss: 49.5, ExitPrice: 49.5, RangeExitPrice: 49.5, Leverage: 1, ClosedInRange: true},
	}, t0.Add(24*time.Hour))
	if math.Abs(two.EndingBalance-9801) > 0.1 {
		t.Fatalf("compound stop %+v", two)
	}

	// Still open at range end (mark 101), TP later at 105. 1% risk, 1% stop → notional = equity.
	// Range mark +1%; after +5%.
	later := SimulateSharedBacktestBook(10000, 1, []BacktestBookFill{
		{Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(48 * time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 105, RangeExitPrice: 101, Leverage: 1, ClosedInRange: false},
	}, t0.Add(24*time.Hour))
	if later.OpenAtRangeEnd != 1 || later.ClosedAfterRange != 1 {
		t.Fatalf("later counts %+v", later)
	}
	if math.Abs(later.RangeEndBalance-10100) > 0.05 {
		t.Fatalf("range mark %+v", later)
	}
	if math.Abs(later.AfterRangeBalance-10500) > 0.05 || math.Abs(later.EndingBalance-10500) > 0.05 {
		t.Fatalf("after later TP %+v", later)
	}
}
