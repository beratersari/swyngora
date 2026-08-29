package domain

import (
	"testing"
	"time"
)

func TestHuntHotZones_SkipsMarkAndClusters(t *testing.T) {
	nBin := 10
	pMin, pMax, step := 90.0, 110.0, 2.0
	col := make([]float64, nBin)
	col[0] = 10 // top, away from mark
	col[1] = 10
	col[8] = 10 // near mark at 91
	col[9] = 10
	zones := huntHotZones(col, pMin, pMax, step, nBin, []float64{91})
	if len(zones) != 1 {
		t.Fatalf("zones=%d %+v", len(zones), zones)
	}
	if zones[0].hi < 107 || zones[0].lo > 107 {
		t.Fatalf("cluster %+v", zones[0])
	}
}

func TestReviewHuntHeatmap_HitMissAndLiqIncrease(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("24h")
	to := time.Date(2026, 8, 29, 18, 0, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	mid := 100_000.0
	var prices []HuntHeatmapPricePoint
	var oi []FuturesSnapshot
	for _, ts := range cols {
		prices = append(prices, HuntHeatmapPricePoint{Time: ts, Price: mid, High: mid, Low: mid})
		oi = append(oi, FuturesSnapshot{SampledAt: ts, Value: 10_000_000})
	}
	dist := HuntLiqDistance(10, HuntMaintenanceMargin)
	up := mid * (1 + dist)
	// After the first column, walk price into the short-liq band.
	t0 := cols[0]
	hitAt := t0.Add(45 * time.Minute)
	prices = append(prices, HuntHeatmapPricePoint{
		Time: hitAt, Price: up, High: up + 50, Low: up - 50,
	})
	liqs := append(reviewCoverLiqs(cols, to), LiquidationEvent{
		Exchange: ExchangeBinance, Side: LiquidationSideShort,
		Time: t0.Add(50 * time.Minute), Price: up, Notional: 200_000,
	})
	got := BuildHuntHeatmap(HuntHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to,
		Venues: []HuntHeatmapVenueSeries{{
			Exchange: ExchangeBinance, Prices: prices, OI: oi, Liquidations: liqs,
		}},
	})
	h1 := horizonByID(got.Review.Binance.Horizons, "1h")
	if h1.Validated == 0 || h1.Hits == 0 {
		t.Fatalf("expected validated hits %+v review=%+v", h1, got.Review.Binance)
	}
	if h1.HitRate != float64(h1.Hits)/float64(h1.Validated) {
		t.Fatalf("hit rate should use validated only %+v", h1)
	}
	if h1.HitRate <= 0 || h1.AvgTimeToHitSec <= 0 || h1.AvgTimeToHitSec > 3600 {
		t.Fatalf("time/rate %+v", h1)
	}
	if h1.LiqIncreased == 0 || h1.LiqIncreaseRate <= 0 {
		t.Fatalf("expected liq increase %+v", h1)
	}
	h4 := horizonByID(got.Review.Binance.Horizons, "4h")
	if h4.Hits < h1.Hits {
		t.Fatalf("4h should include 1h hits 1h=%+v 4h=%+v", h1, h4)
	}
}

func TestReviewHuntHeatmap_FalseSignalWhenPriceNeverReaches(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("24h")
	to := time.Date(2026, 8, 29, 18, 0, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	mid := 100_000.0
	var prices []HuntHeatmapPricePoint
	var oi []FuturesSnapshot
	for _, ts := range cols {
		// Flat path — never walks into the 10x short band.
		prices = append(prices, HuntHeatmapPricePoint{Time: ts, Price: mid, High: mid + 20, Low: mid - 20})
		oi = append(oi, FuturesSnapshot{SampledAt: ts, Value: 10_000_000})
	}
	got := BuildHuntHeatmap(HuntHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to,
		Venues: []HuntHeatmapVenueSeries{{
			Exchange: ExchangeBinance, Prices: prices, OI: oi, Liquidations: reviewCoverLiqs(cols, to),
		}},
	})
	h1 := horizonByID(got.Review.Binance.Horizons, "1h")
	if h1.Validated == 0 || h1.FalseSignals == 0 {
		t.Fatalf("expected validated false signals %+v", h1)
	}
	if h1.Hits != 0 || h1.HitRate != 0 {
		t.Fatalf("unexpected hits %+v", h1)
	}
	if h1.Signals != h1.Validated+h1.Missing {
		t.Fatalf("signals should split validated+missing %+v", h1)
	}
}

func TestReviewHuntHeatmap_CombinedUsesEitherVenuePath(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("24h")
	to := time.Date(2026, 8, 29, 18, 0, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	mid := 100_000.0
	var binP, bybP []HuntHeatmapPricePoint
	var oi []FuturesSnapshot
	for _, ts := range cols {
		binP = append(binP, HuntHeatmapPricePoint{Time: ts, Price: mid, High: mid, Low: mid})
		bybP = append(bybP, HuntHeatmapPricePoint{Time: ts, Price: mid, High: mid, Low: mid})
		oi = append(oi, FuturesSnapshot{SampledAt: ts, Value: 6_000_000})
	}
	dist := HuntLiqDistance(10, HuntMaintenanceMargin)
	up := mid * (1 + dist)
	// Only Bybit later trades through the band.
	bybP = append(bybP, HuntHeatmapPricePoint{
		Time: cols[0].Add(30 * time.Minute), Price: up, High: up, Low: up,
	})
	got := BuildHuntHeatmap(HuntHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to,
		Venues: []HuntHeatmapVenueSeries{
			{Exchange: ExchangeBinance, Prices: binP, OI: oi, Liquidations: reviewCoverLiqs(cols, to)},
			{Exchange: ExchangeBybit, Prices: bybP, OI: oi, Liquidations: reviewCoverLiqs(cols, to)},
		},
	})
	comb := horizonByID(got.Review.Combined.Horizons, "1h")
	if comb.Hits == 0 {
		t.Fatalf("combined should hit via bybit path %+v bin=%+v byb=%+v",
			comb, horizonByID(got.Review.Binance.Horizons, "1h"), horizonByID(got.Review.Bybit.Horizons, "1h"))
	}
}

func TestReviewHuntHeatmap_RecentColumnsPending(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("24h")
	to := time.Date(2026, 8, 29, 18, 0, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	mid := 100_000.0
	var prices []HuntHeatmapPricePoint
	var oi []FuturesSnapshot
	for _, ts := range cols {
		prices = append(prices, HuntHeatmapPricePoint{Time: ts, Price: mid, High: mid, Low: mid})
		oi = append(oi, FuturesSnapshot{SampledAt: ts, Value: 8_000_000})
	}
	got := BuildHuntHeatmap(HuntHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to,
		Venues: []HuntHeatmapVenueSeries{{
			Exchange: ExchangeBinance, Prices: prices, OI: oi, Liquidations: reviewCoverLiqs(cols, to),
		}},
	})
	h12 := horizonByID(got.Review.Binance.Horizons, "12h")
	if h12.Pending == 0 {
		t.Fatalf("12h horizon should leave recent columns pending %+v", h12)
	}
	h1 := horizonByID(got.Review.Binance.Horizons, "1h")
	if h1.Pending == 0 {
		t.Fatalf("1h horizon should leave the last hour pending %+v", h1)
	}
}

func TestReviewHuntHeatmap_MissingLiqIsNotValidated(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("24h")
	to := time.Date(2026, 8, 29, 18, 0, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	mid := 100_000.0
	var prices []HuntHeatmapPricePoint
	var oi []FuturesSnapshot
	for _, ts := range cols {
		prices = append(prices, HuntHeatmapPricePoint{Time: ts, Price: mid, High: mid, Low: mid})
		oi = append(oi, FuturesSnapshot{SampledAt: ts, Value: 8_000_000})
	}
	got := BuildHuntHeatmap(HuntHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to,
		Venues: []HuntHeatmapVenueSeries{{
			Exchange: ExchangeBinance, Prices: prices, OI: oi,
		}},
	})
	h1 := horizonByID(got.Review.Binance.Horizons, "1h")
	if h1.Signals == 0 || h1.LiqMissing == 0 {
		t.Fatalf("expected liq gaps %+v", h1)
	}
	if h1.Validated != 0 || h1.Hits != 0 || h1.FalseSignals != 0 {
		t.Fatalf("unvalidated must not score hit rate %+v", h1)
	}
	if h1.HitRate != 0 || h1.LiqIncreaseRate != 0 {
		t.Fatalf("rates must stay 0 without validated rows %+v", h1)
	}
	if h1.Missing != h1.Signals {
		t.Fatalf("all signals should be missing %+v", h1)
	}
}

func TestReviewHuntHeatmap_MissingPriceIsNotValidated(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("24h")
	to := time.Date(2026, 8, 29, 18, 0, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	t0 := cols[0]
	mid := 100_000.0
	// Only the signal column has a print — no forward path.
	got := BuildHuntHeatmap(HuntHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to,
		Venues: []HuntHeatmapVenueSeries{{
			Exchange:     ExchangeBinance,
			Prices:       []HuntHeatmapPricePoint{{Time: t0, Price: mid, High: mid, Low: mid}},
			OI:           []FuturesSnapshot{{SampledAt: t0, Value: 8_000_000}},
			Liquidations: reviewCoverLiqs(cols, to),
		}},
	})
	h1 := horizonByID(got.Review.Binance.Horizons, "1h")
	if h1.Signals == 0 || h1.PriceMissing == 0 {
		t.Fatalf("expected price gaps %+v", h1)
	}
	if h1.Validated != 0 || h1.HitRate != 0 {
		t.Fatalf("missing price must not enter hit rate %+v", h1)
	}
}

func reviewCoverLiqs(cols []time.Time, to time.Time) []LiquidationEvent {
	start := to.Add(-36 * time.Hour)
	if len(cols) > 0 {
		start = cols[0].Add(-12 * time.Hour)
	}
	return []LiquidationEvent{
		{Time: start, Price: 1, Notional: 1, Side: LiquidationSideLong},
		{Time: to, Price: 1, Notional: 1, Side: LiquidationSideLong},
	}
}

func horizonByID(hs []HuntHeatmapReviewHorizon, id string) HuntHeatmapReviewHorizon {
	for _, h := range hs {
		if h.Horizon == id {
			return h
		}
	}
	return HuntHeatmapReviewHorizon{}
}
