package market

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// PerpDepthHistoryWorker samples market-wide perp bid/ask on a timer.
type PerpDepthHistoryWorker struct {
	Svc      *Service
	Interval time.Duration
	Retain   time.Duration
	Logger   *slog.Logger
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled.
func (w *PerpDepthHistoryWorker) Start(ctx context.Context) {
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
		w.Interval = time.Minute
	}
	if w.Retain <= 0 {
		w.Retain = domain.PerpDepthChartSpan
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("perp depth history worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce stores a fresh liquid-perp book print and drops old rows.
func (w *PerpDepthHistoryWorker) RunOnce(ctx context.Context) {
	if w == nil || w.Svc == nil {
		return
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	got, err := w.Svc.GetPerpDepthOverview(ctx, "all", 0, 0)
	if err != nil {
		w.Logger.Debug("perp depth history sample", "err", err)
	} else if got != nil && got.Known > 0 {
		w.Logger.Debug("perp depth history sample", "coins", got.Known)
	}
	if stored, gaps, berr := w.Svc.SamplePerpBookBands(ctx); berr != nil {
		w.Logger.Debug("perp book band sample", "err", berr)
	} else if stored > 0 || gaps > 0 {
		w.Logger.Debug("perp book band sample", "stored", stored, "gaps", gaps)
	}
	if w.Retain > 0 {
		if dropped, perr := w.Svc.PurgePerpDepthHistory(ctx, w.now().UTC().Add(-w.Retain)); perr != nil {
			w.Logger.Error("perp depth history purge", "err", perr)
		} else if dropped > 0 {
			w.Logger.Info("perp depth history purge", "dropped", dropped)
		}
	}
}
