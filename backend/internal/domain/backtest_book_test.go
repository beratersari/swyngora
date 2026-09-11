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
	stop := SimulateSharedBacktestBook(10000, 1, 3, []BacktestBookFill{
		{Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 99, RangeExitPrice: 99, Leverage: 1, ClosedInRange: true, HasExit: true},
	}, t0.Add(24*time.Hour))
	if stop.TradesTaken != 1 {
		t.Fatalf("taken %+v", stop)
	}
	if math.Abs(stop.EndingBalance-9900) > 0.05 {
		t.Fatalf("stop should lose ~1%%: %+v", stop)
	}
	// Next trade sizes on 9900: another 1% stop loses 99.
	two := SimulateSharedBacktestBook(10000, 1, 3, []BacktestBookFill{
		{Symbol: "A", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 99, RangeExitPrice: 99, Leverage: 1, ClosedInRange: true, HasExit: true},
		{Symbol: "B", Side: BacktestTradeSideLong, EntryAt: t0.Add(2 * time.Hour), ExitAt: t0.Add(3 * time.Hour),
			Entry: 50, StopLoss: 49.5, ExitPrice: 49.5, RangeExitPrice: 49.5, Leverage: 1, ClosedInRange: true, HasExit: true},
	}, t0.Add(24*time.Hour))
	if math.Abs(two.EndingBalance-9801) > 0.1 {
		t.Fatalf("compound stop %+v", two)
	}

	// Still open at range end (mark 101), TP later at 105. 1% risk, 1% stop → notional = equity.
	// Range mark +1%; after +5%.
	later := SimulateSharedBacktestBook(10000, 1, 3, []BacktestBookFill{
		{Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(48 * time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 105, RangeExitPrice: 101, Leverage: 1, ClosedInRange: false, HasExit: true},
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

	// Never hits TP/SL: stay open at the in-range mark, ignore a later tape price.
	stay := SimulateSharedBacktestBook(10000, 1, 3, []BacktestBookFill{
		{Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0,
			Entry: 100, StopLoss: 99, ExitPrice: 140, RangeExitPrice: 101, Leverage: 1, Outcome: BacktestTradeOutcomeOpen},
	}, t0.Add(24*time.Hour))
	if stay.OpenAtEnd != 1 || stay.ClosedAfterRange != 0 {
		t.Fatalf("must stay open %+v", stay)
	}
	if math.Abs(stay.RangeEndBalance-10100) > 0.05 || math.Abs(stay.AfterRangeBalance-10100) > 0.05 {
		t.Fatalf("open mark must stay in-range %+v", stay)
	}
}

func TestSimulateSharedBacktestBook_MaxOpenRisk(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	fill := func(sym string, entry, exit time.Time, hasExit bool) BacktestBookFill {
		return BacktestBookFill{
			Symbol: sym, Side: BacktestTradeSideLong, EntryAt: entry, ExitAt: exit,
			Entry: 100, StopLoss: 99, ExitPrice: 100, RangeExitPrice: 100, Leverage: 10,
			ClosedInRange: hasExit, HasExit: hasExit, Outcome: BacktestTradeOutcomeOpen,
		}
	}
	// 1% each, max 3%: fourth concurrent fill is skipped.
	blocked := SimulateSharedBacktestBook(10000, 1, 3, []BacktestBookFill{
		fill("A", t0, t0.Add(10*time.Hour), false),
		fill("B", t0.Add(time.Hour), t0.Add(10*time.Hour), false),
		fill("C", t0.Add(2*time.Hour), t0.Add(10*time.Hour), false),
		fill("D", t0.Add(3*time.Hour), t0.Add(10*time.Hour), false),
	}, t0.Add(24*time.Hour))
	if blocked.TradesTaken != 3 || blocked.SkippedNoRisk != 1 {
		t.Fatalf("cap %+v", blocked)
	}

	// After A closes, D can take the free 1%.
	freed := SimulateSharedBacktestBook(10000, 1, 3, []BacktestBookFill{
		{Symbol: "A", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(2 * time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 100, RangeExitPrice: 100, Leverage: 10, ClosedInRange: true, HasExit: true, Outcome: BacktestTradeOutcomeTP},
		fill("B", t0.Add(30*time.Minute), t0.Add(10*time.Hour), false),
		fill("C", t0.Add(time.Hour), t0.Add(10*time.Hour), false),
		fill("D", t0.Add(3*time.Hour), t0.Add(10*time.Hour), false),
	}, t0.Add(24*time.Hour))
	if freed.TradesTaken != 4 || freed.SkippedNoRisk != 0 {
		t.Fatalf("free slot %+v", freed)
	}
}

func TestSimulateSharedBacktestBook_SameTimePrefersHigherScore(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	fill := func(sym string, score float64) BacktestBookFill {
		return BacktestBookFill{
			Symbol: sym, Side: BacktestTradeSideLong, EntryAt: t0,
			Entry: 100, StopLoss: 99, ExitPrice: 100, RangeExitPrice: 100, Leverage: 10,
			Score: score, Outcome: BacktestTradeOutcomeOpen,
		}
	}
	// Listed low-score first. Only one 1% slot (max 1%).
	got := SimulateSharedBacktestBook(10000, 1, 1, []BacktestBookFill{
		fill("AAAUSDT", 40),
		fill("ETHUSDT", 80),
		fill("SOLUSDT", 55),
	}, t0.Add(24*time.Hour))
	if got.TradesTaken != 1 || got.SkippedNoRisk != 2 {
		t.Fatalf("taken %+v", got)
	}
	// Reconstruct which symbol was taken: only the first allocated keep OpenAtEnd.
	// Higher score ETH must win the single slot — verify by running with only ETH vs only AAA
	// through the same sorter via a tagged fill that closes.
	eth := SimulateSharedBacktestBook(10000, 1, 1, []BacktestBookFill{
		{Symbol: "AAAUSDT", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 99, RangeExitPrice: 99, Leverage: 10,
			Score: 40, HasExit: true, ClosedInRange: true, Outcome: BacktestTradeOutcomeSL},
		{Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 102, RangeExitPrice: 102, Leverage: 10,
			Score: 80, HasExit: true, ClosedInRange: true, Outcome: BacktestTradeOutcomeTP},
	}, t0.Add(24*time.Hour))
	// ETH +2% on 1% risk / 1% stop → notional = equity, +2% of 10000 = +200 if taken.
	// AAA stop −1% = −100 if taken.
	if math.Abs(eth.EndingBalance-10200) > 0.5 {
		t.Fatalf("higher score must take the slot (ETH +2%%): %+v", eth)
	}
}

func TestSimulateSharedBacktestBook_OneOpenPerCoin(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got := SimulateSharedBacktestBook(10000, 1, 10, []BacktestBookFill{
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0,
			Entry: 100, StopLoss: 99, ExitPrice: 100, RangeExitPrice: 100, Leverage: 10, Outcome: BacktestTradeOutcomeOpen},
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0.Add(time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 102, RangeExitPrice: 102, Leverage: 10, Outcome: BacktestTradeOutcomeOpen},
		{Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: BacktestTradeSideLong, EntryAt: t0.Add(time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 100, RangeExitPrice: 100, Leverage: 10, Outcome: BacktestTradeOutcomeOpen},
	}, t0.Add(24*time.Hour))
	if got.TradesTaken != 2 || got.SkippedSameCoin != 1 {
		t.Fatalf("one ETH open + SOL: %+v", got)
	}

	freed := SimulateSharedBacktestBook(10000, 1, 10, []BacktestBookFill{
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0, ExitAt: t0.Add(2 * time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 100, RangeExitPrice: 100, Leverage: 10,
			HasExit: true, ClosedInRange: true, Outcome: BacktestTradeOutcomeTP},
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: BacktestTradeSideLong, EntryAt: t0.Add(3 * time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 100, RangeExitPrice: 100, Leverage: 10, Outcome: BacktestTradeOutcomeOpen},
	}, t0.Add(24*time.Hour))
	if freed.TradesTaken != 2 || freed.SkippedSameCoin != 0 {
		t.Fatalf("after close next ETH can open: %+v", freed)
	}
}

func TestSimulateSharedBacktestBook_UnknownHitHoldsRiskUntilBarClose(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	// ETH TP sometime in 08:00–12:00. Risk stays until 12:00, not 08:00.
	got := SimulateSharedBacktestBook(10000, 1, 1, []BacktestBookFill{
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Side: BacktestTradeSideLong,
			EntryAt: t0, ExitAt: t0.Add(4 * time.Hour),
			Entry: 100, StopLoss: 99, ExitPrice: 105, RangeExitPrice: 105, Leverage: 10,
			HasExit: true, ClosedInRange: true, HitTimeUnknown: true, Outcome: BacktestTradeOutcomeTP},
		{Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: BacktestTradeSideLong,
			EntryAt: t0.Add(time.Hour),
			Entry:   50, StopLoss: 49.5, ExitPrice: 50, RangeExitPrice: 50, Leverage: 10,
			Outcome: BacktestTradeOutcomeOpen},
		{Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: BacktestTradeSideLong,
			EntryAt: t0.Add(4 * time.Hour),
			Entry:   50, StopLoss: 49.5, ExitPrice: 50, RangeExitPrice: 50, Leverage: 10,
			Outcome: BacktestTradeOutcomeOpen},
	}, t0.Add(24*time.Hour))
	if got.SkippedNoRisk != 1 || got.TradesTaken != 2 {
		t.Fatalf("hold risk until parent close: %+v", got)
	}
}
