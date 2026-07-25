package binance

import "testing"

func TestHasNonCryptoTag(t *testing.T) {
	if !hasNonCryptoTag([]string{"bStocks"}) {
		t.Fatal("bStocks should be non-crypto")
	}
	if !hasNonCryptoTag([]string{"Payments", "tCommodities"}) {
		t.Fatal("tCommodities should be non-crypto")
	}
	if hasNonCryptoTag([]string{"Payments", "Meme", "defi"}) {
		t.Fatal("crypto tags only")
	}
	if hasNonCryptoTag(nil) {
		t.Fatal("empty tags are crypto-eligible")
	}
}
