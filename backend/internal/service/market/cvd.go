package market

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const cvdDisclaimer = "CVD is the running sum of aggressive market-buy notional minus market-sell notional. Futures CVD sums measured prints (price × size) on Binance USD-M, Bybit linear, and OKX USDT-swap. Stored minutes extend the line while this process is recording. range 24h uses 5-minute bars. 7d uses 30-minute bars, 30d uses 2-hour bars, and 90d uses 4-hour bars, folded from those minutes. A range is complete only when the stored tape reaches its start. Spot CVD is who hit the spot book (Binance from 5m klines, Bybit from live spot trades). It is not account long/short. Windows show 15m / 1h / 4h / 24h change on the returned bars. Anchors reset that sum at 00:00 UTC (day) and at the current 4-hour open (00/04/08/12/16/20 UTC). Combined futures/spot each use the overlapping time range. spotFutures flags when spot buying and futures selling (or the reverse). Divergence runs do not merge across a quiet gap. Informational only — not financial advice."

// WithTakerBucketStore attaches durable taker bars for CVD.
func (s *Service) WithTakerBucketStore(store domain.TakerBucketStore) *Service {
	if s != nil {
		s.takerStore = store
	}
	return s
}

// GetCVD returns cumulative taker buy−sell versus price for Binance, Bybit, and combined.
// rangeID is 24h (default), 7d, 30d, or 90d. Empty is 24h.
func (s *Service) GetCVD(ctx context.Context, exchange, symbol, rangeID string) (*domain.CVDReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	span, err := domain.ParseCVDRange(rangeID)
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
	want := configuredFutures(ex, func(v domain.Exchange) bool { return s.takerPort(v) != nil })
	spotWant := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" && ex != string(domain.ExchangeOKX) {
		spotWant = []domain.Exchange{domain.Exchange(ex)}
	}
	if ex == string(domain.ExchangeOKX) {
		spotWant = nil
	}
	now := time.Now().UTC()
	prices := s.cvdPrices(ctx, symbol, span)

	out := &domain.CVDReport{
		Symbol:   symbol,
		Exchange: ex,
		Range:    span.ID,
		Bar:      span.BarLabel(),
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
			ven := s.cvdOne(ctx, v, symbol, prices, now, span)
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

	out.SpotVenues = make([]domain.CVDVenueSeries, 0, len(spotWant))
	for _, v := range spotWant {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.cvdSpotOne(ctx, v, symbol, prices, now, span)
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
		out.SpotCombined = domain.CombineCVDVenues(symbol, out.SpotVenues, prices, now)
	}
	spot := out.SpotCombined
	if spot == nil || len(spot.Points) == 0 {
		for i := range out.SpotVenues {
			if out.SpotVenues[i].Error == "" && len(out.SpotVenues[i].Points) > 0 {
				s := out.SpotVenues[i]
				spot = &s
				break
			}
		}
	}
	fut := out.Combined
	if fut == nil || len(fut.Points) == 0 {
		for i := range out.Venues {
			if out.Venues[i].Error == "" && len(out.Venues[i].Points) > 0 {
				f := out.Venues[i]
				fut = &f
				break
			}
		}
	}
	out.SpotFutures = domain.CompareSpotFutures(spot, fut)
	out.Summary = domain.ExplainCVDReport(*out)
	return out, nil
}

func (s *Service) cvdOne(ctx context.Context, ex domain.Exchange, symbol string, prices []domain.CVDPrice, now time.Time, span domain.CVDSpan) domain.CVDVenueSeries {
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
	from := now.Add(-span.Lookback)
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
	return domain.BuildCVDSeriesSpan(ex, symbol, merged, prices, now, started, span)
}

func (s *Service) cvdSpotOne(ctx context.Context, ex domain.Exchange, symbol string, prices []domain.CVDPrice, now time.Time, span domain.CVDSpan) domain.CVDVenueSeries {
	out := domain.CVDVenueSeries{
		Exchange: ex, Symbol: symbol, Market: domain.CVDMarketSpot,
		Points: []domain.CVDPoint{}, Windows: []domain.CVDWindowStat{},
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
		out.Error = "spot CVD not available"
		out.Summary = out.Error
		return out
	}
	storeEx := domain.SpotStoreExchange(ex)
	from := now.Add(-span.Lookback)
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
	ser := domain.BuildCVDSeriesSpan(ex, symbol, merged, prices, now, started, span)
	ser.Market = domain.CVDMarketSpot
	return ser
}

func (s *Service) spotBucketPort(ex domain.Exchange) domain.SpotTakerBucketPort {
	p := s.takerPort(ex)
	if p == nil {
		return nil
	}
	sp, ok := p.(domain.SpotTakerBucketPort)
	if !ok {
		return nil
	}
	return sp
}

func (s *Service) cvdPrices(ctx context.Context, symbol string, span domain.CVDSpan) []domain.CVDPrice {
	interval := span.CandleInterval
	limit := span.CandleLimit
	if interval == "" {
		interval = "5m"
	}
	if limit <= 0 {
		limit = 320
	}
	c, err := s.GetCandles(ctx, string(domain.ExchangeBinance), symbol, interval, limit, nil, nil)
	if err != nil || len(c) == 0 {
		c, err = s.GetCandles(ctx, string(domain.ExchangeBybit), symbol, interval, limit, nil, nil)
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
