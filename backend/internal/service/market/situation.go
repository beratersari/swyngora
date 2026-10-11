package market

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCoinSituation is the coin-page read: price versus futures CVD, 1h open
// interest, the closer liquidation side, OI-weighted funding, top versus crowd,
// and whale versus retail. A clause stays blank when a venue that publishes it
// does not cover it. One failed upstream leaves that clause blank and still
// returns the others.
func (s *Service) GetCoinSituation(ctx context.Context, symbol string) (*domain.CoinSituation, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var (
		cvd     *domain.CVDReport
		oi      *domain.OpenInterestSnapshot
		levels  *domain.LiquidationLevelsReport
		funding *domain.FundingSnapshot
		longs   *domain.LongShortSnapshot
		wg      sync.WaitGroup
	)
	wg.Add(5)
	go func() {
		defer wg.Done()
		cvd, _ = s.GetCVD(ctx, "all", symbol, "24h", "")
	}()
	go func() {
		defer wg.Done()
		oi, _ = s.GetOpenInterest(ctx, "all", symbol)
	}()
	go func() {
		defer wg.Done()
		levels, _ = s.GetLiquidationLevels(ctx, "all", symbol, "24h")
	}()
	go func() {
		defer wg.Done()
		funding, _ = s.GetFundingRate(ctx, "all", symbol, domain.DefaultFundingHistoryLimit)
	}()
	go func() {
		defer wg.Done()
		longs, _ = s.GetLongShortRatio(ctx, "all", symbol, domain.DefaultLongShortHistoryLimit)
	}()
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return domain.BuildCoinSituation(symbol, time.Now().UTC(), cvd, oi, levels, funding, longs), nil
}
