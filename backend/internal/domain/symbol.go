package domain

import "strings"

// CrossVenueSymbol maps a user pair onto another venue (BTCUSDT ↔ BTC-USD).
// Coinbase USDT-style pairs use USD; Binance/Bybit USD pairs use USDT.
func CrossVenueSymbol(ex Exchange, input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	bin := NormalizeSymbol(ExchangeBinance, input)
	if ex != ExchangeCoinbase {
		if strings.HasSuffix(bin, "USD") && !strings.HasSuffix(bin, "USDT") && !strings.HasSuffix(bin, "USDC") {
			bin = strings.TrimSuffix(bin, "USD") + "USDT"
		}
		return NormalizeSymbol(ex, bin)
	}
	return PriceDiffSymbolForExchange(ex, bin)
}

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

// KnownQuoteAssets are longest-first quote suffixes used to split a pair into base/quote.
func KnownQuoteAssets(ex Exchange) []string {
	if ex == ExchangeCoinbase {
		return []string{"USDT", "USDC", "USD", "EUR", "GBP", "BTC", "ETH"}
	}
	return []string{"USDT", "USDC", "BUSD", "FDUSD", "BTC", "ETH", "BNB", "EUR", "TRY"}
}

// SplitBaseQuote splits a normalized pair into base and quote (e.g. BTCUSDT → BTC, USDT).
func SplitBaseQuote(ex Exchange, symbol string) (base, quote string) {
	symbol = NormalizeSymbol(ex, symbol)
	if symbol == "" {
		return "", ""
	}
	if ex == ExchangeCoinbase && strings.Contains(symbol, "-") {
		parts := strings.SplitN(symbol, "-", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	for _, q := range KnownQuoteAssets(ex) {
		if strings.HasSuffix(symbol, q) && len(symbol) > len(q) {
			return strings.TrimSuffix(symbol, q), q
		}
	}
	return symbol, ""
}

// PairSymbol builds a venue pair from base + quote.
func PairSymbol(ex Exchange, base, quote string) string {
	base = strings.ToUpper(strings.TrimSpace(base))
	quote = strings.ToUpper(strings.TrimSpace(quote))
	if base == "" || quote == "" {
		return ""
	}
	if ex == ExchangeCoinbase {
		return base + "-" + quote
	}
	return NormalizeSymbol(ex, base+quote)
}
