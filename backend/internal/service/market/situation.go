package market

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCoinSituation is the short coin-page read: price versus futures CVD,
// whether 1h open interest is building, and which liquidation side is closer.
// A clause stays blank when Binance, Bybit, or OKX does not cover it.
// One failed upstream leaves that clause blank and still returns the others.
func (s *Service) GetCoinSituation(ctx context.Context, symbol string) (*domain.CoinSituation, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var (
		cvd    *domain.CVDReport
		oi     *domain.OpenInterestSnapshot
		levels *domain.LiquidationLevelsReport
		wg     sync.WaitGroup
	)
	wg.Add(3)
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
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return domain.BuildCoinSituation(symbol, time.Now().UTC(), cvd, oi, levels), nil
}
