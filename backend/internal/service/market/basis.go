package market

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const basisDisclaimer = "Basis is perpetual futures minus spot/index on that venue. last is the tape price vs the spot index; mark vs index is what funding is built from. Expanding means the absolute gap is getting larger. Informational only — not financial advice."

// GetBasis returns futures-vs-spot premium/discount per venue and agreement.
func (s *Service) GetBasis(ctx context.Context, exchange, symbol string) (*domain.BasisReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseBasisExchange(exchange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	now := time.Now().UTC()
	out := &domain.BasisReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.BasisVenueReport, 0, len(want)),
		Note:     basisDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.basisOne(ctx, v, symbol)
			mu.Lock()
			out.Venues = append(out.Venues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" && len(out.Venues) > 0 {
		out.Agreement = domain.CompareBasisVenues(out.Venues)
	}
	return out, nil
}

func (s *Service) basisOne(ctx context.Context, ex domain.Exchange, symbol string) domain.BasisVenueReport {
	out := domain.BasisVenueReport{Exchange: ex, Symbol: symbol}
	p := s.basisPort(ex)
	if p == nil {
		out.Error = "basis not configured"
		out.Summary = out.Error
		return out
	}
	q, err := p.GetBasisQuote(ctx, symbol)
	if err != nil {
		out.Error = err.Error()
		out.Summary = err.Error()
		return out
	}
	if q == nil {
		out.Error = "empty basis"
		return out
	}
	q.FundingRate, q.OIChange1h = s.basisContext(ctx, ex, symbol)
	return domain.BuildBasisVenue(*q)
}

func (s *Service) basisPort(ex domain.Exchange) domain.BasisPort {
	if s == nil || s.basis == nil {
		return nil
	}
	return s.basis[ex]
}

func (s *Service) basisContext(ctx context.Context, ex domain.Exchange, symbol string) (funding, oi1h float64) {
	oi1h = math.NaN()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if p := s.fundingPort(ex); p != nil {
			if ser, err := p.GetFundingSeries(ctx, symbol, 3); err == nil && ser != nil {
				funding = ser.Current.Rate
			}
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.oiPort(ex); p != nil {
			if ser, err := p.GetOpenInterestSeries(ctx, symbol); err == nil && ser != nil {
				oi1h = domain.OIChangePctFromSeries(ser, time.Hour, time.Now().UTC())
			}
		}
	}()
	wg.Wait()
	return funding, oi1h
}
