package delistjob

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/deliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeSource struct {
	n    atomic.Int32
	fail bool
}

func (f *fakeSource) FetchSpotDelistSchedule(ctx context.Context) ([]domain.SpotDelistEntry, error) {
	f.n.Add(1)
	if f.fail {
		return nil, errors.New("upstream down")
	}
	return []domain.SpotDelistEntry{
		{Symbol: "ACXUSDT", DelistTime: time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)},
	}, nil
}

func TestRunnerRunOnStart(t *testing.T) {
	src := &fakeSource{}
	store := deliststore.NewMemory()
	r := &Runner{
		Source:     src,
		Store:      store,
		Interval:   time.Hour,
		RunOnStart: true,
		Logger:     slog.Default(),
		sleep: func(ctx context.Context, d time.Duration) error {
			return context.Canceled
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	// Start returns after first sleep cancel; runOnStart should have run.
	r.Start(ctx)
	cancel()
	if src.n.Load() < 1 {
		t.Fatal("expected startup fetch")
	}
	if _, ok := store.DelistTime(domain.ExchangeBinance, "ACXUSDT"); !ok {
		t.Fatal("store not updated")
	}
}

func TestRunnerFailedFetchKeepsStore(t *testing.T) {
	store := deliststore.NewMemory()
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "KEEPUSDT", DelistTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	})
	src := &fakeSource{fail: true}
	r := &Runner{
		Source:     src,
		Store:      store,
		Interval:   time.Hour,
		RunOnStart: true,
		Logger:     slog.Default(),
		sleep:      func(ctx context.Context, d time.Duration) error { return context.Canceled },
	}
	r.Start(context.Background())
	if _, ok := store.DelistTime(domain.ExchangeBinance, "KEEPUSDT"); !ok {
		t.Fatal("failed fetch should not clear store")
	}
}
