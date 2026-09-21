package domain

import (
	"errors"
	"testing"
)

func TestParsePerpDepthRangePct(t *testing.T) {
	got, err := ParsePerpDepthRangePct(0)
	if err != nil || got != DefaultPerpDepthRangePct {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := ParsePerpDepthRangePct(10); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestParsePerpDepthExchange(t *testing.T) {
	got, err := ParsePerpDepthExchange("")
	if err != nil || got != "all" {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := ParsePerpDepthExchange("okx"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestBuildPerpDepthVenue_OnePercentBand(t *testing.T) {
	book := &RawOrderBook{
		Symbol: "BTCUSDT",
		Bids: []PriceLevel{
			{Price: 100, Quantity: 10},
			{Price: 99.5, Quantity: 402010.05}, // ~40M at 99.5
			{Price: 90, Quantity: 1},           // outside 1%
		},
		Asks: []PriceLevel{
			{Price: 100.1, Quantity: 5},
			{Price: 100.5, Quantity: 149253.73}, // ~15M at 100.5
			{Price: 110, Quantity: 1},           // outside 1%
		},
	}
	got := BuildPerpDepthVenue(ExchangeBinance, "BTCUSDT", book, 1)
	if got.Error != "" {
		t.Fatal(got.Error)
	}
	h := got.Headline
	if h.RangePct != 1 {
		t.Fatalf("headline pct %v", h.RangePct)
	}
	if h.Below.Notional < 39_000_000 || h.Below.Notional > 42_000_000 {
		t.Fatalf("bid below %v", h.Below.Notional)
	}
	if h.Above.Notional < 14_000_000 || h.Above.Notional > 16_000_000 {
		t.Fatalf("ask above %v", h.Above.Notional)
	}
	if h.Lean != PerpDepthLeanBid {
		t.Fatalf("lean %s", h.Lean)
	}
	if !h.Complete {
		t.Fatalf("cover bid=%v ask=%v", got.CoverBidPct, got.CoverAskPct)
	}
}

func TestCombinePerpDepthVenues(t *testing.T) {
	mk := func(ex Exchange, bidQty, askQty float64) PerpDepthVenue {
		return BuildPerpDepthVenue(ex, "BTCUSDT", &RawOrderBook{
			Bids: []PriceLevel{{Price: 99.5, Quantity: bidQty}, {Price: 100, Quantity: 1}},
			Asks: []PriceLevel{{Price: 100.1, Quantity: 1}, {Price: 100.5, Quantity: askQty}},
		}, 1)
	}
	got := CombinePerpDepthVenues("BTCUSDT", []PerpDepthVenue{
		mk(ExchangeBinance, 100_000, 50_000),
		mk(ExchangeBybit, 20_000, 10_000),
	}, 1)
	if got == nil || got.Headline.Below.Notional <= got.Headline.Above.Notional {
		t.Fatalf("%+v", got)
	}
	// 99.5*(100k+20k) vs 100.5*(50k+10k)
	if got.Headline.Below.Notional < 11_000_000 || got.Headline.Above.Notional < 5_000_000 {
		t.Fatalf("combined %+v", got.Headline)
	}
}

func TestBuildPerpDepthVenue_Empty(t *testing.T) {
	got := BuildPerpDepthVenue(ExchangeBybit, "ETHUSDT", nil, 1)
	if got.Error == "" {
		t.Fatalf("%+v", got)
	}
}
