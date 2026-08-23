package domain

import (
	"strings"
	"testing"
	"time"
)

func TestClassifyAbsorption_BidsHoldSells(t *testing.T) {
	got := ClassifyAbsorption(20, 80, 0.02, 0.08)
	if got.Kind != AbsorptionKindBid || got.Absorber != TakerSideBuy || got.Result != AbsorptionResultHeld {
		t.Fatalf("%+v", got)
	}
	if got.Score < 40 || got.Grade == AbsorptionGradeNone {
		t.Fatalf("score %+v", got)
	}
}

func TestClassifyAbsorption_AsksHoldBuys(t *testing.T) {
	got := ClassifyAbsorption(90, 10, -0.01, 0.08)
	if got.Kind != AbsorptionKindAsk || got.Absorber != TakerSideSell || got.Result != AbsorptionResultHeld {
		t.Fatalf("%+v", got)
	}
}

func TestClassifyAbsorption_PushedAgainstSells(t *testing.T) {
	got := ClassifyAbsorption(15, 85, 0.25, 0.08)
	if got.Kind != AbsorptionKindBid || got.Result != AbsorptionResultPushed {
		t.Fatalf("%+v", got)
	}
	held := ClassifyAbsorption(15, 85, 0.01, 0.08)
	if got.Score <= held.Score {
		t.Fatalf("pushed %d held %d", got.Score, held.Score)
	}
}

func TestClassifyAbsorption_PriceFollowsFlow(t *testing.T) {
	// Heavy sells and price dropped — impact, not absorption.
	got := ClassifyAbsorption(10, 90, -0.40, 0.08)
	if got.Kind != "" || got.Score != 0 {
		t.Fatalf("want none %+v", got)
	}
	got = ClassifyAbsorption(90, 10, 0.40, 0.08)
	if got.Kind != "" {
		t.Fatalf("buy follow %+v", got)
	}
}

func TestClassifyAbsorption_MixedFlow(t *testing.T) {
	got := ClassifyAbsorption(50, 51, 0, 0.08)
	if got.Kind != "" {
		t.Fatalf("balanced %+v", got)
	}
}

func TestAbsorptionGrade(t *testing.T) {
	if AbsorptionGrade(90) != AbsorptionGradeExtreme {
		t.Fatal("90")
	}
	if AbsorptionGrade(70) != AbsorptionGradeStrong {
		t.Fatal("70")
	}
	if AbsorptionGrade(45) != AbsorptionGradeModerate {
		t.Fatal("45")
	}
	if AbsorptionGrade(10) != AbsorptionGradeWeak {
		t.Fatal("10")
	}
	if AbsorptionGrade(0) != AbsorptionGradeNone {
		t.Fatal("0")
	}
}

func TestBuildAbsorptionSeries_WindowsAndRun(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	now := t0.Add(50 * time.Minute)
	var buckets []TakerBucket
	var prices []CVDPrice
	px := 70000.0
	for i := 0; i < 10; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		// Heavy sells, price barely ticks down then holds.
		buckets = append(buckets, TakerBucket{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Start: at,
			BuyNotional: 20, SellNotional: 180,
		})
		prices = append(prices, CVDPrice{Time: at, Close: px})
		px -= 2 // ~0.003% per bar — flat
	}
	got := BuildAbsorptionSeries(ExchangeBinance, "BTCUSDT", buckets, prices, now, t0)
	if got.Error != "" || len(got.Points) != 10 {
		t.Fatalf("%+v", got)
	}
	var absorbBars int
	for _, p := range got.Points[1:] {
		if p.Kind != AbsorptionKindBid {
			t.Fatalf("bar %+v", p)
		}
		absorbBars++
	}
	if absorbBars < 8 {
		t.Fatalf("bars %d", absorbBars)
	}
	var w1 AbsorptionWindowStat
	for _, w := range got.Windows {
		if w.Window == CVDWindow1h {
			w1 = w
		}
	}
	if w1.Kind != AbsorptionKindBid || w1.BuyNotional != 200 || w1.SellNotional != 1800 {
		t.Fatalf("window %+v", w1)
	}
	if w1.Score <= 0 || !strings.Contains(w1.Summary, "absorbing") {
		t.Fatalf("summary %+v", w1)
	}
	if got.Current == nil || got.Current.Kind != AbsorptionKindBid || got.Current.Bars < 8 {
		t.Fatalf("current %+v", got.Current)
	}
	if got.Summary == "" {
		t.Fatal("venue summary")
	}
}

func TestBuildAbsorptionSeries_QuietGapSplitsRuns(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	now := t0.Add(40 * time.Minute)
	var buckets []TakerBucket
	var prices []CVDPrice
	for i := 0; i < 8; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		buy, sell := 80.0, 20.0
		if i == 3 || i == 4 {
			buy, sell = 50, 50 // quiet gap
		}
		buckets = append(buckets, TakerBucket{Start: at, BuyNotional: buy, SellNotional: sell})
		prices = append(prices, CVDPrice{Time: at, Close: 100})
	}
	got := BuildAbsorptionSeries(ExchangeBinance, "ETHUSDT", buckets, prices, now, t0)
	if len(got.Episodes) < 2 {
		t.Fatalf("expected split runs %+v", got.Episodes)
	}
	if got.Episodes[0].Kind != AbsorptionKindAsk || got.Episodes[1].Kind != AbsorptionKindAsk {
		t.Fatalf("%+v", got.Episodes)
	}
}

func TestCombineAbsorptionVenues(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	now := t0.Add(30 * time.Minute)
	var prices []CVDPrice
	var bn, by []TakerBucket
	for i := 0; i < 6; i++ {
		at := t0.Add(time.Duration(i) * 5 * time.Minute)
		prices = append(prices, CVDPrice{Time: at, Close: 200})
		bn = append(bn, TakerBucket{Exchange: ExchangeBinance, Start: at, BuyNotional: 10, SellNotional: 90})
		by = append(by, TakerBucket{Exchange: ExchangeBybit, Start: at, BuyNotional: 5, SellNotional: 45})
	}
	a := BuildAbsorptionSeries(ExchangeBinance, "BTCUSDT", bn, prices, now, t0)
	b := BuildAbsorptionSeries(ExchangeBybit, "BTCUSDT", by, prices, now, t0)
	got := CombineAbsorptionVenues("BTCUSDT", []AbsorptionVenue{a, b}, prices, now)
	if got == nil || len(got.Points) == 0 {
		t.Fatal("combined")
	}
	if got.Points[1].BuyNotional != 15 || got.Points[1].SellNotional != 135 {
		t.Fatalf("sum %+v", got.Points[1])
	}
	if got.Points[1].Kind != AbsorptionKindBid {
		t.Fatalf("kind %+v", got.Points[1])
	}
}

func TestExplainAbsorptionReport(t *testing.T) {
	rep := AbsorptionReport{
		Combined: &AbsorptionVenue{Summary: "bids absorbing market sells", Error: ""},
	}
	if ExplainAbsorptionReport(rep) != "Futures bids absorbing market sells" {
		t.Fatalf("%s", ExplainAbsorptionReport(rep))
	}
}
