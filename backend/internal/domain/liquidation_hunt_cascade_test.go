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
	if got.Steps[0].Easier || got.Steps[0].Index != 1 {
		t.Fatalf("first step must not be easier: %+v", got.Steps[0])
	}
	if math.Abs(got.Steps[0].MovePct-0.4) > 0.05 {
		t.Fatalf("move %+v", got.Steps[0])
	}
	if got.Steps[0].ZoneNotional != 50 || got.Steps[2].CumulativeNotional != 50+180+300 {
		t.Fatalf("notionals %+v", got)
	}
}

func TestBuildHuntCascadePath_PriorZoneCheapensNextHop(t *testing.T) {
	// Fat first zone; thin book between 101 and 102 so 50% cascade fill walks through.
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
	if second.PriorCascadeNotional <= 0 {
		t.Fatalf("want prior cascade: %+v", second)
	}
	if !second.Easier && !second.SelfFueling {
		t.Fatalf("second hop should be cheaper or self-fueling: %+v", second)
	}
	if second.Remaining.Notional >= second.Incremental.Notional && !second.SelfFueling {
		t.Fatalf("remaining should drop: inc=%v rem=%v", second.Incremental.Notional, second.Remaining.Notional)
	}
	if !got.ChainEasier {
		t.Fatalf("path should be chain-easier: %s", got.Summary)
	}
}

func TestBuildHuntCascadePath_ThickBookDoesNotSelfFuel(t *testing.T) {
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
	if got.Steps[1].AssistancePct > 25 {
		t.Fatalf("tiny first zone should barely help: %+v", got.Steps[1])
	}
}

func TestBuildHuntCascadePath_UnreachableHopIsNotEasier(t *testing.T) {
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
		t.Fatalf("unreachable hop must not look easier: %+v", got.Steps[1])
	}
	if got.Steps[1].AssistancePct != 0 {
		t.Fatalf("unreachable assistance %+v", got.Steps[1])
	}
}

func TestBuildHuntCascadePath_DownAndSkipsWrongSide(t *testing.T) {
	bands := []HuntBand{
		{Direction: "down", Leverage: 100, Price: 99.40, EstNotional: 10},
		{Direction: "down", Leverage: 50, Price: 98.40, EstNotional: 20},
		{Direction: "down", Leverage: 25, Price: 101, EstNotional: 99}, // wrong side
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
}
