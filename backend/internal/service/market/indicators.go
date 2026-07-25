package market

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetIndicators fetches candles and computes RSI + EMA series for a symbol.
// limit is the number of output bars desired; extra history is fetched for warm-up.
func (s *Service) GetIndicators(ctx context.Context, exchange, symbol, interval string, limit, rsiPeriod int, emaPeriods []int) (*domain.IndicatorSeries, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if !domain.IsValidIntervalFor(ex, interval) {
		return nil, fmt.Errorf("%w: interval must be one of %v for %s", domain.ErrInvalidArgument, domain.SupportedIntervalsFor(ex), ex)
	}

	if rsiPeriod == 0 {
		rsiPeriod = domain.DefaultRSIPeriod
	}
	if rsiPeriod < domain.MinIndicatorPeriod || rsiPeriod > domain.MaxIndicatorPeriod {
		return nil, fmt.Errorf("%w: rsiPeriod must be between %d and %d", domain.ErrInvalidArgument, domain.MinIndicatorPeriod, domain.MaxIndicatorPeriod)
	}
	emaPeriods = domain.NormalizeEMAPeriods(emaPeriods)

	if limit < 0 {
		return nil, fmt.Errorf("%w: limit must be >= 0", domain.ErrInvalidArgument)
	}
	if limit == 0 {
		limit = 100
	}
	if limit > 1000 {
		return nil, fmt.Errorf("%w: limit must be <= 1000", domain.ErrInvalidArgument)
	}

	// Fetch enough bars for warm-up of the slowest indicator.
	warm := rsiPeriod
	for _, p := range emaPeriods {
		if p > warm {
			warm = p
		}
	}
	fetchLimit := limit + warm + 5
	if fetchLimit > 1000 {
		fetchLimit = 1000
	}

	candles, err := s.GetCandles(ctx, string(ex), symbol, interval, fetchLimit, nil, nil)
	if err != nil {
		return nil, err
	}
	if len(candles) == 0 {
		return nil, fmt.Errorf("%w: no candles for indicators", domain.ErrNotFound)
	}

	points := domain.BuildIndicatorSeries(candles, rsiPeriod, emaPeriods)
	// Trim to last `limit` points for the response.
	if len(points) > limit {
		points = points[len(points)-limit:]
	}

	series := &domain.IndicatorSeries{
		Exchange:   ex,
		Symbol:     symbol,
		Interval:   domain.CandleInterval(interval),
		RSIPeriod:  rsiPeriod,
		EMAPeriods: emaPeriods,
		Points:     points,
		LatestEMA:  map[int]*float64{},
	}
	// Latest defined values from the end.
	for i := len(points) - 1; i >= 0; i-- {
		if series.LatestRSI == nil && points[i].RSI != nil {
			v := *points[i].RSI
			series.LatestRSI = &v
		}
		for _, p := range emaPeriods {
			if series.LatestEMA[p] == nil && points[i].EMA != nil && points[i].EMA[p] != nil {
				v := *points[i].EMA[p]
				series.LatestEMA[p] = &v
			}
		}
		if series.LatestRSI != nil {
			all := true
			for _, p := range emaPeriods {
				if series.LatestEMA[p] == nil {
					all = false
					break
				}
			}
			if all {
				break
			}
		}
	}
	return series, nil
}

// ParseEMAPeriodsCSV parses "12,26" into ints.
func ParseEMAPeriodsCSV(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []int
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}
