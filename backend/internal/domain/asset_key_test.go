package domain

import "testing"

func TestNormalizeAssetKey(t *testing.T) {
	cases := map[string]string{
		"BTCUSDT":    "BTC",
		"btc-usd":    "BTC",
		"ETHTRY":     "ETH",
		"BTCEUR":     "BTC",
		"ETHBTC":     "ETH",
		"BNBBTC":     "BNB",
		"SOLBTC":     "SOL",
		"OPBTC":      "OP",
		"WBTCUSDT":   "WBTC",
		"STETHUSDT":  "STETH",
		"WBTC":       "WBTC",
		"WETH":       "WETH",
		"WBNB":       "WBNB",
		"STETH":      "STETH",
		"WSTETH":     "WSTETH",
		"CBETH":      "CBETH",
		"WBETH":      "WBETH",
		"W":          "W",
		"ST":         "ST",
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
