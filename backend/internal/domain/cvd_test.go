package domain

import (
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
