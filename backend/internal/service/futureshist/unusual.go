package futureshist

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// The market service type-asserts this service, not the sqlite store.
var _ domain.FuturesUnusualReader = (*Service)(nil)

func (s *Service) unusualStore() (domain.FuturesUnusualReader, bool) {
	if s == nil || s.Store == nil {
		return nil, false
	}
	r, ok := s.Store.(domain.FuturesUnusualReader)
	return r, ok && r != nil
}

// HourlyOpenInterest delegates to the store aggregate. A store without that
// query yields no samples.
func (s *Service) HourlyOpenInterest(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	r, ok := s.unusualStore()
	if !ok {
		return nil, nil
	}
	return r.HourlyOpenInterest(ctx, exchange, symbol, from)
}

// SettledFunding delegates to stored settlement rows.
func (s *Service) SettledFunding(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	r, ok := s.unusualStore()
	if !ok {
		return nil, nil
	}
	return r.SettledFunding(ctx, exchange, symbol, from)
}

// LatestPredictedFunding delegates to the newest predicted row.
func (s *Service) LatestPredictedFunding(ctx context.Context, exchange, symbol string) (domain.UnusualSample, bool, error) {
	r, ok := s.unusualStore()
	if !ok {
		return domain.UnusualSample{}, false, nil
	}
	return r.LatestPredictedFunding(ctx, exchange, symbol)
}

// HourlyLiquidationNotional delegates to the hourly notional aggregate.
func (s *Service) HourlyLiquidationNotional(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	r, ok := s.unusualStore()
	if !ok {
		return nil, nil
	}
	return r.HourlyLiquidationNotional(ctx, exchange, symbol, from)
}
