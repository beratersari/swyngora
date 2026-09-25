package domain

import (
	"testing"
	"time"
)

func TestWalkStoredLadder_BuyPaysThroughTheAsks(t *testing.T) {
	asks := []BookHistoryLevel{
		{Price: 100, Quantity: 1},
		{Price: 110, Quantity: 1},
	}
	avg, slip, filled, ok := WalkStoredLadder(asks, 2, 100, true)
	if !ok || filled != 2 || avg != 105 || slip < 0.049 || slip > 0.051 {
		t.Fatalf("avg=%v slip=%v filled=%v ok=%v", avg, slip, filled, ok)
	}
}

func TestSimulateSharedBacktestBookLadder_UsesRestingSize(t *testing.T) {
	at := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	exit := at.Add(time.Hour)
	fill := BacktestBookFill{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: BacktestTradeSideLong,
		EntryAt: at, ExitAt: exit, Entry: 100, StopLoss: 90, ExitPrice: 110,
		Leverage: 1, HasExit: true, ClosedInRange: true, FeeRate: 0,
	}
	fixed := SimulateSharedBacktestBook(10000, 1, 3, []BacktestBookFill{fill}, exit)
	ladder := func(ex Exchange, symbol string, when time.Time) ([]BookHistoryLevel, []BookHistoryLevel, bool) {
		if when.Equal(at) {
			return nil, []BookHistoryLevel{{Price: 100, Quantity: 0.01}, {Price: 200, Quantity: 10}}, true
		}
		return []BookHistoryLevel{{Price: 110, Quantity: 10}}, nil, true
	}
	got := SimulateSharedBacktestBookLadder(10000, 1, 3, []BacktestBookFill{fill}, exit, ladder)
	if got.BookLegs < 1 || got.EndingBalance >= fixed.EndingBalance {
		t.Fatalf("book=%+v fixed=%+v", got, fixed)
	}
}

func TestSimulateSharedBacktestBookLadder_ThinBidDoesNotCloseThePosition(t *testing.T) {
	at := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	exitAt := at.Add(time.Hour)
	// 1% risk, 10% stop → notional 1000 at entry 100, so the close needs 10 BTC.
	fill := BacktestBookFill{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: BacktestTradeSideLong,
		EntryAt: at, ExitAt: exitAt, Entry: 100, StopLoss: 90, ExitPrice: 110,
		RangeExitPrice: 100, Leverage: 1, HasExit: true, ClosedInRange: true, FeeRate: 0,
	}
	ladder := func(ex Exchange, symbol string, when time.Time) ([]BookHistoryLevel, []BookHistoryLevel, bool) {
		if when.Equal(at) {
			return nil, []BookHistoryLevel{{Price: 100, Quantity: 10}}, true
		}
		return []BookHistoryLevel{{Price: 110, Quantity: 0.01}}, nil, true
	}
	got := SimulateSharedBacktestBookLadder(10000, 1, 3, []BacktestBookFill{fill}, exitAt, ladder)
	if got.OpenAtEnd != 1 || got.ClosedInRange != 0 || got.BookShortCloses != 1 {
		t.Fatalf("thin bid must leave the position open: %+v", got)
	}
	// 0.01 BTC sold at 110 is 1 notional, +0.10 pnl. The other 9.99 stays marked at 100.
	if got.EndingBalance < 10000.09 || got.EndingBalance > 10000.11 {
		t.Fatalf("ending=%v want ~10000.10", got.EndingBalance)
	}
}
