package domain

import "testing"

func TestWalkBookToPrice_BuyClearsThroughTarget(t *testing.T) {
	asks := []ImpactSourceLevel{
		{Price: 100.10, Quantity: 1},
		{Price: 100.50, Quantity: 2},
		{Price: 101.00, Quantity: 4},
		{Price: 103.00, Quantity: 10},
	}
	got := WalkBookToPrice(ImpactSideBuy, 100, asks, 101)
	if !got.Reachable || got.LevelsUsed != 3 || got.Quantity != 7 {
		t.Fatalf("%+v", got)
	}
	if got.Notional != 100.10+2*100.50+4*101 {
		t.Fatalf("notional %v", got.Notional)
	}
}

func TestWalkBookToPrice_SellAndExhausted(t *testing.T) {
	bids := []ImpactSourceLevel{
		{Price: 99.90, Quantity: 1},
		{Price: 99.50, Quantity: 1},
	}
	ok := WalkBookToPrice(ImpactSideSell, 100, bids, 99.50)
	if !ok.Reachable || ok.Quantity != 2 {
		t.Fatalf("%+v", ok)
	}
	miss := WalkBookToPrice(ImpactSideSell, 100, bids, 90)
	if miss.Reachable || !miss.Exhausted || miss.MaxReachablePrice != 99.50 {
		t.Fatalf("%+v", miss)
	}
}

func TestWalkBookQuantity_Partial(t *testing.T) {
	lv := []ImpactSourceLevel{{Price: 10, Quantity: 2}, {Price: 11, Quantity: 2}}
	filled, spent, avg, end, exh := WalkBookQuantity(lv, 3)
	if filled != 3 || spent != 2*10+11 || avg != spent/3 || end != 11 || exh {
		t.Fatalf("%v %v %v %v %v", filled, spent, avg, end, exh)
	}
}
