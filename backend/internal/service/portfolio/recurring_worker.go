package portfolio

import (
	"context"
	"log/slog"
	"time"
)

// RecurringBuyWorker executes due paper recurring buy plans.
type RecurringBuyWorker struct {
	Portfolio *Service
	Interval  time.Duration
	Logger    *slog.Logger
	now       func() time.Time
	sleep     func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled.
func (w *RecurringBuyWorker) Start(ctx context.Context) {
	if w.Portfolio == nil {
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
		w.Interval = 30 * time.Second
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("recurring buy worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce processes all currently due plans.
func (w *RecurringBuyWorker) RunOnce(ctx context.Context) {
	if w.Portfolio == nil {
		return
	}
	if w.now == nil {
		w.now = time.Now
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	n, err := w.Portfolio.ProcessDueRecurringBuys(ctx, w.now().UTC())
	if err != nil {
		w.Logger.Error("recurring buy process", "err", err)
		return
	}
	if n > 0 {
		w.Logger.Info("recurring buy processed", "count", n)
	}
}
