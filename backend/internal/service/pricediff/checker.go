package pricediff

import (
	"context"
	"log/slog"
	"time"
)

// Checker evaluates active price-diff watches on an interval.
type Checker struct {
	Service  *Service
	Interval time.Duration
	Logger   *slog.Logger
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
}

// Start runs until ctx is canceled.
func (c *Checker) Start(ctx context.Context) {
	if c.Service == nil {
		return
	}
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
		c.Interval = 30 * time.Second
	}
	c.RunOnce(ctx)
	for {
		if err := c.sleep(ctx, c.Interval); err != nil {
			c.Logger.Info("price-diff checker stopped", "err", err)
			return
		}
		c.RunOnce(ctx)
	}
}

// RunOnce evaluates all active watches.
func (c *Checker) RunOnce(ctx context.Context) {
	if c.Service == nil {
		return
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	if c.now == nil {
		c.now = time.Now
	}
	created, closed, touched, err := c.Service.ProcessActiveWatches(ctx, c.now().UTC())
	if err != nil {
		c.Logger.Error("price-diff process", "err", err)
		return
	}
	if created > 0 || closed > 0 {
		c.Logger.Info("price-diff tick", "created", created, "closed", closed, "touched", touched)
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
