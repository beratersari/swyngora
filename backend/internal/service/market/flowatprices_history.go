package market

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const flowSeenCap = 32

func (s *Service) noteFlowSymbol(symbol string) {
	if s == nil || symbol == "" {
		return
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if s.flowSeen == nil {
		s.flowSeen = map[string]struct{}{}
	}
	if _, ok := s.flowSeen[symbol]; ok {
		return
	}
	if len(s.flowSeen) >= flowSeenCap {
		return
	}
	s.flowSeen[symbol] = struct{}{}
}

// FlowSampleSymbols is the futures pairs whose price ladder is stored.
func (s *Service) FlowSampleSymbols() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(sym string) {
		sym = domain.NormalizeLiquidationSymbol(sym)
		if sym == "" {
			return
		}
		if _, ok := seen[sym]; ok {
			return
		}
		seen[sym] = struct{}{}
		out = append(out, sym)
	}
	for _, sym := range domain.DefaultFuturesHistorySymbols {
		add(sym)
	}
	if s != nil {
		s.watchMu.Lock()
		for sym := range s.flowSeen {
			add(sym)
		}
		for _, sym := range s.flowLiquid {
			add(sym)
		}
		s.watchMu.Unlock()
	}
	return out
}

// SetFlowLiquid sets the extra pairs sampled about once a minute.
// Seed pairs and pairs someone opened stay on the faster pass.
func (s *Service) SetFlowLiquid(symbols []string) {
	if s == nil {
		return
	}
	cp := append([]string{}, symbols...)
	s.watchMu.Lock()
	s.flowLiquid = cp
	s.watchMu.Unlock()
}

// FlowSample is one pair on a recorder pass. Slow pairs are the liquid
// names outside the seed list.
type FlowSample struct {
	Symbol string
	Slow   bool
}

// FlowSamplePlan is the flow recorder pass. Seeds and opened pairs are fast.
// The wider liquid list is slow so a 10-second tick does not fetch every name.
func (s *Service) FlowSamplePlan() []FlowSample {
	fast := map[string]struct{}{}
	var out []FlowSample
	addFast := func(sym string) {
		sym = domain.NormalizeLiquidationSymbol(sym)
		if sym == "" {
			return
		}
		if _, ok := fast[sym]; ok {
			return
		}
		fast[sym] = struct{}{}
		out = append(out, FlowSample{Symbol: sym})
	}
	for _, sym := range domain.DefaultFuturesHistorySymbols {
		addFast(sym)
	}
	var liquid []string
	if s != nil {
		s.watchMu.Lock()
		for sym := range s.flowSeen {
			addFast(sym)
		}
		liquid = append(liquid, s.flowLiquid...)
		s.watchMu.Unlock()
	}
	for _, sym := range liquid {
		sym = domain.NormalizeLiquidationSymbol(sym)
		if sym == "" {
			continue
		}
		if _, ok := fast[sym]; ok {
			continue
		}
		fast[sym] = struct{}{}
		out = append(out, FlowSample{Symbol: sym, Slow: true})
	}
	return out
}

// SampleFlowAtPrices stores closed-minute buy/sell rows for one venue.
// Prints already stored are skipped. A gap longer than 20 minutes is not
// backfilled; collection continues from the recent tape.
func (s *Service) SampleFlowAtPrices(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) error {
	if s == nil || s.flowHist == nil {
		return nil
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	closed := now.Truncate(time.Minute)
	if closed.IsZero() {
		return nil
	}
	cursor, err := s.flowHist.FlowCursor(ctx, ex, symbol)
	if err != nil {
		return err
	}
	from := cursor
	if cursor.IsZero() || now.Sub(cursor) > 20*time.Minute {
		from = closed.Add(-15 * time.Minute)
	}
	prints := s.fetchPrintRange(ctx, ex, symbol, from, closed)
	var closedPrints []domain.TakerPrint
	for _, p := range prints {
		if p.Time.IsZero() || !p.Time.Before(closed) {
			continue
		}
		if !cursor.IsZero() && !p.Time.After(cursor) && now.Sub(cursor) <= 20*time.Minute {
			continue
		}
		p.Exchange = ex
		p.Symbol = symbol
		closedPrints = append(closedPrints, p)
	}
	if len(closedPrints) == 0 {
		return nil
	}
	cells, newest := domain.AccumulateFlowPrints(closedPrints, time.Time{})
	if err := s.flowHist.AddFlowCells(ctx, cells); err != nil {
		return err
	}
	return s.flowHist.SetFlowCursor(ctx, ex, symbol, newest)
}

// PurgeFlowAtPrices drops stored rows older than cutoff.
func (s *Service) PurgeFlowAtPrices(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.flowHist == nil {
		return 0, nil
	}
	return s.flowHist.PurgeFlowCells(ctx, cutoff)
}

func flowVenue(ex domain.Exchange, symbol string, prints []domain.TakerPrint, cells []domain.FlowAtPricesCell, tick, last, minPrice, maxPrice float64, from, to time.Time) domain.FlowAtPricesVenue {
	if len(cells) == 0 {
		return domain.BuildFlowAtPricesVenue(ex, symbol, prints, tick, last, minPrice, maxPrice, from, to)
	}
	base := domain.BuildFlowAtPricesFromCells(ex, symbol, cells, tick, last, minPrice, maxPrice, from, to)
	latest := domain.LatestFlowMinute(cells)
	var tail []domain.TakerPrint
	for _, p := range prints {
		if latest.IsZero() || !p.Time.Before(latest.Add(time.Minute)) {
			tail = append(tail, p)
		}
	}
	if len(tail) == 0 || base.Error != "" && len(base.Levels) == 0 {
		if base.Error != "" {
			return domain.BuildFlowAtPricesVenue(ex, symbol, prints, tick, last, minPrice, maxPrice, from, to)
		}
		return base
	}
	extra := domain.BuildFlowAtPricesVenue(ex, symbol, tail, base.TickSize, last, minPrice, maxPrice, from, to)
	if extra.Error != "" || len(extra.Levels) == 0 {
		return base
	}
	merged := domain.CombineFlowAtPricesVenues(symbol, []domain.FlowAtPricesVenue{base, extra}, base.TickSize, last, from, to)
	if merged == nil {
		return base
	}
	merged.Exchange = ex
	if !base.KnownFrom.IsZero() && (merged.KnownFrom.IsZero() || base.KnownFrom.Before(merged.KnownFrom)) {
		merged.KnownFrom = base.KnownFrom
	}
	merged.Complete = domain.FlowRangeCovered(merged.KnownFrom, from, to.Sub(from))
	merged.Truncated = !merged.Complete
	merged.Summary = domain.ExplainFlowAtPricesVenue(*merged)
	return *merged
}

// FlowAtPricesHistoryWorker samples Binance and Bybit prints into minute rows.
type FlowAtPricesHistoryWorker struct {
	Svc      *Service
	Interval time.Duration
	Retain   time.Duration
	Logger   *slog.Logger
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
	lastSlow time.Time
}

// Start runs until ctx is canceled.
func (w *FlowAtPricesHistoryWorker) Start(ctx context.Context) {
	if w == nil || w.Svc == nil || w.Svc.flowHist == nil {
		return
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	if w.sleep == nil {
		w.sleep = func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	if w.Interval <= 0 {
		w.Interval = 10 * time.Second
	}
	if w.Retain <= 0 {
		w.Retain = 30 * 24 * time.Hour
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("flow at prices history worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce stores one pass and drops rows older than the retention window.
func (w *FlowAtPricesHistoryWorker) RunOnce(ctx context.Context) {
	if w == nil || w.Svc == nil {
		return
	}
	if w.now == nil {
		w.now = time.Now
	}
	now := w.now().UTC()
	doSlow := w.lastSlow.IsZero() || now.Sub(w.lastSlow) >= time.Minute
	for _, item := range w.Svc.FlowSamplePlan() {
		if item.Slow && !doSlow {
			continue
		}
		for _, ex := range configuredFutures("all", func(v domain.Exchange) bool { return w.Svc.printPort(v) != nil }) {
			if ctx.Err() != nil {
				return
			}
			if err := w.Svc.SampleFlowAtPrices(ctx, ex, item.Symbol, now); err != nil {
				w.Logger.Debug("flow at prices sample", "exchange", ex, "symbol", item.Symbol, "err", err)
			}
		}
	}
	if doSlow {
		w.lastSlow = now
	}
	if w.Retain > 0 {
		if _, err := w.Svc.PurgeFlowAtPrices(ctx, now.Add(-w.Retain)); err != nil {
			w.Logger.Debug("flow at prices purge", "err", err)
		}
	}
}
