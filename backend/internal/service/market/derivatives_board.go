package market

import (
	"context"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/cache"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	derivativesBoardCacheTTL = 20 * time.Second
	derivativesBoardWorkers  = 4
)

// GetDerivativesBoard returns last mark, funding, open interest, and 24h
// liquidations for the liquid USDT perpetuals.
func (s *Service) GetDerivativesBoard(ctx context.Context, exchange string, limit int) (*domain.DerivativesBoard, error) {
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	limit = domain.ParseDerivativesBoardLimit(limit)
	key := ex + "|" + strconv.Itoa(limit)
	if c := s.derivativesBoardCache(); c != nil {
		if hit, ok := c.Get(key); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}
	symbols, err := s.LiquidPerpSymbols(ctx, limit)
	if err != nil && len(symbols) == 0 {
		return nil, err
	}
	want := domain.ExpectedFuturesVenues(ex)
	coins := make([]domain.DerivativesCoinInput, len(symbols))
	var wg sync.WaitGroup
	sem := make(chan struct{}, derivativesBoardWorkers)
	for i, sym := range symbols {
		wg.Add(1)
		go func(i int, sym string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				coins[i] = domain.DerivativesCoinInput{Symbol: sym}
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			coins[i] = s.derivativesCoin(ctx, ex, sym, want)
		}(i, sym)
	}
	wg.Wait()
	out := domain.BuildDerivativesBoard(ex, coins, time.Now().UTC())
	if c := s.derivativesBoardCache(); c != nil && out != nil {
		cp := *out
		c.Set(key, &cp)
	}
	return out, nil
}

func (s *Service) derivativesCoin(ctx context.Context, exchange, symbol string, want []domain.Exchange) domain.DerivativesCoinInput {
	s.noteFutures(symbol)
	venues := make([]domain.DerivativesVenueInput, len(want))
	var wg sync.WaitGroup
	for i, ex := range want {
		wg.Add(1)
		go func(i int, ex domain.Exchange) {
			defer wg.Done()
			venues[i] = s.derivativesVenue(ctx, ex, symbol)
		}(i, ex)
	}
	wg.Wait()
	longs, shorts := s.derivativesLiquidations(exchange, symbol, time.Now().UTC().Add(-24*time.Hour))
	for i := range venues {
		venues[i].LiqLong = longs[venues[i].Exchange]
		venues[i].LiqShort = shorts[venues[i].Exchange]
	}
	in := domain.DerivativesCoinInput{Symbol: symbol, Venues: venues}
	if s != nil && s.liq != nil {
		in.LiqFeed = s.liq.Snapshot(exchange, symbol).Feed
	}
	return in
}

func (s *Service) derivativesVenue(ctx context.Context, ex domain.Exchange, symbol string) domain.DerivativesVenueInput {
	out := domain.DerivativesVenueInput{Exchange: ex}
	if s == nil {
		return out
	}
	if p := s.oiPort(ex); p != nil {
		ser, err := p.GetOpenInterestSeries(ctx, symbol)
		if err == nil && ser != nil && derivativesSeriesPresent(ser) {
			out.OI = ser
			out.OIValue = ser.Current.Value
			out.OIAt = ser.Current.Time
			out.OIOK = true
		}
	}
	if p := s.fundingPort(ex); p != nil {
		ser, err := p.GetFundingSeries(ctx, symbol, 1)
		if err == nil && ser != nil && !ser.Current.Time.IsZero() {
			out.FundingOK = true
			out.FundingRate = ser.Current.Rate
			out.FundingAt = ser.Current.Time
			out.MarkPrice = ser.Current.MarkPrice
		}
	}
	return out
}

func derivativesSeriesPresent(ser *domain.OpenInterestSeries) bool {
	if ser == nil {
		return false
	}
	cur := ser.Current
	return !cur.Time.IsZero() || cur.Value != 0 || cur.Contracts != 0
}

func (s *Service) derivativesLiquidations(exchange, symbol string, since time.Time) (map[domain.Exchange]float64, map[domain.Exchange]float64) {
	longs := map[domain.Exchange]float64{}
	shorts := map[domain.Exchange]float64{}
	if s == nil || s.liq == nil {
		return longs, shorts
	}
	for _, e := range s.liq.EventsSince(exchange, symbol, since) {
		switch e.Side {
		case domain.LiquidationSideLong:
			longs[e.Exchange] += e.Notional
		case domain.LiquidationSideShort:
			shorts[e.Exchange] += e.Notional
		}
	}
	return longs, shorts
}

func (s *Service) derivativesBoardCache() *cache.TTL[*domain.DerivativesBoard] {
	if s == nil {
		return nil
	}
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if s.boardCache == nil {
		s.boardCache = cache.New[*domain.DerivativesBoard](derivativesBoardCacheTTL)
	}
	return s.boardCache
}
