package domain

import "strings"

// Exchange identifies a supported spot market venue.
type Exchange string

const (
	ExchangeBinance  Exchange = "binance"
	ExchangeCoinbase Exchange = "coinbase"
	ExchangeBybit    Exchange = "bybit"
)

// SupportedExchanges is the ordered list of venues exposed by the API.
var SupportedExchanges = []Exchange{
	ExchangeBinance,
	ExchangeCoinbase,
	ExchangeBybit,
}

// DefaultExchange is used when the client omits ?exchange=.
const DefaultExchange = ExchangeBinance

// IsValidExchange reports whether s is a known venue id.
func IsValidExchange(s string) bool {
	return ParseExchange(s) != ""
}

// ParseExchange normalizes an exchange id; empty input yields DefaultExchange.
// Unknown non-empty values return "".
func ParseExchange(s string) Exchange {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return DefaultExchange
	}
	for _, e := range SupportedExchanges {
		if string(e) == s {
			return e
		}
	}
	return ""
}

// SupportedIntervalsFor returns candle intervals supported for the exchange.
// Coinbase public candles only support a subset of granularities.
func SupportedIntervalsFor(ex Exchange) []CandleInterval {
	switch ex {
	case ExchangeCoinbase:
		return []CandleInterval{
			Interval1m, Interval5m, Interval15m, Interval1h, Interval6h, Interval1d,
		}
	case ExchangeBybit:
		return []CandleInterval{
			Interval1m, Interval3m, Interval5m, Interval15m, Interval30m,
			Interval1h, Interval2h, Interval4h, Interval6h, Interval12h,
			Interval1d, Interval1w, Interval1M,
		}
	default: // binance
		return append([]CandleInterval(nil), SupportedIntervals...)
	}
}

// IsValidIntervalFor reports whether interval is allowed on the exchange.
func IsValidIntervalFor(ex Exchange, interval string) bool {
	for _, iv := range SupportedIntervalsFor(ex) {
		if string(iv) == interval {
			return true
		}
	}
	return false
}
