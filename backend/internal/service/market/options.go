package market

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetOptions returns the BTC/ETH options desk snapshot (Deribit).
func (s *Service) GetOptions(ctx context.Context, exchange, asset string) (*domain.OptionsSnapshot, error) {
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOptionsExchange(exchange)
	if err != nil {
		return nil, err
	}
	if s == nil || s.options == nil {
		return nil, fmt.Errorf("%w: options source not configured", domain.ErrUpstream)
	}
	quotes, err := s.options.GetOptionQuotes(ctx, asset)
	if err != nil {
		return nil, err
	}
	var (
		dvol  domain.OptionsDVOL
		index float64
		wg    sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		got, err := s.options.GetOptionsDVOL(ctx, asset)
		if err == nil {
			dvol = got
		}
	}()
	go func() {
		defer wg.Done()
		got, err := s.options.GetOptionsIndex(ctx, asset)
		if err == nil {
			index = got
		}
	}()
	wg.Wait()
	snap := domain.BuildOptionsSnapshot(asset, quotes, index, dvol, time.Now().UTC())
	snap.Exchange = ex
	s.persistOptionsSample(ctx, snap)
	bars := s.optionsSpotBars(ctx, asset)
	domain.AttachOptionsVolCompare(snap, bars)
	s.attachOptionsHistory(ctx, snap, bars)
	s.attachOptionsStrikeOI(ctx, snap)
	if tape := s.optionsMarketTape(ctx, asset); tape.Usable() {
		snap.VsMarket = domain.CompareOptionsToMarket(snap, tape)
	}
	return snap, nil
}

func (s *Service) persistOptionsSample(ctx context.Context, snap *domain.OptionsSnapshot) {
	if s == nil || s.optionsHist == nil || snap == nil {
		return
	}
	rec, ok := domain.OptionsSampleFromSnapshot(snap)
	if !ok {
		return
	}
	_, _ = s.optionsHist.InsertSample(ctx, rec)
	_ = s.optionsHist.ReplaceStrikeBook(ctx, snap.Asset, rec.SampledAt, domain.OptionsStrikePointsFromSnapshot(snap))
}

func (s *Service) attachOptionsStrikeOI(ctx context.Context, snap *domain.OptionsSnapshot) {
	if s == nil || snap == nil {
		return
	}
	now := snap.AsOf.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	books := map[string]domain.OptionsStrikeBook{}
	if s.optionsHist != nil {
		for _, spec := range domain.OptionsHistoryWindows {
			target := now.Add(-spec.Dur)
			book, ok, err := s.optionsHist.StrikeBookAt(ctx, snap.Asset, target)
			if err != nil || !ok {
				continue
			}
			books[spec.ID] = book
		}
	}
	snap.StrikeOI = domain.BuildOptionsStrikeOI(snap, books, now)
}

func (s *Service) attachOptionsHistory(ctx context.Context, snap *domain.OptionsSnapshot, bars []domain.OHLCBar) {
	if s == nil || snap == nil {
		return
	}
	now := snap.AsOf.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	from := now.Add(-domain.OptionsHistoryLookback)
	var samples []domain.OptionsSample
	if s.optionsHist != nil {
		if got, err := s.optionsHist.ListSamples(ctx, snap.Asset, from, now.Add(time.Minute), domain.OptionsHistoryListLimit); err == nil {
			samples = got
		}
	}
	var dvol []domain.OptionsDVOLPoint
	if p, ok := s.options.(domain.OptionsDVOLHistoryPort); ok && p != nil {
		if got, err := p.GetOptionsDVOLHistory(ctx, snap.Asset, from, now); err == nil {
			dvol = got
		}
	}
	snap.History = domain.BuildOptionsHistory(snap, samples, dvol, bars, now)
}

func (s *Service) optionsSpotBars(ctx context.Context, asset string) []domain.OHLCBar {
	if s == nil {
		return nil
	}
	symbol := domain.OptionsSpotSymbol(asset)
	c, err := s.GetCandles(ctx, string(domain.DefaultExchange), symbol, string(domain.Interval1d), domain.OptionsRealizedCandleLimit, nil, nil)
	if err != nil {
		return nil
	}
	return domain.BarsFromCandles(c)
}

// SampleOptionsHistory records BTC and ETH desk prints and purges stale rows.
func (s *Service) SampleOptionsHistory(ctx context.Context) (inserted int, err error) {
	if s == nil || s.options == nil {
		return 0, fmt.Errorf("%w: options source not configured", domain.ErrUpstream)
	}
	var last error
	for _, asset := range []string{domain.OptionsAssetBTC, domain.OptionsAssetETH} {
		if ctx.Err() != nil {
			return inserted, ctx.Err()
		}
		quotes, qerr := s.options.GetOptionQuotes(ctx, asset)
		if qerr != nil {
			last = qerr
			continue
		}
		var dvol domain.OptionsDVOL
		var index float64
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			got, err := s.options.GetOptionsDVOL(ctx, asset)
			if err == nil {
				dvol = got
			}
		}()
		go func() {
			defer wg.Done()
			got, err := s.options.GetOptionsIndex(ctx, asset)
			if err == nil {
				index = got
			}
		}()
		wg.Wait()
		snap := domain.BuildOptionsSnapshot(asset, quotes, index, dvol, time.Now().UTC())
		if rec, ok := domain.OptionsSampleFromSnapshot(snap); ok && s.optionsHist != nil {
			if ins, ierr := s.optionsHist.InsertSample(ctx, rec); ierr != nil {
				last = ierr
			} else if ins {
				inserted++
			}
			if ierr := s.optionsHist.ReplaceStrikeBook(ctx, asset, rec.SampledAt, domain.OptionsStrikePointsFromSnapshot(snap)); ierr != nil {
				last = ierr
			}
		}
	}
	return inserted, last
}

// PurgeOptionsHistory drops samples older than cutoff.
func (s *Service) PurgeOptionsHistory(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.optionsHist == nil {
		return 0, nil
	}
	return s.optionsHist.PurgeOlderThan(ctx, cutoff)
}

func (s *Service) optionsMarketTape(ctx context.Context, asset string) domain.OptionsMarketTape {
	var tape domain.OptionsMarketTape
	if s == nil {
		return tape
	}
	symbol := domain.OptionsSpotSymbol(asset)
	var (
		tkr  *domain.Ticker24h
		bars []domain.OHLCBar
		fund *domain.FundingSnapshot
		ls   *domain.LongShortSnapshot
		wg   sync.WaitGroup
	)
	wg.Add(4)
	go func() {
		defer wg.Done()
		tkr, _ = s.GetTicker24h(ctx, string(domain.DefaultExchange), symbol)
	}()
	go func() {
		defer wg.Done()
		if c, err := s.GetCandles(ctx, string(domain.DefaultExchange), symbol, "1h", 50, nil, nil); err == nil {
			bars = domain.BarsFromCandles(c)
		}
	}()
	go func() {
		defer wg.Done()
		fund, _ = s.GetFundingRate(ctx, "all", symbol, domain.DefaultFundingHistoryLimit)
	}()
	go func() {
		defer wg.Done()
		ls, _ = s.GetLongShortRatio(ctx, "all", symbol, domain.DefaultLongShortHistoryLimit)
	}()
	wg.Wait()

	if tkr != nil {
		if last, err := strconv.ParseFloat(tkr.LastPrice, 64); err == nil && last > 0 {
			tape.SpotLast = last
			tape.HasSpot = true
		}
		hi, e1 := strconv.ParseFloat(tkr.HighPrice, 64)
		lo, e2 := strconv.ParseFloat(tkr.LowPrice, 64)
		if e1 == nil && e2 == nil && tape.SpotLast > 0 && hi > lo && lo > 0 {
			tape.Range24hPct = (hi - lo) / tape.SpotLast * 100
			tape.HasRange = tape.Range24hPct > 0
		}
	}
	if tape.SpotLast <= 0 {
		if last := lastSnapshotClose(bars); last > 0 {
			tape.SpotLast = last
			tape.HasSpot = true
		}
	}
	now := time.Now().UTC()
	cur := barsInWindow(bars, now.Add(-24*time.Hour), now)
	prev := barsInWindow(bars, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	if !tape.HasRange && tape.SpotLast > 0 && len(cur) >= 12 {
		if r := rangePctOf(cur, tape.SpotLast); r > 0 {
			tape.Range24hPct = r
			tape.HasRange = true
		}
	}
	if tape.SpotLast > 0 && len(prev) >= 12 {
		if r := rangePctOf(prev, tape.SpotLast); r > 0 {
			tape.PrevRange24hPct = r
			tape.HasPrevRange = true
		}
	}
	if rate, ok := fundingRateFromSnap(fund); ok {
		tape.FundingRate = rate
		tape.HasFunding = true
	}
	if pct, ok := longPctFromSnap(ls); ok {
		tape.LongPct = pct
		tape.HasLongPct = true
	}
	return tape
}

func barsInWindow(bars []domain.OHLCBar, from, to time.Time) []domain.OHLCBar {
	out := make([]domain.OHLCBar, 0, len(bars))
	for _, b := range bars {
		if b.Time.Before(from) || !b.Time.Before(to) {
			continue
		}
		out = append(out, b)
	}
	return out
}

func rangePctOf(bars []domain.OHLCBar, last float64) float64 {
	if last <= 0 || len(bars) == 0 {
		return 0
	}
	hi, lo := 0.0, 0.0
	for i, b := range bars {
		if i == 0 || b.High > hi {
			hi = b.High
		}
		if (i == 0 || b.Low < lo) && b.Low > 0 {
			lo = b.Low
		}
	}
	if hi <= lo || lo <= 0 {
		return 0
	}
	return (hi - lo) / last * 100
}

func fundingRateFromSnap(s *domain.FundingSnapshot) (float64, bool) {
	if s == nil {
		return 0, false
	}
	if s.Current != nil {
		if v, err := strconv.ParseFloat(s.Current.Rate, 64); err == nil {
			return v, true
		}
	}
	var sum float64
	n := 0
	for _, v := range s.Venues {
		if r, err := strconv.ParseFloat(v.Current.Rate, 64); err == nil {
			sum += r
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

func longPctFromSnap(s *domain.LongShortSnapshot) (float64, bool) {
	if s == nil {
		return 0, false
	}
	if s.Current != nil {
		if v, err := strconv.ParseFloat(s.Current.LongPct, 64); err == nil && v > 0 {
			return v, true
		}
	}
	var sum float64
	n := 0
	for _, v := range s.Venues {
		if r, err := strconv.ParseFloat(v.Current.LongPct, 64); err == nil && r > 0 {
			sum += r
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}
