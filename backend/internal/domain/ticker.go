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
	// Halted is true when this ticker is a last print before a delist halt,
	// not a live rolling 24h window.
	Halted bool
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
		Halted:      true,
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

// SyncTickerChangeFromOpenLast sets PriceChange / PriceChangePercent from open and last
// so a ticker never mixes last/open from one window with % from another.
func SyncTickerChangeFromOpenLast(t *Ticker24h) {
	if t == nil {
		return
	}
	open, err1 := strconv.ParseFloat(strings.TrimSpace(t.OpenPrice), 64)
	last, err2 := strconv.ParseFloat(strings.TrimSpace(t.LastPrice), 64)
	if err1 != nil || err2 != nil || open == 0 {
		return
	}
	chg := last - open
	t.PriceChange = strconv.FormatFloat(chg, 'f', -1, 64)
	t.PriceChangePercent = strconv.FormatFloat(chg/open*100, 'f', -1, 64)
}
