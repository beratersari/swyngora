package domain

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func openLevelSeries(ex Exchange, times []time.Time, price, oi float64) HuntHeatmapVenueSeries {
	ser := HuntHeatmapVenueSeries{Exchange: ex}
	for _, ts := range times {
		ser.Prices = append(ser.Prices, HuntHeatmapPricePoint{Time: ts, Price: price, High: price, Low: price})
		if oi > 0 {
			ser.OI = append(ser.OI, FuturesSnapshot{SampledAt: ts, Value: oi})
		}
	}
	return ser
}

func TestOpenLevels_FirstColumnPlacesBookAndFlatOIDoesNotAdd(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	times := []time.Time{t0, t0.Add(step), t0.Add(2 * step)}
	ser := openLevelSeries(ExchangeBinance, times, 100, 1_000_000)
	one := walkOpenLevels("binance", openLevelSeries(ExchangeBinance, times[:1], 100, 1_000_000), times[:1], step)
	three := walkOpenLevels("binance", ser, times, step)
	if !three.book.Definite || three.book.OpenLong <= 0 || three.book.OpenShort <= 0 {
		t.Fatalf("book %+v", three.book)
	}
	if math.Abs(one.book.OpenLong-three.book.OpenLong) > 1 || math.Abs(one.book.OpenShort-three.book.OpenShort) > 1 {
		t.Fatalf("flat OI changed size one=%v three=%v", one.book.OpenLong, three.book.OpenLong)
	}
	if math.Abs(three.book.OpenLong-500_000) > 1 || math.Abs(three.book.OpenShort-500_000) > 1 {
		t.Fatalf("even book long=%v short=%v", three.book.OpenLong, three.book.OpenShort)
	}
	if three.book.HitLots != 0 || three.book.ClosedLots != 0 {
		t.Fatalf("hits=%d closed=%d", three.book.HitLots, three.book.ClosedLots)
	}
	if len(three.book.Segments) == 0 || three.book.Segments[0].Status != OpenLevelStatusOpen {
		t.Fatalf("segments %+v", three.book.Segments)
	}
}

func TestOpenLevels_LaterCandleHitsLongsAndDoesNotRefill(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	times := []time.Time{t0, t0.Add(step)}
	ser := HuntHeatmapVenueSeries{Exchange: ExchangeBinance}
	ser.Prices = []HuntHeatmapPricePoint{
		{Time: times[0], Price: 100, High: 100, Low: 100},
		{Time: times[1], Price: 100, High: 100, Low: 90},
	}
	ser.OI = []FuturesSnapshot{
		{SampledAt: times[0], Value: 1_000_000},
		{SampledAt: times[1], Value: 1_000_000},
	}
	got := walkOpenLevels("binance", ser, times, step)
	if got.book.HitLots == 0 {
		t.Fatal("expected hits")
	}
	if got.book.OpenLong >= 500_000-1 {
		t.Fatalf("hit was refilled: long=%v", got.book.OpenLong)
	}
	if got.book.OpenLong <= 0 {
		t.Fatal("every long was removed")
	}
	if got.cum[0].LongNotional <= got.cum[1].LongNotional {
		t.Fatalf("cumulative did not fall: %+v", got.cum)
	}
	var deep bool
	for _, seg := range got.book.Segments {
		if seg.Status == OpenLevelStatusOpen && seg.Side == LiquidationSideLong && seg.Price < 85 {
			deep = true
		}
	}
	if !deep {
		t.Fatalf("5x long should still be open: %+v", got.book.Segments)
	}
	raw, err := json.Marshal(got.book.Segments)
	if err != nil {
		t.Fatal(err)
	}
	var wire []map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	var sawOpen, sawHit bool
	for _, seg := range wire {
		switch seg["status"] {
		case OpenLevelStatusOpen:
			sawOpen = true
			if _, ok := seg["closedAt"]; ok {
				t.Fatalf("open segment published closedAt: %v", seg["closedAt"])
			}
		case OpenLevelStatusHit:
			sawHit = true
			closed, _ := seg["closedAt"].(string)
			if closed == "" || closed[:4] == "0001" {
				t.Fatalf("hit segment missing a real closedAt: %v", seg["closedAt"])
			}
		}
	}
	if !sawOpen || !sawHit {
		t.Fatalf("expected open and hit segments in JSON, got %s", raw)
	}
}

func TestOpenLevels_BirthCandleDoesNotHitItsOwnLevel(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	times := []time.Time{t0, t0.Add(step)}
	ser := HuntHeatmapVenueSeries{Exchange: ExchangeBinance}
	ser.Prices = []HuntHeatmapPricePoint{
		{Time: times[0], Price: 100, High: 100, Low: 50},
		{Time: times[1], Price: 100, High: 100.5, Low: 99},
	}
	ser.OI = []FuturesSnapshot{
		{SampledAt: times[0], Value: 1_000_000},
		{SampledAt: times[1], Value: 1_000_000},
	}
	got := walkOpenLevels("binance", ser, times, step)
	var deep bool
	for _, seg := range got.book.Segments {
		if seg.Status == OpenLevelStatusOpen && seg.Side == LiquidationSideLong && seg.Price < 85 {
			deep = true
		}
	}
	if !deep {
		t.Fatalf("birth wick removed the 5x long: %+v", got.book.Segments)
	}
	if got.book.HitLots == 0 {
		t.Fatal("later candle should still hit the nearest long")
	}
}

func TestOpenLevels_OIDropClosesOldestWithoutAHit(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	times := []time.Time{t0, t0.Add(step)}
	ser := HuntHeatmapVenueSeries{Exchange: ExchangeBinance}
	ser.Prices = []HuntHeatmapPricePoint{
		{Time: times[0], Price: 100, High: 100, Low: 100},
		{Time: times[1], Price: 100, High: 100, Low: 100},
	}
	ser.OI = []FuturesSnapshot{
		{SampledAt: times[0], Value: 1_000_000},
		{SampledAt: times[1], Value: 500_000},
	}
	got := walkOpenLevels("binance", ser, times, step)
	if got.book.HitLots != 0 {
		t.Fatalf("hits=%d", got.book.HitLots)
	}
	if got.book.ClosedLots == 0 {
		t.Fatal("expected closed lots")
	}
	if math.Abs(got.book.OpenLong-250_000) > 1 || math.Abs(got.book.OpenShort-250_000) > 1 {
		t.Fatalf("long=%v short=%v", got.book.OpenLong, got.book.OpenShort)
	}
}

func TestOpenLevels_OIIncreasePlacesOnlyTheIncrement(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(15 * time.Minute)
	times := []time.Time{t0, t1}
	ser := HuntHeatmapVenueSeries{Exchange: ExchangeBinance}
	ser.Prices = []HuntHeatmapPricePoint{
		{Time: t0, Price: 100, High: 100, Low: 100},
		{Time: t1, Price: 110, High: 110, Low: 110},
	}
	ser.OI = []FuturesSnapshot{
		{SampledAt: t0, Value: 1_000_000},
		{SampledAt: t1, Value: 1_500_000},
	}
	got := walkOpenLevels("binance", ser, times, 15*time.Minute)
	if math.Abs(got.book.OpenLong-750_000) > 1 {
		t.Fatalf("long=%v", got.book.OpenLong)
	}
	var bornLater bool
	var keptEarly bool
	for _, seg := range got.book.Segments {
		if seg.Status != OpenLevelStatusOpen || seg.Side != LiquidationSideLong {
			continue
		}
		if seg.OpenedAt.Equal(t1) {
			bornLater = true
		}
		if seg.OpenedAt.Equal(t0) && seg.Price < 100 {
			keptEarly = true
		}
	}
	if !bornLater || !keptEarly {
		t.Fatalf("born=%v kept=%v segs=%+v", bornLater, keptEarly, got.book.Segments)
	}
}

func TestOpenLevels_ShortHitWhenHighReachesIt(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(15 * time.Minute)
	times := []time.Time{t0, t1}
	ser := HuntHeatmapVenueSeries{Exchange: ExchangeBinance}
	ser.Prices = []HuntHeatmapPricePoint{
		{Time: t0, Price: 100, High: 100, Low: 100},
		{Time: t1, Price: 101, High: 110, Low: 100},
	}
	ser.OI = []FuturesSnapshot{
		{SampledAt: t0, Value: 1_000_000},
		{SampledAt: t1, Value: 1_000_000},
	}
	got := walkOpenLevels("binance", ser, times, 15*time.Minute)
	if got.book.OpenShort >= 500_000-1 {
		t.Fatalf("short was not reduced: %v", got.book.OpenShort)
	}
	var hitShort bool
	for _, seg := range got.book.Segments {
		if seg.Status == OpenLevelStatusHit && seg.Side == LiquidationSideShort {
			hitShort = true
		}
	}
	if !hitShort {
		t.Fatalf("no short hit: %+v", got.book.Segments)
	}
}

func TestOpenLevels_CapsDrawnOpenSegments(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	step := time.Hour
	const n = 30
	times := make([]time.Time, n)
	ser := HuntHeatmapVenueSeries{Exchange: ExchangeBinance}
	price := 100.0
	oi := 1_000_000.0
	for i := 0; i < n; i++ {
		times[i] = t0.Add(time.Duration(i) * step)
		ser.Prices = append(ser.Prices, HuntHeatmapPricePoint{Time: times[i], Price: price, High: price, Low: price})
		ser.OI = append(ser.OI, FuturesSnapshot{SampledAt: times[i], Value: oi})
		price *= 1.01
		oi *= 1.2
	}
	got := walkOpenLevels("binance", ser, times, step)
	openN := 0
	for _, seg := range got.book.Segments {
		if seg.Status == OpenLevelStatusOpen {
			openN++
		}
	}
	if openN != openLevelMaxOpen {
		t.Fatalf("open segments=%d", openN)
	}
	if got.book.OpenLong <= 500_000 {
		t.Fatalf("uncapped long was capped: %v", got.book.OpenLong)
	}
}

func TestBuildOpenLevels_WithholdsCombinedWhenVenueMissing(t *testing.T) {
	spec, err := ParseHuntHeatmapRange("12h")
	if err != nil {
		t.Fatal(err)
	}
	to := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	times := HuntHeatmapColumns(spec, to)
	got := BuildOpenLevels(OpenLevelsInput{
		Symbol: "btcusdt",
		Spec:   spec,
		To:     to,
		Venues: []HuntHeatmapVenueSeries{
			openLevelSeries(ExchangeBinance, times, 100, 1_000_000),
			openLevelSeries(ExchangeBybit, times, 100, 1_000_000),
		},
	})
	if got.Symbol != "BTCUSDT" || got.Range != "12h" || got.Note == "" {
		t.Fatalf("report header %+v", got.Symbol)
	}
	if len(got.LeverageMix) != len(DefaultHuntLeverageMix) {
		t.Fatalf("mix %d", len(got.LeverageMix))
	}
	if got.Combined.Definite || len(got.Combined.Segments) != 0 || len(got.Combined.Cumulative) != 0 {
		t.Fatalf("combined leaked %+v", got.Combined)
	}
	found := false
	for _, id := range got.MissingVenues {
		if id == "okx" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing %+v", got.MissingVenues)
	}
	if !got.Binance.Definite || got.Binance.OpenLong <= 0 {
		t.Fatalf("binance %+v", got.Binance)
	}
}

func TestBuildOpenLevels_CombinedSumsOverlapOnly(t *testing.T) {
	spec, err := ParseHuntHeatmapRange("12h")
	if err != nil {
		t.Fatal(err)
	}
	to := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	times := HuntHeatmapColumns(spec, to)
	full := func(ex Exchange) HuntHeatmapVenueSeries {
		return openLevelSeries(ex, times, 100, 1_000_000)
	}
	okx := full(ExchangeOKX)
	if len(times) > 2 {
		okx.OI = okx.OI[:1]
		okx.Prices = okx.Prices[:1]
	}
	got := BuildOpenLevels(OpenLevelsInput{
		Symbol: "BTCUSDT",
		Spec:   spec,
		To:     to,
		Venues: []HuntHeatmapVenueSeries{full(ExchangeBinance), full(ExchangeBybit), okx},
	})
	if !got.Combined.Definite {
		t.Fatal("expected an overlap")
	}
	if len(got.Combined.Cumulative) >= len(times) {
		t.Fatalf("combined kept columns OKX missed: %d of %d", len(got.Combined.Cumulative), len(times))
	}
	all := BuildOpenLevels(OpenLevelsInput{
		Symbol: "BTCUSDT",
		Spec:   spec,
		To:     to,
		Venues: []HuntHeatmapVenueSeries{full(ExchangeBinance), full(ExchangeBybit), full(ExchangeOKX)},
	})
	want := all.Binance.Delta + all.Bybit.Delta + all.OKX.Delta
	if math.Abs(all.Combined.Delta-want) > 1 {
		t.Fatalf("combined delta=%v want=%v", all.Combined.Delta, want)
	}
	if !all.Combined.Definite || len(all.MissingVenues) != 0 {
		t.Fatalf("missing %+v definite=%v", all.MissingVenues, all.Combined.Definite)
	}
}

func TestOpenLevelAnchor_UsesTheLastColumnAtOrBeforeTheClock(t *testing.T) {
	points := []OpenLevelPoint{
		{Time: time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC), LongNotional: 100, ShortNotional: 80, Delta: 20},
		{Time: time.Date(2026, 8, 11, 12, 30, 0, 0, time.UTC), LongNotional: 60, ShortNotional: 90, Delta: -30},
	}
	day := openLevelAnchor(points, OpenLevelAnchorDay, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC))
	if day.Complete || day.LongChange != 0 {
		t.Fatalf("day should stay blank: %+v", day)
	}
	block := openLevelAnchor(points, OpenLevelAnchor4h, time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC))
	if !block.Complete || block.BaselineLong != 100 || block.OpenLong != 60 {
		t.Fatalf("baseline %+v", block)
	}
	if block.LongChange != -40 || block.ShortChange != 10 || block.DeltaChange != -50 {
		t.Fatalf("change %+v", block)
	}
	click := openLevelAnchor(points, OpenLevelAnchorClick, time.Date(2026, 8, 11, 12, 30, 0, 0, time.UTC))
	if !click.Complete || click.LongChange != 0 || click.ShortChange != 0 || click.DeltaChange != 0 {
		t.Fatalf("click %+v", click)
	}
}

func TestBuildOpenLevels_HitAfterTheClockChangesStillOpenSize(t *testing.T) {
	spec, err := ParseHuntHeatmapRange("12h")
	if err != nil {
		t.Fatal(err)
	}
	to := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	times := HuntHeatmapColumns(spec, to)
	click := time.Date(2026, 8, 2, 6, 0, 0, 0, time.UTC)
	flat := BuildOpenLevels(OpenLevelsInput{
		Symbol: "BTCUSDT",
		Spec:   spec,
		To:     to,
		Click:  click,
		Venues: []HuntHeatmapVenueSeries{openLevelSeries(ExchangeBinance, times, 100, 1_000_000)},
	})
	if len(flat.Binance.Anchors) != 3 {
		t.Fatalf("anchors %+v", flat.Binance.Anchors)
	}
	if flat.Binance.Anchors[0].Anchor != OpenLevelAnchorDay || flat.Binance.Anchors[0].Complete {
		t.Fatalf("12h book starts after midnight: %+v", flat.Binance.Anchors[0])
	}
	if flat.Binance.Anchors[1].Anchor != OpenLevelAnchor4h || !flat.Binance.Anchors[1].Complete || flat.Binance.Anchors[1].DeltaChange != 0 {
		t.Fatalf("flat 4h %+v", flat.Binance.Anchors[1])
	}
	if flat.Binance.Anchors[2].Anchor != OpenLevelAnchorClick || !flat.Binance.Anchors[2].Complete || flat.Binance.Anchors[2].LongChange != 0 {
		t.Fatalf("flat click %+v", flat.Binance.Anchors[2])
	}
	if flat.Combined.Definite || len(flat.Combined.Anchors) != 0 {
		t.Fatalf("withheld combined grew anchors %+v", flat.Combined.Anchors)
	}

	t0 := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	step := 15 * time.Minute
	hitTimes := []time.Time{t0, t0.Add(step)}
	ser := HuntHeatmapVenueSeries{Exchange: ExchangeBinance}
	ser.Prices = []HuntHeatmapPricePoint{
		{Time: hitTimes[0], Price: 100, High: 100, Low: 100},
		{Time: hitTimes[1], Price: 100, High: 100, Low: 90},
	}
	ser.OI = []FuturesSnapshot{
		{SampledAt: hitTimes[0], Value: 1_000_000},
		{SampledAt: hitTimes[1], Value: 1_000_000},
	}
	walked := walkOpenLevels("binance", ser, hitTimes, step)
	attachOpenLevelAnchors(&walked.book, t0.Add(30*time.Minute), time.Time{})
	var block OpenLevelAnchor
	for _, anchor := range walked.book.Anchors {
		if anchor.Anchor == OpenLevelAnchor4h {
			block = anchor
		}
		if anchor.Anchor == OpenLevelAnchorDay && anchor.Complete {
			t.Fatalf("day open is before this book: %+v", anchor)
		}
	}
	if !block.Complete || block.LongChange >= 0 || block.ShortChange != 0 {
		t.Fatalf("hit should shrink still-open longs only: %+v cum=%+v", block, walked.cum)
	}
}
