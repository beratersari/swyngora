package domain

import "strings"

// NormalizeSymbol formats a trading pair for the given exchange.
// Coinbase product ids are hyphenated (BTC-USD); bare BTCUSD is expanded when
// a known quote suffix is present. Binance/Bybit use concatenated upper symbols.
func NormalizeSymbol(ex Exchange, symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return ""
	}
	if ex == ExchangeCoinbase {
		symbol = strings.ToUpper(symbol)
		if !strings.Contains(symbol, "-") {
			// Longest quotes first so USDT wins over USD.
			for _, q := range []string{"USDT", "USDC", "USD", "EUR", "GBP", "BTC", "ETH"} {
				if strings.HasSuffix(symbol, q) && len(symbol) > len(q) {
					base := strings.TrimSuffix(symbol, q)
					if base != "" {
						return base + "-" + q
					}
				}
			}
		}
		return symbol
	}
	return strings.ToUpper(strings.ReplaceAll(symbol, "-", ""))
}
