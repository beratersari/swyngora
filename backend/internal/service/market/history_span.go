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

type recordedSpanBatch interface {
	RecordingSpans(ctx context.Context, symbols []string, now time.Time) (map[string][]domain.RecordedSeries, error)
}

type flowSpanBatch interface {
	EarliestFlowSpans(ctx context.Context, symbols []string) ([]domain.StoredSince, error)
}

// recordingBySymbol gathers stored spans for the derivatives board.
// A missing store leaves that kind off the row. A lookup error is returned.
func (s *Service) recordingBySymbol(ctx context.Context, symbols []string, now time.Time) (map[string][]domain.RecordedSeries, error) {
	symbols = domain.SpanQuerySymbols(symbols)
	out := map[string][]domain.RecordedSeries{}
	if len(symbols) == 0 {
		return out, nil
	}
	add := func(by map[string][]domain.RecordedSeries) {
		for sym, rows := range by {
			out[sym] = append(out[sym], rows...)
		}
	}
	if r, ok := s.futHist.(recordedSpanBatch); ok && s.futHist != nil {
		part, err := r.RecordingSpans(ctx, symbols, now)
		if err != nil {
			return nil, err
		}
		add(part)
	}
	if r, ok := s.bookHist.(recordedSpanBatch); ok && s.bookHist != nil {
		part, err := r.RecordingSpans(ctx, symbols, now)
		if err != nil {
			return nil, err
		}
		add(part)
	}
	if f, ok := s.flowHist.(flowSpanBatch); ok && s.flowHist != nil {
		rows, err := f.EarliestFlowSpans(ctx, symbols)
		if err != nil {
			return nil, err
		}
		by := map[string][]domain.RecordedSeries{}
		for _, row := range rows {
			logical := domain.NormalizeLiquidationSymbol(row.Symbol)
			if logical == "" || row.Since.IsZero() {
				continue
			}
			by[logical] = append(by[logical], domain.RecordedSeries{
				Kind: domain.RecordedFlow, Exchange: row.Exchange, Since: row.Since.UTC(),
			})
		}
		add(by)
	}
	return out, nil
}
