package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const netPositionNoteFallback = "Net long and net short need both open interest and taker buy/sell. OKX is not included."

// takerBucketSource is implemented by the Binance and Bybit clients.
type takerBucketSource interface {
	GetTakerBuckets(ctx context.Context, symbol string) ([]domain.TakerBucket, error)
}

// GetNetPosition estimates how much futures exposure went long versus short
// over 15m, 1h, and 4h from open interest and taker buy/sell.
func (s *Service) GetNetPosition(ctx context.Context, exchange, symbol string) (*domain.NetPositionReport, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	if ex == string(domain.ExchangeOKX) {
		return nil, fmt.Errorf("%w: net position needs binance, bybit, or all", domain.ErrInvalidArgument)
	}
	s.noteFutures(symbol)
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	inputs := make([]domain.NetPositionVenueInput, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			inputs[i] = s.netPositionVenue(ctx, v, symbol)
		}(i, v)
	}
	wg.Wait()
	now := time.Now().UTC()
	rep := domain.BuildNetPosition(symbol, ex, inputs, now)
	if rep.Summary == "" {
		rep.Summary = netPositionNoteFallback
	}
	return rep, nil
}

func (s *Service) netPositionVenue(ctx context.Context, ex domain.Exchange, symbol string) domain.NetPositionVenueInput {
	in := domain.NetPositionVenueInput{Exchange: ex, Bar: netPositionBar(ex)}
	if p := s.oiPort(ex); p != nil {
		ser, err := p.GetOpenInterestSeries(ctx, symbol)
		if err != nil {
			in.Error = err.Error()
		} else {
			in.Series = ser
		}
	} else if in.Error == "" {
		in.Error = "open interest not configured"
	}
	if p := s.takerPort(ex); p != nil {
		if src, ok := p.(takerBucketSource); ok {
			buckets, err := src.GetTakerBuckets(ctx, symbol)
			if err != nil && in.Error == "" {
				in.Error = err.Error()
			} else {
				in.Buckets = buckets
			}
		} else if in.Error == "" {
			in.Error = "taker bars not configured"
		}
	} else if in.Error == "" {
		in.Error = "taker flow not configured"
	}
	return in
}

func netPositionBar(ex domain.Exchange) time.Duration {
	if ex == domain.ExchangeBinance {
		return 5 * time.Minute
	}
	return time.Minute
}
