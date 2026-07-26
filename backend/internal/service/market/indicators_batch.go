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
// Fetches only warm-up history (not a full output series) per symbol.
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
	emaPeriods, err = domain.ValidateAndNormalizeEMAPeriods(emaPeriods)
	if err != nil {
		return nil, err
	}

	// Dedupe + cap (do not iterate unbounded junk arrays into work).
	const maxBatch = 50
	const maxSymbolScan = 500
	seen := map[string]struct{}{}
	var list []string
	scan := symbols
	if len(scan) > maxSymbolScan {
		scan = scan[:maxSymbolScan]
	}
	for _, raw := range scan {
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

	// Candle limit for warm-up only (latest values) — not a full series response.
	warm := rsiPeriod
	for _, p := range emaPeriods {
		if p > warm {
			warm = p
		}
	}
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

			// Fetch warm-up candles only; compute series in-domain (avoid double warm-up).
			candles, err := s.GetCandles(ctx, string(ex), sym, interval, fetchLimit, nil, nil)
			if err != nil {
				out[i] = IndicatorSnapshot{Symbol: sym, Error: err.Error()}
				return
			}
			if len(candles) == 0 {
				out[i] = IndicatorSnapshot{Symbol: sym, Error: "no candles"}
				return
			}
			points, err := domain.BuildIndicatorSeries(candles, rsiPeriod, emaPeriods)
			if err != nil {
				out[i] = IndicatorSnapshot{Symbol: sym, Error: err.Error()}
				return
			}
			snap := IndicatorSnapshot{
				Symbol: sym,
				EMA:    map[int]*float64{},
			}
			for j := len(points) - 1; j >= 0; j-- {
				if snap.RSI == nil && points[j].RSI != nil {
					v := *points[j].RSI
					snap.RSI = &v
				}
				for _, p := range emaPeriods {
					if snap.EMA[p] == nil && points[j].EMA != nil && points[j].EMA[p] != nil {
						v := *points[j].EMA[p]
						snap.EMA[p] = &v
					}
				}
				if snap.RSI != nil {
					all := true
					for _, p := range emaPeriods {
						if snap.EMA[p] == nil {
							all = false
							break
						}
					}
					if all {
						break
					}
				}
			}
			out[i] = snap
		}(i, sym)
	}
	wg.Wait()
	// Always return per-symbol results; cancelled work is marked on items.
	// Do not fail the whole batch solely because ctx ended (partial results are useful).
	return out, nil
}
