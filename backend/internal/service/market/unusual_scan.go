package market

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	unusualScanCacheTTL = 45 * time.Second
	unusualScanWorkers  = 4
)

// ScanUnusual ranks the liquid USDT perpetuals whose stored funding,
// open-interest change, liquidation hour, CVD day, or spread is high or low
// versus that coin's own samples. Typical and unknown readings are omitted.
func (s *Service) ScanUnusual(ctx context.Context, exchange, metric string, limit int) (*domain.UnusualScan, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	metric, err = domain.ParseUnusualScanMetric(metric)
	if err != nil {
		return nil, err
	}
	limit = domain.ParseUnusualScanLimit(limit)
	key := ex + "|" + metric + "|" + strconv.Itoa(limit)
	if c := s.unusualScanCache(); c != nil {
		if hit, ok := c.Get(key); ok && hit != nil {
			cp := *hit
			cp.Hits = append([]domain.UnusualScanHit(nil), hit.Hits...)
			return &cp, nil
		}
	}
	symbols, err := s.LiquidPerpSymbols(ctx, limit)
	if err != nil && len(symbols) == 0 {
		return nil, err
	}
	now := time.Now().UTC()
	slots := make([]unusualScanSlot, len(symbols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, unusualScanWorkers)
	for i, sym := range symbols {
		wg.Add(1)
		go func(i int, sym string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				slots[i].skip = true
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			rep, err := s.unusualReport(ctx, ex, sym, now)
			if err != nil {
				slots[i].skip = true
				return
			}
			slots[i].rep = rep
		}(i, sym)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	reports := make([]domain.UnusualReport, 0, len(slots))
	skipped := 0
	for _, slot := range slots {
		if slot.skip || slot.rep == nil {
			skipped++
			continue
		}
		reports = append(reports, *slot.rep)
	}
	out := domain.BuildUnusualScan(ex, metric, reports, skipped, now)
	if c := s.unusualScanCache(); c != nil && out != nil {
		cp := *out
		cp.Hits = append([]domain.UnusualScanHit(nil), out.Hits...)
		c.Set(key, &cp)
	}
	return out, nil
}

type unusualScanSlot struct {
	rep  *domain.UnusualReport
	skip bool
}

func (s *Service) unusualScanCache() *cache.TTL[*domain.UnusualScan] {
	if s == nil {
		return nil
	}
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if s.unusualScan == nil {
		s.unusualScan = cache.New[*domain.UnusualScan](unusualScanCacheTTL)
	}
	return s.unusualScan
}
