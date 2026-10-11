package market

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ScanForensics ranks top-volume coins whose current tape looks like
// the setup before that coin's own important past moves.
func (s *Service) ScanForensics(ctx context.Context, exchange, quote, lookback, interval, direction string, minReturnPct float64, symbolLimit int, window, during, fields, weights, minCoverage, horizons string) (*domain.ForensicsScanReport, error) {
	if lookback == "" {
		lookback = domain.DefaultForensicsScanLookback
	}
	if window == "" {
		window = domain.DefaultAroundWindow
	}
	if quote == "" {
		quote = domain.DefaultForensicsScanQuote
	}
	symbolLimit = domain.ClampForensicsScanLimit(symbolLimit)
	listEx := exchange
	if listEx == "" || listEx == "all" {
		listEx = string(domain.ExchangeBinance)
	}
	spot, err := s.ListSpotMarkets(ctx, listEx, domain.SpotListQuery{
		QuoteAsset: quote,
		SortBy:     domain.SpotSortQuoteVolume,
		Order:      domain.SortDesc,
		Limit:      symbolLimit,
		Status:     "TRADING",
	})
	if err != nil && (exchange == "" || exchange == "all") {
		spot, err = s.ListSpotMarkets(ctx, string(domain.ExchangeBybit), domain.SpotListQuery{
			QuoteAsset: quote,
			SortBy:     domain.SpotSortQuoteVolume,
			Order:      domain.SortDesc,
			Limit:      symbolLimit,
			Status:     "TRADING",
		})
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	out := &domain.ForensicsScanReport{
		Exchange:    exchange,
		Quote:       quote,
		Lookback:    lookback,
		Window:      window,
		Interval:    interval,
		Direction:   direction,
		SymbolLimit: symbolLimit,
		AsOf:        now,
		Hits:        []domain.ForensicsScanHit{},
		Note:        domain.ForensicsScanDisclaimer,
	}
	if exchange == "" {
		out.Exchange = "all"
	}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		sem  = make(chan struct{}, domain.ForensicsScanWorkers())
		hits []domain.ForensicsScanHit
	)
	for _, m := range spot.Items {
		if m.Symbol == "" {
			continue
		}
		out.Scanned++
		sym := m.Symbol
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			rep, err := s.GetAroundSimilar(ctx, exchange, sym, lookback, interval, direction, minReturnPct, domain.DefaultAroundSimilarLimit, window, during, fields, weights, minCoverage, horizons, time.Time{}, time.Time{})
			if err != nil || rep == nil {
				return
			}
			hit, ok := domain.ForensicsScanHitFromSimilar(rep)
			if !ok {
				mu.Lock()
				out.Compared++
				mu.Unlock()
				return
			}
			mu.Lock()
			out.Compared++
			hits = append(hits, hit)
			if out.Fields == nil {
				out.Fields = append([]string(nil), rep.Fields...)
				out.MinCoverage = rep.MinCoverage
				out.Horizons = append([]string(nil), rep.Horizons...)
				if out.Interval == "" {
					out.Interval = rep.Interval
				}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	out.Hits = hits
	domain.FinishForensicsScan(out)
	return out, nil
}
