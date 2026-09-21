package market

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetVolumeShare returns how futures quote volume splits across venues, plus 1h-style mix change.
func (s *Service) GetVolumeShare(ctx context.Context, exchange, symbol string) (*domain.VolumeShareReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseLiquidationExchange(exchange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	want := s.volumeWant(ex)
	now := time.Now().UTC()
	type fetched struct {
		ex   domain.Exchange
		bars []domain.FuturesVolumeBar
	}
	got := make([]fetched, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			p := s.volumePort(v)
			if p == nil {
				return
			}
			bars, err := p.ListFuturesVolumeBars(ctx, symbol)
			if err != nil {
				return
			}
			got[i] = fetched{ex: v, bars: bars}
		}(i, v)
	}
	wg.Wait()
	series := map[domain.Exchange][]domain.FuturesVolumeBar{}
	for _, f := range got {
		if f.ex == "" || len(f.bars) == 0 {
			continue
		}
		series[f.ex] = f.bars
	}
	return domain.BuildVolumeShare(ex, symbol, series, now), nil
}

func (s *Service) volumePort(ex domain.Exchange) domain.FuturesVolumePort {
	if s == nil || s.futVol == nil {
		return nil
	}
	return s.futVol[ex]
}

func (s *Service) volumeWant(exchange string) []domain.Exchange {
	if exchange != "" && exchange != "all" {
		return []domain.Exchange{domain.Exchange(exchange)}
	}
	var out []domain.Exchange
	for _, ex := range domain.FuturesVenues() {
		if s.volumePort(ex) != nil {
			out = append(out, ex)
		}
	}
	if len(out) == 0 {
		return []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit, domain.ExchangeOKX}
	}
	return out
}
