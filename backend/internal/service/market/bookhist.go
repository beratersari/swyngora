package market

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const bookHistoryNote = "Stored books are 1-minute samples of the live spot ladder (top grouped levels, ±2% liquidity and walls), not a full 24h tape. Compare uses the nearest sample to each time. Informational only — not financial advice."

// BookHistoryReader is the durable order-book archive (optional).
type BookHistoryReader interface {
	SnapshotAt(ctx context.Context, exchange, symbol string, at time.Time) (*domain.BookHistorySnapshot, error)
	List(ctx context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error)
	Compare(ctx context.Context, exchange, symbol string, from, to time.Time) (*domain.BookHistoryDiff, error)
	Note(exchange, symbol string)
}

// WithBookHistory attaches the durable order-book archive.
func (s *Service) WithBookHistory(r BookHistoryReader) *Service {
	if s != nil {
		s.bookHist = r
	}
	return s
}

func (s *Service) noteBook(ex domain.Exchange, symbol string) {
	if s != nil && s.bookHist != nil {
		s.bookHist.Note(string(ex), symbol)
	}
}

// GetBookHistory returns a stored book at a time, or a newest-first list.
func (s *Service) GetBookHistory(ctx context.Context, exchange, symbol string, at, from, to *time.Time, limit int) (*domain.BookHistoryReport, error) {
	if s == nil || s.bookHist == nil {
		return nil, fmt.Errorf("%w: order book history not configured", domain.ErrUpstream)
	}
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	s.noteBook(ex, symbol)
	out := &domain.BookHistoryReport{
		Symbol:   symbol,
		Exchange: string(ex),
		Note:     bookHistoryNote,
	}
	if at != nil && !at.IsZero() {
		out.At = at.UTC()
		got, err := s.bookHist.SnapshotAt(ctx, string(ex), symbol, at.UTC())
		if err != nil {
			return nil, err
		}
		out.Snapshot = got
		if got != nil {
			out.Summary = domain.ExplainBookHistory(*got)
		}
		return out, nil
	}
	q := domain.BookHistoryQuery{Exchange: string(ex), Symbol: symbol, Limit: limit}
	if from != nil {
		q.From = from.UTC()
	}
	if to != nil {
		q.To = to.UTC()
	}
	rows, err := s.bookHist.List(ctx, q)
	if err != nil {
		return nil, err
	}
	out.Snapshots = rows
	if len(rows) == 0 {
		out.Summary = prettyEmptyBookHistory(symbol)
		return out, nil
	}
	out.Summary = fmt.Sprintf("%d stored book(s) for %s. Newest mid %s at %s.",
		len(rows), domain.NormalizeSymbol(domain.ExchangeBinance, symbol),
		formatBookQty(rows[0].Mid), rows[0].SampledAt.UTC().Format(time.RFC3339))
	return out, nil
}

// CompareBookHistory diffs stored books nearest to two times.
func (s *Service) CompareBookHistory(ctx context.Context, exchange, symbol string, from, to time.Time) (*domain.BookHistoryDiff, error) {
	if s == nil || s.bookHist == nil {
		return nil, fmt.Errorf("%w: order book history not configured", domain.ErrUpstream)
	}
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("%w: from and to times are required", domain.ErrInvalidArgument)
	}
	s.noteBook(ex, symbol)
	return s.bookHist.Compare(ctx, string(ex), symbol, from.UTC(), to.UTC())
}

func prettyEmptyBookHistory(symbol string) string {
	return "No stored order books yet for " + symbol + ". History starts after the first sample."
}

func formatBookQty(v float64) string {
	s := domain.FormatSignedQty(v)
	if len(s) > 0 && s[0] == '+' {
		return s[1:]
	}
	return s
}
