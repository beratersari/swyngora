package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const cvdDisclaimer = "CVD is the running sum of aggressive market-buy notional minus market-sell notional (who hit the futures book). It is not account long/short. Price is the 5-minute close. Binance uses the public 5m taker series; Bybit uses live trades (24h is complete after this process has been collecting). Combined adds both venues' delta, never averages. Informational only — not financial advice."

// WithTakerBucketStore attaches durable taker bars for CVD.
func (s *Service) WithTakerBucketStore(store domain.TakerBucketStore) *Service {
	if s != nil {
		s.takerStore = store
	}
	return s
}

// GetCVD returns cumulative taker buy−sell versus price for Binance, Bybit, and combined.
func (s *Service) GetCVD(ctx context.Context, exchange, symbol string) (*domain.CVDReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	if s.liqWatch != nil {
		s.liqWatch.Watch(symbol)
	}
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	now := time.Now().UTC()
	prices := s.cvdPrices(ctx, symbol)

	out := &domain.CVDReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.CVDVenueSeries, 0, len(want)),
		Note:     cvdDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.cvdOne(ctx, v, symbol, prices, now)
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
		out.Combined = domain.CombineCVDVenues(symbol, out.Venues, prices, now)
	}
	out.Summary = domain.ExplainCVDReport(*out)
	return out, nil
}

func (s *Service) cvdOne(ctx context.Context, ex domain.Exchange, symbol string, prices []domain.CVDPrice, now time.Time) domain.CVDVenueSeries {
	out := domain.CVDVenueSeries{Exchange: ex, Symbol: symbol, Points: []domain.CVDPoint{}, Windows: []domain.CVDWindowStat{}}
	p := s.bucketPort(ex)
	if p == nil {
		out.Error = "taker buckets not configured"
		out.Summary = out.Error
		return out
	}
	live, err := p.GetTakerBuckets(ctx, symbol)
	if err != nil {
		out.Error = err.Error()
		out.Summary = err.Error()
		return out
	}
	from := now.Add(-domain.DefaultCVDLookback)
	var stored []domain.TakerBucket
	if s.takerStore != nil {
		stored, _ = s.takerStore.ListTakerBuckets(ctx, string(ex), symbol, from, now)
		if len(live) > 0 {
			_, _ = s.takerStore.UpsertTakerBuckets(ctx, live)
		}
	}
	merged := domain.MergeTakerBuckets(stored, live)
	started := now
	for _, b := range merged {
		if !b.Start.IsZero() && b.Start.Before(started) {
			started = b.Start
		}
	}
	return domain.BuildCVDSeries(ex, symbol, merged, prices, now, started)
}

func (s *Service) cvdPrices(ctx context.Context, symbol string) []domain.CVDPrice {
	c, err := s.GetCandles(ctx, string(domain.ExchangeBinance), symbol, "5m", 320, nil, nil)
	if err != nil || len(c) == 0 {
		c, err = s.GetCandles(ctx, string(domain.ExchangeBybit), symbol, "5m", 320, nil, nil)
		if err != nil {
			return nil
		}
	}
	return domain.CVDPricesFromCandles(c)
}

func (s *Service) bucketPort(ex domain.Exchange) domain.TakerBucketPort {
	p := s.takerPort(ex)
	if p == nil {
		return nil
	}
	bp, ok := p.(domain.TakerBucketPort)
	if !ok {
		return nil
	}
	return bp
}
