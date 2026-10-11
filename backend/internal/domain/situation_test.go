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
	}, nil, nil)
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
	}, nil, nil)
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
	}, nil, nil)
	if closer.Liquidation == nil || closer.Liquidation.Side != SituationLiqShort || closer.Liquidation.DistancePct != 1 {
		t.Fatalf("short %+v", closer.Liquidation)
	}
	none := BuildCoinSituation("ETHUSDT", now, nil, nil, &LiquidationLevelsReport{LastPrice: "100"}, nil, nil)
	if none.Liquidation == nil || none.Liquidation.Side != SituationLiqNone || none.Liquidation.DistancePct != 0 {
		t.Fatalf("none %+v", none.Liquidation)
	}
}

func TestBuildCoinSituation_FundingCrowdAndSizeStayBlankUntilCovered(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	partial := BuildCoinSituation("BTCUSDT", now, &CVDReport{
		Combined: &CVDVenueSeries{Complete: true, Points: []CVDPoint{{Time: now, SizeKnown: false}}},
		Venues: []CVDVenueSeries{
			{Exchange: ExchangeBinance, Complete: true, Points: []CVDPoint{{Time: now, SizeKnown: true}}},
		},
	}, nil, nil, &FundingSnapshot{
		Pressure: &FundingPressure{
			Payer:   "long",
			RatePct: "0.01",
			Venues: []FundingPressureVenue{
				{Exchange: "binance"},
				{Exchange: "bybit"},
			},
		},
	}, &LongShortSnapshot{
		Venues: []LongShortVenueSnap{
			{Exchange: "binance", Gap: &LongShortGap{Lean: LongShortGapLeanTopLong, GapPct: "+8"}},
		},
	})
	if partial.Funding != nil || len(partial.FundingMissing) != 1 || partial.FundingMissing[0] != "okx" {
		t.Fatalf("funding %+v missing %v", partial.Funding, partial.FundingMissing)
	}
	if partial.Crowd != nil || len(partial.CrowdMissing) != 1 || partial.CrowdMissing[0] != "okx" {
		t.Fatalf("crowd %+v missing %v", partial.Crowd, partial.CrowdMissing)
	}
	if partial.Size != nil || len(partial.SizeMissing) != 2 || partial.SizeMissing[0] != "bybit" || partial.SizeMissing[1] != "okx" {
		t.Fatalf("size %+v missing %v", partial.Size, partial.SizeMissing)
	}
}

func TestBuildCoinSituation_FundingCrowdAndSizeWhenEveryPublisherIsIn(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	bar := now.Add(-5 * time.Minute)
	point := CVDPoint{Time: bar, SizeKnown: true, WhaleDelta: 2_000_000, RetailDelta: -500_000}
	venue := func(ex Exchange) CVDVenueSeries {
		return CVDVenueSeries{Exchange: ex, Complete: true, Points: []CVDPoint{point}}
	}
	got := BuildCoinSituation("BTCUSDT", now, &CVDReport{
		Combined: &CVDVenueSeries{Complete: true, Points: []CVDPoint{point}},
		Venues:   []CVDVenueSeries{venue(ExchangeBinance), venue(ExchangeBybit), venue(ExchangeOKX)},
	}, nil, nil, &FundingSnapshot{
		Pressure: &FundingPressure{
			Payer:   "short",
			RatePct: "-0.02",
			Venues: []FundingPressureVenue{
				{Exchange: "binance"},
				{Exchange: "bybit"},
				{Exchange: "okx"},
			},
		},
	}, &LongShortSnapshot{
		Venues: []LongShortVenueSnap{
			{Exchange: "binance", Gap: &LongShortGap{Lean: LongShortGapLeanTopLong, GapPct: "+10"}},
			{Exchange: "bybit"},
			{Exchange: "okx", Gap: &LongShortGap{Lean: LongShortGapLeanTopLong, GapPct: "+4"}},
		},
	})
	if got.Funding == nil || got.Funding.Payer != "short" || got.Funding.RatePct >= 0 {
		t.Fatalf("funding %+v", got.Funding)
	}
	if got.Crowd == nil || got.Crowd.Lean != LongShortGapLeanTopLong || got.Crowd.GapPct != 7 {
		t.Fatalf("crowd %+v", got.Crowd)
	}
	if got.Size == nil || got.Size.Kind != SituationSizeWhaleBuyRetailSell {
		t.Fatalf("size %+v", got.Size)
	}
	if len(got.FundingMissing) != 0 || len(got.CrowdMissing) != 0 || len(got.SizeMissing) != 0 {
		t.Fatalf("missing funding %v crowd %v size %v", got.FundingMissing, got.CrowdMissing, got.SizeMissing)
	}
}

func TestSituationLeft_OnlyClausesStoredAtEntry(t *testing.T) {
	entry := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long"},
	}
	same := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long"},
		Size:     &SituationSize{Kind: SituationSizeBothBuy},
	}
	if left, changed := SituationLeft(entry, same); left || len(changed) != 0 {
		t.Fatalf("new clause must not count as leaving: %v %v", left, changed)
	}
	live := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDAgrees},
		Funding:  &SituationFunding{Payer: "short"},
	}
	left, changed := SituationLeft(entry, live)
	if !left || len(changed) != 2 || changed[0] != "priceCvd" || changed[1] != "funding" {
		t.Fatalf("left %v changed %v", left, changed)
	}
	blank, changed := SituationLeft(entry, &CoinSituation{})
	if !blank || len(changed) != 2 {
		t.Fatalf("blank live %v %v", blank, changed)
	}
	if left, _ := SituationLeft(nil, live); left {
		t.Fatal("no entry")
	}
}
