package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	unusualScanMinLimit = 5

	// UnusualScanNote is the English note on the scan payload. The product
	// page builds its own sentence from localized copy.
	UnusualScanNote = "Liquid USDT perpetuals, seeds first, capped at 36. Each row is one venue's reading that is high or low versus that coin's own stored samples. Typical readings and readings with too few samples are left off. Venues are not merged. Print size is not scanned. Informational only — not financial advice."
)

// UnusualScanHit is one standout reading. Distance is how far the percentile sits from 50.
type UnusualScanHit struct {
	Symbol     string  `json:"symbol"`
	Base       string  `json:"base"`
	Exchange   string  `json:"exchange"`
	Metric     string  `json:"metric"`
	Band       string  `json:"band"`
	Percentile float64 `json:"percentile"`
	Distance   float64 `json:"distance"`
	Current    float64 `json:"current"`
	Median     float64 `json:"median"`
	Samples    int     `json:"samples"`
	SpanHours  float64 `json:"spanHours"`
	Unit       string  `json:"unit,omitempty"`
	Label      string  `json:"label,omitempty"`
	Summary    string  `json:"summary"`
}

// UnusualScan ranks standout readings across the liquid perpetuals.
// Quiet is how many scanned coins had no standout row.
type UnusualScan struct {
	Exchange string           `json:"exchange"`
	Metric   string           `json:"metric"`
	Universe int              `json:"universe"`
	Scanned  int              `json:"scanned"`
	Skipped  int              `json:"skipped"`
	Quiet    int              `json:"quiet"`
	AsOf     time.Time        `json:"asOf"`
	Hits     []UnusualScanHit `json:"hits"`
	Note     string           `json:"note"`
}

// ParseUnusualScanMetric accepts all, funding, open_interest, liquidations, cvd, or spread.
// Print size has no stored series, so it is not a scan metric.
func ParseUnusualScanMetric(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "", "all":
		return "all", nil
	case UnusualMetricFunding, UnusualMetricOpenInterest, UnusualMetricLiquidations, UnusualMetricCVD, UnusualMetricSpread:
		return s, nil
	case UnusualMetricPrintSize:
		return "", fmt.Errorf("%w: print size has no stored history to scan", ErrInvalidArgument)
	default:
		return "", fmt.Errorf("%w: metric must be all, funding, open_interest, liquidations, cvd, or spread", ErrInvalidArgument)
	}
}

// ParseUnusualScanLimit clamps the liquid-list size. Zero uses the recorder cap.
func ParseUnusualScanLimit(n int) int {
	if n <= 0 {
		return LiquidRecorderLimit
	}
	if n < unusualScanMinLimit {
		return unusualScanMinLimit
	}
	if n > LiquidRecorderLimit {
		return LiquidRecorderLimit
	}
	return n
}

// BuildUnusualScan keeps high and extreme readings and ranks the farthest from typical first.
// reports are coins that were scored. skipped is how many coins the store could not read.
func BuildUnusualScan(exchange, metric string, reports []UnusualReport, skipped int, now time.Time) *UnusualScan {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	ex := strings.ToLower(strings.TrimSpace(exchange))
	if ex == "" {
		ex = "all"
	}
	metric = strings.ToLower(strings.TrimSpace(metric))
	if metric == "" {
		metric = "all"
	}
	hits := make([]UnusualScanHit, 0)
	quiet := 0
	for _, rep := range reports {
		symbol := NormalizeLiquidationSymbol(rep.Symbol)
		if symbol == "" {
			continue
		}
		before := len(hits)
		for _, venue := range rep.Venues {
			for _, reading := range venue.Readings {
				if !unusualStandout(reading.Band) {
					continue
				}
				if metric != "all" && reading.Metric != metric {
					continue
				}
				base, _ := SplitBaseQuote(ExchangeBinance, symbol)
				hits = append(hits, UnusualScanHit{
					Symbol:     symbol,
					Base:       base,
					Exchange:   string(venue.Exchange),
					Metric:     reading.Metric,
					Band:       reading.Band,
					Percentile: reading.Percentile,
					Distance:   math.Abs(reading.Percentile - 50),
					Current:    reading.Current,
					Median:     reading.Median,
					Samples:    reading.Samples,
					SpanHours:  reading.SpanHours,
					Unit:       reading.Unit,
					Label:      reading.Label,
					Summary:    reading.Summary,
				})
			}
		}
		if len(hits) == before {
			quiet++
		}
	}
	sortUnusualHits(hits)
	return &UnusualScan{
		Exchange: ex,
		Metric:   metric,
		Universe: len(reports) + skipped,
		Scanned:  len(reports),
		Skipped:  skipped,
		Quiet:    quiet,
		AsOf:     now,
		Hits:     hits,
		Note:     UnusualScanNote,
	}
}

func unusualStandout(band string) bool {
	switch band {
	case UnusualBandExtremeHigh, UnusualBandHigh, UnusualBandLow, UnusualBandExtremeLow:
		return true
	default:
		return false
	}
}

func sortUnusualHits(hits []UnusualScanHit) {
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0; j-- {
			if unusualHitBefore(hits[j], hits[j-1]) {
				hits[j], hits[j-1] = hits[j-1], hits[j]
				continue
			}
			break
		}
	}
}

func unusualHitBefore(a, b UnusualScanHit) bool {
	if a.Distance != b.Distance {
		return a.Distance > b.Distance
	}
	if a.Symbol != b.Symbol {
		return a.Symbol < b.Symbol
	}
	if a.Exchange != b.Exchange {
		return a.Exchange < b.Exchange
	}
	return unusualMetricOrder(a.Metric) < unusualMetricOrder(b.Metric)
}

func unusualMetricOrder(metric string) int {
	switch metric {
	case UnusualMetricFunding:
		return 0
	case UnusualMetricOpenInterest:
		return 1
	case UnusualMetricLiquidations:
		return 2
	case UnusualMetricCVD:
		return 3
	case UnusualMetricSpread:
		return 4
	default:
		return 5
	}
}
