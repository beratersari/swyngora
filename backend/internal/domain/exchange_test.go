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

func TestIsValidExchange_EmptyRejected(t *testing.T) {
	if IsValidExchange("") || IsValidExchange("  ") {
		t.Fatal("empty must not be a known venue id")
	}
	if !IsValidExchange("binance") || !IsValidExchange("Coinbase") {
		t.Fatal("known venues")
	}
	if IsValidExchange("kraken") {
		t.Fatal("unknown must be false")
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

func TestIsEquityExchangeAndDefaultQuote(t *testing.T) {
	if !IsEquityExchange(ExchangeNasdaq) || !IsEquityExchange(ExchangeBist) {
		t.Fatal("equity venues")
	}
	if IsEquityExchange(ExchangeBinance) {
		t.Fatal("binance is not equity")
	}
	if DefaultQuoteAsset(ExchangeNasdaq) != "USD" || DefaultQuoteAsset(ExchangeBist) != "TRY" {
		t.Fatal("equity quotes")
	}
	if !IsValidIntervalFor(ExchangeNasdaq, "1d") || IsValidIntervalFor(ExchangeNasdaq, "3m") {
		t.Fatal("nasdaq intervals")
	}
	if got := NormalizeSymbol(ExchangeBist, "thyao.is"); got != "THYAO" {
		t.Fatalf("bist norm %q", got)
	}
	if b, q := SplitBaseQuote(ExchangeNasdaq, "AAPL"); b != "AAPL" || q != "USD" {
		t.Fatalf("split %s %s", b, q)
	}
}

func TestSupportedIntervalsFor_UnknownExchangeNil(t *testing.T) {
	if got := SupportedIntervalsFor(Exchange("kraken")); got != nil {
		t.Fatalf("unknown exchange should return nil, got %v", got)
	}
	// Zero-value / binance still get the full list.
	if len(SupportedIntervalsFor(ExchangeBinance)) == 0 {
		t.Fatal("binance intervals empty")
	}
}
