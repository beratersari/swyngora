package domain

import (
	"math"
	"sort"
	"time"
)

const (
	DefaultLongShortHistoryLimit = 24
	MaxLongShortHistoryLimit     = 100
	LongShortPeriod5m            = "5m"
	LongShortKindAccounts        = "accounts" // share of accounts that are long vs short
)

// LongShortPoint is one long/short account-ratio sample.
type LongShortPoint struct {
	Time       time.Time
	LongShare  float64 // 0–1 share of accounts that are long
	ShortShare float64 // 0–1
	Ratio      float64 // long/short; 1 = even
}

// LongShortSeries is one venue's latest sample plus recent history.
type LongShortSeries struct {
	Exchange Exchange
	Symbol   string
	Kind     string // accounts
	Period   string // 5m
	Current  LongShortPoint
	History  []LongShortPoint // newest first, excluding current
}

// LongShortLevel is a formatted ratio print.
type LongShortLevel struct {
	Time      time.Time
	LongPct   string
	ShortPct  string
	Ratio     string
	Bias      string // long | short | balanced
	LongShare string // 0–1 decimal, for AI
}

// LongShortVenueSnap is one venue inside a snapshot.
type LongShortVenueSnap struct {
	Exchange string
	Kind     string
	Period   string
	Current  LongShortLevel
	Change   string // signed ratio change vs previous print
	History  []LongShortLevel
}

// LongShortSnapshot is current long/short ratio plus recent 5m history.
type LongShortSnapshot struct {
	Symbol     string
	Exchange   string // binance | bybit | all
	Kind       string
	Period     string
	Current    *LongShortLevel
	Venues     []LongShortVenueSnap
	History    []LongShortLevel
	AsOf       time.Time
	VenueCount int
}

// ClampLongShortHistoryLimit bounds history size.
func ClampLongShortHistoryLimit(n int) int {
	if n <= 0 {
		return DefaultLongShortHistoryLimit
	}
	if n > MaxLongShortHistoryLimit {
		return MaxLongShortHistoryLimit
	}
	return n
}

// LongShortRatioFromShares is long/short; 0 short-share → +Inf treated as 0 invalid.
func LongShortRatioFromShares(longShare, shortShare float64) float64 {
	if shortShare > 1e-12 {
		return longShare / shortShare
	}
	if longShare > 1e-12 {
		return 99
	}
	return 0
}

// LongShortBias labels crowding. 5% band around 1.0 is balanced.
func LongShortBias(ratio float64) string {
	switch {
	case ratio >= 1.05:
		return "long"
	case ratio > 0 && ratio <= 0.95:
		return "short"
	default:
		return "balanced"
	}
}

// FormatLongShortLevel renders one sample.
func FormatLongShortLevel(p LongShortPoint) LongShortLevel {
	out := LongShortLevel{
		LongPct:   formatFixed(p.LongShare*100, 2),
		ShortPct:  formatFixed(p.ShortShare*100, 2),
		Ratio:     formatFixed(p.Ratio, 4),
		Bias:      LongShortBias(p.Ratio),
		LongShare: formatFixed(p.LongShare, 4),
	}
	if !p.Time.IsZero() {
		out.Time = p.Time.UTC()
	}
	return out
}

// SortLongShortNewestFirst sorts and dedups by timestamp (keeps last).
func SortLongShortNewestFirst(hist []LongShortPoint) []LongShortPoint {
	if len(hist) == 0 {
		return hist
	}
	sort.SliceStable(hist, func(i, j int) bool {
		return hist[i].Time.After(hist[j].Time)
	})
	out := hist[:0]
	for _, p := range hist {
		if p.Time.IsZero() {
			continue
		}
		if len(out) > 0 && out[len(out)-1].Time.Equal(p.Time) {
			out[len(out)-1] = p
			continue
		}
		out = append(out, p)
	}
	return out
}

func venueLongShortSnap(s *LongShortSeries) LongShortVenueSnap {
	out := LongShortVenueSnap{
		Exchange: string(s.Exchange),
		Kind:     s.Kind,
		Period:   s.Period,
		Current:  FormatLongShortLevel(s.Current),
		History:  make([]LongShortLevel, 0, len(s.History)),
	}
	if len(s.History) > 0 {
		out.Change = FormatSignedQty(s.Current.Ratio - s.History[0].Ratio)
	}
	for _, p := range s.History {
		out.History = append(out.History, FormatLongShortLevel(p))
	}
	return out
}

// BuildLongShortSnapshot folds one or more venue series.
func BuildLongShortSnapshot(exchange, symbol string, series []*LongShortSeries, now time.Time) *LongShortSnapshot {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	out := &LongShortSnapshot{
		Symbol:   symbol,
		Exchange: exchange,
		Kind:     LongShortKindAccounts,
		Period:   LongShortPeriod5m,
		AsOf:     now,
		Venues:   []LongShortVenueSnap{},
		History:  []LongShortLevel{},
	}
	clean := make([]*LongShortSeries, 0, len(series))
	for _, s := range series {
		if s == nil {
			continue
		}
		s.History = SortLongShortNewestFirst(s.History)
		if s.Kind == "" {
			s.Kind = LongShortKindAccounts
		}
		if s.Period == "" {
			s.Period = LongShortPeriod5m
		}
		if s.Current.Time.IsZero() && len(s.History) > 0 {
			s.Current = s.History[0]
			s.History = s.History[1:]
		}
		clean = append(clean, s)
	}
	sort.Slice(clean, func(i, j int) bool {
		return string(clean[i].Exchange) < string(clean[j].Exchange)
	})
	out.VenueCount = len(clean)
	for _, s := range clean {
		out.Venues = append(out.Venues, venueLongShortSnap(s))
	}
	if len(clean) == 1 {
		cur := FormatLongShortLevel(clean[0].Current)
		out.Current = &cur
		out.History = append([]LongShortLevel(nil), out.Venues[0].History...)
		out.Kind = clean[0].Kind
		out.Period = clean[0].Period
	}
	return out
}

// NormalizeLongShortPoint fills ratio from shares when missing.
func NormalizeLongShortPoint(p LongShortPoint) LongShortPoint {
	if p.Ratio <= 0 && (p.LongShare > 0 || p.ShortShare > 0) {
		p.Ratio = LongShortRatioFromShares(p.LongShare, p.ShortShare)
	}
	if p.LongShare+p.ShortShare > 0 {
		sum := p.LongShare + p.ShortShare
		if math.Abs(sum-1) > 0.02 && sum > 0 {
			p.LongShare /= sum
			p.ShortShare /= sum
		}
	}
	return p
}
