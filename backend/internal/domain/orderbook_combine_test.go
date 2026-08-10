package domain

import (
	"strconv"
	"testing"
)

func TestCombineOrderBooks_SumsSameBand(t *testing.T) {
	a := VenueRawBook{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT",
		Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 100, Quantity: 2}, {Price: 80, Quantity: 1000}},
			Asks: []PriceLevel{{Price: 100.2, Quantity: 1}},
		},
	}
	b := VenueRawBook{
		Exchange: ExchangeBybit, Symbol: "BTCUSDT",
		Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 99.8, Quantity: 3}},
			Asks: []PriceLevel{{Price: 100.1, Quantity: 4}, {Price: 120, Quantity: 1000}},
		},
	}
	mid := SharedBookMid([]VenueRawBook{a, b})
	if mid < 99 || mid > 101 {
		t.Fatalf("mid %v", mid)
	}
	got := CombineOrderBooks("BTCUSDT", mid, 2, []VenueRawBook{a, b})
	if got.VenueCount != 2 {
		t.Fatalf("venues %d", got.VenueCount)
	}
	bidQ, _ := strconv.ParseFloat(got.BidQuantity, 64)
	askQ, _ := strconv.ParseFloat(got.AskQuantity, 64)
	// Ask reach is the short side, so 99.8 is outside the symmetric ±%. 80/120 stay out.
	if bidQ < 1.9 || bidQ > 2.1 {
		t.Fatalf("bid qty %s (far 80 and 99.8 outside ±usedPct)", got.BidQuantity)
	}
	if askQ < 4.9 || askQ > 5.1 {
		t.Fatalf("ask qty %s (far 120 must be excluded)", got.AskQuantity)
	}
	if got.Pressure == "" {
		t.Fatalf("%+v", got)
	}
}

func TestCombineOrderBooks_BuyPressureFromTotal(t *testing.T) {
	books := []VenueRawBook{
		{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 100, Quantity: 10}},
			Asks: []PriceLevel{{Price: 100.2, Quantity: 1}},
		}},
		{Exchange: ExchangeCoinbase, Symbol: "ETH-USD", Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 99.9, Quantity: 10}},
			Asks: []PriceLevel{{Price: 100.3, Quantity: 1}},
		}},
	}
	mid := SharedBookMid(books)
	got := CombineOrderBooks("ETHUSDT", mid, 2, books)
	if got.Pressure != OrderBookPressureBuy || got.Imbalance <= 0 {
		t.Fatalf("want buy, %+v", got)
	}
}

func TestCombineOrderBooks_SkipsFailedVenue(t *testing.T) {
	books := []VenueRawBook{
		{Exchange: ExchangeBinance, Symbol: "X", Err: "down"},
		{Exchange: ExchangeBybit, Symbol: "XUSDT", Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 10, Quantity: 1}},
			Asks: []PriceLevel{{Price: 10.1, Quantity: 1}},
		}},
	}
	mid := SharedBookMid(books)
	got := CombineOrderBooks("XUSDT", mid, 2, books)
	if got.VenueCount != 1 || len(got.Venues) != 2 || got.Venues[0].Error == "" {
		t.Fatalf("%+v", got)
	}
}

func TestSharedBookMid_Empty(t *testing.T) {
	if SharedBookMid(nil) != 0 {
		t.Fatal("empty")
	}
}

func TestCombineOrderBooks_UsesOverlapWhenRequestedTooWide(t *testing.T) {
	// Mid ~100. Venue A reaches 2%+; venue B only has depth to ~0.5% on each side
	// plus a huge fake ask far away that must not pull the common range out.
	a := VenueRawBook{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT",
		Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 100, Quantity: 1}, {Price: 98, Quantity: 50}},
			Asks: []PriceLevel{{Price: 100.2, Quantity: 1}, {Price: 102, Quantity: 50}},
		},
	}
	b := VenueRawBook{
		Exchange: ExchangeBybit, Symbol: "BTCUSDT",
		Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 100, Quantity: 1}, {Price: 99.5, Quantity: 1}},
			Asks: []PriceLevel{{Price: 100.2, Quantity: 1}, {Price: 100.5, Quantity: 1}},
		},
	}
	mid := SharedBookMid([]VenueRawBook{a, b})
	got := CombineOrderBooks("BTCUSDT", mid, 2, []VenueRawBook{a, b})
	if got.RequestedReached {
		t.Fatalf("B cannot cover 2%%: %+v", got)
	}
	bidQ, _ := strconv.ParseFloat(got.BidQuantity, 64)
	askQ, _ := strconv.ParseFloat(got.AskQuantity, 64)
	// Overlap is ask-limited (~0.4%); 99.5 bid is outside that symmetric ±%.
	if bidQ < 1.9 || bidQ > 2.1 {
		t.Fatalf("overlap bid qty %s (must exclude 98 wall and 99.5 outside ±usedPct)", got.BidQuantity)
	}
	if askQ < 2.9 || askQ > 3.1 {
		t.Fatalf("overlap ask qty %s (must exclude 102 wall)", got.AskQuantity)
	}
}

func TestCombineOrderBooks_SymmetricPctClipsLongerSide(t *testing.T) {
	// Common bids only reach ~1% below mid; asks reach ~2% above. Use 1% both ways.
	a := VenueRawBook{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT",
		Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 100, Quantity: 1}, {Price: 99, Quantity: 1}},
			Asks: []PriceLevel{{Price: 100.2, Quantity: 1}, {Price: 101.5, Quantity: 40}, {Price: 102, Quantity: 1}},
		},
	}
	b := VenueRawBook{
		Exchange: ExchangeBybit, Symbol: "ETHUSDT",
		Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 100, Quantity: 1}, {Price: 99, Quantity: 1}},
			Asks: []PriceLevel{{Price: 100.2, Quantity: 1}, {Price: 101.5, Quantity: 40}, {Price: 102, Quantity: 1}},
		},
	}
	mid := SharedBookMid([]VenueRawBook{a, b})
	got := CombineOrderBooks("ETHUSDT", mid, 2, []VenueRawBook{a, b})
	if got.RequestedReached {
		t.Fatalf("cannot cover 2%% both ways: %+v", got)
	}
	askQ, _ := strconv.ParseFloat(got.AskQuantity, 64)
	// 101.5 is ~1.4% above mid — outside the 1% common pct — must not count 80 qty.
	if askQ > 5 {
		t.Fatalf("longer ask side leaked into totals qty=%s used=%s-%s", got.AskQuantity, got.UsedLow, got.UsedHigh)
	}
	if got.UsedRangePct <= 0 || got.UsedRangePct > 1.2 {
		t.Fatalf("usedRangePct=%v", got.UsedRangePct)
	}
}

func TestCombineOrderBooks_UsesRequestedWhenAllReach(t *testing.T) {
	mk := func(ex Exchange) VenueRawBook {
		return VenueRawBook{Exchange: ex, Symbol: "ETHUSDT", Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 100, Quantity: 1}, {Price: 98.5, Quantity: 2}, {Price: 97.5, Quantity: 1}},
			Asks: []PriceLevel{{Price: 100.2, Quantity: 1}, {Price: 101.5, Quantity: 2}, {Price: 102.5, Quantity: 1}},
		}}
	}
	books := []VenueRawBook{mk(ExchangeBinance), mk(ExchangeCoinbase), mk(ExchangeBybit)}
	mid := SharedBookMid(books)
	got := CombineOrderBooks("ETHUSDT", mid, 2, books)
	if !got.RequestedReached {
		t.Fatalf("all cover 2%%: low=%s high=%s", got.UsedLow, got.UsedHigh)
	}
	bidQ, _ := strconv.ParseFloat(got.BidQuantity, 64)
	// 3 venues * (1@100 + 2@98.5); 97.5 is outside ±2%.
	if bidQ < 8.9 || bidQ > 9.1 {
		t.Fatalf("requested-band bid qty %s", got.BidQuantity)
	}
}
