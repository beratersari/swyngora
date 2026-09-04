package radar

import (
	"context"
	"log/slog"
	"time"
)

// Worker periodically scans for new Setup Radar hits and fills after-windows.
type Worker struct {
	Radar     *Service
	ScanEvery time.Duration
	FillEvery time.Duration
	Logger    *slog.Logger
	now       func() time.Time
	sleep     func(context.Context, time.Duration) error
}

// Start runs fill+scan until ctx is cancelled. Blocking.
func (w *Worker) Start(ctx context.Context) {
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.now == nil {
		w.now = time.Now
	}
	if w.sleep == nil {
		w.sleep = sleepCtx
	}
	if w.ScanEvery <= 0 {
		w.ScanEvery = 15 * time.Minute
	}
	if w.FillEvery <= 0 {
		w.FillEvery = time.Minute
	}
	nextScan := w.now().UTC()
	w.fill(ctx)
	w.scan(ctx)
	nextScan = w.now().UTC().Add(w.ScanEvery)
	for {
		if err := w.sleep(ctx, w.FillEvery); err != nil {
			w.Logger.Info("setup radar worker stopped", "err", err)
			return
		}
		w.fill(ctx)
		if !w.now().UTC().Before(nextScan) {
			w.scan(ctx)
			nextScan = w.now().UTC().Add(w.ScanEvery)
		}
	}
}

func (w *Worker) fill(ctx context.Context) {
	if w.Radar == nil {
		return
	}
	n, err := w.Radar.FillDue(ctx, w.now().UTC(), 40)
	if err != nil {
		w.Logger.Debug("radar fill due failed", "err", err)
		return
	}
	if n > 0 {
		w.Logger.Info("radar after-windows filled", "count", n)
	}
}

func (w *Worker) scan(ctx context.Context) {
	if w.Radar == nil {
		return
	}
	n, err := w.Radar.ScanAndRecord(ctx)
	if err != nil {
		w.Logger.Debug("radar scan-and-record failed", "err", err)
		return
	}
	if n > 0 {
		w.Logger.Info("radar hits saved", "count", n)
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
