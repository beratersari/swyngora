package domain

import (
	"math"
	"testing"
	"time"
)

func TestBuildOrderCount_RatioFromBuyAndSellCounts(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	var prints []TakerPrint
	for i := 0; i < 8; i++ {
		prints = append(prints, TakerPrint{Side: TakerSideBuy, Time: now.Add(-20 * time.Second)})
	}
	for i := 0; i < 2; i++ {
		prints = append(prints, TakerPrint{Side: TakerSideSell, Time: now.Add(-20 * time.Second)})
	}
	for i := 0; i < 3; i++ {
		prints = append(prints, TakerPrint{Side: TakerSideSell, Time: now.Add(-10 * time.Minute)})
	}
	got := BuildOrderCount(ExchangeBinance, "BTCUSDT", prints, now)
	var m1, m5 OrderCountWindow
	for _, w := range got.Windows {
		switch w.Window {
		case OrderCount1m:
			m1 = w
		case OrderCount5m:
			m5 = w
		}
	}
	if m1.BuyCount != 8 || m1.SellCount != 2 || m1.Delta != 6 || math.Abs(m1.Ratio-0.6) > 0.001 || m1.Lean != OrderCountLeanBuy {
		t.Fatalf("1m %+v", m1)
	}
	if m5.Total != 10 || m5.BuyCount != 8 {
		t.Fatalf("5m should ignore the 10m-old sells: %+v", m5)
	}
}

func TestCombineOrderCount_SumsThenDivides(t *testing.T) {
	a := OrderCountVenue{Windows: []OrderCountWindow{{Window: OrderCount5m, BuyCount: 3, SellCount: 1, Complete: true}}}
	b := OrderCountVenue{Windows: []OrderCountWindow{{Window: OrderCount5m, BuyCount: 1, SellCount: 3, Complete: true}}}
	got := CombineOrderCount("BTCUSDT", []OrderCountVenue{a, b})
	if got == nil || len(got.Windows) == 0 {
		t.Fatal("expected combined")
	}
	w := got.Windows[1]
	if w.Window != OrderCount5m || w.BuyCount != 4 || w.SellCount != 4 || w.Ratio != 0 || w.Lean != OrderCountLeanBalanced {
		t.Fatalf("%+v", w)
	}
}
