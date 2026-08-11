package domain

import (
	"testing"
	"time"
)

func TestClampFundingHistoryLimit(t *testing.T) {
	if ClampFundingHistoryLimit(0) != DefaultFundingHistoryLimit {
		t.Fatal("default")
	}
	if ClampFundingHistoryLimit(99) != MaxFundingHistoryLimit {
		t.Fatal("max")
	}
	if ClampFundingHistoryLimit(5) != 5 {
		t.Fatal("pass")
	}
}

func TestFundingPayerAndFormat(t *testing.T) {
	if FundingPayer(0.0001) != "long" || FundingPayer(-0.0001) != "short" || FundingPayer(0) != "none" {
		t.Fatal("payer")
	}
	dec, pct := FormatFundingRate(0.00008804)
	if dec != "0.00008804" || pct != "0.008804" {
		t.Fatalf("format %s %s", dec, pct)
	}
}

func TestInferFundingIntervalHours(t *testing.T) {
	if InferFundingIntervalHours(4, time.Time{}, time.Time{}) != 4 {
		t.Fatal("hint")
	}
	next := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	last := next.Add(-8 * time.Hour)
	if InferFundingIntervalHours(0, next, last) != 8 {
		t.Fatal("gap")
	}
	if InferFundingIntervalHours(0, time.Time{}, time.Time{}) != 8 {
		t.Fatal("default")
	}
}

func TestAverageFundingRate(t *testing.T) {
	hist := []FundingPoint{{Rate: 0.0003}, {Rate: 0.0001}, {Rate: 0.0002}}
	avg, ok := AverageFundingRate(hist, 3)
	if !ok || avg < 0.000199 || avg > 0.000201 {
		t.Fatalf("avg %v ok=%v", avg, ok)
	}
	if _, ok := AverageFundingRate(nil, 3); ok {
		t.Fatal("empty")
	}
}

func TestBuildFundingSnapshot_SingleAndAll(t *testing.T) {
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	bin := &FundingSeries{
		Exchange:        ExchangeBinance,
		Symbol:          "BTCUSDT",
		Current:         FundingPoint{Time: now, Rate: 0.0001, Predicted: true},
		NextFundingTime: now.Add(time.Hour),
		IntervalHours:   8,
		History: []FundingPoint{
			{Time: now.Add(-8 * time.Hour), Rate: 0.00008},
			{Time: now.Add(-16 * time.Hour), Rate: 0.00004},
		},
	}
	byb := &FundingSeries{
		Exchange:        ExchangeBybit,
		Symbol:          "BTCUSDT",
		Current:         FundingPoint{Time: now, Rate: 0.00005, Predicted: true},
		NextFundingTime: now.Add(time.Hour),
		IntervalHours:   8,
		History: []FundingPoint{
			{Time: now.Add(-8 * time.Hour), Rate: 0.00003},
		},
	}
	one := BuildFundingSnapshot("binance", "btc-usd", []*FundingSeries{bin}, now)
	if one.Symbol != "BTCUSDT" || one.VenueCount != 1 || one.Current == nil || one.Current.Payer != "long" {
		t.Fatalf("one %+v", one)
	}
	if len(one.History) != 2 || one.Venues[0].LastSettled == nil {
		t.Fatalf("hist %+v", one)
	}
	all := BuildFundingSnapshot("all", "BTCUSDT", []*FundingSeries{byb, bin}, now)
	if all.Current != nil || all.VenueCount != 2 || all.Venues[0].Exchange != "binance" {
		t.Fatalf("all %+v", all)
	}
}
