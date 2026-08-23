package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const absorptionDisclaimer = "Absorption is large aggressive market-buy or market-sell volume that does not move price with it — someone on the other side is taking the flow. bid = bids absorbing sells (price holds up). ask = asks absorbing buys (price holds down). Score 0–100 from how one-sided the flow is and how little price delivered. Futures uses Binance USD-M / Bybit linear taker volume; spot uses Binance kline taker-buy and Bybit spot trades. Combined uses the overlapping time range. Informational only — not financial advice."

// GetAbsorption returns aggressive buy/sell versus price movement, plus who is absorbing.
func (s *Service) GetAbsorption(ctx context.Context, exchange, symbol string) (*domain.AbsorptionReport, error) {
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

	out := &domain.AbsorptionReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.AbsorptionVenue, 0, len(want)),
		Note:     absorptionDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.absorptionOne(ctx, v, symbol, prices, now)
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
		out.Combined = domain.CombineAbsorptionVenues(symbol, out.Venues, prices, now)
	}

	out.SpotVenues = make([]domain.AbsorptionVenue, 0, len(want))
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.absorptionSpotOne(ctx, v, symbol, prices, now)
			mu.Lock()
			out.SpotVenues = append(out.SpotVenues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.SpotVenues, func(i, j int) bool {
		return string(out.SpotVenues[i].Exchange) < string(out.SpotVenues[j].Exchange)
	})
	if ex == "all" && len(out.SpotVenues) > 0 {
		out.SpotCombined = domain.CombineAbsorptionVenues(symbol, out.SpotVenues, prices, now)
		if out.SpotCombined != nil {
			out.SpotCombined.Market = domain.CVDMarketSpot
		}
	}
	out.Summary = domain.ExplainAbsorptionReport(*out)
	return out, nil
}

func (s *Service) absorptionOne(ctx context.Context, ex domain.Exchange, symbol string, prices []domain.CVDPrice, now time.Time) domain.AbsorptionVenue {
	out := domain.AbsorptionVenue{
		Exchange: ex, Symbol: symbol, Market: domain.CVDMarketFutures,
		Points: []domain.AbsorptionPoint{}, Windows: []domain.AbsorptionWindowStat{},
	}
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
	return domain.BuildAbsorptionSeries(ex, symbol, merged, prices, now, started)
}

func (s *Service) absorptionSpotOne(ctx context.Context, ex domain.Exchange, symbol string, prices []domain.CVDPrice, now time.Time) domain.AbsorptionVenue {
	out := domain.AbsorptionVenue{
		Exchange: ex, Symbol: symbol, Market: domain.CVDMarketSpot,
		Points: []domain.AbsorptionPoint{}, Windows: []domain.AbsorptionWindowStat{},
	}
	var live []domain.TakerBucket
	if candles, err := s.GetCandles(ctx, string(ex), symbol, "5m", 320, nil, nil); err == nil {
		live = domain.TakerBucketsFromCandles(ex, symbol, candles)
	}
	if len(live) == 0 {
		if p := s.spotBucketPort(ex); p != nil {
			got, err := p.GetSpotTakerBuckets(ctx, symbol)
			if err != nil {
				out.Error = err.Error()
				out.Summary = err.Error()
				return out
			}
			live = got
		}
	}
	if len(live) == 0 {
		out.Error = "spot absorption not available"
		out.Summary = out.Error
		return out
	}
	storeEx := domain.SpotStoreExchange(ex)
	from := now.Add(-domain.DefaultCVDLookback)
	var stored []domain.TakerBucket
	if s.takerStore != nil {
		stored, _ = s.takerStore.ListTakerBuckets(ctx, storeEx, symbol, from, now)
		persist := make([]domain.TakerBucket, len(live))
		copy(persist, live)
		for i := range persist {
			persist[i].Exchange = domain.Exchange(storeEx)
		}
		if len(persist) > 0 {
			_, _ = s.takerStore.UpsertTakerBuckets(ctx, persist)
		}
	}
	merged := domain.MergeTakerBuckets(stored, live)
	started := now
	for _, b := range merged {
		if !b.Start.IsZero() && b.Start.Before(started) {
			started = b.Start
		}
	}
	ser := domain.BuildAbsorptionSeries(ex, symbol, merged, prices, now, started)
	ser.Market = domain.CVDMarketSpot
	return ser
}
