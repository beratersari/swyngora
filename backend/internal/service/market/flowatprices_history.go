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
		s.watchMu.Unlock()
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
		w.Retain = 7 * 24 * time.Hour
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
	for _, sym := range w.Svc.FlowSampleSymbols() {
		for _, ex := range configuredFutures("all", func(v domain.Exchange) bool { return w.Svc.printPort(v) != nil }) {
			if ctx.Err() != nil {
				return
			}
			if err := w.Svc.SampleFlowAtPrices(ctx, ex, sym, now); err != nil {
				w.Logger.Debug("flow at prices sample", "exchange", ex, "symbol", sym, "err", err)
			}
		}
	}
	if w.Retain > 0 {
		if _, err := w.Svc.PurgeFlowAtPrices(ctx, now.Add(-w.Retain)); err != nil {
			w.Logger.Debug("flow at prices purge", "err", err)
		}
	}
}
