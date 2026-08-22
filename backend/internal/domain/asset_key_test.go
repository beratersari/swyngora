package domain

import "testing"

func TestNormalizeAssetKey(t *testing.T) {
	cases := map[string]string{
		"BTCUSDT":    "BTC",
		"btc-usd":    "BTC",
		"ETHTRY":     "ETH",
		"BTCEUR":     "BTC",
		"ETHBTC":     "ETH",
		"RLUSD":      "RLUSD",
		"BFUSD":      "BFUSD",
		"TUSD":       "TUSD",
		"TUSDUSDT":   "TUSD",
		"BTC":        "BTC",
		"  ethusdc ": "ETH",
		"":           "",
	}
	for in, want := range cases {
		if got := NormalizeAssetKey(in); got != want {
			t.Fatalf("NormalizeAssetKey(%q)=%q want %q", in, got, want)
		}
	}
}
