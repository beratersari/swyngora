package domain

import (
	"math"
	"testing"
	"time"
)

func TestHuntLiqDistance_10x(t *testing.T) {
	d := HuntLiqDistance(10, HuntMaintenanceMargin)
	want := 0.1 - HuntMaintenanceMargin
	if math.Abs(d-want) > 1e-12 {
		t.Fatalf("got %v want %v", d, want)
	}
	if HuntLiqDistance(1000, HuntMaintenanceMargin) < 0.002 {
		t.Fatal("floor")
	}
}

func TestBlendAccountShare_Haircut(t *testing.T) {
	long, short := BlendAccountShare(0.80)
	if long >= 0.80 || long <= 0.50 || math.Abs(long+short-1) > 1e-12 {
		t.Fatalf("%v %v", long, short)
	}
}

func TestTiltLeverageMix_CrowdedShiftsHigh(t *testing.T) {
	base := DefaultHuntLeverageMix
	crowded := TiltLeverageMix(base, true)
	calm := TiltLeverageMix(base, false)
	if crowded[len(crowded)-1].Weight <= base[len(base)-1].Weight {
		t.Fatalf("high-L should rise when crowded: %+v", crowded)
	}
	if calm[0].Weight <= base[0].Weight {
		t.Fatalf("low-L should rise when not crowded: %+v", calm)
	}
	var s float64
	for _, b := range crowded {
		s += b.Weight
	}
	if math.Abs(s-1) > 1e-9 {
		t.Fatalf("weights %v", s)
	}
}

func TestBuildHuntVenue_ThinBookLargeShortsLooksProfitable(t *testing.T) {
	// Mid 100. Thin asks so +~2% (50x shorts) is cheap. Huge short OI.
	in := HuntInputs{
		Exchange:    ExchangeBinance,
		Symbol:      "BTCUSDT",
		Price:       100,
		OIValue:     10_000_000,
		LongShare:   0.35,
		ShortShare:  0.65,
		FundingRate: -0.0002, // shorts pay — crowded shorts
		Asks: []ImpactSourceLevel{
			{Price: 100.10, Quantity: 1},
			{Price: 101.00, Quantity: 1},
			{Price: 102.00, Quantity: 2},
			{Price: 105.00, Quantity: 2},
			{Price: 110.00, Quantity: 2},
		},
		Bids: []ImpactSourceLevel{
			{Price: 99.90, Quantity: 1},
			{Price: 99.00, Quantity: 1},
			{Price: 90.00, Quantity: 5},
		},
	}
	got := BuildHuntVenue(in)
	if got.EstShortNotional <= got.EstLongNotional {
		t.Fatalf("shorts should dominate proxy: %+v", got)
	}
	if len(got.UpPressure) == 0 {
		t.Fatal("expected up pressure")
	}
	if !got.UpHunt.Spot.Reachable {
		t.Fatalf("up hunt should reach on this book: %+v", got.UpHunt)
	}
	if got.UpHunt.EstLiquidated <= 0 {
		t.Fatalf("expected short liq: %+v", got.UpHunt)
	}
	if got.UpHunt.BookOnlyPnl >= 0 {
		t.Fatalf("unwinding on the old bid should lose the spread: %+v", got.UpHunt)
	}
	if got.UpHunt.HouseEdge != "profit" {
		t.Fatalf("cascade + take should profit on thin book / fat shorts: %+v", got.UpHunt)
	}
}

func TestBuildHuntVenue_ThickBookTinyOILoses(t *testing.T) {
	asks := make([]ImpactSourceLevel, 0, 20)
	for i := 0; i < 20; i++ {
		asks = append(asks, ImpactSourceLevel{Price: 100.10 + float64(i)*0.5, Quantity: 500})
	}
	bids := make([]ImpactSourceLevel, 0, 20)
	for i := 0; i < 20; i++ {
		bids = append(bids, ImpactSourceLevel{Price: 99.90 - float64(i)*0.5, Quantity: 500})
	}
	got := BuildHuntVenue(HuntInputs{
		Exchange:   ExchangeBybit,
		Symbol:     "ETHUSDT",
		Price:      100,
		OIValue:    5_000,
		LongShare:  0.5,
		ShortShare: 0.5,
		Asks:       asks,
		Bids:       bids,
	})
	if got.UpHunt.HouseEdge == "profit" && got.UpHunt.NetWithCascade > 0 && got.UpHunt.Spot.Notional < 1000 {
		t.Fatalf("thick book should be expensive: %+v", got.UpHunt)
	}
	if got.UpHunt.HouseEdge != "loss" && got.UpHunt.HouseEdge != "unreachable" {
		t.Fatalf("expected loss or unreachable, got %+v", got.UpHunt)
	}
}

func TestClusterHuntLiquidations(t *testing.T) {
	mid := 64000.0
	ev := []LiquidationEvent{
		{Side: LiquidationSideLong, Price: 63680, Notional: 100, Time: time.Unix(1, 0)},
		{Side: LiquidationSideLong, Price: 63690, Notional: 50, Time: time.Unix(2, 0)},
		{Side: LiquidationSideShort, Price: 64640, Notional: 20, Time: time.Unix(3, 0)},
	}
	got := ClusterHuntLiquidations(ev, mid, 0.5)
	if len(got) < 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].Side != LiquidationSideLong || got[0].Notional != 150 {
		t.Fatalf("largest cluster %+v", got[0])
	}
}

func TestPickHuntTarget_PrefersReachableNearBand(t *testing.T) {
	// Book only covers ~+1.1%. 5x (+19.6%) must not win just because it
	// "liquidates everyone".
	asks := []ImpactSourceLevel{
		{Price: 100.10, Quantity: 2},
		{Price: 100.40, Quantity: 2},
		{Price: 100.80, Quantity: 3},
	}
	got := BuildHuntVenue(HuntInputs{
		Exchange:   ExchangeBinance,
		Symbol:     "BTCUSDT",
		Price:      100,
		OIValue:    2_000_000,
		LongShare:  0.5,
		ShortShare: 0.5,
		Asks:       asks,
		Bids: []ImpactSourceLevel{
			{Price: 99.90, Quantity: 2},
			{Price: 99.50, Quantity: 2},
		},
	})
	if got.UpHunt.Target.Leverage == 5 {
		t.Fatalf("must not pick unreachable 5x as primary: %+v", got.UpHunt.Target)
	}
	if got.UpHunt.Target.Price <= 0 {
		t.Fatalf("missing target %+v", got.UpHunt)
	}
}

func TestBuildHuntVenue_VenuesStaySeparate(t *testing.T) {
	a := BuildHuntVenue(HuntInputs{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Price: 100, OIValue: 1, LongShare: 0.5, ShortShare: 0.5})
	b := BuildHuntVenue(HuntInputs{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Price: 100, OIValue: 1, LongShare: 0.5, ShortShare: 0.5})
	if a.Exchange == b.Exchange {
		t.Fatal("exchanges must stay labeled")
	}
}
