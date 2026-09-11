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

func TestSimulateSharedBacktestBook_CompoundsAndSkips(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	// Sequential: first +10% on 10% stake of 10000 = +100 → 10100
	// second -5% on 10% of 10100 = -50.5 → 10049.5
	seq := SimulateSharedBacktestBook(10000, 10, []BacktestBookFill{
		{Symbol: "ETHUSDT", EntryAt: t0, ExitAt: t0.Add(time.Hour), PnLPct: 10},
		{Symbol: "SOLUSDT", EntryAt: t0.Add(2 * time.Hour), ExitAt: t0.Add(3 * time.Hour), PnLPct: -5},
	})
	if seq.TradesTaken != 2 || seq.TradesSkipped != 0 {
		t.Fatalf("seq counts %+v", seq)
	}
	if math.Abs(seq.EndingBalance-10049.5) > 0.05 {
		t.Fatalf("seq end %+v", seq)
	}

	// Overlap: two entries before any exit, each 10% of 10000. Cash left 8000.
	// Third entry while both open: stake 10% of 10000 = 1000, cash 7000. Then all close flat.
	overlap := SimulateSharedBacktestBook(10000, 10, []BacktestBookFill{
		{Symbol: "A", EntryAt: t0, ExitAt: t0.Add(3 * time.Hour), PnLPct: 0},
		{Symbol: "B", EntryAt: t0.Add(time.Hour), ExitAt: t0.Add(3 * time.Hour), PnLPct: 0},
		{Symbol: "C", EntryAt: t0.Add(2 * time.Hour), ExitAt: t0.Add(4 * time.Hour), PnLPct: 0},
	})
	if overlap.TradesTaken != 3 {
		t.Fatalf("overlap taken %+v", overlap)
	}

	// Stake 50% so a third concurrent fill is skipped (cash 0 after two opens).
	tight := SimulateSharedBacktestBook(10000, 50, []BacktestBookFill{
		{Symbol: "A", EntryAt: t0, ExitAt: t0.Add(3 * time.Hour), PnLPct: 0},
		{Symbol: "B", EntryAt: t0.Add(time.Hour), ExitAt: t0.Add(3 * time.Hour), PnLPct: 0},
		{Symbol: "C", EntryAt: t0.Add(2 * time.Hour), ExitAt: t0.Add(4 * time.Hour), PnLPct: 10},
	})
	if tight.TradesTaken != 2 || tight.TradesSkipped != 1 {
		t.Fatalf("tight %+v", tight)
	}
	if math.Abs(tight.EndingBalance-10000) > 0.05 {
		t.Fatalf("flat end %+v", tight)
	}
}
