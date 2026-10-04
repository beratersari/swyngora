package domain

import (
	"testing"
	"time"
)

func TestBuildHuntHeatmap_KindModel(t *testing.T) {
	got := BuildHuntHeatmap(HuntHeatmapInput{})
	if got.Kind != HuntHeatmapKindModel {
		t.Fatalf("kind %q", got.Kind)
	}
	if len(got.LeverageMix) == 0 {
		t.Fatal("model keeps the leverage mix")
	}
}

func TestPickHuntHeatmapGrid_OKX(t *testing.T) {
	rep := HuntHeatmapReport{
		OKX:      HuntHeatmapGrid{Exchange: "okx", MaxIntensity: 3},
		Combined: HuntHeatmapGrid{Exchange: "combined", MaxIntensity: 9},
	}
	got := PickHuntHeatmapGrid(rep, "okx")
	if got.Exchange != "okx" || got.MaxIntensity != 3 {
		t.Fatalf("%+v", got)
	}
	if PickHuntHeatmapGrid(rep, "all").Exchange != "combined" {
		t.Fatal("combined")
	}
}

func TestMeasuredPrintHeatmap_PrintNotionalAndCoverage(t *testing.T) {
	spec, err := ParseHuntHeatmapRange("12h")
	if err != nil {
		t.Fatal(err)
	}
	to := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	times, origin, until := MeasuredHeatmapWindow(spec, to)
	if len(times) != 48 {
		t.Fatalf("columns %d", len(times))
	}
	if !until.Equal(times[len(times)-1].Add(spec.Step / 2)) {
		t.Fatalf("aligned until %s", until)
	}
	if MeasuredTimeIndex(times[0], origin, spec.Step) != 0 {
		t.Fatal("first column")
	}
	last := len(times) - 1
	if MeasuredTimeIndex(times[last], origin, spec.Step) != last {
		t.Fatal("last column")
	}
	const pMin, pMax = 100.0, 148.0
	px := 147.6
	if MeasuredPriceIndex(px, pMin, pMax, HuntHeatmapPriceBins) != 0 {
		t.Fatal("price bin")
	}
	q := MeasuredPrintQuery{
		Symbol: "BTCUSDT", Origin: origin, Until: until, Step: spec.Step,
		Columns: len(times), PriceMin: pMin, PriceMax: pMax, PriceBins: HuntHeatmapPriceBins,
	}
	early := LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: px, Notional: 999, Time: origin.Add(-time.Millisecond),
	}
	missAxis := LiquidationEvent{
		Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: LiquidationSideShort,
		Price: 50, Notional: 10, Time: times[2],
	}
	qFlat := q
	qFlat.PriceMax = qFlat.PriceMin
	flat := BinMeasuredPrintEvents(qFlat, []LiquidationEvent{missAxis})
	if len(flat.Bins) != 0 || !flat.Latest[ExchangeBybit].Equal(times[2]) {
		t.Fatalf("flat axis %+v", flat)
	}
	scan := BinMeasuredPrintEvents(q, []LiquidationEvent{
		early,
		{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: "sell",
			Price: px, Notional: 1000, Time: times[2],
		},
		{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
			Price: px, Notional: 250, Time: times[2].Add(time.Second),
		},
		{
			Exchange: ExchangeOKX, Symbol: "BTCUSDT", Side: LiquidationSideShort,
			Price: px, Notional: 500, Time: times[2],
		},
		{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
			Price: px, Notional: 40, Time: until,
		},
	})
	if _, ok := scan.Latest[ExchangeBinance]; !ok || len(scan.Bins) != 2 {
		t.Fatalf("scan %+v", scan)
	}
	var longN float64
	var longCount int
	for _, b := range scan.Bins {
		if b.Exchange == ExchangeBinance {
			if b.Side != LiquidationSideLong || b.TimeIndex != 2 || b.PriceIndex != 0 {
				t.Fatalf("bin %+v", b)
			}
			longN += b.Notional
			longCount += b.Count
		}
	}
	if longN != 1250 || longCount != 2 {
		t.Fatalf("long notional %v count %d", longN, longCount)
	}

	watch := times[0].Add(-time.Hour)
	got := BuildMeasuredLiquidationHeatmap(MeasuredHeatmapInput{
		Symbol: "btc-usd", Spec: spec, To: to, PriceMin: pMin, PriceMax: pMax,
		Bins: scan.Bins,
		Coverage: []LiquidationCoverage{
			{Exchange: ExchangeBybit, FirstWatch: watch},
			{Exchange: ExchangeBinance, Symbol: "BTCUSDT", FirstWatch: times[20], Gaps: nil},
			{Exchange: ExchangeBinance, Symbol: "", Gaps: []LiquidationGap{{From: times[4], To: times[4].Add(time.Minute)}}},
		},
	})
	if got.Kind != HuntHeatmapKindPrints || got.Symbol != "BTCUSDT" || len(got.LeverageMix) != 0 {
		t.Fatalf("kind %s symbol %s mix %d", got.Kind, got.Symbol, len(got.LeverageMix))
	}
	if len(got.Review.Binance.Horizons) != 0 || len(got.Review.OKX.Horizons) != 0 {
		t.Fatal("print review must not invent horizons")
	}
	if got.Binance.AccountBlend != "" || got.Binance.Longs[2][0] != 1250 || got.Binance.Shorts[2][0] != 0 {
		t.Fatalf("binance cell %+v", got.Binance.Longs[2])
	}
	if !got.Binance.HasData[2] || got.Binance.Totals[2][0] != 1250 || got.Binance.MaxIntensity != 1250 {
		t.Fatalf("binance recorded max %v has %v", got.Binance.MaxIntensity, got.Binance.HasData[2])
	}
	// A print before the pair watch is still recorded. A later quiet column is not.
	if !got.Binance.HasData[2] || got.Binance.HasData[3] {
		t.Fatalf("watch hole has2 %v has3 %v", got.Binance.HasData[2], got.Binance.HasData[3])
	}
	// Column center inside a venue gap with no print stays empty. The print column does not.
	if got.Binance.HasData[4] {
		t.Fatal("gap column should stay empty")
	}
	if !got.Bybit.HasData[3] || got.Bybit.Totals[3][0] != 0 || got.Bybit.ColumnsWithOI != len(times) {
		t.Fatalf("quiet bybit covered %v cols %d", got.Bybit.HasData[3], got.Bybit.ColumnsWithOI)
	}
	if !got.OKX.HasData[2] || got.OKX.Shorts[2][0] != 500 || got.OKX.HasData[3] {
		t.Fatalf("okx print-only has2 %v short %v has3 %v", got.OKX.HasData[2], got.OKX.Shorts[2][0], got.OKX.HasData[3])
	}
	if !got.Combined.HasData[2] || got.Combined.Totals[2][0] != 1750 {
		t.Fatalf("combined col2 has %v total %v", got.Combined.HasData[2], got.Combined.Totals[2][0])
	}
	if got.Combined.HasData[3] || got.Combined.Totals[3][0] != 0 {
		t.Fatalf("combined must skip the OKX hole col3 has %v total %v", got.Combined.HasData[3], got.Combined.Totals[3][0])
	}
	for _, name := range got.MissingVenues {
		if name == "bybit" {
			t.Fatalf("bybit is covered, missing %v", got.MissingVenues)
		}
	}

	// A print inside the gap is still that column's measured notional.
	gapPrint := BinMeasuredPrintEvents(q, []LiquidationEvent{{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideShort,
		Price: px, Notional: 80, Time: times[4],
	}})
	gapped := BuildMeasuredLiquidationHeatmap(MeasuredHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to, PriceMin: pMin, PriceMax: pMax,
		Bins: gapPrint.Bins,
		Coverage: []LiquidationCoverage{{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", FirstWatch: watch,
			Gaps: []LiquidationGap{
				{From: times[4], To: times[4].Add(time.Minute)},
				{From: times[6], To: times[6].Add(time.Minute)},
			},
		}},
	})
	if !gapped.Binance.HasData[4] || gapped.Binance.Shorts[4][0] != 80 {
		t.Fatalf("gap print has4 %v short %v", gapped.Binance.HasData[4], gapped.Binance.Shorts[4][0])
	}
	if !gapped.Binance.HasData[5] || gapped.Binance.Totals[5][0] != 0 || gapped.Binance.HasData[6] {
		t.Fatalf("quiet col5 %v gap col6 %v", gapped.Binance.HasData[5], gapped.Binance.HasData[6])
	}

	one := BuildMeasuredLiquidationHeatmap(MeasuredHeatmapInput{
		Symbol: "BTCUSDT", Spec: spec, To: to, PriceMin: pMin, PriceMax: pMax,
		Venues: []Exchange{ExchangeBinance},
		Bins:   scan.Bins,
	})
	if one.Binance.Longs[2][0] != 1250 || one.Bybit.ColumnsWithOI != 0 || one.OKX.ColumnsWithOI != 0 {
		t.Fatal("single venue must not fill the others")
	}
	if one.Combined.HasData[2] {
		t.Fatal("combined stays empty while bybit and okx were not requested")
	}
}

func TestMeasuredPrintPriceExtent_SmallPad(t *testing.T) {
	lo, hi := MeasuredPrintPriceExtent(9000, 10000)
	if lo >= 9000 || hi <= 10000 {
		t.Fatalf("%v %v", lo, hi)
	}
	if hi-10000 > 10000*0.02 {
		t.Fatalf("pad too wide %v %v", lo, hi)
	}
	if _, hi0 := MeasuredPrintPriceExtent(0, 0); hi0 != 0 {
		t.Fatal("empty")
	}
}

func TestMeasuredPrintHeatmap_OpenTailStaysOnLastColumn(t *testing.T) {
	spec, err := ParseHuntHeatmapRange("12h")
	if err != nil {
		t.Fatal(err)
	}
	to := time.Date(2026, 8, 30, 16, 10, 0, 0, time.UTC)
	times, origin, until := MeasuredHeatmapWindow(spec, to)
	if len(times) != 48 || !until.Equal(to) {
		t.Fatalf("columns %d until %s", len(times), until)
	}
	closed := times[len(times)-1].Add(spec.Step / 2)
	tail := closed.Add(time.Minute)
	if !tail.Before(until) || !tail.After(closed) {
		t.Fatalf("tail %s closed %s until %s", tail, closed, until)
	}
	if MeasuredTimeIndex(tail, origin, spec.Step) != len(times) {
		t.Fatalf("raw index %d", MeasuredTimeIndex(tail, origin, spec.Step))
	}
	const pMin, pMax = 100.0, 148.0
	q := MeasuredPrintQuery{
		Symbol: "BTCUSDT", Origin: origin, Until: until, Step: spec.Step,
		Columns: len(times), PriceMin: pMin, PriceMax: pMax, PriceBins: HuntHeatmapPriceBins,
	}
	scan := BinMeasuredPrintEvents(q, []LiquidationEvent{{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 147.6, Notional: 80, Time: tail,
	}})
	if len(scan.Bins) != 1 || scan.Bins[0].TimeIndex != len(times)-1 || scan.Bins[0].Notional != 80 {
		t.Fatalf("%+v", scan.Bins)
	}
	beyond := BinMeasuredPrintEvents(q, []LiquidationEvent{{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
		Price: 147.6, Notional: 5, Time: until.Add(spec.Step),
	}})
	if len(beyond.Bins) != 0 {
		t.Fatalf("past the open tail %+v", beyond.Bins)
	}
}
