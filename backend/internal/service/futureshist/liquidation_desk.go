package futureshist

import (
	"context"
	"fmt"
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

// SumLiquidationPrintSides reads one symbol's stored long and short notionals.
// A store that cannot answer returns an error so the caller leaves the clause blank.
func (s *Service) SumLiquidationPrintSides(ctx context.Context, symbol string, from, to time.Time, exchanges []domain.Exchange) (domain.LiquidationPrintSideScan, error) {
	if s == nil || s.Store == nil {
		return domain.LiquidationPrintSideScan{}, fmt.Errorf("futures store is nil")
	}
	arch, ok := s.Store.(domain.LiquidationDeskArchive)
	if !ok {
		return domain.LiquidationPrintSideScan{}, fmt.Errorf("liquidation print sides are not stored")
	}
	return arch.SumLiquidationPrintSides(ctx, symbol, from, to, exchanges)
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
