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

func TestBuildLiquidationLevels_FiveTimesSitsPastTheTenTimesBand(t *testing.T) {
	got := BuildLiquidationLevelsFromHunt(HuntReport{
		Symbol: "BTCUSDT", Exchange: "binance",
		Venues: []HuntVenueReport{{
			Exchange: ExchangeBinance, Price: 100, OpenInterestValue: 1_000_000,
			EstShortNotional: 1_000_000,
		}},
	})
	var far float64
	for _, lv := range got.Levels {
		px, err := strconv.ParseFloat(lv.Price, 64)
		if err != nil || px < 112 {
			continue
		}
		for _, sl := range lv.ByLeverage {
			if sl.Leverage != 10 {
				continue
			}
			n, _ := strconv.ParseFloat(sl.ShortNotional, 64)
			far += n
		}
	}
	// 5x is 15% of the short side and liquidates near +20%, past the old 10x edge.
	if far < 100_000 {
		t.Fatalf("5x mass above +12%% = %v", far)
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
	var farShortCum, nearLastCum float64
	for _, lv := range got.Levels {
		px, _ := strconv.ParseFloat(lv.Price, 64)
		cumS, _ := strconv.ParseFloat(lv.CumShort, 64)
		if cumS > maxCumShort {
			maxCumShort = cumS
		}
		if px >= 108 && cumS > farShortCum {
			farShortCum = cumS
		}
		if px > 100 && px < 103 && (nearLastCum == 0 || cumS < nearLastCum) {
			nearLastCum = cumS
		}
		if len(lv.ByLeverage) != 4 || len(lv.CumByLeverage) != 4 {
			t.Fatalf("want all 4 leverage slices, got %+v / %+v", lv.ByLeverage, lv.CumByLeverage)
		}
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
	}
	if !saw100 || !saw10 || !saw50 {
		t.Fatalf("leverage slices missing 100=%v 10=%v 50=%v", saw100, saw10, saw50)
	}
	if maxCumShort < 100 {
		t.Fatalf("cum to far short bar should include nearer shorts, got %v", maxCumShort)
	}
	if farShortCum+1 < nearLastCum {
		t.Fatalf("cumulative shorts should grow away from last: near=%v far=%v", nearLastCum, farShortCum)
	}

	all := BuildLiquidationLevelsFromHunt(HuntReport{Symbol: "BTCUSDT", Exchange: "all", Venues: rep.Venues})
	ref, _ := strconv.ParseFloat(all.LastPrice, 64)
	if ref < 70 || ref > 80 {
		t.Fatalf("combined last should be OI-weighted (~75), got %s", all.LastPrice)
	}
	if all.LastPrices["binance"] == "" || all.LastPrices["bybit"] == "" {
		t.Fatalf("combined lastPrices %+v", all.LastPrices)
	}
	// Bybit 10x at +10% of its own last maps near +10% of the shared last,
	// not onto Binance's absolute $110 bin.
	var bybitMassNearShared, bybitMassAtBinance110 float64
	for _, lv := range all.Levels {
		px, _ := strconv.ParseFloat(lv.Price, 64)
		shortN, _ := strconv.ParseFloat(lv.ShortNotional, 64)
		if px > ref*1.06 && px < ref*1.14 {
			bybitMassNearShared += shortN
		}
		if px > 108 && px < 112 {
			bybitMassAtBinance110 += shortN
		}
	}
	if bybitMassNearShared < 400 {
		t.Fatalf("Bybit 10x should sit at +%% on the shared axis, mass=%v", bybitMassNearShared)
	}
	if bybitMassAtBinance110 > bybitMassNearShared {
		t.Fatalf("must not snap Bybit $55 onto Binance $110: shared=%v abs110=%v", bybitMassNearShared, bybitMassAtBinance110)
	}
}

func TestOiWeightedLast_IgnoresMissingVenue(t *testing.T) {
	got := oiWeightedLast([]HuntVenueReport{
		{Price: 100, OpenInterestValue: 2},
		{Price: 0, OpenInterestValue: 9},
	})
	if got != 100 {
		t.Fatalf("got %v", got)
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
	if len(got.Missing) != 2 {
		t.Fatalf("missing %+v", got.Missing)
	}
	seen := map[string]bool{}
	for _, m := range got.Missing {
		seen[m] = true
	}
	if !seen["bybit"] || !seen["okx"] {
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
