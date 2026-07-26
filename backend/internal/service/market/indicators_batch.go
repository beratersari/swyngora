package market

import (
	"context"
	"fmt"
	"sync"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Process-wide cap on concurrent upstream candle fetches from all batch requests.
// Prevents ingress RPS × per-request fan-out from storming exchange APIs.
var batchUpstreamSem = make(chan struct{}, 24)

// IndicatorSnapshot is the latest RSI/EMA values for one symbol (list enrichment).
type IndicatorSnapshot struct {
	Symbol string
	RSI    *float64
	EMA    map[int]*float64
	Error  string
}

// GetIndicatorsBatch computes latest RSI/EMA for up to maxBatchSymbols on one exchange.
// Used to fill table columns without N+1 browser requests.
func (s *Service) GetIndicatorsBatch(
	ctx context.Context,
	exchange, interval string,
	symbols []string,
	rsiPeriod int,
	emaPeriods []int,
) ([]IndicatorSnapshot, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
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

	// Dedupe + cap
	const maxBatch = 50
	seen := map[string]struct{}{}
	var list []string
	for _, raw := range symbols {
		sym := normalizeSymbolForExchange(ex, raw)
		if sym == "" {
			continue
		}
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}
		list = append(list, sym)
		if len(list) >= maxBatch {
			break
		}
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("%w: symbols required", domain.ErrInvalidArgument)
	}

	// Candle limit for warm-up only (not full series).
	warm := rsiPeriod
	for _, p := range emaPeriods {
		if p > warm {
			warm = p
		}
	}
	// Enough history for RSI(14)/EMA(26) even with a few bad closes.
	fetchLimit := warm + 20
	if fetchLimit < 60 {
		fetchLimit = 60
	}
	if fetchLimit > 200 {
		fetchLimit = 200
	}

	out := make([]IndicatorSnapshot, len(list))
	var wg sync.WaitGroup
	// Per-request concurrency (8) × process-wide semaphore (24).
	reqSem := make(chan struct{}, 8)
	for i, sym := range list {
		wg.Add(1)
		go func(i int, sym string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				out[i] = IndicatorSnapshot{Symbol: sym, Error: ctx.Err().Error()}
				return
			case reqSem <- struct{}{}:
			}
			defer func() { <-reqSem }()

			select {
			case <-ctx.Done():
				out[i] = IndicatorSnapshot{Symbol: sym, Error: ctx.Err().Error()}
				return
			case batchUpstreamSem <- struct{}{}:
			}
			defer func() { <-batchUpstreamSem }()

			if ctx.Err() != nil {
				out[i] = IndicatorSnapshot{Symbol: sym, Error: ctx.Err().Error()}
				return
			}

			ser, err := s.GetIndicators(ctx, string(ex), sym, interval, fetchLimit, rsiPeriod, emaPeriods)
			if err != nil {
				out[i] = IndicatorSnapshot{Symbol: sym, Error: err.Error()}
				return
			}
			snap := IndicatorSnapshot{
				Symbol: sym,
				RSI:    ser.LatestRSI,
				EMA:    map[int]*float64{},
			}
			for k, v := range ser.LatestEMA {
				if v != nil {
					vv := *v
					snap.EMA[k] = &vv
				}
			}
			out[i] = snap
		}(i, sym)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	return out, nil
}
