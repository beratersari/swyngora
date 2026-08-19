package futureshist

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Worker periodically snapshots OI, funding, and long/short per venue.
type Worker struct {
	Hist     *Service
	Interval time.Duration
	Retain   time.Duration
	Logger   *slog.Logger
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled.
func (w *Worker) Start(ctx context.Context) {
	if w == nil || w.Hist == nil {
		return
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	if w.sleep == nil {
		w.sleep = sleepCtx
	}
	if w.Interval <= 0 {
		w.Interval = 5 * time.Minute
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("futures history worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce samples every seed/seen symbol on each venue independently.
func (w *Worker) RunOnce(ctx context.Context) {
	if w == nil || w.Hist == nil {
		return
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	now := w.now().UTC()
	inserted := 0
	failed := 0
	for _, sym := range w.Hist.Symbols() {
		for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit} {
			if ctx.Err() != nil {
				return
			}
			n, err := w.Hist.SaveSymbol(ctx, ex, sym, now)
			inserted += n
			if err != nil {
				failed++
				w.Logger.Debug("futures history venue skip", "exchange", ex, "symbol", sym, "err", err)
			}
		}
	}
	if w.Retain > 0 && w.Hist.Store != nil {
		if n1, n2, err := w.Hist.Store.PurgeOlderThan(ctx, now.Add(-w.Retain)); err != nil {
			w.Logger.Error("futures history purge", "err", err)
		} else if n1+n2 > 0 {
			w.Logger.Info("futures history purge", "snapshots", n1, "liquidations", n2)
		}
	}
	if w.Retain > 0 && w.Hist != nil && w.Hist.TakerStore != nil {
		if n, err := w.Hist.TakerStore.PurgeTakerBuckets(ctx, now.Add(-w.Retain)); err != nil {
			w.Logger.Error("taker bucket purge", "err", err)
		} else if n > 0 {
			w.Logger.Info("taker bucket purge", "buckets", n)
		}
	}
	if inserted > 0 || failed > 0 {
		w.Logger.Info("futures history tick", "inserted", inserted, "venue_errors", failed, "symbols", len(w.Hist.Symbols()))
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
