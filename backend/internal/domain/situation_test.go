package domain

import (
	"testing"
	"time"
)

func TestBuildCoinSituation_CompleteRead(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	day := now.Truncate(24 * time.Hour)
	_, series := VenueCVDSplit(ExchangeBinance, "BTCUSDT", []TakerBucket{
		{Start: day, BuyNotional: 1, SellNotional: 1},
		{Start: now.Add(-20 * time.Minute), BuyNotional: 0, SellNotional: 50},
		{Start: now.Add(-15 * time.Minute), BuyNotional: 0, SellNotional: 50},
		{Start: now.Add(-10 * time.Minute), BuyNotional: 0, SellNotional: 50},
		{Start: now.Add(-5 * time.Minute), BuyNotional: 0, SellNotional: 50},
	}, []CVDPrice{
		{Time: day, Close: 100},
		{Time: now.Add(-20 * time.Minute), Close: 101},
		{Time: now.Add(-15 * time.Minute), Close: 101.3},
		{Time: now.Add(-10 * time.Minute), Close: 101.6},
		{Time: now.Add(-5 * time.Minute), Close: 101.9},
	}, now)
	series.Complete = true
	series.Exchange = "all"
	venue := func(ex Exchange) CVDVenueSeries {
		v := series
		v.Exchange = ex
		v.Complete = true
		return v
	}
	got := BuildCoinSituation("btcusdt", now, &CVDReport{
		Combined: &series,
		Venues:   []CVDVenueSeries{venue(ExchangeBinance), venue(ExchangeBybit), venue(ExchangeOKX)},
	}, &OpenInterestSnapshot{
		Windows: []OpenInterestWindow{{Window: OpenInterestWindow1h, Flow: OpenInterestFlowOpened, Complete: true}},
		Venues: []OpenInterestVenueSnap{
			{Exchange: "binance", Windows: []OpenInterestWindow{{Window: "1h", Complete: true}}},
			{Exchange: "bybit", Windows: []OpenInterestWindow{{Window: "1h", Complete: true}}},
			{Exchange: "okx", Windows: []OpenInterestWindow{{Window: "1h", Complete: true}}},
		},
	}, &LiquidationLevelsReport{
		LastPrice: "100",
		Levels: []LiquidationLevelBar{
			{Price: "99", LongNotional: "10"},
			{Price: "90", LongNotional: "80"},
			{Price: "103", ShortNotional: "40"},
		},
	})
	if got.Symbol != "BTCUSDT" || got.PriceCVD == nil || got.PriceCVD.Kind != SituationCVDPriceUp {
		t.Fatalf("cvd %+v", got.PriceCVD)
	}
	if got.OpenInterest == nil || got.OpenInterest.Flow != SituationOIOpened || got.OpenInterest.Window != "1h" {
		t.Fatalf("oi %+v", got.OpenInterest)
	}
	if got.Liquidation == nil || got.Liquidation.Side != SituationLiqLong || got.Liquidation.DistancePct != 1 {
		t.Fatalf("liq %+v", got.Liquidation)
	}
	if len(got.PriceCVDMissing) != 0 || len(got.OpenInterestMissing) != 0 || len(got.LiquidationMissing) != 0 {
		t.Fatalf("missing should stay empty: %+v", got)
	}
}

func TestBuildCoinSituation_NamesTheMissingVenueAndLeavesThatClauseBlank(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	got := BuildCoinSituation("BTCUSDT", now, &CVDReport{
		Combined: &CVDVenueSeries{Complete: false},
		Venues: []CVDVenueSeries{
			{Exchange: ExchangeBinance, Market: CVDMarketFutures, Complete: true, Points: []CVDPoint{{Price: 1}}},
			{Exchange: ExchangeBybit, Market: CVDMarketFutures, Complete: false},
			{Exchange: ExchangeOKX, Market: CVDMarketFutures, Complete: true, Points: []CVDPoint{{Price: 1}}},
		},
	}, &OpenInterestSnapshot{
		Windows: []OpenInterestWindow{{Window: "1h", Flow: OpenInterestFlowOpened, Complete: true}},
		Venues: []OpenInterestVenueSnap{
			{Exchange: "binance", Windows: []OpenInterestWindow{{Window: "1h", Complete: true}}},
			{Exchange: "okx", Windows: []OpenInterestWindow{{Window: "1h", Complete: false}}},
		},
	}, &LiquidationLevelsReport{
		LastPrice: "100",
		Missing:   []string{"okx"},
		Levels:    []LiquidationLevelBar{{Price: "99", LongNotional: "10"}},
	})
	if got.PriceCVD != nil || len(got.PriceCVDMissing) != 1 || got.PriceCVDMissing[0] != "bybit" {
		t.Fatalf("cvd %+v missing %v", got.PriceCVD, got.PriceCVDMissing)
	}
	if got.OpenInterest != nil || len(got.OpenInterestMissing) != 2 || got.OpenInterestMissing[0] != "bybit" || got.OpenInterestMissing[1] != "okx" {
		t.Fatalf("oi %+v missing %v", got.OpenInterest, got.OpenInterestMissing)
	}
	if got.Liquidation != nil || len(got.LiquidationMissing) != 1 || got.LiquidationMissing[0] != "okx" {
		t.Fatalf("liq %+v missing %v", got.Liquidation, got.LiquidationMissing)
	}
}

func TestBuildCoinSituation_ShortPocketCloserAndNone(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	closer := BuildCoinSituation("ETHUSDT", now, nil, nil, &LiquidationLevelsReport{
		LastPrice: "100",
		Levels: []LiquidationLevelBar{
			{Price: "90", LongNotional: "50"},
			{Price: "101", ShortNotional: "5"},
		},
	})
	if closer.Liquidation == nil || closer.Liquidation.Side != SituationLiqShort || closer.Liquidation.DistancePct != 1 {
		t.Fatalf("short %+v", closer.Liquidation)
	}
	none := BuildCoinSituation("ETHUSDT", now, nil, nil, &LiquidationLevelsReport{LastPrice: "100"})
	if none.Liquidation == nil || none.Liquidation.Side != SituationLiqNone || none.Liquidation.DistancePct != 0 {
		t.Fatalf("none %+v", none.Liquidation)
	}
}
