package supplyjob

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeSupply struct {
	refreshes atomic.Int32
}

func (f *fakeSupply) GetSupply(context.Context, string) (*domain.AssetSupply, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeSupply) Refresh(context.Context) (int, error) {
	f.refreshes.Add(1)
	return 3, nil
}

func TestRunner_StartupAndDaily(t *testing.T) {
	fs := &fakeSupply{}
	ticks := 0
	r := &Runner{
		Supply:     fs,
		Hour:       3,
		Minute:     0,
		Loc:        time.UTC,
		RunOnStart: true,
		Logger:     slog.Default(),
		now: func() time.Time {
			// Fixed "now" so daily wait is deterministic; we cancel quickly.
			return time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC)
		},
		sleep: func(ctx context.Context, d time.Duration) error {
			ticks++
			if ticks >= 1 {
				return context.Canceled
			}
			return nil
		},
	}
	r.Start(context.Background())
	// startup refresh + one daily attempt before sleep cancels? 
	// Flow: RunOnStart runOnce, then loop: sleep returns cancel on first wait → exit
	// So only startup refresh.
	if fs.refreshes.Load() < 1 {
		t.Fatalf("expected at least startup refresh, got %d", fs.refreshes.Load())
	}
}

func TestRunner_DailyFires(t *testing.T) {
	fs := &fakeSupply{}
	sleeps := 0
	r := &Runner{
		Supply:     fs,
		Hour:       3,
		Minute:     0,
		Loc:        time.UTC,
		RunOnStart: false,
		Logger:     slog.Default(),
		now: func() time.Time {
			return time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC)
		},
		sleep: func(ctx context.Context, d time.Duration) error {
			sleeps++
			if sleeps == 1 {
				return nil // wake for daily
			}
			return context.Canceled
		},
	}
	r.Start(context.Background())
	if fs.refreshes.Load() != 1 {
		t.Fatalf("refreshes=%d want 1 daily", fs.refreshes.Load())
	}
}

type flakySupply struct {
	failsLeft atomic.Int32
	ok        atomic.Int32
}

func (f *flakySupply) GetSupply(context.Context, string) (*domain.AssetSupply, error) {
	return nil, domain.ErrNotFound
}

func (f *flakySupply) Refresh(context.Context) (int, error) {
	if f.failsLeft.Add(-1) >= 0 {
		return 0, domain.ErrUpstream
	}
	f.ok.Add(1)
	return 2, nil
}

func TestRunner_RetriesOnFailure(t *testing.T) {
	fs := &flakySupply{}
	fs.failsLeft.Store(2) // first two Refresh calls fail (Add(-1) → 1 then 0, both >=0)
	// Wait: failsLeft starts at 2
	// call1: Add(-1) → 1 >= 0 → fail
	// call2: Add(-1) → 0 >= 0 → fail  
	// call3: Add(-1) → -1 >= 0? false → ok
	// Actually with Store(2): first failsLeft.Add(-1) returns 1, second returns 0, third returns -1.
	// Condition is >= 0 so first two fail, third succeeds. Good with Store(2).

	sleeps := 0
	r := &Runner{
		Supply:        fs,
		Hour:          3,
		Minute:        0,
		Loc:           time.UTC,
		RunOnStart:    true,
		Logger:        slog.Default(),
		RetryBackoffs: []time.Duration{0, time.Millisecond, time.Millisecond},
		now: func() time.Time {
			return time.Date(2026, 7, 25, 1, 0, 0, 0, time.UTC)
		},
		sleep: func(ctx context.Context, d time.Duration) error {
			sleeps++
			// After successful startup retries, cancel daily wait.
			if fs.ok.Load() >= 1 && d > time.Second {
				return context.Canceled
			}
			return nil
		},
	}
	r.Start(context.Background())
	if fs.ok.Load() != 1 {
		t.Fatalf("want 1 successful refresh after retries, ok=%d", fs.ok.Load())
	}
}
