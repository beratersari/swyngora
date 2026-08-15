package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const fxCacheTTL = 15 * time.Minute

const fxCacheKey = "usd"

// FxSource loads spot fiat rates quoted as units per 1 USD.
type FxSource interface {
	LatestUSD(ctx context.Context) (map[string]float64, time.Time, error)
}

// WithFx attaches a spot FX source for display conversion.
func (s *Service) WithFx(src FxSource) *Service {
	if s != nil {
		s.fx = src
		s.fxCache = cache.New[*domain.FxRates](fxCacheTTL)
	}
	return s
}

// GetFxRates returns USD-based spot rates (USDT treated as USD). Cached 15m.
func (s *Service) GetFxRates(ctx context.Context) (*domain.FxRates, error) {
	if s != nil && s.fxCache != nil {
		if hit, ok := s.fxCache.Get(fxCacheKey); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}
	if s == nil || s.fx == nil {
		return fallbackFx(false), nil
	}
	rates, asOf, err := s.fx.LatestUSD(ctx)
	if err != nil {
		if s.fxCache != nil {
			if stale, ok := s.fxCache.GetStale(fxCacheKey); ok && stale != nil {
				cp := *stale
				cp.Stale = true
				return &cp, nil
			}
		}
		fb := fallbackFx(true)
		fb.Note = "FX upstream unavailable; USD/USDT only"
		return fb, nil
	}
	if rates == nil {
		rates = map[string]float64{}
	}
	rates[domain.FxBaseUSD] = 1
	rates[domain.FxUSDT] = 1
	out := &domain.FxRates{
		Base:  domain.FxBaseUSD,
		AsOf:  asOf.UTC(),
		Rates: rates,
		Note:  "Spot ECB reference via Frankfurter. USDT treated as USD. Display only — not for settlement.",
	}
	if s.fxCache != nil {
		s.fxCache.Set(fxCacheKey, out)
	}
	cp := *out
	return &cp, nil
}

func fallbackFx(stale bool) *domain.FxRates {
	return &domain.FxRates{
		Base:  domain.FxBaseUSD,
		AsOf:  time.Now().UTC(),
		Rates: map[string]float64{domain.FxBaseUSD: 1, domain.FxUSDT: 1},
		Stale: stale,
		Note:  "USD/USDT identity only",
	}
}
