package portfolio

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"sync"
	"sync/atomic"
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

	// Limit buy at 100 — reserves cash, not triggered while last is 110
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "pend-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
	})
	if err != nil || o.Status != domain.PendingStatusOpen || math.Abs(o.ReservedCash-100) > 1e-9 {
		t.Fatalf("%+v %v", o, err)
	}
	view, err := svc.View(ctx, "pend-1")
	if err != nil || math.Abs(view.ReservedCash-100) > 1e-9 || math.Abs(view.AvailableCash-9900) > 1e-9 {
		t.Fatalf("view after reserve %+v err=%v", view, err)
	}
	// Reserved cash cannot fund a market buy that needs all cash
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "pend-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 99.1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("should block market buy with reserved cash: %v", err)
	}

	filled, ok, err := svc.TryFillPendingOrder(ctx, *o, 110, 0)
	if err != nil || ok || filled != nil {
		t.Fatalf("should not fill: ok=%v filled=%v err=%v", ok, filled, err)
	}

	// Price drops to 99 → fill at last
	filled, ok, err = svc.TryFillPendingOrder(ctx, *o, 99, 0)
	if err != nil || !ok || filled == nil || filled.Status != domain.PendingStatusFilled {
		t.Fatalf("fill: ok=%v filled=%+v err=%v", ok, filled, err)
	}
	view, err = svc.View(ctx, "pend-1")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-(10000-99)) > 1e-6 || len(view.Positions) != 1 {
		t.Fatalf("%+v", view)
	}
	if view.ReservedCash != 0 {
		t.Fatalf("reserved after fill=%v", view.ReservedCash)
	}

	// Idempotent: second fill attempt no-ops
	_, ok, err = svc.TryFillPendingOrder(ctx, *o, 90, 0)
	if err != nil || ok {
		t.Fatalf("second fill: ok=%v err=%v", ok, err)
	}

	// Cancel path: place then cancel before fill releases reservation
	o2, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "pend-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	view, _ = svc.View(ctx, "pend-1")
	if view.ReservedCash < 49 {
		t.Fatalf("expected reserve after place: %+v", view)
	}
	canceled, err := svc.CancelPendingOrder(ctx, "pend-1", o2.ID)
	if err != nil || canceled.Status != domain.PendingStatusCanceled || canceled.ReservedCash != 0 {
		t.Fatalf("%+v %v", canceled, err)
	}
	view, _ = svc.View(ctx, "pend-1")
	if view.ReservedCash != 0 {
		t.Fatalf("reserved after cancel=%v", view.ReservedCash)
	}
	// Canceled must not execute later
	_, ok, err = svc.TryFillPendingOrder(ctx, *o2, 40, 0)
	if err != nil || ok {
		t.Fatalf("canceled fill: ok=%v err=%v", ok, err)
	}
	open, err := svc.ListPendingOrders(ctx, "pend-1", "open", 10, 0)
	if err != nil || len(open) != 0 {
		t.Fatalf("open=%+v err=%v", open, err)
	}
}

func TestPortfolio_PartialFillsAndSellReservation(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "part-1", StartingBalance: 10000})
	// Buy 5 for inventory
	_, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "part-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 5})
	if err != nil {
		t.Fatal(err)
	}
	// Limit sell 4 — reserves position
	ls, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "part-1", Symbol: "BTCUSDT", Type: "limit_sell", Quantity: 4, TriggerPrice: 110,
	})
	if err != nil || math.Abs(ls.ReservedQuantity-4) > 1e-9 {
		t.Fatalf("%+v %v", ls, err)
	}
	// Only 1 available for market sell
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "part-1", Symbol: "BTCUSDT", Side: "sell", Quantity: 2})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("should not sell reserved qty: %v", err)
	}
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "part-1", Symbol: "BTCUSDT", Side: "sell", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}

	// Partial fill of limit sell: 1.5 then rest
	px.prices["binance|BTCUSDT"] = "120"
	got, ok, err := svc.TryFillPendingOrder(ctx, *ls, 120, 1.5)
	if err != nil || !ok || got.Status != domain.PendingStatusOpen {
		t.Fatalf("partial1 %+v ok=%v err=%v", got, ok, err)
	}
	if math.Abs(got.FilledQuantity-1.5) > 1e-9 || math.Abs(got.RemainingQuantity-2.5) > 1e-9 {
		t.Fatalf("partial sizes %+v", got)
	}
	if math.Abs(got.ReservedQuantity-2.5) > 1e-9 {
		t.Fatalf("reserved after partial=%v", got.ReservedQuantity)
	}
	// Second partial completes
	got, ok, err = svc.TryFillPendingOrder(ctx, *got, 120, 0)
	if err != nil || !ok || got.Status != domain.PendingStatusFilled {
		t.Fatalf("complete %+v ok=%v err=%v", got, ok, err)
	}
	list, total, err := svc.ListTrades(ctx, "part-1", 20, 0)
	if err != nil || total < 4 {
		t.Fatalf("trades total=%d err=%v", total, err)
	}
	// Two partial sells should be in history with pending order id
	var sellFills int
	for _, tr := range list {
		if tr.PendingOrderID == ls.ID {
			sellFills++
		}
	}
	if sellFills != 2 {
		t.Fatalf("want 2 pending fills, got %d list=%+v", sellFills, list)
	}

	// Buy reserve: cannot over-reserve cash
	_, err = svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "part-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 100000, TriggerPrice: 100,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("over-reserve buy: %v", err)
	}
}

func TestPortfolio_StopLossAndSellReserve(t *testing.T) {
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
	if err != nil || math.Abs(sl.ReservedQuantity-1) > 1e-9 {
		t.Fatalf("%+v %v", sl, err)
	}
	// Cannot sell the reserved unit
	_, _, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "stop-1", Symbol: "ETHUSDT", Side: "sell", Quantity: 1})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("reserved sell blocked: %v", err)
	}
	// Above stop — no fill
	_, ok, err := svc.TryFillPendingOrder(ctx, *sl, 1900, 0)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	// Drop through stop
	filled, ok, err := svc.TryFillPendingOrder(ctx, *sl, 1700, 0)
	if err != nil || !ok || filled.Status != domain.PendingStatusFilled {
		t.Fatalf("stop fill %+v ok=%v err=%v", filled, ok, err)
	}
	view, _ := svc.View(ctx, "stop-1")
	if len(view.Positions) != 0 {
		t.Fatalf("expected flat: %+v", view.Positions)
	}

	// Limit sell with no position → fail at place (reservation)
	_, err = svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "stop-1", Symbol: "ETHUSDT", Type: "limit_sell", Quantity: 1, TriggerPrice: 100,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want insufficient position at place: %v", err)
	}
}

func TestPortfolio_GTCExpireIOCAndFOK(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	_, _ = svc.Create(ctx, CreateInput{ClientID: "tif-1", StartingBalance: 10000})

	// GTC with past expiry is rejected at place
	past := time.Now().UTC().Add(-time.Minute)
	_, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
		TimeInForce: "gtc", ExpiresAt: &past,
	})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("past expire: %v", err)
	}

	// GTC expires via ProcessOpenOrder
	exp := time.Now().UTC().Add(time.Hour)
	gtc, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 50,
		TimeInForce: "gtc", ExpiresAt: &exp,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Force expiry by processing with now past expiresAt
	pastNow := exp.Add(time.Second)
	got, ok, err := svc.ProcessOpenOrder(ctx, *gtc, 40, pastNow, 0)
	if err != nil || !ok || got.Status != domain.PendingStatusCanceled || got.CancelReason != domain.CancelReasonExpired {
		t.Fatalf("expired %+v ok=%v err=%v", got, ok, err)
	}
	if got.ReservedCash != 0 {
		t.Fatalf("reserve after expire=%v", got.ReservedCash)
	}

	// IOC: not marketable → cancel no fill
	ioc, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 50,
		TimeInForce: "ioc",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = svc.ProcessOpenOrder(ctx, *ioc, 100, time.Now().UTC(), 0)
	if err != nil || !ok || got.Status != domain.PendingStatusCanceled || got.CancelReason != domain.CancelReasonIOCNoFill {
		t.Fatalf("ioc no fill %+v ok=%v err=%v", got, ok, err)
	}

	// IOC: partial fill then cancel remainder
	ioc2, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 100,
		TimeInForce: "ioc",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = svc.ProcessOpenOrder(ctx, *ioc2, 90, time.Now().UTC(), 1) // fill only 1 of 2
	if err != nil || !ok {
		t.Fatalf("ioc partial err=%v ok=%v", err, ok)
	}
	if got.Status != domain.PendingStatusCanceled || got.CancelReason != domain.CancelReasonIOCRemainder {
		t.Fatalf("ioc remainder status %+v", got)
	}
	if math.Abs(got.FilledQuantity-1) > 1e-9 || got.ReservedCash != 0 {
		t.Fatalf("ioc filled/reserve %+v", got)
	}

	// FOK: cannot full fill → cancel with no fill
	// Reserve cash for 2, but maxFill path: use Process with maxFill that would partial - FOK checks full remaining
	// Place FOK buy 3 at 100 needing 300 cash - we have plenty. Trigger at 90 so marketable.
	// Force incomplete by maxFillQty simulation inside Process: FOK ignores maxFill and requires full can from reservation.
	// To force FOK fail: use sell FOK without enough reserved - actually reservation ensures full size.
	// FOK fails when not triggered, or when maxFillable < remaining (e.g. reserved cash insufficient for full at fill price - impossible for limit buy if reserved correctly).
	// Use maxFillable with sell: buy 2, FOK sell 2, marketable - full fill.
	// For FOK unfilled: not marketable.
	fok, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 10,
		TimeInForce: "fok",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = svc.ProcessOpenOrder(ctx, *fok, 100, time.Now().UTC(), 0)
	if err != nil || !ok || got.Status != domain.PendingStatusCanceled || got.CancelReason != domain.CancelReasonFOKUnfilled {
		t.Fatalf("fok unfilled %+v ok=%v err=%v", got, ok, err)
	}
	if got.FilledQuantity != 0 {
		t.Fatalf("fok must not fill: %+v", got)
	}

	// FOK full fill when marketable
	fok2, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "tif-1", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
		TimeInForce: "fok",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = svc.ProcessOpenOrder(ctx, *fok2, 95, time.Now().UTC(), 0)
	if err != nil || !ok || got.Status != domain.PendingStatusFilled {
		t.Fatalf("fok fill %+v ok=%v err=%v", got, ok, err)
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
		ClientID: "po-persist", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := o.ID
	if math.Abs(o.ReservedCash-200) > 1e-9 {
		t.Fatalf("reserve %+v", o)
	}
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
	if math.Abs(open[0].ReservedCash-200) > 1e-9 || math.Abs(open[0].RemainingQuantity-2) > 1e-9 {
		t.Fatalf("persisted reservation %+v", open[0])
	}
	view, err := svc2.View(ctx, "po-persist")
	if err != nil || math.Abs(view.ReservedCash-200) > 1e-9 {
		t.Fatalf("view %+v err=%v", view, err)
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

// Concurrent market buys of the full cash budget must not both succeed (lost-update race).
func TestPortfolio_ConcurrentBuysDoNotOverspend(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	const start = 1000.0
	if _, err := svc.Create(ctx, CreateInput{ClientID: "race-1", StartingBalance: start}); err != nil {
		t.Fatal(err)
	}
	// Each buy costs 1000 (10 * 100). Only one can succeed with serialized mutations.
	var okCount atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.PlaceOrder(ctx, OrderInput{
				ClientID: "race-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 10,
			})
			if err == nil {
				okCount.Add(1)
			}
		}()
	}
	wg.Wait()
	if okCount.Load() != 1 {
		t.Fatalf("expected exactly 1 successful full-cash buy, got %d", okCount.Load())
	}
	view, err := svc.View(ctx, "race-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.CashBalance < -1e-6 {
		t.Fatalf("negative cash: %+v", view)
	}
	// Cash spent once: 0 left, 10 BTC
	if math.Abs(view.CashBalance) > 1e-6 {
		t.Fatalf("cash=%v want 0", view.CashBalance)
	}
	if len(view.Positions) != 1 || math.Abs(view.Positions[0].Quantity-10) > 1e-6 {
		t.Fatalf("positions=%+v", view.Positions)
	}
}