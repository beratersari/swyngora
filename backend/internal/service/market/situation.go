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
	if s == nil {
		return nil, ctx.Err()
	}
	v, err, _ := s.situationSF.Do(symbol, func() (any, error) {
		return s.loadCoinSituation(ctx, symbol)
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.(*domain.CoinSituation)
	return got, nil
}

func (s *Service) loadCoinSituation(ctx context.Context, symbol string) (*domain.CoinSituation, error) {
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
	out := domain.BuildCoinSituation(symbol, now, cvd, oi, levels, funding, longs, prints)
	s.attachSituationHeld(ctx, symbol, now, out, cvd, funding, longs, prints)
	return out, nil
}

// CloserPocket is the nearer estimated liquidation level used by the paper-fill mark.
// exchange is always the combined Binance, Bybit, and OKX map.
func (s *Service) CloserPocket(ctx context.Context, symbol string) (domain.CloserPocket, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return domain.CloserPocket{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.CloserPocket{}, err
	}
	levels, err := s.GetLiquidationLevels(ctx, "all", symbol, "24h")
	if err != nil {
		return domain.CloserPocket{}, err
	}
	pocket, _ := domain.CloserPocketFromLevels(levels)
	return pocket, nil
}

// situationPrintInput loads the 24h stored print sum plus the day and 4-hour clocks.
// Each window keeps its own archive flag. A failed clock stays blank and does not
// blank the others. ArchiveOK on the 24h clause stays false when that sum cannot be read.
func (s *Service) situationPrintInput(ctx context.Context, symbol string, now time.Time) domain.SituationPrintInput {
	in := domain.SituationPrintInput{}
	if s != nil && s.liq != nil {
		in.Coverage = s.liq.CoverageSnapshot(now)
	}
	day, block := domain.PrintAnchorStarts(now)
	froms := []time.Time{now.Add(-24 * time.Hour), day, block}
	sides := make([][]domain.LiquidationPrintSideRow, len(froms))
	ok := make([]bool, len(froms))
	var wg sync.WaitGroup
	for i, from := range froms {
		wg.Add(1)
		go func(i int, from time.Time) {
			defer wg.Done()
			sides[i], ok[i] = s.summedPrintSides(ctx, symbol, from, now)
		}(i, from)
	}
	wg.Wait()
	in.ArchiveOK = ok[0]
	in.Sides = sides[0]
	in.Anchors = []domain.PrintAnchorWindow{
		{Anchor: domain.PrintAnchorDay, From: day, ArchiveOK: ok[1], Sides: sides[1]},
		{Anchor: domain.PrintAnchor4h, From: block, ArchiveOK: ok[2], Sides: sides[2]},
	}
	if ok[0] {
		from24 := now.Add(-24 * time.Hour)
		var latest map[domain.Exchange]time.Time
		if arch := s.liquidationArchive(); arch != nil {
			if scan, err := arch.SumLiquidationPrintSides(ctx, symbol, from24, now, domain.FuturesVenues()); err == nil {
				latest = scan.Latest
			}
		}
		var tail []domain.LiquidationEvent
		if s.liq != nil {
			tail = s.liq.Events("all", symbol, from24)
		}
		in.Minutes = s.situationPrintMinutes(ctx, symbol, now, latest, tail)
	}
	return in
}

// summedPrintSides reads one symbol's stored long and short notionals in [from, to)
// and adds book prints newer than each venue's newest stored row.
func (s *Service) summedPrintSides(ctx context.Context, symbol string, from, to time.Time) ([]domain.LiquidationPrintSideRow, bool) {
	arch := s.liquidationArchive()
	if arch == nil {
		return nil, false
	}
	venues := domain.FuturesVenues()
	scan, err := arch.SumLiquidationPrintSides(ctx, symbol, from, to, venues)
	if err != nil {
		slog.Warn("situation print sides", "symbol", symbol, "from", from, "err", err)
		return nil, false
	}
	var tail []domain.LiquidationEvent
	if s != nil && s.liq != nil {
		tail = s.liq.Events("all", symbol, from)
	}
	return domain.MergeLiquidationPrintSides(scan.Rows, scan.Latest, tail, from, to), true
}
