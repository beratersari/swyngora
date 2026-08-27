package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	DefaultFundingHistoryLimit = 12
	MaxFundingHistoryLimit     = 30
	DefaultFundingIntervalHrs  = 8
)

// FundingPoint is one settled (or current predicted) funding print.
type FundingPoint struct {
	Time      time.Time
	Rate      float64 // decimal, e.g. 0.0001 = 0.01%
	MarkPrice float64 // 0 when the venue does not publish it
	Predicted bool
}

// FundingSeries is one venue's current predicted rate plus settled history.
type FundingSeries struct {
	Exchange        Exchange
	Symbol          string
	Current         FundingPoint
	NextFundingTime time.Time
	IntervalHours   int
	History         []FundingPoint // newest first, settled only
}

// FundingPrint is a formatted rate for the API.
type FundingPrint struct {
	Time      time.Time
	Rate      string // decimal string
	RatePct   string // percent string
	Payer     string // long | short | none — who pays
	MarkPrice string
	Predicted bool
}

// FundingCurrent is the live / next-settlement estimate.
type FundingCurrent struct {
	Rate            string
	RatePct         string
	Payer           string
	NextFundingTime time.Time
	IntervalHours   int
	Time            time.Time
}

// FundingVenueSnap is one venue inside a snapshot.
type FundingVenueSnap struct {
	Exchange    string
	Current     FundingCurrent
	LastSettled *FundingPrint
	AvgLast3    string // decimal avg of last 3 settlements (~24h at 8h)
	AvgLast3Pct string
	History     []FundingPrint
}

// FundingSnapshot is current funding plus recent settlements.
type FundingSnapshot struct {
	Symbol     string
	Exchange   string // binance | bybit | all
	Current    *FundingCurrent
	Venues     []FundingVenueSnap
	History    []FundingPrint // hoisted when a single venue is requested
	AsOf       time.Time
	VenueCount int
}

// ClampFundingHistoryLimit bounds history size.
func ClampFundingHistoryLimit(n int) int {
	if n <= 0 {
		return DefaultFundingHistoryLimit
	}
	if n > MaxFundingHistoryLimit {
		return MaxFundingHistoryLimit
	}
	return n
}

// InferFundingIntervalHours uses the venue hint, else next-last gap, else 8h.
func InferFundingIntervalHours(hinted int, next, lastSettled time.Time) int {
	if hinted >= 1 && hinted <= 24 {
		return hinted
	}
	if !next.IsZero() && !lastSettled.IsZero() {
		h := int(math.Round(next.Sub(lastSettled).Hours()))
		if h >= 1 && h <= 24 {
			return h
		}
	}
	return DefaultFundingIntervalHrs
}

// FundingPayer is who pays when rate is applied: positive → longs pay shorts.
func FundingPayer(rate float64) string {
	switch {
	case rate > 1e-12:
		return "long"
	case rate < -1e-12:
		return "short"
	default:
		return "none"
	}
}

// FormatFundingRate returns decimal and percent strings.
func FormatFundingRate(rate float64) (decimal, pct string) {
	return formatFixed(rate, 8), formatFixed(rate*100, 6)
}

// FormatFundingPrint renders one point.
func FormatFundingPrint(p FundingPoint) FundingPrint {
	dec, pct := FormatFundingRate(p.Rate)
	out := FundingPrint{
		Time:      p.Time.UTC(),
		Rate:      dec,
		RatePct:   pct,
		Payer:     FundingPayer(p.Rate),
		Predicted: p.Predicted,
	}
	if p.Time.IsZero() {
		out.Time = time.Time{}
	}
	if p.MarkPrice > 0 {
		out.MarkPrice = formatQty(p.MarkPrice)
	}
	return out
}

// FormatFundingCurrent renders the live estimate.
func FormatFundingCurrent(s *FundingSeries) FundingCurrent {
	if s == nil {
		return FundingCurrent{}
	}
	dec, pct := FormatFundingRate(s.Current.Rate)
	out := FundingCurrent{
		Rate:          dec,
		RatePct:       pct,
		Payer:         FundingPayer(s.Current.Rate),
		IntervalHours: s.IntervalHours,
		Time:          s.Current.Time.UTC(),
	}
	if !s.NextFundingTime.IsZero() {
		out.NextFundingTime = s.NextFundingTime.UTC()
	}
	if s.Current.Time.IsZero() {
		out.Time = time.Time{}
	}
	return out
}

// AverageFundingRate averages the first n history points (newest-first).
func AverageFundingRate(hist []FundingPoint, n int) (float64, bool) {
	if n <= 0 || len(hist) == 0 {
		return 0, false
	}
	if n > len(hist) {
		n = len(hist)
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += hist[i].Rate
	}
	return sum / float64(n), true
}

// SortFundingHistoryOldestFirst sorts by time ascending and drops zero times.
func SortFundingHistoryOldestFirst(hist []FundingPoint) []FundingPoint {
	if len(hist) == 0 {
		return hist
	}
	sort.SliceStable(hist, func(i, j int) bool {
		return hist[i].Time.Before(hist[j].Time)
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

// SortFundingHistoryNewestFirst sorts and keeps the newest unique timestamp.
func SortFundingHistoryNewestFirst(hist []FundingPoint) []FundingPoint {
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

func venueFundingSnap(s *FundingSeries) FundingVenueSnap {
	out := FundingVenueSnap{
		Exchange: string(s.Exchange),
		Current:  FormatFundingCurrent(s),
		History:  make([]FundingPrint, 0, len(s.History)),
	}
	if len(s.History) > 0 {
		p := FormatFundingPrint(s.History[0])
		out.LastSettled = &p
	}
	if avg, ok := AverageFundingRate(s.History, 3); ok {
		dec, pct := FormatFundingRate(avg)
		out.AvgLast3, out.AvgLast3Pct = dec, pct
	}
	for _, p := range s.History {
		out.History = append(out.History, FormatFundingPrint(p))
	}
	return out
}

// BuildFundingSnapshot folds one or more venue series.
func BuildFundingSnapshot(exchange, symbol string, series []*FundingSeries, now time.Time) *FundingSnapshot {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	out := &FundingSnapshot{
		Symbol:   symbol,
		Exchange: exchange,
		AsOf:     now,
		Venues:   []FundingVenueSnap{},
		History:  []FundingPrint{},
	}
	clean := make([]*FundingSeries, 0, len(series))
	for _, s := range series {
		if s == nil {
			continue
		}
		s.History = SortFundingHistoryNewestFirst(s.History)
		if s.IntervalHours <= 0 {
			var last time.Time
			if len(s.History) > 0 {
				last = s.History[0].Time
			}
			s.IntervalHours = InferFundingIntervalHours(s.IntervalHours, s.NextFundingTime, last)
		}
		clean = append(clean, s)
	}
	sort.Slice(clean, func(i, j int) bool {
		return string(clean[i].Exchange) < string(clean[j].Exchange)
	})
	out.VenueCount = len(clean)
	for _, s := range clean {
		out.Venues = append(out.Venues, venueFundingSnap(s))
	}
	if len(clean) == 1 {
		cur := FormatFundingCurrent(clean[0])
		out.Current = &cur
		out.History = append([]FundingPrint(nil), out.Venues[0].History...)
	}
	return out
}

// ValidateFundingLimit documents Clamp for callers.
func ValidateFundingLimit(n int) int {
	return ClampFundingHistoryLimit(n)
}

// ErrFundingSymbol is a typed empty-symbol error.
func ErrFundingSymbol() error {
	return fmt.Errorf("%w: symbol is required", ErrInvalidArgument)
}
