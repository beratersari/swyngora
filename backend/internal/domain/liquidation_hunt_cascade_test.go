package domain

import (
	"math"
	"testing"
)

func TestBuildHuntCascadePath_OrdersClosestFirst(t *testing.T) {
	bands := []HuntBand{
		{Side: LiquidationSideShort, Direction: "up", Leverage: 25, Price: 103.6, MovePct: 3.6, EstNotional: 300},
		{Side: LiquidationSideShort, Direction: "up", Leverage: 125, Price: 100.4, MovePct: 0.4, EstNotional: 50},
		{Side: LiquidationSideShort, Direction: "up", Leverage: 50, Price: 101.6, MovePct: 1.6, EstNotional: 180},
	}
	asks := []ImpactSourceLevel{
		{Price: 100.20, Quantity: 2},
		{Price: 100.40, Quantity: 2},
		{Price: 101.60, Quantity: 4},
		{Price: 103.60, Quantity: 8},
	}
	got := BuildHuntCascadePath("up", bands, 100, asks, ImpactSideBuy)
	if len(got.Steps) != 3 {
		t.Fatalf("steps %+v", got.Steps)
	}
	if got.Steps[0].Band.Leverage != 125 || got.Steps[1].Band.Leverage != 50 || got.Steps[2].Band.Leverage != 25 {
		t.Fatalf("order %+v %+v %+v", got.Steps[0].Band, got.Steps[1].Band, got.Steps[2].Band)
	}
	if got.Steps[0].Role != HuntCascadeRoleStart || got.Steps[0].SelfFueling {
		t.Fatalf("first step: %+v", got.Steps[0])
	}
	if math.Abs(got.Steps[0].MovePct-0.4) > 0.05 {
		t.Fatalf("move %+v", got.Steps[0])
	}
	if got.Steps[0].ZoneNotional != 50 || got.Steps[2].CumulativeNotional != 50+180+300 {
		t.Fatalf("notionals %+v", got)
	}
}

func TestBuildHuntCascadePath_SelfFuelingWhenFuelCoversHop(t *testing.T) {
	bands := []HuntBand{
		{Direction: "up", Leverage: 100, Price: 101, EstNotional: 20_000},
		{Direction: "up", Leverage: 50, Price: 102, EstNotional: 5_000},
	}
	asks := []ImpactSourceLevel{
		{Price: 100.50, Quantity: 2},
		{Price: 101.00, Quantity: 2},
		{Price: 101.40, Quantity: 1},
		{Price: 102.00, Quantity: 1},
	}
	got := BuildHuntCascadePath("up", bands, 100, asks, ImpactSideBuy)
	if len(got.Steps) != 2 {
		t.Fatalf("%+v", got)
	}
	second := got.Steps[1]
	if second.PriorCascadeNotional != 10_000 {
		t.Fatalf("fuel %v", second.PriorCascadeNotional)
	}
	if !second.SelfFueling || second.Role != HuntCascadeRoleSelf {
		t.Fatalf("want self-fueling from fuel>=hop: %+v", second)
	}
	if second.Remaining.Notional != 0 || !second.Remaining.Reachable {
		t.Fatalf("remaining must be the final zero walk: %+v", second.Remaining)
	}
	if second.AssistancePct == nil || *second.AssistancePct != 100 || second.Strength == nil || *second.Strength != 100 {
		t.Fatalf("strength %+v assist %+v", second.Strength, second.AssistancePct)
	}
	if got.FeedsUntilIndex != 2 || got.StallsAtIndex != 0 {
		t.Fatalf("feeds/stall %+v", got)
	}
}

func TestBuildHuntCascadePath_TinyFuelAtWallIsNotSelfFueling(t *testing.T) {
	// First ask beyond 101 is already at 102. A $1 consume would set endPrice=102
	// under the old price-touch rule. Fuel must cover WalkBookToPrice's hop cost.
	bands := []HuntBand{
		{Direction: "up", Leverage: 100, Price: 101, EstNotional: 4}, // fuel = 2
		{Direction: "up", Leverage: 50, Price: 102, EstNotional: 5_000},
	}
	asks := []ImpactSourceLevel{
		{Price: 100.50, Quantity: 2},
		{Price: 101.00, Quantity: 2},
		{Price: 102.00, Quantity: 20},
	}
	got := BuildHuntCascadePath("up", bands, 100, asks, ImpactSideBuy)
	if len(got.Steps) != 2 {
		t.Fatalf("%+v", got)
	}
	second := got.Steps[1]
	if !second.Incremental.Reachable || second.Incremental.Notional < 100 {
		t.Fatalf("hop should be the 102 wall: %+v", second.Incremental)
	}
	if second.SelfFueling {
		t.Fatalf("tiny fuel must not self-fuel: %+v", second)
	}
	if second.AssistancePct != nil && *second.AssistancePct == 100 {
		t.Fatalf("assistance left at 100 after non-self hop: %+v", second)
	}
	if second.Remaining.Notional <= 0 && second.Remaining.Reachable {
		t.Fatalf("remaining must be the leftover desk walk, not a zero self-fuel walk: %+v", second.Remaining)
	}
	if second.Role != HuntCascadeRoleHelped && second.Role != HuntCascadeRoleStall {
		t.Fatalf("role %+v", second)
	}
	if got.StallsAtIndex != 2 {
		t.Fatalf("should stall at hop 2: %+v", got)
	}
}

func TestBuildHuntCascadePath_SpentFuelIsNotReused(t *testing.T) {
	// Hop 2 costs almost all of zone-1 fuel. Hop 3 must see leftover + zone-2 add,
	// not 50% of (zone1+zone2) as if nothing was spent.
	bands := []HuntBand{
		{Direction: "up", Leverage: 100, Price: 101, EstNotional: 20_000}, // adds 10_000
		{Direction: "up", Leverage: 50, Price: 102, EstNotional: 10_000},  // adds 5_000 if reached
		{Direction: "up", Leverage: 25, Price: 104, EstNotional: 8_000},
	}
	asks := []ImpactSourceLevel{
		{Price: 100.50, Quantity: 1},
		{Price: 101.00, Quantity: 1},
		{Price: 101.20, Quantity: 40},
		{Price: 102.00, Quantity: 40},
		{Price: 103.00, Quantity: 40},
		{Price: 104.00, Quantity: 40},
	}
	got := BuildHuntCascadePath("up", bands, 100, asks, ImpactSideBuy)
	if len(got.Steps) != 3 {
		t.Fatalf("%+v", got)
	}
	if got.Steps[0].FuelAdds != 10_000 || got.Steps[0].FuelSpent != 0 {
		t.Fatalf("start fuel %+v", got.Steps[0])
	}
	if got.Steps[1].PriorCascadeNotional != 10_000 {
		t.Fatalf("hop2 pool %+v", got.Steps[1])
	}
	if got.Steps[1].FuelSpent <= 0 || got.Steps[1].FuelSpent >= 10_000 {
		t.Fatalf("hop2 should spend some but not all: %+v", got.Steps[1])
	}
	wantNext := got.Steps[1].FuelLeft
	if got.Steps[1].Reachable && got.Steps[1].ZoneEst == "model" {
		wantNext += got.Steps[1].FuelAdds
	}
	if math.Abs(got.Steps[2].PriorCascadeNotional-wantNext) > 1e-6 {
		t.Fatalf("hop3 pool %v want leftover+add %v (not reused spent). step2=%+v",
			got.Steps[2].PriorCascadeNotional, wantNext, got.Steps[1])
	}
	reusedAll := (20_000 + 10_000) * HuntCascadeFillRate
	if math.Abs(got.Steps[2].PriorCascadeNotional-reusedAll) < 1 {
		t.Fatalf("hop3 reused full prior est as fuel: %v", got.Steps[2].PriorCascadeNotional)
	}
}

func TestBuildHuntCascadePath_ThickBookStalls(t *testing.T) {
	asks := make([]ImpactSourceLevel, 0, 20)
	for i := 0; i < 20; i++ {
		asks = append(asks, ImpactSourceLevel{Price: 100.10 + float64(i)*0.4, Quantity: 400})
	}
	bands := []HuntBand{
		{Direction: "up", Leverage: 100, Price: 100.60, EstNotional: 200},
		{Direction: "up", Leverage: 25, Price: 104.00, EstNotional: 800},
	}
	got := BuildHuntCascadePath("up", bands, 100, asks, ImpactSideBuy)
	if len(got.Steps) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Steps[1].SelfFueling {
		t.Fatalf("thick book must not self-fuel: %+v", got.Steps[1])
	}
	if got.Steps[1].AssistancePct != nil && *got.Steps[1].AssistancePct > 25 {
		t.Fatalf("tiny first zone should barely help: %+v", got.Steps[1])
	}
	if got.StallsAtIndex != 2 {
		t.Fatalf("stall %+v", got)
	}
}

func TestBuildHuntCascadePath_UnreachableHopHasNoAssistance(t *testing.T) {
	bands := []HuntBand{
		{Direction: "up", Leverage: 100, Price: 101, EstNotional: 50_000},
		{Direction: "up", Leverage: 10, Price: 110, EstNotional: 80_000},
	}
	asks := []ImpactSourceLevel{
		{Price: 100.50, Quantity: 1},
		{Price: 101.00, Quantity: 1},
	}
	got := BuildHuntCascadePath("up", bands, 100, asks, ImpactSideBuy)
	if len(got.Steps) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Steps[1].Reachable || got.Steps[1].Easier || got.Steps[1].SelfFueling {
		t.Fatalf("unreachable hop: %+v", got.Steps[1])
	}
	if got.Steps[1].AssistancePct != nil || got.Steps[1].Strength != nil {
		t.Fatalf("do not invent assistance/strength: %+v", got.Steps[1])
	}
	if got.Steps[1].Role != HuntCascadeRoleUnreachable && got.Steps[1].Role != HuntCascadeRoleMissing {
		t.Fatalf("role %+v", got.Steps[1])
	}
}

func TestBuildHuntCascadePath_MissingBook(t *testing.T) {
	got := BuildHuntCascadePath("up", []HuntBand{{Direction: "up", Price: 101, EstNotional: 10}}, 100, nil, ImpactSideBuy)
	if len(got.Steps) != 1 || got.Steps[0].Role != HuntCascadeRoleMissing {
		t.Fatalf("%+v", got)
	}
	if got.Steps[0].Strength != nil || got.Steps[0].AssistancePct != nil {
		t.Fatalf("missing book must not invent scores: %+v", got.Steps[0])
	}
	if got.StallsAtIndex != 1 {
		t.Fatalf("stall at start %+v", got)
	}
}

func TestBuildHuntCascadePath_DownAndSkipsWrongSide(t *testing.T) {
	bands := []HuntBand{
		{Direction: "down", Leverage: 100, Price: 99.40, EstNotional: 10},
		{Direction: "down", Leverage: 50, Price: 98.40, EstNotional: 20},
		{Direction: "down", Leverage: 25, Price: 101, EstNotional: 99},
	}
	bids := []ImpactSourceLevel{
		{Price: 99.80, Quantity: 2},
		{Price: 99.40, Quantity: 2},
		{Price: 98.40, Quantity: 4},
	}
	got := BuildHuntCascadePath("down", bands, 100, bids, ImpactSideSell)
	if len(got.Steps) != 2 || got.Steps[0].Band.Price != 99.40 || got.Steps[1].Band.Price != 98.40 {
		t.Fatalf("%+v", got.Steps)
	}
	if got.Direction != "down" {
		t.Fatalf("%s", got.Direction)
	}
}

func TestBuildHuntVenue_AttachesCascadePaths(t *testing.T) {
	got := BuildHuntVenue(HuntInputs{
		Exchange:   ExchangeBinance,
		Symbol:     "BTCUSDT",
		Price:      100,
		OIValue:    2_000_000,
		LongShare:  0.5,
		ShortShare: 0.5,
		Asks: []ImpactSourceLevel{
			{Price: 100.20, Quantity: 3},
			{Price: 100.80, Quantity: 3},
			{Price: 102.00, Quantity: 5},
			{Price: 110.00, Quantity: 8},
		},
		Bids: []ImpactSourceLevel{
			{Price: 99.80, Quantity: 3},
			{Price: 99.20, Quantity: 3},
			{Price: 98.00, Quantity: 5},
			{Price: 90.00, Quantity: 8},
		},
	})
	if len(got.UpCascade.Steps) == 0 || got.UpCascade.Direction != "up" {
		t.Fatalf("up path %+v", got.UpCascade)
	}
	if len(got.DownCascade.Steps) == 0 || got.DownCascade.Direction != "down" {
		t.Fatalf("down path %+v", got.DownCascade)
	}
	if got.UpCascade.Steps[0].MovePct <= 0 || got.DownCascade.Steps[0].MovePct >= 0 {
		t.Fatalf("signs up=%v down=%v", got.UpCascade.Steps[0].MovePct, got.DownCascade.Steps[0].MovePct)
	}
	if got.UpCascade.Steps[0].Role != HuntCascadeRoleStart {
		t.Fatalf("start role %+v", got.UpCascade.Steps[0])
	}
}
