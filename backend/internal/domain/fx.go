package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	// FxBaseUSD is the internal rate base (units of currency per 1 USD).
	FxBaseUSD = "USD"
	FxUSDT    = "USDT"
	FxTRY     = "TRY"
	FxEUR     = "EUR"
)

// DisplayFxCodes are fiat/stable codes the desk can convert between.
var DisplayFxCodes = []string{FxBaseUSD, FxUSDT, FxTRY, FxEUR}

// FxRates is a USD-based spot snapshot for display conversion only.
type FxRates struct {
	Base  string
	AsOf  time.Time
	Rates map[string]float64 // units of currency per 1 USD
	Stale bool
	Note  string
}

// NormalizeFxCode uppercases a currency code.
func NormalizeFxCode(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

// AliasFxCode maps stables onto USD for display FX (USDT ≈ USD).
func AliasFxCode(code string) string {
	switch NormalizeFxCode(code) {
	case FxUSDT, "USDC", "BUSD":
		return FxBaseUSD
	default:
		return NormalizeFxCode(code)
	}
}

// QuoteForVenue is the native quote of last/open/high/low/quoteVolume.
func QuoteForVenue(ex Exchange) string {
	switch ParseExchange(string(ex)) {
	case ExchangeBist:
		return FxTRY
	case ExchangeNasdaq, ExchangeCoinbase:
		return FxBaseUSD
	default:
		return FxUSDT
	}
}

// QuoteForMarketCap is the currency of circulating/total/max market-cap fields.
// Crypto mcap is USD; BIST listings publish TRY; Nasdaq is USD.
func QuoteForMarketCap(ex Exchange) string {
	if ParseExchange(string(ex)) == ExchangeBist {
		return FxTRY
	}
	return FxBaseUSD
}

// ConvertFx converts amount from one currency to another using USD-based rates.
func ConvertFx(amount float64, from, to string, rates map[string]float64) (float64, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, fmt.Errorf("%w: amount must be finite", ErrInvalidArgument)
	}
	src := AliasFxCode(from)
	dst := AliasFxCode(to)
	if src == "" || dst == "" {
		return 0, fmt.Errorf("%w: currency is required", ErrInvalidArgument)
	}
	if src == dst {
		return amount, nil
	}
	fromPerUSD, err := fxPerUSD(src, rates)
	if err != nil {
		return 0, err
	}
	toPerUSD, err := fxPerUSD(dst, rates)
	if err != nil {
		return 0, err
	}
	return amount / fromPerUSD * toPerUSD, nil
}

func fxPerUSD(code string, rates map[string]float64) (float64, error) {
	if code == FxBaseUSD {
		return 1, nil
	}
	if rates == nil {
		return 0, fmt.Errorf("%w: missing FX rate for %s", ErrUpstream, code)
	}
	v, ok := rates[code]
	if !ok || v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%w: missing FX rate for %s", ErrUpstream, code)
	}
	return v, nil
}
