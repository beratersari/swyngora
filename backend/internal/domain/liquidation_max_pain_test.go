package domain

import (
	"math"
	"testing"
)

func TestPickMaxPainPocket_PicksLargestNotClosest(t *testing.T) {
	bands := []HuntBand{
		{Side: LiquidationSideShort, Direction: "up", Price: 101, MovePct: 1, EstNotional: 100, Source: "model"},
		{Side: LiquidationSideShort, Direction: "up", Price: 110, MovePct: 10, EstNotional: 5_000, Source: "model"},
	}
	got := PickMaxPainPocket(bands, "binance")
	if got == nil || got.Price != 110 || got.Notional != 5_000 {
		t.Fatalf("want farthest large pocket, got %+v", got)
	}
	if got.Side != LiquidationSideShort || got.Exchange != "binance" {
		t.Fatalf("%+v", got)
	}
}

func TestPickMaxPainPocket_IncludesObserved(t *testing.T) {
	bands := []HuntBand{
		{Side: LiquidationSideLong, Direction: "down", Price: 99, MovePct: -1, EstNotional: 200, Source: "model"},
		{Side: LiquidationSideLong, Direction: "down", Price: 95, MovePct: -5, ObservedNotional: 800, Source: "observed"},
	}
	got := PickMaxPainPocket(bands, "bybit")
	if got == nil || got.Price != 95 || got.Notional != 800 {
		t.Fatalf("observed cluster should win: %+v", got)
	}
}

func TestMaxPainFromVenue_IgnoresHuntTarget(t *testing.T) {
	in := testHuntInputsCrowdedShorts()
	// Cheap close cluster vs a much larger far cluster — hunt may prefer close.
	in.Asks = []ImpactSourceLevel{
		{Price: 100.10, Quantity: 0.1},
		{Price: 101.00, Quantity: 0.1},
	}
	venue := BuildHuntVenue(in)
	got := MaxPainFromVenue(venue)
	if got.Above == nil || got.Below == nil {
		t.Fatalf("want both sides: %+v", got)
	}
	if got.Above.Side != LiquidationSideShort || got.Below.Side != LiquidationSideLong {
		t.Fatalf("sides %+v / %+v", got.Above, got.Below)
	}
	if got.Above.Notional <= 0 || got.Below.Notional <= 0 {
		t.Fatalf("empty notional %+v", got)
	}
	if venue.UpHunt.Target.Price != 0 && got.Above.Price == venue.UpHunt.Target.Price && got.Above.Notional < HuntBandPressure(venue.UpPressure[0]) {
		t.Fatalf("max pain should not be forced to the hunt target: hunt=%v pain=%v", venue.UpHunt.Target, got.Above)
	}
}

func TestCombineMaxPainPockets_PicksLargerVenueNotAverage(t *testing.T) {
	small := MaxPainVenue{
		Exchange: ExchangeBinance, Price: 100,
		Above: &MaxPainPocket{Exchange: "binance", Price: 105, Notional: 100, Side: LiquidationSideShort},
		Below: &MaxPainPocket{Exchange: "binance", Price: 90, Notional: 9_000, Side: LiquidationSideLong},
	}
	large := MaxPainVenue{
		Exchange: ExchangeBybit, Price: 100,
		Above: &MaxPainPocket{Exchange: "bybit", Price: 112, Notional: 8_000, Side: LiquidationSideShort},
		Below: &MaxPainPocket{Exchange: "bybit", Price: 94, Notional: 200, Side: LiquidationSideLong},
	}
	above, below := CombineMaxPainPockets([]MaxPainVenue{small, large})
	if above == nil || above.Exchange != "bybit" || above.Price != 112 {
		t.Fatalf("above %+v", above)
	}
	if below == nil || below.Exchange != "binance" || below.Price != 90 {
		t.Fatalf("below %+v", below)
	}
	if math.Abs(above.Price-108.5) < 0.01 {
		t.Fatal("must not average prices")
	}
}

func TestMaxPainSummary_BothSides(t *testing.T) {
	s := MaxPainSummary(
		&MaxPainPocket{Price: 110, MovePct: 10, Notional: 1e6},
		&MaxPainPocket{Price: 90, MovePct: -10, Notional: 2e6},
	)
	if s == "" || s[0] == 'N' {
		t.Fatalf("%q", s)
	}
}
