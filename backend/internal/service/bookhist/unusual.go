package bookhist

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// The market service type-asserts this service, not the sqlite store.
var _ domain.SpreadUnusualReader = (*Service)(nil)

// HourlySpreadPct delegates to the store aggregate. A store without that
// query yields no samples.
func (s *Service) HourlySpreadPct(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	if s == nil || s.Store == nil {
		return nil, nil
	}
	r, ok := s.Store.(domain.SpreadUnusualReader)
	if !ok || r == nil {
		return nil, nil
	}
	return r.HourlySpreadPct(ctx, exchange, symbol, from)
}
