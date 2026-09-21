package market

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const fearGreedCacheTTL = 2 * time.Minute

// GetFearGreed scores the market from the tape this process already collects
// and keeps the result so later reads can show how the score moved.
func (s *Service) GetFearGreed(ctx context.Context) (*domain.FearGreedReport, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	if rep, ok := s.cachedFearGreed(time.Now()); ok {
		return rep, nil
	}
	v, err, _ := s.fearGreedSF.Do("fear-greed", func() (any, error) {
		if rep, ok := s.cachedFearGreed(time.Now()); ok {
			return rep, nil
		}
		return s.computeFearGreed(ctx)
	})
	if err != nil {
		return nil, err
	}
	rep, _ := v.(*domain.FearGreedReport)
	if rep == nil {
		return nil, fmt.Errorf("%w: fear and greed", domain.ErrUpstream)
	}
	return rep, nil
}

func (s *Service) computeFearGreed(ctx context.Context) (*domain.FearGreedReport, error) {
	in := s.collectFearGreedInputs(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rep := domain.BuildFearGreed(in, now)
	if rep.Used < domain.FearGreedMinFactors {
		return nil, fmt.Errorf("%w: fear and greed has %d of %d inputs", domain.ErrUpstream, rep.Used, domain.FearGreedFactorCount)
	}
	s.rememberFearGreed(ctx, rep)
	s.storeFearGreedCache(rep, now)
	return rep, nil
}

func (s *Service) rememberFearGreed(ctx context.Context, rep *domain.FearGreedReport) {
	if s == nil || s.fearGreed == nil || rep == nil || rep.Used < domain.FearGreedMinFactors {
		return
	}
	topID := ""
	var topPull float64
	if rep.Top != nil {
		topID = rep.Top.ID
		topPull = rep.Top.Pull
	}
	sample := domain.FearGreedSample{
		Time:    domain.FearGreedSampleTime(rep.AsOf),
		Score:   rep.Score,
		Level:   rep.Level,
		TopID:   topID,
		TopPull: topPull,
	}
	_ = s.fearGreed.SaveFearGreed(ctx, sample)
	since := rep.AsOf.Add(-(domain.FearGreedHistoryKeep + 24*time.Hour))
	rows, err := s.fearGreed.ListFearGreed(ctx, since)
	if err != nil {
		return
	}
	domain.AttachFearGreedHistory(rep, rows)
}

func (s *Service) cachedFearGreed(now time.Time) (*domain.FearGreedReport, bool) {
	if s == nil {
		return nil, false
	}
	s.fearGreedMu.Lock()
	defer s.fearGreedMu.Unlock()
	if s.fearGreedHit == nil || now.Sub(s.fearGreedAt) > fearGreedCacheTTL {
		return nil, false
	}
	return s.fearGreedHit, true
}

func (s *Service) storeFearGreedCache(rep *domain.FearGreedReport, now time.Time) {
	if s == nil || rep == nil {
		return
	}
	s.fearGreedMu.Lock()
	s.fearGreedHit = rep
	s.fearGreedAt = now
	s.fearGreedMu.Unlock()
}

// PurgeFearGreedHistory drops stored scores older than the cutoff.
func (s *Service) PurgeFearGreedHistory(ctx context.Context, before time.Time) (int, error) {
	if s == nil || s.fearGreed == nil {
		return 0, nil
	}
	return s.fearGreed.PurgeFearGreedBefore(ctx, before)
}

func (s *Service) collectFearGreedInputs(ctx context.Context) domain.FearGreedInputs {
	var (
		breadth *domain.BreadthReport
		vol     *domain.VolatilityReport
		fund    *domain.FundingSnapshot
		ls      *domain.LongShortSnapshot
		liq     *domain.LiquidationOverview
		book    *domain.PerpDepthOverview
		taker   *domain.TakerFlowReport
		wg      sync.WaitGroup
	)
	wg.Add(7)
	go func() {
		defer wg.Done()
		breadth, _ = s.GetBreadth(ctx, string(domain.ExchangeBinance), domain.ParseBreadthLimit(0))
	}()
	go func() {
		defer wg.Done()
		vol, _ = s.GetVolatility(ctx, string(domain.ExchangeBinance), "BTCUSDT")
	}()
	go func() {
		defer wg.Done()
		fund, _ = s.GetFundingRate(ctx, "all", "BTCUSDT", 1)
	}()
	go func() {
		defer wg.Done()
		ls, _ = s.GetLongShortRatio(ctx, string(domain.ExchangeBinance), "BTCUSDT", 1)
	}()
	go func() {
		defer wg.Done()
		liq, _ = s.GetLiquidationOverview(ctx, "all", domain.LiquidationWindow24h, 5)
	}()
	go func() {
		defer wg.Done()
		book, _ = s.GetPerpDepthOverview(ctx, "all", 1, 20)
	}()
	go func() {
		defer wg.Done()
		taker, _ = s.GetTakerFlow(ctx, "all", "BTCUSDT")
	}()
	wg.Wait()
	return fearGreedInputsFrom(breadth, vol, fund, ls, liq, book, taker)
}

func fearGreedInputsFrom(
	breadth *domain.BreadthReport,
	vol *domain.VolatilityReport,
	fund *domain.FundingSnapshot,
	ls *domain.LongShortSnapshot,
	liq *domain.LiquidationOverview,
	book *domain.PerpDepthOverview,
	taker *domain.TakerFlowReport,
) domain.FearGreedInputs {
	var in domain.FearGreedInputs
	if breadth != nil {
		for _, w := range breadth.Windows {
			if w.Window != domain.BreadthWindow24h {
				continue
			}
			if w.Counts.Total >= 10 {
				in.HasBreadth = true
				in.BreadthUpPct = w.Counts.UpPct
			}
			if w.BTC != nil && w.BTC.Known {
				in.HasMomentum = true
				in.BTCChangePct = w.BTC.ChangePct
			}
		}
	}
	if vol != nil {
		for _, w := range vol.Windows {
			if w.Window != domain.VolWindow24h || w.TypicalPct <= 0 || w.Coin.RangePct <= 0 {
				continue
			}
			in.HasVol = true
			in.VolRatio = w.Coin.RangePct / w.TypicalPct
			if !in.HasMomentum && w.Coin.Complete {
				in.HasMomentum = true
				in.BTCChangePct = w.Coin.NetPct
			}
		}
	}
	if fund != nil && fund.Pressure != nil && fund.Pressure.RatePct != "" {
		if v, ok := parseFearGreedFloat(fund.Pressure.RatePct); ok {
			in.HasFunding = true
			in.FundingRatePct = v
			in.FundingPayer = fund.Pressure.Payer
		}
	}
	if ls != nil && ls.Current != nil {
		if v, ok := parseFearGreedFloat(ls.Current.LongPct); ok {
			in.HasLong = true
			in.LongPct = v
		}
	}
	if liq != nil {
		for _, w := range liq.Windows {
			if w.Window != domain.LiquidationWindow24h {
				continue
			}
			if !w.Complete {
				continue
			}
			longN, lok := parseFearGreedFloat(w.LongNotional)
			shortN, sok := parseFearGreedFloat(w.ShortNotional)
			if (lok || sok) && longN+shortN > 0 {
				in.HasLiq = true
				in.LiqLong = longN
				in.LiqShort = shortN
			}
		}
	}
	if book != nil && book.Known > 0 && book.Total > 0 {
		in.HasBook = true
		in.BidPct = book.BidPct
	}
	if flow, win := fearGreedTaker(taker); flow != nil {
		in.HasTaker = true
		in.TakerBuyPct = flow.BuyShare * 100
		in.TakerWindow = win
	}
	return in
}

func fearGreedTaker(rep *domain.TakerFlowReport) (*domain.TakerWindowFlow, string) {
	if rep == nil {
		return nil, ""
	}
	src := rep.Combined
	if src == nil && len(rep.Venues) > 0 {
		src = &rep.Venues[0]
	}
	if src == nil {
		return nil, ""
	}
	prefer := []string{domain.TakerWindow1h, domain.TakerWindow4h, domain.TakerWindow5m}
	for _, id := range prefer {
		for i := range src.Windows {
			w := &src.Windows[i]
			if w.Window == id && w.BuyNotional+w.SellNotional > 0 {
				return w, id
			}
		}
	}
	return nil, ""
}

func parseFearGreedFloat(raw string) (float64, bool) {
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
