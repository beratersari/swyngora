package domain

import (
	"math"
	"testing"
	"time"
)

func compareBooks() []PaperRouteBook {
	return []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 1}, {Price: 110, Quantity: 5}},
			Bids: []PriceLevel{{Price: 100, Quantity: 5}}},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BybitFeeRate,
			Asks: []PriceLevel{{Price: 102, Quantity: 5}},
			Bids: []PriceLevel{{Price: 99, Quantity: 5}}},
		{Exchange: ExchangeCoinbase, Symbol: "BTC-USD", Quote: "USD", FeeRate: CoinbaseFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 5}},
			Bids: []PriceLevel{{Price: 99, Quantity: 5}}},
	}
}

func TestCompareExecutions_SpreadRefillBeatsWalkingOneBook(t *testing.T) {
	plan := SpreadSchedule{SliceCount: 2, Minutes: 10, Duration: 10 * time.Minute, Interval: 10 * time.Minute}
	got := CompareExecutions(TradeSideBuy, 2, "USDT", "BTCUSDT", ExchangeBinance, 0, &plan, "", "", 10, 2, compareBooks())
	if got.TouchPrice != 100 || got.TouchExchange != ExchangeBinance || len(got.Rows) != 3 {
		t.Fatalf("touch %v %s rows %d", got.TouchPrice, got.TouchExchange, len(got.Rows))
	}
	market, best, spread := got.Rows[0], got.Rows[1], got.Rows[2]
	if market.Status != ExecutionFilled || math.Abs(market.AveragePrice-105) > 1e-9 || math.Abs(market.SlippagePct-5) > 1e-9 {
		t.Fatalf("market %+v", market)
	}
	if math.Abs(market.Fee-0.21) > 1e-9 || math.Abs(market.CashDebit-210.21) > 1e-9 {
		t.Fatalf("market cash fee %v debit %v", market.Fee, market.CashDebit)
	}
	if best.Status != ExecutionFilled || math.Abs(best.CashDebit-200.7) > 1e-9 || math.Abs(best.SlippagePct) > 1e-9 {
		t.Fatalf("best %+v", best)
	}
	if best.SavedVsMarket == nil || math.Abs(*best.SavedVsMarket-9.51) > 1e-9 {
		t.Fatalf("best saved %+v", best.SavedVsMarket)
	}
	if spread.Status != ExecutionFilled || spread.Assumption != "refill" || math.Abs(spread.CashDebit-200.2) > 1e-9 {
		t.Fatalf("spread %+v", spread)
	}
	if spread.SavedVsMarket == nil || *spread.SavedVsMarket <= *best.SavedVsMarket {
		t.Fatalf("spread should beat best cost: spread %+v best %+v", spread.SavedVsMarket, best.SavedVsMarket)
	}
	if len(spread.Legs) != 1 || spread.Legs[0].Exchange != ExchangeBinance || math.Abs(spread.Legs[0].Quantity-2) > 1e-9 {
		t.Fatalf("spread legs %+v", spread.Legs)
	}
}

func TestCompareExecutions_CapBlocksBestCostAndLeavesTheVenue(t *testing.T) {
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 1}, {Price: 110, Quantity: 5}}},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BybitFeeRate,
			Asks: []PriceLevel{{Price: 102, Quantity: 5}}},
	}
	plan := SpreadSchedule{SliceCount: 2, Minutes: 10, Duration: 10 * time.Minute, Interval: 10 * time.Minute}
	got := CompareExecutions(TradeSideBuy, 2, "USDT", "BTCUSDT", ExchangeBinance, 0.005, &plan, "", "", 10, 2, books)
	market, best, spread := got.Rows[0], got.Rows[1], got.Rows[2]
	if market.Status != ExecutionFilled || market.CashDebit <= 0 {
		t.Fatalf("market %+v", market)
	}
	if best.Status != ExecutionSlippageLimited || best.CashDebit != 0 || best.SavedVsMarket != nil {
		t.Fatalf("best %+v", best)
	}
	if spread.Status != ExecutionFilled || math.Abs(spread.AveragePrice-100) > 1e-9 || spread.SavedVsMarket == nil {
		t.Fatalf("spread %+v", spread)
	}
}

func TestCompareExecutions_MissingBookIsNamedAndIsNotZero(t *testing.T) {
	books := compareBooks()
	books[2].Missing = "empty order book"
	books[2].Asks = nil
	plan := SpreadSchedule{SliceCount: 2, Minutes: 10, Duration: 10 * time.Minute, Interval: 10 * time.Minute}
	got := CompareExecutions(TradeSideBuy, 2, "USDT", "BTCUSDT", ExchangeBinance, 0, &plan, "", "", 10, 2, books)
	best := got.Rows[1]
	if best.Status != ExecutionFilled {
		t.Fatalf("best %+v", best)
	}
	found := false
	for _, name := range best.Missing {
		if name == string(ExchangeCoinbase) {
			found = true
		}
	}
	if !found || best.CashDebit == 0 {
		t.Fatalf("missing %+v debit %v", best.Missing, best.CashDebit)
	}
}

func TestCompareExecutions_SellUsesCoinsOnTheVenueThatHasThem(t *testing.T) {
	binance := 0.0
	bybit := 2.0
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate, CapQty: &binance,
			Bids: []PriceLevel{{Price: 100, Quantity: 5}}},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BybitFeeRate, CapQty: &bybit,
			Bids: []PriceLevel{{Price: 99, Quantity: 5}}},
	}
	plan := SpreadSchedule{SliceCount: 2, Minutes: 10, Duration: 10 * time.Minute, Interval: 10 * time.Minute}
	got := CompareExecutions(TradeSideSell, 1, "USDT", "BTCUSDT", ExchangeBinance, 0, &plan, "", "", 10, 2, books)
	if got.TouchPrice != 100 {
		t.Fatalf("touch %v", got.TouchPrice)
	}
	market, best := got.Rows[0], got.Rows[1]
	if market.Status != ExecutionInventoryShort || market.CashCredit != 0 {
		t.Fatalf("market %+v", market)
	}
	if best.Status != ExecutionFilled || len(best.Legs) != 1 || best.Legs[0].Exchange != ExchangeBybit {
		t.Fatalf("best %+v", best)
	}
	if best.SavedVsMarket != nil {
		t.Fatalf("no venue fill to save against: %+v", best.SavedVsMarket)
	}
	if math.Abs(best.SlippagePct-1) > 1e-9 {
		t.Fatalf("slip %v", best.SlippagePct)
	}
}

func TestCompareExecutions_SpreadSellDoesNotReuseAVenueLock(t *testing.T) {
	binance := 0.2
	bybit := 2.0
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate, CapQty: &binance,
			Bids: []PriceLevel{{Price: 100, Quantity: 5}}},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BybitFeeRate, CapQty: &bybit,
			Bids: []PriceLevel{{Price: 99, Quantity: 5}}},
	}
	plan := SpreadSchedule{SliceCount: 2, Minutes: 10, Duration: 10 * time.Minute, Interval: 10 * time.Minute}
	got := CompareExecutions(TradeSideSell, 1, "USDT", "BTCUSDT", ExchangeBybit, 0, &plan, "", "", 10, 2, books)
	spread := got.Rows[2]
	if spread.Status != ExecutionFilled || math.Abs(spread.FilledQuantity-1) > 1e-9 {
		t.Fatalf("%+v", spread)
	}
	var binanceQty float64
	for _, leg := range spread.Legs {
		if leg.Exchange == ExchangeBinance {
			binanceQty = leg.Quantity
		}
	}
	if binanceQty > 0.11 {
		t.Fatalf("binance lock reused: %v legs %+v", binanceQty, spread.Legs)
	}
}

func TestCompareExecutions_BadScheduleStillPricesTheOtherPaths(t *testing.T) {
	got := CompareExecutions(TradeSideBuy, 2, "USDT", "BTCUSDT", ExchangeBinance, 0, nil, ExecutionScheduleInterval, "too close", 1, 60, compareBooks())
	if got.Rows[0].Status != ExecutionFilled || got.Rows[1].Status != ExecutionFilled {
		t.Fatalf("rows %+v", got.Rows)
	}
	spread := got.Rows[2]
	if spread.Status != ExecutionSchedule || spread.Schedule != ExecutionScheduleInterval || spread.CashDebit != 0 {
		t.Fatalf("spread %+v", spread)
	}
}

func TestCompareExecutions_ThinBookFillsNothing(t *testing.T) {
	books := []PaperRouteBook{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quote: "USDT", FeeRate: BinanceFeeRate,
			Asks: []PriceLevel{{Price: 100, Quantity: 0.2}}},
	}
	plan := SpreadSchedule{SliceCount: 2, Minutes: 10, Duration: 10 * time.Minute, Interval: 10 * time.Minute}
	got := CompareExecutions(TradeSideBuy, 1, "USDT", "BTCUSDT", ExchangeBinance, 0, &plan, "", "", 10, 2, books)
	for _, row := range got.Rows {
		if row.Status == ExecutionFilled || row.CashDebit != 0 {
			t.Fatalf("row %+v", row)
		}
	}
}
