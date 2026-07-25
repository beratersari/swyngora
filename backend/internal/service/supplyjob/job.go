package supplyjob

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/schedule"
)

// Runner periodically refreshes the supply/mcap snapshot into cache.
type Runner struct {
	Supply domain.SupplyPort
	Hour   int
	Minute int
	Loc    *time.Location
	// RunOnStart triggers one refresh shortly after Start (so the cache is warm before 03:00).
	RunOnStart bool
	Logger     *slog.Logger
	// now is injectable for tests.
	now func() time.Time
	// sleep is injectable for tests.
	sleep func(context.Context, time.Duration) error
}

// Start runs the daily loop until ctx is cancelled. Blocking — call in a goroutine.
func (r *Runner) Start(ctx context.Context) {
	if r.Logger == nil {
		r.Logger = slog.Default()
	}
	if r.now == nil {
		r.now = time.Now
	}
	if r.sleep == nil {
		r.sleep = sleepCtx
	}
	if r.Loc == nil {
		r.Loc = time.UTC
	}

	if r.RunOnStart {
		r.runOnce(ctx, "startup")
	}

	for {
		now := r.now()
		wait := schedule.DurationUntilDaily(now, r.Hour, r.Minute, r.Loc)
		next := schedule.NextDailyAt(now, r.Hour, r.Minute, r.Loc)
		r.Logger.Info("supply refresh scheduled",
			"next", next.Format(time.RFC3339),
			"wait", wait.String(),
		)
		if err := r.sleep(ctx, wait); err != nil {
			r.Logger.Info("supply refresh scheduler stopped", "err", err)
			return
		}
		r.runOnce(ctx, "daily")
	}
}

func (r *Runner) runOnce(ctx context.Context, reason string) {
	if r.Supply == nil {
		r.Logger.Error("supply refresh skipped: nil SupplyPort")
		return
	}
	start := r.now()
	n, err := r.Supply.Refresh(ctx)
	if err != nil {
		r.Logger.Error("supply refresh failed", "reason", reason, "err", err, "stored", n)
		return
	}
	r.Logger.Info("supply refresh completed",
		"reason", reason,
		"stored", n,
		"duration", r.now().Sub(start).String(),
	)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
