package okx

import (
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SwapInstID maps BTCUSDT → BTC-USDT-SWAP.
func SwapInstID(symbol string) string {
	raw := strings.ToUpper(strings.TrimSpace(symbol))
	if raw == "" {
		return ""
	}
	if strings.HasSuffix(raw, "-SWAP") {
		return raw
	}
	if strings.HasSuffix(raw, "SWAP") {
		trimmed := strings.TrimSuffix(strings.ReplaceAll(raw, "-", ""), "SWAP")
		return SwapInstID(trimmed)
	}
	sym := domain.NormalizeLiquidationSymbol(symbol)
	if sym == "" {
		return ""
	}
	for _, q := range []string{"USDT", "USDC"} {
		if strings.HasSuffix(sym, q) && len(sym) > len(q) {
			return sym[:len(sym)-len(q)] + "-" + q + "-SWAP"
		}
	}
	return ""
}

// SymbolFromInstID maps BTC-USDT-SWAP → BTCUSDT.
func SymbolFromInstID(instID string) string {
	s := strings.ToUpper(strings.TrimSpace(instID))
	s = strings.TrimSuffix(s, "-SWAP")
	if s == "" {
		return ""
	}
	return strings.ReplaceAll(s, "-", "")
}

// UnderlyingFromSymbol maps BTCUSDT → BTC-USDT (OKX uly).
func UnderlyingFromSymbol(symbol string) string {
	inst := SwapInstID(symbol)
	return strings.TrimSuffix(inst, "-SWAP")
}
