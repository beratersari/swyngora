package domain

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// IndicatorPoint is one bar of indicator values aligned to a candle close.
// RSI/EMA values are nil until the series has enough history (warm-up).
type IndicatorPoint struct {
	OpenTime time.Time
	Close    float64
	// RSI is 0–100 when defined.
	RSI *float64
	// EMA maps period → value (e.g. 12, 26).
	EMA map[int]*float64
}

// IndicatorSeries is the response shape for technical indicators on a symbol.
type IndicatorSeries struct {
	Exchange  Exchange
	Symbol    string
	Interval  CandleInterval
	RSIPeriod int
	EMAPeriods []int
	Points    []IndicatorPoint
	// Latest non-nil values for quick UI badges (may be nil if insufficient data).
	LatestRSI *float64
	LatestEMA map[int]*float64
}

// Default indicator periods (common swing defaults).
const (
	DefaultRSIPeriod = 14
	DefaultEMAFast   = 12
	DefaultEMASlow   = 26
	MinIndicatorPeriod = 2
	MaxIndicatorPeriod = 500
)

// ParseClosePrices extracts close prices from candles (chronological).
// Invalid/empty closes are skipped (gap); better than panicking.
func ParseClosePrices(candles []Candle) []float64 {
	out := make([]float64, 0, len(candles))
	for _, c := range candles {
		v, err := parseClose(c.Close)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func parseClose(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(s, 64)
}

// RSIWilder computes Wilder's RSI for each close. Output length matches closes.
// Values before index period are nil (insufficient data).
func RSIWilder(closes []float64, period int) []*float64 {
	n := len(closes)
	out := make([]*float64, n)
	if period < MinIndicatorPeriod || n <= period {
		return out
	}

	var sumGain, sumLoss float64
	for i := 1; i <= period; i++ {
		d := closes[i] - closes[i-1]
		if d >= 0 {
			sumGain += d
		} else {
			sumLoss -= d
		}
	}
	avgGain := sumGain / float64(period)
	avgLoss := sumLoss / float64(period)
	out[period] = ptr(rsiFromAverages(avgGain, avgLoss))

	for i := period + 1; i < n; i++ {
		d := closes[i] - closes[i-1]
		var gain, loss float64
		if d >= 0 {
			gain = d
		} else {
			loss = -d
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		out[i] = ptr(rsiFromAverages(avgGain, avgLoss))
	}
	return out
}

func rsiFromAverages(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		if avgGain == 0 {
			return 50
		}
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// EMA computes exponential moving average. Output length matches closes.
// Values before index period-1 are nil. Seed is SMA of the first period closes.
func EMA(closes []float64, period int) []*float64 {
	n := len(closes)
	out := make([]*float64, n)
	if period < MinIndicatorPeriod || n < period {
		return out
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	ema := sum / float64(period)
	out[period-1] = ptr(ema)
	k := 2.0 / float64(period+1)
	for i := period; i < n; i++ {
		ema = closes[i]*k + ema*(1-k)
		out[i] = ptr(ema)
	}
	return out
}

// BuildIndicatorSeries aligns RSI and EMA series onto candle timestamps.
// candles must be chronological (oldest first). closes should match candle order
// when all closes parse; if some skips occurred, alignment uses min length.
func BuildIndicatorSeries(candles []Candle, rsiPeriod int, emaPeriods []int) []IndicatorPoint {
	if len(candles) == 0 {
		return nil
	}
	closes := make([]float64, 0, len(candles))
	times := make([]time.Time, 0, len(candles))
	for _, c := range candles {
		v, err := parseClose(c.Close)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		closes = append(closes, v)
		times = append(times, c.OpenTime)
	}
	rsi := RSIWilder(closes, rsiPeriod)
	emaSeries := make(map[int][]*float64, len(emaPeriods))
	for _, p := range emaPeriods {
		emaSeries[p] = EMA(closes, p)
	}

	out := make([]IndicatorPoint, len(closes))
	for i := range closes {
		pt := IndicatorPoint{
			OpenTime: times[i],
			Close:    closes[i],
			RSI:      rsi[i],
			EMA:      make(map[int]*float64, len(emaPeriods)),
		}
		for _, p := range emaPeriods {
			if s, ok := emaSeries[p]; ok && i < len(s) {
				pt.EMA[p] = s[i]
			}
		}
		out[i] = pt
	}
	return out
}

func ptr(f float64) *float64 {
	// Round lightly for stable JSON / tests (8 decimal places is enough for prices/RSI).
	v := math.Round(f*1e8) / 1e8
	return &v
}

// NormalizeEMAPeriods dedupes, sorts ascending, and clamps invalid periods.
func NormalizeEMAPeriods(periods []int) []int {
	if len(periods) == 0 {
		return []int{DefaultEMAFast, DefaultEMASlow}
	}
	seen := map[int]struct{}{}
	var out []int
	for _, p := range periods {
		if p < MinIndicatorPeriod || p > MaxIndicatorPeriod {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []int{DefaultEMAFast, DefaultEMASlow}
	}
	// insertion sort small
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1] > out[j] {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	return out
}
