package portfolio

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestIsolatedCloseGapDoesNotTakeUnassignedCash(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "iso-gap", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "iso-gap", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 40, Leverage: 5,
	})
	if err != nil || pos == nil {
		t.Fatalf("open: %v", err)
	}
	px.prices["binance|BTCUSDT"] = "10"
	if _, _, err := svc.CloseMarginPosition(ctx, MarginCloseInput{
		ClientID: "iso-gap", PositionID: pos.ID,
	}); err != nil {
		t.Fatal(err)
	}
	view, err := svc.View(ctx, "iso-gap")
	if err != nil {
		t.Fatal(err)
	}
	if view.CashBalance < 0 {
		t.Fatalf("isolated close left negative cash %v", view.CashBalance)
	}
	// IM 800 already deducted; unassigned 200 must survive the gap.
	if math.Abs(view.CashBalance-200) > 1e-6 {
		t.Fatalf("cash=%v want 200 (unassigned preserved)", view.CashBalance)
	}
}

func TestIsolatedLiquidationGapDoesNotTakeUnassignedCash(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "iso-liq", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "iso-liq", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 40, Leverage: 5,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "10"
	if _, n, _, err := svc.ProcessMarginMaintenance(ctx, time.Now().UTC()); err != nil || n < 1 {
		t.Fatalf("liq n=%d err=%v", n, err)
	}
	view, err := svc.View(ctx, "iso-liq")
	if err != nil {
		t.Fatal(err)
	}
	if view.CashBalance < 0 {
		t.Fatalf("isolated liq left negative cash %v", view.CashBalance)
	}
	if math.Abs(view.CashBalance-200) > 1e-6 {
		t.Fatalf("cash=%v want 200", view.CashBalance)
	}
}

func TestCrossSpotBuyDoesNotSpendUnrealized(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "x-upnl", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "x-upnl", Mode: "cross"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "x-upnl", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 5, Leverage: 5,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "300"
	_, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "x-upnl", Symbol: "ETHUSDT", Side: "buy", Quantity: 15,
	})
	if err == nil {
		t.Fatal("expected spot buy funded only by wallet cash to be rejected")
	}
	view, err := svc.View(ctx, "x-upnl")
	if err != nil {
		t.Fatal(err)
	}
	if view.CashBalance < 0 {
		t.Fatalf("cash went negative: %v", view.CashBalance)
	}
	if view.CashBalance > 900+1e-6 {
		t.Fatalf("cash=%v — open should have debited IM", view.CashBalance)
	}
}

func TestCrossWithdrawKeepsMaintenance(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "x-wd", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "x-wd", Mode: "cross"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "x-wd", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "80.4"
	view, err := svc.View(ctx, "x-wd")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.Withdraw(ctx, CashMoveInput{ClientID: "x-wd", Amount: view.CashBalance})
	if err == nil {
		t.Fatal("expected withdraw of unused cash to be rejected while under/near maintenance")
	}
	after, err := svc.View(ctx, "x-wd")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(after.CashBalance-view.CashBalance) > 1e-9 {
		t.Fatalf("cash changed after rejected withdraw: %v -> %v", view.CashBalance, after.CashBalance)
	}
}
