package domain

import (
	"fmt"
	"strings"
)

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
	if IsEquityExchange(ex) {
		s := strings.ToUpper(strings.ReplaceAll(symbol, "-", ""))
		s = strings.TrimSuffix(s, ".IS")
		return s
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
	if IsEquityExchange(ex) {
		return []string{DefaultQuoteAsset(ex)}
	}
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
	if IsEquityExchange(ex) {
		return symbol, DefaultQuoteAsset(ex)
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

// AssetQuoteSuffixes are longest-first quote tails stripped to resolve a base
// asset key for supply / holders catalog lookup. Bare "USD" is omitted so
// RLUSD / BFUSD stay intact; hyphenated BASE-USD is handled in NormalizeAssetKey.
// BTC/ETH/BNB are only peeled when the leftover is a known pair base (see
// NormalizeAssetKey) so WBTC / STETH are not collapsed onto W / ST.
var AssetQuoteSuffixes = []string{
	"FDUSD", "USDT", "USDC", "BUSD", "TUSD", "DAI",
	"EUR", "TRY", "BRL", "GBP", "AUD", "CAD", "ARS", "JPY",
	"BTC", "ETH", "BNB",
}

// cryptoQuoteSuffixes can also be real token tails (WBTC, STETH). Only strip
// them when the leftover is a known independent pair base (ETHBTC → ETH).
var cryptoQuoteSuffixes = map[string]struct{}{
	"BTC": {}, "ETH": {}, "BNB": {},
}

// cryptoPairBases are tickers that may be the left side of a BTC/ETH/BNB pair.
// Wrapped or staked names that merely end in those quotes are omitted.
var cryptoPairBases = map[string]struct{}{
	"BTC": {}, "ETH": {}, "BNB": {},
	"SOL": {}, "XRP": {}, "ADA": {}, "DOGE": {}, "AVAX": {}, "DOT": {},
	"LINK": {}, "ATOM": {}, "LTC": {}, "BCH": {}, "UNI": {}, "APT": {},
	"SUI": {}, "NEAR": {}, "FIL": {}, "INJ": {}, "TIA": {}, "SEI": {},
	"OP": {}, "ARB": {}, "AAVE": {}, "MKR": {}, "LDO": {}, "TRX": {},
	"TON": {}, "HBAR": {}, "ICP": {}, "XLM": {}, "ETC": {}, "APE": {},
	"PEPE": {}, "SHIB": {}, "WIF": {}, "BONK": {}, "JUP": {}, "FET": {},
	"GRT": {}, "SAND": {}, "MANA": {}, "CRV": {}, "SNX": {}, "IMX": {},
}

func isCryptoQuoteSuffix(q string) bool {
	_, ok := cryptoQuoteSuffixes[q]
	return ok
}

func isCryptoPairBase(base string) bool {
	_, ok := cryptoPairBases[base]
	return ok
}

// NormalizeAssetKey maps a trading pair or base ticker to the supply/holders
// catalog key (BTCUSDT / BTC-USD / ETHTRY → BTC, ETHBTC → ETH). Exact bases
// that only look like a quote tail (TUSD, RLUSD) stay intact, as do wrapped
// or staked tickers that end in BTC/ETH/BNB (WBTC, STETH, WSTETH).
func NormalizeAssetKey(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '-'); i > 0 {
		s = strings.TrimSpace(s[:i])
		if s == "" {
			return strings.ToUpper(strings.TrimSpace(raw))
		}
	}
	for _, q := range AssetQuoteSuffixes {
		if len(s) > len(q) && strings.HasSuffix(s, q) {
			base := strings.TrimSuffix(s, q)
			if base == "" || base == q {
				continue
			}
			if isCryptoQuoteSuffix(q) && !isCryptoPairBase(base) {
				continue
			}
			return base
		}
	}
	return s
}

// RequireQuoteMatchesCurrency rejects pairs whose quote asset is not the portfolio cash unit.
func RequireQuoteMatchesCurrency(ex Exchange, symbol, currency string) error {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return fmt.Errorf("%w: portfolio currency is required", ErrInvalidArgument)
	}
	_, quote := SplitBaseQuote(ex, symbol)
	if quote == "" {
		return fmt.Errorf("%w: cannot determine quote asset for %s", ErrInvalidArgument, symbol)
	}
	if quote != currency {
		return fmt.Errorf("%w: pair quote %s does not match portfolio currency %s", ErrInvalidArgument, quote, currency)
	}
	return nil
}

// PairSymbol builds a venue pair from base + quote.
func PairSymbol(ex Exchange, base, quote string) string {
	base = strings.ToUpper(strings.TrimSpace(base))
	quote = strings.ToUpper(strings.TrimSpace(quote))
	if base == "" || quote == "" {
		return ""
	}
	if IsEquityExchange(ex) {
		return NormalizeSymbol(ex, base)
	}
	if ex == ExchangeCoinbase {
		return base + "-" + quote
	}
	return NormalizeSymbol(ex, base+quote)
}
