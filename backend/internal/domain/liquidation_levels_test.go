package domain

import (
	"strconv"
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

func TestChartLeverageBucket(t *testing.T) {
	if ChartLeverageBucket(5) != 10 || ChartLeverageBucket(10) != 10 {
		t.Fatal("10x")
	}
	if ChartLeverageBucket(25) != 25 {
		t.Fatal("25x")
	}
	if ChartLeverageBucket(50) != 50 {
		t.Fatal("50x")
	}
	if ChartLeverageBucket(75) != 100 || ChartLeverageBucket(125) != 100 {
		t.Fatal("100x")
	}
}

func TestBuildLiquidationLevelsFromHunt_OwnVenueAndCum(t *testing.T) {
	rep := HuntReport{
		Symbol:   "BTCUSDT",
		Exchange: "binance",
		Venues: []HuntVenueReport{
			{
				Exchange: ExchangeBinance, Price: 100, OpenInterestValue: 1_000_000,
				UpPressure: []HuntBand{
					{Side: LiquidationSideShort, Leverage: 100, Price: 101, EstNotional: 40},
					{Side: LiquidationSideShort, Leverage: 10, Price: 110, EstNotional: 80},
				},
				DownPressure: []HuntBand{
					{Side: LiquidationSideLong, Leverage: 50, Price: 96, EstNotional: 30},
				},
			},
			{
				Exchange: ExchangeBybit, Price: 50, OpenInterestValue: 1_000_000,
				UpPressure: []HuntBand{
					{Side: LiquidationSideShort, Leverage: 10, Price: 55, EstNotional: 999},
				},
			},
		},
	}
	got := BuildLiquidationLevelsFromHunt(rep)
	if got.LastPrice != "100" {
		t.Fatalf("binance last %s", got.LastPrice)
	}
	if got.LastPrices["bybit"] != "" {
		t.Fatal("single-venue report must not copy the other last price")
	}
	var saw100, saw10, saw50 bool
	var maxCumShort float64
	for _, lv := range got.Levels {
		for _, sl := range lv.ByLeverage {
			if sl.Leverage == 100 && sl.ShortNotional != "0" {
				saw100 = true
			}
			if sl.Leverage == 10 && sl.ShortNotional != "0" {
				saw10 = true
			}
			if sl.Leverage == 50 && sl.LongNotional != "0" {
				saw50 = true
			}
		}
		if v, _ := strconv.ParseFloat(lv.CumShort, 64); v > maxCumShort {
			maxCumShort = v
		}
	}
	if !saw100 || !saw10 || !saw50 {
		t.Fatalf("leverage slices missing 100=%v 10=%v 50=%v", saw100, saw10, saw50)
	}
	if maxCumShort < 100 {
		t.Fatalf("cum to far short bar should include nearer shorts, got %v", maxCumShort)
	}

	all := BuildLiquidationLevelsFromHunt(HuntReport{Symbol: "BTCUSDT", Exchange: "all", Venues: rep.Venues})
	if all.LastPrice != "" {
		t.Fatalf("combined last must stay empty, got %s", all.LastPrice)
	}
	if all.LastPrices["binance"] == "" || all.LastPrices["bybit"] == "" {
		t.Fatalf("combined lastPrices %+v", all.LastPrices)
	}
}

func TestBuildLiquidationLevelsFromHunt_MissingVenue(t *testing.T) {
	got := BuildLiquidationLevelsFromHunt(HuntReport{
		Symbol: "ETHUSDT", Exchange: "all",
		Venues: []HuntVenueReport{
			{Exchange: ExchangeBinance, Price: 3000, OpenInterestValue: 100, DownPressure: []HuntBand{
				{Side: LiquidationSideLong, Leverage: 25, Price: 2900, EstNotional: 10},
			}},
		},
	})
	if len(got.Missing) != 1 || got.Missing[0] != "bybit" {
		t.Fatalf("missing %+v", got.Missing)
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
