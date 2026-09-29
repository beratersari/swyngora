package market

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetUnusual compares funding, open-interest change, liquidation notional,
// CVD, spread, and print size with this coin's own stored history.
// Each futures venue is scored on its own. A missing store leaves that
// metric unknown.
func (s *Service) GetUnusual(ctx context.Context, exchange, symbol string) (*domain.UnusualReport, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	sym, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	venues := domain.FuturesVenues()
	if ex != "all" {
		venues = []domain.Exchange{domain.Exchange(ex)}
	}
	var fut domain.FuturesUnusualReader
	if r, ok := s.futHist.(domain.FuturesUnusualReader); ok {
		fut = r
	}
	var cvd domain.CVDUnusualReader
	if r, ok := s.takerStore.(domain.CVDUnusualReader); ok {
		cvd = r
	}
	var spread domain.SpreadUnusualReader
	if r, ok := s.bookHist.(domain.SpreadUnusualReader); ok {
		spread = r
	}
	fromFut := now.Add(-domain.FuturesHistoryMaxLookback)
	fromBook := now.Add(-domain.UnusualSpreadLookback)
	out := &domain.UnusualReport{
		Symbol: sym, Exchange: ex, AsOf: now, Note: domain.UnusualNote,
	}
	for _, venue := range venues {
		block, err := s.unusualVenue(ctx, venue, sym, now, fromFut, fromBook, fut, cvd, spread)
		if err != nil {
			return nil, err
		}
		out.Venues = append(out.Venues, block)
	}
	s.noteFutures(sym)
	return out, nil
}

func (s *Service) unusualVenue(
	ctx context.Context,
	ex domain.Exchange,
	symbol string,
	now, fromFut, fromBook time.Time,
	fut domain.FuturesUnusualReader,
	cvd domain.CVDUnusualReader,
	spread domain.SpreadUnusualReader,
) (domain.UnusualVenue, error) {
	block := domain.UnusualVenue{Exchange: ex}
	settled, predicted, err := loadFunding(ctx, fut, string(ex), symbol, fromFut)
	if err != nil {
		return block, err
	}
	oi, err := loadSamples(ctx, fut, ex, symbol, fromFut, func(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
		if fut == nil {
			return nil, nil
		}
		return fut.HourlyOpenInterest(ctx, exchange, symbol, from)
	})
	if err != nil {
		return block, err
	}
	liq, err := loadSamples(ctx, fut, ex, symbol, fromFut, func(ctx context.Context, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
		if fut == nil {
			return nil, nil
		}
		return fut.HourlyLiquidationNotional(ctx, exchange, symbol, from)
	})
	if err != nil {
		return block, err
	}
	days, err := loadCVD(ctx, cvd, string(ex), symbol, fromFut)
	if err != nil {
		return block, err
	}
	spreadReading, err := s.spreadReading(ctx, spread, ex, symbol, fromBook)
	if err != nil {
		return block, err
	}
	block.Readings = []domain.UnusualReading{
		domain.UnusualFunding(now, settled, predicted),
		domain.UnusualOpenInterest(oi),
		domain.UnusualLiquidations(now, liq),
		domain.UnusualCVD(now, days),
		spreadReading,
		domain.UnusualPrintSize(),
	}
	return block, nil
}

func loadFunding(ctx context.Context, fut domain.FuturesUnusualReader, exchange, symbol string, from time.Time) ([]domain.UnusualSample, *domain.UnusualSample, error) {
	if fut == nil {
		return nil, nil, nil
	}
	settled, err := fut.SettledFunding(ctx, exchange, symbol, from)
	if err != nil {
		return nil, nil, fmt.Errorf("unusual funding %s %s: %w", exchange, symbol, err)
	}
	pred, ok, err := fut.LatestPredictedFunding(ctx, exchange, symbol)
	if err != nil {
		return nil, nil, fmt.Errorf("unusual predicted funding %s %s: %w", exchange, symbol, err)
	}
	if !ok {
		return settled, nil, nil
	}
	return settled, &pred, nil
}

func loadSamples(
	ctx context.Context,
	fut domain.FuturesUnusualReader,
	ex domain.Exchange,
	symbol string,
	from time.Time,
	fn func(context.Context, string, string, time.Time) ([]domain.UnusualSample, error),
) ([]domain.UnusualSample, error) {
	if fut == nil {
		return nil, nil
	}
	got, err := fn(ctx, string(ex), symbol, from)
	if err != nil {
		return nil, fmt.Errorf("unusual %s %s: %w", ex, symbol, err)
	}
	return got, nil
}

func loadCVD(ctx context.Context, cvd domain.CVDUnusualReader, exchange, symbol string, from time.Time) ([]domain.UnusualSample, error) {
	if cvd == nil {
		return nil, nil
	}
	got, err := cvd.DailyCVD(ctx, exchange, symbol, from)
	if err != nil {
		return nil, fmt.Errorf("unusual cvd %s %s: %w", exchange, symbol, err)
	}
	return got, nil
}

func (s *Service) spreadReading(ctx context.Context, spread domain.SpreadUnusualReader, ex domain.Exchange, symbol string, from time.Time) (domain.UnusualReading, error) {
	if ex == domain.ExchangeOKX {
		return domain.UnusualNoBook(), nil
	}
	if spread == nil {
		return domain.UnusualSpread(nil), nil
	}
	if s != nil {
		s.noteBook(ex, symbol)
	}
	pts, err := spread.HourlySpreadPct(ctx, string(ex), symbol, from)
	if err != nil {
		return domain.UnusualReading{}, fmt.Errorf("unusual spread %s %s: %w", ex, symbol, err)
	}
	return domain.UnusualSpread(pts), nil
}
