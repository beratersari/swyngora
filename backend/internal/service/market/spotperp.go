package market

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const spotPerpNote = "Spot sums Binance, Coinbase, and Bybit. Perp sums Binance USD-M, Bybit linear, and OKX USDT-swap. A band counts a venue only when that book reaches the % on both sides. Binance and Bybit rows compare that venue's own spot book with its perp. Informational only — not financial advice."

// GetSpotPerpLiquidity compares resting spot size with perpetual size at the same % of mid.
func (s *Service) GetSpotPerpLiquidity(ctx context.Context, symbol string, rangePct float64) (*domain.SpotPerpLiquidity, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	rangePct, err = domain.ParseSpotPerpRangePct(rangePct)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)

	spotEx := make([]domain.Exchange, 0, 3)
	for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit} {
		if _, err := s.port(ex); err == nil {
			spotEx = append(spotEx, ex)
		}
	}
	var spotBooks []domain.VenueRawBook
	if len(spotEx) > 0 {
		spotBooks, err = s.fetchVenueBooks(ctx, symbol, spotEx)
		if err != nil {
			spotBooks = nil
		}
	}
	perpBooks := s.fetchPerpBooks(ctx, symbol)
	out := domain.CompareSpotPerpLiquidity(symbol, spotBooks, perpBooks, rangePct, time.Now().UTC())
	out.Note = spotPerpNote
	return &out, nil
}

func (s *Service) fetchPerpBooks(ctx context.Context, symbol string) []domain.VenueRawBook {
	want := domain.FuturesVenues()
	got := make([]domain.VenueRawBook, len(want))
	var wg sync.WaitGroup
	for i, ex := range want {
		wg.Add(1)
		go func(i int, ex domain.Exchange) {
			defer wg.Done()
			row := domain.VenueRawBook{Exchange: ex, Symbol: symbol}
			p := s.futuresDepthPort(ex)
			if p == nil {
				row.Err = "perpetual depth not configured"
				got[i] = row
				return
			}
			book, err := p.GetFuturesDepth(ctx, symbol)
			if err != nil || book == nil || (len(book.Bids) == 0 && len(book.Asks) == 0) {
				if err != nil {
					row.Err = err.Error()
				} else {
					row.Err = "no perpetual depth"
				}
				got[i] = row
				return
			}
			row.Book = *book
			got[i] = row
		}(i, ex)
	}
	wg.Wait()
	return got
}
