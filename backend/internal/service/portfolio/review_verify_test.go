package portfolio

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
)

// Finding 1: concurrent PlaceMarginOrder uses a pre-lock CashBalance snapshot
// and can last-write-wins cash, opening more positions than the book can pay for.
func TestVerify_ConcurrentMarginOpensDoNotInventCash(t *testing.T) {
	const (
		startCash = 35.0
		needEach  = 20.0 // 1 BTC long 5x at 100, zero fees
		workers   = 10
		rounds    = 15
	)
	var reproduced atomic.Bool
	var lastCash float64
	var lastOpened int
	for round := 0; round < rounds && !reproduced.Load(); round++ {
		svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
		ctx := context.Background()
		client := "margin-race-" + time.Now().Format("150405.000000")
		if _, err := svc.Create(ctx, CreateInput{ClientID: client, StartingBalance: startCash}); err != nil {
			t.Fatal(err)
		}
		var start sync.WaitGroup
		start.Add(1)
		var wg sync.WaitGroup
		var opened atomic.Int64
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				start.Wait()
				pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
					ClientID: client, Symbol: "BTCUSDT", Side: "long", Type: "market",
					Quantity: 1, Leverage: 5,
				})
				if err == nil && pos != nil {
					opened.Add(1)
				}
			}()
		}
		start.Done()
		wg.Wait()

		view, err := svc.View(ctx, client)
		if err != nil {
			t.Fatal(err)
		}
		n := int(opened.Load())
		lastOpened = n
		lastCash = view.CashBalance
		// Conserved: at most one open, cash = start - n*need (n is 0 or 1).
		if n > 1 {
			reproduced.Store(true)
			t.Logf("round %d: opened=%d cash=%v (invented — two 20-margin opens from 35)", round, n, view.CashBalance)
			break
		}
		wantCash := startCash - float64(n)*needEach
		if math.Abs(view.CashBalance-wantCash) > 1e-6 {
			reproduced.Store(true)
			t.Logf("round %d: opened=%d cash=%v want %v", round, n, view.CashBalance, wantCash)
			break
		}
	}
	if reproduced.Load() {
		t.Errorf("CONFIRMED finding 1: concurrent margin opens invented or desynced cash (opened=%d cash=%v from start=%g)",
			lastOpened, lastCash, startCash)
		return
	}
	t.Logf("NOT REPRODUCED finding 1 over %d rounds (last opened=%d cash=%v)", rounds, lastOpened, lastCash)
}

// Finding 2a: after account.Close, recurring-buy worker still executes plans
// and the filler still fills pending orders (no RequireActive in those paths).
func TestVerify_ClosedAccountWorkersStillMutatePaperBook(t *testing.T) {
	ctx := context.Background()
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	if _, err := svc.Create(ctx, CreateInput{ClientID: "closed-worker", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "closed-worker", Symbol: "BTCUSDT", Amount: 200, Frequency: "daily", StartAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	pend, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "closed-worker", Symbol: "BTCUSDT", Type: "limit_buy",
		Quantity: 1, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}

	acct := account.New(accountstore.NewMemory(), account.DataPurgeDeps{Paper: svc})
	svc.SetAccountChecker(acct)
	if _, err := acct.Close(ctx, "closed-worker"); err != nil {
		t.Fatal(err)
	}
	if err := acct.RequireActive(ctx, "closed-worker"); err == nil {
		t.Fatal("precondition: account should be closed")
	}

	n, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	runs, err := svc.ListRecurringBuyRuns(ctx, "closed-worker", plan.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}

	filled, ok, ferr := svc.TryFillPendingOrder(ctx, *pend, 100, 0)
	if ferr != nil {
		t.Fatal(ferr)
	}
	view, _ := svc.View(ctx, "closed-worker")

	recurringRan := n > 0 || len(runs) > 0
	orderFilled := ok && filled != nil && filled.Status == domain.PendingStatusFilled
	if !recurringRan && !orderFilled {
		t.Log("NOT REPRODUCED finding 2: closed account blocked paper workers")
		return
	}
	t.Errorf("CONFIRMED finding 2: closed account still mutated paper book (recurringRuns=%d filled=%v cash=%v)",
		len(runs), orderFilled, view.CashBalance)
}

// Finding 2b: Close does not pause recurring plans or cancel open orders.
func TestVerify_CloseDoesNotPausePlansOrCancelOrders(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	if _, err := svc.Create(ctx, CreateInput{ClientID: "close-pause", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(time.Hour)
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "close-pause", Symbol: "BTCUSDT", Amount: 50, Frequency: "daily", StartAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "close-pause", Symbol: "BTCUSDT", Type: "limit_buy",
		Quantity: 1, TriggerPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}

	acct := account.New(accountstore.NewMemory(), account.DataPurgeDeps{Paper: svc})
	svc.SetAccountChecker(acct)
	if _, err := acct.Close(ctx, "close-pause"); err != nil {
		t.Fatal(err)
	}

	gotPlan, err := svc.GetRecurringBuyPlan(ctx, "close-pause", plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	gotOrd, err := svc.GetPendingOrder(ctx, "close-pause", o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotPlan.Status == domain.RecurringBuyActive && gotOrd.Status == domain.PendingStatusOpen {
		t.Errorf("CONFIRMED finding 2: Close left plan=%s and order=%s live", gotPlan.Status, gotOrd.Status)
		return
	}
	t.Logf("NOT REPRODUCED finding 2 close side-effects: plan=%s order=%s", gotPlan.Status, gotOrd.Status)
}
