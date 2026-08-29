package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	// DefaultRSIOversold / DefaultRSIOverbought are Wilder RSI bands.
	DefaultRSIOversold   = 30.0
	DefaultRSIOverbought = 70.0

	RSIHeatmapDefaultLimit    = 100
	RSIHeatmapMaxLimit        = 200
	// RSIHeatmapCandleLimit is closed bars for latest Wilder RSI only
	// (~8× default period). This is a scatter seed, not a 300-bar TV chart.
	RSIHeatmapCandleLimit     = 120
	RSIHeatmapDefaultInterval = "1h"
	RSIHeatmapCacheTTL        = time.Minute
	RSIHeatmapBuildTimeout    = 45 * time.Second
)

// RSIHeatmapStableBases are quote-like tickers omitted from the map so
// USDCUSDT does not crowd out real assets (CoinGlass / DataWallet convention).
var RSIHeatmapStableBases = map[string]struct{}{
	"USDT": {}, "USDC": {}, "BUSD": {}, "FDUSD": {}, "TUSD": {}, "DAI": {},
	"USDE": {}, "USD1": {}, "USDS": {}, "PYUSD": {}, "USDP": {}, "GUSD": {},
	"EUR": {}, "EURC": {}, "EURI": {}, "USD": {}, "RLUSD": {}, "BFUSD": {},
	"TUSDUSDT": {},
}

// RSIZone is the conventional oversold / neutral / overbought band.
type RSIZone string

const (
	RSIZoneOversold   RSIZone = "oversold"
	RSIZoneNeutral    RSIZone = "neutral"
	RSIZoneOverbought RSIZone = "overbought"
	RSIZoneUnknown    RSIZone = ""
)

// RSIHeatmapRow is one listed pair on the scatter (rank × RSI).
type RSIHeatmapRow struct {
	Rank                 int
	Symbol               string
	Base                 string
	LastPrice            string
	PriceChangePercent   string
	QuoteVolume          string
	MarketCapCirculating *float64
	RSI                  *float64
	Zone                 RSIZone
	Error                string
}

// RSIHeatmap is a CoinGlass-style ranked RSI scatter for top listed pairs.
type RSIHeatmap struct {
	Exchange        Exchange
	Quote           string
	Interval        string
	Period          int
	Oversold        float64
	Overbought      float64
	SortBy          string
	AverageRSI      *float64
	OversoldCount   int
	NeutralCount    int
	OverboughtCount int
	AsOf            time.Time
	Stale           bool
	Items           []RSIHeatmapRow
	Note            string
}

// RSIZoneFor classifies a Wilder RSI. nil is unknown (no zone).
func RSIZoneFor(rsi *float64, oversold, overbought float64) RSIZone {
	if rsi == nil {
		return RSIZoneUnknown
	}
	v := *rsi
	if oversold <= 0 {
		oversold = DefaultRSIOversold
	}
	if overbought <= 0 {
		overbought = DefaultRSIOverbought
	}
	if v <= oversold {
		return RSIZoneOversold
	}
	if v >= overbought {
		return RSIZoneOverbought
	}
	return RSIZoneNeutral
}

// LatestRSI returns the last defined Wilder RSI in closes, or nil.
func LatestRSI(closes []float64, period int) *float64 {
	if period <= 0 {
		period = DefaultRSIPeriod
	}
	series := RSIWilder(closes, period)
	for i := len(series) - 1; i >= 0; i-- {
		if series[i] != nil {
			v := *series[i]
			return &v
		}
	}
	return nil
}

// ParseRSIHeatmapInterval returns one venue-valid candle interval.
// A comma list is accepted for old clients; only the first value is used.
func ParseRSIHeatmapInterval(raw string, ex Exchange) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		s = RSIHeatmapDefaultInterval
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if s == "" {
		s = RSIHeatmapDefaultInterval
	}
	if !IsValidIntervalFor(ex, s) {
		return "", fmt.Errorf("%w: interval %q is not valid for %s", ErrInvalidArgument, s, ex)
	}
	return s, nil
}

// IsRSIHeatmapStableBase is true for quote-like tickers that should not appear as dots.
func IsRSIHeatmapStableBase(base string) bool {
	_, ok := RSIHeatmapStableBases[strings.ToUpper(strings.TrimSpace(base))]
	return ok
}

// SummarizeRSIHeatmap fills average RSI and zone counts from items.
func SummarizeRSIHeatmap(h *RSIHeatmap) {
	if h == nil {
		return
	}
	h.OversoldCount = 0
	h.NeutralCount = 0
	h.OverboughtCount = 0
	h.AverageRSI = nil
	var sum float64
	var n int
	for i := range h.Items {
		row := &h.Items[i]
		row.Rank = i + 1
		if row.RSI == nil {
			continue
		}
		sum += *row.RSI
		n++
		switch row.Zone {
		case RSIZoneOversold:
			h.OversoldCount++
		case RSIZoneOverbought:
			h.OverboughtCount++
		default:
			h.NeutralCount++
		}
	}
	if n > 0 {
		avg := sum / float64(n)
		h.AverageRSI = &avg
	}
}

// ClampRSIHeatmapLimit applies default and max symbol count.
func ClampRSIHeatmapLimit(n int) int {
	if n <= 0 {
		return RSIHeatmapDefaultLimit
	}
	if n > RSIHeatmapMaxLimit {
		return RSIHeatmapMaxLimit
	}
	return n
}

// RSIHeatmapCacheKey is the TTL key for one scatter (limit is not part of
// the key — a larger cached map can serve a smaller Top-N).
func RSIHeatmapCacheKey(ex Exchange, quote, sortBy, interval string, period int) string {
	return strings.ToUpper(string(ex)) + "|" + strings.ToUpper(strings.TrimSpace(quote)) + "|" +
		sortBy + "|" + interval + "|" + fmt.Sprintf("%d", period)
}

// RSIHeatmapFetchLimit is how many venue klines to pull for latest RSI.
func RSIHeatmapFetchLimit(period int) int {
	n := RSIHeatmapCandleLimit
	if period+50 > n {
		n = period + 50
	}
	return n
}

// ClipRSIHeatmap returns a copy with at most limit rows and fresh ranks/counts.
func ClipRSIHeatmap(h *RSIHeatmap, limit int) *RSIHeatmap {
	if h == nil {
		return nil
	}
	limit = ClampRSIHeatmapLimit(limit)
	cp := *h
	n := len(h.Items)
	if n > limit {
		n = limit
	}
	if n > 0 {
		cp.Items = append([]RSIHeatmapRow(nil), h.Items[:n]...)
	} else {
		cp.Items = nil
	}
	SummarizeRSIHeatmap(&cp)
	return &cp
}
