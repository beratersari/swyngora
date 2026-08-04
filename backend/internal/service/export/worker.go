package export

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Worker claims pending export jobs, builds files, and cleans expired downloads.
type Worker struct {
	Export   *Service
	Interval time.Duration
	Logger   *slog.Logger
	// CleanupEvery controls how often expired files are removed (default 1m).
	CleanupEvery time.Duration
}

// Start runs until ctx is canceled.
func (w *Worker) Start(ctx context.Context) {
	if w.Export == nil {
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

	// Recover jobs left running after restart.
	if n, err := w.Export.RequeueStuckRunning(ctx, 0); err != nil {
		log.Error("export requeue stuck", "err", err)
	} else if n > 0 {
		log.Info("export requeued stuck running jobs", "count", n)
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
			if n, err := w.Export.CleanupExpired(ctx); err != nil {
				log.Error("export cleanup", "err", err)
			} else if n > 0 {
				log.Info("export cleaned expired files", "count", n)
			}
		case <-tick.C:
			w.drainOnce(ctx, log)
		}
	}
}

func (w *Worker) drainOnce(ctx context.Context, log *slog.Logger) {
	if n, err := w.Export.ProcessPending(ctx); err != nil {
		log.Error("export process pending", "err", err)
	} else if n > 0 {
		log.Info("export processed jobs", "count", n)
	}
}

// ProcessPending claims and runs all currently pending export jobs (one pass).
// Used by the worker and tests.
func (s *Service) ProcessPending(ctx context.Context) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	pending, err := s.store.ListPending(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range pending {
		job := &pending[i]
		ok, err := s.store.Claim(ctx, job.ID, time.Now().UTC())
		if err != nil {
			return n, err
		}
		if !ok {
			continue
		}
		claimed, err := s.store.GetByID(ctx, job.ID)
		if err != nil {
			continue
		}
		if err := s.runJob(ctx, claimed); err != nil {
			_ = s.store.Finish(ctx, job.ID, domain.ExportFailed, "", "", 0, nil, err.Error(), time.Now().UTC())
		}
		n++
	}
	return n, nil
}
