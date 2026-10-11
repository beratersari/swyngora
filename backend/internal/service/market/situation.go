package market

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCoinSituation is the coin-page read: price versus futures CVD, 1h open
// interest, the closer estimated liquidation pocket, measured 24h force-order
// prints, OI-weighted funding, top versus crowd, and whale versus retail.
// A clause stays blank when a venue that publishes it does not cover it.
// One failed upstream leaves that clause blank and still returns the others.
func (s *Service) GetCoinSituation(ctx context.Context, symbol string) (*domain.CoinSituation, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var (
		cvd     *domain.CVDReport
		oi      *domain.OpenInterestSnapshot
		levels  *domain.LiquidationLevelsReport
		funding *domain.FundingSnapshot
		longs   *domain.LongShortSnapshot
		prints  domain.SituationPrintInput
		wg      sync.WaitGroup
	)
	wg.Add(6)
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
	go func() {
		defer wg.Done()
		prints = s.situationPrintInput(ctx, symbol, now)
	}()
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return domain.BuildCoinSituation(symbol, now, cvd, oi, levels, funding, longs, prints), nil
}

// situationPrintInput loads the 24h stored print sum plus newer book prints.
// ArchiveOK stays false when that sum cannot be read.
func (s *Service) situationPrintInput(ctx context.Context, symbol string, now time.Time) domain.SituationPrintInput {
	in := domain.SituationPrintInput{}
	if s != nil && s.liq != nil {
		in.Coverage = s.liq.CoverageSnapshot(now)
	}
	arch := s.liquidationArchive()
	if arch == nil {
		return in
	}
	from := now.Add(-24 * time.Hour)
	venues := domain.FuturesVenues()
	scan, err := arch.SumLiquidationPrintSides(ctx, symbol, from, now, venues)
	if err != nil {
		slog.Warn("situation print sides", "symbol", symbol, "err", err)
		return in
	}
	in.ArchiveOK = true
	var tail []domain.LiquidationEvent
	if s.liq != nil {
		tail = s.liq.Events("all", symbol, from)
	}
	in.Sides = domain.MergeLiquidationPrintSides(scan.Rows, scan.Latest, tail, from, now)
	return in
}
