package domain

import (
	"errors"
	"math"
	"strconv"
	"testing"
)

func TestParseGroupSize(t *testing.T) {
	v, err := ParseGroupSize(" 0.1 ")
	if err != nil || v != 0.1 {
		t.Fatalf("%v %v", v, err)
	}
	empty, err := ParseGroupSize("  ")
	if err != nil || empty != 0 {
		t.Fatalf("empty %v %v", empty, err)
	}
	if _, err := ParseGroupSize("-1"); err == nil || !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("neg: %v", err)
	}
	if _, err := ParseGroupSize("nope"); err == nil {
		t.Fatal("want invalid")
	}
}

func TestGroupOrderBook_GroupsAndWalls(t *testing.T) {
	raw := RawOrderBook{
		Symbol: "BTCUSDT",
		Bids: []PriceLevel{
			{Price: 100.04, Quantity: 1},
			{Price: 100.02, Quantity: 1},
			{Price: 99.91, Quantity: 1},
			{Price: 99.50, Quantity: 80}, // wall after group 99.5
			{Price: 99.11, Quantity: 1},
		},
		Asks: []PriceLevel{
			{Price: 100.06, Quantity: 1},
			{Price: 100.09, Quantity: 1},
			{Price: 100.21, Quantity: 1},
			{Price: 100.80, Quantity: 90},
			{Price: 101.00, Quantity: 1},
		},
	}
	book := GroupOrderBook(raw, 0.1, 10)
	if book.GroupSize != "0.1" {
		t.Fatalf("group %q", book.GroupSize)
	}
	if len(book.Bids) < 2 || len(book.Asks) < 2 {
		t.Fatalf("bids=%d asks=%d", len(book.Bids), len(book.Asks))
	}
	// 100.04 and 100.02 floor to 100.0
	if book.Bids[0].Price != "100" {
		t.Fatalf("best grouped bid %s", book.Bids[0].Price)
	}
	qty, _ := strconv.ParseFloat(book.Bids[0].Quantity, 64)
	if math.Abs(qty-2) > 1e-9 {
		t.Fatalf("merged bid qty %s", book.Bids[0].Quantity)
	}
	if book.Asks[0].Price != "100.1" {
		t.Fatalf("best grouped ask %s (ceil 100.06 → 100.1)", book.Asks[0].Price)
	}
	var sawBidWall, sawAskWall bool
	for _, lv := range book.Bids {
		if lv.IsWall {
			sawBidWall = true
		}
	}
	for _, lv := range book.Asks {
		if lv.IsWall {
			sawAskWall = true
		}
	}
	if !sawBidWall || !sawAskWall {
		t.Fatalf("walls bid=%v ask=%v book=%+v", sawBidWall, sawAskWall, book)
	}
	if book.Imbalance == 0 {
		t.Fatal("expected imbalance from wall sizes")
	}
	if len(book.SuggestedGroupSizes) == 0 {
		t.Fatal("suggested empty")
	}
}

func TestSuggestedGroupSizes_BTCLike(t *testing.T) {
	sizes := SuggestedGroupSizes(65000)
	if len(sizes) < 3 {
		t.Fatalf("sizes=%v", sizes)
	}
	// Expect 0.1 and 1-style steps for BTC.
	var hasTenth, hasOne bool
	for _, s := range sizes {
		if math.Abs(s-0.1) < 1e-12 {
			hasTenth = true
		}
		if math.Abs(s-1) < 1e-12 {
			hasOne = true
		}
	}
	if !hasTenth || !hasOne {
		t.Fatalf("want 0.1 and 1 in %v", sizes)
	}
	def := DefaultGroupSize(sizes)
	if def <= 0 {
		t.Fatalf("default %v", def)
	}
}

func TestGroupOrderBook_Empty(t *testing.T) {
	book := GroupOrderBook(RawOrderBook{Symbol: "X"}, 0.01, 20)
	if len(book.Bids) != 0 || len(book.Asks) != 0 {
		t.Fatalf("%+v", book)
	}
}

func TestParsePriceQty(t *testing.T) {
	lv, ok := ParsePriceQty("100.5", "2")
	if !ok || lv.Price != 100.5 || lv.Quantity != 2 {
		t.Fatalf("%+v %v", lv, ok)
	}
	if _, ok := ParsePriceQty("x", "1"); ok {
		t.Fatal("want fail")
	}
}
