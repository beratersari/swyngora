package domain

import (
	"testing"
	"time"
)

func TestNetDiffPctAfterFees(t *testing.T) {
	// buy 100, sell 101, 0 fees → ~1%
	g, n, err := NetDiffPctAfterFees(100, 101, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if g < 0.99 || g > 1.01 || n < 0.99 || n > 1.01 {
		t.Fatalf("gross=%v net=%v", g, n)
	}
	// 0.1% fee each side reduces net
	_, n2, err := NetDiffPctAfterFees(100, 101, 0.1, 0.1)
	if err != nil || n2 >= n {
		t.Fatalf("net with fees=%v without=%v err=%v", n2, n, err)
	}
}

func TestBestPriceDiffRoutes(t *testing.T) {
	prices := map[Exchange]float64{
		ExchangeBinance:  100,
		ExchangeCoinbase: 102,
		ExchangeBybit:    100.5,
	}
	fees := map[Exchange]float64{
		ExchangeBinance: 0.1, ExchangeCoinbase: 0.1, ExchangeBybit: 0.1,
	}
	routes := BestPriceDiffRoutes(prices, fees, 0.5)
	if len(routes) == 0 {
		t.Fatal("expected at least one route")
	}
	// Best should be buy binance sell coinbase
	if routes[0].BuyExchange != ExchangeBinance || routes[0].SellExchange != ExchangeCoinbase {
		t.Fatalf("best=%+v", routes[0])
	}
	// High threshold → none
	if got := BestPriceDiffRoutes(prices, fees, 50); len(got) != 0 {
		t.Fatalf("want 0 got %d", len(got))
	}
}

func TestPriceDiffSymbolForExchange(t *testing.T) {
	if s := PriceDiffSymbolForExchange(ExchangeBinance, "btcusdt"); s != "BTCUSDT" {
		t.Fatalf("%s", s)
	}
	if s := PriceDiffSymbolForExchange(ExchangeCoinbase, "BTCUSDT"); s != "BTC-USD" {
		t.Fatalf("%s", s)
	}
	if s := PriceDiffSymbolForExchange(ExchangeBybit, "eth-usdt"); s != "ETHUSDT" {
		t.Fatalf("%s", s)
	}
}

func TestIsFreshTicker(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	ok := &Ticker24h{CloseTime: now.Add(-30 * time.Second)}
	if !IsFreshTicker(ok, now, 2*time.Minute) {
		t.Fatal("expected fresh")
	}
	stale := &Ticker24h{CloseTime: now.Add(-5 * time.Minute)}
	if IsFreshTicker(stale, now, 2*time.Minute) {
		t.Fatal("expected stale")
	}
	if IsFreshTicker(&Ticker24h{}, now, 2*time.Minute) {
		t.Fatal("zero close not fresh")
	}
	if IsFreshTicker(nil, now, 2*time.Minute) {
		t.Fatal("nil not fresh")
	}
}
