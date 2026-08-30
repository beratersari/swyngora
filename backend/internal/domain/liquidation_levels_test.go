package domain

import (
	"testing"
	"time"
)

func TestParseLiquidationLevelsSymbol(t *testing.T) {
	got, err := ParseLiquidationLevelsSymbol("")
	if err != nil || got != "all" {
		t.Fatalf("empty %s %v", got, err)
	}
	got, err = ParseLiquidationLevelsSymbol("ETH")
	if err != nil || got != "ETH" {
		// ValidateOpenInterestSymbol uses NormalizeLiquidationSymbol
		if got != "ETHUSDT" && got != "ETH" {
			t.Fatalf("ETH → %s %v", got, err)
		}
	}
	got, err = ParseLiquidationLevelsSymbol("btcusdt")
	if err != nil || got != "BTCUSDT" {
		t.Fatalf("btc %s %v", got, err)
	}
}

func TestCollapseHuntHeatmapLevels(t *testing.T) {
	grid := HuntHeatmapGrid{
		Longs:  [][]float64{{10, 0}, {5, 1}},
		Shorts: [][]float64{{0, 8}, {2, 4}},
	}
	got := CollapseHuntHeatmapLevels(grid, []float64{100, 90})
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].LongNotional != "15" || got[0].ShortNotional != "2" {
		t.Fatalf("high bin %+v", got[0])
	}
	if got[1].LongNotional != "1" || got[1].ShortNotional != "12" {
		t.Fatalf("low bin %+v", got[1])
	}
}

func TestLiquidationBook_TimeBarsSeparateVenues(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	from := now.Add(-2 * time.Hour)
	b.Record(LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 1, Quantity: 1, Notional: 100, Time: now.Add(-30 * time.Minute),
	})
	b.Record(LiquidationEvent{
		Exchange: ExchangeBybit, Symbol: "ETHUSDT", Side: LiquidationSideShort,
		Price: 1, Quantity: 1, Notional: 40, Time: now.Add(-90 * time.Minute),
	})
	all := b.TimeBars("all", "all", from, now, time.Hour)
	if len(all) != 2 {
		t.Fatalf("bars %d", len(all))
	}
	if all[1].LongNotional != "100" || all[0].ShortNotional != "40" {
		t.Fatalf("%+v", all)
	}
	bn := b.TimeBars("binance", "all", from, now, time.Hour)
	if bn[1].LongNotional != "100" || bn[0].ShortNotional != "0" {
		t.Fatalf("binance %+v", bn)
	}
	onlyBTC := b.TimeBars("all", "BTCUSDT", from, now, time.Hour)
	if onlyBTC[1].LongNotional != "100" || onlyBTC[0].Count != 0 {
		t.Fatalf("btc %+v", onlyBTC)
	}
}
