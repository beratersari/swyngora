package market

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// attachLiquidationLevelsReview rebuilds past maps from stored OI / last
// price and scores them against later venue-local liquidations.
func (s *Service) attachLiquidationLevelsReview(ctx context.Context, rep *domain.LiquidationLevelsReport, ex, symbol string, spec domain.HuntHeatmapSpec, now time.Time) {
	if s == nil || rep == nil || symbol == "" || symbol == "all" {
		return
	}
	from := now.Add(-spec.Window)
	want := s.futuresWant(ex)
	venues := make([]domain.HuntHeatmapVenueSeries, 0, len(want))
	for _, v := range want {
		ser := s.huntHeatmapVenueSeries(ctx, v, symbol, spec, from, now)
		venues = append(venues, ser)
	}
	rev := domain.ReviewLiquidationLevels(domain.LiquidationLevelsReviewInput{
		Now: now, Step: spec.Step, Venues: venues,
	})
	rep.Review = &rev
}
