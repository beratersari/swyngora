package delistjob

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Runner periodically fetches Binance spot delist schedule into an in-memory store.
type Runner struct {
	Source   domain.SpotDelistSchedulePort
	Store    domain.SpotDelistStore
	// Interval between successful schedule refreshes (default 1h).
	Interval time.Duration
	// RunOnStart triggers one fetch immediately when Start runs.
	RunOnStart bool
	Logger     *slog.Logger
	// Exchange labeled on stored rows (always binance for this job).
	Exchange domain.Exchange

	now   func() time.Time
	sleep func(context.Context, time.Duration) error
}

// Start runs until ctx is cancelled. Blocking — call in a goroutine.
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
	if r.Interval <= 0 {
		r.Interval = time.Hour
	}
	if r.Exchange == "" {
		r.Exchange = domain.ExchangeBinance
	}
	if r.Source == nil || r.Store == nil {
		r.Logger.Warn("delist refresh disabled (missing source or store)")
		return
	}

	if r.RunOnStart {
		r.runOnce(ctx, "startup")
	}

	for {
		if err := r.sleep(ctx, r.Interval); err != nil {
			r.Logger.Info("delist refresh scheduler stopped", "err", err)
			return
		}
		r.runOnce(ctx, "interval")
	}
}

func (r *Runner) runOnce(ctx context.Context, reason string) {
	start := r.now()
	entries, err := r.Source.FetchSpotDelistSchedule(ctx)
	if err != nil {
		r.Logger.Warn("delist refresh failed", "reason", reason, "err", err)
		return
	}
	// Normalize exchange on entries.
	for i := range entries {
		if entries[i].Exchange == "" {
			entries[i].Exchange = r.Exchange
		}
	}
	r.Store.ReplaceAll(r.Exchange, entries)
	r.Logger.Info("delist refresh completed",
		"reason", reason,
		"exchange", r.Exchange,
		"symbols", len(entries),
		"duration", r.now().Sub(start).String(),
	)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
