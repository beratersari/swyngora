package domain

import "testing"

func TestWalkPaperBook_BuyAveragesTheAsks(t *testing.T) {
	book := &RawOrderBook{Asks: []PriceLevel{{Price: 101, Quantity: 1}, {Price: 100, Quantity: 1}}}
	got := WalkPaperBook(TradeSideBuy, 1.5, book, 0)
	if got.Exhausted || got.FilledQuantity != 1.5 || got.LevelsUsed != 2 || got.BestPrice != 100 {
		t.Fatalf("%+v", got)
	}
	want := (100*1 + 101*0.5) / 1.5
	if got.Price != want || got.VisibleQuantity != 2 || got.Source != PaperFillBook {
		t.Fatalf("price %v want %v quote %+v", got.Price, want, got)
	}
}

func TestWalkPaperBook_ExhaustedDoesNotInventAPrice(t *testing.T) {
	book := &RawOrderBook{Asks: []PriceLevel{{Price: 100, Quantity: 1}}}
	got := WalkPaperBook(TradeSideBuy, 3, book, 0)
	if !got.Exhausted || got.FilledQuantity != 1 || got.Price != 100 || got.VisibleQuantity != 1 {
		t.Fatalf("%+v", got)
	}
	empty := WalkPaperBook(TradeSideBuy, 1, nil, 0)
	if !empty.Exhausted || empty.Price != 0 || empty.FilledQuantity != 0 {
		t.Fatalf("empty %+v", empty)
	}
}

func TestWalkPaperBook_LimitKeepsOnlyBetterPrices(t *testing.T) {
	book := &RawOrderBook{
		Asks: []PriceLevel{{Price: 100, Quantity: 1}, {Price: 105, Quantity: 4}},
		Bids: []PriceLevel{{Price: 99, Quantity: 2}, {Price: 90, Quantity: 5}},
	}
	buy := WalkPaperBook(TradeSideBuy, 2, book, 100.5)
	if !buy.Exhausted || buy.FilledQuantity != 1 || buy.Price != 100 || buy.VisibleQuantity != 1 {
		t.Fatalf("buy %+v", buy)
	}
	sell := WalkPaperBook(TradeSideSell, 1, book, 99)
	if sell.Exhausted || sell.Price != 99 || sell.BestPrice != 99 || sell.FilledQuantity != 1 {
		t.Fatalf("sell %+v", sell)
	}
}

func TestQuoteSlippageFill_UsesTheFixedRate(t *testing.T) {
	cost := TradingCost{FeeRate: BinanceFeeRate, SlippageRate: BinanceSlippageRate}
	got := QuoteSlippageFill(TradeSideBuy, 100, 2, cost)
	if got.Source != PaperFillSlippage || got.Exhausted || got.Price != ApplySlippage(100, TradeSideBuy, cost.SlippageRate) {
		t.Fatalf("%+v", got)
	}
}
