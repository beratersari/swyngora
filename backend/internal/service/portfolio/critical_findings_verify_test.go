package portfolio

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func reportFinding(t *testing.T, id, name string, confirmed bool, detail string) {
	t.Helper()
	if confirmed {
		t.Errorf("CONFIRMED %s (%s): %s", id, name, detail)
		return
	}
	t.Logf("FALSE POSITIVE / NOT REPRODUCED %s (%s): %s", id, name, detail)
}

// Finding 1: Transfer moves amount 1:1 with no currency check.
func TestVerify_TransferRejectsDifferentBookCurrency(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	usdt, err := svc.Create(ctx, CreateInput{ClientID: "fx-xfer", Name: "USDT book", StartingBalance: 10000, Currency: "USDT"})
	if err != nil {
		t.Fatal(err)
	}
	tryB, err := svc.Create(ctx, CreateInput{ClientID: "fx-xfer", Name: "TRY book", StartingBalance: 1000, Currency: "TRY"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, fromV, toV, err := svc.Transfer(ctx, TransferInput{
		ClientID: "fx-xfer", FromPortfolioID: usdt.ID, ToPortfolioID: tryB.ID, Amount: 2000,
	})
	if err == nil {
		reportFinding(t, "F1", "cross-currency transfer", true,
			fmt.Sprintf("credited 2000 TRY from 2000 USDT; dest cash=%g src=%g", toV.CashBalance, fromV.CashBalance))
		return
	}
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Logf("rejected with unexpected error (still closed): %v", err)
	}
	afterUSDT, _ := svc.View(ctx, "fx-xfer", usdt.ID)
	afterTRY, _ := svc.View(ctx, "fx-xfer", tryB.ID)
	if math.Abs(afterUSDT.CashBalance-10000) > 1e-9 || math.Abs(afterTRY.CashBalance-1000) > 1e-9 {
		t.Fatalf("balances moved on rejected transfer: usdt=%v try=%v", afterUSDT.CashBalance, afterTRY.CashBalance)
	}
	reportFinding(t, "F1", "cross-currency transfer", false, "transfer rejected: "+err.Error())
}

// Finding 2: Transfer skips the cross-margin maintenance brake that Withdraw enforces.
func TestVerify_TransferHonorsCrossMaintenance(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	main, err := svc.Create(ctx, CreateInput{ClientID: "x-xfer-m", Name: "Main", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}
	other, err := svc.Create(ctx, CreateInput{ClientID: "x-xfer-m", Name: "Park", StartingBalance: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "x-xfer-m", Mode: "cross", PortfolioID: main.ID}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "x-xfer-m", PortfolioID: main.ID, Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "80.4"
	view, err := svc.View(ctx, "x-xfer-m", main.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, _, werr := svc.Withdraw(ctx, CashMoveInput{ClientID: "x-xfer-m", PortfolioID: main.ID, Amount: view.CashBalance})
	if werr == nil {
		t.Fatal("precondition: withdraw of unused cash should be rejected")
	}
	_, _, _, _, xerr := svc.Transfer(ctx, TransferInput{
		ClientID: "x-xfer-m", FromPortfolioID: main.ID, ToPortfolioID: other.ID, Amount: view.CashBalance,
	})
	after, err := svc.View(ctx, "x-xfer-m", main.ID)
	if err != nil {
		t.Fatal(err)
	}
	if xerr == nil {
		reportFinding(t, "F2", "transfer skips cross maint", true,
			fmt.Sprintf("withdraw rejected but transfer of same cash succeeded; cash now %g", after.CashBalance))
		return
	}
	if math.Abs(after.CashBalance-view.CashBalance) > 1e-9 {
		t.Fatalf("cash changed after rejected transfer: %v -> %v", view.CashBalance, after.CashBalance)
	}
	reportFinding(t, "F2", "transfer skips cross maint", false, "transfer also rejected: "+xerr.Error())
}

// Finding 3: PlanRebalance keys holdings by asset only, so a second venue is dropped
// and execute over-buys the same coin.
func TestVerify_AllocationRebalanceDoesNotOverbuyMultiVenue(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100",
		"bybit|BTCUSDT":   "100",
	}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "alloc-mv", StartingBalance: 5000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "alloc-mv", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 10,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "alloc-mv", Exchange: "bybit", Symbol: "BTCUSDT", Side: "buy", Quantity: 10,
	}); err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateAllocationBasket(ctx, AllocationBasketCreateInput{
		ClientID: "alloc-mv", Name: "Half BTC",
		Targets: []domain.AllocationTarget{
			{Asset: "BTC", WeightPct: 50},
			{Asset: "USDT", WeightPct: 50},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	prev, err := svc.PreviewAllocationRebalance(ctx, "alloc-mv", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	var btcLine domain.AllocationLine
	for _, ln := range prev.Plan.Lines {
		if ln.Asset == "BTC" {
			btcLine = ln
		}
	}
	// Equity 5000, BTC 2000 → actual 40%, target 50% → buy ~500 notional, not ~1500.
	droppedVenue := btcLine.CurrentValue < 1500
	var buyNotional float64
	for _, leg := range prev.Plan.Legs {
		if leg.Asset == "BTC" && leg.Side == domain.TradeSideBuy {
			buyNotional += leg.Notional
		}
	}
	if droppedVenue || buyNotional > 800 {
		reportFinding(t, "F3", "allocation multi-venue overwrite", true,
			fmt.Sprintf("btc currentValue=%g buyNotional=%g actualPct=%g legs=%d",
				btcLine.CurrentValue, buyNotional, btcLine.ActualPct, len(prev.Plan.Legs)))
		return
	}
	reportFinding(t, "F3", "allocation multi-venue overwrite", false,
		fmt.Sprintf("btc currentValue=%g buyNotional=%g", btcLine.CurrentValue, buyNotional))
}

// Finding 4: isolated liquidation recomputes liq from fee-baked EntryPrice and can
// close while mark is still above the stored/displayed liquidation from open.
func TestVerify_IsolatedLiqUsesDisplayedPriceOnMaintenanceTick(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvcWithCosts(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "liq-fee", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "liq-fee", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil || pos == nil {
		t.Fatalf("open: %+v %v", pos, err)
	}
	stored := pos.LiquidationPrice
	baked, err := svc.liqPriceFor(pos.Side, pos.EntryPrice, pos.Quantity, pos.Margin, pos.DebtPrincipal, pos.DebtInterest)
	if err != nil {
		t.Fatal(err)
	}
	if baked <= stored+1e-9 {
		reportFinding(t, "F4", "fee-baked isolated liq", false,
			fmt.Sprintf("recomputed liq %g is not above stored %g (entry=%g)", baked, stored, pos.EntryPrice))
		return
	}
	// Mark between stored (displayed) liq and fee-baked recomputed liq.
	mid := (stored + baked) / 2
	px.prices["binance|BTCUSDT"] = fmt.Sprintf("%g", mid)

	_, nLiq, _, err := svc.ProcessMarginMaintenance(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	cur, err := svc.store.GetMarginPosition(ctx, "liq-fee", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	maintClosed := nLiq > 0 || cur.Status == domain.MarginPositionClosed

	did, err := svc.tryLiquidateIfBreached(ctx, "liq-fee", pos.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	cur2, _ := svc.store.GetMarginPosition(ctx, "liq-fee", pos.ID)
	directClosed := did || (cur2 != nil && cur2.Status == domain.MarginPositionClosed)

	switch {
	case maintClosed:
		reportFinding(t, "F4", "fee-baked isolated liq", true,
			fmt.Sprintf("ProcessMarginMaintenance closed while mark=%g storedLiq=%g bakedLiq=%g", mid, stored, baked))
	case directClosed && !maintClosed:
		reportFinding(t, "F4", "fee-baked isolated liq", true,
			fmt.Sprintf("tryLiquidateIfBreached closed while mark=%g storedLiq=%g bakedLiq=%g (maintenance tick did not)", mid, stored, baked))
	default:
		reportFinding(t, "F4", "fee-baked isolated liq", false,
			fmt.Sprintf("stayed open at mark=%g storedLiq=%g bakedLiq=%g", mid, stored, baked))
	}
}

// Finding 5 happy path: canceling one OCO leg must cancel the peer (store healthy).
func TestVerify_CancelOneOCOLegCancelsPeer(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "oco-cxl", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "oco-cxl", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatal(err)
	}
	tp, sl, err := svc.PlaceOCOOrder(ctx, OCOOrderInput{
		ClientID: "oco-cxl", Symbol: "BTCUSDT", Quantity: 1, TakeProfitPrice: 120, StopLossPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelPendingOrder(ctx, "oco-cxl", tp.ID); err != nil {
		t.Fatal(err)
	}
	peer, err := svc.GetPendingOrder(ctx, "oco-cxl", sl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if peer.Status == domain.PendingStatusOpen {
		reportFinding(t, "F5", "OCO peer cancel", true, "peer still open after user cancel of TP")
		return
	}
	reportFinding(t, "F5", "OCO peer cancel", false, "peer status="+string(peer.Status)+" reason="+peer.CancelReason)
}
