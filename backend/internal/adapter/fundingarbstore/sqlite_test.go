package fundingarbstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestWatchAndSignalLifecycle(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "fa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	w, err := s.CreateWatch(ctx, domain.FundingArbWatch{
		ID: "w1", ClientID: "c1", Symbol: "BTCUSDT", Notional: 10000, HoldHours: 24,
		MinProfit: 5, FeeBinancePct: 0.1, FeeBybitPct: 0.1,
		Status: domain.FundingArbWatchActive, Armed: true, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || w.Symbol != "BTCUSDT" || !w.Armed {
		t.Fatalf("%+v %v", w, err)
	}
	list, err := s.ListWatches(ctx, "c1")
	if err != nil || len(list) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	active, err := s.ListActiveWatches(ctx)
	if err != nil || len(active) != 1 {
		t.Fatalf("%+v %v", active, err)
	}
	n, err := s.CountWatches(ctx, "c1")
	if err != nil || n != 1 {
		t.Fatalf("%d %v", n, err)
	}
	if _, err := s.GetOpenSignal(ctx, "w1", "BTCUSDT"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("open %v", err)
	}
	sig, err := s.CreateSignal(ctx, domain.FundingArbSignal{
		ID: "s1", WatchID: "w1", ClientID: "c1", Symbol: "BTCUSDT",
		LongExchange: domain.ExchangeBinance, ShortExchange: domain.ExchangeBybit,
		NetAfterFees: 12, MinProfit: 5, Status: domain.FundingArbSignalOpen,
		OpenedAt: now, LastSeenAt: now,
	})
	if err != nil || sig.Status != domain.FundingArbSignalOpen {
		t.Fatalf("%+v %v", sig, err)
	}
	if err := s.TouchSignal(ctx, "s1", 13, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	open, err := s.GetOpenSignal(ctx, "w1", "BTCUSDT")
	if err != nil || open.NetAfterFees != 13 {
		t.Fatalf("%+v %v", open, err)
	}
	sig2, err := s.CreateSignal(ctx, domain.FundingArbSignal{
		ID: "s2", WatchID: "w1", ClientID: "c1", Symbol: "ETHUSDT",
		LongExchange: domain.ExchangeBybit, ShortExchange: domain.ExchangeBinance,
		NetAfterFees: 8, MinProfit: 5, Status: domain.FundingArbSignalOpen,
		OpenedAt: now, LastSeenAt: now,
	})
	if err != nil || sig2.Symbol != "ETHUSDT" {
		t.Fatalf("second open %+v %v", sig2, err)
	}
	opens, err := s.ListOpenSignals(ctx, "w1")
	if err != nil || len(opens) != 2 {
		t.Fatalf("two open coins %+v %v", opens, err)
	}
	if err := s.CloseSignal(ctx, "s1", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetOpenSignal(ctx, "w1", "BTCUSDT"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	if err := s.SetWatchArmed(ctx, "w1", false, now); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetWatch(ctx, "c1", "w1")
	if err != nil || got.Armed {
		t.Fatalf("%+v %v", got, err)
	}
	exp := now.Add(8 * time.Hour)
	got.Status = domain.FundingArbWatchPaused
	got.MinProfit = 9
	got.ExpiresAt = &exp
	got.UpdatedAt = now
	got, err = s.UpdateWatch(ctx, *got)
	if err != nil || got.Status != domain.FundingArbWatchPaused || got.MinProfit != 9 || got.ExpiresAt == nil {
		t.Fatalf("update %+v %v", got, err)
	}

	if err := s.PurgeClient(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	list, err = s.ListWatches(ctx, "c1")
	if err != nil || len(list) != 0 {
		t.Fatalf("purged %+v %v", list, err)
	}
}
