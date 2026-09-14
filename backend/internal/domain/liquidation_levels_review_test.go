package domain

import (
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

func TestExplainLiquidationLevelsReview_Empty(t *testing.T) {
	s := ExplainLiquidationLevelsReview(LiquidationLevelsReview{})
	if s == "" {
		t.Fatal("empty summary")
	}
}
