package pricealert

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

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

type closedAccounts map[string]bool

func (m closedAccounts) IsClosed(_ context.Context, clientID string) (bool, *domain.Account, error) {
	return m[clientID], nil, nil
}

func TestChecker_SkipsClosedAccount(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "closed-user", Symbol: "BTCUSDT", Condition: "above", TargetPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	mkt := &fakeTicker{prices: map[string]string{"binance|BTCUSDT": "200"}}
	c := &Checker{Alerts: svc, Market: mkt, Accounts: closedAccounts{"closed-user": true}}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "closed-user", a.ID)
	if got.Status != domain.AlertStatusActive {
		t.Fatalf("closed tenant must not fire: %+v", got)
	}
	c.Accounts = closedAccounts{"closed-user": false}
	c.RunOnce(ctx)
	got, _ = svc.Get(ctx, "closed-user", a.ID)
	if got.Status != domain.AlertStatusTriggered {
		t.Fatalf("reopened tenant should fire: %+v", got)
	}
}

func TestChecker_SkipsHaltedLastPrint(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "halt-alert", Symbol: "GONEUSDT", Condition: "above", TargetPrice: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	mkt := haltedTicker{price: "200"}
	c := &Checker{Alerts: svc, Market: mkt}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "halt-alert", a.ID)
	if got.Status != domain.AlertStatusActive {
		t.Fatalf("halted last print must not fire: %+v", got)
	}
}

type haltedTicker struct {
	price string
}

func (h haltedTicker) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	p := h.price
	if p == "" {
		p = "100"
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p, Halted: true}, nil
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

func TestCreate_OrderBookImbalanceAndWall(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	imb, err := svc.Create(ctx, CreateInput{
		ClientID: "ob", Symbol: "BTCUSDT", Kind: "imbalance", Condition: "above", TargetPrice: 0.2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if imb.Kind != domain.AlertKindImbalance || imb.Mode != domain.AlertModeRepeating || !imb.Armed || imb.RangePct != 2 {
		t.Fatalf("%+v", imb)
	}
	wall, err := svc.Create(ctx, CreateInput{
		ClientID: "ob", Symbol: "ETHUSDT", Kind: "wall", Condition: "bid", TargetPrice: 0.15, RangePct: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if wall.Kind != domain.AlertKindWall || wall.RangePct != 5 || !wall.Armed {
		t.Fatalf("%+v", wall)
	}
	if _, err := svc.Create(ctx, CreateInput{
		ClientID: "ob", Symbol: "BTCUSDT", Kind: "imbalance", Condition: "above", TargetPrice: 0.01,
	}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("tiny threshold: %v", err)
	}
}

type prepTape struct {
	calls []struct {
		ex, sym string
		look    time.Duration
	}
}

func (p *prepTape) Prepare(_ context.Context, exchange, symbol string, lookback time.Duration) {
	p.calls = append(p.calls, struct {
		ex, sym string
		look    time.Duration
	}{exchange, symbol, lookback})
}

func TestCreate_LiquidationFeedAndCascade(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	feed, err := svc.Create(ctx, CreateInput{
		ClientID: "lq", Kind: "liquidation_feed", Exchange: "bybit", TargetPrice: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if feed.Kind != domain.AlertKindLiqFeed || feed.Symbol != domain.LiqAlertSymbolAll || !feed.Armed || feed.Mode != domain.AlertModeRepeating {
		t.Fatalf("%+v", feed)
	}
	cas, err := svc.Create(ctx, CreateInput{
		ClientID: "lq", Kind: "liquidation_cascade", Exchange: "all", Symbol: "BTCUSDT", Condition: "cascade",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cas.Kind != domain.AlertKindLiqCascade || cas.Symbol != "BTCUSDT" || string(cas.Exchange) != "all" || !cas.Armed {
		t.Fatalf("%+v", cas)
	}
	mkt, err := svc.Create(ctx, CreateInput{
		ClientID: "lq", Kind: "liquidation_cascade", Symbol: "all",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mkt.Symbol != domain.LiqAlertSymbolAll {
		t.Fatalf("%+v", mkt)
	}
	not, err := svc.Create(ctx, CreateInput{
		ClientID: "lq", Kind: "liquidation_notional", Exchange: "bybit", Symbol: "BTCUSDT",
		Condition: "long", TargetPrice: 20e6, Window: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if not.Kind != domain.AlertKindLiqNotional || not.RangePct != 5 || not.Condition != "long" || !not.Armed {
		t.Fatalf("%+v", not)
	}
}

func TestCreate_LiquidationNotionalPreparesTape(t *testing.T) {
	svc, _ := newSvc(t)
	tape := &prepTape{}
	svc.Tape = tape
	_, err := svc.Create(context.Background(), CreateInput{
		ClientID: "prep", Kind: "liquidation_notional", Exchange: "bybit", Symbol: "PEPEUSDT",
		Condition: "both", TargetPrice: 1e6, Window: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tape.calls) != 1 || tape.calls[0].sym != "PEPEUSDT" || tape.calls[0].ex != "bybit" || tape.calls[0].look != 5*time.Minute {
		t.Fatalf("prepare %+v", tape.calls)
	}
	_, err = svc.Create(context.Background(), CreateInput{
		ClientID: "prep", Kind: "liquidation_feed", Exchange: "bybit", TargetPrice: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tape.calls) != 1 {
		t.Fatalf("feed should not prepare a coin %+v", tape.calls)
	}
}

type fakeBook struct {
	mu    sync.Mutex
	books map[string]*domain.OrderBook
	calls int
}

func (f *fakeBook) GetSpotOrderBook(_ context.Context, exchange, symbol, _ string, _ int, rangePct float64) (*domain.OrderBook, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	key := exchange + "|" + symbol
	b, ok := f.books[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *b
	cp.Analysis.RangePct = rangePct
	return &cp, nil
}

func TestChecker_OrderBookImbalanceNoRetrigger(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "bk", Symbol: "BTCUSDT", Kind: "imbalance", Condition: "above", TargetPrice: 0.2,
	})
	if err != nil {
		t.Fatal(err)
	}
	books := &fakeBook{books: map[string]*domain.OrderBook{
		"binance|BTCUSDT": {Analysis: domain.OrderBookAnalysis{Imbalance: 0.3, Pressure: "buy"}},
	}}
	c := &Checker{Alerts: svc, Books: books}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "bk", a.ID)
	if got.TriggeredPrice != 0.3 || got.Armed || got.Status != domain.AlertStatusActive {
		t.Fatalf("first fire %+v", got)
	}
	c.RunOnce(ctx)
	got2, _ := svc.Get(ctx, "bk", a.ID)
	if got2.TriggeredPrice != 0.3 {
		t.Fatalf("re-fired while still imbalanced: %+v", got2)
	}
	books.mu.Lock()
	books.books["binance|BTCUSDT"] = &domain.OrderBook{Analysis: domain.OrderBookAnalysis{Imbalance: 0.01}}
	books.mu.Unlock()
	c.RunOnce(ctx)
	got3, _ := svc.Get(ctx, "bk", a.ID)
	if !got3.Armed {
		t.Fatalf("re-arm after clear: %+v", got3)
	}
	books.mu.Lock()
	books.books["binance|BTCUSDT"] = &domain.OrderBook{Analysis: domain.OrderBookAnalysis{Imbalance: 0.25}}
	books.mu.Unlock()
	c.RunOnce(ctx)
	got4, _ := svc.Get(ctx, "bk", a.ID)
	if got4.TriggeredPrice != 0.25 {
		t.Fatalf("second appearance: %+v", got4)
	}
}

func TestChecker_OrderBookWallAppearAndClear(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "w", Symbol: "ETHUSDT", Kind: "wall", Condition: "ask", TargetPrice: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	books := &fakeBook{books: map[string]*domain.OrderBook{
		"binance|ETHUSDT": {Analysis: domain.OrderBookAnalysis{Walls: []domain.OrderBookWall{{Side: "ask", Share: 0.4}}}},
	}}
	c := &Checker{Alerts: svc, Books: books}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "w", a.ID)
	if got.TriggeredPrice != 0.4 || got.Armed {
		t.Fatalf("wall fire %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "w", a.ID)
	if still.TriggeredPrice != 0.4 {
		t.Fatalf("still present re-fire %+v", still)
	}
	books.mu.Lock()
	books.books["binance|ETHUSDT"] = &domain.OrderBook{Analysis: domain.OrderBookAnalysis{}}
	books.mu.Unlock()
	c.RunOnce(ctx)
	cleared, _ := svc.Get(ctx, "w", a.ID)
	if !cleared.Armed {
		t.Fatalf("re-arm after wall gone %+v", cleared)
	}
}

type fakeLiq struct {
	feed   domain.LiquidationFeed
	rep    *domain.CascadeReport
	scan   *domain.CascadeScan
	events []domain.LiquidationEvent
}

func (f *fakeLiq) GetLiquidationFeed(string) domain.LiquidationFeed { return f.feed }
func (f *fakeLiq) GetLiquidationCascade(context.Context, string, string) (*domain.CascadeReport, error) {
	return f.rep, nil
}
func (f *fakeLiq) ScanLiquidationCascades(context.Context, string) (*domain.CascadeScan, error) {
	return f.scan, nil
}
func (f *fakeLiq) ListLiquidationEvents(string, string, time.Time) []domain.LiquidationEvent {
	return f.events
}

func TestChecker_LiquidationFeedNoRetrigger(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "ff", Kind: "liquidation_feed", Exchange: "bybit", TargetPrice: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	src := &fakeLiq{feed: domain.LiquidationFeed{Venues: []domain.LiquidationVenueHealth{{
		Exchange: "bybit", Live: false, LastSeenAt: now.Add(-3 * time.Minute),
	}}}}
	c := &Checker{Alerts: svc, Liquidations: src, now: func() time.Time { return now }}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "ff", a.ID)
	if got.Armed || got.TriggeredPrice < 170 {
		t.Fatalf("first fire %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "ff", a.ID)
	if still.TriggeredAt == nil || got.TriggeredAt == nil || !still.TriggeredAt.Equal(*got.TriggeredAt) {
		t.Fatalf("re-fired while down %+v vs %+v", still, got)
	}
	src.feed.Venues[0].Live = true
	src.feed.Venues[0].LastSeenAt = now
	c.RunOnce(ctx)
	ok, _ := svc.Get(ctx, "ff", a.ID)
	if !ok.Armed {
		t.Fatalf("re-arm when live %+v", ok)
	}
}

func TestChecker_LiquidationCascadeNamesCoin(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "cc", Kind: "liquidation_cascade", Exchange: "all", Symbol: "BTCUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := &fakeLiq{rep: &domain.CascadeReport{
		Symbol: "BTCUSDT",
		Venues: []domain.CascadeVenue{{
			Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT",
			Grade: domain.CascadeGradeCascade, Score: 5, Side: "long", Summary: "bybit long cascade",
		}},
	}}
	c := &Checker{Alerts: svc, Liquidations: src, now: time.Now}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "cc", a.ID)
	if got.Armed || got.TriggeredPrice != 5 {
		t.Fatalf("cascade fire %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "cc", a.ID)
	if still.TriggeredPrice != 5 {
		t.Fatalf("re-fired %+v", still)
	}
}

func TestChecker_LiquidationNotionalNoRetriggerUntilNewWave(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, CreateInput{
		ClientID: "nw", Kind: "liquidation_notional", Exchange: "all", Symbol: "BTCUSDT",
		Condition: "both", TargetPrice: 20e6, Window: "5m",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	src := &fakeLiq{events: []domain.LiquidationEvent{
		{Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Notional: 12e6, Time: now.Add(-2 * time.Minute)},
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: domain.LiquidationSideShort, Notional: 10e6, Time: now.Add(-1 * time.Minute)},
	}}
	c := &Checker{Alerts: svc, Liquidations: src, now: func() time.Time { return now }}
	c.RunOnce(ctx)
	got, _ := svc.Get(ctx, "nw", a.ID)
	if got.Armed || got.TriggeredPrice < 20e6 {
		t.Fatalf("first wave %+v", got)
	}
	c.RunOnce(ctx)
	still, _ := svc.Get(ctx, "nw", a.ID)
	if still.TriggeredAt == nil || got.TriggeredAt == nil || !still.TriggeredAt.Equal(*got.TriggeredAt) {
		t.Fatalf("same wave re-fired %+v vs %+v", still, got)
	}
	src.events = nil
	c.RunOnce(ctx)
	armed, _ := svc.Get(ctx, "nw", a.ID)
	if !armed.Armed {
		t.Fatalf("re-arm after wave ends %+v", armed)
	}
	src.events = []domain.LiquidationEvent{
		{Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong, Notional: 25e6, Time: now.Add(-30 * time.Second)},
	}
	c.RunOnce(ctx)
	again, _ := svc.Get(ctx, "nw", a.ID)
	if again.Armed || again.TriggeredPrice != 25e6 {
		t.Fatalf("new wave %+v", again)
	}
}
