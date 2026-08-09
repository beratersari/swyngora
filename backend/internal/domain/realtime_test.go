package domain

import "testing"

func TestNormalizeSymbolRef(t *testing.T) {
	ref, err := NormalizeSymbolRef("BINANCE", "btcusdt")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Exchange != ExchangeBinance || ref.Symbol != "BTCUSDT" {
		t.Fatalf("got %+v", ref)
	}
	if _, err := NormalizeSymbolRef("nope", "BTCUSDT"); err == nil {
		t.Fatal("expected invalid exchange")
	}
	if _, err := NormalizeSymbolRef("binance", "  "); err == nil {
		t.Fatal("expected empty symbol")
	}
	if SymbolKey(ExchangeBinance, "ethusdt") != "binance:ETHUSDT" {
		t.Fatalf("key=%s", SymbolKey(ExchangeBinance, "ethusdt"))
	}
}
