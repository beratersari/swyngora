package domain

import (
	"strconv"
	"testing"
	"time"
)

func TestPredictDisplayLevelBands_SidesAndDistance(t *testing.T) {
	got := PredictDisplayLevelBands(100, 1_000_000, 0.5, 0.5, 0)
	if len(got) < 4 {
		t.Fatalf("bands %d", len(got))
	}
	var saw100s, saw10s, saw100l bool
	for _, b := range got {
		if b.Leverage == 100 && b.Side == LiquidationSideShort {
			saw100s = true
			if b.Price <= 100 {
				t.Fatalf("100x short should sit above last, got %v", b.Price)
			}
			if b.Predicted < LiquidationLevelsReviewMinPredicted {
				t.Fatalf("predicted %v", b.Predicted)
			}
		}
		if b.Leverage == 10 && b.Side == LiquidationSideShort && b.Price <= 100 {
			t.Fatal("10x short below last")
		}
		if b.Leverage == 100 && b.Side == LiquidationSideLong {
			saw100l = true
			if b.Price >= 100 {
				t.Fatalf("100x long should sit below last, got %v", b.Price)
			}
		}
		if b.Leverage == 10 && b.Side == LiquidationSideShort {
			saw10s = true
		}
	}
	if !saw100s || !saw10s || !saw100l {
		t.Fatalf("missing buckets 100s=%v 10s=%v 100l=%v", saw100s, saw10s, saw100l)
	}
}

func TestReviewLiquidationLevels_HitMissPendingAndVenue(t *testing.T) {
	t0 := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	now := t0.Add(2 * time.Hour)
	bands := PredictDisplayLevelBands(100, 2_000_000, 0.5, 0.5, 0)
	var short100 predictedLevelBand
	for _, b := range bands {
		if b.Leverage == 100 && b.Side == LiquidationSideShort {
			short100 = b
		}
	}
	if short100.Price <= 0 {
		t.Fatal("no 100x short band")
	}
	in := LiquidationLevelsReviewInput{
		Now:  now,
		Step: time.Hour,
		Venues: []HuntHeatmapVenueSeries{
			{
				Exchange: ExchangeBinance,
				Prices:   []HuntHeatmapPricePoint{{Time: t0, Price: 100}},
				OI:       []FuturesSnapshot{{SampledAt: t0, Value: 2_000_000}},
				Liquidations: []LiquidationEvent{
					{Exchange: ExchangeBinance, Side: LiquidationSideShort, Price: short100.Price, Notional: 80_000, Time: t0.Add(-time.Hour)},
					{Exchange: ExchangeBinance, Side: LiquidationSideShort, Price: short100.Price, Notional: 80_000, Time: t0.Add(20 * time.Minute)},
					{Exchange: ExchangeBinance, Side: LiquidationSideShort, Price: short100.Price, Notional: 10_000, Time: t0.Add(90 * time.Minute)},
				},
			},
			{
				Exchange: ExchangeBybit,
				Prices:   []HuntHeatmapPricePoint{{Time: t0, Price: 100}},
				OI:       []FuturesSnapshot{{SampledAt: t0, Value: 2_000_000}},
				Liquidations: []LiquidationEvent{
					{Exchange: ExchangeBybit, Side: LiquidationSideShort, Price: short100.Price, Notional: 999_000, Time: t0.Add(10 * time.Minute)},
				},
			},
		},
	}
	got := ReviewLiquidationLevels(in)
	if got.Note == "" || got.Summary == "" {
		t.Fatal("missing copy")
	}
	var hit100 LiquidationLevelsReviewBand
	for _, b := range got.Binance.Samples[0].Bands {
		if b.Leverage == 100 && b.Side == LiquidationSideShort {
			hit100 = b
		}
	}
	if !hit100.Hit || hit100.Status != HuntReviewHit || hit100.Realized < 80_000 {
		t.Fatalf("binance 100x short %+v", hit100)
	}
	// 1h window should not include the 90m print.
	if hit100.Realized > 90_000 {
		t.Fatalf("1h should not take the 90m print %+v", hit100)
	}

	// A long band with no matching long prints is a miss when the feed covers.
	var longMiss LiquidationLevelsReviewBand
	for _, b := range got.Binance.Samples[0].Bands {
		if b.Leverage == 100 && b.Side == LiquidationSideLong {
			longMiss = b
		}
	}
	if longMiss.Status != HuntReviewMiss || longMiss.Hit {
		t.Fatalf("expected miss on empty long band %+v", longMiss)
	}

	// Bybit's huge print must not land on Binance realized.
	if hit100.Realized >= 999_000 {
		t.Fatalf("crossed venues %+v", hit100)
	}

	pending := ReviewLiquidationLevels(LiquidationLevelsReviewInput{
		Now:  t0.Add(10 * time.Minute),
		Step: time.Hour,
		Venues: []HuntHeatmapVenueSeries{{
			Exchange: ExchangeBinance,
			Prices:   []HuntHeatmapPricePoint{{Time: t0, Price: 100}},
			OI:       []FuturesSnapshot{{SampledAt: t0, Value: 2_000_000}},
			Liquidations: []LiquidationEvent{
				{Side: LiquidationSideShort, Price: short100.Price, Notional: 1, Time: t0.Add(-time.Minute)},
			},
		}},
	})
	if pending.Binance.Horizons[0].Pending == 0 {
		t.Fatalf("expected pending 1h %+v", pending.Binance.Horizons)
	}
}

func TestReviewLiquidationLevels_WrongSideAndNoPrice(t *testing.T) {
	t0 := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
	bands := PredictDisplayLevelBands(50, 2_000_000, 0.5, 0.5, 0)
	var short100 predictedLevelBand
	for _, b := range bands {
		if b.Leverage == 100 && b.Side == LiquidationSideShort {
			short100 = b
		}
	}
	got := ReviewLiquidationLevels(LiquidationLevelsReviewInput{
		Now:  t0.Add(3 * time.Hour),
		Step: time.Hour,
		Venues: []HuntHeatmapVenueSeries{{
			Exchange: ExchangeBinance,
			Prices:   []HuntHeatmapPricePoint{{Time: t0, Price: 50}},
			OI:       []FuturesSnapshot{{SampledAt: t0, Value: 2_000_000}},
			Liquidations: []LiquidationEvent{
				{Side: LiquidationSideLong, Price: short100.Price, Notional: 50_000, Time: t0.Add(-time.Hour)},
				{Side: LiquidationSideLong, Price: short100.Price, Notional: 50_000, Time: t0.Add(10 * time.Minute)},
				{Side: LiquidationSideLong, Price: 1, Notional: 1, Time: t0.Add(2 * time.Hour)},
			},
		}},
	})
	for _, b := range got.Binance.Samples[0].Bands {
		if b.Leverage == 100 && b.Side == LiquidationSideShort && b.Hit {
			t.Fatalf("long print must not hit a short band %+v", b)
		}
	}

	empty := ReviewLiquidationLevels(LiquidationLevelsReviewInput{
		Now: t0.Add(time.Hour),
		Venues: []HuntHeatmapVenueSeries{{
			Exchange: ExchangeBinance,
			OI:       []FuturesSnapshot{{SampledAt: t0, Value: 2_000_000}},
		}},
	})
	if len(empty.Binance.Samples) != 0 {
		t.Fatalf("no last price must skip samples %+v", empty.Binance.Samples)
	}
}

func TestApplyLevelConfidence_ShrinksWeakBands(t *testing.T) {
	levels := []LiquidationLevelBar{{
		Price: "101", LongNotional: "0", ShortNotional: "100", TotalNotional: "100",
		CumShort: "100", CumTotal: "100",
		ByLeverage: []LiquidationLeverageSlice{
			{Leverage: 100, LongNotional: "0", ShortNotional: "100"},
			{Leverage: 10, LongNotional: "0", ShortNotional: "100"},
		},
		CumByLeverage: []LiquidationLeverageSlice{
			{Leverage: 100, LongNotional: "0", ShortNotional: "100"},
			{Leverage: 10, LongNotional: "0", ShortNotional: "100"},
		},
	}}
	got := ApplyLevelConfidence(levels, []LiquidationLevelsReviewBucket{
		{Leverage: 100, Side: LiquidationSideShort, Validated: 20, HitRate: 0.50, Reliability: LiquidationLevelsReliabilityHigh},
		{Leverage: 10, Side: LiquidationSideShort, Validated: 20, HitRate: 0, Reliability: LiquidationLevelsReliabilityLow},
	})
	if len(got) != 1 {
		t.Fatal(got)
	}
	var strong, weak float64
	for _, sl := range got[0].ByLeverage {
		n, _ := strconv.ParseFloat(sl.ShortNotional, 64)
		if sl.Leverage == 100 {
			strong = n
			if sl.ShortConfidence < 0.9 {
				t.Fatalf("100x conf %v", sl.ShortConfidence)
			}
		}
		if sl.Leverage == 10 {
			weak = n
			if sl.ShortConfidence > 0.25 {
				t.Fatalf("10x conf %v", sl.ShortConfidence)
			}
		}
	}
	if !(strong > weak*2) {
		t.Fatalf("strong=%v should dwarf weak=%v", strong, weak)
	}
}

func TestLevelConfidenceWeight(t *testing.T) {
	if LevelConfidenceWeight(2, 1) != 0.55 {
		t.Fatal("unproven")
	}
	if LevelConfidenceWeight(10, 0.5) < 0.95 {
		t.Fatal("high hits")
	}
	if LevelConfidenceWeight(10, 0) > 0.16 {
		t.Fatal("zero hits")
	}
}

func TestLevelReliability(t *testing.T) {
	if levelReliability(2, 1, 1) != LiquidationLevelsReliabilityUnknown {
		t.Fatal("unknown")
	}
	if levelReliability(4, 0.6, 0.2) != LiquidationLevelsReliabilityHigh {
		t.Fatal("high")
	}
	if levelReliability(4, 0.35, 0.01) != LiquidationLevelsReliabilityFair {
		t.Fatal("fair")
	}
	if levelReliability(4, 0.1, 0) != LiquidationLevelsReliabilityLow {
		t.Fatal("low")
	}
}

func TestReviewUsesTopTraderSizeWhenOpposite(t *testing.T) {
	t0 := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	base := HuntHeatmapVenueSeries{
		Exchange: ExchangeBinance,
		Prices:   []HuntHeatmapPricePoint{{Time: t0, Price: 100}},
		OI:       []FuturesSnapshot{{SampledAt: t0, Value: 2_000_000}},
	}
	crowd := base
	crowd.LongShort = []FuturesSnapshot{{SampledAt: t0, LongShare: 0.70, ShortShare: 0.30}}
	size := base
	size.LongShort = []FuturesSnapshot{{SampledAt: t0, LongShare: 0.70, ShortShare: 0.30, SizeLongShare: 0.32}}
	in := LiquidationLevelsReviewInput{Now: t0.Add(2 * time.Hour), Step: time.Hour}
	in.Venues = []HuntHeatmapVenueSeries{crowd}
	crowdRev := ReviewLiquidationLevels(in)
	in.Venues = []HuntHeatmapVenueSeries{size}
	sizeRev := ReviewLiquidationLevels(in)
	if len(crowdRev.Binance.Samples) == 0 || len(sizeRev.Binance.Samples) == 0 {
		t.Fatal("no samples")
	}
	if sizeRev.Binance.Samples[0].Blend != "top_positions" {
		t.Fatalf("blend %q", sizeRev.Binance.Samples[0].Blend)
	}
	if crowdRev.Binance.Samples[0].Lean == sizeRev.Binance.Samples[0].Lean &&
		crowdRev.Binance.Samples[0].Lean != "even" {
		// crowd 70% long vs size 32% long should change which side is heavier
		t.Fatalf("top-trader size must change the frozen lean: crowd=%s size=%s",
			crowdRev.Binance.Samples[0].Lean, sizeRev.Binance.Samples[0].Lean)
	}
}

func TestBuildLiquidationLevelsAt_KeepsShortsAboveFrozenLast(t *testing.T) {
	frozenLast := 100.0
	laterLast := 110.0
	frozen := BuildLiquidationLevelsAt(ExchangeBinance, frozenLast, 2_000_000, 0.5, 0.5, 0)
	later := BuildLiquidationLevelsAt(ExchangeBinance, laterLast, 2_000_000, 0.5, 0.5, 0)
	if len(frozen) == 0 || len(later) == 0 {
		t.Fatal("empty maps")
	}
	shortPx := frozenLast * (1 + HuntLiqDistance(100, HuntMaintenanceMargin))
	var frozenShort, laterStillShort bool
	for _, lv := range frozen {
		px, _ := strconv.ParseFloat(lv.Price, 64)
		if px < shortPx*0.997 || px > shortPx*1.003 {
			continue
		}
		s, _ := strconv.ParseFloat(lv.ShortNotional, 64)
		l, _ := strconv.ParseFloat(lv.LongNotional, 64)
		if s > l {
			frozenShort = true
		}
	}
	for _, lv := range later {
		px, _ := strconv.ParseFloat(lv.Price, 64)
		if px < shortPx*0.997 || px > shortPx*1.003 {
			continue
		}
		s, _ := strconv.ParseFloat(lv.ShortNotional, 64)
		if s > 0 {
			laterStillShort = true
		}
	}
	if !frozenShort {
		t.Fatal("frozen map must keep 100x shorts above the last that was shown")
	}
	if laterStillShort {
		t.Fatal("a later last above that pocket must move the short cluster away — that is why samples keep the frozen map")
	}
}

func TestExplainLiquidationLevelsReview_Empty(t *testing.T) {
	s := ExplainLiquidationLevelsReview(LiquidationLevelsReview{})
	if s == "" {
		t.Fatal("empty summary")
	}
}
