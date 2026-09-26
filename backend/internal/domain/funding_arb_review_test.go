package domain

import (
	"testing"
	"time"
)

func TestBuildFundingArbReview_SettledVsExpected(t *testing.T) {
	at := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	clk := at.Add(8 * time.Hour)
	exp := &FundingArbExpected{
		Notional: 10_000, HoldHours: 24,
		LongExchange: ExchangeBinance, ShortExchange: ExchangeBybit,
		HorizonFunding: 20, RoundTripFee: 40, ExpectedNet: -20,
		Payments: []FundingArbExpectedPayment{
			{Time: clk, Exchange: ExchangeBinance, Rate: 0.0001, Amount: -1},
			{Time: clk, Exchange: ExchangeBybit, Rate: 0.0011, Amount: 11},
		},
	}
	// Predicted Bybit 0.11%; settled 0.05% — still collects but less.
	got := BuildFundingArbReview(exp, map[Exchange][]FundingPoint{
		ExchangeBinance: {{Time: clk, Rate: 0.0001}},
		ExchangeBybit:   {{Time: clk, Rate: 0.0005}},
	}, clk.Add(time.Minute))
	if got == nil || !got.Complete || got.SettledCount != 2 {
		t.Fatalf("%+v", got)
	}
	if got.ActualFunding != 4 { // -1 + 5
		t.Fatalf("actual funding %g", got.ActualFunding)
	}
	if got.ActualNet != -36 || got.StillWorthIt {
		t.Fatalf("net %+v", got)
	}
	if got.DeltaNet != -16 {
		t.Fatalf("delta %g", got.DeltaNet)
	}
}

func TestBuildFundingArbReview_PendingClock(t *testing.T) {
	at := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	exp := &FundingArbExpected{
		Notional: 10_000, ExpectedNet: 10, HorizonFunding: 50, RoundTripFee: 40,
		LongExchange: ExchangeBinance, ShortExchange: ExchangeBybit,
		Payments: []FundingArbExpectedPayment{
			{Time: at.Add(8 * time.Hour), Exchange: ExchangeBinance, Rate: 0.0001, Amount: -1},
			{Time: at.Add(16 * time.Hour), Exchange: ExchangeBybit, Rate: 0.001, Amount: 10},
		},
	}
	got := BuildFundingArbReview(exp, map[Exchange][]FundingPoint{
		ExchangeBinance: {{Time: at.Add(8 * time.Hour), Rate: 0.0002}},
	}, at.Add(9*time.Hour))
	if got.Complete || got.PendingCount != 1 || got.SettledCount != 1 {
		t.Fatalf("%+v", got)
	}
	if !got.StillWorthIt && got.Complete {
		t.Fatal("incomplete must not claim still-worth")
	}
}

func TestBuildFundingArbReview_MissingPrint(t *testing.T) {
	clk := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	exp := &FundingArbExpected{
		Notional: 1000, ExpectedNet: 1, Payments: []FundingArbExpectedPayment{
			{Time: clk, Exchange: ExchangeBinance, Rate: 0.001, Amount: -1},
		},
	}
	got := BuildFundingArbReview(exp, nil, clk.Add(time.Hour))
	if !got.Complete || got.MissingCount != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestBuildFundingArbReview_OKXPrint(t *testing.T) {
	clk := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	exp := &FundingArbExpected{
		Notional: 10_000, ExpectedNet: 20, HorizonFunding: 20, RoundTripFee: 0,
		LongExchange: ExchangeBinance, ShortExchange: ExchangeOKX,
		Payments: []FundingArbExpectedPayment{
			{Time: clk, Exchange: ExchangeOKX, Rate: 0.002, Amount: 20},
		},
	}
	got := BuildFundingArbReview(exp, map[Exchange][]FundingPoint{
		ExchangeOKX: {{Time: clk, Rate: 0.002}},
	}, clk.Add(time.Minute))
	if got == nil || got.MissingCount != 0 || got.SettledCount != 1 {
		t.Fatalf("%+v", got)
	}
	if got.Payments[0].Exchange != ExchangeOKX || got.ActualFunding != 20 {
		t.Fatalf("%+v", got.Payments[0])
	}
}

func TestFindSettledFundingPrint_MinuteMatch(t *testing.T) {
	want := time.Date(2026, 8, 20, 8, 0, 30, 0, time.UTC)
	pt, ok := findSettledFundingPrint([]FundingPoint{
		{Time: time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC), Rate: 0.001},
	}, want)
	if !ok || pt.Rate != 0.001 {
		t.Fatalf("%v %+v", ok, pt)
	}
}
