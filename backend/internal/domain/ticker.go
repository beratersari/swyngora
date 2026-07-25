package domain

import "time"

// Ticker24h is rolling 24-hour market statistics for a symbol.
type Ticker24h struct {
	Symbol             string
	PriceChange        string
	PriceChangePercent string
	LastPrice          string
	OpenPrice          string
	HighPrice          string
	LowPrice           string
	// Volume is base-asset volume over the last 24 hours.
	Volume string
	// QuoteVolume is quote-asset volume over the last 24 hours.
	QuoteVolume string
	OpenTime    time.Time
	CloseTime   time.Time
	TradeCount  int64
}
