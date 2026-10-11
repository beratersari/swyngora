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
	}, nil, nil, SituationPrintInput{})
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
	}, nil, nil, SituationPrintInput{})
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
	}, nil, nil, SituationPrintInput{})
	if closer.Liquidation == nil || closer.Liquidation.Side != SituationLiqShort || closer.Liquidation.DistancePct != 1 {
		t.Fatalf("short %+v", closer.Liquidation)
	}
	none := BuildCoinSituation("ETHUSDT", now, nil, nil, &LiquidationLevelsReport{LastPrice: "100"}, nil, nil, SituationPrintInput{})
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
	}, SituationPrintInput{})
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
	}, SituationPrintInput{})
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
	withPrints := &CoinSituation{Prints: &SituationPrints{Side: SituationPrintLong, LongNotional: 100}}
	sameDollars := &CoinSituation{Prints: &SituationPrints{Side: SituationPrintLong, LongNotional: 40, ShortNotional: 10}}
	if left, changed := SituationLeft(withPrints, sameDollars); left || len(changed) != 0 {
		t.Fatalf("print dollars must stay quiet: %v %v", left, changed)
	}
	leftPrint, changedPrint := SituationLeft(withPrints, &CoinSituation{Prints: &SituationPrints{Side: SituationPrintShort}})
	if !leftPrint || len(changedPrint) != 1 || changedPrint[0] != "prints" {
		t.Fatalf("print side %v %v", leftPrint, changedPrint)
	}
	liveNew := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDPriceUp},
		Funding:  &SituationFunding{Payer: "long"},
		Prints:   &SituationPrints{Side: SituationPrintLong},
	}
	if left, changed := SituationLeft(entry, liveNew); left || len(changed) != 0 {
		t.Fatalf("a new print clause must stay quiet: %v %v", left, changed)
	}
}

func watchedPrints(now time.Time, sides []LiquidationPrintSideRow) SituationPrintInput {
	start := now.Add(-48 * time.Hour)
	cov := make([]LiquidationCoverage, 0, 3)
	for _, ex := range FuturesVenues() {
		cov = append(cov, LiquidationCoverage{Exchange: ex, Symbol: "BTCUSDT", FirstWatch: start})
	}
	return SituationPrintInput{ArchiveOK: true, Coverage: cov, Sides: sides}
}

func TestBuildCoinSituation_MeasuredPrintsStaySeparateFromTheModel(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	got := BuildCoinSituation("BTCUSDT", now, nil, nil, &LiquidationLevelsReport{
		LastPrice: "100",
		Levels:    []LiquidationLevelBar{{Price: "99", LongNotional: "10"}},
	}, nil, nil, watchedPrints(now, []LiquidationPrintSideRow{
		{Exchange: ExchangeBinance, Long: 90, Short: 10},
		{Exchange: ExchangeBybit, Long: 20, Short: 5},
		{Exchange: ExchangeOKX, Short: 5},
	}))
	if got.Liquidation == nil || got.Liquidation.Side != SituationLiqLong {
		t.Fatalf("model %+v", got.Liquidation)
	}
	if got.Prints == nil || got.Prints.Side != SituationPrintLong || got.Prints.Window != "24h" {
		t.Fatalf("prints %+v", got.Prints)
	}
	if got.Prints.LongNotional != 110 || got.Prints.ShortNotional != 20 || len(got.PrintsMissing) != 0 {
		t.Fatalf("notionals %+v missing %v", got.Prints, got.PrintsMissing)
	}

	even := BuildCoinSituation("BTCUSDT", now, nil, nil, nil, nil, nil, watchedPrints(now, []LiquidationPrintSideRow{
		{Exchange: ExchangeBinance, Long: 59, Short: 41},
	}))
	if even.Prints == nil || even.Prints.Side != SituationPrintEven {
		t.Fatalf("even %+v", even.Prints)
	}
	lead := BuildCoinSituation("BTCUSDT", now, nil, nil, nil, nil, nil, watchedPrints(now, []LiquidationPrintSideRow{
		{Exchange: ExchangeBinance, Long: 60, Short: 40},
	}))
	if lead.Prints == nil || lead.Prints.Side != SituationPrintLong {
		t.Fatalf("60pct %+v", lead.Prints)
	}
	quiet := BuildCoinSituation("BTCUSDT", now, nil, nil, nil, nil, nil, watchedPrints(now, nil))
	if quiet.Prints == nil || quiet.Prints.Side != SituationPrintQuiet || quiet.Prints.LongNotional != 0 {
		t.Fatalf("quiet %+v", quiet.Prints)
	}
	shortOnly := BuildCoinSituation("BTCUSDT", now, nil, nil, nil, nil, nil, watchedPrints(now, []LiquidationPrintSideRow{
		{Exchange: ExchangeOKX, Short: 5},
	}))
	if shortOnly.Prints == nil || shortOnly.Prints.Side != SituationPrintShort || shortOnly.Prints.ShortNotional != 5 {
		t.Fatalf("short %+v", shortOnly.Prints)
	}
}

func TestBuildCoinSituation_MeasuredPrintsStayBlankWhenTheWindowIsOpen(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	from := now.Add(-24 * time.Hour)
	start := now.Add(-48 * time.Hour)
	late := from.Add(time.Minute)
	cov := []LiquidationCoverage{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", FirstWatch: start},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", FirstWatch: late},
		{Exchange: ExchangeOKX, FirstWatch: start, Gaps: []LiquidationGap{{From: now.Add(-time.Hour)}}},
	}
	got := BuildCoinSituation("BTCUSDT", now, nil, nil, nil, nil, nil, SituationPrintInput{
		ArchiveOK: true,
		Coverage:  cov,
		Sides: []LiquidationPrintSideRow{
			{Exchange: ExchangeBinance, Long: 100},
			{Exchange: ExchangeBybit, Long: 80},
			{Exchange: ExchangeOKX, Long: 70},
		},
	})
	if got.Prints != nil || len(got.PrintsMissing) != 2 || got.PrintsMissing[0] != "bybit" || got.PrintsMissing[1] != "okx" {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}

	ended := BuildCoinSituation("BTCUSDT", now, nil, nil, nil, nil, nil, SituationPrintInput{
		ArchiveOK: true,
		Coverage: []LiquidationCoverage{
			{Exchange: ExchangeBinance, Symbol: "BTCUSDT", FirstWatch: start, Gaps: []LiquidationGap{{From: from.Add(-2 * time.Hour), To: from}}},
			{Exchange: ExchangeBybit, FirstWatch: start},
			{Exchange: ExchangeOKX, FirstWatch: start},
		},
		Sides: []LiquidationPrintSideRow{{Exchange: ExchangeBinance, Long: 3}},
	})
	if ended.Prints == nil || ended.Prints.Side != SituationPrintLong {
		t.Fatalf("gap that ends at the open is recorded: %+v missing %v", ended.Prints, ended.PrintsMissing)
	}

	offline := BuildCoinSituation("BTCUSDT", now, nil, nil, nil, nil, nil, SituationPrintInput{
		Coverage: []LiquidationCoverage{{Exchange: ExchangeBinance, FirstWatch: start}},
		Sides:    []LiquidationPrintSideRow{{Exchange: ExchangeBinance, Long: 100}},
	})
	if offline.Prints != nil || len(offline.PrintsMissing) != 3 {
		t.Fatalf("unread archive %+v missing %v", offline.Prints, offline.PrintsMissing)
	}

	pairLate := BuildCoinSituation("BTCUSDT", now, nil, nil, nil, nil, nil, SituationPrintInput{
		ArchiveOK: true,
		Coverage: []LiquidationCoverage{
			{Exchange: ExchangeBinance, FirstWatch: start},
			{Exchange: ExchangeBinance, Symbol: "BTCUSDT", FirstWatch: late},
			{Exchange: ExchangeBybit, FirstWatch: start},
			{Exchange: ExchangeOKX, FirstWatch: start},
		},
	})
	if pairLate.Prints != nil || len(pairLate.PrintsMissing) != 1 || pairLate.PrintsMissing[0] != "binance" {
		t.Fatalf("pair clock %+v missing %v", pairLate.Prints, pairLate.PrintsMissing)
	}
}
