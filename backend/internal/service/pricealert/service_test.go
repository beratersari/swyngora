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
	// Tests use httptest (loopback) and offline hostnames; production leaves this false.
	svc := New(store)
	svc.AllowPrivateWebhooks = true
	return svc, store
}

func TestService_SetWebhook_SSRFBlockedByDefault(t *testing.T) {
	store, err := alertstore.Open(filepath.Join(t.TempDir(), "ssrf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := New(store) // AllowPrivateWebhooks == false
	_, err = svc.SetWebhook(context.Background(), "ssrf-user", domain.WebhookSettings{
		URL: "http://127.0.0.1/hook", DeliveryMode: "immediate",
	})
	if err == nil {
		t.Fatal("expected SSRF rejection for loopback webhook")
	}
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
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

func TestChecker_RepeatingCrossAndRearm(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "rep", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 100, Mode: "repeating",
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Mode != domain.AlertModeRepeating || a.Armed {
		t.Fatalf("defaults: %+v", a)
	}
	mkt := &fakeTicker{prices: map[string]string{"binance|BTCUSDT": "105"}}
	c := &Checker{Alerts: svc, Market: mkt}
	// Start already above, disarmed — must not fire
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "rep", a.ID)
	if got.Status != domain.AlertStatusActive || got.TriggeredPrice != 0 {
		t.Fatalf("no fire while already hot: %+v", got)
	}
	// Safe side re-arms
	mkt.prices["binance|BTCUSDT"] = "90"
	c.RunOnce(ctx)
	got, _ = svc.Get(ctx, "rep", a.ID)
	if !got.Armed {
		t.Fatalf("expected armed after safe side: %+v", got)
	}
	// Cross up — fire, stay active, disarmed
	mkt.prices["binance|BTCUSDT"] = "101"
	c.RunOnce(ctx)
	got, _ = svc.Get(ctx, "rep", a.ID)
	if got.Status != domain.AlertStatusActive || got.TriggeredPrice != 101 || got.Armed {
		t.Fatalf("after first cross: %+v", got)
	}
	// Stay above — no re-fire (triggered price unchanged)
	mkt.prices["binance|BTCUSDT"] = "120"
	c.RunOnce(ctx)
	got2, _ := svc.Get(ctx, "rep", a.ID)
	if got2.TriggeredPrice != 101 {
		t.Fatalf("re-fired while staying hot: %+v", got2)
	}
	// Back safe then cross again
	mkt.prices["binance|BTCUSDT"] = "95"
	c.RunOnce(ctx)
	mkt.prices["binance|BTCUSDT"] = "102"
	c.RunOnce(ctx)
	got3, _ := svc.Get(ctx, "rep", a.ID)
	if got3.TriggeredPrice != 102 || got3.Status != domain.AlertStatusActive {
		t.Fatalf("second cross: %+v", got3)
	}
}

func TestCreate_DefaultModeOneTime(t *testing.T) {
	svc, _ := newSvc(t)
	a, err := svc.Create(context.Background(), CreateInput{
		ClientID: "m", Symbol: "ETHUSDT", Condition: "below", TargetPrice: 1,
	})
	if err != nil || a.Mode != domain.AlertModeOneTime {
		t.Fatalf("%+v %v", a, err)
	}
}

func TestCreate_InvalidMode(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.Create(context.Background(), CreateInput{
		ClientID: "m", Symbol: "ETHUSDT", Condition: "above", TargetPrice: 1, Mode: "daily",
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}