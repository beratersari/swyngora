package domain

import (
	"strings"
	"testing"
	"time"
)

func bounceBars() []OHLCBar {
	// Price sits near 100 three separate times, then lifts to 110.
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	seq := []float64{
		108, 105, 101, 100, 100.2, 104, 107,
		106, 103, 100.1, 99.8, 103, 108,
		107, 104, 100.3, 100, 105, 109, 110, 110.2,
	}
	out := make([]OHLCBar, len(seq))
	for i, c := range seq {
		lo, hi := c-0.4, c+0.4
		if c <= 101 {
			lo = 99.6
		}
		out[i] = OHLCBar{
			Time: now.Add(time.Duration(i) * time.Hour),
			Open: c, High: hi, Low: lo, Close: c, QuoteVol: 1000,
		}
	}
	return out
}

func TestFindPriceLevels_SupportTestsAndBook(t *testing.T) {
	bars := bounceBars()
	book := &RawOrderBook{
		Bids: []PriceLevel{
			{Price: 100, Quantity: 50}, {Price: 99.8, Quantity: 20},
			{Price: 104.5, Quantity: 15}, {Price: 107.4, Quantity: 12},
		},
		Asks: []PriceLevel{{Price: 110.5, Quantity: 10}},
	}
	zones := FindPriceLevels(bars, book, nil, 110.2)
	var sup *PriceLevelZone
	for i := range zones {
		if zones[i].Kind == LevelKindSupport && zones[i].Price > 98 && zones[i].Price < 102 {
			sup = &zones[i]
			break
		}
	}
	if sup == nil {
		t.Fatalf("no support near 100: %+v", zones)
	}
	if sup.Tests < 2 {
		t.Fatalf("tests %d zone=%+v", sup.Tests, sup)
	}
	if sup.BidNotional < 1000 {
		t.Fatalf("bid liq %v", sup.BidNotional)
	}
	if sup.DistancePct >= 0 {
		t.Fatalf("support should be below price %+v", sup)
	}
}

func TestScoreLevelBreakout_Resistance(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	bars := make([]OHLCBar, 30)
	for i := 0; i < 30; i++ {
		c := 100.0
		vol := 10.0
		if i >= 24 {
			c = 105 + float64(i-24)
			vol = 40
		}
		bars[i] = OHLCBar{Time: now.Add(time.Duration(i) * time.Hour), Open: c, High: c + 0.2, Low: c - 0.2, Close: c, QuoteVol: vol}
	}
	z := PriceLevelZone{Kind: LevelKindResistance, Price: 104, Low: 103.5, High: 104.5, DistancePct: 2}
	taker := &TakerVenueFlow{Windows: []TakerWindowFlow{
		SummarizeTakerWindow(80, 20, TakerWindow1h, true),
	}}
	got := ScoreLevelBreakout(z, bars, 107, taker)
	if got == nil || got.Status != LevelBreakBroken {
		t.Fatalf("%+v", got)
	}
	if got.Score < 30 || got.Taker != TakerSideBuy {
		t.Fatalf("score %+v", got)
	}
}

func TestExplainLevels(t *testing.T) {
	sup := []PriceLevelZone{{Kind: LevelKindSupport, Price: 100, DistancePct: -4.5, Tests: 3}}
	res := []PriceLevelZone{{Kind: LevelKindResistance, Price: 110, DistancePct: 5, Tests: 2}}
	got := ExplainLevels("BTCUSDT", 105, sup, res, nil)
	if !strings.Contains(got, "support") || !strings.Contains(got, "resistance") {
		t.Fatalf("%s", got)
	}
}
