package pricealert

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/alertstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func newSvc(t *testing.T) (*Service, *alertstore.SQLite) {
	t.Helper()
	store, err := alertstore.Open(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return New(store), store
}

func TestService_CreateListGetDelete(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "user-1", Exchange: "binance", Symbol: "btcusdt",
		Condition: "above", TargetPrice: 90000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Symbol != "BTCUSDT" || a.Status != domain.AlertStatusActive {
		t.Fatalf("%+v", a)
	}
	list, err := svc.List(ctx, "user-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	got, err := svc.Get(ctx, "user-1", a.ID)
	if err != nil || got.ID != a.ID {
		t.Fatalf("%+v %v", got, err)
	}
	if err := svc.Delete(ctx, "user-1", a.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Get(ctx, "user-1", a.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("%v", err)
	}
}

func TestService_Validation(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	_, err := svc.Create(ctx, CreateInput{ClientID: "", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty client: %v", err)
	}
	_, err = svc.Create(ctx, CreateInput{ClientID: "c", Symbol: "BTCUSDT", Condition: "sideways", TargetPrice: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("bad cond: %v", err)
	}
	_, err = svc.Create(ctx, CreateInput{ClientID: "c", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 0})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("bad price: %v", err)
	}
}

func TestService_MaxAlerts(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	for i := 0; i < domain.MaxPriceAlertsPerClient; i++ {
		_, err := svc.Create(ctx, CreateInput{
			ClientID: "maxc", Symbol: "S" + strconv.Itoa(i) + "USDT",
			Condition: "above", TargetPrice: float64(i + 1),
		})
		if err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}
	}
	_, err := svc.Create(ctx, CreateInput{
		ClientID: "maxc", Symbol: "OVERFLOWUSDT", Condition: "above", TargetPrice: 1,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected max error: %v", err)
	}
}

func TestService_PersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alerts.db")
	ctx := context.Background()

	store1, err := alertstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	svc1 := New(store1)
	a, err := svc1.Create(ctx, CreateInput{
		ClientID: "restart-user", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 2500,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = store1.Close()

	store2, err := alertstore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()
	svc2 := New(store2)
	list, err := svc2.List(ctx, "restart-user")
	if err != nil || len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("after restart: %+v err=%v", list, err)
	}
}

type fakeTicker struct {
	mu     sync.Mutex
	prices map[string]string // exchange|symbol -> last
	calls  int
}

func (f *fakeTicker) GetTicker24h(_ context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	key := exchange + "|" + symbol
	p, ok := f.prices[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p}, nil
}

func TestChecker_TriggersOnce(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "chk", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Below target — should not fire
	mkt := &fakeTicker{prices: map[string]string{"binance|BTCUSDT": "99"}}
	c := &Checker{Alerts: svc, Market: mkt, Logger: nil}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "chk", a.ID)
	if got.Status != domain.AlertStatusActive {
		t.Fatalf("should still active: %+v", got)
	}

	// Meet target
	mkt.prices["binance|BTCUSDT"] = "100.5"
	c.RunOnce(ctx)
	got, err = svc.Get(ctx, "chk", a.ID)
	if err != nil || got.Status != domain.AlertStatusTriggered {
		t.Fatalf("%+v %v", got, err)
	}
	if got.TriggeredPrice != 100.5 {
		t.Fatalf("triggered price=%v", got.TriggeredPrice)
	}
	// Run again with higher price — must not change triggered price / re-fire
	mkt.prices["binance|BTCUSDT"] = "200"
	c.RunOnce(ctx)
	got2, _ := svc.Get(ctx, "chk", a.ID)
	if got2.TriggeredPrice != 100.5 || got2.Status != domain.AlertStatusTriggered {
		t.Fatalf("re-fire mutated alert: %+v", got2)
	}
}

func TestChecker_BelowAndDedupeFetch(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "a", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 2000})
	_, _ = svc.Create(ctx, CreateInput{ClientID: "b", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 2100})
	mkt := &fakeTicker{prices: map[string]string{"binance|ETHUSDT": "1999"}}
	c := &Checker{Alerts: svc, Market: mkt}
	c.RunOnce(ctx)
	if mkt.calls != 1 {
		t.Fatalf("want 1 ticker fetch for same pair, got %d", mkt.calls)
	}
	listA, _ := svc.List(ctx, "a")
	listB, _ := svc.List(ctx, "b")
	if listA[0].Status != domain.AlertStatusTriggered || listB[0].Status != domain.AlertStatusTriggered {
		t.Fatalf("a=%+v b=%+v", listA, listB)
	}
}

func TestChecker_DoesNotTriggerOnStaleWithoutRefresh(t *testing.T) {
	// Sanity: missing ticker does not mark triggered.
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, _ := svc.Create(ctx, CreateInput{ClientID: "x", Symbol: "SOLUSDT", Condition: "above", TargetPrice: 1})
	c := &Checker{Alerts: svc, Market: &fakeTicker{prices: map[string]string{}}}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "x", a.ID)
	if got.Status != domain.AlertStatusActive {
		t.Fatalf("%+v", got)
	}
}