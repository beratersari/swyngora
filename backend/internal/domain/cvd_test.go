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
