package scanner

import (
	"context"
	"log/slog"
	"time"
)

// Checker periodically evaluates scanner rules against watchlists.
type Checker struct {
	Scanner  *Service
	Interval time.Duration
	Logger   *slog.Logger
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
}

// Start runs until ctx is cancelled.
func (c *Checker) Start(ctx context.Context) {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	if c.now == nil {
		c.now = time.Now
	}
	if c.sleep == nil {
		c.sleep = sleepCtx
	}
	if c.Interval <= 0 {
		c.Interval = 60 * time.Second
	}
	c.RunOnce(ctx)
	for {
		if err := c.sleep(ctx, c.Interval); err != nil {
			c.Logger.Info("scanner checker stopped", "err", err)
			return
		}
		c.RunOnce(ctx)
	}
}

// RunOnce evaluates all enabled rules once.
func (c *Checker) RunOnce(ctx context.Context) {
	if c.Scanner == nil {
		return
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	n, err := c.Scanner.RunOnce(ctx)
	if err != nil {
		c.Logger.Error("scanner run failed", "err", err)
		return
	}
	if n > 0 {
		c.Logger.Info("scanner matches saved", "count", n)
	}
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
