package domain

import "time"

// CandleInterval is a supported OHLCV bar size (Binance-compatible values).
type CandleInterval string

const (
	Interval1m  CandleInterval = "1m"
	Interval3m  CandleInterval = "3m"
	Interval5m  CandleInterval = "5m"
	Interval15m CandleInterval = "15m"
	Interval30m CandleInterval = "30m"
	Interval1h  CandleInterval = "1h"
	Interval2h  CandleInterval = "2h"
	Interval4h  CandleInterval = "4h"
	Interval6h  CandleInterval = "6h"
	Interval8h  CandleInterval = "8h"
	Interval12h CandleInterval = "12h"
	Interval1d  CandleInterval = "1d"
	Interval3d  CandleInterval = "3d"
	Interval1w  CandleInterval = "1w"
	Interval1M  CandleInterval = "1M"
)

// SupportedIntervals lists every interval accepted by the market API.
var SupportedIntervals = []CandleInterval{
	Interval1m, Interval3m, Interval5m, Interval15m, Interval30m,
	Interval1h, Interval2h, Interval4h, Interval6h, Interval8h, Interval12h,
	Interval1d, Interval3d, Interval1w, Interval1M,
}

// IsValidInterval reports whether s is a supported candle interval.
func IsValidInterval(s string) bool {
	for _, iv := range SupportedIntervals {
		if string(iv) == s {
			return true
		}
	}
	return false
}

// Candle is one OHLCV bar for a trading pair.
type Candle struct {
	OpenTime  time.Time
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
	CloseTime time.Time
	// QuoteVolume is volume in quote asset (e.g. USDT for BTCUSDT).
	QuoteVolume string
	TradeCount  int64
	// TakerBuyQuote is aggressive buy volume in quote asset when the venue
	// publishes it (Binance klines). Empty when unknown.
	TakerBuyQuote string
}

// CandleQuery is the input for fetching historical candles.
type CandleQuery struct {
	Symbol   string
	Interval CandleInterval
	Limit    int
	// StartTime / EndTime are optional (zero means omitted).
	StartTime time.Time
	EndTime   time.Time
}
