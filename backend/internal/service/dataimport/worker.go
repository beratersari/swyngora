package dataimport

import (
	"context"
	"log/slog"
	"time"
)

// Worker claims pending imports and cleans expired uploads.
type Worker struct {
	Import       *Service
	Interval     time.Duration
	CleanupEvery time.Duration
	Logger       *slog.Logger
}

// Start runs until ctx is canceled.
func (w *Worker) Start(ctx context.Context) {
	if w.Import == nil {
		return
	}
	interval := w.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	cleanupEvery := w.CleanupEvery
	if cleanupEvery <= 0 {
		cleanupEvery = time.Minute
	}
	log := w.Logger
	if log == nil {
		log = slog.Default()
	}
	if n, err := w.Import.RequeueStuckRunning(ctx, 0); err != nil {
		log.Error("import requeue stuck", "err", err)
	} else if n > 0 {
		log.Info("import requeued stuck running jobs", "count", n)
	}
	tick := time.NewTicker(interval)
	defer tick.Stop()
	cleanupTick := time.NewTicker(cleanupEvery)
	defer cleanupTick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanupTick.C:
			if n, err := w.Import.CleanupExpired(ctx); err != nil {
				log.Error("import cleanup", "err", err)
			} else if n > 0 {
				log.Info("import cleaned expired jobs", "count", n)
			}
		case <-tick.C:
			if n, err := w.Import.ProcessPending(ctx); err != nil {
				log.Error("import process pending", "err", err)
			} else if n > 0 {
				log.Info("import processed jobs", "count", n)
			}
		}
	}
}
