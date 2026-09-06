package market

import (
	"context"
	"log/slog"
	"time"
)

// OptionsHistoryWorker samples BTC/ETH options OI, put/call, and IV on a timer.
type OptionsHistoryWorker struct {
	Svc      *Service
	Interval time.Duration
	Retain   time.Duration
	Logger   *slog.Logger
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled.
func (w *OptionsHistoryWorker) Start(ctx context.Context) {
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
		w.Interval = 5 * time.Minute
	}
	if w.Retain <= 0 {
		w.Retain = 14 * 24 * time.Hour
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("options history worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce stores a fresh print for BTC and ETH and drops old rows.
func (w *OptionsHistoryWorker) RunOnce(ctx context.Context) {
	if w == nil || w.Svc == nil {
		return
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	n, err := w.Svc.SampleOptionsHistory(ctx)
	if err != nil {
		w.Logger.Debug("options history sample", "inserted", n, "err", err)
	} else if n > 0 {
		w.Logger.Info("options history sample", "inserted", n)
	}
	if w.Retain > 0 {
		if dropped, perr := w.Svc.PurgeOptionsHistory(ctx, w.now().UTC().Add(-w.Retain)); perr != nil {
			w.Logger.Error("options history purge", "err", perr)
		} else if dropped > 0 {
			w.Logger.Info("options history purge", "dropped", dropped)
		}
	}
}
