package domain

import (
	"strconv"
	"testing"
)

func TestSimulateMarketImpact_BuyQuantity(t *testing.T) {
	levels := []ImpactSourceLevel{
		{Exchange: "binance", Price: 100, Quantity: 1},
		{Exchange: "binance", Price: 101, Quantity: 2},
		{Exchange: "binance", Price: 110, Quantity: 10},
	}
	got := SimulateMarketImpact("BTCUSDT", "binance", ImpactSideBuy, 99.5, levels, 2, 0)
	if got.Exhausted {
		t.Fatal("should fill")
	}
	avg, _ := strconv.ParseFloat(got.AveragePrice, 64)
	// 1@100 + 1@101 = 201 / 2 = 100.5
	if avg < 100.49 || avg > 100.51 {
		t.Fatalf("avg %s", got.AveragePrice)
	}
	if got.EndPrice != "101" || got.LevelsUsed != 2 {
		t.Fatalf("%+v", got)
	}
	if got.SlippagePct <= 0 {
		t.Fatalf("slip=%v", got.SlippagePct)
	}
	// first ask fully eaten; new best is 101 → (101-100)/100 = 1%
	if got.NewBestPrice != "101" || got.ImpactPct < 0.99 || got.ImpactPct > 1.01 {
		t.Fatalf("newBest=%s impact=%v", got.NewBestPrice, got.ImpactPct)
	}
}

func TestSimulateMarketImpact_PartialBestIsZero(t *testing.T) {
	levels := []ImpactSourceLevel{
		{Exchange: "binance", Price: 100, Quantity: 2},
		{Exchange: "binance", Price: 101, Quantity: 5},
	}
	got := SimulateMarketImpact("BTCUSDT", "binance", ImpactSideBuy, 99.5, levels, 1, 0)
	if got.ImpactPct != 0 || got.NewBestPrice != "100" {
		t.Fatalf("partial best should not move price: %+v", got)
	}
	if got.AveragePrice == "" || got.EndPrice != "100" {
		t.Fatalf("%+v", got)
	}
}

func TestSimulateMarketImpact_SamePriceOtherVenueKeepsTouch(t *testing.T) {
	levels := []ImpactSourceLevel{
		{Exchange: "bybit", Price: 100, Quantity: 1},
		{Exchange: "binance", Price: 100, Quantity: 2},
		{Exchange: "binance", Price: 102, Quantity: 4},
	}
	got := SimulateMarketImpact("BTCUSDT", "combined", ImpactSideBuy, 100, levels, 1, 0)
	if got.ImpactPct != 0 || got.NewBestPrice != "100" {
		t.Fatalf("same-price leftover must keep impact 0: %+v", got)
	}
}

func TestSimulateMarketImpact_NotionalAndExhaust(t *testing.T) {
	levels := []ImpactSourceLevel{
		{Exchange: "binance", Price: 100, Quantity: 1},
	}
	got := SimulateMarketImpact("ETHUSDT", "binance", ImpactSideBuy, 100, levels, 0, 250)
	if !got.Exhausted {
		t.Fatal("want exhausted")
	}
	spent, _ := strconv.ParseFloat(got.SpentNotional, 64)
	unf, _ := strconv.ParseFloat(got.UnfilledNotional, 64)
	if spent < 99 || spent > 101 || unf < 149 || unf > 151 {
		t.Fatalf("spent=%s unfilled=%s", got.SpentNotional, got.UnfilledNotional)
	}
}

func TestSimulateMarketImpact_Sell(t *testing.T) {
	levels := []ImpactSourceLevel{
		{Exchange: "binance", Price: 100, Quantity: 1},
		{Exchange: "binance", Price: 99, Quantity: 1},
	}
	got := SimulateMarketImpact("BTCUSDT", "binance", ImpactSideSell, 100.5, levels, 1.5, 0)
	avg, _ := strconv.ParseFloat(got.AveragePrice, 64)
	// 1@100 + 0.5@99 = 149.5 / 1.5 ≈ 99.666
	if avg < 99.6 || avg > 99.7 {
		t.Fatalf("avg %s", got.AveragePrice)
	}
	if got.SlippagePct <= 0 {
		t.Fatalf("sell slip %v", got.SlippagePct)
	}
	// best bid 100 fully eaten; new best bid 99 → 1%
	if got.NewBestPrice != "99" || got.ImpactPct < 0.99 || got.ImpactPct > 1.01 {
		t.Fatalf("newBest=%s impact=%v", got.NewBestPrice, got.ImpactPct)
	}
}

func TestCollectImpactLevels_CombinedCheapestFirst(t *testing.T) {
	books := []VenueRawBook{
		{Exchange: ExchangeBinance, Book: RawOrderBook{Asks: []PriceLevel{{Price: 101, Quantity: 1}}}},
		{Exchange: ExchangeBybit, Book: RawOrderBook{Asks: []PriceLevel{{Price: 100, Quantity: 2}}}},
	}
	got := CollectImpactLevels(ImpactSideBuy, books)
	if len(got) != 2 || got[0].Price != 100 || got[0].Exchange != "bybit" {
		t.Fatalf("%+v", got)
	}
}

func TestCollectImpactLevels_SellHighestFirst(t *testing.T) {
	books := []VenueRawBook{
		{Exchange: ExchangeBinance, Book: RawOrderBook{Bids: []PriceLevel{{Price: 99, Quantity: 1}}}},
		{Exchange: ExchangeCoinbase, Book: RawOrderBook{Bids: []PriceLevel{{Price: 100, Quantity: 2}}}},
	}
	got := CollectImpactLevels(ImpactSideSell, books)
	if len(got) != 2 || got[0].Price != 100 || got[0].Exchange != "coinbase" {
		t.Fatalf("%+v", got)
	}
}

func TestSimulateMarketImpact_EmptyBook(t *testing.T) {
	got := SimulateMarketImpact("BTCUSDT", "combined", ImpactSideBuy, 100, nil, 1, 0)
	if !got.Exhausted || got.UnfilledQuantity == "" {
		t.Fatalf("%+v", got)
	}
}

func TestImpactBookMid_MergedBBO(t *testing.T) {
	books := []VenueRawBook{
		{Exchange: ExchangeBinance, Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 100, Quantity: 1}},
			Asks: []PriceLevel{{Price: 101, Quantity: 1}},
		}},
		{Exchange: ExchangeBybit, Book: RawOrderBook{
			Bids: []PriceLevel{{Price: 99.5, Quantity: 1}},
			Asks: []PriceLevel{{Price: 100.2, Quantity: 1}},
		}},
	}
	mid := ImpactBookMid(books)
	// best bid 100, best ask 100.2
	if mid < 100.09 || mid > 100.11 {
		t.Fatalf("mid=%v", mid)
	}
}

func TestSimulateMarketImpact_NoNegativeSlip(t *testing.T) {
	levels := []ImpactSourceLevel{{Exchange: "binance", Price: 99, Quantity: 1}}
	got := SimulateMarketImpact("BTCUSDT", "combined", ImpactSideBuy, 100, levels, 1, 0)
	if got.SlippagePct != 0 || got.ImpactPct != 0 {
		t.Fatalf("want 0 adverse, got slip=%v impact=%v", got.SlippagePct, got.ImpactPct)
	}
}

func TestValidateImpactSize(t *testing.T) {
	if err := ValidateImpactSize(1, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateImpactSize(0, 100); err != nil {
		t.Fatal(err)
	}
	if err := ValidateImpactSize(1, 100); err == nil {
		t.Fatal("both")
	}
	if err := ValidateImpactSize(0, 0); err == nil {
		t.Fatal("none")
	}
	side, err := ParseImpactSide("")
	if err != nil || side != ImpactSideBuy {
		t.Fatal(side, err)
	}
}
