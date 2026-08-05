package portfolio

import (
	"context"
	"math"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestMargin_OpenLongPartialCloseAndPnL(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	// 1 BTC long 5x → margin 20
	pos, ord, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m1", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil || ord != nil || pos == nil {
		t.Fatalf("pos=%+v ord=%+v err=%v", pos, ord, err)
	}
	if math.Abs(pos.Margin-20) > 1e-6 || pos.Leverage != 5 {
		t.Fatalf("%+v", pos)
	}
	// liq long 5x mmr 0.5%: 100*(1-0.2+0.005)=80.5
	if math.Abs(pos.LiquidationPrice-80.5) > 1e-6 {
		t.Fatalf("liq=%v", pos.LiquidationPrice)
	}
	view, _ := svc.View(ctx, "m1")
	if math.Abs(view.CashBalance-9980) > 1e-6 {
		t.Fatalf("cash=%v", view.CashBalance)
	}
	if math.Abs(view.MarginLocked-20) > 1e-6 {
		t.Fatalf("locked=%v", view.MarginLocked)
	}

	// Price up to 110 → unrealized +10
	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "110"}}
	pos, _ = svc.GetMarginPosition(ctx, "m1", pos.ID)
	if math.Abs(pos.UnrealizedPnL-10) > 1e-6 {
		t.Fatalf("upnl=%v", pos.UnrealizedPnL)
	}

	// Partial close 0.5 → realize +5, release margin 10
	closed, tr, err := svc.CloseMarginPosition(ctx, MarginCloseInput{
		ClientID: "m1", PositionID: pos.ID, Quantity: 0.5,
	})
	if err != nil || tr == nil {
		t.Fatal(err)
	}
	if math.Abs(tr.RealizedPnL-5) > 1e-6 {
		t.Fatalf("realized=%v", tr.RealizedPnL)
	}
	if closed.Status != domain.MarginPositionOpen || math.Abs(closed.Quantity-0.5) > 1e-9 {
		t.Fatalf("%+v", closed)
	}
	view, _ = svc.View(ctx, "m1")
	// cash: 9980 + 10 margin + 5 pnl = 9995
	if math.Abs(view.CashBalance-9995) > 1e-6 {
		t.Fatalf("cash after partial=%v", view.CashBalance)
	}
}

func TestMargin_ShortLiquidation(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|ETHUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m2", StartingBalance: 5000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m2", Symbol: "ETHUSDT", Side: "short", Type: "market",
		Quantity: 1, Leverage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	// short 10x liq = 100*(1+0.1-0.005)=109.5
	if math.Abs(pos.LiquidationPrice-109.5) > 1e-6 {
		t.Fatalf("liq=%v", pos.LiquidationPrice)
	}
	// Mark above liq
	svc.market = &fakePx{prices: map[string]string{"binance|ETHUSDT": "110"}}
	_, liq, _, err := svc.ProcessMarginMaintenance(ctx, time.Now().UTC())
	if err != nil || liq != 1 {
		t.Fatalf("liq=%d err=%v", liq, err)
	}
	got, err := svc.GetMarginPosition(ctx, "m2", pos.ID)
	if err != nil || got.Status != domain.MarginPositionClosed || got.CloseReason != domain.MarginCloseLiquidation {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestMargin_StopLossAndTakeProfit(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m3", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	sl, tp := 95.0, 120.0
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m3", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 2, StopLoss: &sl, TakeProfit: &tp,
	})
	if err != nil {
		t.Fatal(err)
	}
	// SL hit
	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "94"}}
	_, _, stopped, err := svc.ProcessMarginMaintenance(ctx, time.Now().UTC())
	if err != nil || stopped != 1 {
		t.Fatalf("stopped=%d err=%v", stopped, err)
	}
	got, _ := svc.GetMarginPosition(ctx, "m3", pos.ID)
	if got.Status != domain.MarginPositionClosed || got.CloseReason != domain.MarginCloseStopLoss {
		t.Fatalf("%+v", got)
	}

	// TP on new position
	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	pos2, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m3", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 0.5, Leverage: 2, TakeProfit: &tp,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "121"}}
	_, _, stopped, err = svc.ProcessMarginMaintenance(ctx, time.Now().UTC())
	if err != nil || stopped != 1 {
		t.Fatalf("tp stopped=%d err=%v", stopped, err)
	}
	got2, _ := svc.GetMarginPosition(ctx, "m3", pos2.ID)
	if got2.CloseReason != domain.MarginCloseTakeProfit {
		t.Fatalf("%+v", got2)
	}
}

func TestMargin_LimitOrderFill(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m4", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, ord, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m4", Symbol: "BTCUSDT", Side: "long", Type: "limit",
		Quantity: 1, Leverage: 5, LimitPrice: 90,
	})
	if err != nil || pos != nil || ord == nil || ord.Status != domain.MarginOrderOpen {
		t.Fatalf("pos=%v ord=%+v err=%v", pos, ord, err)
	}
	// Not filled yet
	f, _, _, _ := svc.ProcessMarginMaintenance(ctx, time.Now().UTC())
	if f != 0 {
		t.Fatalf("filled=%d", f)
	}
	// Price drops to 90
	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "90"}}
	f, _, _, err = svc.ProcessMarginMaintenance(ctx, time.Now().UTC())
	if err != nil || f != 1 {
		t.Fatalf("filled=%d err=%v", f, err)
	}
	list, err := svc.ListMarginPositions(ctx, "m4")
	if err != nil || len(list) != 1 || math.Abs(list[0].EntryPrice-90) > 1e-9 {
		t.Fatalf("%+v %v", list, err)
	}
}

func TestMargin_InsufficientCash(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m5", StartingBalance: 10}); err != nil {
		t.Fatal(err)
	}
	// need margin 100/5=20 > 10
	_, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m5", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err == nil {
		t.Fatal("expected insufficient cash")
	}
}
