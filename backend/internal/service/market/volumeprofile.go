package market

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const volumeProfileDisclaimer = "Volume profile bins quote (USDT) volume from spot candles across each bar's high–low — not individual prints. Buy/sell uses taker-buy when the venue publishes it (Binance klines). Bybit klines do not, so those rows are total-only. Open-interest change is the USDT added or closed between readings, spread across the candles that traded during that step, on the same price rows. Liquidations use the print price on those rows. Combined adds Binance and Bybit at the same price levels. The point of control is the row with the most volume; the value area is the block around it that holds about 70% of volume. Informational only — not financial advice."

// GetVolumeProfile returns traded volume by price for Binance, Bybit, and combined.
func (s *Service) GetVolumeProfile(ctx context.Context, exchange, symbol, window string, start, end *time.Time, tickSize float64) (*domain.VolumeProfileReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	if tickSize < 0 || (tickSize > 0 && tickSize != tickSize) {
		return nil, fmt.Errorf("%w: tickSize must be > 0", domain.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	from, to, windowID, err := domain.ResolveVolumeProfileRange(window, start, end, now)
	if err != nil {
		return nil, err
	}
	interval := domain.ProfileBarInterval(to.Sub(from))
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}

	type fetched struct {
		ex   domain.Exchange
		bars []domain.VolumeProfileBar
		last float64
		err  error
	}
	got := make([]fetched, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			bars, last, ferr := s.volumeProfileVenue(ctx, v, symbol, interval, from, to)
			got[i] = fetched{ex: v, bars: bars, last: last, err: ferr}
		}(i, v)
	}
	wg.Wait()

	var rangeLow, rangeHigh, last float64
	for _, f := range got {
		if f.last > 0 && last == 0 {
			last = f.last
		}
		for _, b := range f.bars {
			if rangeLow == 0 || b.Low < rangeLow {
				rangeLow = b.Low
			}
			if b.High > rangeHigh {
				rangeHigh = b.High
			}
			if last == 0 && b.Close > 0 {
				last = b.Close
			}
		}
	}
	tick := tickSize
	if tick <= 0 {
		tick = domain.AutoTickSize(rangeLow, rangeHigh)
	} else {
		tick = domain.ClampVolumeProfileTick(rangeLow, rangeHigh, tick)
	}

	out := &domain.VolumeProfileReport{
		Symbol:   symbol,
		Exchange: ex,
		Window:   windowID,
		From:     from,
		To:       to,
		AsOf:     now,
		Venues:   make([]domain.VolumeProfileVenue, 0, len(want)),
		Note:     volumeProfileDisclaimer,
	}
	for _, f := range got {
		if f.err != nil && len(f.bars) == 0 {
			out.Venues = append(out.Venues, domain.VolumeProfileVenue{
				Exchange: f.ex, Symbol: symbol, From: from, To: to,
				Interval: string(interval), TickSize: tick, LastPrice: f.last,
				Bins: []domain.VolumeProfileBin{}, Error: f.err.Error(), Summary: f.err.Error(),
			})
			continue
		}
		px := f.last
		if px <= 0 {
			px = last
		}
		out.Venues = append(out.Venues, domain.BuildVolumeProfile(f.ex, symbol, f.bars, tick, px, from, to, interval))
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	barsByEx := map[domain.Exchange][]domain.VolumeProfileBar{}
	for _, f := range got {
		barsByEx[f.ex] = f.bars
	}
	s.attachProfileActivity(ctx, out, barsByEx, from, to)
	if ex == "all" && len(out.Venues) > 0 {
		out.Combined = domain.CombineVolumeProfiles(symbol, out.Venues, tick, last, from, to, interval)
	}
	out.Summary = domain.ExplainVolumeProfileReport(*out)
	return out, nil
}

func (s *Service) attachProfileActivity(ctx context.Context, out *domain.VolumeProfileReport, barsByEx map[domain.Exchange][]domain.VolumeProfileBar, from, to time.Time) {
	if s == nil || out == nil {
		return
	}
	for i := range out.Venues {
		v := &out.Venues[i]
		if v.Error != "" || len(v.Bins) == 0 {
			continue
		}
		oi := s.profileOiPoints(ctx, v.Exchange, out.Symbol, from, to)
		liqs := s.profileLiquidations(ctx, v.Exchange, out.Symbol, from, to)
		domain.AttachVolumeProfileActivity(v, barsByEx[v.Exchange], oi, liqs)
	}
}

func (s *Service) volumeProfileVenue(ctx context.Context, ex domain.Exchange, symbol string, interval domain.CandleInterval, from, to time.Time) ([]domain.VolumeProfileBar, float64, error) {
	var (
		candles []domain.Candle
		last    float64
		cerr    error
		wg      sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		candles, cerr = s.candlesForProfile(ctx, ex, symbol, interval, from, to)
	}()
	go func() {
		defer wg.Done()
		if tkr, err := s.GetTicker24h(ctx, string(ex), symbol); err == nil && tkr != nil {
			if px, err := strconv.ParseFloat(tkr.LastPrice, 64); err == nil && px > 0 {
				last = px
			}
		}
	}()
	wg.Wait()
	if cerr != nil {
		return nil, last, cerr
	}
	bars := domain.VolumeProfileBarsFromCandles(filterCandlesRange(candles, from, to, interval))
	return bars, last, nil
}

func (s *Service) candlesForProfile(ctx context.Context, ex domain.Exchange, symbol string, interval domain.CandleInterval, from, to time.Time) ([]domain.Candle, error) {
	p, err := s.port(ex)
	if err != nil {
		return nil, err
	}
	var all []domain.Candle
	start := from
	seen := map[int64]struct{}{}
	for page := 0; page < domain.MaxVolumeProfilePages; page++ {
		if ctx.Err() != nil {
			return all, ctx.Err()
		}
		if !start.Before(to) {
			break
		}
		batch, err := p.GetCandles(ctx, domain.CandleQuery{
			Symbol:    symbol,
			Interval:  interval,
			Limit:     domain.VolumeProfilePageLimit,
			StartTime: start,
			EndTime:   to,
		})
		if err != nil {
			if len(all) > 0 {
				return all, nil
			}
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		sort.Slice(batch, func(i, j int) bool { return batch[i].OpenTime.Before(batch[j].OpenTime) })
		added := 0
		for _, c := range batch {
			ms := c.OpenTime.UTC().UnixMilli()
			if _, ok := seen[ms]; ok {
				continue
			}
			seen[ms] = struct{}{}
			all = append(all, c)
			added++
		}
		lastOpen := batch[len(batch)-1].OpenTime.UTC()
		next := lastOpen.Add(time.Millisecond)
		if !next.After(start) {
			break
		}
		start = next
		if added == 0 || len(batch) < domain.VolumeProfilePageLimit {
			break
		}
	}
	return all, nil
}

func filterCandlesRange(in []domain.Candle, from, to time.Time, interval domain.CandleInterval) []domain.Candle {
	if len(in) == 0 {
		return nil
	}
	step := domain.IntervalDuration(interval)
	if step <= 0 {
		step = time.Minute
	}
	out := make([]domain.Candle, 0, len(in))
	for _, c := range in {
		t := c.OpenTime.UTC()
		if t.IsZero() {
			continue
		}
		// Keep bars that overlap [from, to).
		end := t.Add(step)
		if !end.After(from) || !t.Before(to) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (s *Service) profileOiPoints(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) []domain.VolumeProfileOiPoint {
	var pts []domain.VolumeProfileOiPoint
	if p := s.oiPort(ex); p != nil {
		if ser, err := p.GetOpenInterestSeries(ctx, symbol); err == nil && ser != nil {
			if ser.Current.Value > 0 && !ser.Current.Time.IsZero() {
				pts = append(pts, domain.VolumeProfileOiPoint{Time: ser.Current.Time, Value: ser.Current.Value})
			}
			for _, h := range ser.History {
				if h.Value > 0 && !h.Time.IsZero() {
					pts = append(pts, domain.VolumeProfileOiPoint{Time: h.Time, Value: h.Value})
				}
			}
		}
	}
	if s.futHist != nil {
		raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
			Metric:   domain.FuturesMetricOpenInterest,
			Exchange: string(ex),
			Symbol:   symbol,
			From:     from,
			To:       to,
			Limit:    domain.MaxFuturesHistoryLimit,
		})
		if err == nil {
			for _, rec := range futuresSnapshots(raw) {
				if rec.Exchange != ex || rec.Value <= 0 || rec.SampledAt.IsZero() {
					continue
				}
				pts = append(pts, domain.VolumeProfileOiPoint{Time: rec.SampledAt, Value: rec.Value})
			}
		}
	}
	seen := map[int64]struct{}{}
	out := make([]domain.VolumeProfileOiPoint, 0, len(pts))
	for _, p := range pts {
		ms := p.Time.UTC().UnixMilli()
		if _, ok := seen[ms]; ok {
			continue
		}
		seen[ms] = struct{}{}
		out = append(out, p)
	}
	return out
}

func (s *Service) profileLiquidations(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) []domain.LiquidationEvent {
	var ev []domain.LiquidationEvent
	if s.liq != nil {
		ev = append(ev, s.liq.EventsSince(string(ex), symbol, from)...)
	}
	if s.futHist != nil {
		raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
			Metric:   "liquidations",
			Exchange: string(ex),
			Symbol:   symbol,
			From:     from,
			To:       to,
			Limit:    domain.MaxFuturesHistoryLimit,
		})
		if err == nil {
			if rows, ok := raw.([]domain.LiquidationEvent); ok {
				ev = append(ev, rows...)
			}
		}
	}
	if len(ev) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]domain.LiquidationEvent, 0, len(ev))
	for _, e := range ev {
		if e.Exchange != "" && e.Exchange != ex {
			continue
		}
		e.Exchange = ex
		if !to.IsZero() && e.Time.After(to) {
			continue
		}
		key := domain.LiquidationEventKey(e)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, e)
	}
	return out
}
