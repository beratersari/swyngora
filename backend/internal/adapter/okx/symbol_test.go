package okx

import "testing"

func TestSwapInstID(t *testing.T) {
	if got := SwapInstID("btcusdt"); got != "BTC-USDT-SWAP" {
		t.Fatalf("got %s", got)
	}
	if got := SwapInstID("ETH-USDT-SWAP"); got != "ETH-USDT-SWAP" {
		t.Fatalf("got %s", got)
	}
	if SwapInstID("") != "" {
		t.Fatal("empty")
	}
}

func TestSymbolFromInstID(t *testing.T) {
	if got := SymbolFromInstID("BTC-USDT-SWAP"); got != "BTCUSDT" {
		t.Fatalf("got %s", got)
	}
	if got := UnderlyingFromSymbol("ETHUSDT"); got != "ETH-USDT" {
		t.Fatalf("uly %s", got)
	}
}
