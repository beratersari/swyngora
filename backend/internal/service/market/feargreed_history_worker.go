package market

import (
	"context"
	"log/slog"
	"time"
)

// FearGreedHistoryWorker samples the fear-and-greed score on a timer.
type FearGreedHistoryWorker struct {
	Svc      *Service
	Interval time.Duration
	Retain   time.Duration
	Logger   *slog.Logger
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled. The first sample is taken immediately.
func (w *FearGreedHistoryWorker) Start(ctx context.Context) {
	if w == nil || w.Svc == nil {
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
		w.Interval = 15 * time.Minute
	}
	if w.Retain <= 0 {
		w.Retain = 90 * 24 * time.Hour
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("fear greed history worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce stores a fresh score and drops old rows.
func (w *FearGreedHistoryWorker) RunOnce(ctx context.Context) {
	if w == nil || w.Svc == nil {
		return
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	if _, err := w.Svc.GetFearGreed(ctx); err != nil {
		w.Logger.Debug("fear greed history sample", "err", err)
	}
	if w.Retain > 0 {
		if dropped, err := w.Svc.PurgeFearGreedHistory(ctx, w.now().UTC().Add(-w.Retain)); err != nil {
			w.Logger.Error("fear greed history purge", "err", err)
		} else if dropped > 0 {
			w.Logger.Info("fear greed history purge", "dropped", dropped)
		}
	}
}
