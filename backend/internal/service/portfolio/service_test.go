package portfolio

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakePx struct {
	prices map[string]string
}

func (f *fakePx) GetTicker24h(_ context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	p, ok := f.prices[exchange+"|"+symbol]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p}, nil
}

func newSvc(t *testing.T, px *fakePx) *Service {
	t.Helper()
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if px == nil {
		px = &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	}
	return New(st, px)
}

func TestPortfolio_CreateBuySellAndPnL(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()

	p, err := svc.Create(ctx, CreateInput{ClientID: "trader-1", StartingBalance: 10000})
	if err != nil || p.CashBalance != 10000 {
		t.Fatalf("%+v %v", p, err)
	}
	// Duplicate create fails
	if _, err := svc.Create(ctx, CreateInput{ClientID: "trader-1", StartingBalance: 1}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("dup: %v", err)
	}

	tr, view, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "trader-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tr.Side != domain.TradeSideBuy || math.Abs(tr.Notional-200) > 1e-9 {
		t.Fatalf("%+v", tr)
	}
	if math.Abs(view.CashBalance-9800) > 1e-6 || len(view.Positions) != 1 {
		t.Fatalf("%+v", view)
	}
	// Mark up for unrealized
	px.prices["binance|BTCUSDT"] = "110"
	view, err = svc.View(ctx, "trader-1")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.UnrealizedPnL-20) > 1e-6 {
		t.Fatalf("unreal=%v", view.UnrealizedPnL)
	}

	// Sell 1 at 110
	tr, view, err = svc.PlaceOrder(ctx, OrderInput{
		ClientID: "trader-1", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(tr.RealizedPnL-10) > 1e-6 {
		t.Fatalf("realized trade=%v", tr.RealizedPnL)
	}
	if math.Abs(view.RealizedPnLTotal-10) > 1e-6 {
		t.Fatalf("realized total=%v", view.RealizedPnLTotal)
	}
	if math.Abs(view.CashBalance-(9800+110)) > 1e-6 {
		t.Fatalf("cash=%v", view.CashBalance)
	}
	if len(view.Positions) != 1 || math.Abs(view.Positions[0].Quantity-1) > 1e-9 {
		t.Fatalf("pos=%+v", view.Positions)
	}

	list, total, err := svc.ListTrades(ctx, "trader-1", 10, 0)
	if err != nil || total != 2 || len(list) != 2 {
		t.Fatalf("trades total=%d list=%d err=%v", total, len(list), err)
	}
}

func TestPortfolio_InsufficientCashAndQty(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|ETHUSDT": "1000"}})
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "poor", StartingBalance: 100})
	_, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "poor", Symbol: "ETHUSDT", Side: "buy", Quantity: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "poor", Symbol: "ETHUSDT", Side: "sell", Quantity: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestPortfolio_PendingLimitBuyFillAndCancel(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "110"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, err := svc.Create(ctx, CreateInput{ClientID: "pend-1", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}

	// Limit buy at 100 — not triggered while last is 110
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "pend-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
	})
	if err != nil || o.Status != domain.PendingStatusOpen {
		t.Fatalf("%+v %v", o, err)
	}
	filled, ok, err := svc.TryFillPendingOrder(ctx, *o, 110)
	if err != nil || ok || filled != nil {
		t.Fatalf("should not fill: ok=%v filled=%v err=%v", ok, filled, err)
	}

	// Price drops to 99 → fill at last
	filled, ok, err = svc.TryFillPendingOrder(ctx, *o, 99)
	if err != nil || !ok || filled == nil || filled.Status != domain.PendingStatusFilled {
		t.Fatalf("fill: ok=%v filled=%+v err=%v", ok, filled, err)
	}
	view, err := svc.View(ctx, "pend-1")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-(10000-99)) > 1e-6 || len(view.Positions) != 1 {
		t.Fatalf("%+v", view)
	}

	// Idempotent: second fill attempt no-ops
	_, ok, err = svc.TryFillPendingOrder(ctx, *o, 90)
	if err != nil || ok {
		t.Fatalf("second fill: ok=%v err=%v", ok, err)
	}

	// Cancel path: place then cancel before fill
	o2, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "pend-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	canceled, err := svc.CancelPendingOrder(ctx, "pend-1", o2.ID)
	if err != nil || canceled.Status != domain.PendingStatusCanceled {
		t.Fatalf("%+v %v", canceled, err)
	}
	// Canceled must not execute later
	_, ok, err = svc.TryFillPendingOrder(ctx, *o2, 40)
	if err != nil || ok {
		t.Fatalf("canceled fill: ok=%v err=%v", ok, err)
	}
	open, err := svc.ListPendingOrders(ctx, "pend-1", "open", 10, 0)
	if err != nil || len(open) != 0 {
		t.Fatalf("open=%+v err=%v", open, err)
	}
}

func TestPortfolio_StopLossAndInsufficientOnFill(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|ETHUSDT": "2000"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "stop-1", StartingBalance: 5000})
	// Buy 1 ETH
	_, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "stop-1", Symbol: "ETHUSDT", Side: "buy", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	// Stop loss at 1800
	sl, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "stop-1", Symbol: "ETHUSDT", Type: "stop_loss", Quantity: 1, TriggerPrice: 1800,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Above stop — no fill
	_, ok, err := svc.TryFillPendingOrder(ctx, *sl, 1900)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	// Drop through stop
	filled, ok, err := svc.TryFillPendingOrder(ctx, *sl, 1700)
	if err != nil || !ok || filled.Status != domain.PendingStatusFilled {
		t.Fatalf("stop fill %+v ok=%v err=%v", filled, ok, err)
	}
	view, _ := svc.View(ctx, "stop-1")
	if len(view.Positions) != 0 {
		t.Fatalf("expected flat: %+v", view.Positions)
	}

	// Limit sell with no position → reject at trigger
	ls, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "stop-1", Symbol: "ETHUSDT", Type: "limit_sell", Quantity: 1, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, ok, err = svc.TryFillPendingOrder(ctx, *ls, 100)
	if err != nil || ok {
		t.Fatalf("should reject not fill: ok=%v err=%v", ok, err)
	}
	got, err := svc.ListPendingOrders(ctx, "stop-1", "rejected", 10, 0)
	if err != nil || len(got) != 1 || got[0].Status != domain.PendingStatusRejected {
		t.Fatalf("rejected=%+v err=%v", got, err)
	}
}

func TestPortfolio_OrderFillerRunOnce(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "95"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "fill-w", StartingBalance: 10000})
	_, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "fill-w", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	f := &OrderFiller{Portfolio: svc, Market: px, Interval: time.Hour}
	f.RunOnce(ctx)
	open, _ := svc.ListPendingOrders(ctx, "fill-w", "open", 10, 0)
	if len(open) != 0 {
		t.Fatalf("expected filled open=%+v", open)
	}
	view, _ := svc.View(ctx, "fill-w")
	if math.Abs(view.CashBalance-(10000-190)) > 1e-6 {
		t.Fatalf("cash=%v", view.CashBalance)
	}
}

func TestPortfolio_PendingPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pf.db")
	ctx := context.Background()
	st1, err := portfoliostore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc1 := New(st1, px)
	_, _ = svc1.Create(ctx, CreateInput{ClientID: "po-persist", StartingBalance: 1000})
	o, err := svc1.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "po-persist", Symbol: "BTCUSDT", Type: "limit_sell", Quantity: 1, TriggerPrice: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := o.ID
	_ = st1.Close()

	st2, err := portfoliostore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	svc2 := New(st2, px)
	open, err := svc2.ListPendingOrders(ctx, "po-persist", "open", 10, 0)
	if err != nil || len(open) != 1 || open[0].ID != id {
		t.Fatalf("%+v err=%v", open, err)
	}
}

func TestPortfolio_PersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pf.db")
	ctx := context.Background()
	st1, err := portfoliostore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "50"}}
	svc1 := New(st1, px)
	_, err = svc1.Create(ctx, CreateInput{ClientID: "persist", StartingBalance: 5000})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc1.PlaceOrder(ctx, OrderInput{ClientID: "persist", Symbol: "BTCUSDT", Side: "buy", Quantity: 10})
	if err != nil {
		t.Fatal(err)
	}
	_ = st1.Close()

	st2, err := portfoliostore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	svc2 := New(st2, px)
	view, err := svc2.View(ctx, "persist")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-4500) > 1e-6 || len(view.Positions) != 1 {
		t.Fatalf("%+v", view)
	}
	list, total, err := svc2.ListTrades(ctx, "persist", 10, 0)
	if err != nil || total != 1 || list[0].Side != domain.TradeSideBuy {
		t.Fatalf("%+v total=%d err=%v", list, total, err)
	}
}