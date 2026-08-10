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
	if bidQ < 4.9 || bidQ > 5.1 {
		t.Fatalf("bid qty %s (far 80 must be excluded)", got.BidQuantity)
	}
	if askQ < 4.9 || askQ > 5.1 {
		t.Fatalf("ask qty %s (far 120 must be excluded)", got.AskQuantity)
	}
	if got.Pressure == "" || len(got.Bands) != 4 {
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
