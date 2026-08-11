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
