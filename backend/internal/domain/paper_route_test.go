package domain

import (
	"math"
	"testing"
)

func TestRoutePaperBooks_FeeBeatsACheaperCoinbaseAsk(t *testing.T) {
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 1}}},
		{Exchange: ExchangeCoinbase, Symbol: "BTC-USD", Quote: "USD", FeeRate: CoinbaseFeeRate,
			Asks: []PriceLevel{{Price: 99.60, Quantity: 1}}},
	}
	got := RoutePaperBooks(TradeSideBuy, 1, "USDT", books)
	if got.Exhausted || got.InventoryShort || len(got.Legs) != 1 {
		t.Fatalf("%+v", got)
	}
	if got.Legs[0].Exchange != ExchangeBinance || got.Legs[0].Aliased {
		t.Fatalf("leg %+v", got.Legs[0])
	}
	if math.Abs(got.CashDebit-100.1) > 1e-9 {
		t.Fatalf("debit %v", got.CashDebit)
	}
}

func TestRoutePaperBooks_SplitsWhenTheFirstLevelIsThin(t *testing.T) {
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 0.4}, {Price: 101, Quantity: 2}}},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BybitFeeRate,
			Asks: []PriceLevel{{Price: 100.05, Quantity: 2}}},
		{Exchange: ExchangeCoinbase, Symbol: "BTC-USD", Quote: "USD", FeeRate: CoinbaseFeeRate,
			Asks: []PriceLevel{{Price: 110, Quantity: 2}}},
	}
	got := RoutePaperBooks(TradeSideBuy, 1, "USDT", books)
	if got.Exhausted || len(got.Legs) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Legs[0].Exchange != ExchangeBinance || math.Abs(got.Legs[0].Quantity-0.4) > 1e-9 {
		t.Fatalf("first %+v", got.Legs[0])
	}
	if got.Legs[1].Exchange != ExchangeBybit || math.Abs(got.Legs[1].Quantity-0.6) > 1e-9 {
		t.Fatalf("second %+v", got.Legs[1])
	}
	cmp := ComparePaperRoute(got, ExchangeBinance, TradeSideBuy, 1, "USDT", books)
	if !cmp.Known || cmp.Exhausted || cmp.SavedCash <= 0.5 {
		t.Fatalf("compare %+v", cmp)
	}
}

func TestRoutePaperBooks_SellStopsAtTheCoinsHeldOnEachVenue(t *testing.T) {
	binanceHeld := 0.2
	bybitHeld := 2.0
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate, CapQty: &binanceHeld,
			Bids: []PriceLevel{{Price: 100, Quantity: 5}}},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BybitFeeRate, CapQty: &bybitHeld,
			Bids: []PriceLevel{{Price: 99, Quantity: 5}}},
	}
	got := RoutePaperBooks(TradeSideSell, 1, "USDT", books)
	if got.Exhausted || got.InventoryShort || len(got.Legs) != 2 {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.Legs[0].Quantity-0.2) > 1e-9 || got.Legs[0].Exchange != ExchangeBinance {
		t.Fatalf("binance %+v", got.Legs[0])
	}
	if math.Abs(got.Legs[1].Quantity-0.8) > 1e-9 || got.Legs[1].Exchange != ExchangeBybit {
		t.Fatalf("bybit %+v", got.Legs[1])
	}
}

func TestRoutePaperBooks_SellNamesAShortInventory(t *testing.T) {
	held := 0.3
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate, CapQty: &held,
			Bids: []PriceLevel{{Price: 100, Quantity: 5}}},
	}
	got := RoutePaperBooks(TradeSideSell, 1, "USDT", books)
	if !got.InventoryShort || got.Exhausted || math.Abs(got.FilledQuantity-0.3) > 1e-9 {
		t.Fatalf("%+v", got)
	}
}

func TestRoutePaperBooks_NamesAMissingBookAndStillFills(t *testing.T) {
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 2}}},
		{Exchange: ExchangeCoinbase, Symbol: "BTC-USD", Quote: "USD", Missing: "timeout"},
	}
	got := RoutePaperBooks(TradeSideBuy, 1, "USDT", books)
	if got.Exhausted || len(got.Legs) != 1 || len(got.Missing) != 1 || got.Missing[0] != "coinbase" {
		t.Fatalf("%+v", got)
	}
	note := ExplainPaperRoute("USDT", got)
	if note == "" {
		t.Fatal("empty note")
	}
}

func TestRoutePaperBooks_ExhaustedWhenTheBooksAreShort(t *testing.T) {
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 0.2}}},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BybitFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 0.2}}},
	}
	got := RoutePaperBooks(TradeSideBuy, 1, "USDT", books)
	if !got.Exhausted || got.InventoryShort || math.Abs(got.FilledQuantity-0.4) > 1e-9 {
		t.Fatalf("%+v", got)
	}
}

func TestRoutePaperBooks_CoinbaseLegIsAliasedOntoUSDT(t *testing.T) {
	books := []PaperRouteBook{
		{Exchange: ExchangeCoinbase, Symbol: "BTC-USD", Quote: "USD", FeeRate: CoinbaseFeeRate,
			Asks: []PriceLevel{{Price: 90, Quantity: 1}}},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 1}}},
	}
	got := RoutePaperBooks(TradeSideBuy, 1, "USDT", books)
	if len(got.Legs) != 1 || got.Legs[0].Exchange != ExchangeCoinbase || !got.Legs[0].Aliased {
		t.Fatalf("%+v", got.Legs)
	}
}

func TestCanonicalRouteSymbol_RejectsAnEquityTicker(t *testing.T) {
	if _, err := CanonicalRouteSymbol("AAPL"); err == nil {
		t.Fatal("expected an equity ticker to be rejected")
	}
	got, err := CanonicalRouteSymbol("BTC-USD")
	if err != nil || got != "BTCUSDT" {
		t.Fatalf("canon %s err %v", got, err)
	}
}
