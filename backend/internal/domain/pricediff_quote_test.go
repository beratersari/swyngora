package domain

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

func testQuoteBooks() (buy, sell *RawOrderBook) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	buy = &RawOrderBook{
		Symbol: "BTCUSDT", Live: true, FetchedAt: now,
		Asks: []PriceLevel{
			{Price: 100, Quantity: 1},
			{Price: 101, Quantity: 1},
			{Price: 110, Quantity: 10},
		},
		Bids: []PriceLevel{{Price: 99, Quantity: 1}},
	}
	sell = &RawOrderBook{
		Symbol: "BTC-USD", Live: true, FetchedAt: now.Add(time.Second),
		Bids: []PriceLevel{
			{Price: 103, Quantity: 1},
			{Price: 102, Quantity: 1},
			{Price: 90, Quantity: 10},
		},
		Asks: []PriceLevel{{Price: 104, Quantity: 1}},
	}
	return buy, sell
}

func mustQuote(t *testing.T, q PriceDiffQuoteQuery, buy, sell *RawOrderBook) *PriceDiffQuote {
	t.Helper()
	got, err := QuotePriceDiffRoute(q, buy, sell)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestQuotePriceDiffRoute_SmallNotional(t *testing.T) {
	buy, sell := testQuoteBooks()
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeCoinbase,
		BuyFeePct: 0.1, SellFeePct: 0.1, Notional: 100,
	}, buy, sell)
	if got.AverageBuyPrice != "100" || got.AverageSellPrice != "103" {
		t.Fatalf("avg buy=%s sell=%s", got.AverageBuyPrice, got.AverageSellPrice)
	}
	if got.BuySlippagePct != 0 || got.SellSlippagePct != 0 || got.SlippagePct != 0 {
		t.Fatalf("slip %+v", got)
	}
	if !got.FilledRequested || !got.Profitable || !got.Executable {
		t.Fatalf("flags %+v", got)
	}
	if got.BuyExhausted || got.SellExhausted {
		t.Fatalf("exhausted %+v", got)
	}
	// cost=100.1  proceeds=103*0.999=102.897  profit=2.797
	profit, err := strconv.ParseFloat(got.ProfitAfterFees, 64)
	if err != nil || profit < 2.79 || profit > 2.81 {
		t.Fatalf("profit=%s err=%v", got.ProfitAfterFees, err)
	}
	if got.BestAsk != "100" || got.BestBid != "103" {
		t.Fatalf("best ask=%s bid=%s", got.BestAsk, got.BestBid)
	}
	if !got.Live {
		t.Fatal("expected live")
	}
}

func TestQuotePriceDiffRoute_WalksDeeperAndSlips(t *testing.T) {
	buy, sell := testQuoteBooks()
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeBybit,
		BuyFeePct: 0, SellFeePct: 0, Notional: 201,
	}, buy, sell)
	if got.AverageBuyPrice != "100.5" {
		t.Fatalf("avg buy=%s want 100.5", got.AverageBuyPrice)
	}
	if got.AverageSellPrice != "102.5" {
		t.Fatalf("avg sell=%s want 102.5", got.AverageSellPrice)
	}
	if got.BuySlippagePct <= 0 || got.SellSlippagePct <= 0 {
		t.Fatalf("expected slippage vs top: buy=%v sell=%v", got.BuySlippagePct, got.SellSlippagePct)
	}
}

func TestQuotePriceDiffRoute_MaxStopsBeforeLosingLevel(t *testing.T) {
	buy, sell := testQuoteBooks()
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeCoinbase,
		BuyFeePct: 0.1, SellFeePct: 0.1, Notional: 100,
	}, buy, sell)
	// First two levels still have positive edge; 110 vs 90 does not.
	qty, err := strconv.ParseFloat(got.MaxQuantity, 64)
	if err != nil || qty < 1.99 || qty > 2.01 {
		t.Fatalf("max qty=%s err=%v", got.MaxQuantity, err)
	}
	if got.MaxLimitedBy != PriceDiffMaxLimitedByProfit {
		t.Fatalf("limitedBy=%s", got.MaxLimitedBy)
	}
	maxProfit, err := strconv.ParseFloat(got.MaxProfitAfterFees, 64)
	if err != nil || maxProfit <= 0 {
		t.Fatalf("max profit=%s", got.MaxProfitAfterFees)
	}
}

func TestQuotePriceDiffRoute_FeesKillEdge(t *testing.T) {
	buy, sell := testQuoteBooks()
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeCoinbase,
		BuyFeePct: 10, SellFeePct: 10, Notional: 100,
	}, buy, sell)
	if got.Profitable || got.Executable {
		t.Fatalf("fees should wipe the 3%% gap: %+v", got)
	}
	if got.MaxQuantity != "" && got.MaxQuantity != "0" {
		t.Fatalf("max should be empty/zero, got %s", got.MaxQuantity)
	}
	if got.MaxLimitedBy != PriceDiffMaxLimitedByProfit {
		t.Fatalf("limitedBy=%s", got.MaxLimitedBy)
	}
}

func TestQuotePriceDiffRoute_SellBookThinner(t *testing.T) {
	buy, _ := testQuoteBooks()
	thinSell := &RawOrderBook{
		Symbol: "BTCUSDT", Live: true,
		Bids: []PriceLevel{{Price: 103, Quantity: 0.4}},
	}
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeBybit,
		Notional: 100, // would buy 1.0 on the ask
	}, buy, thinSell)
	qty, _ := strconv.ParseFloat(got.FilledQuantity, 64)
	if qty < 0.39 || qty > 0.41 {
		t.Fatalf("filled=%s want ~0.4", got.FilledQuantity)
	}
	if !got.SellExhausted {
		t.Fatal("expected sell exhausted")
	}
	if got.FilledRequested || got.Executable {
		t.Fatalf("should not fill the full 100 USDT: %+v", got)
	}
	buyN, _ := strconv.ParseFloat(got.BuyNotional, 64)
	if buyN < 39 || buyN > 41 {
		t.Fatalf("buy notional rematch=%s", got.BuyNotional)
	}
}

func TestQuotePriceDiffRoute_Quantity(t *testing.T) {
	buy, sell := testQuoteBooks()
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeCoinbase,
		Quantity: 1,
	}, buy, sell)
	if got.FilledQuantity != "1" || got.AverageBuyPrice != "100" || got.AverageSellPrice != "103" {
		t.Fatalf("%+v", got)
	}
}

func TestQuotePriceDiffRoute_Rejects(t *testing.T) {
	buy, sell := testQuoteBooks()
	_, err := QuotePriceDiffRoute(PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeBinance, Notional: 100,
	}, buy, sell)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("same venue: %v", err)
	}
	_, err = QuotePriceDiffRoute(PriceDiffQuoteQuery{
		Symbol: "AAPL", BuyExchange: ExchangeNasdaq, SellExchange: ExchangeBinance, Notional: 100,
	}, buy, sell)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("equity: %v", err)
	}
	_, err = QuotePriceDiffRoute(PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeBybit,
	}, buy, sell)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("missing size: %v", err)
	}
	_, err = QuotePriceDiffRoute(PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeBybit,
		Notional: 100, Quantity: 1,
	}, buy, sell)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("both sizes: %v", err)
	}
}

func TestQuotePriceDiffRoute_EmptyBooks(t *testing.T) {
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeBybit, Notional: 100,
	}, &RawOrderBook{Symbol: "BTCUSDT"}, &RawOrderBook{Symbol: "BTCUSDT"})
	if got.Profitable || got.FilledRequested || got.Executable {
		t.Fatalf("%+v", got)
	}
	if got.MaxLimitedBy != PriceDiffMaxLimitedByEmpty && got.MaxLimitedBy != PriceDiffMaxLimitedByBuyBook {
		t.Fatalf("limitedBy=%s", got.MaxLimitedBy)
	}
}

func TestQuotePriceDiffRoute_MeetsMinNet(t *testing.T) {
	buy, sell := testQuoteBooks()
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeCoinbase,
		BuyFeePct: 0.1, SellFeePct: 0.1, Notional: 100, MinNetDiffPct: 0.5,
	}, buy, sell)
	if !got.MeetsMinNet {
		t.Fatalf("3%% gap should meet 0.5%%: profitPct=%v", got.ProfitPct)
	}
	got = mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "BTCUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeCoinbase,
		BuyFeePct: 0.1, SellFeePct: 0.1, Notional: 100, MinNetDiffPct: 50,
	}, buy, sell)
	if got.MeetsMinNet {
		t.Fatalf("should miss 50%% min: profitPct=%v", got.ProfitPct)
	}
}

func TestQuotePriceDiffRoute_MaxLimitedByBook(t *testing.T) {
	buy := &RawOrderBook{
		Asks: []PriceLevel{{Price: 100, Quantity: 2}},
	}
	sell := &RawOrderBook{
		Bids: []PriceLevel{{Price: 110, Quantity: 5}},
	}
	got := mustQuote(t, PriceDiffQuoteQuery{
		Symbol: "ETHUSDT", BuyExchange: ExchangeBinance, SellExchange: ExchangeBybit, Notional: 50,
	}, buy, sell)
	if got.MaxLimitedBy != PriceDiffMaxLimitedByBuyBook {
		t.Fatalf("limitedBy=%s", got.MaxLimitedBy)
	}
	qty, _ := strconv.ParseFloat(got.MaxQuantity, 64)
	if qty < 1.99 || qty > 2.01 {
		t.Fatalf("max qty=%s", got.MaxQuantity)
	}
}
