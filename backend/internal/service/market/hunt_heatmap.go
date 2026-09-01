package market

import (
	"context"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationHuntHeatmap builds a price × time liquidation intensity map.
// range is 12h, 24h, 3d, or 7d. exchange=all returns Binance, Bybit, and combined.
func (s *Service) GetLiquidationHuntHeatmap(ctx context.Context, exchange, symbol, rawRange string) (*domain.HuntHeatmapReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseHuntHeatmapRange(rawRange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	if s.liqWatch != nil {
		s.liqWatch.Watch(symbol)
	}
	now := time.Now().UTC()
	from := now.Add(-spec.Window)
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	venues := make([]domain.HuntHeatmapVenueSeries, 0, len(want))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ser := s.huntHeatmapVenueSeries(ctx, v, symbol, spec, from, now)
			mu.Lock()
			venues = append(venues, ser)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	got := domain.BuildHuntHeatmap(domain.HuntHeatmapInput{
		Symbol: symbol, Spec: spec, To: now, Venues: venues,
	})
	return &got, nil
}

func (s *Service) huntHeatmapPrices(ctx context.Context, ex domain.Exchange, symbol string, spec domain.HuntHeatmapSpec, from, to time.Time) []domain.HuntHeatmapPricePoint {
	limit := spec.CandleLim
	if limit < 10 {
		limit = 80
	}
	f, t := from.Add(-spec.Step), to
	bars, err := s.GetCandles(ctx, string(ex), symbol, spec.CandleIV, limit, &f, &t)
	if err != nil {
		bars = nil
	}
	out := make([]domain.HuntHeatmapPricePoint, 0, len(bars))
	lo, hi := from.Add(-2*time.Hour), to.Add(2*time.Hour)
	for _, c := range bars {
		px, perr := strconv.ParseFloat(c.Close, 64)
		if perr != nil || px <= 0 {
			continue
		}
		ts := c.CloseTime.UTC()
		if ts.Before(lo) || ts.After(hi) {
			continue
		}
		high, low := px, px
		if v, err := strconv.ParseFloat(c.High, 64); err == nil && v > 0 {
			high = v
		}
		if v, err := strconv.ParseFloat(c.Low, 64); err == nil && v > 0 {
			low = v
		}
		out = append(out, domain.HuntHeatmapPricePoint{Time: ts, Price: px, High: high, Low: low})
	}
	if len(out) == 0 {
		if tkr, terr := s.GetTicker24h(ctx, string(ex), symbol); terr == nil && tkr != nil {
			if px, perr := strconv.ParseFloat(tkr.LastPrice, 64); perr == nil && px > 0 {
				out = append(out, domain.HuntHeatmapPricePoint{Time: to, Price: px, High: px, Low: px})
			}
		}
	}
	return out
}

func (s *Service) huntHeatmapVenueSeries(ctx context.Context, ex domain.Exchange, symbol string, spec domain.HuntHeatmapSpec, from, to time.Time) domain.HuntHeatmapVenueSeries {
	ser := domain.HuntHeatmapVenueSeries{Exchange: ex}
	ser.Prices = s.huntHeatmapPrices(ctx, ex, symbol, spec, from, to)
	if s.futHist != nil {
		ser.OI = s.huntHistSnapshots(ctx, domain.FuturesMetricOpenInterest, string(ex), symbol, from, to)
		ser.LongShort = s.huntHistSnapshots(ctx, domain.FuturesMetricLongShort, string(ex), symbol, from, to)
		ser.Funding = s.huntHistSnapshots(ctx, domain.FuturesMetricFunding, string(ex), symbol, from, to)
		ser.Liquidations = s.huntHistLiquidations(ctx, string(ex), symbol, from, to)
	}
	if len(ser.OI) == 0 {
		if p := s.oiPort(ex); p != nil {
			if cur, err := p.GetOpenInterestSeries(ctx, symbol); err == nil && cur != nil && cur.Current.Value > 0 {
				ser.OI = []domain.FuturesSnapshot{{
					Metric: domain.FuturesMetricOpenInterest, Exchange: ex, Symbol: symbol,
					SampledAt: to, Value: cur.Current.Value, Contracts: cur.Current.Contracts,
				}}
			}
		}
	}
	if len(ser.LongShort) == 0 {
		if p := s.longShortPort(ex); p != nil {
			if cur, err := p.GetLongShortSeries(ctx, symbol, 1); err == nil && cur != nil {
				pt := domain.NormalizeLongShortPoint(cur.Current)
				ser.LongShort = []domain.FuturesSnapshot{{
					Metric: domain.FuturesMetricLongShort, Exchange: ex, Symbol: symbol,
					SampledAt: to, LongShare: pt.LongShare, ShortShare: pt.ShortShare, Ratio: pt.Ratio,
				}}
			}
		}
	}
	if len(ser.Funding) == 0 {
		if p := s.fundingPort(ex); p != nil {
			if cur, err := p.GetFundingSeries(ctx, symbol, 1); err == nil && cur != nil {
				ser.Funding = []domain.FuturesSnapshot{{
					Metric: domain.FuturesMetricFunding, Exchange: ex, Symbol: symbol,
					SampledAt: to, FundingRate: cur.Current.Rate,
				}}
			}
		}
	}
	if len(ser.Liquidations) == 0 && s.liq != nil {
		ser.Liquidations = s.liq.Events(string(ex), symbol, from)
	}
	return ser
}

func (s *Service) huntHistSnapshots(ctx context.Context, metric, exchange, symbol string, from, to time.Time) []domain.FuturesSnapshot {
	if s.futHist == nil {
		return nil
	}
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric: metric, Exchange: exchange, Symbol: symbol,
		From: from.Add(-30 * time.Minute), To: to, Limit: domain.MaxHuntHeatmapHistory,
	})
	if err != nil || raw == nil {
		return nil
	}
	rows, ok := raw.([]domain.FuturesSnapshot)
	if !ok {
		return nil
	}
	return rows
}

func (s *Service) huntHistLiquidations(ctx context.Context, exchange, symbol string, from, to time.Time) []domain.LiquidationEvent {
	if s.futHist == nil {
		return nil
	}
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric: "liquidations", Exchange: exchange, Symbol: symbol,
		From: from, To: to, Limit: domain.MaxHuntHeatmapHistory,
	})
	if err != nil || raw == nil {
		return nil
	}
	rows, ok := raw.([]domain.LiquidationEvent)
	if !ok {
		return nil
	}
	return rows
}
