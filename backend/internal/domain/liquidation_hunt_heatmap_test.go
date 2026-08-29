package domain

import (
	"testing"
	"time"
)

func TestParseHuntHeatmapRange(t *testing.T) {
	s, err := ParseHuntHeatmapRange("")
	if err != nil || s.Range != HuntHeatmap24h {
		t.Fatalf("%+v %v", s, err)
	}
	s, err = ParseHuntHeatmapRange("7d")
	if err != nil || s.Window != 168*time.Hour || s.Step != 2*time.Hour {
		t.Fatalf("%+v %v", s, err)
	}
	if _, err := ParseHuntHeatmapRange("9h"); err == nil {
		t.Fatal("expected invalid range")
	}
}

func TestHuntHeatmapColumnsAligned(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("12h")
	to := time.Date(2026, 8, 29, 13, 22, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	if len(cols) != 48 {
		t.Fatalf("cols=%d", len(cols))
	}
	if cols[len(cols)-1] != time.Date(2026, 8, 29, 13, 15, 0, 0, time.UTC) {
		t.Fatalf("last=%v", cols[len(cols)-1])
	}
	if cols[1].Sub(cols[0]) != 15*time.Minute {
		t.Fatalf("step=%v", cols[1].Sub(cols[0]))
	}
}

func TestBuildHuntHeatmap_IntensityAtLeverageBand(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("24h")
	to := time.Date(2026, 8, 29, 14, 0, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	mid := 100_000.0
	prices := make([]HuntHeatmapPricePoint, 0, len(cols))
	for _, ts := range cols {
		prices = append(prices, HuntHeatmapPricePoint{Time: ts, Price: mid})
	}
	oi := make([]FuturesSnapshot, 0, len(cols))
	for _, ts := range cols {
		oi = append(oi, FuturesSnapshot{
			Metric: FuturesMetricOpenInterest, Exchange: ExchangeBinance,
			SampledAt: ts, Value: 10_000_000,
		})
	}
	got := BuildHuntHeatmap(HuntHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to, Prices: prices,
		Venues: []HuntHeatmapVenueSeries{{
			Exchange: ExchangeBinance, OI: oi,
		}},
	})
	if got.Range != "24h" || len(got.Times) != 48 || len(got.Prices) != HuntHeatmapPriceBins {
		t.Fatalf("shape times=%d prices=%d", len(got.Times), len(got.Prices))
	}
	if got.Binance.ColumnsWithOI != 48 || got.Binance.MaxIntensity <= 0 {
		t.Fatalf("binance %+v", got.Binance)
	}
	// 10x shorts liquidate ~9.6% above mid.
	dist := HuntLiqDistance(10, HuntMaintenanceMargin)
	up := mid * (1 + dist)
	idx := huntPriceBin(up, got.PriceMin, got.PriceMax, got.PriceStep, len(got.Prices))
	if idx < 0 || got.Binance.Shorts[len(got.Times)-1][idx] <= 0 {
		t.Fatalf("expected short intensity at 10x band idx=%d up=%.0f shorts=%v", idx, up, lastCol(got.Binance.Shorts))
	}
	if got.Combined.MaxIntensity < got.Binance.MaxIntensity-1 {
		t.Fatalf("combined should include binance %v vs %v", got.Combined.MaxIntensity, got.Binance.MaxIntensity)
	}
	if got.Bybit.MaxIntensity != 0 {
		t.Fatalf("bybit should be empty")
	}
}

func TestBuildHuntHeatmap_ObservedLiquidationAndCombined(t *testing.T) {
	spec, _ := ParseHuntHeatmapRange("12h")
	to := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	cols := HuntHeatmapColumns(spec, to)
	last := cols[len(cols)-1]
	prices := []HuntHeatmapPricePoint{{Time: last, Price: 50_000}}
	bin := HuntHeatmapVenueSeries{
		Exchange: ExchangeBinance,
		OI:       []FuturesSnapshot{{SampledAt: last, Value: 1_000_000}},
		Liquidations: []LiquidationEvent{{
			Exchange: ExchangeBinance, Side: LiquidationSideLong,
			Time: last.Add(-time.Minute), Price: 49_000, Notional: 80_000,
		}},
	}
	byb := HuntHeatmapVenueSeries{
		Exchange: ExchangeBybit,
		OI:       []FuturesSnapshot{{SampledAt: last, Value: 2_000_000}},
	}
	got := BuildHuntHeatmap(HuntHeatmapInput{
		Symbol: "ETHUSDT", Spec: spec, To: to, Prices: prices,
		Venues: []HuntHeatmapVenueSeries{bin, byb},
	})
	idx := huntPriceBin(49_000, got.PriceMin, got.PriceMax, got.PriceStep, len(got.Prices))
	if idx < 0 || got.Binance.Longs[len(got.Times)-1][idx] < 80_000 {
		t.Fatalf("observed long not in binance cell idx=%d v=%v", idx, lastCol(got.Binance.Longs))
	}
	if got.Combined.MaxIntensity+1 < got.Binance.MaxIntensity || got.Combined.MaxIntensity+1 < got.Bybit.MaxIntensity {
		t.Fatalf("combined max=%v bin=%v byb=%v", got.Combined.MaxIntensity, got.Binance.MaxIntensity, got.Bybit.MaxIntensity)
	}
	sumLast := 0.0
	for j := range got.Combined.Totals[len(got.Times)-1] {
		sumLast += got.Combined.Totals[len(got.Times)-1][j]
	}
	if sumLast < 3_000_000 {
		t.Fatalf("combined last-column total too small %v", sumLast)
	}
}

func lastCol(m [][]float64) []float64 {
	if len(m) == 0 {
		return nil
	}
	return m[len(m)-1]
}

func TestHuntPriceBinEdges(t *testing.T) {
	if huntPriceBin(110, 10, 110, 10, 10) != 0 {
		t.Fatal(huntPriceBin(110, 10, 110, 10, 10))
	}
	if huntPriceBin(10, 10, 110, 10, 10) != 9 {
		t.Fatal(huntPriceBin(10, 10, 110, 10, 10))
	}
	if huntPriceBin(0, 10, 110, 10, 10) != -1 {
		t.Fatal(huntPriceBin(0, 10, 110, 10, 10))
	}
}

func TestHuntHeatmapPriceExtent_FallsBackToStalePrints(t *testing.T) {
	now := time.Date(2026, 8, 29, 14, 0, 0, 0, time.UTC)
	lo, hi := huntHeatmapPriceExtent(
		[]HuntHeatmapPricePoint{{Time: time.Unix(0, 0).UTC(), Price: 50_000}},
		[]time.Time{now},
	)
	if lo <= 0 || hi <= lo {
		t.Fatalf("lo=%v hi=%v", lo, hi)
	}
}
