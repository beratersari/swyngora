package market

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationHuntHeatmap builds a price × time liquidation intensity map.
// range is 12h, 24h, 3d, 7d, 14d, 30d, or 90d. exchange=all returns Binance, Bybit, OKX, and combined.
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
	want := s.futuresWant(ex)
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
	var bars []domain.Candle
	if ex == domain.ExchangeOKX {
		if p := s.futuresCandle(ex); p != nil {
			q := domain.CandleQuery{
				Symbol: symbol, Interval: domain.CandleInterval(spec.CandleIV), Limit: limit,
				StartTime: f, EndTime: t,
			}
			got, err := p.GetFuturesCandles(ctx, q)
			if err != nil {
				slog.Warn("okx heatmap candles", "symbol", symbol, "interval", spec.CandleIV, "err", err)
			} else {
				bars = got
			}
		} else {
			slog.Warn("okx heatmap candles", "symbol", symbol, "err", "no futures candle port")
		}
	} else {
		got, err := s.GetCandles(ctx, string(ex), symbol, spec.CandleIV, limit, &f, &t)
		if err == nil {
			bars = got
		}
	}
	out := huntPointsFromCandles(bars, from, to)
	if len(out) > 0 {
		return out
	}
	if ex == domain.ExchangeOKX {
		if s.basis != nil {
			if b := s.basis[ex]; b != nil {
				if q, err := b.GetBasisQuote(ctx, symbol); err == nil && q != nil && q.FuturesLast > 0 {
					px := q.FuturesLast
					return []domain.HuntHeatmapPricePoint{{Time: to, Price: px, High: px, Low: px}}
				}
			}
		}
		return nil
	}
	if tkr, terr := s.GetTicker24h(ctx, string(ex), symbol); terr == nil && tkr != nil {
		if px, perr := strconv.ParseFloat(tkr.LastPrice, 64); perr == nil && px > 0 {
			return []domain.HuntHeatmapPricePoint{{Time: to, Price: px, High: px, Low: px}}
		}
	}
	return nil
}

func huntPointsFromCandles(bars []domain.Candle, from, to time.Time) []domain.HuntHeatmapPricePoint {
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
	ser.OI = s.fillHuntOpenInterest(ctx, ex, symbol, ser.OI, domain.HuntHeatmapColumns(spec, to), spec.Step, to)
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

// openInterestHistLister is implemented by the Binance, Bybit, and OKX
// clients. The heatmap uses it only to fill columns the archive left empty,
// and each client clamps the request to that endpoint's public lookback.
type openInterestHistLister interface {
	ListOpenInterestHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.OpenInterestPoint, error)
}

// fillHuntOpenInterest keeps stored samples and, when those leave a column
// empty, adds that venue's public open-interest history. The public request
// stays inside the documented lookback. It does not extend the archive.
func (s *Service) fillHuntOpenInterest(ctx context.Context, ex domain.Exchange, symbol string, rows []domain.FuturesSnapshot, times []time.Time, step time.Duration, to time.Time) []domain.FuturesSnapshot {
	gapFrom, gapTo, missing := domain.HuntOIGap(rows, times, step)
	if !missing {
		return rows
	}
	p := s.oiPort(ex)
	if p == nil {
		return rows
	}
	if hist, ok := p.(openInterestHistLister); ok {
		pts, err := hist.ListOpenInterestHistory(ctx, symbol, gapFrom, gapTo)
		if err != nil {
			slog.Warn("heatmap open interest history", "exchange", string(ex), "symbol", symbol, "err", err)
		} else {
			rows = appendHuntOI(ex, symbol, rows, pts)
		}
	}
	if domain.HuntOICovers(rows, times, step) {
		return rows
	}
	cur, err := p.GetOpenInterestSeries(ctx, symbol)
	if err != nil || cur == nil {
		return rows
	}
	rows = appendHuntOI(ex, symbol, rows, cur.History)
	ts := cur.Current.Time
	if ts.IsZero() {
		ts = to
	}
	if cur.Current.Value > 0 {
		rows = appendHuntOI(ex, symbol, rows, []domain.OpenInterestPoint{{
			Time: ts, Value: cur.Current.Value, Contracts: cur.Current.Contracts,
		}})
	}
	return rows
}

func appendHuntOI(ex domain.Exchange, symbol string, rows []domain.FuturesSnapshot, pts []domain.OpenInterestPoint) []domain.FuturesSnapshot {
	if len(pts) == 0 {
		return rows
	}
	out := append([]domain.FuturesSnapshot(nil), rows...)
	for _, pt := range pts {
		if pt.Value <= 0 || pt.Time.IsZero() {
			continue
		}
		out = append(out, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricOpenInterest, Exchange: ex, Symbol: symbol,
			SampledAt: pt.Time.UTC(), Value: pt.Value, Contracts: pt.Contracts,
		})
	}
	return out
}

func (s *Service) huntHistSnapshots(ctx context.Context, metric, exchange, symbol string, from, to time.Time) []domain.FuturesSnapshot {
	if s.futHist == nil {
		return nil
	}
	lead := 30 * time.Minute
	q := domain.FuturesHistoryQuery{
		Metric: metric, Exchange: exchange, Symbol: symbol,
		From: from.Add(-lead), To: to, Limit: domain.MaxHuntHeatmapHistory,
	}
	// Past a day, the newest 2,500 five-minute rows hide older hourly
	// archive samples. One sample per bucket keeps those hours, and a
	// bucket with no row is left out.
	if bucket := domain.HuntSnapshotBucket(to.Sub(from)); bucket > 0 {
		q.Bucket = bucket
		if bucket > lead {
			q.From = from.Add(-bucket)
		}
	}
	raw, err := s.futHist.History(ctx, q)
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
