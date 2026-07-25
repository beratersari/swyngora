package domain

import "testing"

func TestParseExchange(t *testing.T) {
	if ParseExchange("") != ExchangeBinance {
		t.Fatal("default")
	}
	if ParseExchange("Bybit") != ExchangeBybit {
		t.Fatal("bybit")
	}
	if ParseExchange("nope") != "" {
		t.Fatal("unknown")
	}
}

func TestSupportedIntervalsFor_CoinbaseSubset(t *testing.T) {
	if IsValidIntervalFor(ExchangeCoinbase, "1h") != true {
		t.Fatal("1h ok")
	}
	if IsValidIntervalFor(ExchangeCoinbase, "3m") {
		t.Fatal("3m not on coinbase")
	}
	if !IsValidIntervalFor(ExchangeBybit, "1h") {
		t.Fatal("bybit 1h")
	}
}
