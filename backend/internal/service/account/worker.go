package account

import (
	"context"
	"log/slog"
	"time"
)

// PurgeWorker purges closed accounts after the grace period.
type PurgeWorker struct {
	Accounts *Service
	Interval time.Duration
	Logger   *slog.Logger
}

// Start runs until ctx is canceled.
func (w *PurgeWorker) Start(ctx context.Context) {
	if w.Accounts == nil {
		return
	}
	interval := w.Interval
	if interval <= 0 {
		interval = time.Hour
	}
	log := w.Logger
	if log == nil {
		log = slog.Default()
	}
	// Run once soon after start.
	if n, err := w.Accounts.PurgeDue(ctx); err != nil {
		log.Error("account purge", "err", err)
	} else if n > 0 {
		log.Info("account purge completed", "count", n)
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n, err := w.Accounts.PurgeDue(ctx); err != nil {
				log.Error("account purge", "err", err)
			} else if n > 0 {
				log.Info("account purge completed", "count", n)
			}
		}
	}
}
