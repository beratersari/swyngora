package market

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type recordingSince interface {
	RecordingSince(ctx context.Context, symbol string, now time.Time) ([]domain.RecordedSeries, error)
}

type flowEarliest interface {
	EarliestFlow(ctx context.Context, exchange domain.Exchange, symbol string) (time.Time, bool, error)
}

// HistorySpan reports how long stored futures, book, and flow samples cover
// for one coin. Venues with no rows are omitted.
func (s *Service) HistorySpan(ctx context.Context, symbol string) (*domain.HistorySpan, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var series []domain.RecordedSeries
	if r, ok := s.futHist.(recordingSince); ok && s.futHist != nil {
		part, err := r.RecordingSince(ctx, symbol, now)
		if err != nil {
			return nil, err
		}
		series = append(series, part...)
	}
	if r, ok := s.bookHist.(recordingSince); ok && s.bookHist != nil {
		part, err := r.RecordingSince(ctx, symbol, now)
		if err != nil {
			return nil, err
		}
		series = append(series, part...)
	}
	if f, ok := s.flowHist.(flowEarliest); ok && s.flowHist != nil {
		for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit, domain.ExchangeOKX} {
			since, found, err := f.EarliestFlow(ctx, ex, symbol)
			if err != nil {
				return nil, err
			}
			if !found || since.IsZero() {
				continue
			}
			series = append(series, domain.RecordedSeries{
				Kind:     domain.RecordedFlow,
				Exchange: ex,
				Since:    since.UTC(),
			})
		}
	}
	out := domain.BuildHistorySpan(symbol, now, series)
	return &out, nil
}
