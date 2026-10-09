package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBuildCVDSeries_AccumulatesAndAbsorption(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	var buckets []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 12; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		buckets = append(buckets, TakerBucket{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Start: at,
			BuyNotional: 100, SellNotional: 20,
		})
		prices = append(prices, CVDPrice{Time: at, Close: 64000})
	}
	now := t0.Add(12 * 5 * time.Minute)
	got := BuildCVDSeries(ExchangeBinance, "BTCUSDT", buckets, prices, now, t0)
	if len(got.Points) != 12 {
		t.Fatalf("points %d", len(got.Points))
	}
	if got.LastCVD != 12*80 {
		t.Fatalf("cvd %v", got.LastCVD)
	}
	var w1 CVDWindowStat
	for _, w := range got.Windows {
		if w.Window == CVDWindow1h {
			w1 = w
		}
	}
	if w1.VsPrice != CVDVsAbsorption {
		t.Fatalf("want absorption, got %+v", w1)
	}
	if !strings.Contains(w1.Summary, "absorption") {
		t.Fatalf("summary %s", w1.Summary)
	}
}

func TestBuildCVDSeries_OppositeVsPrice(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	var buckets []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 12; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		buckets = append(buckets, TakerBucket{
			Exchange: ExchangeBinance, Symbol: "ETHUSDT", Start: at,
			BuyNotional: 20, SellNotional: 100,
		})
		prices = append(prices, CVDPrice{Time: at, Close: 3000 + float64(i)*5})
	}
	now := t0.Add(12 * 5 * time.Minute)
	got := BuildCVDSeries(ExchangeBinance, "ETHUSDT", buckets, prices, now, t0)
	var w1 CVDWindowStat
	for _, w := range got.Windows {
		if w.Window == CVDWindow1h {
			w1 = w
		}
	}
	if w1.VsPrice != CVDVsOpposite {
		t.Fatalf("want opposite, got %+v", w1)
	}
}

func TestCombineCVDVenues_SumsDelta(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	a := BuildCVDSeries(ExchangeBinance, "BTCUSDT", []TakerBucket{
		{Start: t0, BuyNotional: 100, SellNotional: 0},
	}, []CVDPrice{{Time: t0, Close: 1}}, t0.Add(5*time.Minute), t0)
	b := BuildCVDSeries(ExchangeBybit, "BTCUSDT", []TakerBucket{
		{Start: t0, BuyNotional: 50, SellNotional: 10},
	}, []CVDPrice{{Time: t0, Close: 1}}, t0.Add(5*time.Minute), t0)
	got := CombineCVDVenues("BTCUSDT", []CVDVenueSeries{a, b}, []CVDPrice{{Time: t0, Close: 1}}, t0.Add(5*time.Minute))
	if got == nil || len(got.Points) != 1 || got.Points[0].Delta != 140 {
		t.Fatalf("%+v", got)
	}
	if got.Complete {
		t.Fatal("short overlap must not be complete")
	}
	if len(got.Contributions) != 2 {
		t.Fatalf("contributions %+v", got.Contributions)
	}
	var bin, byb CVDShare
	for _, s := range got.Contributions {
		switch s.Exchange {
		case ExchangeBinance:
			bin = s
		case ExchangeBybit:
			byb = s
		}
	}
	if bin.CVD != 100 || byb.CVD != 40 {
		t.Fatalf("shares bin=%+v byb=%+v", bin, byb)
	}
	if bin.SharePct < 70 || bin.SharePct > 72 || byb.SharePct < 28 || byb.SharePct > 30 {
		t.Fatalf("share pct bin=%v byb=%v", bin.SharePct, byb.SharePct)
	}
}

func TestCombineCVDVenues_OverlapOnlyAndIncomplete(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	now := t0.Add(24 * time.Hour)
	var binBuckets []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 24*12; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		binBuckets = append(binBuckets, TakerBucket{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Start: at,
			BuyNotional: 10, SellNotional: 0,
		})
		prices = append(prices, CVDPrice{Time: at, Close: 64000})
	}
	// Bybit only has the last 2 hours.
	var bybBuckets []TakerBucket
	bybStart := now.Add(-2 * time.Hour)
	for i := 0; i < 24; i++ {
		at := bybStart.Add(time.Duration(i) * 5 * time.Minute)
		bybBuckets = append(bybBuckets, TakerBucket{
			Exchange: ExchangeBybit, Symbol: "BTCUSDT", Start: at,
			BuyNotional: 4, SellNotional: 0,
		})
	}
	bin := BuildCVDSeries(ExchangeBinance, "BTCUSDT", binBuckets, prices, now, t0)
	byb := BuildCVDSeries(ExchangeBybit, "BTCUSDT", bybBuckets, prices, now, bybStart)
	if !bin.Complete || byb.Complete {
		t.Fatalf("setup complete bin=%v byb=%v", bin.Complete, byb.Complete)
	}
	got := CombineCVDVenues("BTCUSDT", []CVDVenueSeries{bin, byb}, prices, now)
	if got == nil {
		t.Fatal("nil")
	}
	if got.Complete {
		t.Fatal("combined must not be complete while Bybit history is short")
	}
	if len(got.Points) != 24 {
		t.Fatalf("overlap points %d (want 24 = 2h)", len(got.Points))
	}
	// Combined CVD is only the overlap: 24 bars * (10+4) = 336, not Binance's full 24h.
	if got.LastCVD != 24*14 {
		t.Fatalf("combined last %v (would be inflated if Binance-only bars were included)", got.LastCVD)
	}
	if got.OverlapFrom == nil || !got.OverlapFrom.Equal(bybStart) {
		t.Fatalf("overlap from %+v", got.OverlapFrom)
	}
	var w24 CVDWindowStat
	for _, w := range got.Windows {
		if w.Window == CVDWindow24h {
			w24 = w
		}
	}
	if w24.Complete {
		t.Fatal("24h window must not be complete")
	}
	if !strings.Contains(got.Summary, "not complete") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestBuildCVDSeries_PointDivergence(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	buckets := []TakerBucket{
		{Start: t0, BuyNotional: 100, SellNotional: 0},
		{Start: t0.Add(5 * time.Minute), BuyNotional: 0, SellNotional: 80},
	}
	prices := []CVDPrice{
		{Time: t0, Close: 100},
		{Time: t0.Add(5 * time.Minute), Close: 101},
	}
	got := BuildCVDSeries(ExchangeBinance, "BTCUSDT", buckets, prices, t0.Add(10*time.Minute), t0)
	if len(got.Points) != 2 {
		t.Fatalf("points %d", len(got.Points))
	}
	if got.Points[1].Divergence != CVDDivPriceUpCVDDown {
		t.Fatalf("point %+v", got.Points[1])
	}
	if got.Points[1].VsPrice != CVDVsOpposite {
		t.Fatalf("vsPrice %s", got.Points[1].VsPrice)
	}
	if got.Points[1].PriceChangePct <= 0 {
		t.Fatalf("price change %v", got.Points[1].PriceChangePct)
	}
}

func TestCombineCVDVenues_OneVenueNotComplete(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	a := BuildCVDSeries(ExchangeBinance, "BTCUSDT", []TakerBucket{
		{Start: t0, BuyNotional: 100, SellNotional: 0},
	}, []CVDPrice{{Time: t0, Close: 1}}, t0.Add(5*time.Minute), t0)
	a.Complete = true
	got := CombineCVDVenues("BTCUSDT", []CVDVenueSeries{a}, []CVDPrice{{Time: t0, Close: 1}}, t0.Add(5*time.Minute))
	if got == nil || got.Complete || len(got.Points) != 0 {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Summary, "not shown as complete") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestPriceAt_DoesNotUseNextBar(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	prices := []CVDPrice{{Time: t0, Close: 100}, {Time: t0.Add(5 * time.Minute), Close: 101}}
	if got := priceAt(prices, t0); got != 100 {
		t.Fatalf("bar0 %v", got)
	}
	if got := priceAt(prices, t0.Add(5*time.Minute)); got != 101 {
		t.Fatalf("bar1 %v", got)
	}
}

func TestCombineCVDVenues_CompleteDespiteBucketSkew(t *testing.T) {
	// Both venues cover 24h, but the first shared 5m slot is one bar after the cut
	// (Bybit missing the first bucket). Combined must still be complete.
	t0 := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	now := t0.Add(24 * time.Hour)
	var binBuckets, bybBuckets []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 24*12; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		binBuckets = append(binBuckets, TakerBucket{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Start: at, BuyNotional: 10,
		})
		prices = append(prices, CVDPrice{Time: at, Close: 64000})
		if i == 0 {
			continue
		}
		bybBuckets = append(bybBuckets, TakerBucket{
			Exchange: ExchangeBybit, Symbol: "BTCUSDT", Start: at, BuyNotional: 4,
		})
	}
	bin := BuildCVDSeries(ExchangeBinance, "BTCUSDT", binBuckets, prices, now, t0)
	byb := BuildCVDSeries(ExchangeBybit, "BTCUSDT", bybBuckets, prices, now, t0.Add(5*time.Minute))
	if !bin.Complete || !byb.Complete {
		t.Fatalf("setup complete bin=%v byb=%v", bin.Complete, byb.Complete)
	}
	got := CombineCVDVenues("BTCUSDT", []CVDVenueSeries{bin, byb}, prices, now)
	if got == nil || !got.Complete {
		t.Fatalf("combined complete=%v summary=%s", got.Complete, got.Summary)
	}
	var w24 CVDWindowStat
	for _, w := range got.Windows {
		if w.Window == CVDWindow24h {
			w24 = w
		}
	}
	if !w24.Complete {
		t.Fatal("24h window should be complete with one-bar slack")
	}
}

func TestCombineCVDVenues_MissingVenueIsNotComplete(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	now := t0.Add(24 * time.Hour)
	var binBuckets, bybBuckets []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 24*12; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		binBuckets = append(binBuckets, TakerBucket{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Start: at, BuyNotional: 10,
		})
		bybBuckets = append(bybBuckets, TakerBucket{
			Exchange: ExchangeBybit, Symbol: "BTCUSDT", Start: at, BuyNotional: 4,
		})
		prices = append(prices, CVDPrice{Time: at, Close: 64000})
	}
	bin := BuildCVDSeries(ExchangeBinance, "BTCUSDT", binBuckets, prices, now, t0)
	byb := BuildCVDSeries(ExchangeBybit, "BTCUSDT", bybBuckets, prices, now, t0)
	okx := CVDVenueSeries{Exchange: ExchangeOKX, Symbol: "BTCUSDT", Error: "no tape"}
	if !bin.Complete || !byb.Complete {
		t.Fatalf("setup complete bin=%v byb=%v", bin.Complete, byb.Complete)
	}
	got := CombineCVDVenues("BTCUSDT", []CVDVenueSeries{bin, byb, okx}, prices, now)
	if got == nil {
		t.Fatal("nil")
	}
	if got.Complete {
		t.Fatalf("complete summary=%s", got.Summary)
	}
	if len(got.Points) == 0 {
		t.Fatal("overlap of the venues that have bars should still be returned")
	}
	if !strings.Contains(got.Summary, "okx") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestCombineCVDVenues_FillsMissingMiddleBucket(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	now := t0.Add(15 * time.Minute)
	bin := BuildCVDSeries(ExchangeBinance, "BTCUSDT", []TakerBucket{
		{Start: t0, BuyNotional: 10},
		{Start: t0.Add(10 * time.Minute), BuyNotional: 10},
	}, []CVDPrice{{Time: t0, Close: 1}, {Time: t0.Add(10 * time.Minute), Close: 1}}, now, t0)
	byb := BuildCVDSeries(ExchangeBybit, "BTCUSDT", []TakerBucket{
		{Start: t0, BuyNotional: 5},
		{Start: t0.Add(5 * time.Minute), BuyNotional: 5},
		{Start: t0.Add(10 * time.Minute), BuyNotional: 5},
	}, []CVDPrice{{Time: t0, Close: 1}, {Time: t0.Add(5 * time.Minute), Close: 1}, {Time: t0.Add(10 * time.Minute), Close: 1}}, now, t0)
	got := CombineCVDVenues("BTCUSDT", []CVDVenueSeries{bin, byb}, nil, now)
	if got == nil || len(got.Points) != 3 {
		t.Fatalf("points %d", len(got.Points))
	}
	if got.Points[1].Delta != 5 {
		t.Fatalf("middle should keep Bybit-only delta, got %+v", got.Points[1])
	}
}

func TestCombineCVDVenues_VenueSplitAnd15mChange(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	now := t0.Add(20 * time.Minute)
	var binB, bybB []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 4; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		binB = append(binB, TakerBucket{Start: at, BuyNotional: 100, SellNotional: 0})
		bybB = append(bybB, TakerBucket{Start: at, BuyNotional: 0, SellNotional: 80})
		prices = append(prices, CVDPrice{Time: at, Close: 100 + float64(i)})
	}
	bin := BuildCVDSeries(ExchangeBinance, "BTCUSDT", binB, prices, now, t0)
	byb := BuildCVDSeries(ExchangeBybit, "BTCUSDT", bybB, prices, now, t0)
	got := CombineCVDVenues("BTCUSDT", []CVDVenueSeries{bin, byb}, prices, now)
	if got == nil {
		t.Fatal("nil")
	}
	var w15 CVDWindowStat
	for _, w := range got.Windows {
		if w.Window == CVDWindow15m {
			w15 = w
		}
	}
	if w15.Window == "" {
		t.Fatal("missing 15m window")
	}
	if w15.VenueSplit == nil || w15.VenueSplit.Alignment != AlignOpposite {
		t.Fatalf("split %+v", w15.VenueSplit)
	}
	if w15.VenueSplit.Binance != CVDDirUp || w15.VenueSplit.Bybit != CVDDirDown {
		t.Fatalf("dirs %+v", w15.VenueSplit)
	}
	if got.VenueSplit == nil || got.VenueSplit.Alignment != AlignOpposite {
		t.Fatalf("series split %+v", got.VenueSplit)
	}
}

func TestBuildCVDSeries_DivergenceDurationAndMove(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	var buckets []TakerBucket
	var prices []CVDPrice
	// 3 bars: price up, CVD down each time.
	for i := 0; i < 3; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		buckets = append(buckets, TakerBucket{Start: at, BuyNotional: 0, SellNotional: 50})
		prices = append(prices, CVDPrice{Time: at, Close: 100 + float64(i)})
	}
	got := BuildCVDSeries(ExchangeBinance, "BTCUSDT", buckets, prices, t0.Add(15*time.Minute), t0)
	d := got.Divergence
	if d.Kind != CVDDivPriceUpCVDDown {
		t.Fatalf("kind %s", d.Kind)
	}
	if d.DurationSeconds < 10*60 {
		t.Fatalf("duration %d %s", d.DurationSeconds, d.Duration)
	}
	if d.PriceMovePct <= 0 || d.CVDMove >= 0 {
		t.Fatalf("moves price=%v cvd=%v", d.PriceMovePct, d.CVDMove)
	}
	if !strings.Contains(d.Summary, "for ") || !strings.Contains(d.Summary, "CVD") {
		t.Fatalf("summary %s", d.Summary)
	}
	var w15 CVDWindowStat
	for _, w := range got.Windows {
		if w.Window == CVDWindow15m {
			w15 = w
		}
	}
	if w15.Window == "" || w15.CVDChange == 0 {
		t.Fatalf("15m %+v", w15)
	}
}

func TestCurrentCVDDivergenceRun_DoesNotBridgeGap(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	var buckets []TakerBucket
	var prices []CVDPrice
	// Episode 1: price up, CVD down
	for i := 0; i < 3; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		buckets = append(buckets, TakerBucket{Start: at, SellNotional: 50})
		prices = append(prices, CVDPrice{Time: at, Close: 100 + float64(i)})
	}
	// Quiet gap (price and CVD flat-ish: tiny price move, zero delta)
	for i := 3; i < 8; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		buckets = append(buckets, TakerBucket{Start: at, BuyNotional: 1, SellNotional: 1})
		prices = append(prices, CVDPrice{Time: at, Close: 102})
	}
	// Episode 2: same kind again
	for i := 8; i < 11; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		buckets = append(buckets, TakerBucket{Start: at, SellNotional: 50})
		prices = append(prices, CVDPrice{Time: at, Close: 102 + float64(i-7)})
	}
	got := BuildCVDSeries(ExchangeBinance, "BTCUSDT", buckets, prices, t0.Add(55*time.Minute), t0)
	d := got.Divergence
	if d.Kind != CVDDivPriceUpCVDDown {
		t.Fatalf("kind %s", d.Kind)
	}
	if d.DurationSeconds > 20*60 {
		t.Fatalf("bridged the gap: duration %d %s since %s last %s", d.DurationSeconds, d.Duration, d.Since, d.LastAt)
	}
	if d.Since.Before(t0.Add(35 * time.Minute)) {
		t.Fatalf("run started too early %s", d.Since)
	}
}

func TestTakerBucketsFromCandles(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	got := TakerBucketsFromCandles(ExchangeBinance, "BTCUSDT", []Candle{
		{OpenTime: t0, QuoteVolume: "200", TakerBuyQuote: "150"},
		{OpenTime: t0.Add(5 * time.Minute), QuoteVolume: "100", TakerBuyQuote: "20"},
	})
	if len(got) != 2 || got[0].BuyNotional != 150 || got[0].SellNotional != 50 {
		t.Fatalf("%+v", got)
	}
	if got[1].BuyNotional != 20 || got[1].SellNotional != 80 {
		t.Fatalf("bar1 %+v", got[1])
	}
}

func TestCompareSpotFutures_Opposite(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	now := t0.Add(20 * time.Minute)
	var spotB, futB []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 4; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		spotB = append(spotB, TakerBucket{Start: at, BuyNotional: 100})
		futB = append(futB, TakerBucket{Start: at, SellNotional: 80})
		prices = append(prices, CVDPrice{Time: at, Close: 100})
	}
	spot := BuildCVDSeries(ExchangeBinance, "BTCUSDT", spotB, prices, now, t0)
	spot.Market = CVDMarketSpot
	fut := BuildCVDSeries(ExchangeBinance, "BTCUSDT", futB, prices, now, t0)
	got := CompareSpotFutures(&spot, &fut)
	if got == nil || got.Alignment != AlignOpposite {
		t.Fatalf("%+v", got)
	}
	if got.Spot != CVDDirUp || got.Futures != CVDDirDown {
		t.Fatalf("dirs %+v", got)
	}
	if !strings.Contains(got.Summary, "Spot CVD") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestResampleTakerBuckets_1mTo5m(t *testing.T) {
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	in := []TakerBucket{
		{Start: t0, BuyNotional: 10},
		{Start: t0.Add(time.Minute), BuyNotional: 5, SellNotional: 2},
		{Start: t0.Add(5 * time.Minute), SellNotional: 3},
	}
	got := ResampleTakerBuckets(in, 5*time.Minute)
	if len(got) != 2 || got[0].BuyNotional != 15 || got[0].SellNotional != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestBuildCVDSeries_AnchorsResetAtDayAnd4hOpen(t *testing.T) {
	// 10:30 UTC. Rolling 4h starts at 06:30. The 4h anchor starts at 08:00.
	// Day anchor starts at 00:00, but bars only exist from 07:00, so day is partial.
	day := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	bars := []struct {
		at    time.Time
		buy   float64
		sell  float64
		price float64
	}{
		{day.Add(6*time.Hour + 35*time.Minute), 1, 0, 99},
		{day.Add(7 * time.Hour), 100, 0, 100},
		{day.Add(8 * time.Hour), 0, 40, 101},
		{day.Add(10*time.Hour + 25*time.Minute), 10, 0, 102},
	}
	var buckets []TakerBucket
	var prices []CVDPrice
	for _, b := range bars {
		buckets = append(buckets, TakerBucket{Start: b.at, BuyNotional: b.buy, SellNotional: b.sell})
		prices = append(prices, CVDPrice{Time: b.at, Close: b.price})
	}
	now := day.Add(10*time.Hour + 30*time.Minute)
	got := BuildCVDSeries(ExchangeBinance, "BTCUSDT", buckets, prices, now, bars[0].at)
	var dayA, h4 CVDAnchor
	for _, a := range got.Anchors {
		switch a.Anchor {
		case CVDAnchorDay:
			dayA = a
		case CVDAnchor4h:
			h4 = a
		}
	}
	if dayA.From != day || dayA.CVD != 71 || dayA.Complete {
		t.Fatalf("day %+v", dayA)
	}
	if !strings.Contains(dayA.Summary, "00:00 UTC") || !strings.Contains(dayA.Summary, "bars on hand") {
		t.Fatalf("day summary %s", dayA.Summary)
	}
	if h4.From != day.Add(8*time.Hour) || h4.CVD != -30 || !h4.Complete {
		t.Fatalf("4h %+v", h4)
	}
	if !strings.Contains(h4.Summary, "08:00 UTC") {
		t.Fatalf("4h summary %s", h4.Summary)
	}
	var roll CVDWindowStat
	for _, w := range got.Windows {
		if w.Window == CVDWindow4h {
			roll = w
		}
	}
	if roll.CVDChange == h4.CVD {
		t.Fatalf("rolling 4h should include the 07:00 bar, anchored 4h should not: roll %v anchor %v", roll.CVDChange, h4.CVD)
	}
}

func TestAttachCVDClickAnchor_SumsFromThatBar(t *testing.T) {
	day := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	bars := []struct {
		at    time.Time
		buy   float64
		sell  float64
		price float64
	}{
		{day.Add(6*time.Hour + 35*time.Minute), 1, 0, 99},
		{day.Add(7 * time.Hour), 100, 0, 100},
		{day.Add(8 * time.Hour), 0, 40, 101},
		{day.Add(10*time.Hour + 25*time.Minute), 10, 0, 102},
	}
	var buckets []TakerBucket
	var prices []CVDPrice
	for _, b := range bars {
		buckets = append(buckets, TakerBucket{Start: b.at, BuyNotional: b.buy, SellNotional: b.sell})
		prices = append(prices, CVDPrice{Time: b.at, Close: b.price})
	}
	now := day.Add(10*time.Hour + 30*time.Minute)
	series := BuildCVDSeries(ExchangeBinance, "BTCUSDT", buckets, prices, now, bars[0].at)
	rep := &CVDReport{Symbol: "BTCUSDT", Venues: []CVDVenueSeries{series}}
	AttachCVDClickAnchor(rep, day.Add(8*time.Hour))
	var click CVDAnchor
	for _, a := range rep.Venues[0].Anchors {
		if a.Anchor == CVDAnchorClick {
			click = a
		}
	}
	if click.From != day.Add(8*time.Hour) || click.CVD != -30 || !click.Complete {
		t.Fatalf("click %+v", click)
	}
	if !strings.Contains(click.Summary, "chosen candle") || !strings.Contains(click.Summary, "2026-09-26 08:00 UTC") {
		t.Fatalf("summary %s", click.Summary)
	}
	var dayA CVDAnchor
	for _, a := range rep.Venues[0].Anchors {
		if a.Anchor == CVDAnchorDay {
			dayA = a
		}
	}
	if dayA.CVD != 71 {
		t.Fatalf("day anchor changed %+v", dayA)
	}

	early := BuildCVDSeries(ExchangeBinance, "BTCUSDT", buckets, prices, now, bars[0].at)
	rep = &CVDReport{Venues: []CVDVenueSeries{early}}
	AttachCVDClickAnchor(rep, day)
	click = CVDAnchor{}
	for _, a := range rep.Venues[0].Anchors {
		if a.Anchor == CVDAnchorClick {
			click = a
		}
	}
	if click.Complete || !strings.Contains(click.Summary, "bars on hand") {
		t.Fatalf("day click should be partial %+v", click)
	}
}

func TestParseCVDAnchor(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	if at, err := ParseCVDAnchor("", now); err != nil || !at.IsZero() {
		t.Fatalf("empty %+v %v", at, err)
	}
	at, err := ParseCVDAnchor("2026-10-09T08:15:00Z", now)
	if err != nil || !at.Equal(now.Add(-4*time.Hour+15*time.Minute)) {
		t.Fatalf("%+v %v", at, err)
	}
	if _, err := ParseCVDAnchor("tomorrow", now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("bad %v", err)
	}
	if _, err := ParseCVDAnchor(now.Add(2*time.Hour).Format(time.RFC3339), now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("future %v", err)
	}
	if _, err := ParseCVDAnchor(now.Add(-91*24*time.Hour).Format(time.RFC3339), now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("old %v", err)
	}
}

func TestParseCVDRange(t *testing.T) {
	def, err := ParseCVDRange("")
	if err != nil || def.ID != "24h" || def.Bar != CVDBar || def.Lookback != DefaultCVDLookback {
		t.Fatalf("%+v %v", def, err)
	}
	week, err := ParseCVDRange("7d")
	if err != nil || week.Bar != 30*time.Minute || week.Lookback != 7*24*time.Hour || week.BarLabel() != "30m" {
		t.Fatalf("%+v %v", week, err)
	}
	month, err := ParseCVDRange("30d")
	if err != nil || month.Bar != 2*time.Hour || month.BarLabel() != "2h" {
		t.Fatalf("%+v %v", month, err)
	}
	quarter, err := ParseCVDRange("90d")
	if err != nil || quarter.Bar != 4*time.Hour || quarter.Lookback != 90*24*time.Hour || quarter.BarLabel() != "4h" {
		t.Fatalf("%+v %v", quarter, err)
	}
	if _, err := ParseCVDRange("1y"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestBuildCVDSeriesSpan_KeepsDaysBeyond24h(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	old := now.Add(-3 * 24 * time.Hour).Truncate(30 * time.Minute)
	recent := now.Add(-20 * time.Minute).Truncate(30 * time.Minute)
	span, err := ParseCVDRange("7d")
	if err != nil {
		t.Fatal(err)
	}
	got := BuildCVDSeriesSpan(ExchangeBinance, "BTCUSDT", []TakerBucket{
		{Start: old, BuyNotional: 10, SellNotional: 1},
		{Start: old.Add(time.Minute), BuyNotional: 5, SellNotional: 0},
		{Start: recent, BuyNotional: 2, SellNotional: 8},
	}, nil, now, old, span)
	if got.Range != "7d" || got.Bar != 30*time.Minute {
		t.Fatalf("span %+v", got)
	}
	if len(got.Points) != 2 {
		t.Fatalf("points %d %+v", len(got.Points), got.Points)
	}
	if !got.Points[0].Time.Equal(old) || got.Points[0].Delta != 14 {
		t.Fatalf("old bar %+v", got.Points[0])
	}
	if got.Points[1].Delta != -6 {
		t.Fatalf("recent %+v", got.Points[1])
	}
	if got.Complete {
		t.Fatal("three days of bars must not mark a 7d range complete")
	}
	short := BuildCVDSeries(ExchangeBinance, "BTCUSDT", []TakerBucket{
		{Start: old, BuyNotional: 10, SellNotional: 1},
		{Start: recent, BuyNotional: 2, SellNotional: 8},
	}, nil, now, old)
	for _, p := range short.Points {
		if !p.Time.After(now.Add(-DefaultCVDLookback)) && !p.Time.Equal(now.Add(-DefaultCVDLookback)) {
			if p.Time.Before(now.Add(-DefaultCVDLookback)) {
				t.Fatalf("24h series kept %s", p.Time)
			}
		}
	}
	if len(short.Points) != 1 || short.Points[0].Time.Before(now.Add(-DefaultCVDLookback)) {
		t.Fatalf("24h points %+v", short.Points)
	}
}

func TestVenueCVDSplit_CurrentRun(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	day := now.Truncate(24 * time.Hour)
	buckets := []TakerBucket{{Start: day, BuyNotional: 1, SellNotional: 1}}
	prices := []CVDPrice{{Time: day, Close: 100}}
	for i, at := range []time.Time{
		now.Add(-20 * time.Minute),
		now.Add(-15 * time.Minute),
		now.Add(-10 * time.Minute),
		now.Add(-5 * time.Minute),
	} {
		buckets = append(buckets, TakerBucket{Start: at, BuyNotional: 0, SellNotional: 50})
		prices = append(prices, CVDPrice{Time: at, Close: 101 + float64(i)*0.3})
	}
	split, series := VenueCVDSplit(ExchangeBinance, "BTCUSDT", buckets, prices, now)
	if !split.OK || split.Kind != CVDDivPriceUpCVDDown || split.Seconds <= 0 || split.Since.IsZero() {
		t.Fatalf("split %+v divergence %+v", split, series.Divergence)
	}
	if split.Duration == "" || !split.At.Equal(now.Add(-5*time.Minute)) {
		t.Fatalf("duration %q at %s", split.Duration, split.At)
	}
}

func TestVenueCVDSplit_LaterAgreementIsInLine(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	day := now.Truncate(24 * time.Hour)
	buckets := []TakerBucket{{Start: day, BuyNotional: 1, SellNotional: 1}}
	prices := []CVDPrice{{Time: day, Close: 100}}
	for _, step := range []struct {
		ago   time.Duration
		buy   float64
		sell  float64
		close float64
	}{
		{30 * time.Minute, 0, 50, 101},
		{25 * time.Minute, 0, 50, 101.4},
		{10 * time.Minute, 50, 0, 102.2},
		{5 * time.Minute, 40, 0, 102.8},
	} {
		at := now.Add(-step.ago)
		buckets = append(buckets, TakerBucket{Start: at, BuyNotional: step.buy, SellNotional: step.sell})
		prices = append(prices, CVDPrice{Time: at, Close: step.close})
	}
	split, series := VenueCVDSplit(ExchangeBinance, "BTCUSDT", buckets, prices, now)
	if series.Divergence.Kind != CVDDivPriceUpCVDDown || series.Divergence.Since.IsZero() {
		t.Fatalf("latest episode should stay on the series, got %+v", series.Divergence)
	}
	if !split.OK || split.Kind != CVDSplitAgrees || split.Seconds != 0 || !split.Since.IsZero() {
		t.Fatalf("closed run leaked %+v", split)
	}
}

func TestVenueCVDSplit_TrailingQuietKeepsTheRun(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	day := now.Truncate(24 * time.Hour)
	buckets := []TakerBucket{{Start: day, BuyNotional: 1, SellNotional: 1}}
	prices := []CVDPrice{{Time: day, Close: 100}}
	for _, step := range []struct {
		ago   time.Duration
		buy   float64
		sell  float64
		close float64
	}{
		{20 * time.Minute, 0, 50, 101},
		{15 * time.Minute, 0, 50, 101.4},
		{10 * time.Minute, 1, 1, 101.4},
		{5 * time.Minute, 1, 1, 101.4},
	} {
		at := now.Add(-step.ago)
		buckets = append(buckets, TakerBucket{Start: at, BuyNotional: step.buy, SellNotional: step.sell})
		prices = append(prices, CVDPrice{Time: at, Close: step.close})
	}
	split, _ := VenueCVDSplit(ExchangeBinance, "BTCUSDT", buckets, prices, now)
	if !split.OK || split.Kind != CVDDivPriceUpCVDDown || split.Seconds <= 0 || split.Since.IsZero() {
		t.Fatalf("quiet tail closed the run %+v", split)
	}
}

func TestVenueCVDSplit_WithholdsWhenTapeMissesDayOpen(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	start := now.Truncate(24 * time.Hour).Add(time.Hour)
	var buckets []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 4; i++ {
		at := start.Add(time.Duration(i) * 5 * time.Minute)
		buckets = append(buckets, TakerBucket{Start: at, SellNotional: 50})
		prices = append(prices, CVDPrice{Time: at, Close: 100 + float64(i)})
	}
	split, _ := VenueCVDSplit(ExchangeBybit, "BTCUSDT", buckets, prices, now)
	if split.OK || split.Kind != "" {
		t.Fatalf("late tape leaked %+v", split)
	}
}

func TestVenueCVDSplit_WindowFallbackIsInLine(t *testing.T) {
	now := time.Date(2026, 9, 29, 4, 0, 0, 0, time.UTC)
	day := now.Truncate(24 * time.Hour)
	var buckets []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 48; i++ {
		at := day.Add(time.Duration(i) * 5 * time.Minute)
		buckets = append(buckets, TakerBucket{Start: at, SellNotional: 10})
		prices = append(prices, CVDPrice{Time: at, Close: 100 + 0.02*float64(i)})
	}
	split, series := VenueCVDSplit(ExchangeBinance, "BTCUSDT", buckets, prices, now)
	if series.Divergence.Kind != CVDDivPriceUpCVDDown || !series.Divergence.Since.IsZero() {
		t.Fatalf("expected a window fallback, got %+v", series.Divergence)
	}
	if !split.OK || split.Kind != CVDSplitAgrees || split.Seconds != 0 || !split.Since.IsZero() {
		t.Fatalf("split %+v", split)
	}
}

func TestVenueCVDSplit_NoPriceStaysBlank(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	day := now.Truncate(24 * time.Hour)
	buckets := []TakerBucket{
		{Start: day, BuyNotional: 10, SellNotional: 1},
		{Start: now.Add(-time.Minute), BuyNotional: 4, SellNotional: 1},
	}
	split, _ := VenueCVDSplit(ExchangeOKX, "BTCUSDT", buckets, nil, now)
	if split.OK || split.Kind != "" {
		t.Fatalf("price-less tape leaked %+v", split)
	}
}
