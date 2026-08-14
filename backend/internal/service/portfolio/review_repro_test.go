package portfolio

import (
	"context"
	"math"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Review claim: after interest, tryLiquidateIfBreached uses isolated liq on a
// healthy cross book and full-closes it. Correct behavior: stay open.
func TestRepro_InterestDoesNotIsolateLiquidateHealthyCross(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "80.4"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "x-healthy", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetMarginMode(ctx, SetMarginModeInput{ClientID: "x-healthy", Mode: "cross"}); err != nil {
		t.Fatal(err)
	}
	// Open at 100 so isolated liq is 80.5; mark 80.4 would isolate-liquidate.
	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "x-healthy", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil || pos == nil {
		t.Fatalf("open: %+v %v", pos, err)
	}
	if pos.Mode != domain.MarginModeCross {
		t.Fatalf("want cross mode, got %s", pos.Mode)
	}
	if math.Abs(pos.LiquidationPrice-80.5) > 0.5 {
		// Cross display liq differs; isolated formula is 80.5. Still need debt for interest.
		t.Logf("open liq=%v isolated-would-be 80.5", pos.LiquidationPrice)
	}

	cur, err := svc.store.GetMarginPosition(ctx, "x-healthy", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	cur.LastInterestAt = time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	if err := svc.store.UpdateMarginPosition(ctx, *cur); err != nil {
		t.Fatal(err)
	}

	svc.market = &fakePx{prices: map[string]string{"binance|BTCUSDT": "80.4"}}
	accrued, liquidated, err := svc.ProcessMarginInterest(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.store.GetMarginPosition(ctx, "x-healthy", pos.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Cross equity at 80.4: cash~9980 + margin 20 + U(-19.6) ≈ 9980.4 vs maint 0.5.
	risk, rerr := svc.loadCrossAccountRisk(ctx, "x-healthy")
	if rerr != nil {
		t.Fatal(rerr)
	}
	if got.Status == domain.MarginPositionClosed {
		t.Fatalf("healthy cross long was liquidated by interest worker (accrued=%d liquidated=%d equity=%v maint=%v isolatedLiqWould=%v)",
			accrued, liquidated, risk.equity, risk.totalMaint, 80.5)
	}
	if liquidated != 0 {
		t.Fatalf("interest worker reported liquidated=%d on a healthy cross book", liquidated)
	}
}

// Review claim: a stale display update (recompute/brackets) must not rewind
// a committed AccrueInterestCAS.
func TestRepro_StaleUpdateMarginPositionDoesNotRewindInterest(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rewind", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "rewind", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	stale, err := svc.store.GetMarginPosition(ctx, "rewind", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	stale.LastInterestAt = time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Hour)
	stale.DebtInterest = 0
	if err := svc.store.UpdateMarginPosition(ctx, *stale); err != nil {
		t.Fatal(err)
	}
	stale, err = svc.store.GetMarginPosition(ctx, "rewind", pos.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = svc.ProcessMarginInterest(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	afterCAS, err := svc.store.GetMarginPosition(ctx, "rewind", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterCAS.DebtInterest <= stale.DebtInterest {
		t.Fatalf("precondition: interest should have accrued, got %v (was %v)", afterCAS.DebtInterest, stale.DebtInterest)
	}

	// What recomputeCrossLiquidations / SetMarginBrackets do: write a previously
	// loaded row back through UpdateMarginPositionMeta (no debt columns).
	if err := svc.store.UpdateMarginPositionMeta(ctx, *stale); err != nil {
		t.Fatal(err)
	}
	got, err := svc.store.GetMarginPosition(ctx, "rewind", pos.ID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.DebtInterest-afterCAS.DebtInterest) > 1e-12 {
		t.Fatalf("stale meta update rewound interest: after CAS %v, after stale write %v lastAt=%v",
			afterCAS.DebtInterest, got.DebtInterest, got.LastInterestAt)
	}
}

// Review claim: amend with remaining snapshotted before a partial fill grows
// the live order. Correct: remaining stays at post-fill leftover.
func TestRepro_AmendDoesNotRestorePreFillRemaining(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "amd-stale", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "amd-stale", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 1, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	filled, ok, err := svc.TryFillPendingOrder(ctx, *o, 100, 0.4)
	if err != nil || !ok || filled == nil {
		t.Fatalf("partial fill ok=%v err=%v filled=%+v", ok, err, filled)
	}
	if math.Abs(filled.RemainingQuantity-0.6) > 1e-9 {
		t.Fatalf("want remaining 0.6 after 0.4 fill, got %v", filled.RemainingQuantity)
	}
	// Price-only amend (frontend omits remaining when it did not change).
	trig := 99.0
	got, _, err := svc.AmendPendingOrder(ctx, AmendPendingOrderInput{
		ClientID: "amd-stale", OrderID: o.ID, TriggerPrice: &trig,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.RemainingQuantity-0.6) > 1e-9 {
		t.Fatalf("price-only amend changed remaining: remaining=%v filled=%v qty=%v",
			got.RemainingQuantity, got.FilledQuantity, got.Quantity)
	}
}
