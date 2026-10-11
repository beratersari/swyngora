package futureshist

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SumLiquidationSymbols reads the print archive. A store without the desk
// query returns an empty scan so the live book can still answer.
func (s *Service) SumLiquidationSymbols(ctx context.Context, from, to time.Time, exchanges []domain.Exchange) (domain.LiquidationSymbolScan, error) {
	if s == nil || s.Store == nil {
		return domain.LiquidationSymbolScan{}, nil
	}
	arch, ok := s.Store.(domain.LiquidationDeskArchive)
	if !ok {
		return domain.LiquidationSymbolScan{}, nil
	}
	return arch.SumLiquidationSymbols(ctx, from, to, exchanges)
}

// BucketLiquidations reads all-coin time bars from the print archive.
func (s *Service) BucketLiquidations(ctx context.Context, from, to time.Time, step time.Duration, exchanges []domain.Exchange) (domain.LiquidationBucketScan, error) {
	if s == nil || s.Store == nil {
		return domain.LiquidationBucketScan{}, nil
	}
	arch, ok := s.Store.(domain.LiquidationDeskArchive)
	if !ok {
		return domain.LiquidationBucketScan{}, nil
	}
	return arch.BucketLiquidations(ctx, from, to, step, exchanges)
}
