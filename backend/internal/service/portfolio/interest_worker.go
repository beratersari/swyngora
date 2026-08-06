package portfolio

import (
	"context"
	"log/slog"
	"time"
)

// MarginInterestWorker is a persistent background task that accrues margin debt
// interest for open positions. Catch-up after downtime is O(1) per position
// (full hours elapsed × rate × principal), never hour-by-hour loops.
// Compare-and-swap on last_interest_at prevents double accrual across workers/restarts.
type MarginInterestWorker struct {
	Portfolio *Service
	Interval  time.Duration
	Logger    *slog.Logger
	now       func() time.Time
	sleep     func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled. Runs once immediately so restarts catch up offline hours.
func (w *MarginInterestWorker) Start(ctx context.Context) {
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
		w.Interval = time.Minute
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("margin interest worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce accrues interest for all open debts and liquidates if liq is breached after accrual.
func (w *MarginInterestWorker) RunOnce(ctx context.Context) {
	if w.Portfolio == nil {
		return
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	accrued, liquidated, err := w.Portfolio.ProcessMarginInterest(ctx, w.now().UTC())
	if err != nil {
		w.Logger.Error("margin interest process", "err", err)
		return
	}
	if accrued > 0 || liquidated > 0 {
		w.Logger.Info("margin interest tick", "accrued", accrued, "liquidated", liquidated)
	}
}
