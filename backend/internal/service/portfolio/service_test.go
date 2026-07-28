package portfolio

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"

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