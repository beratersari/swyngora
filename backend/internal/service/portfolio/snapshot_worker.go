package portfolio

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SnapshotWorker records mark-to-market equity buckets for performance charts.
type SnapshotWorker struct {
	Portfolio *Service
	Interval  time.Duration
	Retention time.Duration
	Logger    *slog.Logger
	now       func() time.Time
	sleep     func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled.
func (w *SnapshotWorker) Start(ctx context.Context) {
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
		w.Interval = domain.DefaultSnapshotInterval
	}
	if w.Retention <= 0 {
		w.Retention = domain.DefaultSnapshotRetention
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("portfolio snapshot worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce snapshots all portfolios and prunes expired buckets.
func (w *SnapshotWorker) RunOnce(ctx context.Context) {
	if w.Portfolio == nil {
		return
	}
	if w.now == nil {
		w.now = time.Now
	}
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	now := w.now().UTC()
	n, err := w.Portfolio.SnapshotAll(ctx, now)
	if err != nil {
		w.Logger.Error("portfolio equity snapshot", "err", err, "ok", n)
	} else if n > 0 {
		w.Logger.Debug("portfolio equity snapshot", "count", n)
	}
	ret := w.Retention
	if ret <= 0 {
		ret = domain.DefaultSnapshotRetention
	}
	if deleted, err := w.Portfolio.PruneSnapshots(ctx, now.Add(-ret)); err != nil {
		w.Logger.Error("portfolio equity snapshot prune", "err", err)
	} else if deleted > 0 {
		w.Logger.Info("portfolio equity snapshot prune", "deleted", deleted)
	}
}
