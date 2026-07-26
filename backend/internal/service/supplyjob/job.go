package supplyjob

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/platform/schedule"
)

// Default retry backoffs after a failed refresh (before the next daily slot).
// First entry 0 = attempt immediately.
var defaultRetryBackoffs = []time.Duration{
	0,
	1 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
	1 * time.Hour,
}

// Runner periodically refreshes the supply/mcap snapshot into cache.
type Runner struct {
	Supply domain.SupplyPort
	Hour   int
	Minute int
	Loc    *time.Location
	// RunOnStart triggers one refresh shortly after Start (so the cache is warm before 03:00).
	RunOnStart bool
	Logger     *slog.Logger
	// RetryBackoffs overrides defaultRetryBackoffs (tests may inject short delays).
	RetryBackoffs []time.Duration
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
		r.runWithRetry(ctx, "startup")
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
		r.runWithRetry(ctx, "daily")
	}
}

// runWithRetry attempts a refresh with backoff so a single transient failure
// does not leave supply/mcap stale until the next calendar day.
func (r *Runner) runWithRetry(ctx context.Context, reason string) {
	backoffs := r.RetryBackoffs
	if len(backoffs) == 0 {
		backoffs = defaultRetryBackoffs
	}
	for i, wait := range backoffs {
		if ctx.Err() != nil {
			return
		}
		if wait > 0 {
			if err := r.sleep(ctx, wait); err != nil {
				return
			}
		}
		if r.runOnce(ctx, reason) {
			return
		}
		r.Logger.Info("supply refresh will retry",
			"reason", reason,
			"attempt", i+1,
			"maxAttempts", len(backoffs),
		)
	}
	r.Logger.Error("supply refresh exhausted retries", "reason", reason, "attempts", len(backoffs))
}

// runOnce performs one refresh. Returns true on success.
func (r *Runner) runOnce(ctx context.Context, reason string) bool {
	if r.Supply == nil {
		r.Logger.Error("supply refresh skipped: nil SupplyPort")
		return false
	}
	start := r.now()
	n, err := r.Supply.Refresh(ctx)
	if err != nil {
		r.Logger.Error("supply refresh failed", "reason", reason, "err", err, "stored", n)
		return false
	}
	r.Logger.Info("supply refresh completed",
		"reason", reason,
		"stored", n,
		"duration", r.now().Sub(start).String(),
	)
	return true
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
