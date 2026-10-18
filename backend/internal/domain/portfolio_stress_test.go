package domain

import "testing"

func TestStressPortfolio_SpotDropAndMarginLiquidation(t *testing.T) {
	view := PortfolioView{
		Equity: 10000, TotalPnL: 0, UnrealizedPnL: 0,
		PositionsValue: 600, MarginEquity: 20, MarginUnrealizedPnL: 0,
		Positions: []PositionView{
			{Symbol: "BTCUSDT", Quantity: 1, AvgCost: 100, MarkPrice: 100, MarketValue: 100, UnrealizedPnL: 0},
			{Symbol: "ETHUSDT", Quantity: 10, AvgCost: 50, MarkPrice: 50, MarketValue: 500, UnrealizedPnL: 0},
		},
		MarginPositions: []MarginPosition{
			{Symbol: "BTCUSDT", Side: MarginLong, Status: MarginPositionOpen, Quantity: 1, EntryPrice: 100, MarkPrice: 100, Margin: 20, LiquidationPrice: 91, UnrealizedPnL: 0},
		},
	}
	got, err := StressPortfolio(view, []StressMove{{Asset: "BTC", MovePct: -10}, {Asset: "eth", MovePct: -15}})
	if err != nil {
		t.Fatal(err)
	}
	if got.ShockedPositionsValue < 514 || got.ShockedPositionsValue > 516 {
		t.Fatalf("spot value %v", got.ShockedPositionsValue)
	}
	if got.Liquidated != 1 {
		t.Fatalf("btc margin 100→90 crosses 91, liquidated=%d", got.Liquidated)
	}
	// Spot BTC -10, spot ETH -75, margin loses the 20 locked. Equity 10000-105.
	if got.EquityDelta > -104 || got.EquityDelta < -106 {
		t.Fatalf("equity delta %v", got.EquityDelta)
	}
	if got.ShockedMarginEquity > 1 {
		t.Fatalf("liquidated margin equity %v", got.ShockedMarginEquity)
	}
	if got.ShockedTotalPnL > -104 || got.ShockedTotalPnL < -106 {
		t.Fatalf("pnl %v", got.ShockedTotalPnL)
	}
}

func TestStressPortfolio_VenueMoveLeavesTheOtherLot(t *testing.T) {
	view := PortfolioView{
		Equity: 10000, PositionsValue: 400,
		Positions: []PositionView{
			{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, AvgCost: 100, MarkPrice: 100, MarketValue: 100},
			{Exchange: ExchangeCoinbase, Symbol: "BTC-USD", Quantity: 6, AvgCost: 50, MarkPrice: 50, MarketValue: 300},
		},
	}
	got, err := StressPortfolio(view, []StressMove{{
		Asset: "BTC", Exchange: "binance", Symbol: "BTCUSDT", MovePct: -20,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Lines) != 1 || got.Lines[0].Symbol != "BTCUSDT" {
		t.Fatalf("lines %+v", got.Lines)
	}
	if got.EquityDelta > -19 || got.EquityDelta < -21 {
		t.Fatalf("only the binance lot should move, delta %v", got.EquityDelta)
	}
}

func TestParseStressMoves_RejectsWipeout(t *testing.T) {
	if _, err := ParseStressMoves([]StressMove{{Asset: "BTC", MovePct: -100}}); err == nil {
		t.Fatal("expected reject")
	}
}
