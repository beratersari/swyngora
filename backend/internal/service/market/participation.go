package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const participationDisclaimer = "Participation is open-interest USDT change divided by futures taker volume (buy+sell) over the same window. 2B volume with OI +300M is 15% — that share of volume stuck as new positions; the rest is transfer between existing books. 15m / 1h / 4h on Binance USD-M and Bybit linear. Informational only — not financial advice."

// GetParticipation returns OI-delta / futures volume for 15m, 1h, and 4h.
func (s *Service) GetParticipation(ctx context.Context, exchange, symbol string) (*domain.ParticipationReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	now := time.Now().UTC()
	out := &domain.ParticipationReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.ParticipationVenue, 0, len(want)),
		Note:     participationDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.participationOne(ctx, v, symbol, now)
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
		out.Combined = domain.CombineParticipationVenues(symbol, out.Venues)
	}
	return out, nil
}

func (s *Service) participationOne(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) domain.ParticipationVenue {
	var oi *domain.OpenInterestSeries
	if p := s.oiPort(ex); p != nil {
		if ser, err := p.GetOpenInterestSeries(ctx, symbol); err == nil {
			oi = ser
		}
	}
	var buckets []domain.TakerBucket
	if p := s.bucketPort(ex); p != nil {
		if got, err := p.GetTakerBuckets(ctx, symbol); err == nil {
			buckets = got
		}
	}
	return domain.BuildParticipationVenue(ex, symbol, oi, buckets, now)
}
