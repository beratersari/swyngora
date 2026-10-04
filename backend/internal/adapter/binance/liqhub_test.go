package binance

import (
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestParseForceOrder_LongFromSell(t *testing.T) {
	raw := []byte(`{"e":"forceOrder","E":1700000000000,"o":{"s":"BTCUSDT","S":"SELL","p":"64000","ap":"63990","q":"2","z":"2","T":1700000000000}}`)
	ev, ok := ParseForceOrder(raw)
	if !ok || ev.Side != domain.LiquidationSideLong || ev.Symbol != "BTCUSDT" {
		t.Fatalf("%+v ok=%v", ev, ok)
	}
	if ev.Notional < 127900 || ev.Notional > 128000 {
		t.Fatalf("notional %v", ev.Notional)
	}
}

func TestParseForceOrder_WrappedStream(t *testing.T) {
	raw := []byte(`{"stream":"!forceOrder@arr","data":{"e":"forceOrder","E":1,"o":{"s":"ETHUSDT","S":"BUY","p":"3000","ap":"3000","q":"1","z":"1","T":1}}}`)
	ev, ok := ParseForceOrder(raw)
	if !ok || ev.Side != domain.LiquidationSideShort || ev.Symbol != "ETHUSDT" {
		t.Fatalf("%+v ok=%v", ev, ok)
	}
}

func TestParseForceOrder_IgnoreNoise(t *testing.T) {
	if _, ok := ParseForceOrder([]byte(`{"e":"ping"}`)); ok {
		t.Fatal("ping")
	}
}

func TestParseForceOrder_SkipsCoinMargined(t *testing.T) {
	raw := []byte(`{"e":"forceOrder","E":1,"st":2,"ps":"BTCUSD","o":{"s":"BTCUSD_PERP","S":"SELL","p":"64000","ap":"64000","q":"1","z":"1","T":1}}`)
	if _, ok := ParseForceOrder(raw); ok {
		t.Fatal("coin-margined print")
	}
	wrapped := []byte(`{"stream":"!forceOrder@arr","data":{"e":"forceOrder","E":1,"st":2,"o":{"s":"BTCUSD_PERP","S":"SELL","p":"64000","ap":"64000","q":"1","z":"1","T":1}}}`)
	if _, ok := ParseForceOrder(wrapped); ok {
		t.Fatal("wrapped coin-margined print")
	}
}

func TestParseForceOrder_KeepsUSDMarginedMark(t *testing.T) {
	raw := []byte(`{"e":"forceOrder","E":1,"st":1,"ps":"BTCUSDT","o":{"s":"BTCUSDT","S":"SELL","p":"64000","ap":"64000","q":"1","z":"1","T":1}}`)
	ev, ok := ParseForceOrder(raw)
	if !ok || ev.Symbol != "BTCUSDT" || ev.Exchange != domain.ExchangeBinance || ev.Side != domain.LiquidationSideLong {
		t.Fatalf("%+v ok=%v", ev, ok)
	}
}

func TestLiquidationStreamURL_UsesMarketPath(t *testing.T) {
	got := liquidationStreamURL("wss://fstream.binance.com/")
	want := "wss://fstream.binance.com/market/ws/!forceOrder@arr"
	if got != want {
		t.Fatalf("url %s", got)
	}
	if liquidationStreamURL("") != defaultFuturesWSURL+liquidationStreamSuffix {
		t.Fatal("empty base")
	}
}
