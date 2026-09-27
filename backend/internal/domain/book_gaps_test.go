package domain

import "testing"

func gapBook() RawOrderBook {
	asks := []PriceLevel{
		{Price: 100.05, Quantity: 10},
		{Price: 100.12, Quantity: 10},
		{Price: 101.2, Quantity: 8},
	}
	bids := make([]PriceLevel, 0, 20)
	for i := 1; i <= 20; i++ {
		bids = append(bids, PriceLevel{Price: 100 - float64(i)*0.1, Quantity: 20})
	}
	return RawOrderBook{Symbol: "SOLUSDT", Bids: bids, Asks: asks}
}

func TestFindBookGaps_ThinAsksAreEasierUp(t *testing.T) {
	got := FindBookGaps("SOLUSDT", "binance", []RawOrderBook{gapBook()}, 100, 2)
	if got.Easier != BookGapEasierUp {
		t.Fatalf("easier %+v", got)
	}
	if len(got.Gaps) == 0 || got.Gaps[0].Direction != BookGapEasierUp {
		t.Fatalf("gaps %+v", got.Gaps)
	}
	if got.Gaps[0].WidthPct < bookGapMinWidthPct {
		t.Fatalf("width %v", got.Gaps[0].WidthPct)
	}
	if got.UpGapPct <= got.DownGapPct {
		t.Fatalf("ask empty share %v bid %v", got.UpGapPct, got.DownGapPct)
	}
}

func TestFindBookGaps_BalancedIsEven(t *testing.T) {
	level := func(px float64) PriceLevel { return PriceLevel{Price: px, Quantity: 5} }
	raw := RawOrderBook{
		Bids: []PriceLevel{level(99.9), level(99.7), level(99.4), level(99.1)},
		Asks: []PriceLevel{level(100.1), level(100.3), level(100.6), level(100.9)},
	}
	got := FindBookGaps("BTCUSDT", "all", []RawOrderBook{raw}, 100, 1)
	if got.Easier != BookGapEasierEven {
		t.Fatalf("%+v summary %s", got.Easier, got.Summary)
	}
}

func TestBuildBookGapScan_ThinnerSideFirst(t *testing.T) {
	hits := []BookGapScanHit{
		{Symbol: "BTCUSDT", Easier: BookGapEasierEven, UpNotional: "100", DownNotional: "100", WidestPct: 0.2},
		{Symbol: "SOLUSDT", Easier: BookGapEasierUp, UpNotional: "10", DownNotional: "400", WidestPct: 0.8},
		{Symbol: "ETHUSDT", Easier: BookGapEasierDown, UpNotional: "200", DownNotional: "80", WidestPct: 0.4},
	}
	got := BuildBookGapScan("binance", "usdt", 2, hits)
	if got.Hits[0].Symbol != "SOLUSDT" || got.Quote != "USDT" {
		t.Fatalf("%+v", got.Hits)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
}
