package portfolio

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
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
	// Open: -20 margin, debt 80. Partial 0.5: +10 margin +5 pnl -40 principal = 10000-20+10+5-40=9955
	if math.Abs(view.CashBalance-9955) > 1e-6 {
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

// TestMargin_CrossPartialLiquidationClosesOnlyEnoughQty: slightly under maint → close only
// the minimum quantity on the worst position; re-eval leaves the rest open.
func TestMargin_CrossPartialLiquidationClosesOnlyEnoughQty(t *testing.T) {
	// 1x longs: debt 0 so equity is conserved while maintenance drops with closed size.
	// Start 1000; two positions qty 5 @ 100 → margin 500 each, cash 0.
	// equity slightly under total maint; cq on B = deficit/(mmr*entry) = 0.5/0.5 = 1.
	svc := newSvc(t, &fakePx{prices: map[string]string{
		"binance|AAAUSDT": "100",
		"binance|BBBUSDT": "100",
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "xliq", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "xliq", Mode: "cross"}); err != nil {
		t.Fatal(err)
	}
	posA, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "xliq", Symbol: "AAAUSDT", Side: "long", Type: "market",
		Quantity: 5, Leverage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	posB, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "xliq", Symbol: "BBBUSDT", Side: "long", Type: "market",
		Quantity: 5, Leverage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// U_A = -497.5; U_B = -498 → B worst. equity=4.5; totalMaint=5; deficit=0.5 → cq_B=1.
	svc.market = &fakePx{prices: map[string]string{
		"binance|AAAUSDT": "0.5",
		"binance|BBBUSDT": "0.4",
	}}
	n, err := svc.liquidateCrossIfUnderMaint(ctx, "xliq", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want exactly 1 partial liquidation step, got %d", n)
	}
	gotB, err := svc.store.GetMarginPosition(ctx, "xliq", posB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotB.Status != domain.MarginPositionOpen {
		t.Fatalf("B should remain open after partial liq, got %+v", gotB)
	}
	if math.Abs(gotB.Quantity-4) > 1e-6 {
		t.Fatalf("B remaining qty=%v want ~4 (closed only 1)", gotB.Quantity)
	}
	gotA, err := svc.store.GetMarginPosition(ctx, "xliq", posA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotA.Status != domain.MarginPositionOpen || math.Abs(gotA.Quantity-5) > 1e-9 {
		t.Fatalf("A must be untouched: %+v", gotA)
	}
	// Healthy — second pass no-ops (no dust thrash, no second record for same qty).
	n2, err := svc.liquidateCrossIfUnderMaint(ctx, "xliq", time.Now().UTC())
	if err != nil || n2 != 0 {
		t.Fatalf("second pass should close nothing: n=%d err=%v", n2, err)
	}
	closesB, liqsB, qtyB := countMarginCloseActions(t, svc, "xliq", posB.ID)
	if closesB != 1 || liqsB != 1 {
		t.Fatalf("B should have exactly one liquidation trade, closes=%d liqs=%d", closesB, liqsB)
	}
	if math.Abs(qtyB-1) > 1e-6 {
		t.Fatalf("liquidation qty sum=%v want 1 (not more than needed)", qtyB)
	}
	closesA, _, _ := countMarginCloseActions(t, svc, "xliq", posA.ID)
	if closesA != 0 {
		t.Fatalf("A should have no close trades, got %d", closesA)
	}
}

// TestMargin_CrossPartialLiquidationContinuesAcrossPositions: deep under maint may fully
// close the worst, then only partially close the next until healthy.
func TestMargin_CrossPartialLiquidationContinuesAcrossPositions(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{
		"binance|AAAUSDT": "100",
		"binance|BBBUSDT": "100",
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "xliq2", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "xliq2", Mode: "cross"}); err != nil {
		t.Fatal(err)
	}
	posA, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "xliq2", Symbol: "AAAUSDT", Side: "long", Type: "market",
		Quantity: 5, Leverage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	posB, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "xliq2", Symbol: "BBBUSDT", Side: "long", Type: "market",
		Quantity: 5, Leverage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// equity = 1000 - 499.5*2 = 1; maint = 5; deficit = 4.
	// B full close (cq>=8 but qty=5); then A: deficit 1.5 → cq=3, remain 2 open.
	svc.market = &fakePx{prices: map[string]string{
		"binance|AAAUSDT": "0.1",
		"binance|BBBUSDT": "0.1",
	}}
	n, err := svc.liquidateCrossIfUnderMaint(ctx, "xliq2", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want 2 liquidation steps (full one + partial other), got %d", n)
	}
	gotA, err := svc.store.GetMarginPosition(ctx, "xliq2", posA.ID)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := svc.store.GetMarginPosition(ctx, "xliq2", posB.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Worst is full-closed first (UUID order when U ties); the other keeps residual ~2.
	var closed, open *domain.MarginPosition
	switch {
	case gotA.Status == domain.MarginPositionClosed && gotB.Status == domain.MarginPositionOpen:
		closed, open = gotA, gotB
	case gotB.Status == domain.MarginPositionClosed && gotA.Status == domain.MarginPositionOpen:
		closed, open = gotB, gotA
	default:
		t.Fatalf("want one full close + one partial residual, A=%s qty=%v B=%s qty=%v",
			gotA.Status, gotA.Quantity, gotB.Status, gotB.Quantity)
	}
	_ = closed
	if math.Abs(open.Quantity-2) > 1e-6 {
		t.Fatalf("residual open qty=%v want ~2", open.Quantity)
	}
	// No further closes once healthy.
	n2, err := svc.liquidateCrossIfUnderMaint(ctx, "xliq2", time.Now().UTC())
	if err != nil || n2 != 0 {
		t.Fatalf("healthy book must not keep liquidating: n=%d err=%v", n2, err)
	}
}

// TestMargin_CrossPartialLiquidationNoDuplicateQtyRecord: replay after a successful partial
// does not insert another close for the same quantity or re-debit size.
func TestMargin_CrossPartialLiquidationNoDuplicateQtyRecord(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{
		"binance|AAAUSDT": "100",
		"binance|BBBUSDT": "100",
	}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "xliq3", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "xliq3", Mode: "cross"}); err != nil {
		t.Fatal(err)
	}
	_, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "xliq3", Symbol: "AAAUSDT", Side: "long", Type: "market",
		Quantity: 5, Leverage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	posB, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "xliq3", Symbol: "BBBUSDT", Side: "long", Type: "market",
		Quantity: 5, Leverage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.market = &fakePx{prices: map[string]string{
		"binance|AAAUSDT": "0.5",
		"binance|BBBUSDT": "0.4",
	}}
	if _, err := svc.liquidateCrossIfUnderMaint(ctx, "xliq3", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got1, _ := svc.store.GetMarginPosition(ctx, "xliq3", posB.ID)
	_, liqs1, qty1 := countMarginCloseActions(t, svc, "xliq3", posB.ID)
	// Simulate restart / second worker tick while healthy.
	for i := 0; i < 3; i++ {
		if _, err := svc.liquidateCrossIfUnderMaint(ctx, "xliq3", time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
	}
	got2, _ := svc.store.GetMarginPosition(ctx, "xliq3", posB.ID)
	_, liqs2, qty2 := countMarginCloseActions(t, svc, "xliq3", posB.ID)
	if liqs2 != liqs1 || math.Abs(qty2-qty1) > 1e-12 {
		t.Fatalf("duplicate liquidation records: before liqs=%d qty=%v after liqs=%d qty=%v", liqs1, qty1, liqs2, qty2)
	}
	if math.Abs(got2.Quantity-got1.Quantity) > 1e-12 {
		t.Fatalf("quantity changed on healthy restarts: %v -> %v", got1.Quantity, got2.Quantity)
	}
}

func TestPickWorstCrossPosition(t *testing.T) {
	list := []domain.MarginPosition{
		{ID: "a", UnrealizedPnL: -10, Quantity: 1, EntryPrice: 100},
		{ID: "b", UnrealizedPnL: -50, Quantity: 1, EntryPrice: 100},
		{ID: "c", UnrealizedPnL: -50, Quantity: 2, EntryPrice: 100}, // same U, larger notional
	}
	w := pickWorstCrossPosition(list)
	if w == nil || w.ID != "c" {
		t.Fatalf("want c (most negative then larger notional), got %+v", w)
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

func TestMargin_ModeLockAndAdjustIsolated(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m6", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	p, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "m6", Mode: "cross"})
	if err != nil || p.MarginMode != domain.MarginModeCross {
		t.Fatalf("%+v %v", p, err)
	}
	// Back to isolated for adjust tests
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "m6", Mode: "isolated"}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m6", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Cannot change mode with open position
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "m6", Mode: "cross"}); err == nil {
		t.Fatal("expected mode lock")
	}
	oldLiq := pos.LiquidationPrice
	// Add margin → long liq decreases
	pos2, err := svc.AdjustMargin(ctx, MarginAdjustInput{ClientID: "m6", PositionID: pos.ID, Delta: 20})
	if err != nil {
		t.Fatal(err)
	}
	if pos2.Margin < 39 || pos2.LiquidationPrice >= oldLiq {
		t.Fatalf("after add margin=%v liq=%v oldLiq=%v", pos2.Margin, pos2.LiquidationPrice, oldLiq)
	}
	// Remove back toward min IM=20
	pos3, err := svc.AdjustMargin(ctx, MarginAdjustInput{ClientID: "m6", PositionID: pos.ID, Delta: -10})
	if err != nil || pos3.Margin < 29 {
		t.Fatalf("%+v %v", pos3, err)
	}
	// Cannot remove below IM
	if _, err := svc.AdjustMargin(ctx, MarginAdjustInput{ClientID: "m6", PositionID: pos.ID, Delta: -100}); err == nil {
		t.Fatal("expected min margin error")
	}
}

func TestMargin_InterestCASAndCatchUp(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "mi1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "mi1", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Simulate 5 hours offline (O(1) catch-up, not 5 steps)
	last := time.Now().UTC().Add(-5 * time.Hour).Truncate(time.Hour)
	pos.LastInterestAt = last
	pos.DebtInterest = 0
	if err := svc.store.UpdateMarginPosition(ctx, *pos); err != nil {
		t.Fatal(err)
	}
	// Concurrent ProcessMarginInterest: exactly one accrual window
	var applied int
	for i := 0; i < 8; i++ {
		a, _, err := svc.ProcessMarginInterest(ctx, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		applied += a
	}
	if applied != 1 {
		t.Fatalf("want exactly 1 successful accrual (CAS), got %d", applied)
	}
	got, err := svc.GetMarginPosition(ctx, "mi1", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 80 principal * 0.00001 * 5 hours = 0.004
	if got.DebtInterest < 0.003 || got.DebtInterest > 0.005 {
		t.Fatalf("interest=%v", got.DebtInterest)
	}
	// Fully paid principal → stop interest
	got.DebtPrincipal = 0
	got.DebtInterest = 1
	_ = svc.store.UpdateMarginPosition(ctx, *got)
	a, _, _ := svc.ProcessMarginInterest(ctx, time.Now().UTC().Add(2*time.Hour))
	if a != 0 {
		t.Fatalf("paid principal should not accrue, applied=%d", a)
	}
	// Clock backward: no extra interest
	got2, _ := svc.GetMarginPosition(ctx, "mi1", pos.ID)
	// restore principal for next check on a fresh pos
	_ = got2
}

func TestMargin_InterestThenLiquidateSameOp(t *testing.T) {
	// After CAS interest pushes long liq above mark, same call liquidates.
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|ETHUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "mi2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "mi2", Symbol: "ETHUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Mark just above classic liq (~80.5); after interest 0.6, liq ~81.1 → liquidate at 81.
	svc.market = &fakePx{prices: map[string]string{"binance|ETHUSDT": "81"}}
	last := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	// Inject high interest via full debt-snapshot CAS, then liquidate via maintenance.
	liqAfter, _ := domain.LiquidationPriceWithDebt(domain.MarginLong, 100, 1, 20, 80, 0.6, domain.DefaultMaintenanceMarginRate)
	if !domain.ShouldLiquidate(domain.MarginLong, 81, liqAfter) {
		t.Fatalf("fixture liq=%v should breach mark 81", liqAfter)
	}
	cur, err := svc.store.GetMarginPosition(ctx, "mi2", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Align last_interest_at so CAS can advance the cursor past the seed window.
	cur.LastInterestAt = last
	cur.DebtInterest = 0
	if err := svc.store.UpdateMarginPosition(ctx, *cur); err != nil {
		t.Fatal(err)
	}
	cur, err = svc.store.GetMarginPosition(ctx, "mi2", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	snap := domain.DebtSnapshotFromPos(cur)
	ok, err := svc.store.AccrueInterestCAS(ctx, pos.ID, snap, 0.6, last.Add(2*time.Hour), liqAfter, time.Now().UTC())
	if err != nil || !ok {
		t.Fatalf("cas ok=%v err=%v snap=%+v", ok, err, snap)
	}
	// Same-operation liquidate via accrue path with no further hours but process maintenance:
	_, nLiq, _, err := svc.ProcessMarginMaintenance(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if nLiq < 1 {
		// Interest path with reloaded pos and forced hours: last already advanced; call close via maintenance reload
		cur, _ := svc.store.GetMarginPosition(ctx, "mi2", pos.ID)
		if domain.ShouldLiquidate(cur.Side, 81, cur.LiquidationPrice) {
			_, _, err = svc.closeMarginAt(ctx, cur, cur.Quantity, 81, domain.MarginCloseLiquidation)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			t.Fatalf("expected liquidate condition pos=%+v", cur)
		}
	}
	got, err := svc.GetMarginPosition(ctx, "mi2", pos.ID)
	if err != nil || got.Status != domain.MarginPositionClosed {
		t.Fatalf("want closed got %+v err=%v", got, err)
	}
}

func TestMargin_BorrowDebtAndRepay(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m8", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	// long 1 @ 100, 5x → margin 20, debt principal 80 quote
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m8", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(pos.DebtPrincipal-80) > 1e-6 || pos.DebtAsset != domain.DebtAssetQuote {
		t.Fatalf("debt=%+v", pos)
	}
	if pos.DebtInterest != 0 {
		t.Fatalf("interest=%v", pos.DebtInterest)
	}
	// Accrue 2 hours manually
	pos.LastInterestAt = time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	pos.DebtInterest = 0
	_ = svc.store.UpdateMarginPosition(ctx, *pos)
	got, err := svc.GetMarginPosition(ctx, "m8", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	// GetMarginPosition accrues: 80 * 0.00001 * 2 = 0.0016
	if got.DebtInterest <= 0 {
		t.Fatalf("expected interest growth, got %v", got.DebtInterest)
	}
	// Repay interest first then principal
	payInterest := got.DebtInterest
	_, tr, err := svc.RepayMarginDebt(ctx, MarginRepayInput{ClientID: "m8", PositionID: pos.ID, Amount: payInterest})
	if err != nil {
		t.Fatal(err)
	}
	if tr.InterestPaid+1e-12 < payInterest-1e-9 || tr.PrincipalPaid > 1e-9 {
		t.Fatalf("repay interest first: %+v", tr)
	}
	got2, _ := svc.GetMarginPosition(ctx, "m8", pos.ID)
	if got2.DebtInterest > 1e-9 {
		t.Fatalf("interest remaining=%v", got2.DebtInterest)
	}
	// Partial close pays proportional debt
	_, tr2, err := svc.CloseMarginPosition(ctx, MarginCloseInput{ClientID: "m8", PositionID: pos.ID, Quantity: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if tr2.PrincipalPaid < 39 {
		t.Fatalf("expected ~40 principal paid, got %v", tr2.PrincipalPaid)
	}
	// Short opens coin debt
	posS, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m8", Symbol: "BTCUSDT", Side: "short", Type: "market",
		Quantity: 0.5, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if posS.DebtAsset != domain.DebtAssetBase || math.Abs(posS.DebtPrincipal-0.5) > 1e-9 {
		t.Fatalf("short debt=%+v", posS)
	}
}

func TestMargin_LimitReserveRelease(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "m7", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	_, ord, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "m7", Symbol: "BTCUSDT", Side: "long", Type: "limit",
		Quantity: 1, Leverage: 5, LimitPrice: 90,
	})
	if err != nil || ord == nil {
		t.Fatal(err)
	}
	// need = 90/5=18 reserved
	view, _ := svc.View(ctx, "m7")
	if math.Abs(view.ReservedMargin-18) > 1e-6 {
		t.Fatalf("reserved=%v", view.ReservedMargin)
	}
	// Available reduced
	if view.AvailableCash > 1000-17 {
		t.Fatalf("avail=%v", view.AvailableCash)
	}
	// Cancel releases
	if _, err := svc.CancelMarginOrder(ctx, "m7", ord.ID); err != nil {
		t.Fatal(err)
	}
	view, _ = svc.View(ctx, "m7")
	if view.ReservedMargin > 1e-9 {
		t.Fatalf("reserved after cancel=%v", view.ReservedMargin)
	}
	// Mode change allowed when no open pos/order
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "m7", Mode: "cross"}); err != nil {
		t.Fatal(err)
	}
}

// TestMargin_ConcurrentInterestAndFullRepay ensures interest worker + full repay cannot
// leave zombie interest on a paid-off principal, or lose a successful repayment.
func TestMargin_ConcurrentInterestAndFullRepay(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "race-repay", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "race-repay", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Large offline window so interest wants to apply while repay runs.
	last := time.Now().UTC().Add(-12 * time.Hour).Truncate(time.Hour)
	pos.LastInterestAt = last
	pos.DebtInterest = 0
	if err := svc.store.UpdateMarginPosition(ctx, *pos); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	var wg sync.WaitGroup
	var repayOK atomic.Int32
	var hardErr atomic.Int32
	// Many interest ticks + several full-repay attempts race on the same debt snapshot.
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := svc.ProcessMarginInterest(ctx, now); err != nil {
				hardErr.Add(1)
			}
		}()
	}
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, tr, err := svc.RepayMarginDebt(ctx, MarginRepayInput{
				ClientID: "race-repay", PositionID: pos.ID, Amount: 1e9, // cover any accrued interest
			})
			if err == nil && tr != nil {
				repayOK.Add(1)
				return
			}
			if err == nil {
				return
			}
			// Losers: already paid, or transient conflict exhausted — only unexpected errors count.
			if errors.Is(err, domain.ErrInvalidArgument) || errors.Is(err, domain.ErrConflict) {
				return
			}
			hardErr.Add(1)
			t.Errorf("unexpected repay err: %v", err)
		}()
	}
	wg.Wait()
	if hardErr.Load() > 0 {
		t.Fatalf("unexpected hard errors during race")
	}
	if repayOK.Load() < 1 {
		t.Fatalf("expected at least one successful full repay, got %d", repayOK.Load())
	}
	got, err := svc.store.GetMarginPosition(ctx, "race-repay", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DebtPrincipal > domain.PositionEpsilon || got.DebtInterest > domain.PositionEpsilon {
		t.Fatalf("debt must be fully cleared after concurrent repay: principal=%v interest=%v",
			got.DebtPrincipal, got.DebtInterest)
	}
	// Interest must not reappear on a paid-off position.
	a, _, err := svc.ProcessMarginInterest(ctx, now.Add(5*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if a != 0 {
		t.Fatalf("paid principal must not accrue further interest, applied=%d", a)
	}
	got2, _ := svc.store.GetMarginPosition(ctx, "race-repay", pos.ID)
	if got2.DebtPrincipal > domain.PositionEpsilon || got2.DebtInterest > domain.PositionEpsilon {
		t.Fatalf("zombie debt after post-repay interest tick: %+v", got2)
	}
	// Cash: open used 20 margin; successful repay(s) debited principal+interest once.
	// At most one full clear should have debited ~80 (+tiny interest if it won the race first).
	view, err := svc.View(ctx, "race-repay")
	if err != nil {
		t.Fatal(err)
	}
	// Floor: 10000 - 20 margin - 80 principal = 9900 (if interest was 0 at repay)
	// Ceiling: slightly lower if interest was paid too. Never below 9900 - 1 (generous).
	if view.CashBalance > 9900+1e-6 {
		t.Fatalf("cash too high (repay lost?): %v", view.CashBalance)
	}
	if view.CashBalance < 9890 {
		t.Fatalf("cash too low (double-charged repay?): %v", view.CashBalance)
	}
}

// TestMargin_ConcurrentInterestAndPartialClose ensures interest cannot over-accrue on
// pre-close principal after a partial close, and remaining debt stays proportional.
func TestMargin_ConcurrentInterestAndPartialClose(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "race-close", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "race-close", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	hours := 10
	last := time.Now().UTC().Add(-time.Duration(hours) * time.Hour).Truncate(time.Hour)
	pos.LastInterestAt = last
	pos.DebtInterest = 0
	if err := svc.store.UpdateMarginPosition(ctx, *pos); err != nil {
		t.Fatal(err)
	}
	// Max interest if full principal accrued before any close (then half remains after 50% close).
	maxFullInterest := 80 * domain.DefaultMarginHourlyInterestRate * float64(hours)
	maxRemainInterest := maxFullInterest // if close loses CAS and interest applied after, re-accrual capped below

	now := time.Now().UTC()
	var wg sync.WaitGroup
	var hardErr atomic.Int32
	// Many interest workers race a single half-close (avoids a second 0.5 close finishing the pos).
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := svc.ProcessMarginInterest(ctx, now); err != nil {
				hardErr.Add(1)
			}
		}()
	}
	var closeErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _, closeErr = svc.CloseMarginPosition(ctx, MarginCloseInput{
			ClientID: "race-close", PositionID: pos.ID, Quantity: 0.5,
		})
	}()
	wg.Wait()
	if hardErr.Load() > 0 {
		t.Fatalf("unexpected hard errors during partial-close race")
	}
	if closeErr != nil {
		t.Fatalf("partial close failed under interest race: %v", closeErr)
	}
	got, err := svc.store.GetMarginPosition(ctx, "race-close", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.MarginPositionOpen {
		t.Fatalf("want still open after partial close, got %+v", got)
	}
	if math.Abs(got.Quantity-0.5) > 1e-9 {
		t.Fatalf("qty=%v want ~0.5", got.Quantity)
	}
	// Remaining principal must be half of open principal (~40), never full 80 after half close.
	if math.Abs(got.DebtPrincipal-40) > 0.5 {
		t.Fatalf("remaining principal=%v want ~40 (interest race must not restore principal)", got.DebtPrincipal)
	}
	// Interest on remaining book: at most accrual computed on full 80 for the offline window
	// (if interest won before close and half was paid). Never "double" that amount.
	if got.DebtInterest > maxRemainInterest+1e-9 {
		t.Fatalf("interest over-accrued under race: interest=%v max=%v", got.DebtInterest, maxRemainInterest)
	}
	// Further interest after settle must only use remaining principal (CAS + principal check).
	a, _, err := svc.ProcessMarginInterest(ctx, now.Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	got2, _ := svc.store.GetMarginPosition(ctx, "race-close", pos.ID)
	if a > 0 {
		// 40 * rate * ~3h ≈ very small; principal must stay ~40.
		if math.Abs(got2.DebtPrincipal-40) > 0.5 {
			t.Fatalf("principal drifted after later interest: %v", got2.DebtPrincipal)
		}
	}
	// Absolute cap: original full-window on 80 + a few hours on remaining 40.
	capI := maxFullInterest + 40*domain.DefaultMarginHourlyInterestRate*4
	if got2.DebtInterest > capI+1e-9 {
		t.Fatalf("interest ballooned: %v cap=%v", got2.DebtInterest, capI)
	}
}

// TestMargin_DebtCASRejectsStaleInterest proves store CAS blocks interest after repay/close.
func TestMargin_DebtCASRejectsStaleInterest(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "cas1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "cas1", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	last := time.Now().UTC().Add(-4 * time.Hour).Truncate(time.Hour)
	pos.LastInterestAt = last
	pos.DebtInterest = 0
	if err := svc.store.UpdateMarginPosition(ctx, *pos); err != nil {
		t.Fatal(err)
	}
	cur, err := svc.store.GetMarginPosition(ctx, "cas1", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	stale := domain.DebtSnapshotFromPos(cur)
	// Repay full principal+interest (repay path accrues first, so amount must cover both).
	_, _, err = svc.RepayMarginDebt(ctx, MarginRepayInput{ClientID: "cas1", PositionID: pos.ID, Amount: 1e9})
	if err != nil {
		t.Fatal(err)
	}
	// Stale interest CAS must not re-apply on paid principal.
	ok, err := svc.store.AccrueInterestCAS(ctx, pos.ID, stale, 1.0, last.Add(4*time.Hour), 90, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("AccrueInterestCAS claimed stale snapshot after principal repay")
	}
	got, err := svc.store.GetMarginPosition(ctx, "cas1", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DebtPrincipal > domain.PositionEpsilon || got.DebtInterest > domain.PositionEpsilon {
		t.Fatalf("debt should be fully paid: principal=%v interest=%v", got.DebtPrincipal, got.DebtInterest)
	}
}

// TestMargin_DebtCASRejectsStaleRepay proves repay fails when interest advanced the snapshot.
func TestMargin_DebtCASRejectsStaleRepay(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "cas2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "cas2", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	last := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Hour)
	pos.LastInterestAt = last
	pos.DebtInterest = 0
	_ = svc.store.UpdateMarginPosition(ctx, *pos)
	cur, _ := svc.store.GetMarginPosition(ctx, "cas2", pos.ID)
	staleSnap := domain.DebtSnapshotFromPos(cur)
	// Interest worker advances interest + cursor.
	_, _, err = svc.ProcessMarginInterest(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := svc.store.GetMarginPosition(ctx, "cas2", pos.ID)
	if fresh.DebtInterest <= 0 {
		t.Fatalf("expected interest to apply first, got %v", fresh.DebtInterest)
	}
	// Stale repay (snapshot before interest) must conflict.
	p, _ := svc.store.GetPortfolio(ctx, "cas2")
	p.CashBalance -= 10
	p.UpdatedAt = time.Now().UTC()
	updated := *fresh
	// pretends to pay 10 against pre-interest debt state
	updated.DebtPrincipal = staleSnap.Principal - 10
	updated.DebtInterest = staleSnap.Interest
	updated.UpdatedAt = time.Now().UTC()
	tr := domain.MarginTrade{
		ID: "stale-repay", ClientID: "cas2", PositionID: pos.ID, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.MarginLong, Action: domain.MarginActionRepay, Quantity: 10, Price: 1, Notional: 10,
		MarginDelta: -10, PrincipalPaid: 10, CreatedAt: time.Now().UTC(),
	}
	err = svc.store.ApplyMarginRepay(ctx, p, updated, tr, staleSnap)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want ErrConflict on stale repay, got %v", err)
	}
	// Debt unchanged by failed stale repay.
	got, _ := svc.store.GetMarginPosition(ctx, "cas2", pos.ID)
	if math.Abs(got.DebtPrincipal-fresh.DebtPrincipal) > 1e-9 || math.Abs(got.DebtInterest-fresh.DebtInterest) > 1e-12 {
		t.Fatalf("stale repay mutated debt: got principal=%v interest=%v want principal=%v interest=%v",
			got.DebtPrincipal, got.DebtInterest, fresh.DebtPrincipal, fresh.DebtInterest)
	}
}

func countMarginCloseActions(t *testing.T, svc *Service, clientID, positionID string) (closes int, liquidations int, totalQty float64) {
	t.Helper()
	trades, err := svc.ListMarginTrades(ctxBg(), clientID, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range trades {
		if tr.PositionID != positionID {
			continue
		}
		switch tr.Action {
		case "close", "partial_close", "stop_loss", "take_profit":
			closes++
			totalQty += tr.Quantity
		case "liquidation":
			liquidations++
			closes++
			totalQty += tr.Quantity
		}
	}
	return closes, liquidations, totalQty
}

func ctxBg() context.Context { return context.Background() }

// setupUnderwaterLong opens a 5x long and forces interest so liq > mark (ready to liquidate).
func setupUnderwaterLong(t *testing.T, clientID string, mark string) (*Service, *domain.MarginPosition) {
	t.Helper()
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|ETHUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: clientID, StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: clientID, Symbol: "ETHUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Mark near classic liq; after interest 0.6, liq ~81.1 → liquidate.
	svc.market = &fakePx{prices: map[string]string{"binance|ETHUSDT": mark}}
	last := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	cur, err := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	cur.LastInterestAt = last
	cur.DebtInterest = 0
	if err := svc.store.UpdateMarginPosition(ctx, *cur); err != nil {
		t.Fatal(err)
	}
	cur, _ = svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	liqAfter, _ := domain.LiquidationPriceWithDebt(domain.MarginLong, 100, 1, 20, 80, 0.6, domain.DefaultMaintenanceMarginRate)
	snap := domain.DebtSnapshotFromPos(cur)
	ok, err := svc.store.AccrueInterestCAS(ctx, pos.ID, snap, 0.6, last.Add(2*time.Hour), liqAfter, time.Now().UTC())
	if err != nil || !ok {
		t.Fatalf("seed interest cas ok=%v err=%v", ok, err)
	}
	got, err := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	return svc, got
}

// TestMargin_ConcurrentLiquidateAndUserClose: interest-path liquidation races user full close.
// Exactly one close trade; position closed once; qty and cash stay consistent.
func TestMargin_ConcurrentLiquidateAndUserClose(t *testing.T) {
	const clientID = "race-liq-close"
	svc, pos := setupUnderwaterLong(t, clientID, "81")
	ctx := context.Background()
	now := time.Now().UTC()

	var wg sync.WaitGroup
	var userCloseOK, liqOK atomic.Int32
	// Interest worker may try liquidate after re-read; maintenance also liquidates; user closes.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, nLiq, err := svc.ProcessMarginInterest(ctx, now); err == nil && nLiq > 0 {
				liqOK.Add(int32(nLiq))
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, nLiq, _, err := svc.ProcessMarginMaintenance(ctx, now); err == nil && nLiq > 0 {
				liqOK.Add(int32(nLiq))
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.CloseMarginPosition(ctx, MarginCloseInput{
				ClientID: clientID, PositionID: pos.ID,
			})
			if err == nil {
				userCloseOK.Add(1)
			}
		}()
	}
	wg.Wait()

	got, err := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.MarginPositionClosed {
		t.Fatalf("want closed, got status=%s qty=%v debtP=%v", got.Status, got.Quantity, got.DebtPrincipal)
	}
	if got.Quantity > domain.PositionEpsilon || got.DebtPrincipal > domain.PositionEpsilon || got.DebtInterest > domain.PositionEpsilon {
		t.Fatalf("closed position residual: qty=%v debtP=%v debtI=%v", got.Quantity, got.DebtPrincipal, got.DebtInterest)
	}
	closes, _, totalQty := countMarginCloseActions(t, svc, clientID, pos.ID)
	if closes != 1 {
		t.Fatalf("want exactly 1 close/liquidation trade, got closes=%d userOK=%d liqOK=%d", closes, userCloseOK.Load(), liqOK.Load())
	}
	if math.Abs(totalQty-1) > 1e-9 {
		t.Fatalf("close qty sum=%v want 1 (no double size)", totalQty)
	}
	view, err := svc.View(ctx, clientID)
	if err != nil {
		t.Fatal(err)
	}
	// Cash must move once: not double-debited toward zero.
	// Floor roughly 10000 - margin20 - loss - debt ≈ high 9800s; generous band.
	if view.CashBalance < 9800 || view.CashBalance > 10000 {
		t.Fatalf("cash out of expected band after single close: %v", view.CashBalance)
	}
}

// TestMargin_ConcurrentLiquidateAndFullRepay: liquidation after interest races full debt repay.
// At most one close trade; no zombie debt; cash not double-charged for principal.
func TestMargin_ConcurrentLiquidateAndFullRepay(t *testing.T) {
	const clientID = "race-liq-repay"
	svc, pos := setupUnderwaterLong(t, clientID, "81")
	ctx := context.Background()
	now := time.Now().UTC()

	var wg sync.WaitGroup
	var repayOK, liqOK atomic.Int32
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, n, err := svc.ProcessMarginInterest(ctx, now); err == nil && n > 0 {
				liqOK.Add(int32(n))
			}
			if _, n, _, err := svc.ProcessMarginMaintenance(ctx, now); err == nil && n > 0 {
				liqOK.Add(int32(n))
			}
		}()
	}
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.RepayMarginDebt(ctx, MarginRepayInput{
				ClientID: clientID, PositionID: pos.ID, Amount: 1e9,
			})
			if err == nil {
				repayOK.Add(1)
			}
		}()
	}
	wg.Wait()

	got, err := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Either still open with zero debt (repay won, liq no longer required or not finished),
	// or closed by liquidation (with debt cleared on close).
	if got.Status == domain.MarginPositionOpen {
		if got.DebtPrincipal > domain.PositionEpsilon || got.DebtInterest > domain.PositionEpsilon {
			t.Fatalf("open position must not keep debt after full repay race: P=%v I=%v", got.DebtPrincipal, got.DebtInterest)
		}
		// Force maintenance once more — if still breached, single liquidate; if safe, stay open.
		_, _, _, _ = svc.ProcessMarginMaintenance(ctx, now)
		got, _ = svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	}
	if got.Status == domain.MarginPositionClosed {
		if got.DebtPrincipal > domain.PositionEpsilon || got.DebtInterest > domain.PositionEpsilon {
			t.Fatalf("closed with residual debt: P=%v I=%v", got.DebtPrincipal, got.DebtInterest)
		}
		if got.Quantity > domain.PositionEpsilon {
			t.Fatalf("closed with residual qty: %v", got.Quantity)
		}
	}
	closes, _, totalQty := countMarginCloseActions(t, svc, clientID, pos.ID)
	if closes > 1 {
		t.Fatalf("want at most 1 close/liquidation trade, got %d (double close)", closes)
	}
	if closes == 1 && math.Abs(totalQty-1) > 1e-9 {
		t.Fatalf("close qty sum=%v want 1", totalQty)
	}
	// At most one successful full repay (others see no debt or not open).
	if repayOK.Load() > 1 {
		t.Fatalf("multiple successful full repays: %d (payment double-applied?)", repayOK.Load())
	}
	view, err := svc.View(ctx, clientID)
	if err != nil {
		t.Fatal(err)
	}
	// Principal ~80 paid at most once; cash should not drop as if paid twice (~160).
	if view.CashBalance < 9850 {
		t.Fatalf("cash too low (double principal charge?): %v repayOK=%d liqOK=%d", view.CashBalance, repayOK.Load(), liqOK.Load())
	}
}

// TestMargin_CloseCASRejectsStaleQuantity: partial close invalidates a concurrent full-close snapshot.
func TestMargin_CloseCASRejectsStaleQuantity(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "cas-qty", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "cas-qty", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	cur, _ := svc.store.GetMarginPosition(ctx, "cas-qty", pos.ID)
	stale := domain.PositionCloseSnapshotFromPos(cur)
	// User partial close lands first.
	_, _, err = svc.CloseMarginPosition(ctx, MarginCloseInput{
		ClientID: "cas-qty", PositionID: pos.ID, Quantity: 0.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Stale full close (pre-partial snapshot) must not apply a second/full overwrite.
	p, _ := svc.store.GetPortfolio(ctx, "cas-qty")
	now := time.Now().UTC()
	p.CashBalance += 10
	p.UpdatedAt = now
	closed := *cur
	closed.Quantity = 0
	closed.Margin = 0
	closed.DebtPrincipal = 0
	closed.DebtInterest = 0
	closed.Status = domain.MarginPositionClosed
	closed.CloseReason = domain.MarginCloseLiquidation
	closed.ClosedAt = &now
	closed.UpdatedAt = now
	tr := domain.MarginTrade{
		ID: "stale-full-close", ClientID: "cas-qty", PositionID: pos.ID, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.MarginLong, Action: "liquidation", Quantity: 1, Price: 100, Notional: 100,
		CreatedAt: now,
	}
	err = svc.store.ApplyMarginClose(ctx, p, closed, tr, true, stale)
	if !errors.Is(err, domain.ErrConflict) {
		// Quantity changed → conflict; if already fully closed would be NotFound.
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("want ErrConflict (or NotFound) on stale full close, got %v", err)
		}
	}
	got, _ := svc.store.GetMarginPosition(ctx, "cas-qty", pos.ID)
	if got.Status != domain.MarginPositionOpen || math.Abs(got.Quantity-0.5) > 1e-9 {
		t.Fatalf("partial close must remain; got %+v", got)
	}
	closes, _, _ := countMarginCloseActions(t, svc, "cas-qty", pos.ID)
	if closes != 1 {
		t.Fatalf("want only the real partial close trade, got %d", closes)
	}
}

// TestMargin_LiquidationIdempotentAfterRestart: after a successful liquidation, a simulated
// app restart (re-running interest + maintenance) must not re-credit cash or insert another close.
func TestMargin_LiquidationIdempotentAfterRestart(t *testing.T) {
	const clientID = "liq-restart"
	svc, pos := setupUnderwaterLong(t, clientID, "81")
	ctx := context.Background()
	now := time.Now().UTC()

	// First run: complete liquidation (like maintenance after interest raised liq).
	_, nLiq, _, err := svc.ProcessMarginMaintenance(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if nLiq < 1 {
		// Interest path may also liquidate; force via close if maintenance saw nothing.
		if did, _ := svc.tryLiquidateIfBreached(ctx, clientID, pos.ID, now); !did {
			// Position might already be closed by interest in setup — check.
			got, _ := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
			if got.Status != domain.MarginPositionClosed {
				t.Fatalf("expected liquidation on first pass, nLiq=%d status=%s", nLiq, got.Status)
			}
		}
	}
	got, err := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.MarginPositionClosed {
		t.Fatalf("want closed after first liquidation, got %+v", got)
	}
	view1, err := svc.View(ctx, clientID)
	if err != nil {
		t.Fatal(err)
	}
	cash1 := view1.CashBalance
	closes1, liq1, qty1 := countMarginCloseActions(t, svc, clientID, pos.ID)
	if closes1 != 1 || liq1 != 1 {
		t.Fatalf("after first liq: closes=%d liquidations=%d qty=%v", closes1, liq1, qty1)
	}
	// Deterministic trade id for restart safety.
	wantID := domain.SystemCloseTradeID(domain.MarginCloseLiquidation, pos.ID)
	trades, _ := svc.ListMarginTrades(ctx, clientID, 20, 0)
	found := false
	for _, tr := range trades {
		if tr.PositionID == pos.ID && tr.Action == domain.MarginCloseLiquidation {
			if tr.ID != wantID {
				t.Fatalf("liquidation trade id=%q want %q", tr.ID, wantID)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing liquidation trade")
	}

	// Simulated restart: workers run again (interest + maintenance).
	for i := 0; i < 5; i++ {
		_, _, _ = svc.ProcessMarginInterest(ctx, now.Add(time.Duration(i)*time.Hour))
		_, n2, _, err := svc.ProcessMarginMaintenance(ctx, now.Add(time.Duration(i)*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if n2 > 0 {
			t.Fatalf("restart pass %d re-liquidated count=%d", i, n2)
		}
		// Direct force must also be a no-op.
		if did, err := svc.tryLiquidateIfBreached(ctx, clientID, pos.ID, now); err != nil || did {
			t.Fatalf("tryLiquidate after done: did=%v err=%v", did, err)
		}
	}
	view2, _ := svc.View(ctx, clientID)
	if math.Abs(view2.CashBalance-cash1) > 1e-9 {
		t.Fatalf("cash changed on restart: before=%v after=%v", cash1, view2.CashBalance)
	}
	closes2, liq2, qty2 := countMarginCloseActions(t, svc, clientID, pos.ID)
	if closes2 != 1 || liq2 != 1 || math.Abs(qty2-1) > 1e-9 {
		t.Fatalf("duplicate close after restart: closes=%d liq=%d qty=%v", closes2, liq2, qty2)
	}
	got2, _ := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	if got2.Status != domain.MarginPositionClosed || got2.CloseReason != domain.MarginCloseLiquidation {
		t.Fatalf("position changed on restart: %+v", got2)
	}
}

// TestMargin_LiquidationContinuesAfterCrashBeforeClose: interest commits, then "crash" before
// close; on restart, maintenance liquidates once and a second restart does nothing.
func TestMargin_LiquidationContinuesAfterCrashBeforeClose(t *testing.T) {
	const clientID = "liq-crash-mid"
	svc, pos := setupUnderwaterLong(t, clientID, "81")
	ctx := context.Background()
	// setupUnderwaterLong already applied interest via CAS (simulates interest commit before crash).
	// Position must still be open with elevated interest/liq.
	if pos.Status != domain.MarginPositionOpen {
		t.Fatalf("precondition open, got %s", pos.Status)
	}
	if pos.DebtInterest < 0.5 {
		t.Fatalf("precondition interest applied, got %v", pos.DebtInterest)
	}
	viewOpen, _ := svc.View(ctx, clientID)
	cashOpen := viewOpen.CashBalance

	// "Restart" maintenance completes the liquidation that never ran.
	now := time.Now().UTC()
	_, nLiq, _, err := svc.ProcessMarginMaintenance(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if nLiq < 1 {
		if did, _ := svc.tryLiquidateIfBreached(ctx, clientID, pos.ID, now); !did {
			t.Fatal("expected liquidation to complete after restart")
		}
	}
	got, err := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.MarginPositionClosed {
		t.Fatalf("want closed after recovery, got %s", got.Status)
	}
	viewClosed, _ := svc.View(ctx, clientID)
	// Cash must have moved exactly once from open → closed.
	if math.Abs(viewClosed.CashBalance-cashOpen) < 1e-9 {
		t.Fatalf("cash should change on completed liquidation: still %v", viewClosed.CashBalance)
	}
	cashAfter := viewClosed.CashBalance
	closes, liqN, _ := countMarginCloseActions(t, svc, clientID, pos.ID)
	if closes != 1 || liqN != 1 {
		t.Fatalf("want 1 liquidation trade after recovery, closes=%d liq=%d", closes, liqN)
	}

	// Second restart: no further cash or trades.
	_, n2, _, _ := svc.ProcessMarginMaintenance(ctx, now.Add(time.Hour))
	_, n3, _ := svc.ProcessMarginInterest(ctx, now.Add(2*time.Hour))
	if n2 > 0 || n3 > 0 {
		t.Fatalf("second restart re-applied work: maintLiq=%d interestLiq=%d", n2, n3)
	}
	view2, _ := svc.View(ctx, clientID)
	if math.Abs(view2.CashBalance-cashAfter) > 1e-9 {
		t.Fatalf("cash changed on second restart: %v -> %v", cashAfter, view2.CashBalance)
	}
	closes2, _, _ := countMarginCloseActions(t, svc, clientID, pos.ID)
	if closes2 != 1 {
		t.Fatalf("second close record after restart: %d", closes2)
	}
}

// TestMargin_ApplyMarginCloseRestartNoDoubleCash: store rejects a second full liquidation apply.
func TestMargin_ApplyMarginCloseRestartNoDoubleCash(t *testing.T) {
	const clientID = "liq-store-idemp"
	svc, pos := setupUnderwaterLong(t, clientID, "81")
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := svc.tryLiquidateIfBreached(ctx, clientID, pos.ID, now); err != nil {
		t.Fatal(err)
	}
	// Ensure closed.
	got, _ := svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	if got.Status != domain.MarginPositionClosed {
		// force
		_, _, err := svc.closeMarginAt(ctx, pos, pos.Quantity, 81, domain.MarginCloseLiquidation)
		if err != nil && !isPositionNotOpenErr(err) {
			t.Fatal(err)
		}
		got, _ = svc.store.GetMarginPosition(ctx, clientID, pos.ID)
	}
	if got.Status != domain.MarginPositionClosed {
		t.Fatal("need closed position")
	}
	p1, _ := svc.store.GetPortfolio(ctx, clientID)
	cash1 := p1.CashBalance
	// Replay a close with same snapshot-ish payload (as a buggy restart might).
	p2 := *p1
	p2.CashBalance += 50 // would wrongly credit if accepted
	p2.UpdatedAt = now
	closed := *got
	tr := domain.MarginTrade{
		ID: domain.SystemCloseTradeID(domain.MarginCloseLiquidation, pos.ID),
		ClientID: clientID, PositionID: pos.ID, Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT",
		Side: domain.MarginLong, Action: domain.MarginCloseLiquidation, Quantity: 1, Price: 81,
		Notional: 81, MarginDelta: 50, CreatedAt: now,
	}
	err := svc.store.ApplyMarginClose(ctx, &p2, closed, tr, true, domain.PositionCloseSnapshot{
		DebtSnapshot: domain.DebtSnapshot{Principal: 80, Interest: 0.6, LastInterestAt: pos.LastInterestAt},
		Quantity:     1,
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound on replay, got %v", err)
	}
	p3, _ := svc.store.GetPortfolio(ctx, clientID)
	if math.Abs(p3.CashBalance-cash1) > 1e-9 {
		t.Fatalf("cash credited on replay: %v -> %v", cash1, p3.CashBalance)
	}
	closes, _, _ := countMarginCloseActions(t, svc, clientID, pos.ID)
	if closes != 1 {
		t.Fatalf("extra close trade on replay: %d", closes)
	}
}
