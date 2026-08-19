package bookhist

import (
	"context"
	"log/slog"
	"time"
)

// Worker periodically snapshots live spot books.
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
		w.Interval = time.Minute
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("order book history worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce samples every seed/seen pair on each venue independently.
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
	jobs := w.Hist.Jobs()
	for _, job := range jobs {
		if ctx.Err() != nil {
			return
		}
		ok, err := w.Hist.SaveSymbol(ctx, job.ex, job.symbol, now)
		if err != nil {
			failed++
			w.Logger.Debug("order book history venue skip", "exchange", job.ex, "symbol", job.symbol, "err", err)
			continue
		}
		if ok {
			inserted++
		}
	}
	if w.Retain > 0 && w.Hist.Store != nil {
		if n, err := w.Hist.Store.PurgeOlderThan(ctx, now.Add(-w.Retain)); err != nil {
			w.Logger.Error("order book history purge", "err", err)
		} else if n > 0 {
			w.Logger.Info("order book history purge", "snapshots", n)
		}
	}
	if inserted > 0 || failed > 0 {
		w.Logger.Info("order book history tick", "inserted", inserted, "venue_errors", failed, "jobs", len(jobs))
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
