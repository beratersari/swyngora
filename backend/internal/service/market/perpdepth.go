package market

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const perpDepthDisclaimer = "Resting USD-M / linear perpetual depth: bid notional within N% below mid vs ask notional within N% above. Binance USD-M (1000 levels) and Bybit linear (500 levels) separately plus combined. Combined adds each venue's own ±% band (not a shared price grid). Snapshot may not fully cover 2%. Informational only — not financial advice."

// GetPerpDepth returns bid-below vs ask-above liquidity on BTC (and other) perps.
func (s *Service) GetPerpDepth(ctx context.Context, exchange, symbol string, rangePct float64) (*domain.PerpDepthReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParsePerpDepthExchange(exchange)
	if err != nil {
		return nil, err
	}
	rangePct, err = domain.ParsePerpDepthRangePct(rangePct)
	if err != nil {
		return nil, err
	}
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	s.noteFutures(symbol)

	type fetched struct {
		ex   domain.Exchange
		book *domain.RawOrderBook
		err  error
	}
	got := make([]fetched, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			p := s.futuresDepthPort(v)
			if p == nil {
				got[i] = fetched{ex: v, err: fmt.Errorf("perpetual depth not configured")}
				return
			}
			book, ferr := p.GetFuturesDepth(ctx, symbol)
			got[i] = fetched{ex: v, book: book, err: ferr}
		}(i, v)
	}
	wg.Wait()

	out := &domain.PerpDepthReport{
		Symbol:   symbol,
		Exchange: ex,
		RangePct: rangePct,
		AsOf:     time.Now().UTC(),
		Venues:   make([]domain.PerpDepthVenue, 0, len(want)),
		Note:     perpDepthDisclaimer,
	}
	for _, f := range got {
		if f.err != nil && (f.book == nil || (len(f.book.Bids) == 0 && len(f.book.Asks) == 0)) {
			out.Venues = append(out.Venues, domain.PerpDepthVenue{
				Exchange: f.ex, Symbol: symbol, Error: f.err.Error(), Summary: f.err.Error(),
			})
			continue
		}
		out.Venues = append(out.Venues, domain.BuildPerpDepthVenue(f.ex, symbol, f.book, rangePct))
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" && len(out.Venues) > 0 {
		out.Combined = domain.CombinePerpDepthVenues(symbol, out.Venues, rangePct)
	}
	out.Summary = domain.ExplainPerpDepthReport(*out)
	return out, nil
}

func (s *Service) futuresDepthPort(ex domain.Exchange) domain.FuturesDepthPort {
	p, err := s.port(ex)
	if err != nil || p == nil {
		return nil
	}
	fp, ok := p.(domain.FuturesDepthPort)
	if !ok {
		return nil
	}
	return fp
}
