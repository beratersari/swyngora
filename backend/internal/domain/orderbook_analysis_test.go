package domain

import (
	"errors"
	"math"
	"strconv"
	"testing"
)

func TestParseRangePct(t *testing.T) {
	v, err := ParseRangePct(" 2.5 ")
	if err != nil || v != 2.5 {
		t.Fatalf("%v %v", v, err)
	}
	empty, err := ParseRangePct(" ")
	if err != nil || empty != 0 {
		t.Fatalf("empty %v %v", empty, err)
	}
	if _, err := ParseRangePct("-1"); err == nil || !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("neg: %v", err)
	}
	if _, err := ParseRangePct("nope"); err == nil {
		t.Fatal("want invalid")
	}
}

func TestClampRangePct(t *testing.T) {
	if ClampRangePct(0) != DefaultOrderBookRangePct {
		t.Fatal("default")
	}
	if ClampRangePct(0.01) != MinOrderBookRangePct {
		t.Fatal("min")
	}
	if ClampRangePct(50) != MaxOrderBookRangePct {
		t.Fatal("max")
	}
	if ClampRangePct(2) != 2 {
		t.Fatal("keep")
	}
}

func TestAnalyzeOrderBook_IgnoresFarDepth(t *testing.T) {
	// Mid ~100. Near band ±2% is 98–102. Far 80/120 must not drive pressure.
	raw := RawOrderBook{
		Symbol: "BTCUSDT",
		Bids: []PriceLevel{
			{Price: 99.9, Quantity: 1},
			{Price: 99.5, Quantity: 1},
			{Price: 99.0, Quantity: 1},
			{Price: 80, Quantity: 10_000}, // ~20% away
		},
		Asks: []PriceLevel{
			{Price: 100.1, Quantity: 1},
			{Price: 100.4, Quantity: 1},
			{Price: 100.8, Quantity: 1},
			{Price: 120, Quantity: 10_000},
		},
	}
	got := AnalyzeOrderBook(raw, 2)
	if got.RangePct != 2 || got.Pressure != OrderBookPressureBalanced {
		t.Fatalf("pressure=%s imb=%v", got.Pressure, got.Imbalance)
	}
	if got.BidLevels != 3 || got.AskLevels != 3 {
		t.Fatalf("levels bid=%d ask=%d (far size must be excluded)", got.BidLevels, got.AskLevels)
	}
	for _, w := range got.Walls {
		p, _ := strconv.ParseFloat(w.Price, 64)
		if p < 90 || p > 110 {
			t.Fatalf("wall outside band: %+v", w)
		}
	}
	if len(got.Bands) != 4 {
		t.Fatalf("bands=%d", len(got.Bands))
	}
}

func TestAnalyzeOrderBook_BuyPressureAndWall(t *testing.T) {
	raw := RawOrderBook{
		Symbol: "ETHUSDT",
		Bids: []PriceLevel{
			{Price: 100, Quantity: 2},
			{Price: 99.8, Quantity: 2},
			{Price: 99.5, Quantity: 2},
			{Price: 99.0, Quantity: 80}, // bid wall inside 2%
			{Price: 98.8, Quantity: 2},
		},
		Asks: []PriceLevel{
			{Price: 100.1, Quantity: 1},
			{Price: 100.3, Quantity: 1},
			{Price: 100.6, Quantity: 1},
			{Price: 101.0, Quantity: 1},
		},
	}
	got := AnalyzeOrderBook(raw, 2)
	if got.Pressure != OrderBookPressureBuy {
		t.Fatalf("want buy, got %s imb=%v", got.Pressure, got.Imbalance)
	}
	if got.Imbalance <= 0 {
		t.Fatalf("imbalance %v", got.Imbalance)
	}
	var sawBid bool
	for _, w := range got.Walls {
		if w.Side == "bid" {
			sawBid = true
			if w.Share <= 0 {
				t.Fatalf("wall share %+v", w)
			}
		}
	}
	if !sawBid {
		t.Fatalf("expected bid wall, walls=%+v", got.Walls)
	}
}

func TestAnalyzeOrderBook_SellPressure(t *testing.T) {
	raw := RawOrderBook{
		Symbol: "SOLUSDT",
		Bids:   []PriceLevel{{Price: 100, Quantity: 1}, {Price: 99.5, Quantity: 1}},
		Asks: []PriceLevel{
			{Price: 100.1, Quantity: 2},
			{Price: 100.4, Quantity: 2},
			{Price: 100.8, Quantity: 40},
			{Price: 101.2, Quantity: 2},
		},
	}
	got := AnalyzeOrderBook(raw, 2)
	if got.Pressure != OrderBookPressureSell || got.Imbalance >= 0 {
		t.Fatalf("want sell, got %s imb=%v", got.Pressure, got.Imbalance)
	}
}

func TestAnalyzeOrderBook_Empty(t *testing.T) {
	got := AnalyzeOrderBook(RawOrderBook{Symbol: "X"}, 2)
	if got.Pressure != OrderBookPressureBalanced || got.MidPrice != "" {
		t.Fatalf("%+v", got)
	}
}

func TestPressureFromImbalance(t *testing.T) {
	if pressureFromImbalance(0) != OrderBookPressureBalanced {
		t.Fatal("0")
	}
	if pressureFromImbalance(0.079) != OrderBookPressureBalanced {
		t.Fatal("under")
	}
	if pressureFromImbalance(0.08) != OrderBookPressureBuy {
		t.Fatal("buy")
	}
	if pressureFromImbalance(-0.2) != OrderBookPressureSell {
		t.Fatal("sell")
	}
}

func TestAnalyzeOrderBook_CoveredPct(t *testing.T) {
	raw := RawOrderBook{
		Bids: []PriceLevel{{Price: 99, Quantity: 1}},
		Asks: []PriceLevel{{Price: 101, Quantity: 1}},
	}
	got := AnalyzeOrderBook(raw, 2)
	bid, _ := strconv.ParseFloat(got.CoveredBidPct, 64)
	ask, _ := strconv.ParseFloat(got.CoveredAskPct, 64)
	if math.Abs(bid-1) > 0.05 || math.Abs(ask-1) > 0.05 {
		t.Fatalf("covered bid=%s ask=%s", got.CoveredBidPct, got.CoveredAskPct)
	}
}
