package binance

import "strings"

// nonCryptoTags are Binance product/marketing tags for non-crypto spot products
// we intentionally exclude (tokenized equities, commodity wrappers).
// Crypto focus: spot coins only — not NVDA/TSLA-style bStocks, not PAXG/XAUT commodities.
var nonCryptoTags = map[string]struct{}{
	"bstocks":       {}, // e.g. NVDABUSDT, TSLABUSDT
	"tcommodities":  {}, // e.g. PAXG, XAUT
}

// hasNonCryptoTag reports whether any tag marks a non-crypto product.
func hasNonCryptoTag(tags []string) bool {
	for _, t := range tags {
		if _, ok := nonCryptoTags[strings.ToLower(strings.TrimSpace(t))]; ok {
			return true
		}
	}
	return false
}
