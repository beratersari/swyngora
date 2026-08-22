package domain

import (
	"strconv"
	"strings"
	"time"
)

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

// TickerFromLastCandle builds a last-print ticker from the final kline before a halt.
func TickerFromLastCandle(symbol string, c Candle) Ticker24h {
	out := Ticker24h{
		Symbol:      strings.ToUpper(strings.TrimSpace(symbol)),
		LastPrice:   c.Close,
		OpenPrice:   c.Open,
		HighPrice:   c.High,
		LowPrice:    c.Low,
		Volume:      c.Volume,
		QuoteVolume: c.QuoteVolume,
		OpenTime:    c.OpenTime,
		CloseTime:   c.CloseTime,
		TradeCount:  c.TradeCount,
	}
	o, e1 := strconv.ParseFloat(c.Open, 64)
	cl, e2 := strconv.ParseFloat(c.Close, 64)
	if e1 == nil && e2 == nil && o != 0 {
		chg := cl - o
		out.PriceChange = strconv.FormatFloat(chg, 'f', -1, 64)
		out.PriceChangePercent = strconv.FormatFloat(chg/o*100, 'f', 2, 64)
	}
	return out
}
