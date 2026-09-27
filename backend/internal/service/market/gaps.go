package market

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const bookGapNote = "Empty areas are 0.1% price steps inside ±2% of mid that hold under 15% of the typical step on that side, joined when they sit together. Easier up means asks in that band are well below bids, so a move up meets less resting size. Easier down is the mirror. One snapshot of the visible book, not a forecast. Not financial advice."

// GetBookGaps finds thin stretches in the live spot book.
func (s *Service) GetBookGaps(ctx context.Context, exchange, symbol string, rangePct float64) (*domain.BookGaps, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	rangePct = domain.ClampBookGapRangePct(rangePct)
	var only []domain.Exchange
	rawEx := strings.ToLower(strings.TrimSpace(exchange))
	scope := "all"
	if rawEx != "" && rawEx != "all" && rawEx != "combined" {
		ex, err := s.ResolveExchange(exchange)
		if err != nil {
			return nil, err
		}
		only = []domain.Exchange{ex}
		scope = string(ex)
	}
	books, err := s.fetchVenueBooks(ctx, symbol, only)
	if err != nil {
		return nil, err
	}
	raws := make([]domain.RawOrderBook, 0, len(books))
	for _, b := range books {
		if b.Err == "" {
			raws = append(raws, b.Book)
		}
	}
	display := domain.CrossVenueSymbol(domain.ExchangeBinance, symbol)
	if display == "" {
		display = strings.ToUpper(symbol)
	}
	out := domain.FindBookGaps(display, scope, raws, domain.SharedBookMid(books), rangePct)
	out.Note = bookGapNote
	return &out, nil
}

// ScanBookGaps ranks liquid pairs by which side of the book is thinner.
func (s *Service) ScanBookGaps(ctx context.Context, exchange, quote string, rangePct float64, limit int) (*domain.BookGapScan, error) {
	ex, err := domain.ParseWallScanExchange(exchange)
	if err != nil {
		return nil, err
	}
	if quote == "" {
		quote = "USDT"
	}
	rangePct = domain.ClampBookGapRangePct(rangePct)
	limit = domain.ParseBookGapScanLimit(limit)
	listEx := ex
	if listEx == "all" {
		listEx = string(domain.ExchangeBinance)
	}
	symbols, err := s.orderCountUniverse(ctx, listEx, quote, limit)
	if err != nil {
		return nil, err
	}
	hits := make([]domain.BookGapScanHit, len(symbols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, symbol := range symbols {
		wg.Add(1)
		go func(i int, symbol string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			hits[i] = s.gapScanOne(ctx, ex, symbol, rangePct)
		}(i, symbol)
	}
	wg.Wait()
	out := domain.BuildBookGapScan(ex, quote, rangePct, hits)
	out.Note = bookGapNote
	return &out, nil
}

func (s *Service) gapScanOne(ctx context.Context, exchange, symbol string, rangePct float64) domain.BookGapScanHit {
	got, err := s.GetBookGaps(ctx, exchange, symbol, rangePct)
	if err != nil {
		return domain.BookGapScanHit{Symbol: symbol, Error: err.Error(), Summary: err.Error()}
	}
	if got == nil {
		return domain.BookGapScanHit{Symbol: symbol, Error: "empty book", Summary: "empty book"}
	}
	hit := domain.BookGapScanHit{
		Symbol: got.Symbol, Scope: got.Exchange, Easier: got.Easier,
		UpNotional: got.UpNotional, DownNotional: got.DownNotional,
		UpGapPct: got.UpGapPct, DownGapPct: got.DownGapPct,
		Summary: got.Summary,
	}
	if len(got.Gaps) > 0 {
		hit.WidestSide = got.Gaps[0].Direction
		hit.WidestPct = got.Gaps[0].WidthPct
	}
	return hit
}
