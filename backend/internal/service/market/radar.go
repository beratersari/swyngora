package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// RadarCoin compares this coin's current last-window tape to its own past
// pre-move setups. Used by Setup Radar alerts (defaults: 3d lookback).
func (s *Service) RadarCoin(ctx context.Context, exchange, symbol string) (*domain.AroundSimilarReport, error) {
	return s.GetAroundSimilar(ctx, exchange, symbol, domain.DefaultForensicsScanLookback, "", "", 0, 0, "", "", "", "", "", "", time.Time{})
}

// RadarScan ranks top-volume coins whose current tape looks like past setups.
func (s *Service) RadarScan(ctx context.Context, exchange string) (*domain.ForensicsScanReport, error) {
	return s.ScanForensics(ctx, exchange, "", domain.DefaultForensicsScanLookback, "", "", 0, 0, "", "", "", "", "", "")
}
